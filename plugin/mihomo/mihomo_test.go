package mihomo

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/klauspost/compress/zstd"
	"go4.org/netipx"
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

func a16(s string) [16]byte { return netip.MustParseAddr(s).As16() }

// rawMRS serialises parts ([]byte, byte, int64, [16]byte) and zstd-compresses them.
func rawMRS(t *testing.T, parts ...any) []byte {
	t.Helper()
	var buf bytes.Buffer
	for _, p := range parts {
		switch v := p.(type) {
		case []byte:
			buf.Write(v)
		case byte:
			buf.WriteByte(v)
		case int64, [16]byte:
			if err := binary.Write(&buf, binary.BigEndian, v); err != nil {
				t.Fatal(err)
			}
		default:
			t.Fatalf("unsupported part %T", p)
		}
	}
	enc, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer enc.Close()
	return enc.EncodeAll(buf.Bytes(), nil)
}

func writeFile(t *testing.T, path string, data []byte) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

var (
	magic = mrsMagicBytes[:]

	cnDef = entryDef{"cn", []string{"1.0.0.0/24", "1.0.1.0/24", "2001:db8::/33", "2001:db8:8000::/33"}}
	usDef = entryDef{"us", []string{"8.8.8.0/24", "2001:4860::/32"}}
	v6Def = entryDef{"v6only", []string{"2400:cb00::/32"}}

	cnWant = []string{"1.0.0.0/23", "2001:db8::/32"}
	usWant = []string{"8.8.8.0/24", "2001:4860::/32"}
)

// ---------- constructors ----------

func TestNewMRSIn(t *testing.T) {
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
		{name: "uri without name", opts: []lib.InputOption{WithNameAndURI("", "cn.mrs")}, wantErr: true},
		{name: "uri without name but with dir", opts: []lib.InputOption{WithNameAndURI("", "cn.mrs"), WithInputDir("d")}, wantErr: true},
		{name: "input dir", opts: []lib.InputOption{nil, WithInputDir(" d ")}, wantDir: "d"},
		{
			name: "name and uri with options",
			opts: []lib.InputOption{
				WithNameAndURI(" cn ", " cn.mrs "),
				WithInputWantedList([]string{" cn", "", "Us "}),
				WithInputOnlyIPType(lib.IPv6),
			},
			wantName: "cn", wantURI: "cn.mrs", wantWant: []string{"CN", "US"}, wantType: lib.IPv6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewMRSIn(lib.ActionAdd, tt.opts...)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			m := conv.(*mrs_in)
			if m.Name != tt.wantName || m.URI != tt.wantURI || m.InputDir != tt.wantDir || m.OnlyIPType != tt.wantType {
				t.Errorf("unexpected converter %+v", m)
			}
			if len(m.Want) != len(tt.wantWant) {
				t.Errorf("Want = %v, want %v", m.Want, tt.wantWant)
			}
			for _, w := range tt.wantWant {
				if !m.Want[w] {
					t.Errorf("Want missing %s", w)
				}
			}
			if conv.GetType() != TypeMRSIn || conv.GetAction() != lib.ActionAdd || conv.GetDescription() != DescMRSIn {
				t.Errorf("getters = %q %q %q", conv.GetType(), conv.GetAction(), conv.GetDescription())
			}
		})
	}
}

func TestNewMRSInFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{name: "empty data", data: "", wantErr: true},
		{name: "invalid json", data: "{", wantErr: true},
		{name: "uri only", data: `{"uri":"cn.mrs"}`, wantErr: true},
		{name: "valid", data: `{"name":"cn","uri":"cn.mrs","inputDir":"d","wantedList":["cn"],"onlyIPType":"ipv4"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewMRSInFromBytes(lib.ActionRemove, []byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			m := conv.(*mrs_in)
			if m.Name != "cn" || m.URI != "cn.mrs" || m.InputDir != "d" || !m.Want["CN"] || m.OnlyIPType != lib.IPv4 || m.Action != lib.ActionRemove {
				t.Errorf("unexpected converter %+v", m)
			}
		})
	}
}

func TestNewMRSOut(t *testing.T) {
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
				WithOutputDir(" o "), WithOutputWantedList([]string{"a", "b"}),
				WithOutputExcludedList([]string{"c"}), WithOutputOnlyIPType(lib.IPv4),
			},
			wantDir: "o", wantType: lib.IPv4, wantWLen: 2, wantELen: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewMRSOut(lib.ActionOutput, tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			m := conv.(*mrs_out)
			if m.OutputDir != tt.wantDir || m.OnlyIPType != tt.wantType || len(m.Want) != tt.wantWLen || len(m.Exclude) != tt.wantELen {
				t.Errorf("unexpected converter %+v", m)
			}
			if conv.GetType() != TypeMRSOut || conv.GetAction() != lib.ActionOutput || conv.GetDescription() != DescMRSOut {
				t.Errorf("getters = %q %q %q", conv.GetType(), conv.GetAction(), conv.GetDescription())
			}
		})
	}
}

func TestNewMRSOutFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		wantDir string
	}{
		{name: "empty data", data: "", wantDir: defaultOutputDir},
		{name: "invalid json", data: "[", wantErr: true},
		{name: "valid", data: `{"outputDir":"d","wantedList":["cn"],"excludedList":["us"],"onlyIPType":"ipv6"}`, wantDir: "d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewMRSOutFromBytes(lib.ActionOutput, []byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if m := conv.(*mrs_out); m.OutputDir != tt.wantDir {
				t.Errorf("OutputDir = %q, want %q", m.OutputDir, tt.wantDir)
			}
		})
	}
}

func TestConfigRegistration(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	out, _ := NewMRSOut(lib.ActionOutput, WithOutputDir(src))
	if err := out.Output(newContainer(t, cnDef, usDef)); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "dst")
	conf := fmt.Sprintf(`{
		"input": [{"type": %q, "action": "add", "args": {"inputDir": %q}}],
		"output": [{"type": %q, "action": "output", "args": {"outputDir": %q}}]
	}`, TypeMRSIn, src, TypeMRSOut, dst)

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
	in, _ := NewMRSIn(lib.ActionAdd, WithInputDir(dst))
	c, err := in.Input(lib.NewContainer())
	if err != nil {
		t.Fatal(err)
	}
	assertContainer(t, c, map[string][]string{"CN": cnWant, "US": usWant})
}

// ---------- round trip ----------

func TestMRSRoundTrip(t *testing.T) {
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
			wantFiles: []string{"cn.mrs", "us.mrs", "v6only.mrs"},
			want:      map[string][]string{"CN": cnWant, "US": usWant, "V6ONLY": {"2400:cb00::/32"}},
		},
		{
			name:      "non-prefix-aligned ranges",
			defs:      []entryDef{{"odd", []string{"10.0.0.1/32", "10.0.0.2/31", "10.0.0.4/30", "::1/128", "::2/127"}}},
			wantFiles: []string{"odd.mrs"},
			want:      map[string][]string{"ODD": {"10.0.0.1/32", "10.0.0.2/31", "10.0.0.4/30", "::1/128", "::2/127"}},
		},
		{
			name:      "single entry via local file",
			defs:      []entryDef{cnDef},
			mode:      viaFile,
			wantFiles: []string{"cn.mrs"},
			want:      map[string][]string{"CN": cnWant},
		},
		{
			name:      "single entry via http",
			defs:      []entryDef{cnDef},
			mode:      viaHTTP,
			wantFiles: []string{"cn.mrs"},
			want:      map[string][]string{"CN": cnWant},
		},
		{
			name:      "output wanted list with blank and missing entries",
			defs:      []entryDef{cnDef, usDef},
			outOpts:   []lib.OutputOption{WithOutputWantedList([]string{" cn ", "", "zz"})},
			wantFiles: []string{"cn.mrs"},
			want:      map[string][]string{"CN": cnWant},
		},
		{
			name:      "output excluded list",
			defs:      []entryDef{cnDef, usDef, v6Def},
			outOpts:   []lib.OutputOption{WithOutputExcludedList([]string{"US", " ", "v6only"})},
			wantFiles: []string{"cn.mrs"},
			want:      map[string][]string{"CN": cnWant},
		},
		{
			name:      "output wanted minus excluded",
			defs:      []entryDef{cnDef, usDef},
			outOpts:   []lib.OutputOption{WithOutputWantedList([]string{"cn", "us"}), WithOutputExcludedList([]string{"cn"})},
			wantFiles: []string{"us.mrs"},
			want:      map[string][]string{"US": usWant},
		},
		{
			name:      "output only ipv4",
			defs:      []entryDef{cnDef, usDef},
			outOpts:   []lib.OutputOption{WithOutputOnlyIPType(lib.IPv4)},
			wantFiles: []string{"cn.mrs", "us.mrs"},
			want:      map[string][]string{"CN": {"1.0.0.0/23"}, "US": {"8.8.8.0/24"}},
		},
		{
			name:      "output only ipv6",
			defs:      []entryDef{cnDef, v6Def},
			outOpts:   []lib.OutputOption{WithOutputOnlyIPType(lib.IPv6)},
			wantFiles: []string{"cn.mrs", "v6only.mrs"},
			want:      map[string][]string{"CN": {"2001:db8::/32"}, "V6ONLY": {"2400:cb00::/32"}},
		},
		{
			name:      "input wanted list via dir",
			defs:      []entryDef{cnDef, usDef},
			inOpts:    []lib.InputOption{WithInputWantedList([]string{"us"})},
			wantFiles: []string{"cn.mrs", "us.mrs"},
			want:      map[string][]string{"US": usWant},
		},
		{
			name:      "input only ipv4 via dir",
			defs:      []entryDef{cnDef, usDef},
			inOpts:    []lib.InputOption{WithInputOnlyIPType(lib.IPv4)},
			wantFiles: []string{"cn.mrs", "us.mrs"},
			want:      map[string][]string{"CN": {"1.0.0.0/23"}, "US": {"8.8.8.0/24"}},
		},
		{
			name:      "input only ipv6 via http",
			defs:      []entryDef{cnDef},
			inOpts:    []lib.InputOption{WithInputOnlyIPType(lib.IPv6)},
			mode:      viaHTTP,
			wantFiles: []string{"cn.mrs"},
			want:      map[string][]string{"CN": {"2001:db8::/32"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			out, err := NewMRSOut(lib.ActionOutput, append([]lib.OutputOption{WithOutputDir(dir)}, tt.outOpts...)...)
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
				src = []lib.InputOption{WithNameAndURI("cn", filepath.Join(dir, "cn.mrs"))}
			case viaHTTP:
				src = []lib.InputOption{WithNameAndURI("cn", serveFile(t, filepath.Join(dir, "cn.mrs")))}
			}
			in, err := NewMRSIn(lib.ActionAdd, append(src, tt.inOpts...)...)
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

func TestMRSOutputFileLayout(t *testing.T) {
	dir := t.TempDir()
	out, _ := NewMRSOut(lib.ActionOutput, WithOutputDir(dir))
	if err := out.Output(newContainer(t, cnDef)); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(dir, "cn.mrs"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dec, err := zstd.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer dec.Close()
	raw, err := io.ReadAll(dec)
	if err != nil {
		t.Fatal(err)
	}

	want := rawMRSPlain(t,
		magic, byte(1), int64(2), int64(0), byte(1), int64(2),
		a16("1.0.0.0"), a16("1.0.1.255"),
		a16("2001:db8::"), a16("2001:db8:ffff:ffff:ffff:ffff:ffff:ffff"),
	)
	if !bytes.Equal(raw, want) {
		t.Fatalf("decoded payload mismatch:\n got %x\nwant %x", raw, want)
	}
}

// rawMRSPlain is rawMRS without compression.
func rawMRSPlain(t *testing.T, parts ...any) []byte {
	t.Helper()
	dec, err := zstd.NewReader(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dec.Close()
	out, err := dec.DecodeAll(rawMRS(t, parts...), nil)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestConvertToMrs(t *testing.T) {
	tests := []struct {
		name   string
		ranges []netipx.IPRange
		want   []string
	}{
		{
			name: "ipv4 and ipv6 ranges",
			ranges: []netipx.IPRange{
				netipx.MustParseIPRange("1.0.0.0-1.0.1.255"),
				netipx.MustParseIPRange("2001:db8::-2001:db8::ffff"),
			},
			want: []string{"1.0.0.0/23", "2001:db8::/112"},
		},
		{
			name:   "single ipv4 range split into prefixes",
			ranges: []netipx.IPRange{netipx.MustParseIPRange("10.0.0.1-10.0.0.6")},
			want:   []string{"10.0.0.1/32", "10.0.0.2/31", "10.0.0.4/31", "10.0.0.6/32"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			m := &mrs_out{Type: TypeMRSOut}
			if err := m.convertToMrs(tt.ranges, &buf); err != nil {
				t.Fatal(err)
			}
			in := &mrs_in{}
			e := lib.NewEntry("x")
			if err := in.parseMRS(buf.Bytes(), e); err != nil {
				t.Fatal(err)
			}
			got, err := e.MarshalText()
			if err != nil {
				t.Fatal(err)
			}
			compareStrings(t, "prefixes", got, tt.want)
		})
	}
}

func TestMRSOutputNoFile(t *testing.T) {
	tests := []struct {
		name string
		defs []entryDef
		opts []lib.OutputOption
	}{
		{name: "empty container"},
		{name: "wanted all excluded", defs: []entryDef{cnDef}, opts: []lib.OutputOption{
			WithOutputWantedList([]string{"cn"}), WithOutputExcludedList([]string{"cn"}),
		}},
		{name: "wanted entry missing", defs: []entryDef{cnDef}, opts: []lib.OutputOption{WithOutputWantedList([]string{"zz"})}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "out")
			out, _ := NewMRSOut(lib.ActionOutput, append([]lib.OutputOption{WithOutputDir(dir)}, tt.opts...)...)
			if err := out.Output(newContainer(t, tt.defs...)); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(dir); !os.IsNotExist(err) {
				t.Errorf("output dir should not exist, stat err = %v", err)
			}
		})
	}
}

func TestMRSOutputErrors(t *testing.T) {
	base := t.TempDir()
	blocker := filepath.Join(base, "file")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	occupied := filepath.Join(base, "occupied")
	if err := os.MkdirAll(filepath.Join(occupied, "cn.mrs"), 0755); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		defs []entryDef
		opts []lib.OutputOption
	}{
		{name: "entry without prefix", defs: []entryDef{{"empty", nil}}},
		{name: "ipv6 only on ipv4-only entry", defs: []entryDef{{"v4", []string{"1.1.1.0/24"}}}, opts: []lib.OutputOption{WithOutputOnlyIPType(lib.IPv6)}},
		{name: "output dir is a file", defs: []entryDef{cnDef}, opts: []lib.OutputOption{WithOutputDir(blocker)}},
		{name: "output file is a directory", defs: []entryDef{cnDef}, opts: []lib.OutputOption{WithOutputDir(occupied)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _ := NewMRSOut(lib.ActionOutput, append([]lib.OutputOption{WithOutputDir(t.TempDir())}, tt.opts...)...)
			if err := out.Output(newContainer(t, tt.defs...)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

// ---------- input ----------

func TestParseMRS(t *testing.T) {
	header := []any{magic, byte(1), int64(1), int64(0), byte(1)}
	with := func(extra ...any) []any { return append(append([]any{}, header...), extra...) }

	tests := []struct {
		name    string
		data    func(t *testing.T) []byte
		wantErr bool
		want    []string
	}{
		{
			name: "valid with extra bytes skipped",
			data: func(t *testing.T) []byte {
				return rawMRS(t, magic, byte(1), int64(1), int64(3), []byte{9, 9, 9}, byte(1), int64(1),
					a16("192.168.0.0"), a16("192.168.255.255"))
			},
			want: []string{"192.168.0.0/16"},
		},
		{
			name: "ipv4-mapped addresses are unmapped",
			data: func(t *testing.T) []byte {
				return rawMRS(t, with(int64(2), a16("::ffff:10.0.0.0"), a16("::ffff:10.255.255.255"),
					a16("fe80::"), a16("fe80::ffff"))...)
			},
			want: []string{"10.0.0.0/8", "fe80::/112"},
		},
		{name: "not zstd", data: func(t *testing.T) []byte { return []byte("plain text, not zstd") }, wantErr: true},
		{name: "empty payload", data: func(t *testing.T) []byte { return rawMRS(t) }, wantErr: true},
		{name: "wrong magic", data: func(t *testing.T) []byte { return rawMRS(t, []byte{'M', 'R', 'S', 2}) }, wantErr: true},
		{name: "missing behavior", data: func(t *testing.T) []byte { return rawMRS(t, magic) }, wantErr: true},
		{name: "domain behavior", data: func(t *testing.T) []byte { return rawMRS(t, magic, byte(0)) }, wantErr: true},
		{name: "missing count", data: func(t *testing.T) []byte { return rawMRS(t, magic, byte(1)) }, wantErr: true},
		{name: "missing extra length", data: func(t *testing.T) []byte { return rawMRS(t, magic, byte(1), int64(1)) }, wantErr: true},
		{name: "negative extra length", data: func(t *testing.T) []byte { return rawMRS(t, magic, byte(1), int64(1), int64(-1)) }, wantErr: true},
		{name: "truncated extra", data: func(t *testing.T) []byte { return rawMRS(t, magic, byte(1), int64(1), int64(10), []byte{1}) }, wantErr: true},
		{name: "missing version", data: func(t *testing.T) []byte { return rawMRS(t, magic, byte(1), int64(1), int64(0)) }, wantErr: true},
		{name: "wrong version", data: func(t *testing.T) []byte { return rawMRS(t, magic, byte(1), int64(1), int64(0), byte(2)) }, wantErr: true},
		{name: "missing rule length", data: func(t *testing.T) []byte { return rawMRS(t, header...) }, wantErr: true},
		{name: "zero rule length", data: func(t *testing.T) []byte { return rawMRS(t, with(int64(0))...) }, wantErr: true},
		{name: "missing from address", data: func(t *testing.T) []byte { return rawMRS(t, with(int64(1))...) }, wantErr: true},
		{name: "truncated from address", data: func(t *testing.T) []byte { return rawMRS(t, with(int64(1), []byte{1, 2, 3})...) }, wantErr: true},
		{name: "missing to address", data: func(t *testing.T) []byte { return rawMRS(t, with(int64(1), a16("1.0.0.0"))...) }, wantErr: true},
		{name: "second range missing", data: func(t *testing.T) []byte {
			return rawMRS(t, with(int64(2), a16("1.0.0.0"), a16("1.0.0.255"))...)
		}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := lib.NewEntry("x")
			err := (&mrs_in{}).parseMRS(tt.data(t), e)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got, err := e.MarshalText()
			if err != nil {
				t.Fatal(err)
			}
			compareStrings(t, "prefixes", got, tt.want)
		})
	}
}

func TestMRSInputDirNaming(t *testing.T) {
	data := func(t *testing.T, from, to string) []byte {
		return rawMRS(t, magic, byte(1), int64(1), int64(0), byte(1), int64(1), a16(from), a16(to))
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a-b_c.d.mrs"), data(t, "10.0.0.0", "10.255.255.255"))
	writeFile(t, filepath.Join(dir, "sub", "noext"), data(t, "fd00::", "fdff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"))
	writeFile(t, filepath.Join(dir, ".hidden"), data(t, "192.168.0.0", "192.168.255.255"))

	in, _ := NewMRSIn(lib.ActionAdd, WithInputDir(dir))
	c, err := in.Input(lib.NewContainer())
	if err != nil {
		t.Fatal(err)
	}
	assertContainer(t, c, map[string][]string{
		"A-B_C.D": {"10.0.0.0/8"},
		"NOEXT":   {"fd00::/8"},
		".HIDDEN": {"192.168.0.0/16"},
	})
}

func TestMRSInputErrors(t *testing.T) {
	base := t.TempDir()
	valid := rawMRS(t, magic, byte(1), int64(1), int64(0), byte(1), int64(1), a16("1.0.0.0"), a16("1.0.0.255"))
	good := writeFile(t, filepath.Join(base, "good", "cn.mrs"), valid)
	garbage := writeFile(t, filepath.Join(base, "garbage", "bad.mrs"), []byte("not an mrs file"))
	badName := filepath.Join(base, "badname")
	writeFile(t, filepath.Join(badName, "c n.mrs"), valid)
	dup := filepath.Join(base, "dup")
	writeFile(t, filepath.Join(dup, "cn.mrs"), valid)
	writeFile(t, filepath.Join(dup, "sub", "CN.mrs"), valid)

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
		{name: "missing local file", action: lib.ActionAdd, opts: []lib.InputOption{WithNameAndURI("cn", filepath.Join(base, "nope.mrs"))}},
		{name: "local path is directory", action: lib.ActionAdd, opts: []lib.InputOption{WithNameAndURI("cn", base)}},
		{name: "http 404", action: lib.ActionAdd, opts: []lib.InputOption{WithNameAndURI("cn", notFound.URL+"/cn.mrs")}},
		{name: "http connection refused", action: lib.ActionAdd, opts: []lib.InputOption{WithNameAndURI("cn", closedURL+"/cn.mrs")}},
		{name: "garbage via http", action: lib.ActionAdd, opts: []lib.InputOption{WithNameAndURI("cn", serveFile(t, garbage))}},
		{name: "wanted list filters everything", action: lib.ActionAdd, opts: []lib.InputOption{WithNameAndURI("cn", good), WithInputWantedList([]string{"us"})}},
		{name: "remove from container without entry", action: lib.ActionRemove, opts: []lib.InputOption{WithNameAndURI("cn", good)}},
		{name: "unknown action", action: lib.Action("unknown"), opts: []lib.InputOption{WithNameAndURI("cn", good)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in, err := NewMRSIn(tt.action, tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := in.Input(lib.NewContainer()); err == nil {
				t.Fatal("expected error")
			}
		})
	}

	t.Run("struct without source", func(t *testing.T) {
		m := &mrs_in{Type: TypeMRSIn, Action: lib.ActionAdd}
		if _, err := m.Input(lib.NewContainer()); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestMRSInputActions(t *testing.T) {
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
			name:     "remove only ipv6 prefixes",
			action:   lib.ActionRemove,
			cidrs:    []string{"1.0.1.0/24", "2001:db8:8000::/33"},
			onlyType: lib.IPv6,
			want:     map[string][]string{"CN": {"1.0.0.0/23", "2001:db8::/33"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			out, _ := NewMRSOut(lib.ActionOutput, WithOutputDir(dir))
			if err := out.Output(newContainer(t, entryDef{"cn", tt.cidrs})); err != nil {
				t.Fatal(err)
			}
			in, _ := NewMRSIn(tt.action, WithNameAndURI("cn", filepath.Join(dir, "cn.mrs")), WithInputOnlyIPType(tt.onlyType))
			c, err := in.Input(newContainer(t, cnDef))
			if err != nil {
				t.Fatal(err)
			}
			assertContainer(t, c, tt.want)
		})
	}
}
