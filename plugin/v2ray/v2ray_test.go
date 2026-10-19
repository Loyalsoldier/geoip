package v2ray

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"google.golang.org/protobuf/proto"
)

// ---------- helpers ----------

type entryDef struct {
	name  string
	cidrs []string
}

func newEntry(t *testing.T, name string, cidrs ...string) *lib.Entry {
	t.Helper()
	e := lib.NewEntry(name)
	for _, c := range cidrs {
		if err := e.AddPrefix(c); err != nil {
			t.Fatalf("AddPrefix(%q): %v", c, err)
		}
	}
	return e
}

func newContainer(t *testing.T, defs ...entryDef) lib.Container {
	t.Helper()
	c := lib.NewContainer()
	for _, d := range defs {
		if err := c.Add(newEntry(t, d.name, d.cidrs...)); err != nil {
			t.Fatalf("container.Add(%s): %v", d.name, err)
		}
	}
	return c
}

func compareStrings(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: count = %d, want %d (got %v, want %v)", label, len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s: [%d] = %q, want %q", label, i, got[i], want[i])
		}
	}
}

// assertContainer checks that the container holds exactly the wanted entries
// with exactly the wanted CIDR strings.
func assertContainer(t *testing.T, c lib.Container, want map[string][]string) {
	t.Helper()
	if c.Len() != len(want) {
		names := []string{}
		for e := range c.Loop() {
			names = append(names, e.GetName())
		}
		t.Fatalf("container Len = %d, want %d (entries %v)", c.Len(), len(want), names)
	}
	for name, cidrs := range want {
		e, ok := c.GetEntry(name)
		if !ok {
			t.Fatalf("entry %s not found", name)
		}
		got, err := e.MarshalText()
		if err != nil {
			got = nil
		}
		compareStrings(t, "entry "+name, got, cidrs)
	}
}

func serveFile(t *testing.T, path string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func writeDat(t *testing.T, list *GeoIPList) string {
	t.Helper()
	b, err := proto.Marshal(list)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "in.dat")
	if err := os.WriteFile(p, b, 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

var (
	cnDef = entryDef{"cn", []string{"1.0.0.0/24", "1.0.1.0/24", "2001:db8::/33", "2001:db8:8000::/33"}}
	usDef = entryDef{"us", []string{"8.8.8.0/24", "2001:4860::/32"}}
	v6Def = entryDef{"v6only", []string{"2400:cb00::/32"}}

	cnWant = []string{"1.0.0.0/23", "2001:db8::/32"}
	usWant = []string{"8.8.8.0/24", "2001:4860::/32"}
)

// ---------- constructors ----------

func TestNewGeoIPDatIn(t *testing.T) {
	tests := []struct {
		name     string
		opts     []lib.InputOption
		wantErr  bool
		wantURI  string
		wantWant map[string]bool
		wantType lib.IPType
	}{
		{name: "missing uri", opts: nil, wantErr: true},
		{name: "blank uri", opts: []lib.InputOption{WithURI("   ")}, wantErr: true},
		{name: "nil option ignored", opts: []lib.InputOption{nil, WithURI(" a.dat ")}, wantURI: "a.dat"},
		{
			name: "all options",
			opts: []lib.InputOption{
				WithURI("http://x/y.dat"),
				WithInputWantedList([]string{" cn ", "", "Us"}),
				WithInputOnlyIPType(lib.IPv6),
			},
			wantURI:  "http://x/y.dat",
			wantWant: map[string]bool{"CN": true, "US": true},
			wantType: lib.IPv6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewGeoIPDatIn(lib.ActionAdd, tt.opts...)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			g := conv.(*geoip_dat_in)
			if g.URI != tt.wantURI {
				t.Errorf("URI = %q, want %q", g.URI, tt.wantURI)
			}
			if len(g.Want) != len(tt.wantWant) {
				t.Errorf("Want = %v, want %v", g.Want, tt.wantWant)
			}
			for k := range tt.wantWant {
				if !g.Want[k] {
					t.Errorf("Want missing %s", k)
				}
			}
			if g.OnlyIPType != tt.wantType {
				t.Errorf("OnlyIPType = %q, want %q", g.OnlyIPType, tt.wantType)
			}
			if conv.GetType() != TypeGeoIPDatIn || conv.GetAction() != lib.ActionAdd || conv.GetDescription() != DescGeoIPDatIn {
				t.Errorf("getters = %q %q %q", conv.GetType(), conv.GetAction(), conv.GetDescription())
			}
		})
	}
}

func TestNewGeoIPDatInFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		wantURI string
	}{
		{name: "empty data", data: "", wantErr: true},
		{name: "invalid json", data: "{", wantErr: true},
		{name: "missing uri", data: `{"wantedList":["cn"]}`, wantErr: true},
		{name: "valid", data: `{"uri":"a.dat","wantedList":["cn"],"onlyIPType":"ipv4"}`, wantURI: "a.dat"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewGeoIPDatInFromBytes(lib.ActionRemove, []byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			g := conv.(*geoip_dat_in)
			if g.URI != tt.wantURI || !g.Want["CN"] || g.OnlyIPType != lib.IPv4 || g.Action != lib.ActionRemove {
				t.Errorf("unexpected converter %+v", g)
			}
		})
	}
}

