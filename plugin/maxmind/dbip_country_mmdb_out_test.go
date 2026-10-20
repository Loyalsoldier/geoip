package maxmind

import (
	"maps"
	"net/http"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/oschwald/geoip2-golang/v2"
)

func TestNewDBIPCountryMMDBOut(t *testing.T) {
	t.Parallel()

	testMMDBOutConstructors(t, findMMDBOutputType(t, TypeDBIPCountryMMDBOut))
}

// withoutNames checks that every "continent" and "country" sub-record of
// record contains a "names" record with exactly the given languages and
// returns a copy of record without these "names" records.
func withoutNames(t *testing.T, network string, record map[string]any, languages []string) map[string]any {
	t.Helper()

	wantLanguages := slices.Sorted(slices.Values(languages))
	stripped := maps.Clone(record)
	for _, key := range []string{"continent", "country"} {
		sub, ok := record[key].(map[string]any)
		if !ok {
			continue
		}

		names, ok := sub["names"].(map[string]any)
		if !ok {
			t.Fatalf("network %s: %s.names = %#v, want a map", network, key, sub["names"])
		}
		if gotLanguages := sortedKeys(names); !slices.Equal(gotLanguages, wantLanguages) {
			t.Fatalf("network %s: %s.names has languages %v, want %v", network, key, gotLanguages, wantLanguages)
		}

		sub = maps.Clone(sub)
		delete(sub, "names")
		stripped[key] = sub
	}

	return stripped
}

// TestDBIPCountryMMDBOut_OutputWithSource checks the records written with
// extra info from a source MMDB file. The localized names are only checked
// for their languages, because the embedded geoip2 types take over the
// decoding of DB-IP records so that the localized names are not decoded.
func TestDBIPCountryMMDBOut_OutputWithSource(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.StatusOK, buildMMDB(t, geoLite2StyleRecords()...))

	cn := map[string]any{
		"continent": map[string]any{"code": "AS", "geoname_id": uint64(6255147)},
		"country":   map[string]any{"iso_code": "CN", "geoname_id": uint64(1814991), "is_in_european_union": false},
	}
	us := map[string]any{
		"continent": map[string]any{"code": "NA", "geoname_id": uint64(6255149)},
		"country":   map[string]any{"iso_code": "US", "geoname_id": uint64(6252001), "is_in_european_union": false},
	}
	it := map[string]any{
		"continent": map[string]any{"code": "EU", "geoname_id": uint64(6255148)},
		"country":   map[string]any{"iso_code": "IT", "geoname_id": uint64(3175395), "is_in_european_union": true},
	}
	de := map[string]any{
		"country": map[string]any{"iso_code": "DE", "geoname_id": uint64(2921044), "is_in_european_union": true},
	}

	outputDir := t.TempDir()
	converter, err := NewDBIPCountryMMDBOut(lib.ActionOutput,
		WithOutputDir(outputDir),
		WithOutputName("dbip.mmdb"),
		WithSourceMMDBURI(srv.URL+"/source.mmdb"),
	)
	assertError(t, err, false, "", nil)

	container := newContainer(t, map[string][]string{
		"CN": {"1.0.0.0/24", "2402:4e00::/32"},
		"US": {"2.0.0.0/16", "2400:cb00::/32"},
		"IT": {"3.0.0.0/16"},
		"DE": {"2001:db8::/32"},
		"XX": {"5.0.0.0/16"}, // no extra info in source MMDB
	})
	assertError(t, converter.Output(container), false, "", nil)

	networks, metadata := readMMDBNetworks(t, filepath.Join(outputDir, "dbip.mmdb"))
	if metadata.DatabaseType != "DBIP-Country-Lite" {
		t.Errorf("DatabaseType = %q, want %q", metadata.DatabaseType, "DBIP-Country-Lite")
	}

	stripped := make(map[string]map[string]any, len(networks))
	for network, record := range networks {
		if network == "5.0.0.0/16" { // record without extra info has no names
			stripped[network] = record
			continue
		}
		stripped[network] = withoutNames(t, network, record, dbipLanguages)
	}
	assertMMDBNetworks(t, stripped, map[string]map[string]any{
		"1.0.0.0/24":     cn,
		"2402:4e00::/32": cn,
		"2.0.0.0/16":     us,
		"2400:cb00::/32": us,
		"3.0.0.0/16":     it,
		"2001:db8::/32":  de,
		"5.0.0.0/16":     isoCodeRecord("XX"),
	})
}

