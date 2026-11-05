package maxmind

import (
	"reflect"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestNewGeoLite2CountryCSVIn(t *testing.T) {
	tests := []struct {
		name string
		opts []lib.InputOption
		want *geolite2_country_csv_in
	}{
		{
			name: "Defaults",
			opts: []lib.InputOption{nil},
			want: &geolite2_country_csv_in{
				Type: TypeGeoLite2CountryCSVIn, Action: lib.ActionAdd, Description: DescGeoLite2CountryCSVIn,
				CountryCodeFile: defaultGeoLite2CountryCodeFile,
				IPv4File:        defaultGeoLite2CountryIPv4File, IPv6File: defaultGeoLite2CountryIPv6File,
			},
		},
		{
			name: "Custom files",
			opts: []lib.InputOption{WithCountryCodeFile(" cc.csv "), WithIPv6File("v6.csv")},
			want: &geolite2_country_csv_in{
				Type: TypeGeoLite2CountryCSVIn, Action: lib.ActionAdd, Description: DescGeoLite2CountryCSVIn,
				CountryCodeFile: "cc.csv", IPv6File: "v6.csv",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGeoLite2CountryCSVIn(lib.ActionAdd, tt.opts...)
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
			if got.GetType() != tt.want.Type || got.GetAction() != tt.want.Action || got.GetDescription() != tt.want.Description {
				t.Errorf("getter mismatch")
			}
		})
	}
}

func TestNewGeoLite2CountryCSVInFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *geolite2_country_csv_in
		wantErr bool
	}{
		{
			name: "Empty data",
			want: &geolite2_country_csv_in{
				Type: TypeGeoLite2CountryCSVIn, Action: lib.ActionAdd, Description: DescGeoLite2CountryCSVIn,
				CountryCodeFile: defaultGeoLite2CountryCodeFile,
				IPv4File:        defaultGeoLite2CountryIPv4File, IPv6File: defaultGeoLite2CountryIPv6File,
				Want: map[string]bool{},
			},
		},
		{
			name: "Full data",
			data: []byte(`{"country":"cc.csv","ipv4":"v4.csv","ipv6":"v6.csv","wantedList":["cn"],"onlyIPType":"ipv6"}`),
			want: &geolite2_country_csv_in{
				Type: TypeGeoLite2CountryCSVIn, Action: lib.ActionAdd, Description: DescGeoLite2CountryCSVIn,
				CountryCodeFile: "cc.csv", IPv4File: "v4.csv", IPv6File: "v6.csv",
				Want: map[string]bool{"CN": true}, OnlyIPType: lib.IPv6,
			},
		},
		{name: "Invalid JSON", data: []byte(`{`), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGeoLite2CountryCSVInFromBytes(lib.ActionAdd, tt.data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

const countryCodeCSV = `geoname_id,locale_code,continent_code,continent_name,country_iso_code,country_name,is_in_european_union
1814991,en,AS,Asia,cn,China,0
6252001,en,NA,"North America",US,"United States",0
6255148,en,EU,Europe,,,0
,en,EU,Europe,DE,Germany,1
`

const countryIPv4CSV = `network,geoname_id,registered_country_geoname_id,represented_country_geoname_id,is_anonymous_proxy,is_satellite_provider
1.0.1.0/24,1814991,1814991,,0,0
1.0.2.0/23,6255148,1814991,,0,0
8.8.8.0/24,,,6252001,0,0
9.9.9.0/24,,,,0,0
`

const countryIPv6CSV = `network,geoname_id,registered_country_geoname_id,represented_country_geoname_id,is_anonymous_proxy,is_satellite_provider
2001:DA8::/32,1814991,1814991,,0,0
2600::/16,6252001,6252001,,0,0
`

func TestGeoLite2CountryCSVIn_Input(t *testing.T) {
	cc := writeTempFile(t, "cc.csv", []byte(countryCodeCSV))
	v4 := writeTempFile(t, "v4.csv", []byte(countryIPv4CSV))
	v6 := writeTempFile(t, "v6.csv", []byte(countryIPv6CSV))
	srv := newFileServer(t, map[string][]byte{"/cc.csv": []byte(countryCodeCSV), "/v4.csv": []byte(countryIPv4CSV)})

	emptyFile := writeTempFile(t, "empty.csv", []byte(""))
	ccShort := writeTempFile(t, "ccshort.csv", []byte("a,b,c\n1,2,3\n"))
	ccNoData := writeTempFile(t, "ccnodata.csv", []byte("a,b,c,d,e\n1,2,3,4,\n"))
	ccInconsistent := writeTempFile(t, "ccinc.csv", []byte("a,b,c,d,e\n1,2\n"))
	ipShort := writeTempFile(t, "ipshort.csv", []byte("a,b,c\n1.0.0.0/24,1814991,1\n"))
	ipInconsistent := writeTempFile(t, "ipinc.csv", []byte("a,b,c,d\n1.0.0.0/24\n"))
	ipBadCIDR := writeTempFile(t, "ipbad.csv", []byte("a,b,c,d\nbad,1814991,,\n"))

	tests := []struct {
		name      string
		g         *geolite2_country_csv_in
		container lib.Container
		want      map[string][]string
		wantErr   bool
	}{
		{
			name: "Add all",
			g:    &geolite2_country_csv_in{Action: lib.ActionAdd, CountryCodeFile: cc, IPv4File: v4, IPv6File: v6},
			want: map[string][]string{
				"CN": {"1.0.1.0/24", "1.0.2.0/23", "2001:da8::/32"},
				"US": {"8.8.8.0/24", "2600::/16"},
			},
		},
		{
			name: "Wanted list",
			g:    &geolite2_country_csv_in{Action: lib.ActionAdd, CountryCodeFile: cc, IPv4File: v4, IPv6File: v6, Want: map[string]bool{"US": true}},
			want: map[string][]string{"US": {"8.8.8.0/24", "2600::/16"}},
		},
		{
			name: "Only IPv6",
			g:    &geolite2_country_csv_in{Action: lib.ActionAdd, CountryCodeFile: cc, IPv4File: v4, IPv6File: v6, OnlyIPType: lib.IPv6},
			want: map[string][]string{"CN": {"2001:da8::/32"}, "US": {"2600::/16"}},
		},
		{
			name: "Remote files",
			g:    &geolite2_country_csv_in{Action: lib.ActionAdd, CountryCodeFile: srv.URL + "/cc.csv", IPv4File: srv.URL + "/v4.csv"},
			want: map[string][]string{"CN": {"1.0.1.0/24", "1.0.2.0/23"}, "US": {"8.8.8.0/24"}},
		},
		{
			name:      "Remove",
			g:         &geolite2_country_csv_in{Action: lib.ActionRemove, CountryCodeFile: cc, IPv4File: v4, Want: map[string]bool{"CN": true}},
			container: newContainerWith(t, map[string][]string{"CN": {"1.0.0.0/22", "2001:da8::/32"}}),
			want:      map[string][]string{"CN": {"1.0.0.0/24", "2001:da8::/32"}},
		},
		{
			name:      "Remove non-existent entry",
			g:         &geolite2_country_csv_in{Action: lib.ActionRemove, CountryCodeFile: cc, IPv4File: v4},
			container: lib.NewContainer(),
			wantErr:   true,
		},
		{name: "Unknown action", g: &geolite2_country_csv_in{Action: "unknown", CountryCodeFile: cc, IPv4File: v4}, wantErr: true},
		{name: "No entry generated", g: &geolite2_country_csv_in{Action: lib.ActionAdd, CountryCodeFile: cc, IPv4File: v4, Want: map[string]bool{"JP": true}}, wantErr: true},
		{name: "Country code file not found", g: &geolite2_country_csv_in{Action: lib.ActionAdd, CountryCodeFile: "/non/existent", IPv4File: v4}, wantErr: true},
		{name: "Remote country code file 404", g: &geolite2_country_csv_in{Action: lib.ActionAdd, CountryCodeFile: srv.URL + "/missing", IPv4File: v4}, wantErr: true},
		{name: "Empty country code file", g: &geolite2_country_csv_in{Action: lib.ActionAdd, CountryCodeFile: emptyFile, IPv4File: v4}, wantErr: true},
		{name: "Short country code record", g: &geolite2_country_csv_in{Action: lib.ActionAdd, CountryCodeFile: ccShort, IPv4File: v4}, wantErr: true},
		{name: "No country code data", g: &geolite2_country_csv_in{Action: lib.ActionAdd, CountryCodeFile: ccNoData, IPv4File: v4}, wantErr: true},
		{name: "Inconsistent country code file", g: &geolite2_country_csv_in{Action: lib.ActionAdd, CountryCodeFile: ccInconsistent, IPv4File: v4}, wantErr: true},
		{name: "IPv4 file not found", g: &geolite2_country_csv_in{Action: lib.ActionAdd, CountryCodeFile: cc, IPv4File: "/non/existent"}, wantErr: true},
		{name: "IPv6 file not found", g: &geolite2_country_csv_in{Action: lib.ActionAdd, CountryCodeFile: cc, IPv6File: "/non/existent"}, wantErr: true},
		{name: "Remote IP file 404", g: &geolite2_country_csv_in{Action: lib.ActionAdd, CountryCodeFile: cc, IPv4File: srv.URL + "/missing"}, wantErr: true},
		{name: "Short IP record", g: &geolite2_country_csv_in{Action: lib.ActionAdd, CountryCodeFile: cc, IPv4File: ipShort}, wantErr: true},
		{name: "Inconsistent IP file", g: &geolite2_country_csv_in{Action: lib.ActionAdd, CountryCodeFile: cc, IPv4File: ipInconsistent}, wantErr: true},
		{name: "Invalid CIDR", g: &geolite2_country_csv_in{Action: lib.ActionAdd, CountryCodeFile: cc, IPv4File: ipBadCIDR}, wantErr: true},
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

func TestGeoLite2CountryCSVIn_Process(t *testing.T) {
	v4 := writeTempFile(t, "v4.csv", []byte(countryIPv4CSV))
	tests := []struct {
		name    string
		ccMap   map[string]string
		wantErr bool
	}{
		{name: "Empty country code map", ccMap: nil, wantErr: true},
		{name: "Nil entries map", ccMap: map[string]string{"1814991": "CN"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_country_csv_in{}
			err := g.process(v4, tt.ccMap, nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
