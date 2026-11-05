package maxmind

import (
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

func countryRecord(field, isoCode string) mmdbtype.Map {
	return mmdbtype.Map{
		mmdbtype.String(field): mmdbtype.Map{"iso_code": mmdbtype.String(isoCode)},
	}
}

func TestGeoLite2CountryMMDBIn_Getters(t *testing.T) {
	g := &geolite2_country_mmdb_in{Type: "t", Action: lib.ActionAdd, Description: "d"}
	if g.GetType() != "t" || g.GetAction() != lib.ActionAdd || g.GetDescription() != "d" {
		t.Errorf("getter mismatch: %+v", g)
	}
}

func TestGeoLite2CountryMMDBIn_Input(t *testing.T) {
	countryDB := buildMMDB(t, "GeoLite2-Country", 24, []mmdbRecord{
		{"1.0.1.0/24", countryRecord("country", " cn ")},
		{"1.0.2.0/23", countryRecord("registered_country", "cn")},
		{"8.8.8.0/24", countryRecord("represented_country", "us")},
		{"9.9.9.0/24", mmdbtype.Map{"continent": mmdbtype.Map{"code": mmdbtype.String("EU")}}},
		{"2001:da8::/32", countryRecord("country", "CN")},
		{"2600::/16", countryRecord("country", "US")},
	})
	ipinfoDB := buildMMDB(t, "IPInfo-Lite", 32, []mmdbRecord{
		{"1.0.1.0/24", mmdbtype.Map{"country_code": mmdbtype.String("cn")}},
		{"9.9.9.0/24", mmdbtype.Map{"country_code": mmdbtype.String("")}},
		{"2600::/16", mmdbtype.Map{"country_code": mmdbtype.String("US")}},
	})
	badCountryDB := buildMMDB(t, "GeoLite2-Country", 24, []mmdbRecord{
		{"1.0.1.0/24", mmdbtype.Map{"country": mmdbtype.String("not a map")}},
	})
	badIPInfoDB := buildMMDB(t, "IPInfo-Lite", 32, []mmdbRecord{
		{"1.0.1.0/24", mmdbtype.Map{"country_code": mmdbtype.Map{}}},
	})

	countryFile := writeTempFile(t, "country.mmdb", countryDB)
	ipinfoFile := writeTempFile(t, "ipinfo.mmdb", ipinfoDB)
	badCountryFile := writeTempFile(t, "bad.mmdb", badCountryDB)
	badIPInfoFile := writeTempFile(t, "badipinfo.mmdb", badIPInfoDB)
	garbageFile := writeTempFile(t, "garbage.mmdb", []byte("not a mmdb"))
	srv := newFileServer(t, map[string][]byte{"/country.mmdb": countryDB})

	allCountry := map[string][]string{
		"CN": {"1.0.1.0/24", "1.0.2.0/23", "2001:da8::/32"},
		"US": {"8.8.8.0/24", "2600::/16"},
	}

	tests := []struct {
		name      string
		g         *geolite2_country_mmdb_in
		container lib.Container
		want      map[string][]string
		wantErr   bool
	}{
		{name: "GeoLite2 add all", g: &geolite2_country_mmdb_in{Type: TypeGeoLite2CountryMMDBIn, Action: lib.ActionAdd, URI: countryFile}, want: allCountry},
		{name: "DBIP add all", g: &geolite2_country_mmdb_in{Type: TypeDBIPCountryMMDBIn, Action: lib.ActionAdd, URI: countryFile}, want: allCountry},
		{name: "Remote URI", g: &geolite2_country_mmdb_in{Type: TypeGeoLite2CountryMMDBIn, Action: lib.ActionAdd, URI: srv.URL + "/country.mmdb"}, want: allCountry},
		{
			name: "Wanted list",
			g:    &geolite2_country_mmdb_in{Type: TypeGeoLite2CountryMMDBIn, Action: lib.ActionAdd, URI: countryFile, Want: map[string]bool{"US": true}},
			want: map[string][]string{"US": {"8.8.8.0/24", "2600::/16"}},
		},
		{
			name: "Only IPv4",
			g:    &geolite2_country_mmdb_in{Type: TypeGeoLite2CountryMMDBIn, Action: lib.ActionAdd, URI: countryFile, OnlyIPType: lib.IPv4},
			want: map[string][]string{"CN": {"1.0.1.0/24", "1.0.2.0/23"}, "US": {"8.8.8.0/24"}},
		},
		{
			name: "IPInfo add all",
			g:    &geolite2_country_mmdb_in{Type: TypeIPInfoCountryMMDBIn, Action: lib.ActionAdd, URI: ipinfoFile},
			want: map[string][]string{"CN": {"1.0.1.0/24"}, "US": {"2600::/16"}},
		},
		{
			name:      "Remove",
			g:         &geolite2_country_mmdb_in{Type: TypeGeoLite2CountryMMDBIn, Action: lib.ActionRemove, URI: countryFile, Want: map[string]bool{"CN": true}},
			container: newContainerWith(t, map[string][]string{"CN": {"1.0.0.0/22", "2001:da8::/31"}}),
			want:      map[string][]string{"CN": {"1.0.0.0/24", "2001:da9::/32"}},
		},
		{
			name:      "Remove non-existent entry",
			g:         &geolite2_country_mmdb_in{Type: TypeGeoLite2CountryMMDBIn, Action: lib.ActionRemove, URI: countryFile},
			container: lib.NewContainer(),
			wantErr:   true,
		},
		{name: "Unknown action", g: &geolite2_country_mmdb_in{Type: TypeGeoLite2CountryMMDBIn, Action: "unknown", URI: countryFile}, wantErr: true},
		{name: "Unsupported type", g: &geolite2_country_mmdb_in{Type: "unknown", Action: lib.ActionAdd, URI: countryFile}, wantErr: true},
		{name: "No entry generated", g: &geolite2_country_mmdb_in{Type: TypeGeoLite2CountryMMDBIn, Action: lib.ActionAdd, URI: countryFile, Want: map[string]bool{"JP": true}}, wantErr: true},
		{name: "File not found", g: &geolite2_country_mmdb_in{Type: TypeGeoLite2CountryMMDBIn, Action: lib.ActionAdd, URI: "/non/existent"}, wantErr: true},
		{name: "Remote 404", g: &geolite2_country_mmdb_in{Type: TypeGeoLite2CountryMMDBIn, Action: lib.ActionAdd, URI: srv.URL + "/missing"}, wantErr: true},
		{name: "Invalid MMDB", g: &geolite2_country_mmdb_in{Type: TypeGeoLite2CountryMMDBIn, Action: lib.ActionAdd, URI: garbageFile}, wantErr: true},
		{name: "Country decode error", g: &geolite2_country_mmdb_in{Type: TypeGeoLite2CountryMMDBIn, Action: lib.ActionAdd, URI: badCountryFile}, wantErr: true},
		{name: "IPInfo decode error", g: &geolite2_country_mmdb_in{Type: TypeIPInfoCountryMMDBIn, Action: lib.ActionAdd, URI: badIPInfoFile}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.container
			if c == nil {
				c = lib.NewContainer()
			}
			got, err := tt.g.Input(c)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			assertCIDRs(t, containerCIDRs(t, got), tt.want)
		})
	}
}