func TestDBIPCountryMMDBOut_marshalData(t *testing.T) {
	t.Parallel()

	names := dbipLanguageNames{
		Names: geoip2.Names{
			German:              "de-name",
			English:             "en-name",
			Spanish:             "es-name",
			French:              "fr-name",
			Japanese:            "ja-name",
			BrazilianPortuguese: "pt-BR-name",
			Russian:             "ru-name",
			SimplifiedChinese:   "zh-CN-name",
		},
		Persian: "fa-name",
		Korean:  "ko-name",
	}
	wantNames := map[string]any{
		"de":    "de-name",
		"en":    "en-name",
		"es":    "es-name",
		"fa":    "fa-name",
		"fr":    "fr-name",
		"ja":    "ja-name",
		"ko":    "ko-name",
		"pt-BR": "pt-BR-name",
		"ru":    "ru-name",
		"zh-CN": "zh-CN-name",
	}

	extraInfo := map[string]any{
		"CN": dbipCountry{
			Continent: dbipContinent{
				Continent: geoip2.Continent{Code: "AS", GeoNameID: 1},
				Names:     names,
			},
			Country: dbipCountryRecord{
				CountryRecord: geoip2.CountryRecord{ISOCode: "should-be-ignored", GeoNameID: 2, IsInEuropeanUnion: true},
				Names:         names,
			},
		},
		"DE": dbipCountry{
			Country: dbipCountryRecord{
				CountryRecord: geoip2.CountryRecord{GeoNameID: 3},
				Names:         names,
			},
		},
		"ES": dbipCountry{
			Country: dbipCountryRecord{CountryRecord: geoip2.CountryRecord{GeoNameID: 4}},
		},
		// a geoip2.Country is not a valid DB-IP extra info
		"GB": geoip2.Country{Country: geoip2.CountryRecord{GeoNameID: 5}},
	}

	cnRecord := map[string]any{
		"continent": map[string]any{"code": "AS", "geoname_id": uint64(1), "names": wantNames},
		"country":   map[string]any{"iso_code": "CN", "geoname_id": uint64(2), "is_in_european_union": true, "names": wantNames},
	}

	tests := []struct {
		name  string
		entry *lib.Entry
		want  map[string]map[string]any
	}{
		{
			name:  "extra info with continent",
			entry: newEntry(t, "CN", "1.0.0.0/24", "2402:4e00::/32"),
			want: map[string]map[string]any{
				"1.0.0.0/24":     cnRecord,
				"2402:4e00::/32": cnRecord,
			},
		},
		{
			name:  "extra info without continent",
			entry: newEntry(t, "DE", "2001:db8::/32"),
			want: map[string]map[string]any{
				"2001:db8::/32": {
					"country": map[string]any{"iso_code": "DE", "geoname_id": uint64(3), "is_in_european_union": false, "names": wantNames},
				},
			},
		},
		{
			name:  "extra info without names",
			entry: newEntry(t, "ES", "7.0.0.0/16"),
			want: map[string]map[string]any{
				"7.0.0.0/16": {
					"country": map[string]any{
						"iso_code":             "ES",
						"geoname_id":           uint64(4),
						"is_in_european_union": false,
						"names":                namesRecord(dbipLanguages, nil),
					},
				},
			},
		},
		{
			name:  "extra info not found",
			entry: newEntry(t, "XX", "5.0.0.0/16"),
			want:  map[string]map[string]any{"5.0.0.0/16": isoCodeRecord("XX")},
		},
		{
			name:  "extra info of wrong type",
			entry: newEntry(t, "GB", "6.0.0.0/16"),
			want:  map[string]map[string]any{"6.0.0.0/16": isoCodeRecord("GB")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := &geolite2_country_mmdb_out{Type: TypeDBIPCountryMMDBOut, SourceMMDBURI: "source.mmdb"}
			assertMMDBNetworks(t, marshalAndRead(t, g, tt.entry, extraInfo), tt.want)
		})
	}
}
