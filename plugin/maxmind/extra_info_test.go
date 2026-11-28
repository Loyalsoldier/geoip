package maxmind

import (
	"errors"
	"maps"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/oschwald/geoip2-golang/v2"
)

func TestEnrichedCountryOutput(t *testing.T) {
	c := containerWith(t, map[string][]string{
		"DE": {"192.0.2.0/25", "2001:db8::/32"}, "JP": {"192.0.2.128/25"}, "KR": {"198.51.100.0/24"}, "ZZ": {"203.0.113.0/24"},
	})
	for _, tt := range []struct {
		typ       string
		languages []string
	}{
		{TypeGeoLite2CountryMMDBOut, geoLiteLanguages}, {TypeDBIPCountryMMDBOut, dbipLanguages},
	} {
		t.Run(tt.typ, func(t *testing.T) {
			// Distinct values make a swapped language or a country/continent mix-up observable.
			names := func(label string) mmdbtype.Map {
				m := mmdbtype.Map{}
				for _, lang := range tt.languages {
					m[mmdbtype.String(lang)] = mmdbtype.String(label + "-" + lang)
				}
				return m
			}
			country := func(code string, id uint32, eu bool) mmdbtype.Map {
				return mmdbtype.Map{
					"iso_code": mmdbtype.String(code), "geoname_id": mmdbtype.Uint32(id),
					"names": names(code), "is_in_european_union": mmdbtype.Bool(eu),
				}
			}
			europe := mmdbtype.Map{"code": mmdbtype.String("EU"), "geoname_id": mmdbtype.Uint32(10), "names": names("Europe")}
			de, jp, kr := country("DE", 1, true), country("JP", 2, false), country("KR", 3, false)
			sourceDE := maps.Clone(de)
			sourceDE["iso_code"] = mmdbtype.String(" de ")
			representedKR := maps.Clone(kr)
			representedKR["type"] = mmdbtype.String("military")
			source := mmdbFixture(t, "Country", map[string]mmdbtype.DataType{
				"192.0.2.1/32": mmdbtype.Map{"country": sourceDE, "registered_country": countryCode("ZZ"), "continent": europe},
				"192.0.2.2/32": mmdbtype.Map{"registered_country": jp, "represented_country": countryCode("ZZ")},
				"192.0.2.3/32": mmdbtype.Map{"represented_country": representedKR},
				"192.0.2.4/32": mmdbtype.Map{"country": countryCode("DE")}, // first record supplies the extra info
			})
			g := geolite2_country_mmdb_out{
				Type: tt.typ, OutputDir: t.TempDir(), OutputName: "enriched.mmdb",
				SourceMMDBURI: serveFixture(t, source), OnlyIPType: lib.IPv4,
			}
			if err := g.Output(c); err != nil {
				t.Fatal(err)
			}
			assertDatabaseRecords(t, openOutput(t, &g), map[string]mmdbtype.Map{
				"192.0.2.0/25": {"country": de, "continent": europe}, "192.0.2.128/25": {"country": jp},
				"198.51.100.0/24": {"country": kr}, "203.0.113.0/24": {"country": countryCode("ZZ")},
			})
		})
	}
}

func TestEnrichedIPInfoOutput(t *testing.T) {
	source := mmdbFixture(t, "IPInfo-Lite", map[string]mmdbtype.DataType{
		"192.0.2.0/25": mmdbtype.Map{
			"country_code": mmdbtype.String(" de "), "country": mmdbtype.String("Germany"),
			"continent_code": mmdbtype.String("EU"), "continent": mmdbtype.String("Europe"),
			"asn": mmdbtype.String("AS64496"), "as_name": mmdbtype.String("Example"), "as_domain": mmdbtype.String("example.org"),
		},
		"192.0.2.128/25": mmdbtype.Map{"country_code": mmdbtype.String("DE")},
	})
	g := geolite2_country_mmdb_out{
		Type: TypeIPInfoCountryMMDBOut, OutputDir: t.TempDir(), OutputName: "enriched.mmdb",
		SourceMMDBURI: writeFixture(t, "source.mmdb", source), OnlyIPType: lib.IPv6,
	}
	c := containerWith(t, map[string][]string{"DE": {"192.0.2.0/24", "2001:db8:1::/48"}, "ZZ": {"2001:db8:2::/48"}})
	if err := g.Output(c); err != nil {
		t.Fatal(err)
	}
	assertDatabaseRecords(t, openOutput(t, &g), map[string]mmdbtype.Map{
		"2001:db8:1::/48": {
			"country_code": mmdbtype.String("DE"), "country": mmdbtype.String("Germany"),
			"continent_code": mmdbtype.String("EU"), "continent": mmdbtype.String("Europe"),
			"asn": mmdbtype.String(""), "as_name": mmdbtype.String(""), "as_domain": mmdbtype.String(""),
		},
		"2001:db8:2::/48": {"country_code": mmdbtype.String("ZZ")},
	})
}

func TestExtraInfoUnavailable(t *testing.T) {
	t.Run("no source requested", func(t *testing.T) {
		g := geolite2_country_mmdb_out{SourceMMDBURI: " "}
		if got, err := g.GetExtraInfo(); err != nil || got != nil {
			t.Errorf("GetExtraInfo = (%v, %v), want (nil, nil)", got, err)
		}
	})
	t.Run("source has no countries", func(t *testing.T) {
		source := mmdbFixture(t, "Country", map[string]mmdbtype.DataType{"192.0.2.0/24": mmdbtype.Map{"country": countryCode(" ")}})
		g := geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: writeFixture(t, "empty.mmdb", source)}
		if got, err := g.GetExtraInfo(); got != nil || err == nil || !strings.Contains(err.Error(), "no extra info found") {
			t.Errorf("GetExtraInfo = (%v, %v), want no extra info error", got, err)
		}
	})
	t.Run("unsupported enriched record format", func(t *testing.T) {
		g := geolite2_country_mmdb_out{Type: "unknown", SourceMMDBURI: "source.mmdb"}
		entry := lib.NewEntry("DE")
		if err := entry.AddPrefix("192.0.2.0/24"); err != nil {
			t.Fatal(err)
		}
		if err := g.marshalData(nil, entry, nil); !errors.Is(err, lib.ErrNotSupportedFormat) {
			t.Errorf("got %v, want ErrNotSupportedFormat", err)
		}
	})
}

func TestDBIPHasData(t *testing.T) {
	names := dbipLanguageNames{Persian: "name-fa"}
	continent := dbipContinent{Names: names}
	country := dbipCountryRecord{geoCountryFields: geoCountryFields{ISOCode: "DE"}}
	represented := dbipRepresentedCountry{Names: dbipLanguageNames{Korean: "name-ko"}}
	for _, tt := range []struct {
		name string
		data interface{ HasData() bool }
		want bool
	}{
		{"empty names", dbipLanguageNames{}, false}, {"localized names", names, true},
		{"empty continent", dbipContinent{}, false}, {"empty country", dbipCountryRecord{}, false},
		{"empty represented country", dbipRepresentedCountry{}, false}, {"empty record", dbipCountry{}, false},
		{"continent", dbipCountry{Continent: continent}, true},
		{"country", dbipCountry{Country: country}, true},
		{"represented", dbipCountry{RepresentedCountry: represented}, true},
		{"registered", dbipCountry{RegisteredCountry: country}, true},
		{"traits", dbipCountry{Traits: geoip2.CountryTraits{IsAnycast: true}}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.data.HasData(); got != tt.want {
				t.Errorf("HasData = %t, want %t", got, tt.want)
			}
		})
	}
}
