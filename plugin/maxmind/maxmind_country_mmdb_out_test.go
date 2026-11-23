package maxmind

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter"
	"github.com/oschwald/maxminddb-golang/v2"
)

func assertOutputMMDB(t *testing.T, path string, wantCIDRs map[string][]string, wantRecords map[string]map[string]any) {
	t.Helper()
	db, err := maxminddb.Open(path)
	if err != nil {
		t.Fatalf("maxminddb.Open() error = %v", err)
	}
	defer db.Close()

	cidrs := make(map[string][]string)
	for network := range db.Networks() {
		var record map[string]any
		if err := network.Decode(&record); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		name, _ := record["country_code"].(string)
		if country, ok := record["country"].(map[string]any); ok {
			name, _ = country["iso_code"].(string)
		}
		prefix := network.Prefix().String()
		if want, ok := wantRecords[name]; !ok || !reflect.DeepEqual(record, want) {
			t.Errorf("record for %s (%q) = %v, want %v", prefix, name, record, want)
		}
		cidrs[name] = append(cidrs[name], prefix)
	}
	assertCIDRs(t, sortedCIDRs(cidrs), sortedCIDRs(wantCIDRs))
}

func sortedCIDRs(m map[string][]string) map[string][]string {
	out := make(map[string][]string, len(m))
	for k, v := range m {
		s := slices.Clone(v)
		slices.Sort(s)
		out[k] = s
	}
	return out
}