func TestNewGeoIPDatOut(t *testing.T) {
	tests := []struct {
		name     string
		opts     []lib.OutputOption
		wantName string
		wantDir  string
		wantOne  bool
		wantType lib.IPType
		wantWLen int
		wantELen int
	}{
		{name: "defaults", wantName: defaultOutputName, wantDir: defaultOutputDir},
		{
			name:     "blank name and dir fall back to default",
			opts:     []lib.OutputOption{nil, WithOutputName("  "), WithOutputDir(" ")},
			wantName: defaultOutputName, wantDir: defaultOutputDir,
		},
		{
			name: "all options",
			opts: []lib.OutputOption{
				WithOutputName(" x.dat "), WithOutputDir(" /tmp/o "),
				WithOutputWantedList([]string{"a", "b"}), WithOutputExcludedList([]string{"c"}),
				WithOneFilePerList(true), WithOutputOnlyIPType(lib.IPv4),
			},
			wantName: "x.dat", wantDir: "/tmp/o", wantOne: true, wantType: lib.IPv4, wantWLen: 2, wantELen: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewGeoIPDatOut(lib.ActionOutput, tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			g := conv.(*geoip_dat_out)
			if g.OutputName != tt.wantName || g.OutputDir != tt.wantDir || g.OneFilePerList != tt.wantOne ||
				g.OnlyIPType != tt.wantType || len(g.Want) != tt.wantWLen || len(g.Exclude) != tt.wantELen {
				t.Errorf("unexpected converter %+v", g)
			}
			if conv.GetType() != TypeGeoIPDatOut || conv.GetAction() != lib.ActionOutput || conv.GetDescription() != DescGeoIPDatOut {
				t.Errorf("getters = %q %q %q", conv.GetType(), conv.GetAction(), conv.GetDescription())
			}
		})
	}
}

func TestNewGeoIPDatOutFromBytes(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		wantErr  bool
		wantName string
		wantDir  string
		wantOne  bool
	}{
		{name: "empty data", data: "", wantName: defaultOutputName, wantDir: defaultOutputDir},
		{name: "invalid json", data: "[", wantErr: true},
		{
			name:     "valid",
			data:     `{"outputName":"g.dat","outputDir":"d","wantedList":["cn"],"excludedList":["us"],"oneFilePerList":true,"onlyIPType":"ipv6"}`,
			wantName: "g.dat", wantDir: "d", wantOne: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewGeoIPDatOutFromBytes(lib.ActionOutput, []byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			g := conv.(*geoip_dat_out)
			if g.OutputName != tt.wantName || g.OutputDir != tt.wantDir || g.OneFilePerList != tt.wantOne {
				t.Errorf("unexpected converter %+v", g)
			}
		})
	}
}

