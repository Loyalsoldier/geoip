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

// readOutputMMDB reads an MMDB file and returns, per list name, the sorted CIDRs and the decoded record.
func readOutputMMDB(t *testing.T, path string) (map[string][]string, map[string]map[string]any) {
	t.Helper()
	db, err := maxminddb.Open(path)
	if err != nil {
		t.Fatalf("maxminddb.Open() error = %v", err)
	}
	defer db.Close()

	cidrs := make(map[string][]string)
	records := make(map[string]map[string]any)
	for network := range db.Networks() {
		var record map[string]any
		if err := network.Decode(&record); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		var name string
		if country, ok := record["country"].(map[string]any); ok {
			name = country["iso_code"].(string)
		} else {
			name = record["country_code"].(string)
		}
		cidrs[name] = append(cidrs[name], network.Prefix().String())
		records[name] = record
	}
	for name := range cidrs {
		slices.Sort(cidrs[name])
	}
	return cidrs, records
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
			},
		},
		{
			name:      "DBIP without source, only IPv6",
			g:         &geolite2_country_mmdb_out{Type: TypeDBIPCountryMMDBOut, OnlyIPType: lib.IPv6, Want: []string{"CN", "US"}},
			data:      data,
			wantCIDRs: map[string][]string{"CN": {"2001:da8::/32"}, "US": {"2600::/16"}},
			wantRecord: map[string]map[string]any{
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
						"names": map[string]any{"de": "Asia-de", "en": "Asia-en", "es": "Asia-es", "fr": "Asia-fr", "ja": "Asia-ja", "pt-BR": "Asia-pt-BR", "ru": "Asia-ru", "zh-CN": "Asia-zh-CN"},
					},
					"country": map[string]any{
						"iso_code": "CN", "geoname_id": uint64(1814991), "is_in_european_union": false,
						"names": map[string]any{"de": "China-de", "en": "China-en", "es": "China-es", "fr": "China-fr", "ja": "China-ja", "pt-BR": "China-pt-BR", "ru": "China-ru", "zh-CN": "China-zh-CN"},
					},
				},
				"US": {
					"country": map[string]any{
						"iso_code": "US", "geoname_id": uint64(6252001), "is_in_european_union": false,
						"names": map[string]any{"de": "US-de", "en": "US-en", "es": "US-es", "fr": "US-fr", "ja": "US-ja", "pt-BR": "US-pt-BR", "ru": "US-ru", "zh-CN": "US-zh-CN"},
					},
				},
				"JP": {"country": map[string]any{"iso_code": "JP"}},
			},
		},
		{
			name:      "DBIP with source",
			g:         &geolite2_country_mmdb_out{Type: TypeDBIPCountryMMDBOut, SourceMMDBURI: countrySrc},
			data:      data,
			wantCIDRs: data,
			wantRecord: map[string]map[string]any{
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
			gotCIDRs, gotRecords := readOutputMMDB(t, filepath.Join(tt.g.OutputDir, tt.g.OutputName))
			assertCIDRs(t, gotCIDRs, sortedCIDRs(tt.wantCIDRs))
			for name, want := range tt.wantRecord {
				if !reflect.DeepEqual(gotRecords[name], want) {
					t.Errorf("record[%s] = %v, want %v", name, gotRecords[name], want)
				}
			}
		})
	}
}

func TestGeoLite2CountryMMDBOut_OutputDBIPWithSourceContinent(t *testing.T) {
	src := writeTempFile(t, "country.mmdb", buildMMDB(t, "DBIP-Country-Lite", 24, countrySourceRecords()))
	g := &geolite2_country_mmdb_out{
		Type: TypeDBIPCountryMMDBOut, Action: lib.ActionOutput, SourceMMDBURI: src,
		OutputDir: t.TempDir(), OutputName: "out.mmdb",
	}
	if err := g.Output(newContainerWith(t, map[string][]string{"CN": {"1.0.1.0/24"}, "DE": {"1.0.3.0/24"}})); err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	gotCIDRs, gotRecords := readOutputMMDB(t, filepath.Join(g.OutputDir, g.OutputName))
	assertCIDRs(t, gotCIDRs, map[string][]string{"CN": {"1.0.1.0/24"}, "DE": {"1.0.3.0/24"}})

	cn := gotRecords["CN"]
	continent := cn["continent"].(map[string]any)
	if continent["code"] != "AS" || continent["geoname_id"] != uint64(6255147) {
		t.Errorf("CN continent = %v", continent)
	}
	country := cn["country"].(map[string]any)
	if country["iso_code"] != "CN" || country["geoname_id"] != uint64(1814991) {
		t.Errorf("CN country = %v", country)
	}
	if _, ok := gotRecords["DE"]["continent"]; ok {
		t.Errorf("DE should not have continent: %v", gotRecords["DE"])
	}
	de := gotRecords["DE"]["country"].(map[string]any)
	if de["iso_code"] != "DE" || de["is_in_european_union"] != true {
		t.Errorf("DE country = %v", de)
	}
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