func TestGeoLite2CountryMMDBOut_FilterAndSortList(t *testing.T) {
	c := newContainerWith(t, map[string][]string{
		"CN": {"1.0.1.0/24"}, "US": {"8.8.8.0/24"}, "JP": {"1.0.16.0/24"}, "DE": {"1.0.3.0/24"},
	})
	tests := []struct {
		name string
		g    *geolite2_country_mmdb_out
		want []string
	}{
		{name: "All sorted", g: &geolite2_country_mmdb_out{}, want: []string{"CN", "DE", "JP", "US"}},
		{name: "Wanted list keeps order", g: &geolite2_country_mmdb_out{Want: []string{" us ", "", "cn", "xx"}}, want: []string{"US", "CN", "XX"}},
		{name: "Wanted list all excluded", g: &geolite2_country_mmdb_out{Want: []string{"cn"}, Exclude: []string{"CN"}}, want: []string{}},
		{name: "Wanted list with exclude", g: &geolite2_country_mmdb_out{Want: []string{"cn", "us"}, Exclude: []string{" us", ""}}, want: []string{"CN"}},
		{
			name: "Overwrite and exclude",
			g:    &geolite2_country_mmdb_out{Overwrite: []string{"cn", " ", "jp", "de"}, Exclude: []string{"de"}},
			want: []string{"US", "CN", "JP"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.g.filterAndSortList(c)
			if len(got) != len(tt.want) || !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGeoLite2CountryMMDBOut_Output(t *testing.T) {
	data := map[string][]string{
		"CN": {"1.0.1.0/24", "1.0.2.0/23", "2001:da8::/32"},
		"US": {"8.8.8.0/24", "2600::/16"},
		"JP": {"1.0.16.0/24"},
	}
	countrySrc := writeTempFile(t, "country.mmdb", buildMMDB(t, "GeoLite2-Country", 24, countrySourceRecords()))
	ipinfoSrc := writeTempFile(t, "ipinfo.mmdb", buildMMDB(t, "IPInfo-Lite", 32, ipinfoSourceRecords()))

	dbipData := map[string][]string{
		"CN": data["CN"], "US": data["US"], "JP": data["JP"], "DE": {"1.0.4.0/24"},
	}

	tests := []struct {
		name       string
		g          *geolite2_country_mmdb_out
		data       map[string][]string
		wantCIDRs  map[string][]string
		wantRecord map[string]map[string]any
	}{
		{
			name:      "GeoLite2 without source",
			g:         &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut},
			data:      data,
			wantCIDRs: data,
			wantRecord: map[string]map[string]any{
				"CN": {"country": map[string]any{"iso_code": "CN"}},
				"US": {"country": map[string]any{"iso_code": "US"}},
				"JP": {"country": map[string]any{"iso_code": "JP"}},
			},
		},
		{
			name:      "DBIP without source, only IPv6",
			g:         &geolite2_country_mmdb_out{Type: TypeDBIPCountryMMDBOut, OnlyIPType: lib.IPv6, Want: []string{"CN", "US"}},
			data:      data,
			wantCIDRs: map[string][]string{"CN": {"2001:da8::/32"}, "US": {"2600::/16"}},
			wantRecord: map[string]map[string]any{
				"CN": {"country": map[string]any{"iso_code": "CN"}},
				"US": {"country": map[string]any{"iso_code": "US"}},
			},
		},
		{
			name:      "IPInfo without source",
			g:         &geolite2_country_mmdb_out{Type: TypeIPInfoCountryMMDBOut, Want: []string{"JP"}},
			data:      data,
			wantCIDRs: map[string][]string{"JP": {"1.0.16.0/24"}},
			wantRecord: map[string]map[string]any{
				"JP": {"country_code": "JP"},
			},
		},
		{
			name:      "GeoLite2 with source",
			g:         &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: countrySrc},
			data:      data,
			wantCIDRs: data,
			wantRecord: map[string]map[string]any{
				"CN": {
					"continent": map[string]any{
						"code": "AS", "geoname_id": uint64(6255147),
						"names": wantNames("Asia", false),
					},
					"country": map[string]any{
						"iso_code": "CN", "geoname_id": uint64(1814991), "is_in_european_union": false,
						"names": wantNames("China", false),
					},
				},
				"US": {
					"country": map[string]any{
						"iso_code": "US", "geoname_id": uint64(6252001), "is_in_european_union": false,
						"names": wantNames("US", false),
					},
				},
				"JP": {"country": map[string]any{"iso_code": "JP"}},
			},
		},
		{
			name:      "DBIP with source",
			g:         &geolite2_country_mmdb_out{Type: TypeDBIPCountryMMDBOut, SourceMMDBURI: countrySrc},
			data:      dbipData,
			wantCIDRs: dbipData,
			wantRecord: map[string]map[string]any{
				"CN": {
					"continent": map[string]any{"code": "AS", "geoname_id": uint64(6255147), "names": wantNames("Asia", true)},
					"country":   map[string]any{"iso_code": "CN", "geoname_id": uint64(1814991), "is_in_european_union": false, "names": wantNames("China", true)},
				},
				"DE": {"country": map[string]any{"iso_code": "DE", "geoname_id": uint64(2921044), "is_in_european_union": true, "names": wantNames("Germany", true)}},
				"US": {"country": map[string]any{"iso_code": "US", "geoname_id": uint64(6252001), "is_in_european_union": false, "names": wantNames("US", true)}},
				"JP": {"country": map[string]any{"iso_code": "JP"}},
			},
		},
		{
			name:      "IPInfo with source",
			g:         &geolite2_country_mmdb_out{Type: TypeIPInfoCountryMMDBOut, SourceMMDBURI: ipinfoSrc},
			data:      data,
			wantCIDRs: data,
			wantRecord: map[string]map[string]any{
				"CN": {"as_domain": "", "as_name": "", "asn": "", "continent": "Asia", "continent_code": "AS", "country": "China", "country_code": "CN"},
				"US": {"country_code": "US"},
				"JP": {"country_code": "JP"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.g.Action = lib.ActionOutput
			tt.g.OutputDir = t.TempDir()
			tt.g.OutputName = "out.mmdb"
			if err := tt.g.Output(newContainerWith(t, tt.data)); err != nil {
				t.Fatalf("Output() error = %v", err)
			}
			assertOutputMMDB(t, filepath.Join(tt.g.OutputDir, tt.g.OutputName), tt.wantCIDRs, tt.wantRecord)
		})
	}
}

func wantNames(prefix string, dbip bool) map[string]any {
	names := map[string]any{}
	for _, lang := range []string{"de", "en", "es", "fr", "ja", "pt-BR", "ru", "zh-CN"} {
		names[lang] = prefix + "-" + lang
	}
	if dbip {
		names["fa"] = prefix + "-fa"
		names["ko"] = prefix + "-ko"
	}
	return names
}

func TestGeoLite2CountryMMDBOut_OutputNoWrite(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "out")
	g := &geolite2_country_mmdb_out{
		Type: TypeGeoLite2CountryMMDBOut, Action: lib.ActionOutput, OutputDir: dir, OutputName: "out.mmdb",
		Want: []string{"XX"},
	}
	if err := g.Output(newContainerWith(t, map[string][]string{"CN": {"1.0.1.0/24"}})); err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("output dir should not be created, stat error = %v", err)
	}
}

func TestGeoLite2CountryMMDBOut_OutputErrors(t *testing.T) {
	data := map[string][]string{"CN": {"2001:da8::/32"}}
	fileAsDir := writeTempFile(t, "file", []byte("x"))

	tests := []struct {
		name string
		g    *geolite2_country_mmdb_out
	}{
		{name: "Extra info error", g: &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: "/non/existent"}},
		{name: "Marshal error with no prefix", g: &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, OnlyIPType: lib.IPv4}},
		{name: "Unsupported type", g: &geolite2_country_mmdb_out{Type: "unknown"}},
		{name: "Output dir is a file", g: &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, OutputDir: fileAsDir}},
		{name: "Output name is a dir", g: &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, OutputName: "."}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.g.Action = lib.ActionOutput
			if tt.g.OutputDir == "" {
				tt.g.OutputDir = t.TempDir()
			}
			if tt.g.OutputName == "" {
				tt.g.OutputName = "out.mmdb"
			}
			if err := tt.g.Output(newContainerWith(t, data)); err == nil {
				t.Errorf("Output() expected error, got nil")
			}
		})
	}
}

func TestGeoLite2CountryMMDBOut_MarshalDataUnsupportedTypeWithSource(t *testing.T) {
	writer, err := mmdbwriter.New(mmdbwriter.Options{})
	if err != nil {
		t.Fatalf("mmdbwriter.New() error = %v", err)
	}
	entry := lib.NewEntry("CN")
	if err := entry.AddPrefix("1.0.1.0/24"); err != nil {
		t.Fatalf("AddPrefix() error = %v", err)
	}
	g := &geolite2_country_mmdb_out{Type: "unknown", SourceMMDBURI: "src.mmdb"}
	if err := g.marshalData(writer, entry, nil); err != lib.ErrNotSupportedFormat {
		t.Errorf("marshalData() error = %v, want %v", err, lib.ErrNotSupportedFormat)
	}
}