func TestConfigRegistration(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.dat")
	conf := fmt.Sprintf(`{
		"input": [{"type": %q, "action": "add", "args": {"uri": %q}}],
		"output": [{"type": %q, "action": "output", "args": {"outputDir": %q}}]
	}`, TypeGeoIPDatIn, src, TypeGeoIPDatOut, filepath.Join(dir, "out"))

	out, _ := NewGeoIPDatOut(lib.ActionOutput, WithOutputDir(dir), WithOutputName("src.dat"))
	if err := out.Output(newContainer(t, cnDef)); err != nil {
		t.Fatal(err)
	}

	inst, err := lib.NewInstance()
	if err != nil {
		t.Fatal(err)
	}
	if err := inst.InitConfigFromBytes([]byte(conf)); err != nil {
		t.Fatal(err)
	}
	if err := inst.Run(); err != nil {
		t.Fatal(err)
	}
	in, _ := NewGeoIPDatIn(lib.ActionAdd, WithURI(filepath.Join(dir, "out", defaultOutputName)))
	c, err := in.Input(lib.NewContainer())
	if err != nil {
		t.Fatal(err)
	}
	assertContainer(t, c, map[string][]string{"CN": cnWant})
}

// ---------- round trip ----------

func TestDatRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		defs    []entryDef
		outOpts []lib.OutputOption
		inOpts  []lib.InputOption
		remote  bool
		want    map[string][]string
	}{
		{
			name: "all entries via file",
			defs: []entryDef{cnDef, usDef, v6Def},
			want: map[string][]string{"CN": cnWant, "US": usWant, "V6ONLY": {"2400:cb00::/32"}},
		},
		{
			name:   "all entries via http",
			defs:   []entryDef{cnDef, usDef},
			remote: true,
			want:   map[string][]string{"CN": cnWant, "US": usWant},
		},
		{
			name:    "output wanted list with blank, missing entry and case",
			defs:    []entryDef{cnDef, usDef},
			outOpts: []lib.OutputOption{WithOutputWantedList([]string{" cn ", "", "xx"})},
			want:    map[string][]string{"CN": cnWant},
		},
		{
			name:    "output excluded list",
			defs:    []entryDef{cnDef, usDef, v6Def},
			outOpts: []lib.OutputOption{WithOutputExcludedList([]string{" us", "", "V6ONLY"})},
			want:    map[string][]string{"CN": cnWant},
		},
		{
			name:    "output wanted minus excluded",
			defs:    []entryDef{cnDef, usDef},
			outOpts: []lib.OutputOption{WithOutputWantedList([]string{"cn", "us"}), WithOutputExcludedList([]string{"cn"})},
			want:    map[string][]string{"US": usWant},
		},
		{
			name:    "output only ipv4",
			defs:    []entryDef{cnDef, usDef},
			outOpts: []lib.OutputOption{WithOutputOnlyIPType(lib.IPv4)},
			want:    map[string][]string{"CN": {"1.0.0.0/23"}, "US": {"8.8.8.0/24"}},
		},
		{
			name:    "output only ipv6",
			defs:    []entryDef{cnDef, usDef, v6Def},
			outOpts: []lib.OutputOption{WithOutputOnlyIPType(lib.IPv6)},
			want:    map[string][]string{"CN": {"2001:db8::/32"}, "US": {"2001:4860::/32"}, "V6ONLY": {"2400:cb00::/32"}},
		},
		{
			name:   "input wanted list",
			defs:   []entryDef{cnDef, usDef},
			inOpts: []lib.InputOption{WithInputWantedList([]string{" us "})},
			want:   map[string][]string{"US": usWant},
		},
		{
			name:   "input only ipv4",
			defs:   []entryDef{cnDef, usDef},
			inOpts: []lib.InputOption{WithInputOnlyIPType(lib.IPv4)},
			want:   map[string][]string{"CN": {"1.0.0.0/23"}, "US": {"8.8.8.0/24"}},
		},
		{
			name:   "input only ipv6 via http",
			defs:   []entryDef{cnDef},
			inOpts: []lib.InputOption{WithInputOnlyIPType(lib.IPv6)},
			remote: true,
			want:   map[string][]string{"CN": {"2001:db8::/32"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			outOpts := append([]lib.OutputOption{WithOutputDir(dir), WithOutputName("geo.dat")}, tt.outOpts...)
			out, err := NewGeoIPDatOut(lib.ActionOutput, outOpts...)
			if err != nil {
				t.Fatal(err)
			}
			if err := out.Output(newContainer(t, tt.defs...)); err != nil {
				t.Fatalf("Output: %v", err)
			}

			path := filepath.Join(dir, "geo.dat")
			uri := path
			if tt.remote {
				uri = serveFile(t, path).URL + "/geo.dat"
			}
			in, err := NewGeoIPDatIn(lib.ActionAdd, append([]lib.InputOption{WithURI(uri)}, tt.inOpts...)...)
			if err != nil {
				t.Fatal(err)
			}
			c, err := in.Input(lib.NewContainer())
			if err != nil {
				t.Fatalf("Input: %v", err)
			}
			assertContainer(t, c, tt.want)
		})
	}
}

