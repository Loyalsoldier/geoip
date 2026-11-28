package maxmind

import (
	"encoding/csv"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	locationsHeader = "geoname_id,locale_code,continent_code,continent_name,country_iso_code\n"
	countryHeader   = "network,geoname_id,registered_country_geoname_id,represented_country_geoname_id\n"
	asnHeader       = "network,autonomous_system_number,autonomous_system_organization\n"
)

func TestCountryCSVInput(t *testing.T) {
	locations := serveFixture(t, []byte(locationsHeader+"1,en,EU,Europe, de \n2,en,AS,Asia,JP\n"))
	v4 := writeFixture(t, "blocks.csv", []byte(countryHeader+
		"192.0.2.0/25,1,2,2\n"+ // location wins over registered and represented
		"198.51.100.0/24,999, 2 ,1\n"+ // unknown location: fall back to registered
		"203.0.113.1/32,999,,999\n")) // no country: skip
	v6 := serveFixture(t, []byte(countryHeader+
		" 2001:DB8:1::/48 ,,, 1 \n"+ // represented country only
		"2001:db8:2::/48,2,,\n"))
	tests := []struct {
		name    string
		action  lib.Action
		wanted  []string
		only    lib.IPType
		initial map[string][]string
		want    map[string][]string
	}{
		{"all countries", lib.ActionAdd, nil, "", nil, map[string][]string{
			"DE": {"192.0.2.0/25", "2001:db8:1::/48"}, "JP": {"198.51.100.0/24", "2001:db8:2::/48"},
		}},
		{"wanted IPv6 country", lib.ActionAdd, []string{" jp "}, lib.IPv6, nil, map[string][]string{
			"JP": {"2001:db8:2::/48"},
		}},
		{"remove only IPv4", lib.ActionRemove, []string{"DE"}, lib.IPv4, map[string][]string{
			"DE": {"192.0.2.0/24", "2001:db8:1::/48"},
		}, map[string][]string{"DE": {"192.0.2.128/25", "2001:db8:1::/48"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in, err := NewGeoLite2CountryCSVIn(tt.action, WithCountryCodeFile(locations), WithIPv4File(v4), WithIPv6File(v6),
				WithInputWantedList(tt.wanted), WithInputOnlyIPType(tt.only))
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
	t.Run("action errors", func(t *testing.T) {
		testInputActionErrors(t, func(action lib.Action) (lib.InputConverter, error) {
			return NewGeoLite2CountryCSVIn(action, WithCountryCodeFile(locations), WithIPv4File(v4))
		})
	})
}

func TestASNCSVInput(t *testing.T) {
	v4 := serveFixture(t, []byte(asnHeader+
		" 192.0.2.0/25 , 64496 ,\"Example, Inc.\"\n"+
		"198.51.100.0/24,64497,Example\n"+
		"203.0.113.0/24,64498,Other\n"))
	v6 := writeFixture(t, "blocks.csv", []byte(asnHeader+"2001:db8::/32,64496,Example\n"))
	tests := []struct {
		name    string
		action  lib.Action
		wanted  lib.InputOption
		only    lib.IPType
		initial map[string][]string
		want    map[string][]string
	}{
		{"all ASNs", lib.ActionAdd, nil, "", nil, map[string][]string{
			"AS64496": {"192.0.2.0/25", "2001:db8::/32"}, "AS64497": {"198.51.100.0/24"}, "AS64498": {"203.0.113.0/24"},
		}},
		{"one ASN in multiple custom lists", lib.ActionAdd, WithGeoLite2ASNCSVWantedList(map[string][]uint{
			"group": {64496, 64497}, "single": {64496},
		}), lib.IPv4, nil, map[string][]string{"GROUP": {"192.0.2.0/25", "198.51.100.0/24"}, "SINGLE": {"192.0.2.0/25"}}},
		{"remove only IPv6", lib.ActionRemove, WithGeoLite2ASNCSVWantedList("AS64496"), lib.IPv6, map[string][]string{
			"AS64496": {"192.0.2.0/25", "2001:db8::/32", "2001:db9::1/128"},
		}, map[string][]string{"AS64496": {"192.0.2.0/25", "2001:db9::1/128"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in, err := NewGeoLite2ASNCSVIn(tt.action, WithIPv4File(v4), WithIPv6File(v6), tt.wanted, WithInputOnlyIPType(tt.only))
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
	t.Run("action errors", func(t *testing.T) {
		testInputActionErrors(t, func(action lib.Action) (lib.InputConverter, error) {
			return NewGeoLite2ASNCSVIn(action, WithIPv4File(v4))
		})
	})
}

func TestCountryCSVLocations(t *testing.T) {
	path := writeFixture(t, "locations.csv", []byte(locationsHeader+
		" 42 ,en,EU,Europe, de \n23,en,AS,Asia,JP\n,en,AS,Asia,KR\n100,en,AS,Asia,  \n"))
	g := geolite2_country_csv_in{CountryCodeFile: path}
	got, err := g.getCountryCode()
	if err != nil {
		t.Fatal(err)
	}
	if want := (map[string]string{"42": "DE", "23": "JP"}); !maps.Equal(got, want) {
		t.Errorf("locations = %v, want %v", got, want)
	}
	for _, tt := range []struct{ name, data, message string }{
		{"empty file", "", "empty country code file"},
		{"no usable IDs", locationsHeader + "1,en,EU,Europe,\n", "invalid country code data"},
		{"too few columns", "id,locale\n1,en\n", "invalid record"},
		{"malformed CSV", locationsHeader + "\"unclosed", csv.ErrQuote.Error()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := geolite2_country_csv_in{CountryCodeFile: writeFixture(t, "locations.csv", []byte(tt.data))}
			if _, err := g.getCountryCode(); err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Errorf("got %v, want error containing %q", err, tt.message)
			}
		})
	}
}

func TestCSVInputErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.csv")
	locations := writeFixture(t, "locations.csv", []byte(locationsHeader+"1,en,EU,Europe,DE\n"))
	country := func(loc, v4, v6 string) *geolite2_country_csv_in {
		return &geolite2_country_csv_in{Action: lib.ActionAdd, CountryCodeFile: loc, IPv4File: v4, IPv6File: v6}
	}
	asn := func(v4, v6 string) *geolite2_asn_csv_in {
		return &geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: v4, IPv6File: v6}
	}
	tests := []struct {
		name    string
		in      lib.InputConverter
		cause   error
		message string
	}{
		{"missing locations", country(missing, "", ""), os.ErrNotExist, ""},
		{"missing country IPv4", country(locations, missing, ""), os.ErrNotExist, ""},
		{"missing country IPv6", country(locations, "", missing), os.ErrNotExist, ""},
		{"missing ASN IPv4", asn(missing, ""), os.ErrNotExist, ""},
		{"missing ASN IPv6", asn("", missing), os.ErrNotExist, ""},
		{"malformed country blocks", country(locations, writeFixture(t, "bad.csv", []byte(countryHeader+"\"")), ""), csv.ErrQuote, ""},
		{"malformed ASN blocks", asn("", writeFixture(t, "bad.csv", []byte(asnHeader+"\""))), csv.ErrQuote, ""},
		{"short country record", country(locations, writeFixture(t, "short.csv", []byte("network,id\n192.0.2.0/24,1\n")), ""), nil, "invalid record"},
		{"short ASN record", asn(writeFixture(t, "short.csv", []byte("network\n192.0.2.0/24\n")), ""), nil, "invalid record"},
		{"bad country CIDR", country(locations, writeFixture(t, "bad.csv", []byte(countryHeader+"bad/24,1,,\n")), ""), lib.ErrInvalidCIDR, ""},
		{"bad ASN CIDR", asn(writeFixture(t, "bad.csv", []byte(asnHeader+"bad/24,64496,Example\n")), ""), lib.ErrInvalidCIDR, ""},
		{"unmatched country", country(locations, writeFixture(t, "unknown.csv", []byte(countryHeader+"192.0.2.0/24,999,,\n")), ""), nil, "no entry is generated"},
		{"empty ASN blocks", asn(writeFixture(t, "empty.csv", []byte(asnHeader)), ""), nil, "no entry is generated"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.in.Input(lib.NewContainer())
			if got != nil || err == nil || (tt.cause != nil && !errors.Is(err, tt.cause)) || !strings.Contains(err.Error(), tt.message) {
				t.Errorf("Input = (%v, %v), want nil and error %v %q", got, err, tt.cause, tt.message)
			}
		})
	}
}

func TestCSVProcessErrors(t *testing.T) {
	t.Run("country needs locations", func(t *testing.T) {
		g := geolite2_country_csv_in{}
		if err := g.process("", nil, nil); err == nil || !strings.Contains(err.Error(), "invalid country code data") {
			t.Errorf("got %v, want invalid country code data", err)
		}
	})
	t.Run("country invalid prefix with nil entries", func(t *testing.T) {
		g := geolite2_country_csv_in{}
		path := writeFixture(t, "blocks.csv", []byte(countryHeader+"bad/24,1,,\n"))
		if err := g.process(path, map[string]string{"1": "DE"}, nil); !errors.Is(err, lib.ErrInvalidCIDR) {
			t.Errorf("got %v, want ErrInvalidCIDR", err)
		}
	})
	t.Run("invalid prefix in wanted ASN", func(t *testing.T) {
		g := geolite2_asn_csv_in{Want: map[string][]string{"64496": {"CUSTOM"}}}
		path := writeFixture(t, "asn.csv", []byte(asnHeader+"bad/24,64496,Example\n"))
		if err := g.process(path, nil); !errors.Is(err, lib.ErrInvalidCIDR) {
			t.Errorf("got %v, want ErrInvalidCIDR", err)
		}
	})
}
