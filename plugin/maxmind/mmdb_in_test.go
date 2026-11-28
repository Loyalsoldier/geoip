package maxmind

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/oschwald/maxminddb-golang/v2"
)

func countryCode(code string) mmdbtype.Map {
	return mmdbtype.Map{"iso_code": mmdbtype.String(code)}
}

func TestMMDBInput(t *testing.T) {
	countries := mmdbFixture(t, "Country", map[string]mmdbtype.DataType{
		"192.0.2.0/25":    mmdbtype.Map{"country": countryCode(" de "), "registered_country": countryCode("US"), "represented_country": countryCode("GB")},
		"198.51.100.0/24": mmdbtype.Map{"country": countryCode(" "), "registered_country": countryCode("jp"), "represented_country": countryCode("KR")},
		"2001:db8:1::/48": mmdbtype.Map{"represented_country": countryCode("DE")},
		"2001:db8:2::/48": mmdbtype.Map{"continent": mmdbtype.Map{"code": mmdbtype.String("AS")}},
	})
	ipinfo := mmdbFixture(t, "IPInfo-Lite", map[string]mmdbtype.DataType{
		"192.0.2.0/25":    mmdbtype.Map{"country_code": mmdbtype.String(" de ")},
		"198.51.100.0/24": mmdbtype.Map{"country_code": mmdbtype.String("JP")},
		"2001:db8:1::/48": mmdbtype.Map{"country_code": mmdbtype.String("DE")},
		"2001:db8:2::/48": mmdbtype.Map{"country_code": mmdbtype.String(" ")},
	})
	countryURI := writeFixture(t, "country.mmdb", countries)
	all := map[string][]string{"DE": {"192.0.2.0/25", "2001:db8:1::/48"}, "JP": {"198.51.100.0/24"}}
	tests := []struct {
		name, typ, uri string
		action         lib.Action
		wanted         []string
		only           lib.IPType
		initial, want  map[string][]string
	}{
		{"GeoLite2", TypeGeoLite2CountryMMDBIn, countryURI, lib.ActionAdd, nil, "", nil, all},
		{"DB-IP", TypeDBIPCountryMMDBIn, serveFixture(t, countries), lib.ActionAdd, nil, "", nil, all},
		{"IPinfo", TypeIPInfoCountryMMDBIn, writeFixture(t, "ipinfo.mmdb", ipinfo), lib.ActionAdd, nil, "", nil, all},
		{"wanted IPv6", TypeGeoLite2CountryMMDBIn, countryURI, lib.ActionAdd, []string{" de "}, lib.IPv6, nil,
			map[string][]string{"DE": {"2001:db8:1::/48"}}},
		{"remove IPv4", TypeGeoLite2CountryMMDBIn, countryURI, lib.ActionRemove, []string{"DE"}, lib.IPv4,
			map[string][]string{"DE": {"192.0.2.0/24", "2001:db8:1::/48"}}, map[string][]string{"DE": {"192.0.2.128/25", "2001:db8:1::/48"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in, err := newGeoLite2CountryMMDBIn(tt.typ, "", tt.action, WithURI(tt.uri), WithInputWantedList(tt.wanted), WithInputOnlyIPType(tt.only))
			if err != nil {
				t.Fatal(err)
			}
			got, err := in.Input(containerWith(t, tt.initial))
			if err != nil {
				t.Fatal(err)
			}
			assertContainer(t, got, tt.want)
		})
	}
	t.Run("no matching country", func(t *testing.T) {
		in, err := NewGeoLite2CountryMMDBIn(lib.ActionAdd, WithURI(countryURI), WithInputWantedList([]string{"CA"}))
		if err != nil {
			t.Fatal(err)
		}
		if got, err := in.Input(lib.NewContainer()); got != nil || err == nil || !strings.Contains(err.Error(), "no entry is generated") {
			t.Errorf("Input = (%v, %v), want no entry error", got, err)
		}
	})
	t.Run("action errors", func(t *testing.T) {
		testInputActionErrors(t, func(action lib.Action) (lib.InputConverter, error) {
			return NewGeoLite2CountryMMDBIn(action, WithURI(countryURI))
		})
	})
}

// Both importing networks and reading source metadata must reject invalid databases.
func TestMMDBSourceErrors(t *testing.T) {
	records := map[string]mmdbtype.DataType{"192.0.2.0/24": mmdbtype.Map{"country": countryCode("DE")}}
	badSchema := writeFixture(t, "wrong-schema.mmdb", mmdbFixture(t, "Test", map[string]mmdbtype.DataType{
		"192.0.2.0/24": mmdbtype.String("not a country record"),
	}))
	invalidDB := func(err error) bool { return errors.As(err, new(maxminddb.InvalidDatabaseError)) }
	badRecord := func(err error) bool { return errors.As(err, new(maxminddb.UnmarshalTypeError)) }
	tests := []struct {
		name, typ, uri string
		isError        func(error) bool
	}{
		{"missing file", TypeGeoLite2CountryMMDBIn, filepath.Join(t.TempDir(), "missing"), func(err error) bool { return errors.Is(err, os.ErrNotExist) }},
		{"not an MMDB", TypeGeoLite2CountryMMDBIn, serveFixture(t, []byte("not an MMDB")), invalidDB},
		{"missing database type", TypeGeoLite2CountryMMDBIn, writeFixture(t, "unverified.mmdb", mmdbFixture(t, "", records)), invalidDB},
		{"wrong GeoLite2 schema", TypeGeoLite2CountryMMDBIn, badSchema, badRecord},
		{"wrong DB-IP schema", TypeDBIPCountryMMDBIn, badSchema, badRecord},
		{"wrong IPinfo schema", TypeIPInfoCountryMMDBIn, badSchema, badRecord},
		{"unsupported format", "unknown", writeFixture(t, "valid.mmdb", mmdbFixture(t, "Test", records)), func(err error) bool {
			return errors.Is(err, lib.ErrNotSupportedFormat)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := geolite2_country_mmdb_in{Type: tt.typ, Action: lib.ActionAdd, URI: tt.uri}
			if got, err := in.Input(lib.NewContainer()); got != nil || !tt.isError(err) {
				t.Errorf("Input = (%v, %v), want %s error", got, err, tt.name)
			}
			out := geolite2_country_mmdb_out{Type: tt.typ, SourceMMDBURI: tt.uri}
			if got, err := out.GetExtraInfo(); got != nil || !tt.isError(err) {
				t.Errorf("GetExtraInfo = (%v, %v), want %s error", got, err, tt.name)
			}
		})
	}
}