func TestDatOutputSortedEntries(t *testing.T) {
	dir := t.TempDir()
	out, _ := NewGeoIPDatOut(lib.ActionOutput, WithOutputDir(dir))
	if err := out.Output(newContainer(t, usDef, v6Def, cnDef)); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, defaultOutputName))
	if err != nil {
		t.Fatal(err)
	}
	var list GeoIPList
	if err := proto.Unmarshal(b, &list); err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, e := range list.GetEntry() {
		got = append(got, e.GetCountryCode())
	}
	compareStrings(t, "country codes", got, []string{"CN", "US", "V6ONLY"})

	// Raw CIDR bytes for CN: IPv4 as 4 bytes, IPv6 as 16 bytes.
	cidrs := []string{}
	for _, c := range list.GetEntry()[0].GetCidr() {
		addr, _ := netip.AddrFromSlice(c.GetIp())
		cidrs = append(cidrs, fmt.Sprintf("%d:%s/%d", len(c.GetIp()), addr, c.GetPrefix()))
	}
	compareStrings(t, "CN raw cidrs", cidrs, []string{"4:1.0.0.0/23", "16:2001:db8::/32"})
}

func TestDatOneFilePerList(t *testing.T) {
	tests := []struct {
		name   string
		remote bool
	}{
		{name: "file"},
		{name: "http", remote: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			out, _ := NewGeoIPDatOut(lib.ActionOutput, WithOutputDir(dir), WithOneFilePerList(true))
			if err := out.Output(newContainer(t, cnDef, usDef)); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(dir, defaultOutputName)); !os.IsNotExist(err) {
				t.Errorf("combined file should not exist, stat err = %v", err)
			}
			for name, want := range map[string][]string{"cn": cnWant, "us": usWant} {
				path := filepath.Join(dir, name+".dat")
				uri := path
				if tt.remote {
					uri = serveFile(t, path).URL + "/" + name + ".dat"
				}
				in, _ := NewGeoIPDatIn(lib.ActionAdd, WithURI(uri))
				c, err := in.Input(lib.NewContainer())
				if err != nil {
					t.Fatal(err)
				}
				upper := map[string]string{"cn": "CN", "us": "US"}[name]
				assertContainer(t, c, map[string][]string{upper: want})
			}
		})
	}
}

