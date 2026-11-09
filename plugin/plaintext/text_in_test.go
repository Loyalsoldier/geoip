package plaintext

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestTextInInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok.txt":
			w.Write([]byte("1.1.1.0/24\n2001:db8::/32\n"))
		case "/bad.txt":
			w.Write([]byte("invalid\n"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	localFile := writeTestFile(t, dir, "local.txt", "1.1.1.0/24\n")
	badFile := writeTestFile(t, dir, "bad.txt", "invalid\n")

	inputDir := filepath.Join(dir, "input")
	writeTestFile(t, inputDir, "cn.txt", "1.0.1.0/24\n")
	writeTestFile(t, inputDir, "sub/us.list", "2.0.0.0/24\n2001:db8::/32\n")
	writeTestFile(t, inputDir, ".hidden", "3.0.0.0/24\n")

	dupDir := filepath.Join(dir, "dup")
	writeTestFile(t, dupDir, "cn.txt", "1.0.1.0/24\n")
	writeTestFile(t, dupDir, "cn.list", "1.0.2.0/24\n")

	specialDir := filepath.Join(dir, "special")
	writeTestFile(t, specialDir, "c n.txt", "1.0.1.0/24\n")

	badDir := filepath.Join(dir, "baddir")
	writeTestFile(t, badDir, "cn.txt", "invalid\n")

	type wantEntry struct {
		name  string
		cidrs []string
	}

	tests := []struct {
		name      string
		in        *text_in
		container func() lib.Container
		want      []wantEntry
		wantErr   bool
	}{
		{
			name: "input dir",
			in:   &text_in{Type: TypeTextIn, Action: lib.ActionAdd, InputDir: inputDir},
			want: []wantEntry{
				{".HIDDEN", []string{"3.0.0.0/24"}},
				{"CN", []string{"1.0.1.0/24"}},
				{"US", []string{"2.0.0.0/24", "2001:db8::/32"}},
			},
		},
		{
			name: "input dir with wanted list and only ipv6",
			in:   &text_in{Type: TypeTextIn, Action: lib.ActionAdd, InputDir: inputDir, Want: map[string]bool{"US": true}, OnlyIPType: lib.IPv6},
			want: []wantEntry{{"US", []string{"2001:db8::/32"}}},
		},
		{name: "input dir not exist", in: &text_in{Type: TypeTextIn, Action: lib.ActionAdd, InputDir: filepath.Join(dir, "none")}, wantErr: true},
		{name: "input dir duplicated list", in: &text_in{Type: TypeTextIn, Action: lib.ActionAdd, InputDir: dupDir}, wantErr: true},
		{name: "input dir special filename", in: &text_in{Type: TypeTextIn, Action: lib.ActionAdd, InputDir: specialDir}, wantErr: true},
		{name: "input dir invalid content", in: &text_in{Type: TypeTextIn, Action: lib.ActionAdd, InputDir: badDir}, wantErr: true},
		{
			name: "local file with ipOrCIDR",
			in:   &text_in{Type: TypeTextIn, Action: lib.ActionAdd, Name: "test", URI: localFile, IPOrCIDR: []string{" 2.2.2.2 "}},
			want: []wantEntry{{"TEST", []string{"1.1.1.0/24", "2.2.2.2/32"}}},
		},
		{
			name: "only ipOrCIDR",
			in:   &text_in{Type: TypeTextIn, Action: lib.ActionAdd, Name: "test", IPOrCIDR: []string{"2.2.2.2", "2001:db8::1"}},
			want: []wantEntry{{"TEST", []string{"2.2.2.2/32", "2001:db8::1/128"}}},
		},
		{name: "invalid ipOrCIDR", in: &text_in{Type: TypeTextIn, Action: lib.ActionAdd, Name: "test", IPOrCIDR: []string{"bad"}}, wantErr: true},
		{name: "local file not exist", in: &text_in{Type: TypeTextIn, Action: lib.ActionAdd, Name: "test", URI: filepath.Join(dir, "none.txt")}, wantErr: true},
		{name: "local file invalid content", in: &text_in{Type: TypeTextIn, Action: lib.ActionAdd, Name: "test", URI: badFile}, wantErr: true},
		{name: "local file not wanted", in: &text_in{Type: TypeTextIn, Action: lib.ActionAdd, Name: "test", URI: localFile, Want: map[string]bool{"OTHER": true}}, wantErr: true},
		{
			name: "remote file",
			in:   &text_in{Type: TypeTextIn, Action: lib.ActionAdd, Name: "remote", URI: server.URL + "/ok.txt", OnlyIPType: lib.IPv4},
			want: []wantEntry{{"REMOTE", []string{"1.1.1.0/24"}}},
		},
		{name: "remote file not wanted", in: &text_in{Type: TypeTextIn, Action: lib.ActionAdd, Name: "remote", URI: server.URL + "/ok.txt", Want: map[string]bool{"OTHER": true}}, wantErr: true},
		{name: "remote file 404", in: &text_in{Type: TypeTextIn, Action: lib.ActionAdd, Name: "remote", URI: server.URL + "/404"}, wantErr: true},
		{name: "remote file invalid content", in: &text_in{Type: TypeTextIn, Action: lib.ActionAdd, Name: "remote", URI: server.URL + "/bad.txt"}, wantErr: true},
		{name: "remote file request error", in: &text_in{Type: TypeTextIn, Action: lib.ActionAdd, Name: "remote", URI: "http://127.0.0.1:0/x"}, wantErr: true},
		{name: "missing arguments", in: &text_in{Type: TypeTextIn, Action: lib.ActionAdd}, wantErr: true},
		{
			name: "remove action",
			in:   &text_in{Type: TypeTextIn, Action: lib.ActionRemove, Name: "test", IPOrCIDR: []string{"1.1.1.0/25"}},
			container: func() lib.Container {
				c := lib.NewContainer()
				e := lib.NewEntry("test")
				e.AddPrefix("1.1.1.0/24")
				c.Add(e)
				return c
			},
			want: []wantEntry{{"TEST", []string{"1.1.1.128/25"}}},
		},
		{name: "remove action entry not found", in: &text_in{Type: TypeTextIn, Action: lib.ActionRemove, Name: "test", IPOrCIDR: []string{"1.1.1.0/25"}}, wantErr: true},
		{name: "unknown action", in: &text_in{Type: TypeTextIn, Action: lib.ActionOutput, Name: "test", IPOrCIDR: []string{"1.1.1.1"}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := lib.NewContainer()
			if tt.container != nil {
				container = tt.container()
			}
			got, err := tt.in.Input(container)
			assertErr(t, err, tt.wantErr)
			if tt.wantErr {
				return
			}
			if got.Len() != len(tt.want) {
				t.Fatalf("container length = %d, want %d", got.Len(), len(tt.want))
			}
			for _, w := range tt.want {
				entry, found := got.GetEntry(w.name)
				if !found {
					t.Fatalf("entry %s not found", w.name)
				}
				assertCIDRs(t, entryCIDRs(t, entry), w.cidrs)
			}
		})
	}
}

func TestTextInWalkLocalFileUnwantedFromDir(t *testing.T) {
	dir := t.TempDir()
	p := writeTestFile(t, dir, "cn.txt", "1.1.1.1\n")
	in := &text_in{Type: TypeTextIn, Want: map[string]bool{"US": true}}
	entries := make(map[string]*lib.Entry)
	if err := in.walkLocalFile(p, "", entries); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no entries, got %d", len(entries))
	}
}

func TestTextInAppendIPOrCIDR(t *testing.T) {
	in := &text_in{}
	entries := make(map[string]*lib.Entry)
	if err := in.appendIPOrCIDR(nil, "test", entries); err != nil || len(entries) != 0 {
		t.Fatalf("empty input should be a no-op: err=%v len=%d", err, len(entries))
	}
	if err := in.appendIPOrCIDR([]string{"1.1.1.1"}, "test", entries); err != nil {
		t.Fatal(err)
	}
	if err := in.appendIPOrCIDR([]string{"1.1.1.2"}, "TEST", entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries length = %d, want 1", len(entries))
	}
	assertCIDRs(t, entryCIDRs(t, entries["TEST"]), []string{"1.1.1.1/32", "1.1.1.2/32"})
}

func TestTextInWalkDirNames(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "a.txt", "1.1.1.1\n")
	writeTestFile(t, dir, "b.txt", "2.2.2.2\n")
	in := &text_in{Type: TypeTextIn}
	entries := make(map[string]*lib.Entry)
	if err := in.walkDir(dir, entries); err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for n := range entries {
		names = append(names, n)
	}
	sort.Strings(names)
	assertCIDRs(t, names, []string{"A", "B"})
}
