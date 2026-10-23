package maxmind

import (
	"net/http"
	"path/filepath"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestNewIPInfoCountryMMDBOut(t *testing.T) {
	t.Parallel()

	testMMDBOutConstructors(t, findMMDBOutputType(t, TypeIPInfoCountryMMDBOut))
}

func ipInfoRecord(asDomain, asName, asn, continent, continentCode, country, countryCode string) map[string]any {
	return map[string]any{
		"as_domain":      asDomain,
		"as_name":        asName,
		"asn":            asn,
		"continent":      continent,
		"continent_code": continentCode,
		"country":        country,
		"country_code":   countryCode,
	}
}

func TestIPInfoCountryMMDBOut_OutputWithSource(t *testing.T) {
	t.Parallel()

	records := append(ipInfoStyleRecords(),
		mmdbRecord{"4.0.0.0/16", mMap{"asn": mStr("AS1")}},
	)
	srv := newTestServer(t, http.StatusOK, buildMMDB(t, records...))

	// ASN info of the source MMDB must not be written into the output.
	cn := ipInfoRecord("", "", "", "Asia", "AS", "China", "CN")
	us := ipInfoRecord("", "", "", "North America", "NA", "United States", "US")
	de := ipInfoRecord("", "", "", "Europe", "EU", "Germany", "DE")

	tests := []struct {
		name       string
		onlyIPType lib.IPType
		// lists must have prefixes of onlyIPType, otherwise Output fails
		wantedList []string
		want       map[string]map[string]any
	}{
		{
			name: "all IP types",
			want: map[string]map[string]any{
				"1.0.0.0/24":     cn,
				"2402:4e00::/32": cn,
				"2.0.0.0/16":     us,
				"2001:db8::/32":  de,
				"5.0.0.0/16":     countryCodeRecord("XX"),
			},
		},
		{
			name:       "only IPv4",
			onlyIPType: lib.IPv4,
			wantedList: []string{"cn", "us", "xx"},
			want: map[string]map[string]any{
				"1.0.0.0/24": cn,
				"2.0.0.0/16": us,
				"5.0.0.0/16": countryCodeRecord("XX"),
			},
		},
		{
			name:       "only IPv6",
			onlyIPType: lib.IPv6,
			wantedList: []string{"cn", "de"},
			want: map[string]map[string]any{
				"2402:4e00::/32": cn,
				"2001:db8::/32":  de,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			outputDir := t.TempDir()
			converter, err := NewIPInfoCountryMMDBOut(lib.ActionOutput,
				WithOutputDir(outputDir),
				WithOutputOnlyIPType(tt.onlyIPType),
				WithOutputWantedList(tt.wantedList),
				WithSourceMMDBURI(srv.URL+"/country.mmdb"),
			)
			assertError(t, err, false, "", nil)

			container := newContainer(t, map[string][]string{
				"CN": {"1.0.0.0/24", "2402:4e00::/32"},
				"US": {"2.0.0.0/16"},
				"DE": {"2001:db8::/32"},
				"XX": {"5.0.0.0/16"}, // no extra info in source MMDB
			})
			assertError(t, converter.Output(container), false, "", nil)

			networks, metadata := readMMDBNetworks(t, filepath.Join(outputDir, defaultGeoLite2CountryMMDBOutputName))
			if metadata.DatabaseType != "IPInfo-Lite" {
				t.Errorf("DatabaseType = %q, want %q", metadata.DatabaseType, "IPInfo-Lite")
			}
			assertMMDBNetworks(t, networks, tt.want)
		})
	}
}

func TestIPInfoCountryMMDBOut_marshalData(t *testing.T) {
	t.Parallel()

	extraInfo := map[string]any{
		"CN": ipInfoLite{
			ASN:           "AS1",
			ASName:        "Name",
			ASDomain:      "example.com",
			Continent:     "Asia",
			ContinentCode: "AS",
			Country:       "China",
			CountryCode:   "should-be-ignored",
		},
		"DE": ipInfoLite{},
		// a map is not a valid IPInfo extra info
		"GB": map[string]string{"country_code": "GB"},
	}

	cn := ipInfoRecord("example.com", "Name", "AS1", "Asia", "AS", "China", "CN")

	tests := []struct {
		name  string
		entry *lib.Entry
		want  map[string]map[string]any
	}{
		{
			name:  "extra info found",
			entry: newEntry(t, "CN", "1.0.0.0/24", "2402:4e00::/32"),
			want: map[string]map[string]any{
				"1.0.0.0/24":     cn,
				"2402:4e00::/32": cn,
			},
		},
		{
			name:  "empty extra info",
			entry: newEntry(t, "DE", "2001:db8::/32"),
			want: map[string]map[string]any{
				"2001:db8::/32": ipInfoRecord("", "", "", "", "", "", "DE"),
			},
		},
		{
			name:  "extra info not found",
			entry: newEntry(t, "XX", "5.0.0.0/16"),
			want:  map[string]map[string]any{"5.0.0.0/16": countryCodeRecord("XX")},
		},
		{
			name:  "extra info of wrong type",
			entry: newEntry(t, "GB", "6.0.0.0/16"),
			want:  map[string]map[string]any{"6.0.0.0/16": countryCodeRecord("GB")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := &geolite2_country_mmdb_out{Type: TypeIPInfoCountryMMDBOut, SourceMMDBURI: "source.mmdb"}
			assertMMDBNetworks(t, marshalAndRead(t, g, tt.entry, extraInfo), tt.want)
		})
	}
}
