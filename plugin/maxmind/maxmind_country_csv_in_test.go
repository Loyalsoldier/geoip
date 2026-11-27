package maxmind

import (
	"maps"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	testCountryCodeCSV = `geoname_id,locale_code,continent_code,continent_name,country_iso_code,country_name,is_in_european_union
6255147,en,AS,Asia,,,0
1814991,en,AS,Asia,cn,China,0
6252001,en,NA,"North America",US,"United States",0
1861060,en,AS,Asia,JP,Japan,0
,en,AS,Asia,KR,Korea,0
`
	testCountryIPv4CSV = `network,geoname_id,registered_country_geoname_id,represented_country_geoname_id,is_anonymous_proxy,is_satellite_provider
1.0.0.0/24,1814991,6252001,1861060,0,0
1.0.1.0/24,6255147,1814991,1861060,0,0
2.0.0.0/24,,,6252001,0,0
3.0.0.0/24,1861060,,,0,0
4.0.0.0/24,6255147,,,0,0
`
	testCountryIPv6CSV = `network,geoname_id,registered_country_geoname_id,represented_country_geoname_id,is_anonymous_proxy,is_satellite_provider
2001:DB8::/32,1814991,,,0,0
`
)

func TestNewGeoLite2CountryCSVIn(t *testing.T) {
	tests := []struct {
		name     string
		opts     []lib.InputOption
		wantCC   string
		wantV4   string
		wantV6   string
		wantWant map[string]bool
		wantOnly lib.IPType
	}{
		{"default", nil, defaultGeoLite2CountryCodeFile, defaultGeoLite2CountryIPv4File, defaultGeoLite2CountryIPv6File, nil, ""},
		{"only ipv4 file", []lib.InputOption{nil, WithIPv4File(" v4.csv ")}, defaultGeoLite2CountryCodeFile, "v4.csv", "", nil, ""},
		{
			"all options",
			[]lib.InputOption{WithCountryCodeFile(" cc.csv "), WithIPv6File("v6.csv"), WithInputWantedList([]string{"cn"}), WithInputOnlyIPType(lib.IPv6)},
			"cc.csv", "", "v6.csv", map[string]bool{"CN": true}, lib.IPv6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewGeoLite2CountryCSVIn(lib.ActionAdd, tt.opts...)
			assertErr(t, err, false)
			g := conv.(*geolite2_country_csv_in)
			if g.GetType() != TypeGeoLite2CountryCSVIn || g.GetAction() != lib.ActionAdd || g.GetDescription() != DescGeoLite2CountryCSVIn {
				t.Errorf("unexpected meta: %+v", g)
			}
			if g.CountryCodeFile != tt.wantCC || g.IPv4File != tt.wantV4 || g.IPv6File != tt.wantV6 || g.OnlyIPType != tt.wantOnly {
				t.Errorf("got %+v", g)
			}
			if !maps.Equal(g.Want, tt.wantWant) {
				t.Errorf("Want = %v, want %v", g.Want, tt.wantWant)
			}
		})
	}
}

func TestNewGeoLite2CountryCSVInFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantCC  string
		wantV4  string
		wantV6  string
		want    map[string]bool
		only    lib.IPType
		wantErr bool
	}{
		{"empty", "", defaultGeoLite2CountryCodeFile, defaultGeoLite2CountryIPv4File, defaultGeoLite2CountryIPv6File, nil, "", false},
		{"custom", `{"country":"cc.csv","ipv4":"v4.csv","ipv6":"v6.csv","wantedList":["cn"],"onlyIPType":"ipv4"}`, "cc.csv", "v4.csv", "v6.csv", map[string]bool{"CN": true}, lib.IPv4, false},
		{"invalid", `[`, "", "", "", nil, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewGeoLite2CountryCSVInFromBytes(lib.ActionAdd, []byte(tt.data))
			assertErr(t, err, tt.wantErr)
			if tt.wantErr {
				return
			}
			g := conv.(*geolite2_country_csv_in)
			if g.CountryCodeFile != tt.wantCC || g.IPv4File != tt.wantV4 || g.IPv6File != tt.wantV6 {
				t.Errorf("got %+v", g)
			}
			if !maps.Equal(g.Want, tt.want) || g.OnlyIPType != tt.only {
				t.Errorf("got wanted list %v, IP type %q; want %v, %q", g.Want, g.OnlyIPType, tt.want, tt.only)
			}
		})
	}
}

