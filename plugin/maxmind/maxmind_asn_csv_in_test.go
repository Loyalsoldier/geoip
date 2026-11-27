package maxmind

import (
	"reflect"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

const testASNIPv4CSV = `network,autonomous_system_number,autonomous_system_organization
1.0.0.0/24,13335,CLOUDFLARENET
1.0.4.0/22,38803,"Gtelecom Pty Ltd"
1.0.16.0/24,2519,"ARTERIA Networks Corporation"
1.1.1.0/24,13335,CLOUDFLARENET
`

const testASNIPv6CSV = `network,autonomous_system_number,autonomous_system_organization
2606:4700::/32,13335,CLOUDFLARENET
`

func TestNewGeoLite2ASNCSVIn(t *testing.T) {
	tests := []struct {
		name     string
		opts     []lib.InputOption
		wantV4   string
		wantV6   string
		wantOnly lib.IPType
		wantWant map[string][]string
	}{
		{"default", nil, defaultGeoLite2ASNCSVIPv4File, defaultGeoLite2ASNCSVIPv6File, "", nil},
		{"ipv6 only file", []lib.InputOption{nil, WithIPv6File(" v6.csv ")}, "", "v6.csv", "", nil},
		{
			"options", []lib.InputOption{WithIPv4File("v4.csv"), WithInputOnlyIPType(lib.IPv4), WithGeoLite2ASNCSVWantedList(uint(13335))},
			"v4.csv", "", lib.IPv4, map[string][]string{"13335": {"AS13335"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewGeoLite2ASNCSVIn(lib.ActionAdd, tt.opts...)
			assertErr(t, err, false)
			g := conv.(*geolite2_asn_csv_in)
			if g.GetType() != TypeGeoLite2ASNCSVIn || g.GetAction() != lib.ActionAdd || g.GetDescription() != DescGeoLite2ASNCSVIn {
				t.Errorf("unexpected meta: %+v", g)
			}
			if g.IPv4File != tt.wantV4 || g.IPv6File != tt.wantV6 || g.OnlyIPType != tt.wantOnly {
				t.Errorf("got %+v", g)
			}
			if len(g.Want) != len(tt.wantWant) || (len(tt.wantWant) > 0 && !reflect.DeepEqual(g.Want, tt.wantWant)) {
				t.Errorf("Want = %v, want %v", g.Want, tt.wantWant)
			}
		})
	}
}

func TestNewGeoLite2ASNCSVInFromBytes(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		wantV4   string
		wantV6   string
		wantWant map[string][]string
		wantErr  bool
	}{
		{"empty", "", defaultGeoLite2ASNCSVIPv4File, defaultGeoLite2ASNCSVIPv6File, map[string][]string{}, false},
		{"slice num", `{"ipv4":"a","wantedList":[13335, 2519]}`, "a", "", map[string][]string{"13335": {"AS13335"}, "2519": {"AS2519"}}, false},
		{"slice str", `{"ipv6":"b","wantedList":["AS13335"," as2519 ",""]}`, "", "b", map[string][]string{"13335": {"AS13335"}, "2519": {"AS2519"}}, false},
		{"map num", `{"wantedList":{"cf":[13335],"":[2519]}}`, defaultGeoLite2ASNCSVIPv4File, defaultGeoLite2ASNCSVIPv6File, map[string][]string{"13335": {"CF"}}, false},
		{"map str", `{"wantedList":{"cf":["AS13335","13335"],"x":["as2519"]}}`, defaultGeoLite2ASNCSVIPv4File, defaultGeoLite2ASNCSVIPv6File, map[string][]string{"13335": {"CF"}, "2519": {"X"}}, false},
		{"invalid wantedList", `{"wantedList":true}`, "", "", nil, true},
		{"invalid json", `{`, "", "", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewGeoLite2ASNCSVInFromBytes(lib.ActionAdd, []byte(tt.data))
			assertErr(t, err, tt.wantErr)
			if tt.wantErr {
				return
			}
			g := conv.(*geolite2_asn_csv_in)
			if g.IPv4File != tt.wantV4 || g.IPv6File != tt.wantV6 {
				t.Errorf("got %+v", g)
			}
			if !reflect.DeepEqual(g.Want, tt.wantWant) {
				t.Errorf("Want = %v, want %v", g.Want, tt.wantWant)
			}
		})
	}
}

func TestGeoLite2ASNCSVWantedList_UnmarshalJSON(t *testing.T) {
	var w GeoLite2ASNCSVWantedList
	if err := w.UnmarshalJSON([]byte("  ")); err != nil {
		t.Fatal(err)
	}
	if len(w.TypeSliceNum) != 0 || len(w.TypeSliceStr) != 0 || len(w.TypeMapNum) != 0 || len(w.TypeMapStr) != 0 {
		t.Errorf("expected empty, got %+v", w)
	}
}

