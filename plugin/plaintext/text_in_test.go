package plaintext

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

// newRemoteServer serves a few fixed files for remote-URI tests.
func newRemoteServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/cn.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("1.0.0.0/24\n1.0.1.0/24\n2001:db8::/32\n::ffff:8.8.8.8\n"))
	})
	mux.HandleFunc("/surge.list", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("IP-CIDR,9.9.9.0/24,no-resolve\nIP-CIDR6,2606:4700::/32\nDOMAIN,example.com\n"))
	})
	mux.HandleFunc("/bad.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("bad-ip\n"))
	})
	mux.HandleFunc("/missing.txt", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestTextInGetters(t *testing.T) {
	tests := []struct {
		name     string
		newFn    func(lib.Action, ...lib.InputOption) (lib.InputConverter, error)
		fromFn   func(lib.Action, []byte) (lib.InputConverter, error)
		data     string
		wantType string
		wantDesc string
	}{
		{"text", NewTextIn, NewTextInFromBytes, `{"name":"cn","uri":"a"}`, TypeTextIn, DescTextIn},
		{"json", NewJSONIn, NewJSONInFromBytes, `{"name":"cn","uri":"a","jsonPath":["x"]}`, TypeJSONIn, DescJSONIn},
		{"clash classical", NewClashRuleSetClassicalIn, NewClashRuleSetClassicalInFromBytes, `{"name":"cn","uri":"a"}`, TypeClashRuleSetClassicalIn, DescClashRuleSetClassicalIn},
		{"clash ipcidr", NewClashRuleSetIPCIDRIn, NewClashRuleSetIPCIDRInFromBytes, `{"name":"cn","uri":"a"}`, TypeClashRuleSetIPCIDRIn, DescClashRuleSetIPCIDRIn},
		{"surge", NewSurgeRuleSetIn, NewSurgeRuleSetInFromBytes, `{"name":"cn","uri":"a"}`, TypeSurgeRuleSetIn, DescSurgeRuleSetIn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := []lib.InputOption{WithNameAndURI("cn", "a")}
			if tt.wantType == TypeJSONIn {
				opts = append(opts, WithJSONPath([]string{"x"}))
			}
			c1, err := tt.newFn(lib.ActionRemove, opts...)
			if err != nil {
				t.Fatal(err)
			}
			c2, err := tt.fromFn(lib.ActionRemove, []byte(tt.data))
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range []lib.InputConverter{c1, c2} {
				if c.GetType() != tt.wantType {
					t.Errorf("GetType = %q, want %q", c.GetType(), tt.wantType)
				}
				if c.GetAction() != lib.ActionRemove {
					t.Errorf("GetAction = %q", c.GetAction())
				}
				if c.GetDescription() != tt.wantDesc {
					t.Errorf("GetDescription = %q, want %q", c.GetDescription(), tt.wantDesc)
				}
			}
			if _, err := tt.fromFn(lib.ActionAdd, []byte("{")); err == nil {
				t.Error("expected error for invalid JSON bytes")
			}
		})
	}
}

func TestTextInRegisteredConfigCreators(t *testing.T) {
	// The init() functions register config creators; exercise them through lib.
	tests := []struct {
		name     string
		iType    string
		args     string
		wantType string
	}{
		{"text", TypeTextIn, `{"name":"cn","ipOrCIDR":["1.1.1.1"]}`, TypeTextIn},
		{"json", TypeJSONIn, `{"name":"cn","uri":"a.json","jsonPath":["a"]}`, TypeJSONIn},
		{"clash classical", TypeClashRuleSetClassicalIn, `{"name":"cn","uri":"a.yaml"}`, TypeClashRuleSetClassicalIn},
		{"clash ipcidr", TypeClashRuleSetIPCIDRIn, `{"name":"cn","uri":"a.yaml"}`, TypeClashRuleSetIPCIDRIn},
		{"surge", TypeSurgeRuleSetIn, `{"name":"cn","uri":"a.list"}`, TypeSurgeRuleSetIn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst, err := lib.NewInstance()
			if err != nil {
				t.Fatal(err)
			}
			cfg := `{"input":[{"type":"` + tt.iType + `","action":"add","args":` + tt.args + `}],` +
				`"output":[{"type":"text","action":"output","args":{"outputDir":"` + filepath.ToSlash(t.TempDir()) + `"}}]}`
			if err := inst.InitConfigFromBytes([]byte(cfg)); err != nil {
				t.Fatalf("InitConfigFromBytes: %v", err)
			}
		})
	}
}

