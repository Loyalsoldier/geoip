package special

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

// newContainer builds a container from a map of entry name -> CIDR list.
func newContainer(t *testing.T, data map[string][]string) lib.Container {
	t.Helper()
	c := lib.NewContainer()
	for name, cidrs := range data {
		e := lib.NewEntry(name)
		for _, cidr := range cidrs {
			if err := e.AddPrefix(cidr); err != nil {
				t.Fatalf("AddPrefix(%q): %v", cidr, err)
			}
		}
		if err := c.Add(e); err != nil {
			t.Fatalf("Add(%q): %v", name, err)
		}
	}
	return c
}

// dump returns entry name -> plaintext CIDR list for every entry in the container.
func dump(t *testing.T, c lib.Container) map[string][]string {
	t.Helper()
	out := make(map[string][]string)
	for e := range c.Loop() {
		list, err := e.MarshalText()
		if err != nil {
			list = nil
		}
		out[e.GetName()] = list
	}
	return out
}

func assertContainer(t *testing.T, c lib.Container, want map[string][]string) {
	t.Helper()
	got := dump(t, c)
	if len(got) != len(want) {
		t.Fatalf("entry count: got %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for name, wantList := range want {
		gotList, ok := got[name]
		if !ok {
			t.Fatalf("entry %q missing; got %v", name, got)
		}
		if len(gotList) != len(wantList) {
			t.Errorf("entry %q count: got %d %v, want %d %v", name, len(gotList), gotList, len(wantList), wantList)
			continue
		}
		if !slices.Equal(gotList, wantList) {
			t.Errorf("entry %q: got %v, want %v", name, gotList, wantList)
		}
	}
}

// captureStdout runs fn while redirecting os.Stdout and returns what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	defer func() { os.Stdout = orig }()
	fn()
	w.Close()
	os.Stdout = orig
	return <-done
}

// withStdin replaces os.Stdin with a file containing content for the duration of the test.
func withStdin(t *testing.T, content string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "stdin")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdin
	os.Stdin = f
	t.Cleanup(func() {
		os.Stdin = orig
		f.Close()
	})
}