func TestWithGeoLite2ASNCSVWantedList(t *testing.T) {
	apply := func(opt lib.InputOption) map[string][]string {
		g := &geolite2_asn_csv_in{}
		opt(g)
		return g.Want
	}
	tests := []struct {
		name string
		got  map[string][]string
		want map[string][]string
	}{
		{"uint", apply(WithGeoLite2ASNCSVWantedList(uint(1))), map[string][]string{"1": {"AS1"}}},
		{"[]uint", apply(WithGeoLite2ASNCSVWantedList([]uint{1, 2, 1})), map[string][]string{"1": {"AS1"}, "2": {"AS2"}}},
		{"string", apply(WithGeoLite2ASNCSVWantedList(" AS3 ")), map[string][]string{"3": {"AS3"}}},
		{"empty string", apply(WithGeoLite2ASNCSVWantedList("as")), map[string][]string{}},
		{"[]string", apply(WithGeoLite2ASNCSVWantedList([]string{"4", "as5"})), map[string][]string{"4": {"AS4"}, "5": {"AS5"}}},
		{"map uint", apply(WithGeoLite2ASNCSVWantedList(map[string][]uint{"b": {6}, "a": {6}})), map[string][]string{"6": {"A", "B"}}},
		{"map string", apply(WithGeoLite2ASNCSVWantedList(map[string][]string{"x": {"AS7", "7"}, " ": {"8"}})), map[string][]string{"7": {"X"}}},
		{
			"struct",
			apply(WithGeoLite2ASNCSVWantedList(GeoLite2ASNCSVWantedList{
				TypeSliceNum: []uint{1},
				TypeSliceStr: []string{"AS2"},
				TypeMapNum:   map[string][]uint{"n": {3}},
				TypeMapStr:   map[string][]string{"s": {"AS3"}},
			})),
			map[string][]string{"1": {"AS1"}, "2": {"AS2"}, "3": {"N", "S"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.got, tt.want) {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestGeoLite2ASNCSVIn_Input(t *testing.T) {
	dir := t.TempDir()
	v4 := writeFile(t, dir, "v4.csv", []byte(testASNIPv4CSV))
	v6 := writeFile(t, dir, "v6.csv", []byte(testASNIPv6CSV))
	short := writeFile(t, dir, "short.csv", []byte("h\n1.0.0.0/24\n"))
	badCIDR := writeFile(t, dir, "badcidr.csv", []byte("h,h\nbad,13335\n"))
	badQuote := writeFile(t, dir, "badquote.csv", []byte("h,h\n1.0.0.0/24,\"1\n"))
	srv := newServer(t, map[string][]byte{"/v4.csv": []byte(testASNIPv4CSV)})
	missing := dir + "/missing.csv"

	tests := []struct {
		name    string
		v4, v6  string
		want    lib.InputOption
		only    lib.IPType
		action  lib.Action
		pre     map[string][]string
		failing bool
		wantErr bool
		expect  map[string][]string
	}{
		{
			name: "all", v4: v4, v6: v6, action: lib.ActionAdd,
			expect: map[string][]string{
				"AS13335": {"1.0.0.0/24", "1.1.1.0/24", "2606:4700::/32"},
				"AS38803": {"1.0.4.0/22"},
				"AS2519":  {"1.0.16.0/24"},
			},
		},
		{
			name: "custom lists ipv4 only", v4: v4, v6: v6, action: lib.ActionAdd, only: lib.IPv4,
			want: WithGeoLite2ASNCSVWantedList(map[string][]string{"cf": {"13335"}, "jp": {"2519", "13335"}}),
			expect: map[string][]string{
				"CF": {"1.0.0.0/24", "1.1.1.0/24"},
				"JP": {"1.0.0.0/24", "1.0.16.0/24", "1.1.1.0/24"},
			},
		},
		{
			name: "remote", v4: srv.URL + "/v4.csv", action: lib.ActionAdd, want: WithGeoLite2ASNCSVWantedList(uint(38803)),
			expect: map[string][]string{"AS38803": {"1.0.4.0/22"}},
		},
		{
			name: "remove", v4: v4, action: lib.ActionRemove, want: WithGeoLite2ASNCSVWantedList("AS2519"),
			pre:    map[string][]string{"AS2519": {"1.0.16.0/23"}},
			expect: map[string][]string{"AS2519": {"1.0.17.0/24"}},
		},
		{name: "remove not found", v4: v4, action: lib.ActionRemove, wantErr: true},
		{name: "add fails", v4: v4, action: lib.ActionAdd, failing: true, wantErr: true},
		{name: "unknown action", v4: v4, action: lib.Action("x"), wantErr: true},
		{name: "no entry", v4: v4, action: lib.ActionAdd, want: WithGeoLite2ASNCSVWantedList(uint(1)), wantErr: true},
		{name: "v4 missing", v4: missing, action: lib.ActionAdd, wantErr: true},
		{name: "v6 missing", v4: v4, v6: missing, action: lib.ActionAdd, wantErr: true},
		{name: "remote 404", v4: srv.URL + "/none", action: lib.ActionAdd, wantErr: true},
		{name: "short record", v4: short, action: lib.ActionAdd, wantErr: true},
		{name: "bad cidr", v4: badCIDR, action: lib.ActionAdd, wantErr: true},
		{name: "bad cidr wanted", v4: badCIDR, action: lib.ActionAdd, want: WithGeoLite2ASNCSVWantedList(uint(13335)), wantErr: true},
		{name: "bad csv", v4: badQuote, action: lib.ActionAdd, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_asn_csv_in{Type: TypeGeoLite2ASNCSVIn, Action: tt.action, IPv4File: tt.v4, IPv6File: tt.v6, OnlyIPType: tt.only}
			if tt.want != nil {
				tt.want(g)
			}

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

func TestGeoLite2ASNCSVIn_ProcessNilEntries(t *testing.T) {
	v4 := writeFile(t, t.TempDir(), "v4.csv", []byte(testASNIPv4CSV))
	g := &geolite2_asn_csv_in{}
	assertErr(t, g.process(v4, nil), false)
}
