package maxmind

import (
	"reflect"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestNewGeoLite2ASNCSVIn(t *testing.T) {
	tests := []struct {
		name string
		opts []lib.InputOption
		want *geolite2_asn_csv_in
	}{
		{
			name: "Default files",
			opts: []lib.InputOption{nil},
			want: &geolite2_asn_csv_in{
				Type: TypeGeoLite2ASNCSVIn, Action: lib.ActionAdd, Description: DescGeoLite2ASNCSVIn,
				IPv4File: defaultGeoLite2ASNCSVIPv4File, IPv6File: defaultGeoLite2ASNCSVIPv6File,
			},
		},
		{
			name: "Only IPv4 file",
			opts: []lib.InputOption{WithIPv4File("v4.csv")},
			want: &geolite2_asn_csv_in{
				Type: TypeGeoLite2ASNCSVIn, Action: lib.ActionAdd, Description: DescGeoLite2ASNCSVIn,
				IPv4File: "v4.csv",
			},
		},
		{
			name: "Only IPv6 file",
			opts: []lib.InputOption{WithIPv6File("v6.csv")},
			want: &geolite2_asn_csv_in{
				Type: TypeGeoLite2ASNCSVIn, Action: lib.ActionAdd, Description: DescGeoLite2ASNCSVIn,
				IPv6File: "v6.csv",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGeoLite2ASNCSVIn(lib.ActionAdd, tt.opts...)
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

func TestWithASNWantedList(t *testing.T) {
	tests := []struct {
		name string
		want lib.WantedListExtended
		exp  map[string][]string
	}{
		{"Empty", lib.WantedListExtended{}, map[string][]string{}},
		{
			name: "Slice",
			want: lib.WantedListExtended{TypeSlice: []string{"AS13335", " as2519 ", "", "as", "15169"}},
			exp:  map[string][]string{"13335": {"AS13335"}, "2519": {"AS2519"}, "15169": {"AS15169"}},
		},
		{
			name: "Map",
			want: lib.WantedListExtended{TypeMap: map[string][]string{
				"cloud": {"AS13335", " ", "as"},
				" ":     {"AS1"},
			}},
			exp: map[string][]string{"13335": {"CLOUD"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_asn_csv_in{}
			WithASNWantedList(tt.want)(g)
			if !reflect.DeepEqual(g.Want, tt.exp) {
				t.Errorf("Want = %v, want %v", g.Want, tt.exp)
			}
		})
	}
}

func TestNewGeoLite2ASNCSVInFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *geolite2_asn_csv_in
		wantErr bool
	}{
		{
			name: "Empty data",
			want: &geolite2_asn_csv_in{
				Type: TypeGeoLite2ASNCSVIn, Action: lib.ActionAdd, Description: DescGeoLite2ASNCSVIn,
				IPv4File: defaultGeoLite2ASNCSVIPv4File, IPv6File: defaultGeoLite2ASNCSVIPv6File,
				Want: map[string][]string{},
			},
		},
		{
			name: "Full data",
			data: []byte(`{"ipv4":"v4.csv","ipv6":"v6.csv","wantedList":{"cloud":["AS13335"]},"onlyIPType":"ipv4"}`),
			want: &geolite2_asn_csv_in{
				Type: TypeGeoLite2ASNCSVIn, Action: lib.ActionAdd, Description: DescGeoLite2ASNCSVIn,
				IPv4File: "v4.csv", IPv6File: "v6.csv",
				Want: map[string][]string{"13335": {"CLOUD"}}, OnlyIPType: lib.IPv4,
			},
		},
		{name: "Invalid JSON", data: []byte(`[`), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGeoLite2ASNCSVInFromBytes(lib.ActionAdd, tt.data)
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

const asnIPv4CSV = `network,autonomous_system_number,autonomous_system_organization
1.0.0.0/24,13335,CLOUDFLARENET
1.0.4.0/22,38803,"Gtelecom Pty Ltd"
1.1.1.0/24,13335,CLOUDFLARENET
`

const asnIPv6CSV = `network,autonomous_system_number,autonomous_system_organization
2606:4700::/32,13335,CLOUDFLARENET
2001:200::/32,2500,"WIDE Project"
`

func TestGeoLite2ASNCSVIn_Input(t *testing.T) {
	v4 := writeTempFile(t, "v4.csv", []byte(asnIPv4CSV))
	v6 := writeTempFile(t, "v6.csv", []byte(asnIPv6CSV))
	srv := newFileServer(t, map[string][]byte{"/v4.csv": []byte(asnIPv4CSV)})

	invalidRecord := writeTempFile(t, "invalid.csv", []byte("network\n1.0.0.0/24\n"))
	inconsistent := writeTempFile(t, "inconsistent.csv", []byte("a,b,c\n1.0.0.0/24,1\n"))
	badCIDR := writeTempFile(t, "badcidr.csv", []byte("a,b\nbad,1\n"))
	badCIDRWant := writeTempFile(t, "badcidrwant.csv", []byte("a,b\nbad,13335\n"))

	tests := []struct {
		name      string
		g         *geolite2_asn_csv_in
		container lib.Container
		want      map[string][]string
		wantErr   bool
	}{
		{
			name: "Add all ASNs",
			g:    &geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: v4, IPv6File: v6},
			want: map[string][]string{
				"AS13335": {"1.0.0.0/24", "1.1.1.0/24", "2606:4700::/32"},
				"AS38803": {"1.0.4.0/22"},
				"AS2500":  {"2001:200::/32"},
			},
		},
		{
			name: "Add wanted ASNs with custom list names",
			g: &geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: v4, IPv6File: v6,
				Want: map[string][]string{"13335": {"CLOUD", "CF"}, "2500": {"AS2500"}}},
			want: map[string][]string{
				"CLOUD":  {"1.0.0.0/24", "1.1.1.0/24", "2606:4700::/32"},
				"CF":     {"1.0.0.0/24", "1.1.1.0/24", "2606:4700::/32"},
				"AS2500": {"2001:200::/32"},
			},
		},
		{
			name: "Only IPv4",
			g:    &geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: v4, IPv6File: v6, OnlyIPType: lib.IPv4, Want: map[string][]string{"13335": {"AS13335"}}},
			want: map[string][]string{"AS13335": {"1.0.0.0/24", "1.1.1.0/24"}},
		},
		{
			name: "Remote IPv4 file",
			g:    &geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: srv.URL + "/v4.csv"},
			want: map[string][]string{"AS13335": {"1.0.0.0/24", "1.1.1.0/24"}, "AS38803": {"1.0.4.0/22"}},
		},
		{
			name:      "Remove",
			g:         &geolite2_asn_csv_in{Action: lib.ActionRemove, IPv6File: v6, Want: map[string][]string{"13335": {"AS13335"}}},
			container: newContainerWith(t, map[string][]string{"AS13335": {"1.0.0.0/24", "2606:4700::/32"}}),
			want:      map[string][]string{"AS13335": {"1.0.0.0/24"}},
		},
		{
			name:      "Remove non-existent entry",
			g:         &geolite2_asn_csv_in{Action: lib.ActionRemove, IPv6File: v6, Want: map[string][]string{"13335": {"AS13335"}}},
			container: lib.NewContainer(),
			wantErr:   true,
		},
		{name: "Unknown action", g: &geolite2_asn_csv_in{Action: "unknown", IPv4File: v4}, wantErr: true},
		{name: "No entry generated", g: &geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: v4, Want: map[string][]string{"1": {"X"}}}, wantErr: true},
		{name: "IPv4 file not found", g: &geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: "/non/existent"}, wantErr: true},
		{name: "IPv6 file not found", g: &geolite2_asn_csv_in{Action: lib.ActionAdd, IPv6File: "/non/existent"}, wantErr: true},
		{name: "Remote file 404", g: &geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: srv.URL + "/missing.csv"}, wantErr: true},
		{name: "Invalid record", g: &geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: invalidRecord}, wantErr: true},
		{name: "Inconsistent fields", g: &geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: inconsistent}, wantErr: true},
		{name: "Invalid CIDR", g: &geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: badCIDR}, wantErr: true},
		{name: "Invalid CIDR with wanted list", g: &geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: badCIDRWant, Want: map[string][]string{"13335": {"X"}}}, wantErr: true},
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

func TestGeoLite2ASNCSVIn_ProcessNilEntries(t *testing.T) {
	v4 := writeTempFile(t, "v4.csv", []byte(asnIPv4CSV))
	g := &geolite2_asn_csv_in{}
	if err := g.process(v4, nil); err != nil {
		t.Fatalf("process() error = %v", err)
	}
}
