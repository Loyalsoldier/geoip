package maxmind

import (
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

func geoliteFixture(t *testing.T) []byte {
	return buildMMDB(t,
		mmdbRecord{"1.0.0.0/24", mmdbtype.Map{
			"country":             mmdbtype.Map{"iso_code": mmdbtype.String("cn")},
			"registered_country":  mmdbtype.Map{"iso_code": mmdbtype.String("us")},
			"represented_country": mmdbtype.Map{"iso_code": mmdbtype.String("jp")},
		}},
		mmdbRecord{"1.0.1.0/24", countryRec("country", "CN")},
		mmdbRecord{"2.0.0.0/24", mmdbtype.Map{
			"registered_country":  mmdbtype.Map{"iso_code": mmdbtype.String("us")},
			"represented_country": mmdbtype.Map{"iso_code": mmdbtype.String("jp")},
		}},
		mmdbRecord{"3.0.0.0/24", countryRec("represented_country", "jp")},
		mmdbRecord{"4.0.0.0/24", mmdbtype.Map{"continent": mmdbtype.Map{"code": mmdbtype.String("AS")}}},
		mmdbRecord{"2001:db8::/32", countryRec("country", "CN")},
	)
}

func ipinfoFixture(t *testing.T) []byte {
	return buildMMDB(t,
		mmdbRecord{"1.0.0.0/24", mmdbtype.Map{"country_code": mmdbtype.String("cn")}},
		mmdbRecord{"2.0.0.0/24", mmdbtype.Map{"country_code": mmdbtype.String("US")}},
		mmdbRecord{"4.0.0.0/24", mmdbtype.Map{"asn": mmdbtype.String("AS1")}},
		mmdbRecord{"2001:db8::/32", mmdbtype.Map{"country_code": mmdbtype.String("US")}},
	)
}

func TestMMDBIn_Input(t *testing.T) {
	dir := t.TempDir()
	geoFile := writeFile(t, dir, "geo.mmdb", geoliteFixture(t))
	ipinfoFile := writeFile(t, dir, "ipinfo.mmdb", ipinfoFixture(t))
	badDecodeFile := writeFile(t, dir, "bad.mmdb", badCountryMMDB(t))
	badIPInfoFile := writeFile(t, dir, "badipinfo.mmdb", badIPInfoMMDB(t))
	badMetadata := writeFile(t, dir, "metadata.mmdb", unverifiedMMDB(t))
	emptyFile := writeFile(t, dir, "empty.mmdb", buildMMDB(t))
	garbageFile := writeFile(t, dir, "garbage.mmdb", []byte("not a mmdb"))
	srv := newServer(t, map[string][]byte{"/geo.mmdb": geoliteFixture(t)})

	tests := []struct {
		name    string
		iType   string
		uri     string
		action  lib.Action
		want    []string
		only    lib.IPType
		pre     map[string][]string // entries pre-populated into container
		failing bool
		wantErr bool
		expect  map[string][]string
	}{
		{
			name: "maxmind all", iType: TypeGeoLite2CountryMMDBIn, uri: geoFile, action: lib.ActionAdd,
			expect: map[string][]string{
				"CN": {"1.0.0.0/23", "2001:db8::/32"},
				"US": {"2.0.0.0/24"},
				"JP": {"3.0.0.0/24"},
			},
		},
		{
			name: "dbip wanted ipv4 only", iType: TypeDBIPCountryMMDBIn, uri: geoFile, action: lib.ActionAdd,
			want: []string{"cn"}, only: lib.IPv4,
			expect: map[string][]string{"CN": {"1.0.0.0/23"}},
		},
		{
			name: "remote uri", iType: TypeGeoLite2CountryMMDBIn, uri: srv.URL + "/geo.mmdb", action: lib.ActionAdd,
			want:   []string{"US"},
			expect: map[string][]string{"US": {"2.0.0.0/24"}},
		},
		{
			name: "ipinfo all", iType: TypeIPInfoCountryMMDBIn, uri: ipinfoFile, action: lib.ActionAdd,
			expect: map[string][]string{
				"CN": {"1.0.0.0/24"},
				"US": {"2.0.0.0/24", "2001:db8::/32"},
			},
		},
		{
			name: "remove", iType: TypeGeoLite2CountryMMDBIn, uri: geoFile, action: lib.ActionRemove,
			want: []string{"CN"},
			pre:  map[string][]string{"CN": {"1.0.0.0/16", "2001:db8::/32"}},
			expect: map[string][]string{
				"CN": {"1.0.2.0/23", "1.0.4.0/22", "1.0.8.0/21", "1.0.16.0/20", "1.0.32.0/19", "1.0.64.0/18", "1.0.128.0/17"},
			},
		},
		{name: "remove entry not found", iType: TypeGeoLite2CountryMMDBIn, uri: geoFile, action: lib.ActionRemove, wantErr: true},
		{name: "add fails", iType: TypeGeoLite2CountryMMDBIn, uri: geoFile, action: lib.ActionAdd, failing: true, wantErr: true},
		{name: "unknown action", iType: TypeGeoLite2CountryMMDBIn, uri: geoFile, action: lib.Action("unknown"), wantErr: true},
		{name: "file not exist", iType: TypeGeoLite2CountryMMDBIn, uri: dir + "/none.mmdb", action: lib.ActionAdd, wantErr: true},
		{name: "remote 404", iType: TypeGeoLite2CountryMMDBIn, uri: srv.URL + "/none", action: lib.ActionAdd, wantErr: true},
		{name: "garbage", iType: TypeGeoLite2CountryMMDBIn, uri: garbageFile, action: lib.ActionAdd, wantErr: true},
		{name: "invalid metadata", iType: TypeGeoLite2CountryMMDBIn, uri: badMetadata, action: lib.ActionAdd, wantErr: true},
		{name: "empty db", iType: TypeGeoLite2CountryMMDBIn, uri: emptyFile, action: lib.ActionAdd, wantErr: true},
		{name: "no wanted entry", iType: TypeGeoLite2CountryMMDBIn, uri: geoFile, action: lib.ActionAdd, want: []string{"XX"}, wantErr: true},
		{name: "decode error", iType: TypeGeoLite2CountryMMDBIn, uri: badDecodeFile, action: lib.ActionAdd, wantErr: true},
		{name: "ipinfo decode error", iType: TypeIPInfoCountryMMDBIn, uri: badIPInfoFile, action: lib.ActionAdd, wantErr: true},
		{name: "unsupported type", iType: "unknown", uri: geoFile, action: lib.ActionAdd, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_country_mmdb_in{Type: tt.iType, Action: tt.action, URI: tt.uri, OnlyIPType: tt.only}
			WithInputWantedList(tt.want)(g)

			container := lib.NewContainer()
			for name, cidrs := range tt.pre {
				e := lib.NewEntry(name)
				for _, c := range cidrs {
					if err := e.AddPrefix(c); err != nil {
						t.Fatal(err)
					}
				}
				if err := container.Add(e); err != nil {
					t.Fatal(err)
				}
			}
			if tt.failing {
				container = &failingContainer{container}
			}

			got, err := g.Input(container)
			assertErr(t, err, tt.wantErr)
			if tt.wantErr {
				return
			}
			assertContainer(t, got, tt.expect)
		})
	}
}