func TestTextInInput(t *testing.T) {
	srv := newRemoteServer(t)

	type build func(t *testing.T, dir string) (lib.InputConverter, error)

	tests := []struct {
		name    string
		files   map[string]string   // relative path -> content, created in temp dir
		pre     map[string][]string // initial container content
		build   build
		want    map[string][]string
		wantErr string
	}{
		{
			name:  "local file with name (ipv4, ipv6, ipv4-mapped)",
			files: map[string]string{"cn.txt": "1.0.0.0/24\n1.0.1.0/24\n2001:db8::/32\n::ffff:8.8.8.8\n::ffff:10.0.0.0/104\n"},
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd, WithNameAndURI("cn", filepath.Join(dir, "cn.txt")))
			},
			want: map[string][]string{"CN": {"1.0.0.0/23", "8.8.8.8/32", "10.0.0.0/8", "2001:db8::/32"}},
		},
		{
			name:  "local file plus ipOrCIDR are merged",
			files: map[string]string{"cn.txt": "1.0.0.0/24\n"},
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd,
					WithNameAndURI("cn", filepath.Join(dir, "cn.txt")),
					WithIPOrCIDR([]string{" 1.0.1.0/24 ", "2001:db8::1", "::ffff:2.2.2.2"}))
			},
			want: map[string][]string{"CN": {"1.0.0.0/23", "2.2.2.2/32", "2001:db8::1/128"}},
		},
		{
			name: "ipOrCIDR only",
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd,
					WithNameAndURI("test", ""),
					WithIPOrCIDR([]string{"192.168.0.0/16", "2001:db8::/32", "::ffff:172.16.0.0/108", "fe80::1"}))
			},
			want: map[string][]string{"TEST": {"172.16.0.0/12", "192.168.0.0/16", "2001:db8::/32", "fe80::1/128"}},
		},
		{
			name: "ipOrCIDR with onlyIPType ipv4",
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd,
					WithNameAndURI("test", ""),
					WithInputOnlyIPType(lib.IPv4),
					WithIPOrCIDR([]string{"1.1.1.1", "2001:db8::/32", "::ffff:2.2.2.2"}))
			},
			want: map[string][]string{"TEST": {"1.1.1.1/32", "2.2.2.2/32"}},
		},
		{
			name: "ipOrCIDR with onlyIPType ipv6",
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd,
					WithNameAndURI("test", ""),
					WithInputOnlyIPType(lib.IPv6),
					WithIPOrCIDR([]string{"1.1.1.1", "2001:db8::/32", "::ffff:2.2.2.2"}))
			},
			want: map[string][]string{"TEST": {"2001:db8::/32"}},
		},
		{
			name: "ipOrCIDR invalid",
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd, WithNameAndURI("test", ""), WithIPOrCIDR([]string{"1.1.1.1", "nope"}))
			},
			wantErr: "invalid IP address",
		},
		{
			name: "inputDir walks recursively, strips extension, keeps hidden file name",
			files: map[string]string{
				"cn.txt":       "1.0.0.0/24\n",
				"us.list":      "2001:db8::/32\n",
				"sub/jp.txt":   "::ffff:3.3.3.3\n",
				".hidden":      "4.4.4.4\n",
				"a.b.c.txt":    "5.5.5.5\n",
				"no_ext-file1": "6.6.6.0/24\n",
			},
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd, WithInputDir(dir))
			},
			want: map[string][]string{
				"CN":           {"1.0.0.0/24"},
				"US":           {"2001:db8::/32"},
				"JP":           {"3.3.3.3/32"},
				".HIDDEN":      {"4.4.4.4/32"},
				"A.B.C":        {"5.5.5.5/32"},
				"NO_EXT-FILE1": {"6.6.6.0/24"},
			},
		},
		{
			name: "inputDir with wanted list",
			files: map[string]string{
				"cn.txt":  "1.0.0.0/24\n",
				"us.txt":  "2.0.0.0/24\n",
				"jp.list": "2001:db8::/32\n",
			},
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd, WithInputDir(dir), WithInputWantedList([]string{"cn", " JP "}))
			},
			want: map[string][]string{"CN": {"1.0.0.0/24"}, "JP": {"2001:db8::/32"}},
		},
		{
			name:  "inputDir surge format",
			files: map[string]string{"cf.list": "IP-CIDR,1.1.1.0/24\nIP-CIDR6,2606:4700::/32,no-resolve\nIP-CIDR,::ffff:1.0.0.0/120\n"},
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewSurgeRuleSetIn(lib.ActionAdd, WithInputDir(dir))
			},
			want: map[string][]string{"CF": {"1.0.0.0/24", "1.1.1.0/24", "2606:4700::/32"}},
		},
		{
			name:  "inputDir clash ipcidr format",
			files: map[string]string{"cf.yaml": "payload:\n  - '1.1.1.0/24'\n  - '2606:4700::/32'\n  - '::ffff:1.0.0.1'\n"},
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewClashRuleSetIPCIDRIn(lib.ActionAdd, WithInputDir(dir))
			},
			want: map[string][]string{"CF": {"1.0.0.1/32", "1.1.1.0/24", "2606:4700::/32"}},
		},
		{
			name:  "local clash classical format",
			files: map[string]string{"cf.yaml": "payload:\n  - IP-CIDR,1.1.1.0/24\n  - IP-CIDR6,2606:4700::/32\n  - DOMAIN-SUFFIX,example.com\n"},
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewClashRuleSetClassicalIn(lib.ActionAdd, WithNameAndURI("cloudflare", filepath.Join(dir, "cf.yaml")))
			},
			want: map[string][]string{"CLOUDFLARE": {"1.1.1.0/24", "2606:4700::/32"}},
		},
		{
			name:  "local json format",
			files: map[string]string{"data.json": `{"v4":["1.1.1.0/24"],"v6":["2606:4700::/32","::ffff:1.0.0.1"]}`},
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewJSONIn(lib.ActionAdd, WithNameAndURI("cf", filepath.Join(dir, "data.json")), WithJSONPath([]string{"v4", "v6"}))
			},
			want: map[string][]string{"CF": {"1.0.0.1/32", "1.1.1.0/24", "2606:4700::/32"}},
		},
		{
			name: "remote text file",
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd, WithNameAndURI("cn", srv.URL+"/cn.txt"))
			},
			want: map[string][]string{"CN": {"1.0.0.0/23", "8.8.8.8/32", "2001:db8::/32"}},
		},
		{
			name: "remote uri with upper-case scheme, plus ipOrCIDR",
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				u := "HTTP://" + strings.TrimPrefix(srv.URL, "http://") + "/cn.txt"
				return NewTextIn(lib.ActionAdd, WithNameAndURI("cn", u), WithIPOrCIDR([]string{"2001:db9::/32"}))
			},
			want: map[string][]string{"CN": {"1.0.0.0/23", "8.8.8.8/32", "2001:db8::/31"}},
		},
		{
			name: "remote surge file",
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewSurgeRuleSetIn(lib.ActionAdd, WithNameAndURI("cf", srv.URL+"/surge.list"))
			},
			want: map[string][]string{"CF": {"9.9.9.0/24", "2606:4700::/32"}},
		},
		{
			name: "remote file with matching wanted list",
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd, WithNameAndURI("cn", srv.URL+"/cn.txt"), WithInputWantedList([]string{"CN"}))
			},
			want: map[string][]string{"CN": {"1.0.0.0/23", "8.8.8.8/32", "2001:db8::/32"}},
		},
		{
			name: "remote file filtered out by wanted list",
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd, WithNameAndURI("cn", srv.URL+"/cn.txt"), WithInputWantedList([]string{"us"}))
			},
			wantErr: "no entry is generated",
		},
		{
			name: "remote file non-200 status",
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd, WithNameAndURI("cn", srv.URL+"/missing.txt"))
			},
			wantErr: "http status code 404",
		},
		{
			name: "remote file with invalid content",
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd, WithNameAndURI("cn", srv.URL+"/bad.txt"))
			},
			wantErr: "invalid IP address",
		},
		{
			name: "remote connection error",
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				closed := httptest.NewServer(http.NotFoundHandler())
				u := closed.URL
				closed.Close()
				return NewTextIn(lib.ActionAdd, WithNameAndURI("cn", u+"/cn.txt"))
			},
			wantErr: "connect",
		},
		{
			name:  "local file filtered out by wanted list",
			files: map[string]string{"cn.txt": "1.0.0.0/24\n"},
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd, WithNameAndURI("cn", filepath.Join(dir, "cn.txt")), WithInputWantedList([]string{"us"}))
			},
			wantErr: "no entry is generated",
		},
		{
			name: "local file does not exist",
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd, WithNameAndURI("cn", filepath.Join(dir, "nope.txt")))
			},
			wantErr: "no such file",
		},
		{
			name:  "local file with invalid content",
			files: map[string]string{"cn.txt": "1.0.0.0/24\nxyz\n"},
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd, WithNameAndURI("cn", filepath.Join(dir, "cn.txt")))
			},
			wantErr: "invalid IP address",
		},
		{
			name: "inputDir does not exist",
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd, WithInputDir(filepath.Join(dir, "nope")))
			},
			wantErr: "no such file",
		},
		{
			name:  "inputDir with invalid filename",
			files: map[string]string{"bad name.txt": "1.1.1.1\n"},
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd, WithInputDir(dir))
			},
			wantErr: "cannot be entry name",
		},
		{
			name:  "inputDir with duplicated list names",
			files: map[string]string{"cn.list": "1.1.1.1\n", "cn.txt": "2.2.2.2\n"},
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd, WithInputDir(dir))
			},
			wantErr: "found duplicated list CN",
		},
		{
			name:  "empty directory generates no entry",
			files: map[string]string{"emptydir/": ""},
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd, WithInputDir(filepath.Join(dir, "emptydir")))
			},
			wantErr: "no entry is generated",
		},
		{
			name: "missing arguments (struct built directly)",
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return &text_in{Type: TypeTextIn, Action: lib.ActionAdd}, nil
			},
			wantErr: "config missing argument",
		},
		{
			name: "name with neither uri nor ipOrCIDR (struct built directly)",
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return &text_in{Type: TypeTextIn, Action: lib.ActionAdd, Name: "cn"}, nil
			},
			wantErr: "config missing argument",
		},
		{
			name: "remove action",
			pre: map[string][]string{
				"CN": {"1.0.0.0/23", "2001:db8::/32", "8.8.8.8"},
			},
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionRemove, WithNameAndURI("cn", ""),
					WithIPOrCIDR([]string{"1.0.1.0/24", "2001:db8:8000::/33", "::ffff:8.8.8.8"}))
			},
			want: map[string][]string{"CN": {"1.0.0.0/24", "2001:db8::/33"}},
		},
		{
			name: "remove action with onlyIPType ipv6",
			pre: map[string][]string{
				"CN": {"1.0.0.0/23", "2001:db8::/32"},
			},
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionRemove, WithNameAndURI("cn", ""), WithInputOnlyIPType(lib.IPv6),
					WithIPOrCIDR([]string{"1.0.1.0/24", "2001:db8:8000::/33"}))
			},
			want: map[string][]string{"CN": {"1.0.0.0/23", "2001:db8::/33"}},
		},
		{
			name: "malformed ipv4-mapped ipv6 literal",
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd, WithNameAndURI("cn", ""), WithIPOrCIDR([]string{"::ffff:1.2.3"}))
			},
			wantErr: "invalid IP address",
		},
		{
			name: "add action merges into existing entry",
			pre:  map[string][]string{"CN": {"1.0.0.0/24"}, "US": {"2.0.0.0/24"}},
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionAdd, WithNameAndURI("cn", ""), WithIPOrCIDR([]string{"1.0.1.0/24", "2001:db8::/32"}))
			},
			want: map[string][]string{"CN": {"1.0.0.0/23", "2001:db8::/32"}, "US": {"2.0.0.0/24"}},
		},
		{
			name: "remove action when entry is absent in container",
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionRemove, WithNameAndURI("cn", ""), WithIPOrCIDR([]string{"1.1.1.1"}))
			},
			wantErr: "entry CN not found",
		},
		{
			name: "unknown action",
			build: func(t *testing.T, dir string) (lib.InputConverter, error) {
				return NewTextIn(lib.ActionOutput, WithNameAndURI("cn", ""), WithIPOrCIDR([]string{"1.1.1.1"}))
			},
			wantErr: lib.ErrUnknownAction.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for rel, content := range tt.files {
				if strings.HasSuffix(rel, "/") {
					if err := os.MkdirAll(filepath.Join(dir, rel), 0755); err != nil {
						t.Fatal(err)
					}
					continue
				}
				writeTestFile(t, filepath.Join(dir, rel), content)
			}

			conv, err := tt.build(t, dir)
			if err != nil {
				t.Fatalf("build converter: %v", err)
			}

			container := newContainerWith(t, tt.pre)
			got, err := conv.Input(container)
			if checkErr(t, err, tt.wantErr) {
				if got != nil {
					t.Errorf("container should be nil on error")
				}
				return
			}
			compareLists(t, containerText(got), tt.want)
		})
	}
}

