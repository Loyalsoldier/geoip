package singbox

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/sagernet/sing-box/common/srs"
	"github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
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

// buildSRS builds valid SRS data from a list of CIDRs.
func buildSRS(t *testing.T, cidrs ...string) []byte {
	t.Helper()
	var rs option.PlainRuleSet
	rs.Rules = []option.HeadlessRule{{
		Type:           constant.RuleTypeDefault,
		DefaultOptions: option.DefaultHeadlessRule{IPCIDR: cidrs},
	}}
	var buf bytes.Buffer
	if err := srs.Write(&buf, rs, constant.RuleSetVersion1); err != nil {
		t.Fatalf("srs.Write: %v", err)
	}
	return buf.Bytes()
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

func TestNewSRSIn(t *testing.T) {
	tests := []struct {
		name    string
		opts    []lib.InputOption
		wantErr string
		check   func(t *testing.T, m *srs_in)
	}{
		{name: "no options", wantErr: "missing name or uri or inputDir"},
		{name: "nil option ignored", opts: []lib.InputOption{nil}, wantErr: "missing name or uri or inputDir"},
		{name: "name only", opts: []lib.InputOption{WithNameAndURI("cn", "")}, wantErr: "name and uri must be specified together"},
		{name: "uri only", opts: []lib.InputOption{WithNameAndURI("", "a.srs")}, wantErr: "name and uri must be specified together"},
		{
			name: "name and uri",
			opts: []lib.InputOption{WithNameAndURI(" cn ", " a.srs ")},
			check: func(t *testing.T, m *srs_in) {
				if m.Name != "cn" || m.URI != "a.srs" {
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
			check: func(t *testing.T, m *srs_in) {
				if m.InputDir != "dir" {
					t.Errorf("InputDir = %q", m.InputDir)
				}
				if len(m.Want) != 2 || !m.Want["CN"] || !m.Want["US"] {
					t.Errorf("Want = %v", m.Want)
				}
				if m.OnlyIPType != lib.IPv4 {
					t.Errorf("OnlyIPType = %v", m.OnlyIPType)
				}
				if m.GetType() != TypeSRSIn || m.GetAction() != lib.ActionAdd || m.GetDescription() != DescSRSIn {
					t.Errorf("unexpected type/action/description: %s %s %s", m.GetType(), m.GetAction(), m.GetDescription())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewSRSIn(lib.ActionAdd, tt.opts...)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.check(t, conv.(*srs_in))
		})
	}
}

func TestNewSRSInFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		check   func(t *testing.T, m *srs_in)
	}{
		{name: "empty data", data: "", wantErr: true},
		{name: "invalid json", data: "{", wantErr: true},
		{
			name: "valid name and uri",
			data: `{"name":"cn","uri":"cn.srs","wantedList":["cn"],"onlyIPType":"ipv6"}`,
			check: func(t *testing.T, m *srs_in) {
				if m.Name != "cn" || m.URI != "cn.srs" || !m.Want["CN"] || m.OnlyIPType != lib.IPv6 {
					t.Errorf("unexpected: %+v", m)
				}
			},
		},
		{
			name: "valid input dir",
			data: `{"inputDir":"dir"}`,
			check: func(t *testing.T, m *srs_in) {
				if m.InputDir != "dir" {
					t.Errorf("InputDir = %q", m.InputDir)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewSRSInFromBytes(lib.ActionAdd, []byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.check(t, conv.(*srs_in))
		})
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read error") }

func TestSRSIn_generateEntries(t *testing.T) {
	valid := buildSRS(t, "1.0.0.0/24")

	t.Run("not wanted is skipped", func(t *testing.T) {
		m := &srs_in{Want: map[string]bool{"US": true}}
		entries := map[string]*lib.Entry{}
		if err := m.generateEntries("cn", bytes.NewReader(valid), entries); err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Fatalf("expected no entries, got %d", len(entries))
		}
	})

	t.Run("read error", func(t *testing.T) {
		m := &srs_in{}
		if err := m.generateEntries("cn", errReader{}, map[string]*lib.Entry{}); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("merges into existing entry", func(t *testing.T) {
		m := &srs_in{Want: map[string]bool{"CN": true}}
		entries := map[string]*lib.Entry{}
		if err := m.generateEntries("cn", bytes.NewReader(valid), entries); err != nil {
			t.Fatal(err)
		}
		if err := m.generateEntries("CN", bytes.NewReader(buildSRS(t, "2001:db8::/32")), entries); err != nil {
			t.Fatal(err)
		}
		got, _ := entries["CN"].MarshalText()
		assertCIDRs(t, got, []string{"1.0.0.0/24", "2001:db8::/32"})
	})
}

func TestSRSIn_Input(t *testing.T) {
	cnData := buildSRS(t, "1.0.0.0/24", "2001:db8::/32")
	usData := buildSRS(t, "8.8.8.0/24")

	tmp := t.TempDir()

	// valid input directory
	validDir := filepath.Join(tmp, "valid")
	writeFile(t, filepath.Join(validDir, "cn.srs"), cnData)
	writeFile(t, filepath.Join(validDir, "sub", "us.srs"), usData)

	// directory containing invalid filename
	badNameDir := filepath.Join(tmp, "badname")
	writeFile(t, filepath.Join(badNameDir, "c n.srs"), cnData)

	// directory containing duplicated lists
	dupDir := filepath.Join(tmp, "dup")
	writeFile(t, filepath.Join(dupDir, "cn.srs"), cnData)
	writeFile(t, filepath.Join(dupDir, "sub", "cn.srs"), cnData)

	// directory containing invalid data
	badDataDir := filepath.Join(tmp, "baddata")
	writeFile(t, filepath.Join(badDataDir, "cn.srs"), []byte("invalid"))

	// hidden file keeps full name
	hiddenDir := filepath.Join(tmp, "hidden")
	writeFile(t, filepath.Join(hiddenDir, ".hidden"), cnData)

	cnFile := filepath.Join(validDir, "cn.srs")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cn.srs":
			w.Write(cnData)
		case "/invalid.srs":
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
		input     *srs_in
		container func(t *testing.T) lib.Container
		wantErr   string
		want      map[string][]string
	}{
		{
			name:  "input dir",
			input: &srs_in{Action: lib.ActionAdd, InputDir: validDir},
			want: map[string][]string{
				"CN": {"1.0.0.0/24", "2001:db8::/32"},
				"US": {"8.8.8.0/24"},
			},
		},
		{
			name:  "input dir with wanted list",
			input: &srs_in{Action: lib.ActionAdd, InputDir: validDir, Want: map[string]bool{"US": true}},
			want:  map[string][]string{"US": {"8.8.8.0/24"}},
		},
		{
			name:  "input dir only ipv6",
			input: &srs_in{Action: lib.ActionAdd, InputDir: validDir, Want: map[string]bool{"CN": true}, OnlyIPType: lib.IPv6},
			want:  map[string][]string{"CN": {"2001:db8::/32"}},
		},
		{
			name:  "hidden file",
			input: &srs_in{Action: lib.ActionAdd, InputDir: hiddenDir},
			want:  map[string][]string{".HIDDEN": {"1.0.0.0/24", "2001:db8::/32"}},
		},
		{name: "input dir not exist", input: &srs_in{Action: lib.ActionAdd, InputDir: filepath.Join(tmp, "nope")}, wantErr: "no such file"},
		{name: "invalid filename", input: &srs_in{Action: lib.ActionAdd, InputDir: badNameDir}, wantErr: "cannot be entry name"},
		{name: "duplicated list", input: &srs_in{Action: lib.ActionAdd, InputDir: dupDir}, wantErr: "found duplicated list"},
		{name: "invalid data", input: &srs_in{Action: lib.ActionAdd, InputDir: badDataDir}, wantErr: "invalid"},
		{
			name:  "local file",
			input: &srs_in{Action: lib.ActionAdd, Name: "test", URI: cnFile},
			want:  map[string][]string{"TEST": {"1.0.0.0/24", "2001:db8::/32"}},
		},
		{
			name:  "local file only ipv4",
			input: &srs_in{Action: lib.ActionAdd, Name: "test", URI: cnFile, OnlyIPType: lib.IPv4},
			want:  map[string][]string{"TEST": {"1.0.0.0/24"}},
		},
		{name: "local file not exist", input: &srs_in{Action: lib.ActionAdd, Name: "test", URI: filepath.Join(tmp, "nope.srs")}, wantErr: "no such file"},
		{
			name:  "remote file",
			input: &srs_in{Action: lib.ActionAdd, Name: "remote", URI: server.URL + "/cn.srs"},
			want:  map[string][]string{"REMOTE": {"1.0.0.0/24", "2001:db8::/32"}},
		},
		{
			name:  "remote file uppercase scheme",
			input: &srs_in{Action: lib.ActionAdd, Name: "remote", URI: strings.Replace(server.URL, "http://", "HTTP://", 1) + "/cn.srs"},
			want:  map[string][]string{"REMOTE": {"1.0.0.0/24", "2001:db8::/32"}},
		},
		{name: "remote invalid data", input: &srs_in{Action: lib.ActionAdd, Name: "remote", URI: server.URL + "/invalid.srs"}, wantErr: "invalid"},
		{name: "remote file 404", input: &srs_in{Action: lib.ActionAdd, Name: "remote", URI: server.URL + "/missing"}, wantErr: "http status code 404"},
		{name: "remote connection error", input: &srs_in{Action: lib.ActionAdd, Name: "remote", URI: closedURL + "/cn.srs"}, wantErr: "connect"},
		{name: "remote https connection error", input: &srs_in{Action: lib.ActionAdd, Name: "remote", URI: strings.Replace(closedURL, "http://", "https://", 1)}, wantErr: "connect"},
		{name: "missing arguments", input: &srs_in{Action: lib.ActionAdd}, wantErr: "config missing argument"},
		{
			name:    "no entry generated",
			input:   &srs_in{Action: lib.ActionAdd, Name: "test", URI: cnFile, Want: map[string]bool{"OTHER": true}},
			wantErr: "no entry is generated",
		},
		{
			name:  "remove prefix",
			input: &srs_in{Action: lib.ActionRemove, Name: "cn", URI: filepath.Join(validDir, "sub", "us.srs")},
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
		{name: "remove from missing entry", input: &srs_in{Action: lib.ActionRemove, Name: "cn", URI: cnFile}, wantErr: "not found"},
		{name: "unknown action", input: &srs_in{Action: lib.Action("unknown"), Name: "cn", URI: cnFile}, wantErr: lib.ErrUnknownAction.Error()},
		{
			name:  "container add error",
			input: &srs_in{Action: lib.ActionAdd, Name: "cn", URI: cnFile},
			container: func(t *testing.T) lib.Container {
				return &mockContainer{Container: lib.NewContainer(), addErr: errors.New("add failed")}
			},
			wantErr: "add failed",
		},
		{
			name:  "container remove error",
			input: &srs_in{Action: lib.ActionRemove, Name: "cn", URI: cnFile},
			container: func(t *testing.T) lib.Container {
				return &mockContainer{Container: lib.NewContainer(), removeErr: errors.New("remove failed")}
			},
			wantErr: "remove failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.input.Type = TypeSRSIn
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