func TestDatOutputNoFile(t *testing.T) {
	tests := []struct {
		name string
		defs []entryDef
		opts []lib.OutputOption
	}{
		{name: "empty container"},
		{name: "everything excluded by wanted list", defs: []entryDef{cnDef}, opts: []lib.OutputOption{
			WithOutputWantedList([]string{"cn"}), WithOutputExcludedList([]string{"cn"}),
		}},
		{name: "wanted entries missing", defs: []entryDef{cnDef}, opts: []lib.OutputOption{WithOutputWantedList([]string{"zz"})}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "out")
			out, _ := NewGeoIPDatOut(lib.ActionOutput, append([]lib.OutputOption{WithOutputDir(dir)}, tt.opts...)...)
			if err := out.Output(newContainer(t, tt.defs...)); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(dir); !os.IsNotExist(err) {
				t.Errorf("output dir should not be created, stat err = %v", err)
			}
		})
	}
}

func TestDatOutputErrors(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		defs []entryDef
		opts []lib.OutputOption
	}{
		{name: "entry without prefix", defs: []entryDef{{"empty", nil}}},
		{name: "ipv4 only on ipv6-only entry", defs: []entryDef{v6Def}, opts: []lib.OutputOption{WithOutputOnlyIPType(lib.IPv4)}},
		{name: "output dir is a file", defs: []entryDef{cnDef}, opts: []lib.OutputOption{WithOutputDir(blocker)}},
		{name: "output dir is a file one file per list", defs: []entryDef{cnDef}, opts: []lib.OutputOption{WithOutputDir(blocker), WithOneFilePerList(true)}},
		{name: "output file path is a directory", defs: []entryDef{cnDef}, opts: []lib.OutputOption{WithOutputDir(filepath.Dir(blocker)), WithOutputName(".")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := append([]lib.OutputOption{WithOutputDir(t.TempDir())}, tt.opts...)
			out, _ := NewGeoIPDatOut(lib.ActionOutput, opts...)
			if err := out.Output(newContainer(t, tt.defs...)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

// ---------- input ----------

func TestDatInputHandcrafted(t *testing.T) {
	mapped := func(s string) []byte { return netip.MustParseAddr(s).AsSlice() }
	tests := []struct {
		name string
		list *GeoIPList
		want map[string][]string
	}{
		{
			name: "duplicated country codes are merged, names normalized",
			list: &GeoIPList{Entry: []*GeoIP{
				{CountryCode: " cn ", Cidr: []*CIDR{{Ip: []byte{1, 0, 0, 0}, Prefix: 24}}},
				{CountryCode: "CN", Cidr: []*CIDR{{Ip: []byte{1, 0, 1, 0}, Prefix: 24}, {Ip: mapped("2001:db8::"), Prefix: 32}}},
			}},
			want: map[string][]string{"CN": {"1.0.0.0/23", "2001:db8::/32"}},
		},
		{
			name: "ipv4-mapped ipv6 with ipv6 prefix length",
			list: &GeoIPList{Entry: []*GeoIP{
				{CountryCode: "m", Cidr: []*CIDR{{Ip: mapped("::ffff:1.2.3.0"), Prefix: 120}}},
			}},
			want: map[string][]string{"M": {"1.2.3.0/24"}},
		},
		{
			name: "ipv4-mapped ipv6 with ipv4 prefix length",
			list: &GeoIPList{Entry: []*GeoIP{
				{CountryCode: "m", Cidr: []*CIDR{{Ip: mapped("::ffff:1.2.3.0"), Prefix: 24}}},
			}},
			want: map[string][]string{"M": {"1.2.3.0/24"}},
		},
		{
			name: "non-canonical host bits are masked",
			list: &GeoIPList{Entry: []*GeoIP{
				{CountryCode: "h", Cidr: []*CIDR{{Ip: []byte{10, 1, 2, 3}, Prefix: 8}}},
			}},
			want: map[string][]string{"H": {"10.0.0.0/8"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in, _ := NewGeoIPDatIn(lib.ActionAdd, WithURI(writeDat(t, tt.list)))
			c, err := in.Input(lib.NewContainer())
			if err != nil {
				t.Fatal(err)
			}
			assertContainer(t, c, tt.want)
		})
	}
}

func TestDatInputErrors(t *testing.T) {
	dir := t.TempDir()
	garbage := filepath.Join(dir, "garbage.dat")
	if err := os.WriteFile(garbage, []byte{0xff, 0xff}, 0644); err != nil {
		t.Fatal(err)
	}
	good := writeDat(t, &GeoIPList{Entry: []*GeoIP{{CountryCode: "CN", Cidr: []*CIDR{{Ip: []byte{1, 0, 0, 0}, Prefix: 24}}}}})

	notFound := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(notFound.Close)
	closed := httptest.NewServer(http.NotFoundHandler())
	closedURL := closed.URL
	closed.Close()

	tests := []struct {
		name   string
		uri    string
		action lib.Action
		opts   []lib.InputOption
	}{
		{name: "missing local file", uri: filepath.Join(dir, "missing.dat"), action: lib.ActionAdd},
		{name: "directory instead of file", uri: dir, action: lib.ActionAdd},
		{name: "http 404", uri: notFound.URL + "/x.dat", action: lib.ActionAdd},
		{name: "http connection refused", uri: closedURL + "/x.dat", action: lib.ActionAdd},
		{name: "invalid protobuf", uri: garbage, action: lib.ActionAdd},
		{name: "invalid ip length", uri: writeDat(t, &GeoIPList{Entry: []*GeoIP{{CountryCode: "x", Cidr: []*CIDR{{Ip: []byte{1, 2, 3}, Prefix: 8}}}}}), action: lib.ActionAdd},
		{name: "invalid prefix length", uri: writeDat(t, &GeoIPList{Entry: []*GeoIP{{CountryCode: "x", Cidr: []*CIDR{{Ip: []byte{1, 2, 3, 4}, Prefix: 40}}}}}), action: lib.ActionAdd},
		{name: "empty list", uri: writeDat(t, &GeoIPList{}), action: lib.ActionAdd},
		{name: "wanted list filters everything", uri: good, action: lib.ActionAdd, opts: []lib.InputOption{WithInputWantedList([]string{"us"})}},
		{name: "remove from container without entry", uri: good, action: lib.ActionRemove},
		{name: "unknown action", uri: good, action: lib.Action("unknown")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in, err := NewGeoIPDatIn(tt.action, append([]lib.InputOption{WithURI(tt.uri)}, tt.opts...)...)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := in.Input(lib.NewContainer()); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestDatInputActions(t *testing.T) {
	tests := []struct {
		name     string
		action   lib.Action
		fileDefs []entryDef
		onlyType lib.IPType
		want     map[string][]string
	}{
		{
			name:     "add merges into existing entry",
			action:   lib.ActionAdd,
			fileDefs: []entryDef{{"cn", []string{"1.0.2.0/23", "2001:db9::/32"}}},
			want:     map[string][]string{"CN": {"1.0.0.0/22", "2001:db8::/31"}},
		},
		{
			name:     "remove prefixes",
			action:   lib.ActionRemove,
			fileDefs: []entryDef{{"cn", []string{"1.0.1.0/24", "2001:db8:8000::/33"}}},
			want:     map[string][]string{"CN": {"1.0.0.0/24", "2001:db8::/33"}},
		},
		{
			name:     "remove only ipv6 prefixes",
			action:   lib.ActionRemove,
			fileDefs: []entryDef{{"cn", []string{"1.0.1.0/24", "2001:db8:8000::/33"}}},
			onlyType: lib.IPv6,
			want:     map[string][]string{"CN": {"1.0.0.0/23", "2001:db8::/33"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			out, _ := NewGeoIPDatOut(lib.ActionOutput, WithOutputDir(dir))
			if err := out.Output(newContainer(t, tt.fileDefs...)); err != nil {
				t.Fatal(err)
			}
			in, _ := NewGeoIPDatIn(tt.action, WithURI(filepath.Join(dir, defaultOutputName)), WithInputOnlyIPType(tt.onlyType))
			c, err := in.Input(newContainer(t, cnDef))
			if err != nil {
				t.Fatal(err)
			}
			assertContainer(t, c, tt.want)
		})
	}
}
