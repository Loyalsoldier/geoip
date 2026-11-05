package mihomo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/klauspost/compress/zstd"
	"go4.org/netipx"
)

// mockContainer wraps a real container and allows forcing errors on Add/Remove.
type mockContainer struct {
	lib.Container
	addErr    error
	removeErr error
}

func (m *mockContainer) Add(entry *lib.Entry, opts ...lib.IgnoreIPOption) error {
	if m.addErr != nil {
		return m.addErr
	}
	return m.Container.Add(entry, opts...)
}

func (m *mockContainer) Remove(entry *lib.Entry, rCase lib.CaseRemove, opts ...lib.IgnoreIPOption) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	return m.Container.Remove(entry, rCase, opts...)
}

// zstdBytes compresses raw payload with zstd.
func zstdBytes(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc, err := zstd.NewWriter(&buf)
	if err != nil {
		t.Fatalf("zstd.NewWriter: %v", err)
	}
	if _, err := enc.Write(payload); err != nil {
		t.Fatalf("zstd write: %v", err)
	}
	if err := enc.Close(); err != nil {
		t.Fatalf("zstd close: %v", err)
	}
	return buf.Bytes()
}

// buildMRS builds valid MRS data from a list of CIDRs.
func buildMRS(t *testing.T, cidrs ...string) []byte {
	t.Helper()
	var b netipx.IPSetBuilder
	for _, c := range cidrs {
		b.AddPrefix(netip.MustParsePrefix(c))
	}
	set, err := b.IPSet()
	if err != nil {
		t.Fatalf("IPSet: %v", err)
	}
	var buf bytes.Buffer
	if err := (&mrs_out{}).convertToMrs(set.Ranges(), &buf); err != nil {
		t.Fatalf("convertToMrs: %v", err)
	}
	return buf.Bytes()
}

type rawMRS struct {
	magic      []byte
	behavior   []byte
	count      []byte
	extraLen   []byte
	extra      []byte
	version    []byte
	ruleLength []byte
	rules      []byte
}

