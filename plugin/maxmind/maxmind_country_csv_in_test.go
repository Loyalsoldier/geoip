package maxmind

import (
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

const countryLocationsCSV = `geoname_id,locale_code,continent_code,continent_name,country_iso_code,country_name,is_in_european_union
1814991,en,AS,Asia,cn,China,0
6252001,en,NA,"North America",US,"United States",0
1861060,en,AS,Asia,JP,Japan,0
6255147,en,AS,Asia,,,0
,en,EU,Europe,DE,Germany,1
`

const countryIPv4CSV = `network,geoname_id,registered_country_geoname_id,represented_country_geoname_id,is_anonymous_proxy,is_satellite_provider
1.0.0.0/24,1814991,1814991,,0,0
8.8.8.0/24,,6252001,,0,0
9.9.9.0/24,6255147,,1861060,0,0
4.4.4.0/24,,,,0,0
5.5.5.0/24,999,,,0,0
`

const countryIPv6CSV = `network,geoname_id,registered_country_geoname_id,represented_country_geoname_id,is_anonymous_proxy,is_satellite_provider
2001:4860::/32,1814991,,,0,0
2400:CB00::/32,6252001,6252001,,0,0
`

func TestNewGeoLite2CountryCSVIn(t *testing.T) {
	tests := []struct {
		name        string
		opts        []lib.InputOption
		wantCountry string
		wantV4      string
		wantV6      string
		wantWant    []string
		wantOnly    lib.IPType
	}{
		{"defaults", nil, defaultGeoLite2CountryCodeFile, defaultGeoLite2CountryIPv4File, defaultGeoLite2CountryIPv6File, nil, ""},
		{"nil option", []lib.InputOption{nil}, defaultGeoLite2CountryCodeFile, defaultGeoLite2CountryIPv4File, defaultGeoLite2CountryIPv6File, nil, ""},
		{"only ipv4 file", []lib.InputOption{WithIPv4File(" v4.csv ")}, defaultGeoLite2CountryCodeFile, "v4.csv", "", nil, ""},
		{"only ipv6 file", []lib.InputOption{WithIPv6File("v6.csv")}, defaultGeoLite2CountryCodeFile, "", "v6.csv", nil, ""},
		{
			"all options",
			[]lib.InputOption{WithCountryCodeFile(" cc.csv "), WithIPv4File("v4"), WithIPv6File("v6"), WithInputWantedList([]string{"us", " "}), WithInputOnlyIPType(lib.IPv6)},
			"cc.csv", "v4", "v6", []string{"US"}, lib.IPv6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewGeoLite2CountryCSVIn(lib.ActionAdd, tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			g := conv.(*geolite2_country_csv_in)
			if g.GetType() != TypeGeoLite2CountryCSVIn || g.GetDescription() != DescGeoLite2CountryCSVIn || g.GetAction() != lib.ActionAdd {
				t.Errorf("bad metadata: %+v", g)
			}
			if g.CountryCodeFile != tt.wantCountry || g.IPv4File != tt.wantV4 || g.IPv6File != tt.wantV6 || g.OnlyIPType != tt.wantOnly {
				t.Errorf("got %+v", g)
			}
			if len(g.Want) != len(tt.wantWant) {
				t.Errorf("Want: got %v want %v", g.Want, tt.wantWant)
			}
			for _, w := range tt.wantWant {
				if !g.Want[w] {
					t.Errorf("Want missing %s", w)
				}
			}
		})
	}
}

func TestNewGeoLite2CountryCSVInFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		wantCC  string
		wantV4  string
		wantV6  string
		want    []string
		only    lib.IPType
	}{
		{"empty", "", false, defaultGeoLite2CountryCodeFile, defaultGeoLite2CountryIPv4File, defaultGeoLite2CountryIPv6File, nil, ""},
		{"full", `{"country":"c","ipv4":"4","ipv6":"6","wantedList":["jp"],"onlyIPType":"ipv4"}`, false, "c", "4", "6", []string{"JP"}, lib.IPv4},
		{"invalid", `{"country":`, true, "", "", "", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewGeoLite2CountryCSVInFromBytes(lib.ActionAdd, []byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			g := conv.(*geolite2_country_csv_in)
			if g.CountryCodeFile != tt.wantCC || g.IPv4File != tt.wantV4 || g.IPv6File != tt.wantV6 || g.OnlyIPType != tt.only || len(g.Want) != len(tt.want) {
				t.Errorf("got %+v", g)
			}
			for _, w := range tt.want {
				if !g.Want[w] {
					t.Errorf("Want missing %s", w)
				}
			}
		})
	}
}

