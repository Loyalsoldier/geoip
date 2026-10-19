package maxmind

import (
	"slices"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

const asnIPv4CSV = `network,autonomous_system_number,autonomous_system_organization
1.0.0.0/24,13335,CLOUDFLARENET
1.0.4.0/22,38803,"Gtelecom Pty Ltd"
8.8.8.0/24, 15169 ,GOOGLE
`

const asnIPv6CSV = `network,autonomous_system_number,autonomous_system_organization
2400:cb00::/32,13335,CLOUDFLARENET
2001:4860::/32,15169,GOOGLE
`

func TestWithASNWantedList(t *testing.T) {
	tests := []struct {
		name string
		in   lib.WantedListExtended
		want map[string][]string
	}{
		{"empty", lib.WantedListExtended{}, map[string][]string{}},
		{
			"slice",
			lib.WantedListExtended{TypeSlice: []string{"AS13335", " as15169 ", "15169", "", "AS"}},
			map[string][]string{"13335": {"AS13335"}, "15169": {"AS15169"}},
		},
		{
			"map",
			lib.WantedListExtended{TypeMap: map[string][]string{
				" cloud ": {"AS13335", "", "as"},
				"google":  {"15169", "13335"},
				"  ":      {"1"},
			}},
			nil, // checked separately (map iteration order)
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_asn_csv_in{}
			WithASNWantedList(tt.in)(g)
			if tt.want == nil {
				if len(g.Want) != 2 {
					t.Fatalf("got %v", g.Want)
				}
				got13335 := slices.Clone(g.Want["13335"])
				slices.Sort(got13335)
				if !slices.Equal(got13335, []string{"CLOUD", "GOOGLE"}) {
					t.Errorf("13335: got %v", got13335)
				}
				if !slices.Equal(g.Want["15169"], []string{"GOOGLE"}) {
					t.Errorf("15169: got %v", g.Want["15169"])
				}
				return
			}
			if len(g.Want) != len(tt.want) {
				t.Fatalf("got %v, want %v", g.Want, tt.want)
			}
			for k, v := range tt.want {
				if !slices.Equal(g.Want[k], v) {
					t.Errorf("%s: got %v, want %v", k, g.Want[k], v)
				}
			}
		})
	}
}

func TestNewGeoLite2ASNCSVIn(t *testing.T) {
	tests := []struct {
		name   string
		opts   []lib.InputOption
		wantV4 string
		wantV6 string
		only   lib.IPType
	}{
		{"defaults", nil, defaultGeoLite2ASNCSVIPv4File, defaultGeoLite2ASNCSVIPv6File, ""},
		{"nil option", []lib.InputOption{nil}, defaultGeoLite2ASNCSVIPv4File, defaultGeoLite2ASNCSVIPv6File, ""},
		{"v4 only", []lib.InputOption{WithIPv4File("a.csv")}, "a.csv", "", ""},
		{"v6 only + ipType + wanted list ignored", []lib.InputOption{WithIPv6File(" b.csv "), WithInputOnlyIPType(lib.IPv4), WithInputWantedList([]string{"x"})}, "", "b.csv", lib.IPv4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewGeoLite2ASNCSVIn(lib.ActionAdd, tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			g := conv.(*geolite2_asn_csv_in)
			if g.GetType() != TypeGeoLite2ASNCSVIn || g.GetDescription() != DescGeoLite2ASNCSVIn || g.GetAction() != lib.ActionAdd {
				t.Errorf("bad metadata %+v", g)
			}
			if g.IPv4File != tt.wantV4 || g.IPv6File != tt.wantV6 || g.OnlyIPType != tt.only {
				t.Errorf("got %+v", g)
			}
			if len(g.Want) != 0 {
				t.Errorf("Want should be empty: %v", g.Want)
			}
		})
	}
}

func TestNewGeoLite2ASNCSVInFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		v4, v6  string
		want    map[string][]string
		only    lib.IPType
	}{
		{"empty", "", false, defaultGeoLite2ASNCSVIPv4File, defaultGeoLite2ASNCSVIPv6File, map[string][]string{}, ""},
		{"slice", `{"ipv4":"a","wantedList":["AS1"],"onlyIPType":"ipv6"}`, false, "a", "", map[string][]string{"1": {"AS1"}}, lib.IPv6},
		{"map", `{"ipv6":"b","wantedList":{"cf":["13335"]}}`, false, "", "b", map[string][]string{"13335": {"CF"}}, ""},
		{"invalid", `{"wantedList":1}`, true, "", "", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewGeoLite2ASNCSVInFromBytes(lib.ActionAdd, []byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			g := conv.(*geolite2_asn_csv_in)
			if g.IPv4File != tt.v4 || g.IPv6File != tt.v6 || g.OnlyIPType != tt.only || len(g.Want) != len(tt.want) {
				t.Errorf("got %+v", g)
			}
			for k, v := range tt.want {
				if !slices.Equal(g.Want[k], v) {
					t.Errorf("%s: got %v want %v", k, g.Want[k], v)
				}
			}
		})
	}
}