func TestTextInWalkLocalFile(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "cn.txt"), "1.0.0.0/24\n::ffff:2.2.2.2\n2001:db8::/32\n")
	writeTestFile(t, filepath.Join(dir, "bad#name.txt"), "1.1.1.1\n")

	tests := []struct {
		name     string
		path     string
		listName string
		want     map[string]bool
		existing []string
		wantList string
		wantText []string
		wantErr  string
	}{
		{name: "name from filename", path: "cn.txt", wantList: "CN", wantText: []string{"1.0.0.0/24", "2.2.2.2/32", "2001:db8::/32"}},
		{name: "explicit name with spaces", path: "cn.txt", listName: "  china ", wantList: "CHINA", wantText: []string{"1.0.0.0/24", "2.2.2.2/32", "2001:db8::/32"}},
		{name: "wanted list hit", path: "cn.txt", want: map[string]bool{"CN": true}, wantList: "CN", wantText: []string{"1.0.0.0/24", "2.2.2.2/32", "2001:db8::/32"}},
		{name: "wanted list miss", path: "cn.txt", want: map[string]bool{"US": true}},
		{name: "duplicated", path: "cn.txt", existing: []string{"CN"}, wantErr: "found duplicated list CN"},
		{name: "invalid filename", path: "bad#name.txt", wantErr: "please remove special characters"},
		{name: "open error with explicit name", path: "nope.txt", listName: "x", wantErr: "no such file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := &text_in{Type: TypeTextIn, Want: tt.want}
			entries := make(map[string]*lib.Entry)
			for _, name := range tt.existing {
				entries[name] = lib.NewEntry(name)
			}
			err := in.walkLocalFile(filepath.Join(dir, tt.path), tt.listName, entries)
			if checkErr(t, err, tt.wantErr) {
				return
			}
			if tt.wantList == "" {
				if len(entries) != 0 {
					t.Fatalf("entries = %v, want none", entries)
				}
				return
			}
			if len(entries) != 1 {
				t.Fatalf("entries count = %d, want 1", len(entries))
			}
			e, ok := entries[tt.wantList]
			if !ok {
				t.Fatalf("entry %s missing", tt.wantList)
			}
			compareStrings(t, tt.wantList, entryText(e), tt.wantText)
		})
	}
}