func i64(v int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (r rawMRS) bytes() []byte {
	var out []byte
	for _, p := range [][]byte{r.magic, r.behavior, r.count, r.extraLen, r.extra, r.version, r.ruleLength, r.rules} {
		out = append(out, p...)
	}
	return out
}

func validRaw() rawMRS {
	from := netip.MustParseAddr("1.0.0.0").As16()
	to := netip.MustParseAddr("1.0.0.255").As16()
	return rawMRS{
		magic:      mrsMagicBytes[:],
		behavior:   []byte{1},
		count:      i64(1),
		extraLen:   i64(0),
		version:    []byte{1},
		ruleLength: i64(1),
		rules:      append(from[:], to[:]...),
	}
}

func entryCIDRs(t *testing.T, c lib.Container, name string) []string {
	t.Helper()
	entry, found := c.GetEntry(name)
	if !found {
		t.Fatalf("entry %s not found", name)
	}
	cidrs, err := entry.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	return cidrs
}

func assertCIDRs(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("CIDR count mismatch: got %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("CIDR content mismatch: got %v, want %v", got, want)
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestNewMRSIn(t *testing.T) {
	tests := []struct {
		name    string
		opts    []lib.InputOption
		wantErr string
		check   func(t *testing.T, m *mrs_in)
	}{
		{name: "no options", wantErr: "missing name or uri or inputDir"},
		{name: "nil option ignored", opts: []lib.InputOption{nil}, wantErr: "missing name or uri or inputDir"},
		{name: "name only", opts: []lib.InputOption{WithNameAndURI("cn", "")}, wantErr: "name and uri must be specified together"},
		{name: "uri only", opts: []lib.InputOption{WithNameAndURI("", "a.mrs")}, wantErr: "name and uri must be specified together"},
		{
			name: "name and uri",
			opts: []lib.InputOption{WithNameAndURI(" cn ", " a.mrs ")},
			check: func(t *testing.T, m *mrs_in) {
				if m.Name != "cn" || m.URI != "a.mrs" {
					t.Errorf("got name %q uri %q", m.Name, m.URI)
				}
			},
		},
		{
			name: "input dir with wanted list and ip type",
			opts: []lib.InputOption{
				WithInputDir(" dir "),
				WithInputWantedList([]string{" cn ", "", "us"}),
				WithInputOnlyIPType(lib.IPv4),
			},
			check: func(t *testing.T, m *mrs_in) {
				if m.InputDir != "dir" {
					t.Errorf("InputDir = %q", m.InputDir)
				}
				if len(m.Want) != 2 || !m.Want["CN"] || !m.Want["US"] {
					t.Errorf("Want = %v", m.Want)
				}
				if m.OnlyIPType != lib.IPv4 {
					t.Errorf("OnlyIPType = %v", m.OnlyIPType)
				}
				if m.GetType() != TypeMRSIn || m.GetAction() != lib.ActionAdd || m.GetDescription() != DescMRSIn {
					t.Errorf("unexpected type/action/description: %s %s %s", m.GetType(), m.GetAction(), m.GetDescription())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewMRSIn(lib.ActionAdd, tt.opts...)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.check(t, conv.(*mrs_in))
		})
	}
}

func TestNewMRSInFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		check   func(t *testing.T, m *mrs_in)
	}{
		{name: "empty data", data: "", wantErr: true},
		{name: "invalid json", data: "{", wantErr: true},
		{
			name: "valid name and uri",
			data: `{"name":"cn","uri":"cn.mrs","wantedList":["cn"],"onlyIPType":"ipv6"}`,
			check: func(t *testing.T, m *mrs_in) {
				if m.Name != "cn" || m.URI != "cn.mrs" || !m.Want["CN"] || m.OnlyIPType != lib.IPv6 {
					t.Errorf("unexpected: %+v", m)
				}
			},
		},
		{
			name: "valid input dir",
			data: `{"inputDir":"dir"}`,
			check: func(t *testing.T, m *mrs_in) {
				if m.InputDir != "dir" {
					t.Errorf("InputDir = %q", m.InputDir)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewMRSInFromBytes(lib.ActionAdd, []byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.check(t, conv.(*mrs_in))
		})
	}
}

func TestMRSIn_parseMRS(t *testing.T) {
	tests := []struct {
		name    string
		data    func(t *testing.T) []byte
		wantErr string
		want    []string
	}{
		{
			name: "valid ipv4 and ipv6",
			data: func(t *testing.T) []byte {
				return buildMRS(t, "1.0.0.0/24", "1.0.1.0/24", "2001:db8::/32", "10.0.0.0/8")
			},
			want: []string{"1.0.0.0/23", "10.0.0.0/8", "2001:db8::/32"},
		},
		{
			name: "range not aligned to prefix",
			data: func(t *testing.T) []byte {
				r := validRaw()
				from := netip.MustParseAddr("1.0.0.1").As16()
				to := netip.MustParseAddr("1.0.0.6").As16()
				r.rules = append(from[:], to[:]...)
				return zstdBytes(t, r.bytes())
			},
			want: []string{"1.0.0.1/32", "1.0.0.2/31", "1.0.0.4/31", "1.0.0.6/32"},
		},
		{
			name: "extra data skipped",
			data: func(t *testing.T) []byte {
				r := validRaw()
				r.extraLen = i64(3)
				r.extra = []byte{9, 9, 9}
				return zstdBytes(t, r.bytes())
			},
			want: []string{"1.0.0.0/24"},
		},
		{name: "not zstd", data: func(t *testing.T) []byte { return []byte("not zstd data") }, wantErr: "magic"},
		{name: "empty", data: func(t *testing.T) []byte { return zstdBytes(t, nil) }, wantErr: "EOF"},
		{
			name:    "invalid magic",
			data:    func(t *testing.T) []byte { r := validRaw(); r.magic = []byte("ABCD"); return zstdBytes(t, r.bytes()) },
			wantErr: "invalid MRS format",
		},
		{
			name:    "missing behavior",
			data:    func(t *testing.T) []byte { return zstdBytes(t, mrsMagicBytes[:]) },
			wantErr: "EOF",
		},
		{
			name:    "invalid behavior",
			data:    func(t *testing.T) []byte { r := validRaw(); r.behavior = []byte{0}; return zstdBytes(t, r.bytes()) },
			wantErr: "invalid MRS IPCIDR data",
		},
		{
			name: "missing count",
			data: func(t *testing.T) []byte {
				r := validRaw()
				return zstdBytes(t, append(r.magic, r.behavior...))
			},
			wantErr: "EOF",
		},
		{
			name: "missing extra length",
			data: func(t *testing.T) []byte {
				r := rawMRS{magic: mrsMagicBytes[:], behavior: []byte{1}, count: i64(1)}
				return zstdBytes(t, r.bytes())
			},
			wantErr: "EOF",
		},
		{
			name:    "negative extra length",
			data:    func(t *testing.T) []byte { r := validRaw(); r.extraLen = i64(-1); return zstdBytes(t, r.bytes()) },
			wantErr: "invalid MRS extra length",
		},
		{
			name: "truncated extra",
			data: func(t *testing.T) []byte {
				r := rawMRS{magic: mrsMagicBytes[:], behavior: []byte{1}, count: i64(1), extraLen: i64(10), extra: []byte{1}}
				return zstdBytes(t, r.bytes())
			},
			wantErr: "EOF",
		},
		{
			name: "missing version",
			data: func(t *testing.T) []byte {
				r := rawMRS{magic: mrsMagicBytes[:], behavior: []byte{1}, count: i64(1), extraLen: i64(0)}
				return zstdBytes(t, r.bytes())
			},
			wantErr: "EOF",
		},
		{
			name:    "invalid version",
			data:    func(t *testing.T) []byte { r := validRaw(); r.version = []byte{2}; return zstdBytes(t, r.bytes()) },
			wantErr: "invalid MRS rule version",
		},
		{
			name: "missing rule length",
			data: func(t *testing.T) []byte {
				r := validRaw()
				r.ruleLength, r.rules = nil, nil
				return zstdBytes(t, r.bytes())
			},
			wantErr: "EOF",
		},
		{
			name:    "zero rule length",
			data:    func(t *testing.T) []byte { r := validRaw(); r.ruleLength = i64(0); return zstdBytes(t, r.bytes()) },
			wantErr: "invalid MRS rule length",
		},
		{
			name:    "missing from address",
			data:    func(t *testing.T) []byte { r := validRaw(); r.rules = nil; return zstdBytes(t, r.bytes()) },
			wantErr: "EOF",
		},
		{
			name:    "missing to address",
			data:    func(t *testing.T) []byte { r := validRaw(); r.rules = r.rules[:16]; return zstdBytes(t, r.bytes()) },
			wantErr: "EOF",
		},
		{
			name: "invalid range yields no prefix",
			data: func(t *testing.T) []byte {
				r := validRaw()
				from := netip.MustParseAddr("1.0.0.255").As16()
				to := netip.MustParseAddr("1.0.0.0").As16()
				r.rules = append(from[:], to[:]...)
				return zstdBytes(t, r.bytes())
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mrs_in{}
			entry := lib.NewEntry("TEST")
			err := m.parseMRS(tt.data(t), entry)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got, _ := entry.MarshalText()
			assertCIDRs(t, got, tt.want)
		})
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read error") }

func TestMRSIn_generateEntries(t *testing.T) {
	valid := buildMRS(t, "1.0.0.0/24")

	t.Run("not wanted is skipped", func(t *testing.T) {
		m := &mrs_in{Want: map[string]bool{"US": true}}
		entries := map[string]*lib.Entry{}
		if err := m.generateEntries("cn", bytes.NewReader(valid), entries); err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Fatalf("expected no entries, got %d", len(entries))
		}
	})

	t.Run("read error", func(t *testing.T) {
		m := &mrs_in{}
		if err := m.generateEntries("cn", errReader{}, map[string]*lib.Entry{}); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("merges into existing entry", func(t *testing.T) {
		m := &mrs_in{Want: map[string]bool{"CN": true}}
		entries := map[string]*lib.Entry{}
		if err := m.generateEntries("cn", bytes.NewReader(valid), entries); err != nil {
			t.Fatal(err)
		}
		if err := m.generateEntries("CN", bytes.NewReader(buildMRS(t, "2001:db8::/32")), entries); err != nil {
			t.Fatal(err)
		}
		got, _ := entries["CN"].MarshalText()
		assertCIDRs(t, got, []string{"1.0.0.0/24", "2001:db8::/32"})
	})
}

func TestMRSIn_Input(t *testing.T) {
	cnData := buildMRS(t, "1.0.0.0/24", "2001:db8::/32")
	usData := buildMRS(t, "8.8.8.0/24")

	tmp := t.TempDir()

	// valid input directory
	validDir := filepath.Join(tmp, "valid")
	writeFile(t, filepath.Join(validDir, "cn.mrs"), cnData)
	writeFile(t, filepath.Join(validDir, "sub", "us.mrs"), usData)

	// directory containing invalid filename
	badNameDir := filepath.Join(tmp, "badname")
	writeFile(t, filepath.Join(badNameDir, "c n.mrs"), cnData)

	// directory containing duplicated lists
	dupDir := filepath.Join(tmp, "dup")
	writeFile(t, filepath.Join(dupDir, "cn.mrs"), cnData)
	writeFile(t, filepath.Join(dupDir, "sub", "cn.mrs"), cnData)

	// directory containing invalid data
	badDataDir := filepath.Join(tmp, "baddata")
	writeFile(t, filepath.Join(badDataDir, "cn.mrs"), []byte("invalid"))

	// hidden file keeps full name
	hiddenDir := filepath.Join(tmp, "hidden")
	writeFile(t, filepath.Join(hiddenDir, ".hidden"), cnData)

	cnFile := filepath.Join(validDir, "cn.mrs")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cn.mrs":
			w.Write(cnData)
		case "/invalid.mrs":
			w.Write([]byte("invalid"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	closedServer := httptest.NewServer(http.NotFoundHandler())
	closedURL := closedServer.URL
	closedServer.Close()

	tests := []struct {
		name      string
		input     *mrs_in
		container func(t *testing.T) lib.Container
		wantErr   string
		want      map[string][]string
	}{
		{
			name:  "input dir",
			input: &mrs_in{Action: lib.ActionAdd, InputDir: validDir},
			want: map[string][]string{
				"CN": {"1.0.0.0/24", "2001:db8::/32"},
				"US": {"8.8.8.0/24"},
			},
		},
		{
			name:  "input dir with wanted list",
			input: &mrs_in{Action: lib.ActionAdd, InputDir: validDir, Want: map[string]bool{"US": true}},
			want:  map[string][]string{"US": {"8.8.8.0/24"}},
		},
		{
			name:  "input dir only ipv6",
			input: &mrs_in{Action: lib.ActionAdd, InputDir: validDir, Want: map[string]bool{"CN": true}, OnlyIPType: lib.IPv6},
			want:  map[string][]string{"CN": {"2001:db8::/32"}},
		},
		{
			name:  "hidden file",
			input: &mrs_in{Action: lib.ActionAdd, InputDir: hiddenDir},
			want:  map[string][]string{".HIDDEN": {"1.0.0.0/24", "2001:db8::/32"}},
		},
		{name: "input dir not exist", input: &mrs_in{Action: lib.ActionAdd, InputDir: filepath.Join(tmp, "nope")}, wantErr: "no such file"},
		{name: "invalid filename", input: &mrs_in{Action: lib.ActionAdd, InputDir: badNameDir}, wantErr: "cannot be entry name"},
		{name: "duplicated list", input: &mrs_in{Action: lib.ActionAdd, InputDir: dupDir}, wantErr: "found duplicated list"},
		{name: "invalid data", input: &mrs_in{Action: lib.ActionAdd, InputDir: badDataDir}, wantErr: "magic"},
		{
			name:  "local file",
			input: &mrs_in{Action: lib.ActionAdd, Name: "test", URI: cnFile},
			want:  map[string][]string{"TEST": {"1.0.0.0/24", "2001:db8::/32"}},
		},
		{
			name:  "local file only ipv4",
			input: &mrs_in{Action: lib.ActionAdd, Name: "test", URI: cnFile, OnlyIPType: lib.IPv4},
			want:  map[string][]string{"TEST": {"1.0.0.0/24"}},
		},
		{name: "local file not exist", input: &mrs_in{Action: lib.ActionAdd, Name: "test", URI: filepath.Join(tmp, "nope.mrs")}, wantErr: "no such file"},
		{
			name:  "remote file",
			input: &mrs_in{Action: lib.ActionAdd, Name: "remote", URI: server.URL + "/cn.mrs"},
			want:  map[string][]string{"REMOTE": {"1.0.0.0/24", "2001:db8::/32"}},
		},
		{
			name:  "remote file uppercase scheme",
			input: &mrs_in{Action: lib.ActionAdd, Name: "remote", URI: strings.Replace(server.URL, "http://", "HTTP://", 1) + "/cn.mrs"},
			want:  map[string][]string{"REMOTE": {"1.0.0.0/24", "2001:db8::/32"}},
		},
		{name: "remote invalid data", input: &mrs_in{Action: lib.ActionAdd, Name: "remote", URI: server.URL + "/invalid.mrs"}, wantErr: "magic"},
		{name: "remote file 404", input: &mrs_in{Action: lib.ActionAdd, Name: "remote", URI: server.URL + "/missing"}, wantErr: "http status code 404"},
		{name: "remote connection error", input: &mrs_in{Action: lib.ActionAdd, Name: "remote", URI: closedURL + "/cn.mrs"}, wantErr: "connect"},
		{name: "remote https connection error", input: &mrs_in{Action: lib.ActionAdd, Name: "remote", URI: strings.Replace(closedURL, "http://", "https://", 1)}, wantErr: "connect"},
		{name: "missing arguments", input: &mrs_in{Action: lib.ActionAdd}, wantErr: "config missing argument"},
		{
			name:    "no entry generated",
			input:   &mrs_in{Action: lib.ActionAdd, Name: "test", URI: cnFile, Want: map[string]bool{"OTHER": true}},
			wantErr: "no entry is generated",
		},
		{
			name:  "remove prefix",
			input: &mrs_in{Action: lib.ActionRemove, Name: "cn", URI: filepath.Join(validDir, "sub", "us.mrs")},
			container: func(t *testing.T) lib.Container {
				c := lib.NewContainer()
				e := lib.NewEntry("CN")
				for _, p := range []string{"8.8.0.0/16", "2001:db8::/32"} {
					if err := e.AddPrefix(p); err != nil {
						t.Fatal(err)
					}
				}
				if err := c.Add(e); err != nil {
					t.Fatal(err)
				}
				return c
			},
			want: map[string][]string{"CN": {
				"8.8.0.0/21", "8.8.9.0/24", "8.8.10.0/23", "8.8.12.0/22", "8.8.16.0/20",
				"8.8.32.0/19", "8.8.64.0/18", "8.8.128.0/17", "2001:db8::/32",
			}},
		},
		{name: "remove from missing entry", input: &mrs_in{Action: lib.ActionRemove, Name: "cn", URI: cnFile}, wantErr: "not found"},
		{name: "unknown action", input: &mrs_in{Action: lib.Action("unknown"), Name: "cn", URI: cnFile}, wantErr: lib.ErrUnknownAction.Error()},
		{
			name:  "container add error",
			input: &mrs_in{Action: lib.ActionAdd, Name: "cn", URI: cnFile},
			container: func(t *testing.T) lib.Container {
				return &mockContainer{Container: lib.NewContainer(), addErr: errors.New("add failed")}
			},
			wantErr: "add failed",
		},
		{
			name:  "container remove error",
			input: &mrs_in{Action: lib.ActionRemove, Name: "cn", URI: cnFile},
			container: func(t *testing.T) lib.Container {
				return &mockContainer{Container: lib.NewContainer(), removeErr: errors.New("remove failed")}
			},
			wantErr: "remove failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.input.Type = TypeMRSIn
			var c lib.Container
			if tt.container != nil {
				c = tt.container(t)
			} else {
				c = lib.NewContainer()
			}
			got, err := tt.input.Input(c)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Len() != len(tt.want) {
				t.Fatalf("entry count mismatch: got %d, want %d", got.Len(), len(tt.want))
			}
			for name, want := range tt.want {
				assertCIDRs(t, entryCIDRs(t, got, name), want)
			}
		})
	}
}