func TestGeoLite2CountryCSVIn_Input(t *testing.T) {
	dir := t.TempDir()
	cc := writeFile(t, dir, "cc.csv", []byte(testCountryCodeCSV))
	v4 := writeFile(t, dir, "v4.csv", []byte(testCountryIPv4CSV))
	v6 := writeFile(t, dir, "v6.csv", []byte(testCountryIPv6CSV))
	emptyCC := writeFile(t, dir, "empty.csv", nil)
	shortCC := writeFile(t, dir, "short.csv", []byte("h1,h2\na,b\n"))
	noCodeCC := writeFile(t, dir, "nocode.csv", []byte("h1,h2,h3,h4,h5\n1,en,AS,Asia,\n"))
	badQuoteCC := writeFile(t, dir, "badquote.csv", []byte("a,\"b\n"))
	shortV4 := writeFile(t, dir, "shortv4.csv", []byte("h1,h2\n1.0.0.0/24,1814991\n"))
	badCIDR := writeFile(t, dir, "badcidr.csv", []byte("h1,h2,h3,h4\nnot-cidr,1814991,,\n"))
	badQuoteV4 := writeFile(t, dir, "badquotev4.csv", []byte("h\n1.0.0.0/24,\"1814991\n"))
	srv := newServer(t, map[string][]byte{
		"/cc.csv": []byte(testCountryCodeCSV),
		"/v4.csv": []byte(testCountryIPv4CSV),
	})
	missing := dir + "/missing.csv"

	tests := []struct {
		name    string
		cc      string
		v4, v6  string
		want    []string
		only    lib.IPType
		action  lib.Action
		pre     map[string][]string
		failing bool
		wantErr bool
		expect  map[string][]string
	}{
		{
			name: "all", cc: cc, v4: v4, v6: v6, action: lib.ActionAdd,
			expect: map[string][]string{
				"CN": {"1.0.0.0/23", "2001:db8::/32"},
				"US": {"2.0.0.0/24"},
				"JP": {"3.0.0.0/24"},
			},
		},
		{
			name: "wanted ipv6 only", cc: cc, v4: v4, v6: v6, action: lib.ActionAdd, want: []string{"CN"}, only: lib.IPv6,
			expect: map[string][]string{"CN": {"2001:db8::/32"}},
		},
		{
			name: "remote", cc: srv.URL + "/cc.csv", v4: srv.URL + "/v4.csv", action: lib.ActionAdd, want: []string{"jp"},
			expect: map[string][]string{"JP": {"3.0.0.0/24"}},
		},
		{
			name: "remove", cc: cc, v4: v4, action: lib.ActionRemove, want: []string{"US"},
			pre:    map[string][]string{"US": {"2.0.0.0/23"}},
			expect: map[string][]string{"US": {"2.0.1.0/24"}},
		},
		{name: "remove not found", cc: cc, v4: v4, action: lib.ActionRemove, wantErr: true},
		{name: "add fails", cc: cc, v4: v4, action: lib.ActionAdd, failing: true, wantErr: true},
		{name: "unknown action", cc: cc, v4: v4, action: lib.Action("x"), wantErr: true},
		{name: "no entry", cc: cc, v4: v4, action: lib.ActionAdd, want: []string{"XX"}, wantErr: true},
		{name: "cc missing", cc: missing, v4: v4, action: lib.ActionAdd, wantErr: true},
		{name: "cc remote 404", cc: srv.URL + "/none", v4: v4, action: lib.ActionAdd, wantErr: true},
		{name: "cc empty", cc: emptyCC, v4: v4, action: lib.ActionAdd, wantErr: true},
		{name: "cc short record", cc: shortCC, v4: v4, action: lib.ActionAdd, wantErr: true},
		{name: "cc no code", cc: noCodeCC, v4: v4, action: lib.ActionAdd, wantErr: true},
		{name: "cc bad csv", cc: badQuoteCC, v4: v4, action: lib.ActionAdd, wantErr: true},
		{name: "v4 missing", cc: cc, v4: missing, action: lib.ActionAdd, wantErr: true},
		{name: "v6 missing", cc: cc, v4: v4, v6: missing, action: lib.ActionAdd, wantErr: true},
		{name: "v4 remote 404", cc: cc, v4: srv.URL + "/none", action: lib.ActionAdd, wantErr: true},
		{name: "v4 short record", cc: cc, v4: shortV4, action: lib.ActionAdd, wantErr: true},
		{name: "v4 bad cidr", cc: cc, v4: badCIDR, action: lib.ActionAdd, wantErr: true},
		{name: "v4 bad csv", cc: cc, v4: badQuoteV4, action: lib.ActionAdd, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_country_csv_in{
				Type: TypeGeoLite2CountryCSVIn, Action: tt.action,
				CountryCodeFile: tt.cc, IPv4File: tt.v4, IPv6File: tt.v6, OnlyIPType: tt.only,
			}
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

func TestGeoLite2CountryCSVIn_Process(t *testing.T) {
	dir := t.TempDir()
	v4 := writeFile(t, dir, "v4.csv", []byte(testCountryIPv4CSV))
	g := &geolite2_country_csv_in{}

	t.Run("empty ccMap", func(t *testing.T) {
		assertErr(t, g.process(v4, nil, map[string]*lib.Entry{}), true)
	})
	t.Run("nil entries", func(t *testing.T) {
		assertErr(t, g.process(v4, map[string]string{"1814991": "CN"}, nil), false)
	})
}