func TestRegistration(t *testing.T) {
	tests := []struct {
		name    string
		section string
		typ     string
		action  lib.Action
		args    string
	}{
		{"cutter", "input", TypeCutter, lib.ActionRemove, `{"wantedList":["cn"]}`},
		{"private", "input", TypePrivate, lib.ActionAdd, `{}`},
		{"stdin", "input", TypeStdin, lib.ActionAdd, `{"name":"x"}`},
		{"test", "input", typeTest, lib.ActionAdd, `{}`},
		{"lookup", "output", TypeLookup, lib.ActionOutput, `{"search":"1.1.1.1"}`},
		{"stdout", "output", TypeStdout, lib.ActionOutput, `{}`},
		{"cutter bad args", "input", TypeCutter, lib.ActionAdd, `{"wantedList":["cn"]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := fmt.Sprintf(`{%q:[{"type":%q,"action":%q,"args":%s}]}`, tt.section, tt.typ, tt.action, tt.args)
			inst, err := lib.NewInstance()
			if err != nil {
				t.Fatal(err)
			}
			err = inst.InitConfigFromBytes([]byte(conf))
			if tt.name == "cutter bad args" {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("InitConfigFromBytes(%s): %v", conf, err)
			}
		})
	}
}

func TestGetters(t *testing.T) {
	cut, _ := NewCutter(lib.ActionRemove, WithCutterWantedList([]string{"a"}))
	priv, _ := NewPrivate(lib.ActionAdd)
	in, _ := NewStdin(lib.ActionRemove, WithStdinName("n"))
	tst, _ := newTest(lib.ActionAdd, nil)
	look, _ := NewLookup(lib.ActionOutput, WithLookupSearch("1.1.1.1"))
	out, _ := NewStdout(lib.ActionOutput, nil)

	tests := []struct {
		name              string
		typ, desc         string
		action            lib.Action
		wantTyp, wantDesc string
		wantAction        lib.Action
	}{
		{"cutter", cut.GetType(), cut.GetDescription(), cut.GetAction(), TypeCutter, DescCutter, lib.ActionRemove},
		{"private", priv.GetType(), priv.GetDescription(), priv.GetAction(), TypePrivate, DescPrivate, lib.ActionAdd},
		{"stdin", in.GetType(), in.GetDescription(), in.GetAction(), TypeStdin, DescStdin, lib.ActionRemove},
		{"test", tst.GetType(), tst.GetDescription(), tst.GetAction(), typeTest, descTest, lib.ActionAdd},
		{"lookup", look.GetType(), look.GetDescription(), look.GetAction(), TypeLookup, DescLookup, lib.ActionOutput},
		{"stdout", out.GetType(), out.GetDescription(), out.GetAction(), TypeStdout, DescStdout, lib.ActionOutput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.typ != tt.wantTyp || tt.desc != tt.wantDesc || tt.action != tt.wantAction {
				t.Errorf("got (%q,%q,%q), want (%q,%q,%q)", tt.typ, tt.desc, tt.action, tt.wantTyp, tt.wantDesc, tt.wantAction)
			}
		})
	}
}

func TestNewCutterFromBytes(t *testing.T) {
	tests := []struct {
		name     string
		action   lib.Action
		data     string
		wantErr  string
		wantWant map[string]bool
		wantType lib.IPType
	}{
		{"valid", lib.ActionRemove, `{"wantedList":[" cn ","", "us"],"onlyIPType":"ipv4"}`, "", map[string]bool{"CN": true, "US": true}, lib.IPv4},
		{"add action rejected", lib.ActionAdd, `{"wantedList":["cn"]}`, "only supports `remove` action", nil, ""},
		{"empty wanted list", lib.ActionRemove, `{"wantedList":[" ",""]}`, "wantedList must be specified", nil, ""},
		{"no data", lib.ActionRemove, ``, "wantedList must be specified", nil, ""},
		{"invalid json", lib.ActionRemove, `{`, "unexpected end of JSON input", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewCutterFromBytes(tt.action, []byte(tt.data))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			c := conv.(*cutter)
			if len(c.Want) != len(tt.wantWant) {
				t.Fatalf("want: got %v, want %v", c.Want, tt.wantWant)
			}
			for k := range tt.wantWant {
				if !c.Want[k] {
					t.Errorf("missing %q in %v", k, c.Want)
				}
			}
			if c.OnlyIPType != tt.wantType {
				t.Errorf("onlyIPType: got %q want %q", c.OnlyIPType, tt.wantType)
			}
		})
	}
}

func TestCutterInput(t *testing.T) {
	base := map[string][]string{
		"CN": {"1.0.1.0/24", "2001:250::/32"},
		"US": {"8.8.8.0/24", "2001:4860::/32"},
		"JP": {"1.0.16.0/20"},
	}
	tests := []struct {
		name   string
		want   []string
		onlyIP lib.IPType
		expect map[string][]string
	}{
		{
			name: "remove whole entries",
			want: []string{"cn", "us", "notexist"},
			expect: map[string][]string{
				"JP": {"1.0.16.0/20"},
			},
		},
		{
			name:   "remove only ipv4 part",
			want:   []string{"CN"},
			onlyIP: lib.IPv4,
			expect: map[string][]string{
				"CN": {"2001:250::/32"},
				"US": {"8.8.8.0/24", "2001:4860::/32"},
				"JP": {"1.0.16.0/20"},
			},
		},
		{
			name:   "remove only ipv6 part",
			want:   []string{"US"},
			onlyIP: lib.IPv6,
			expect: map[string][]string{
				"CN": {"1.0.1.0/24", "2001:250::/32"},
				"US": {"8.8.8.0/24"},
				"JP": {"1.0.16.0/20"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewCutter(lib.ActionRemove, WithCutterWantedList(tt.want), WithCutterOnlyIPType(tt.onlyIP), nil)
			if err != nil {
				t.Fatal(err)
			}
			c, err := conv.(lib.InputConverter).Input(newContainer(t, base))
			if err != nil {
				t.Fatal(err)
			}
			assertContainer(t, c, tt.expect)
		})
	}
}

func TestNewPrivateFromBytes(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		wantErr  bool
		wantType lib.IPType
	}{
		{"empty", ``, false, ""},
		{"ipv6 only", `{"onlyIPType":"ipv6"}`, false, lib.IPv6},
		{"invalid json", `[`, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewPrivateFromBytes(lib.ActionAdd, []byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && conv.(*private).OnlyIPType != tt.wantType {
				t.Errorf("onlyIPType: got %q want %q", conv.(*private).OnlyIPType, tt.wantType)
			}
		})
	}
}

func TestPrivateInput(t *testing.T) {
	wantV4 := []string{
		"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16",
		"172.16.0.0/12", "192.0.0.0/24", "192.0.2.0/24", "192.88.99.0/24", "192.168.0.0/16",
		"198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "224.0.0.0/3",
	}
	wantV6 := []string{"::/127", "fc00::/7", "fe80::/10", "ff00::/8"}
	all := append(slices.Clone(wantV4), wantV6...)

	tests := []struct {
		name    string
		action  lib.Action
		onlyIP  lib.IPType
		initial map[string][]string
		expect  map[string][]string
		wantErr string
	}{
		{"add all", lib.ActionAdd, "", nil, map[string][]string{"PRIVATE": all}, ""},
		{"add ipv4 only", lib.ActionAdd, lib.IPv4, nil, map[string][]string{"PRIVATE": wantV4}, ""},
		{"add ipv6 only", lib.ActionAdd, lib.IPv6, nil, map[string][]string{"PRIVATE": wantV6}, ""},
		{
			name:    "merge into existing",
			action:  lib.ActionAdd,
			initial: map[string][]string{"private": {"11.0.0.0/8", "2000::/16"}},
			expect: map[string][]string{"PRIVATE": {
				"0.0.0.0/8", "10.0.0.0/7", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16",
				"172.16.0.0/12", "192.0.0.0/24", "192.0.2.0/24", "192.88.99.0/24", "192.168.0.0/16",
				"198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "224.0.0.0/3",
				"::/127", "2000::/16", "fc00::/7", "fe80::/10", "ff00::/8",
			}},
		},
		{
			name:    "remove from existing",
			action:  lib.ActionRemove,
			initial: map[string][]string{"private": {"10.0.0.0/7", "192.168.1.0/24", "2000::/16", "fc00::/8"}},
			expect:  map[string][]string{"PRIVATE": {"11.0.0.0/8", "2000::/16"}},
		},
		{
			name:    "remove ipv4 only",
			action:  lib.ActionRemove,
			onlyIP:  lib.IPv4,
			initial: map[string][]string{"private": {"10.0.0.0/7", "fc00::/8"}},
			expect:  map[string][]string{"PRIVATE": {"11.0.0.0/8", "fc00::/8"}},
		},
		{"remove missing entry", lib.ActionRemove, "", nil, nil, "entry PRIVATE not found"},
		{"unknown action", lib.Action("bogus"), "", nil, nil, lib.ErrUnknownAction.Error()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, _ := NewPrivate(tt.action, WithPrivateOnlyIPType(tt.onlyIP), nil)
			c, err := conv.Input(newContainer(t, tt.initial))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			assertContainer(t, c, tt.expect)
		})
	}
}

func TestTestInput(t *testing.T) {
	tests := []struct {
		name    string
		action  lib.Action
		initial map[string][]string
		expect  map[string][]string
		wantErr string
	}{
		{"add", lib.ActionAdd, nil, map[string][]string{"TEST": {"127.0.0.0/8"}}, ""},
		{"add merge", lib.ActionAdd, map[string][]string{"test": {"126.0.0.0/8"}}, map[string][]string{"TEST": {"126.0.0.0/7"}}, ""},
		{"remove", lib.ActionRemove, map[string][]string{"test": {"126.0.0.0/7", "::1/128"}}, map[string][]string{"TEST": {"126.0.0.0/8", "::1/128"}}, ""},
		{"remove missing", lib.ActionRemove, nil, nil, "not found"},
		{"unknown action", lib.Action("x"), nil, nil, lib.ErrUnknownAction.Error()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := newTestFromBytes(tt.action, nil)
			if err != nil {
				t.Fatal(err)
			}
			c, err := conv.Input(newContainer(t, tt.initial))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			assertContainer(t, c, tt.expect)
		})
	}
}

func TestNewStdinFromBytes(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		wantErr  string
		wantName string
		wantType lib.IPType
	}{
		{"valid", `{"name":"  mylist ","onlyIPType":"ipv4"}`, "", "mylist", lib.IPv4},
		{"missing name", `{}`, "name must be specified", "", ""},
		{"blank name", `{"name":"   "}`, "name must be specified", "", ""},
		{"no data", ``, "name must be specified", "", ""},
		{"invalid json", `{"name":1}`, "cannot unmarshal", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewStdinFromBytes(lib.ActionAdd, []byte(tt.data))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			s := conv.(*stdin)
			if s.Name != tt.wantName || s.OnlyIPType != tt.wantType {
				t.Errorf("got (%q,%q), want (%q,%q)", s.Name, s.OnlyIPType, tt.wantName, tt.wantType)
			}
		})
	}
}

func TestStdinInput(t *testing.T) {
	const input = `
# full-line comment
192.168.0.3/24
192.168.1.5/24   # trailing hash comment
10.0.0.1 // trailing slash comment
2001:db8::1/64 /* block comment */
::ffff:172.16.0.1/108
::ffff:1.2.3.4
not-an-ip
::ffff:1.2.3.4/64
1.2.3.4/33
   // only comment
2001:db8:1::/48
`
	tests := []struct {
		name    string
		action  lib.Action
		onlyIP  lib.IPType
		stdin   string
		initial map[string][]string
		expect  map[string][]string
		wantErr string
	}{
		{
			name:   "add all, merge and skip invalid lines",
			action: lib.ActionAdd,
			stdin:  input,
			expect: map[string][]string{"MYLIST": {
				"1.2.3.4/32", "10.0.0.1/32", "172.16.0.0/12", "192.168.0.0/23",
				"2001:db8::/64", "2001:db8:1::/48",
			}},
		},
		{
			name:   "add ipv4 only",
			action: lib.ActionAdd,
			onlyIP: lib.IPv4,
			stdin:  input,
			expect: map[string][]string{"MYLIST": {"1.2.3.4/32", "10.0.0.1/32", "172.16.0.0/12", "192.168.0.0/23"}},
		},
		{
			name:   "add ipv6 only",
			action: lib.ActionAdd,
			onlyIP: lib.IPv6,
			stdin:  input,
			expect: map[string][]string{"MYLIST": {"2001:db8::/64", "2001:db8:1::/48"}},
		},
		{
			name:    "remove splits existing prefixes",
			action:  lib.ActionRemove,
			stdin:   "192.168.1.0/24\n2001:db8:0:1::/64\n",
			initial: map[string][]string{"mylist": {"192.168.0.0/23", "2001:db8::/63"}},
			expect:  map[string][]string{"MYLIST": {"192.168.0.0/24", "2001:db8::/64"}},
		},
		{
			name:    "remove missing entry",
			action:  lib.ActionRemove,
			stdin:   "1.1.1.1\n",
			wantErr: "entry MYLIST not found",
		},
		{
			name:    "unknown action",
			action:  lib.Action("bogus"),
			stdin:   "1.1.1.1\n",
			wantErr: lib.ErrUnknownAction.Error(),
		},
		{
			name:    "line too long",
			action:  lib.ActionAdd,
			stdin:   strings.Repeat("a", 70*1024) + "\n",
			wantErr: "token too long",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withStdin(t, tt.stdin)
			conv, err := NewStdin(tt.action, WithStdinName("mylist"), WithStdinOnlyIPType(tt.onlyIP), nil)
			if err != nil {
				t.Fatal(err)
			}
			c, err := conv.Input(newContainer(t, tt.initial))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			assertContainer(t, c, tt.expect)
		})
	}
}

func TestNewLookupFromBytes(t *testing.T) {
	tests := []struct {
		name       string
		data       string
		wantErr    string
		wantSearch string
		wantList   []string
	}{
		{"valid", `{"search":" 1.1.1.1 ","searchList":["cn","us"]}`, "", "1.1.1.1", []string{"cn", "us"}},
		{"missing search", `{"searchList":["cn"]}`, "search must be specified", "", nil},
		{"no data", ``, "search must be specified", "", nil},
		{"invalid json", `{"search":[]}`, "cannot unmarshal", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewLookupFromBytes(lib.ActionOutput, []byte(tt.data))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			l := conv.(*lookup)
			if l.Search != tt.wantSearch || !slices.Equal(l.SearchList, tt.wantList) {
				t.Errorf("got (%q,%v), want (%q,%v)", l.Search, l.SearchList, tt.wantSearch, tt.wantList)
			}
		})
	}
}

func TestLookupOutput(t *testing.T) {
	data := map[string][]string{
		"CN":      {"1.0.1.0/24", "2001:250::/32"},
		"PRIVATE": {"192.168.0.0/16", "10.0.0.0/8", "fc00::/7"},
		"LAN":     {"192.168.0.0/23"},
		"US":      {"8.8.8.0/24"},
	}
	tests := []struct {
		name       string
		search     string
		searchList []string
		wantOut    string
		wantErr    string
	}{
		{"ipv4 single match", "1.0.1.1", nil, "cn\n", ""},
		{"ipv4 multiple matches sorted lowercase", "192.168.1.1", nil, "lan,private\n", ""},
		{"ipv4 with search list", "192.168.1.1", []string{"private"}, "private\n", ""},
		{"ipv4 not found", "9.9.9.9", nil, "false\n", ""},
		{"ipv4 cidr", "192.168.0.0/24", nil, "lan,private\n", ""},
		{"ipv4 cidr not fully contained", "192.168.0.0/15", nil, "false\n", ""},
		{"ipv6 ip", "2001:250::1", nil, "cn\n", ""},
		{"ipv6 cidr", "fd00::/8", nil, "private\n", ""},
		{"ipv4-mapped ipv6 ip", "::ffff:8.8.8.8", nil, "us\n", ""},
		{"ipv4-mapped ipv6 cidr", "::ffff:192.168.1.0/120", nil, "lan,private\n", ""},
		{"ipv4-mapped ipv6 cidr too short", "::ffff:192.168.1.0/64", nil, "", lib.ErrInvalidCIDR.Error()},
		{"invalid ip", "1.2.3", nil, "", "invalid IP or CIDR"},
		{"invalid cidr", "1.2.3.4/33", nil, "", "invalid IP or CIDR"},
		{"invalid mapped cidr bits", "::ffff:1.2.3.4/129", nil, "", "invalid IP or CIDR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewLookup(lib.ActionOutput, WithLookupSearch(tt.search), WithLookupSearchList(tt.searchList), nil)
			if err != nil {
				t.Fatal(err)
			}
			c := newContainer(t, data)
			var outErr error
			got := captureStdout(t, func() { outErr = conv.Output(c) })
			if tt.wantErr != "" {
				if outErr == nil || !strings.Contains(outErr.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", outErr, tt.wantErr)
				}
				return
			}
			if outErr != nil {
				t.Fatal(outErr)
			}
			if got != tt.wantOut {
				t.Errorf("output: got %q, want %q", got, tt.wantOut)
			}
		})
	}
}

func TestNewStdoutFromBytes(t *testing.T) {
	tests := []struct {
		name        string
		data        string
		wantErr     bool
		wantWant    []string
		wantExclude []string
		wantType    lib.IPType
	}{
		{"empty", ``, false, nil, nil, ""},
		{"full", `{"wantedList":["cn"],"excludedList":["us"],"onlyIPType":"ipv6"}`, false, []string{"cn"}, []string{"us"}, lib.IPv6},
		{"invalid json", `{"wantedList":"cn"}`, true, nil, nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewStdoutFromBytes(lib.ActionOutput, []byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			s := conv.(*stdout)
			if !slices.Equal(s.Want, tt.wantWant) || !slices.Equal(s.Exclude, tt.wantExclude) || s.OnlyIPType != tt.wantType {
				t.Errorf("got (%v,%v,%q)", s.Want, s.Exclude, s.OnlyIPType)
			}
		})
	}
}

func TestStdoutOutput(t *testing.T) {
	data := map[string][]string{
		"CN": {"1.0.1.0/24", "1.0.2.0/23", "2001:250::/32"},
		"US": {"8.8.8.0/24", "8.8.9.0/24", "2001:4860::/32"},
		"V6": {"fc00::/8", "fd00::/8"},
	}
	tests := []struct {
		name    string
		want    []string
		exclude []string
		onlyIP  lib.IPType
		output  string
	}{
		{
			name:   "all sorted by name",
			output: "1.0.1.0/24\n1.0.2.0/23\n2001:250::/32\n8.8.8.0/23\n2001:4860::/32\nfc00::/7\n",
		},
		{
			name:   "wanted list with missing and blank names",
			want:   []string{" us ", "", "missing"},
			output: "8.8.8.0/23\n2001:4860::/32\n",
		},
		{
			name:    "excluded list",
			exclude: []string{"cn", " "},
			output:  "8.8.8.0/23\n2001:4860::/32\nfc00::/7\n",
		},
		{
			name:    "wanted list fully excluded",
			want:    []string{"cn"},
			exclude: []string{"CN"},
			output:  "",
		},
		{
			name:   "ipv4 only skips ipv6-only entry",
			onlyIP: lib.IPv4,
			output: "1.0.1.0/24\n1.0.2.0/23\n8.8.8.0/23\n",
		},
		{
			name:   "ipv6 only",
			want:   []string{"us", "cn"},
			onlyIP: lib.IPv6,
			output: "2001:250::/32\n2001:4860::/32\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewStdout(lib.ActionOutput,
				WithStdoutWantedList(tt.want),
				WithStdoutExcludedList(tt.exclude),
				WithStdoutOnlyIPType(tt.onlyIP),
				nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			c := newContainer(t, data)
			var outErr error
			got := captureStdout(t, func() { outErr = conv.Output(c) })
			if outErr != nil {
				t.Fatal(outErr)
			}
			if got != tt.output {
				t.Errorf("output:\n got %q\nwant %q", got, tt.output)
			}
			gotLines := strings.Count(got, "\n")
			wantLines := strings.Count(tt.output, "\n")
			if gotLines != wantLines {
				t.Errorf("line count: got %d want %d", gotLines, wantLines)
			}
		})
	}
}

func TestStdoutGenerateCIDRList(t *testing.T) {
	s := &stdout{}
	empty := lib.NewEntry("empty")
	if _, err := s.generateCIDRList(empty); err == nil {
		t.Error("expected error for entry without data")
	}

	v6only := lib.NewEntry("v6")
	if err := v6only.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	s.OnlyIPType = lib.IPv4
	if _, err := s.generateCIDRList(v6only); err == nil {
		t.Error("expected error when filtered list is empty")
	}
}