func TestGeoLite2ASNCSVInInput(t *testing.T) {
	v4Path := writeFile(t, "v4.csv", []byte(asnIPv4CSV))
	v6Path := writeFile(t, "v6.csv", []byte(asnIPv6CSV))
	shortPath := writeFile(t, "short.csv", []byte("network\n1.0.0.0/24\n"))
	badCIDRAll := writeFile(t, "bad1.csv", []byte("network,asn\n300.0.0.0/24,1\n"))
	badRow := writeFile(t, "bad2.csv", []byte("network,asn\n1.0.0.0/24,1\n1.0.0.0/24\n"))
	srv := serveFiles(t, map[string][]byte{"/v4.csv": []byte(asnIPv4CSV), "/v6.csv": []byte(asnIPv6CSV)})

	all := map[string][]string{
		"AS13335": {"1.0.0.0/24", "2400:cb00::/32"},
		"AS38803": {"1.0.4.0/22"},
		"AS15169": {"8.8.8.0/24", "2001:4860::/32"},
	}

	tests := []struct {
		name    string
		v4, v6  string
		action  lib.Action
		want    lib.WantedListExtended
		only    lib.IPType
		pre     map[string][]string
		wantErr string
		expect  map[string][]string
	}{
		{name: "all local", v4: v4Path, v6: v6Path, action: lib.ActionAdd, expect: all},
		{name: "all remote", v4: srv.URL + "/v4.csv", v6: srv.URL + "/v6.csv", action: lib.ActionAdd, expect: all},
		{name: "v6 only", v6: v6Path, action: lib.ActionAdd,
			expect: map[string][]string{"AS13335": {"2400:cb00::/32"}, "AS15169": {"2001:4860::/32"}}},
		{name: "wanted slice", v4: v4Path, v6: v6Path, action: lib.ActionAdd, want: lib.WantedListExtended{TypeSlice: []string{"as15169"}},
			expect: map[string][]string{"AS15169": {"8.8.8.0/24", "2001:4860::/32"}}},
		{name: "wanted map", v4: v4Path, v6: v6Path, action: lib.ActionAdd,
			want: lib.WantedListExtended{TypeMap: map[string][]string{"cdn": {"13335", "15169"}, "cloudflare": {"AS13335"}}},
			expect: map[string][]string{
				"CDN":        {"1.0.0.0/24", "8.8.8.0/24", "2001:4860::/32", "2400:cb00::/32"},
				"CLOUDFLARE": {"1.0.0.0/24", "2400:cb00::/32"},
			}},
		{name: "only ipv4", v4: v4Path, v6: v6Path, action: lib.ActionAdd, only: lib.IPv4, want: lib.WantedListExtended{TypeSlice: []string{"13335"}},
			expect: map[string][]string{"AS13335": {"1.0.0.0/24"}}},
		{name: "remove", v4: v4Path, v6: v6Path, action: lib.ActionRemove, want: lib.WantedListExtended{TypeSlice: []string{"13335"}},
			pre:    map[string][]string{"AS13335": {"1.0.0.0/23", "2400:cb00::/32"}},
			expect: map[string][]string{"AS13335": {"1.0.1.0/24"}}},
		{name: "remove missing", v4: v4Path, action: lib.ActionRemove, wantErr: "not found"},
		{name: "unknown action", v4: v4Path, action: "bogus", wantErr: lib.ErrUnknownAction.Error()},
		{name: "wanted none match", v4: v4Path, v6: v6Path, action: lib.ActionAdd, want: lib.WantedListExtended{TypeSlice: []string{"1"}}, wantErr: "no entry is generated"},
		{name: "v4 missing", v4: "/nonexistent/v4.csv", action: lib.ActionAdd, wantErr: "no such file"},
		{name: "v6 missing", v4: v4Path, v6: "/nonexistent/v6.csv", action: lib.ActionAdd, wantErr: "no such file"},
		{name: "remote 404", v4: srv.URL + "/nope", action: lib.ActionAdd, wantErr: "404"},
		{name: "short record", v4: shortPath, action: lib.ActionAdd, wantErr: "invalid record"},
		{name: "bad cidr all", v4: badCIDRAll, action: lib.ActionAdd, wantErr: lib.ErrInvalidCIDR.Error()},
		{name: "bad cidr wanted", v4: badCIDRAll, action: lib.ActionAdd, want: lib.WantedListExtended{TypeSlice: []string{"1"}}, wantErr: lib.ErrInvalidCIDR.Error()},
		{name: "bad row", v4: badRow, action: lib.ActionAdd, wantErr: "wrong number of fields"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_asn_csv_in{Type: TypeGeoLite2ASNCSVIn, Action: tt.action, IPv4File: tt.v4, IPv6File: tt.v6, OnlyIPType: tt.only}
			WithASNWantedList(tt.want)(g)
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

func TestGeoLite2ASNCSVInProcessNilEntries(t *testing.T) {
	v4Path := writeFile(t, "v4.csv", []byte(asnIPv4CSV))
	tests := []struct {
		name string
		file string
	}{
		{"nil entries map is accepted", v4Path},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_asn_csv_in{}
			if err := g.process(tt.file, nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}