func TestGeoLite2CountryCSVInInput(t *testing.T) {
	ccPath := writeFile(t, "cc.csv", []byte(countryLocationsCSV))
	v4Path := writeFile(t, "v4.csv", []byte(countryIPv4CSV))
	v6Path := writeFile(t, "v6.csv", []byte(countryIPv6CSV))
	emptyPath := writeFile(t, "empty.csv", nil)
	badCSVPath := writeFile(t, "bad.csv", []byte("a,b\n\"unterminated,x\n"))
	shortCCPath := writeFile(t, "shortcc.csv", []byte("a,b,c\n1,2,3\n"))
	noCCPath := writeFile(t, "nocc.csv", []byte("a,b,c,d,e\n,x,y,z,\n1,x,y,z,\n"))
	shortIPPath := writeFile(t, "shortip.csv", []byte("network,geoname_id\n1.0.0.0/24,1814991\n"))
	badCIDRPath := writeFile(t, "badcidr.csv", []byte("network,a,b,c\n300.0.0.0/24,1814991,,\n"))
	badRowPath := writeFile(t, "badrow.csv", []byte("network,a,b,c\n1.0.0.0/24,1814991,,\n1.0.1.0/24,1814991\n"))
	noMatchPath := writeFile(t, "nomatch.csv", []byte("network,a,b,c\n1.0.0.0/24,1,2,3\n"))

	srv := serveFiles(t, map[string][]byte{
		"/cc.csv": []byte(countryLocationsCSV),
		"/v4.csv": []byte(countryIPv4CSV),
		"/v6.csv": []byte(countryIPv6CSV),
	})

	all := map[string][]string{
		"CN": {"1.0.0.0/24", "2001:4860::/32"},
		"US": {"8.8.8.0/24", "2400:cb00::/32"},
		"JP": {"9.9.9.0/24"},
	}

	tests := []struct {
		name    string
		cc      string
		v4      string
		v6      string
		action  lib.Action
		want    []string
		only    lib.IPType
		pre     map[string][]string
		wantErr string
		expect  map[string][]string
	}{
		{name: "local all", cc: ccPath, v4: v4Path, v6: v6Path, action: lib.ActionAdd, expect: all},
		{name: "remote all", cc: srv.URL + "/cc.csv", v4: srv.URL + "/v4.csv", v6: srv.URL + "/v6.csv", action: lib.ActionAdd, expect: all},
		{name: "ipv4 file only", cc: ccPath, v4: v4Path, action: lib.ActionAdd,
			expect: map[string][]string{"CN": {"1.0.0.0/24"}, "US": {"8.8.8.0/24"}, "JP": {"9.9.9.0/24"}}},
		{name: "ipv6 file only", cc: ccPath, v6: v6Path, action: lib.ActionAdd,
			expect: map[string][]string{"CN": {"2001:4860::/32"}, "US": {"2400:cb00::/32"}}},
		{name: "wanted", cc: ccPath, v4: v4Path, v6: v6Path, action: lib.ActionAdd, want: []string{"us"},
			expect: map[string][]string{"US": {"8.8.8.0/24", "2400:cb00::/32"}}},
		{name: "only ipv6", cc: ccPath, v4: v4Path, v6: v6Path, action: lib.ActionAdd, only: lib.IPv6, want: []string{"CN", "US"},
			expect: map[string][]string{"CN": {"2001:4860::/32"}, "US": {"2400:cb00::/32"}}},
		{name: "remove", cc: ccPath, v4: v4Path, v6: v6Path, action: lib.ActionRemove, want: []string{"US"},
			pre:    map[string][]string{"US": {"8.8.8.0/23", "2400:cb00::/31"}},
			expect: map[string][]string{"US": {"8.8.9.0/24", "2400:cb01::/32"}}},
		{name: "remove missing", cc: ccPath, v4: v4Path, action: lib.ActionRemove, wantErr: "not found"},
		{name: "unknown action", cc: ccPath, v4: v4Path, action: "bogus", wantErr: lib.ErrUnknownAction.Error()},
		{name: "country file missing", cc: "/nonexistent/cc.csv", v4: v4Path, action: lib.ActionAdd, wantErr: "no such file"},
		{name: "country file remote 404", cc: srv.URL + "/nope.csv", v4: v4Path, action: lib.ActionAdd, wantErr: "404"},
		{name: "country file empty", cc: emptyPath, v4: v4Path, action: lib.ActionAdd, wantErr: "empty country code file"},
		{name: "country file bad csv", cc: badCSVPath, v4: v4Path, action: lib.ActionAdd, wantErr: "❌"},
		{name: "country file short record", cc: shortCCPath, v4: v4Path, action: lib.ActionAdd, wantErr: "invalid record"},
		{name: "country file no codes", cc: noCCPath, v4: v4Path, action: lib.ActionAdd, wantErr: "invalid country code data"},
		{name: "ipv4 file missing", cc: ccPath, v4: "/nonexistent/v4.csv", action: lib.ActionAdd, wantErr: "no such file"},
		{name: "ipv6 file missing", cc: ccPath, v4: v4Path, v6: "/nonexistent/v6.csv", action: lib.ActionAdd, wantErr: "no such file"},
		{name: "ipv4 file remote 404", cc: ccPath, v4: srv.URL + "/nope.csv", action: lib.ActionAdd, wantErr: "404"},
		{name: "ip file short record", cc: ccPath, v4: shortIPPath, action: lib.ActionAdd, wantErr: "invalid record"},
		{name: "ip file bad cidr", cc: ccPath, v4: badCIDRPath, action: lib.ActionAdd, wantErr: lib.ErrInvalidCIDR.Error()},
		{name: "ip file bad row", cc: ccPath, v4: badRowPath, action: lib.ActionAdd, wantErr: "wrong number of fields"},
		{name: "no entries", cc: ccPath, v4: noMatchPath, action: lib.ActionAdd, wantErr: "no entry is generated"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_country_csv_in{
				Type:            TypeGeoLite2CountryCSVIn,
				Action:          tt.action,
				CountryCodeFile: tt.cc,
				IPv4File:        tt.v4,
				IPv6File:        tt.v6,
				OnlyIPType:      tt.only,
			}
			WithInputWantedList(tt.want)(g)
			got, err := g.Input(containerWith(t, tt.pre))
			if tt.wantErr != "" {
				assertErrContains(t, err, tt.wantErr)
				return
			}
			if err != nil {
				t.Fatalf("Input: %v", err)
			}
			assertEntries(t, got, tt.expect)
		})
	}
}

func TestGeoLite2CountryCSVInProcess(t *testing.T) {
	v4Path := writeFile(t, "v4.csv", []byte(countryIPv4CSV))
	tests := []struct {
		name    string
		ccMap   map[string]string
		entries map[string]*lib.Entry
		wantErr string
	}{
		{"empty ccMap", nil, map[string]*lib.Entry{}, "invalid country code data"},
		{"nil entries map", map[string]string{"1814991": "CN"}, nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_country_csv_in{}
			err := g.process(v4Path, tt.ccMap, tt.entries)
			if tt.wantErr != "" {
				assertErrContains(t, err, tt.wantErr)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
