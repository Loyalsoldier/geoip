package singbox

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/sagernet/sing-box/common/srs"
	"github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
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

func serveFile(t *testing.T, path string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path)
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/" + filepath.Base(path)
}

func writeSRS(t *testing.T, path string, rules ...option.HeadlessRule) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var rs option.PlainRuleSet
	rs.Rules = rules
	if err := srs.Write(f, rs, constant.RuleSetVersion1); err != nil {
		t.Fatal(err)
	}
}

func ipRule(cidrs ...string) option.HeadlessRule {
	var r option.DefaultHeadlessRule
	r.IPCIDR = cidrs
	return option.HeadlessRule{Type: constant.RuleTypeDefault, DefaultOptions: r}
}

func domainRule(domains ...string) option.HeadlessRule {
	var r option.DefaultHeadlessRule
	r.Domain = domains
	return option.HeadlessRule{Type: constant.RuleTypeDefault, DefaultOptions: r}
}

func readSRS(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rs, err := srs.Read(f, true)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, r := range rs.Options.Rules {
		out = append(out, r.DefaultOptions.IPCIDR...)
	}
	return out
}

var (
	cnDef = entryDef{"cn", []string{"1.0.0.0/24", "1.0.1.0/24", "2001:db8::/33", "2001:db8:8000::/33"}}
	usDef = entryDef{"us", []string{"8.8.8.0/24", "2001:4860::/32"}}
	v6Def = entryDef{"v6only", []string{"2400:cb00::/32"}}

	cnWant = []string{"1.0.0.0/23", "2001:db8::/32"}
	usWant = []string{"8.8.8.0/24", "2001:4860::/32"}
)

// ---------- constructors ----------

func TestNewSRSIn(t *testing.T) {
	tests := []struct {
		name     string
		opts     []lib.InputOption
		wantErr  bool
		wantName string
		wantURI  string
		wantDir  string
		wantWant []string
		wantType lib.IPType
	}{
		{name: "nothing", wantErr: true},
		{name: "blank everything", opts: []lib.InputOption{WithNameAndURI(" ", " "), WithInputDir(" ")}, wantErr: true},
		{name: "name without uri", opts: []lib.InputOption{WithNameAndURI("cn", "")}, wantErr: true},
		{name: "uri without name", opts: []lib.InputOption{WithNameAndURI("", "cn.srs")}, wantErr: true},
		{name: "name without uri but with dir", opts: []lib.InputOption{WithNameAndURI("cn", ""), WithInputDir("d")}, wantErr: true},
		{name: "input dir", opts: []lib.InputOption{nil, WithInputDir(" d ")}, wantDir: "d"},
		{
			name: "name and uri with options",
			opts: []lib.InputOption{
				WithNameAndURI(" cn ", " cn.srs "),
				WithInputWantedList([]string{" cn", "", "Us "}),
				WithInputOnlyIPType(lib.IPv4),
			},
			wantName: "cn", wantURI: "cn.srs", wantWant: []string{"CN", "US"}, wantType: lib.IPv4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewSRSIn(lib.ActionAdd, tt.opts...)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			s := conv.(*srs_in)
			if s.Name != tt.wantName || s.URI != tt.wantURI || s.InputDir != tt.wantDir || s.OnlyIPType != tt.wantType {
				t.Errorf("unexpected converter %+v", s)
			}
			if len(s.Want) != len(tt.wantWant) {
				t.Errorf("Want = %v, want %v", s.Want, tt.wantWant)
			}
			for _, w := range tt.wantWant {
				if !s.Want[w] {
					t.Errorf("Want missing %s", w)
				}
			}
			if conv.GetType() != TypeSRSIn || conv.GetAction() != lib.ActionAdd || conv.GetDescription() != DescSRSIn {
				t.Errorf("getters = %q %q %q", conv.GetType(), conv.GetAction(), conv.GetDescription())
			}
		})
	}
}

func TestNewSRSInFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{name: "empty data", data: "", wantErr: true},
		{name: "invalid json", data: "{", wantErr: true},
		{name: "name only", data: `{"name":"cn"}`, wantErr: true},
		{name: "valid", data: `{"name":"cn","uri":"cn.srs","inputDir":"d","wantedList":["cn"],"onlyIPType":"ipv6"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewSRSInFromBytes(lib.ActionRemove, []byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			s := conv.(*srs_in)
			if s.Name != "cn" || s.URI != "cn.srs" || s.InputDir != "d" || !s.Want["CN"] || s.OnlyIPType != lib.IPv6 || s.Action != lib.ActionRemove {
				t.Errorf("unexpected converter %+v", s)
			}
		})
	}
}

func TestNewSRSOut(t *testing.T) {
	tests := []struct {
		name     string
		opts     []lib.OutputOption
		wantDir  string
		wantType lib.IPType
		wantWLen int
		wantELen int
	}{
		{name: "defaults", wantDir: defaultOutputDir},
		{name: "blank dir falls back", opts: []lib.OutputOption{nil, WithOutputDir("  ")}, wantDir: defaultOutputDir},
		{
			name: "all options",
			opts: []lib.OutputOption{
				WithOutputDir(" o "), WithOutputWantedList([]string{"a"}),
				WithOutputExcludedList([]string{"b", "c"}), WithOutputOnlyIPType(lib.IPv6),
			},
			wantDir: "o", wantType: lib.IPv6, wantWLen: 1, wantELen: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewSRSOut(lib.ActionOutput, tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			s := conv.(*srs_out)
			if s.OutputDir != tt.wantDir || s.OnlyIPType != tt.wantType || len(s.Want) != tt.wantWLen || len(s.Exclude) != tt.wantELen {
				t.Errorf("unexpected converter %+v", s)
			}
			if conv.GetType() != TypeSRSOut || conv.GetAction() != lib.ActionOutput || conv.GetDescription() != DescSRSOut {
				t.Errorf("getters = %q %q %q", conv.GetType(), conv.GetAction(), conv.GetDescription())
			}
		})
	}
}

func TestNewSRSOutFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		wantDir string
	}{
		{name: "empty data", data: "", wantDir: defaultOutputDir},
		{name: "invalid json", data: "[", wantErr: true},
		{name: "valid", data: `{"outputDir":"d","wantedList":["cn"],"excludedList":["us"],"onlyIPType":"ipv4"}`, wantDir: "d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewSRSOutFromBytes(lib.ActionOutput, []byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if s := conv.(*srs_out); s.OutputDir != tt.wantDir {
				t.Errorf("OutputDir = %q, want %q", s.OutputDir, tt.wantDir)
			}
		})
	}
}

func TestConfigRegistration(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	out, _ := NewSRSOut(lib.ActionOutput, WithOutputDir(src))
	if err := out.Output(newContainer(t, cnDef, usDef)); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "dst")
	conf := fmt.Sprintf(`{
		"input": [{"type": %q, "action": "add", "args": {"inputDir": %q}}],
		"output": [{"type": %q, "action": "output", "args": {"outputDir": %q}}]
	}`, TypeSRSIn, src, TypeSRSOut, dst)

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
	in, _ := NewSRSIn(lib.ActionAdd, WithInputDir(dst))
	c, err := in.Input(lib.NewContainer())
	if err != nil {
		t.Fatal(err)
	}
	assertContainer(t, c, map[string][]string{"CN": cnWant, "US": usWant})
}

// ---------- round trip ----------

func TestSRSRoundTrip(t *testing.T) {
	type mode int
	const (
		viaDir mode = iota
		viaFile
		viaHTTP
	)
	tests := []struct {
		name      string
		defs      []entryDef
		outOpts   []lib.OutputOption
		inOpts    []lib.InputOption
		mode      mode
		wantFiles []string
		want      map[string][]string
	}{
		{
			name:      "all entries via dir",
			defs:      []entryDef{cnDef, usDef, v6Def},
			wantFiles: []string{"cn.srs", "us.srs", "v6only.srs"},
			want:      map[string][]string{"CN": cnWant, "US": usWant, "V6ONLY": {"2400:cb00::/32"}},
		},
		{
			name:      "single entry via local file",
			defs:      []entryDef{cnDef},
			mode:      viaFile,
			wantFiles: []string{"cn.srs"},
			want:      map[string][]string{"CN": cnWant},
		},
		{
			name:      "single entry via http",
			defs:      []entryDef{cnDef},
			mode:      viaHTTP,
			wantFiles: []string{"cn.srs"},
			want:      map[string][]string{"CN": cnWant},
		},
		{
			name:      "output wanted list with blank and missing entries",
			defs:      []entryDef{cnDef, usDef},
			outOpts:   []lib.OutputOption{WithOutputWantedList([]string{" US ", "", "zz"})},
			wantFiles: []string{"us.srs"},
			want:      map[string][]string{"US": usWant},
		},
		{
			name:      "output excluded list",
			defs:      []entryDef{cnDef, usDef, v6Def},
			outOpts:   []lib.OutputOption{WithOutputExcludedList([]string{"cn", " ", "v6only"})},
			wantFiles: []string{"us.srs"},
			want:      map[string][]string{"US": usWant},
		},
		{
			name:      "output wanted minus excluded",
			defs:      []entryDef{cnDef, usDef},
			outOpts:   []lib.OutputOption{WithOutputWantedList([]string{"cn", "us"}), WithOutputExcludedList([]string{"us"})},
			wantFiles: []string{"cn.srs"},
			want:      map[string][]string{"CN": cnWant},
		},
		{
			name:      "output only ipv4",
			defs:      []entryDef{cnDef, usDef},
			outOpts:   []lib.OutputOption{WithOutputOnlyIPType(lib.IPv4)},
			wantFiles: []string{"cn.srs", "us.srs"},
			want:      map[string][]string{"CN": {"1.0.0.0/23"}, "US": {"8.8.8.0/24"}},
		},
		{
			name:      "output only ipv6",
			defs:      []entryDef{cnDef, v6Def},
			outOpts:   []lib.OutputOption{WithOutputOnlyIPType(lib.IPv6)},
			wantFiles: []string{"cn.srs", "v6only.srs"},
			want:      map[string][]string{"CN": {"2001:db8::/32"}, "V6ONLY": {"2400:cb00::/32"}},
		},
		{
			name:      "input wanted list via dir",
			defs:      []entryDef{cnDef, usDef},
			inOpts:    []lib.InputOption{WithInputWantedList([]string{"cn"})},
			wantFiles: []string{"cn.srs", "us.srs"},
			want:      map[string][]string{"CN": cnWant},
		},
		{
			name:      "input only ipv6 via dir",
			defs:      []entryDef{cnDef, usDef},
			inOpts:    []lib.InputOption{WithInputOnlyIPType(lib.IPv6)},
			wantFiles: []string{"cn.srs", "us.srs"},
			want:      map[string][]string{"CN": {"2001:db8::/32"}, "US": {"2001:4860::/32"}},
		},
		{
			name:      "input only ipv4 via http",
			defs:      []entryDef{cnDef},
			inOpts:    []lib.InputOption{WithInputOnlyIPType(lib.IPv4)},
			mode:      viaHTTP,
			wantFiles: []string{"cn.srs"},
			want:      map[string][]string{"CN": {"1.0.0.0/23"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			out, err := NewSRSOut(lib.ActionOutput, append([]lib.OutputOption{WithOutputDir(dir)}, tt.outOpts...)...)
			if err != nil {
				t.Fatal(err)
			}
			if err := out.Output(newContainer(t, tt.defs...)); err != nil {
				t.Fatalf("Output: %v", err)
			}

			des, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			files := []string{}
			for _, de := range des {
				files = append(files, de.Name())
			}
			compareStrings(t, "output files", files, tt.wantFiles)

			var src []lib.InputOption
			switch tt.mode {
			case viaDir:
				src = []lib.InputOption{WithInputDir(dir)}
			case viaFile:
				src = []lib.InputOption{WithNameAndURI("cn", filepath.Join(dir, "cn.srs"))}
			case viaHTTP:
				src = []lib.InputOption{WithNameAndURI("cn", serveFile(t, filepath.Join(dir, "cn.srs")))}
			}
			in, err := NewSRSIn(lib.ActionAdd, append(src, tt.inOpts...)...)
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

func TestSRSOutputFileContent(t *testing.T) {
	dir := t.TempDir()
	out, _ := NewSRSOut(lib.ActionOutput, WithOutputDir(dir))
	if err := out.Output(newContainer(t, cnDef)); err != nil {
		t.Fatal(err)
	}
	compareStrings(t, "cn.srs ip_cidr", readSRS(t, filepath.Join(dir, "cn.srs")), cnWant)
}

func TestSRSOutputNoFile(t *testing.T) {
	tests := []struct {
		name string
		defs []entryDef
		opts []lib.OutputOption
	}{
		{name: "empty container"},
		{name: "wanted all excluded", defs: []entryDef{cnDef}, opts: []lib.OutputOption{
			WithOutputWantedList([]string{"cn"}), WithOutputExcludedList([]string{"CN"}),
		}},
		{name: "wanted entry missing", defs: []entryDef{cnDef}, opts: []lib.OutputOption{WithOutputWantedList([]string{"zz"})}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "out")
			out, _ := NewSRSOut(lib.ActionOutput, append([]lib.OutputOption{WithOutputDir(dir)}, tt.opts...)...)
			if err := out.Output(newContainer(t, tt.defs...)); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(dir); !os.IsNotExist(err) {
				t.Errorf("output dir should not exist, stat err = %v", err)
			}
		})
	}
}

func TestSRSOutputErrors(t *testing.T) {
	base := t.TempDir()
	blocker := filepath.Join(base, "file")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	// A directory occupying the target file name makes os.Create fail.
	occupied := filepath.Join(base, "occupied")
	if err := os.MkdirAll(filepath.Join(occupied, "cn.srs"), 0755); err != nil {
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
		{name: "output file is a directory", defs: []entryDef{cnDef}, opts: []lib.OutputOption{WithOutputDir(occupied)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _ := NewSRSOut(lib.ActionOutput, append([]lib.OutputOption{WithOutputDir(t.TempDir())}, tt.opts...)...)
			if err := out.Output(newContainer(t, tt.defs...)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

// ---------- input ----------

func TestSRSInputHandcrafted(t *testing.T) {
	tests := []struct {
		name  string
		files map[string][]option.HeadlessRule
		want  map[string][]string
	}{
		{
			name: "non-ip rules ignored and multiple ip rules merged",
			files: map[string][]option.HeadlessRule{
				"cn.srs": {domainRule("example.com"), ipRule("1.0.0.0/24"), ipRule("1.0.1.0/24", "2001:db8::/32")},
			},
			want: map[string][]string{"CN": {"1.0.0.0/23", "2001:db8::/32"}},
		},
		{
			name: "file name normalisation, nested dirs, hidden and extension-less files",
			files: map[string][]option.HeadlessRule{
				"a-b_c.d.srs":    {ipRule("10.0.0.0/8")},
				"sub/noext":      {ipRule("fd00::/8")},
				".hidden":        {ipRule("192.168.0.0/16")},
				"sub/deeper/x.y": {ipRule("172.16.0.0/12")},
			},
			want: map[string][]string{
				"A-B_C.D": {"10.0.0.0/8"},
				"NOEXT":   {"fd00::/8"},
				".HIDDEN": {"192.168.0.0/16"},
				"X":       {"172.16.0.0/12"},
			},
		},
		{
			name: "domain-only file yields empty entry",
			files: map[string][]option.HeadlessRule{
				"cn.srs":     {ipRule("1.0.0.0/24")},
				"domain.srs": {domainRule("example.com")},
			},
			want: map[string][]string{"CN": {"1.0.0.0/24"}, "DOMAIN": nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, rules := range tt.files {
				writeSRS(t, filepath.Join(dir, name), rules...)
			}
			in, _ := NewSRSIn(lib.ActionAdd, WithInputDir(dir))
			c, err := in.Input(lib.NewContainer())
			if err != nil {
				t.Fatal(err)
			}
			assertContainer(t, c, tt.want)
		})
	}
}

func TestSRSInputErrors(t *testing.T) {
	base := t.TempDir()
	good := filepath.Join(base, "good", "cn.srs")
	writeSRS(t, good, ipRule("1.0.0.0/24"))

	garbage := filepath.Join(base, "garbage", "bad.srs")
	if err := os.MkdirAll(filepath.Dir(garbage), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(garbage, []byte("not an srs file"), 0644); err != nil {
		t.Fatal(err)
	}

	badName := filepath.Join(base, "badname")
	writeSRS(t, filepath.Join(badName, "c n.srs"), ipRule("1.0.0.0/24"))

	dup := filepath.Join(base, "dup")
	writeSRS(t, filepath.Join(dup, "cn.srs"), ipRule("1.0.0.0/24"))
	writeSRS(t, filepath.Join(dup, "sub", "CN.srs"), ipRule("2.0.0.0/24"))

	notFound := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(notFound.Close)
	closed := httptest.NewServer(http.NotFoundHandler())
	closedURL := closed.URL
	closed.Close()

	tests := []struct {
		name   string
		action lib.Action
		opts   []lib.InputOption
	}{
		{name: "missing dir", action: lib.ActionAdd, opts: []lib.InputOption{WithInputDir(filepath.Join(base, "missing"))}},
		{name: "invalid file name", action: lib.ActionAdd, opts: []lib.InputOption{WithInputDir(badName)}},
		{name: "duplicated list in dir", action: lib.ActionAdd, opts: []lib.InputOption{WithInputDir(dup)}},
		{name: "garbage file in dir", action: lib.ActionAdd, opts: []lib.InputOption{WithInputDir(filepath.Dir(garbage))}},
		{name: "missing local file", action: lib.ActionAdd, opts: []lib.InputOption{WithNameAndURI("cn", filepath.Join(base, "nope.srs"))}},
		{name: "http 404", action: lib.ActionAdd, opts: []lib.InputOption{WithNameAndURI("cn", notFound.URL+"/cn.srs")}},
		{name: "http connection refused", action: lib.ActionAdd, opts: []lib.InputOption{WithNameAndURI("cn", closedURL+"/cn.srs")}},
		{name: "garbage via http", action: lib.ActionAdd, opts: []lib.InputOption{WithNameAndURI("cn", serveFile(t, garbage))}},
		{name: "wanted list filters everything", action: lib.ActionAdd, opts: []lib.InputOption{WithNameAndURI("cn", good), WithInputWantedList([]string{"us"})}},
		{name: "remove from container without entry", action: lib.ActionRemove, opts: []lib.InputOption{WithNameAndURI("cn", good)}},
		{name: "unknown action", action: lib.Action("unknown"), opts: []lib.InputOption{WithNameAndURI("cn", good)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in, err := NewSRSIn(tt.action, tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := in.Input(lib.NewContainer()); err == nil {
				t.Fatal("expected error")
			}
		})
	}

	t.Run("struct without source", func(t *testing.T) {
		s := &srs_in{Type: TypeSRSIn, Action: lib.ActionAdd}
		if _, err := s.Input(lib.NewContainer()); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestSRSInputActions(t *testing.T) {
	tests := []struct {
		name     string
		action   lib.Action
		cidrs    []string
		onlyType lib.IPType
		want     map[string][]string
	}{
		{
			name:   "add merges into existing entry",
			action: lib.ActionAdd,
			cidrs:  []string{"1.0.2.0/23", "2001:db9::/32"},
			want:   map[string][]string{"CN": {"1.0.0.0/22", "2001:db8::/31"}},
		},
		{
			name:   "remove prefixes",
			action: lib.ActionRemove,
			cidrs:  []string{"1.0.1.0/24", "2001:db8:8000::/33"},
			want:   map[string][]string{"CN": {"1.0.0.0/24", "2001:db8::/33"}},
		},
		{
			name:     "remove only ipv4 prefixes",
			action:   lib.ActionRemove,
			cidrs:    []string{"1.0.1.0/24", "2001:db8:8000::/33"},
			onlyType: lib.IPv4,
			want:     map[string][]string{"CN": {"1.0.0.0/24", "2001:db8::/32"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			out, _ := NewSRSOut(lib.ActionOutput, WithOutputDir(dir))
			if err := out.Output(newContainer(t, entryDef{"cn", tt.cidrs})); err != nil {
				t.Fatal(err)
			}
			in, _ := NewSRSIn(tt.action, WithNameAndURI("cn", filepath.Join(dir, "cn.srs")), WithInputOnlyIPType(tt.onlyType))
			c, err := in.Input(newContainer(t, cnDef))
			if err != nil {
				t.Fatal(err)
			}
			assertContainer(t, c, tt.want)
		})
	}
}