func TestTextInAppendIPOrCIDR(t *testing.T) {
	tests := []struct {
		name     string
		ipOrCIDR []string
		existing []string
		want     []string
		wantNone bool
		wantErr  string
	}{
		{name: "empty input does nothing", ipOrCIDR: nil, wantNone: true},
		{name: "new entry", ipOrCIDR: []string{"1.1.1.1", " 2001:db8::/32 ", "::ffff:3.3.3.0/120"}, want: []string{"1.1.1.1/32", "3.3.3.0/24", "2001:db8::/32"}},
		{name: "append to existing entry", existing: []string{"1.0.0.0/24"}, ipOrCIDR: []string{"1.0.1.0/24"}, want: []string{"1.0.0.0/23"}},
		{name: "comment-only item is ignored", ipOrCIDR: []string{"# comment", "1.1.1.1"}, want: []string{"1.1.1.1/32"}},
		{name: "invalid", ipOrCIDR: []string{"1.1.1.1/40"}, wantErr: "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := &text_in{Type: TypeTextIn}
			entries := make(map[string]*lib.Entry)
			if tt.existing != nil {
				entries["CN"] = newEntryWith(t, "CN", tt.existing...)
			}
			err := in.appendIPOrCIDR(tt.ipOrCIDR, "cn", entries)
			if checkErr(t, err, tt.wantErr) {
				return
			}
			if tt.wantNone {
				if len(entries) != 0 {
					t.Fatalf("entries = %v", entries)
				}
				return
			}
			compareStrings(t, "CN", entryText(entries["CN"]), tt.want)
		})
	}
}

func TestTextInWalkRemoteFileError(t *testing.T) {
	in := &text_in{Type: TypeTextIn}
	err := in.walkRemoteFile("http://[::1]:namedport/x", "cn", map[string]*lib.Entry{})
	if err == nil {
		t.Fatal("expected error for malformed URL")
	}
}
