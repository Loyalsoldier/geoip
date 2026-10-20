package plaintext

import (
	"crypto/x509"
	"errors"
	"io/fs"
	"maps"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestTextInConstructors(t *testing.T) {
	t.Parallel()

	testInputConstructors(t, TypeTextIn, DescTextIn, NewTextIn, NewTextInFromBytes,
		[]lib.InputOption{WithNameAndURI("cn", ""), WithIPOrCIDR([]string{"1.1.1.1"})},
		`{"name":"cn","uri":"./cn.txt","ipOrCIDR":["1.1.1.1"]}`,
	)
}

func TestTextIn_Config(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, dir, "data/cn.txt", "192.168.0.3/24\n192.168.1.5/24\n")
	writeTestFile(t, dir, "data/us.txt", "2001:db8::/33\n2001:db8:8000::/33\n")

	instance, err := newInstanceFromConfig(t, map[string]any{
		"input": []map[string]any{
			{"type": "text", "action": "add", "args": map[string]any{"inputDir": filepath.Join(dir, "data")}},
			{"type": "TEXT", "action": "add", "args": map[string]any{"name": "jp", "ipOrCIDR": []string{"::ffff:1.2.3.0/120"}}},
			{"type": "text", "action": "remove", "args": map[string]any{"name": "cn", "ipOrCIDR": []string{"192.168.1.0/24"}}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	container := lib.NewContainer()
	if err := instance.RunInput(container); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContainer(t, container, map[string][]string{
		"CN": {"192.168.0.0/24"},
		"US": {"2001:db8::/32"},
		"JP": {"1.2.3.0/24"},
	})
}

func TestTextIn_ConfigInvalid(t *testing.T) {
	t.Parallel()

	_, err := newInstanceFromConfig(t, map[string]any{
		"input": []map[string]any{
			{"type": "text", "action": "add", "args": map[string]any{"inputDir": "./dir", "name": "cn"}},
		},
	})
	assertErrContains(t, err, "❌ [type text | action add] inputDir is not allowed to be used with name or uri or ipOrCIDR")
}

func TestTextIn_Getters(t *testing.T) {
	t.Parallel()

	ti := &text_in{Type: "my-type", Action: lib.Action("my-action"), Description: "my description"}
	if got := ti.GetType(); got != "my-type" {
		t.Errorf("GetType() = %q, want %q", got, "my-type")
	}
	if got := ti.GetAction(); got != lib.Action("my-action") {
		t.Errorf("GetAction() = %q, want %q", got, "my-action")
	}
	if got := ti.GetDescription(); got != "my description" {
		t.Errorf("GetDescription() = %q, want %q", got, "my description")
	}
}

func TestTextIn_Input_LocalFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		content   string
		args      map[string]any
		action    lib.Action
		initial   map[string][]string
		want      map[string][]string
		wantErr   string
		wantErrIs error
	}{
		{
			name: "add new list next to existing ones",
			content: "# CN list\n" +
				"192.168.0.3/24\n" +
				"192.168.1.5/24 // merged with previous line\n" +
				"\n" +
				"2001:db8::/33\n" +
				"2001:db8:8000::/33 /* merged with previous line */\n" +
				"::ffff:1.2.3.4\n",
			args:    map[string]any{"name": "cn"},
			action:  lib.ActionAdd,
			initial: map[string][]string{"US": {"8.8.8.8"}},
			want: map[string][]string{
				"CN": {"1.2.3.4/32", "192.168.0.0/23", "2001:db8::/32"},
				"US": {"8.8.8.8/32"},
			},
		},
		{
			name:    "name is case-insensitive",
			content: "1.1.1.1\n",
			args:    map[string]any{"name": " Cn "},
			action:  lib.ActionAdd,
			initial: map[string][]string{"CN": {"1.1.1.2"}},
			want:    map[string][]string{"CN": {"1.1.1.1/32", "1.1.1.2/32"}},
		},
		{
			name:    "add merges into existing list",
			content: "11.0.0.0/8\n2001:db8::/32\n",
			args:    map[string]any{"name": "cn"},
			action:  lib.ActionAdd,
			initial: map[string][]string{"CN": {"10.0.0.0/8"}},
			want:    map[string][]string{"CN": {"10.0.0.0/7", "2001:db8::/32"}},
		},
		{
			name:    "uri and ipOrCIDR are combined",
			content: "192.168.0.3/24\n2001:db8::/33\n",
			args: map[string]any{
				"name":     "cn",
				"ipOrCIDR": []string{"192.168.1.5/24", " 2001:db8:8000::/33 ", "::ffff:1.2.3.0/120"},
			},
			action: lib.ActionAdd,
			want:   map[string][]string{"CN": {"1.2.3.0/24", "192.168.0.0/23", "2001:db8::/32"}},
		},
		{
			name:    "add only IPv4 into existing list",
			content: "11.0.0.0/8\n2001:db9::/32\n",
			args:    map[string]any{"name": "cn", "onlyIPType": "ipv4"},
			action:  lib.ActionAdd,
			initial: map[string][]string{"CN": {"10.0.0.0/8", "2001:db8::/32"}},
			want:    map[string][]string{"CN": {"10.0.0.0/7", "2001:db8::/32"}},
		},
		{
			name:    "add only IPv6 into new list",
			content: "1.1.1.1\n2001:db8::/32\n",
			args:    map[string]any{"name": "cn", "onlyIPType": "ipv6"},
			action:  lib.ActionAdd,
			want:    map[string][]string{"CN": {"2001:db8::/32"}},
		},
		{
			name:    "add with unknown onlyIPType does not filter",
			content: "1.1.1.1\n2001:db8::/32\n",
			args:    map[string]any{"name": "cn", "onlyIPType": "ipv5"},
			action:  lib.ActionAdd,
			want:    map[string][]string{"CN": {"1.1.1.1/32", "2001:db8::/32"}},
		},
		{
			name:    "remove prefixes",
			content: "10.0.0.0/9\n2001:db8::/33\n",
			args:    map[string]any{"name": "cn"},
			action:  lib.ActionRemove,
			initial: map[string][]string{"CN": {"10.0.0.0/8", "2001:db8::/32"}, "US": {"10.0.0.0/8"}},
			want: map[string][]string{
				"CN": {"10.128.0.0/9", "2001:db8:8000::/33"},
				"US": {"10.0.0.0/8"},
			},
		},
		{
			name:    "remove only IPv6",
			content: "10.0.0.0/9\n2001:db8::/33\n",
			args:    map[string]any{"name": "cn", "onlyIPType": "ipv6"},
			action:  lib.ActionRemove,
			initial: map[string][]string{"CN": {"10.0.0.0/8", "2001:db8::/32"}},
			want:    map[string][]string{"CN": {"10.0.0.0/8", "2001:db8:8000::/33"}},
		},
		{
			name:    "remove only IPv4",
			content: "10.0.0.0/9\n2001:db8::/33\n",
			args:    map[string]any{"name": "cn", "onlyIPType": "ipv4"},
			action:  lib.ActionRemove,
			initial: map[string][]string{"CN": {"10.0.0.0/8", "2001:db8::/32"}},
			want:    map[string][]string{"CN": {"10.128.0.0/9", "2001:db8::/32"}},
		},
		{
			name:    "remove everything keeps an empty list",
			content: "0.0.0.0/0\n::/0\n",
			args:    map[string]any{"name": "cn"},
			action:  lib.ActionRemove,
			initial: map[string][]string{"CN": {"10.0.0.0/8", "2001:db8::/32"}},
			want:    map[string][]string{"CN": nil},
		},
		{
			name:    "remove from missing list",
			content: "10.0.0.0/8\n",
			args:    map[string]any{"name": "cn"},
			action:  lib.ActionRemove,
			initial: map[string][]string{"US": {"10.0.0.0/8"}},
			wantErr: "entry CN not found",
		},
		{
			name:      "unknown action",
			content:   "10.0.0.0/8\n",
			args:      map[string]any{"name": "cn"},
			action:    lib.Action("unknown"),
			wantErrIs: lib.ErrUnknownAction,
		},
		{
			name:      "output action is not supported for input",
			content:   "10.0.0.0/8\n",
			args:      map[string]any{"name": "cn"},
			action:    lib.ActionOutput,
			wantErrIs: lib.ErrUnknownAction,
		},
		{
			name: "remove prefixes and suffixes in line",
			content: "IP-CIDR,1.1.1.0/24,no-resolve\n" +
				"IP-CIDR6,2001:db8::/32,no-resolve\n" +
				"HOST,8.8.8.8\n",
			args: map[string]any{
				"name":                 "cn",
				"removePrefixesInLine": []string{"IP-CIDR,", "IP-CIDR6,", "HOST,"},
				"removeSuffixesInLine": []string{",no-resolve"},
			},
			action: lib.ActionAdd,
			want:   map[string][]string{"CN": {"1.1.1.0/24", "8.8.8.8/32", "2001:db8::/32"}},
		},
		{
			name:    "wantedList containing the list",
			content: "1.1.1.1\n",
			args:    map[string]any{"name": "cn", "wantedList": []string{"us", " CN "}},
			action:  lib.ActionAdd,
			want:    map[string][]string{"CN": {"1.1.1.1/32"}},
		},
		{
			name:    "wantedList not containing the list",
			content: "1.1.1.1\n",
			args:    map[string]any{"name": "cn", "wantedList": []string{"us"}},
			action:  lib.ActionAdd,
			wantErr: "no entry is generated",
		},
		{
			name:    "file with only comments creates an empty list",
			content: "# nothing here\n\n// still nothing\n",
			args:    map[string]any{"name": "cn"},
			action:  lib.ActionAdd,
			want:    map[string][]string{"CN": nil},
		},
		{
			name:      "invalid line",
			content:   "1.1.1.1\nnot-an-ip\n",
			args:      map[string]any{"name": "cn"},
			action:    lib.ActionAdd,
			wantErrIs: lib.ErrInvalidIP,
		},
		{
			name:      "malformed IPv4-mapped CIDR in file",
			content:   "::ffff:1.2.3.0/95\n",
			args:      map[string]any{"name": "cn"},
			action:    lib.ActionAdd,
			wantErrIs: lib.ErrInvalidCIDR,
		},
		{
			name:      "malformed ipOrCIDR after valid file",
			content:   "1.1.1.1\n",
			args:      map[string]any{"name": "cn", "ipOrCIDR": []string{"::ffff:1.2.3.4/129"}},
			action:    lib.ActionAdd,
			wantErrIs: lib.ErrInvalidCIDR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			args := maps.Clone(tt.args)
			args["uri"] = writeTestFile(t, t.TempDir(), "list.txt", tt.content)

			got, err := runInput(t, TypeTextIn, tt.action, args, tt.initial)
			assertInputResult(t, TypeTextIn, tt.action, got, err, tt.want, tt.wantErr, tt.wantErrIs)
		})
	}
}

func TestTextIn_Input_LocalFileNotFound(t *testing.T) {
	t.Parallel()

	args := map[string]any{"name": "cn", "uri": filepath.Join(t.TempDir(), "missing.txt")}
	got, err := runInput(t, TypeTextIn, lib.ActionAdd, args, nil)
	assertInputResult(t, TypeTextIn, lib.ActionAdd, got, err, nil, "", fs.ErrNotExist)
}

func TestTextIn_Input_IPOrCIDR(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      map[string]any
		action    lib.Action
		initial   map[string][]string
		want      map[string][]string
		wantErrIs error
	}{
		{
			name: "mixed addresses, CIDRs, comments and blanks",
			args: map[string]any{
				"name": "cn",
				"ipOrCIDR": []string{
					"1.1.1.1",
					"2001:db8::1",
					"10.0.0.0/8",
					" 192.168.0.3/24 ",
					"192.168.1.5/24",
					"2001:DB8:1::/48",
					"::ffff:1.2.3.4",
					"::ffff:5.6.7.0/120",
					"9.9.9.9 # comment",
					"",
					"   ",
					"// comment",
				},
			},
			action: lib.ActionAdd,
			want: map[string][]string{
				"CN": {
					"1.1.1.1/32",
					"1.2.3.4/32",
					"5.6.7.0/24",
					"9.9.9.9/32",
					"10.0.0.0/8",
					"192.168.0.0/23",
					"2001:db8::1/128",
					"2001:db8:1::/48",
				},
			},
		},
		{
			name:   "IPv6 CIDRs are merged",
			args:   map[string]any{"name": "cn", "ipOrCIDR": []string{"2001:db8::/33", "2001:db8:8000::/33"}},
			action: lib.ActionAdd,
			want:   map[string][]string{"CN": {"2001:db8::/32"}},
		},
		{
			name:   "only comments and blanks create an empty list",
			args:   map[string]any{"name": "cn", "ipOrCIDR": []string{"", "# comment"}},
			action: lib.ActionAdd,
			want:   map[string][]string{"CN": nil},
		},
		{
			name:    "add only IPv4",
			args:    map[string]any{"name": "cn", "ipOrCIDR": []string{"1.1.1.1", "2001:db8::/32"}, "onlyIPType": "ipv4"},
			action:  lib.ActionAdd,
			initial: map[string][]string{"US": {"8.8.8.8"}},
			want:    map[string][]string{"CN": {"1.1.1.1/32"}, "US": {"8.8.8.8/32"}},
		},
		{
			name:    "remove",
			args:    map[string]any{"name": "cn", "ipOrCIDR": []string{"10.0.0.0/9", "::ffff:10.128.0.0/112"}},
			action:  lib.ActionRemove,
			initial: map[string][]string{"CN": {"10.0.0.0/8"}},
			want:    map[string][]string{"CN": {"10.129.0.0/16", "10.130.0.0/15", "10.132.0.0/14", "10.136.0.0/13", "10.144.0.0/12", "10.160.0.0/11", "10.192.0.0/10"}},
		},
		{
			name:      "IPv4-mapped CIDR with too short prefix",
			args:      map[string]any{"name": "cn", "ipOrCIDR": []string{"::ffff:1.2.3.0/95"}},
			action:    lib.ActionAdd,
			wantErrIs: lib.ErrInvalidCIDR,
		},
		{
			name:      "IPv4-mapped CIDR with too long prefix",
			args:      map[string]any{"name": "cn", "ipOrCIDR": []string{"::ffff:1.2.3.4/129"}},
			action:    lib.ActionAdd,
			wantErrIs: lib.ErrInvalidCIDR,
		},
		{
			name:      "negative prefix length",
			args:      map[string]any{"name": "cn", "ipOrCIDR": []string{"1.1.1.1/-1"}},
			action:    lib.ActionAdd,
			wantErrIs: lib.ErrInvalidCIDR,
		},
		{
			name:      "invalid IP",
			args:      map[string]any{"name": "cn", "ipOrCIDR": []string{"1.1.1.1", "abc"}},
			action:    lib.ActionAdd,
			wantErrIs: lib.ErrInvalidIP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := runInput(t, TypeTextIn, tt.action, tt.args, tt.initial)
			assertInputResult(t, TypeTextIn, tt.action, got, err, tt.want, "", tt.wantErrIs)
		})
	}
}

func TestTextIn_Input_InputDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		files     map[string]string
		subPath   string // appended to the temp dir to form inputDir
		args      map[string]any
		action    lib.Action
		initial   map[string][]string
		want      map[string][]string
		wantErr   string
		wantErrIs error
	}{
		{
			name: "entry names are derived from file names",
			files: map[string]string{
				"cn.txt":             "1.1.1.1\n",
				"US.LIST":            "2.2.2.2\n",
				"jp":                 "3.3.3.3\n",
				".hidden":            "4.4.4.4\n",
				"geo-ip_list.v2.txt": "5.5.5.5\n",
			},
			action: lib.ActionAdd,
			want: map[string][]string{
				"CN":             {"1.1.1.1/32"},
				"US":             {"2.2.2.2/32"},
				"JP":             {"3.3.3.3/32"},
				".HIDDEN":        {"4.4.4.4/32"},
				"GEO-IP_LIST.V2": {"5.5.5.5/32"},
			},
		},
		{
			name: "files in subdirectories are walked too",
			files: map[string]string{
				"cn.txt":        "1.1.1.1\n",
				"sub/us.txt":    "2.2.2.2\n",
				"sub/deep/jp":   "3.3.3.3\n",
				"empty/.keep/x": "",
			},
			action: lib.ActionAdd,
			want: map[string][]string{
				"CN": {"1.1.1.1/32"},
				"US": {"2.2.2.2/32"},
				"JP": {"3.3.3.3/32"},
				"X":  nil,
			},
		},
		{
			name: "wantedList filters files",
			files: map[string]string{
				"cn.txt": "192.168.0.3/24\n192.168.1.5/24\n",
				"us.txt": "2.2.2.2\n",
				"jp.txt": "2001:db8::/33\n2001:db8:8000::/33\n",
				"bad":    "this file is not wanted so it is never parsed\n",
			},
			args:   map[string]any{"wantedList": []string{"cn", " JP ", ""}},
			action: lib.ActionAdd,
			want: map[string][]string{
				"CN": {"192.168.0.0/23"},
				"JP": {"2001:db8::/32"},
			},
		},
		{
			name:    "wantedList matches nothing",
			files:   map[string]string{"cn.txt": "1.1.1.1\n"},
			args:    map[string]any{"wantedList": []string{"us"}},
			action:  lib.ActionAdd,
			wantErr: "no entry is generated",
		},
		{
			name:    "empty directory",
			files:   map[string]string{},
			action:  lib.ActionAdd,
			wantErr: "no entry is generated",
		},
		{
			name: "onlyIPType ipv6",
			files: map[string]string{
				"cn.txt": "1.1.1.1\n2001:db8::/32\n",
				"us.txt": "2.2.2.2\n",
			},
			args:   map[string]any{"onlyIPType": "ipv6"},
			action: lib.ActionAdd,
			want: map[string][]string{
				"CN": {"2001:db8::/32"},
				"US": nil,
			},
		},
		{
			name: "removePrefixesInLine and removeSuffixesInLine apply to every file",
			files: map[string]string{
				"cn.txt": "IP-CIDR,1.1.1.1/32,no-resolve\n",
				"us.txt": "IP-CIDR6,2001:db8::/32,no-resolve\n",
			},
			args: map[string]any{
				"removePrefixesInLine": []string{"ip-cidr,", "ip-cidr6,"},
				"removeSuffixesInLine": []string{",no-resolve"},
			},
			action: lib.ActionAdd,
			want: map[string][]string{
				"CN": {"1.1.1.1/32"},
				"US": {"2001:db8::/32"},
			},
		},
		{
			name: "remove from existing lists",
			files: map[string]string{
				"cn.txt": "10.0.0.0/9\n",
				"us.txt": "2001:db8::/33\n",
			},
			action: lib.ActionRemove,
			initial: map[string][]string{
				"CN": {"10.0.0.0/8"},
				"US": {"2001:db8::/32"},
				"JP": {"10.0.0.0/8"},
			},
			want: map[string][]string{
				"CN": {"10.128.0.0/9"},
				"US": {"2001:db8:8000::/33"},
				"JP": {"10.0.0.0/8"},
			},
		},
		{
			name:    "inputDir pointing to a regular file reads that file",
			files:   map[string]string{"cn.txt": "1.1.1.1\n"},
			subPath: "cn.txt",
			action:  lib.ActionAdd,
			want:    map[string][]string{"CN": {"1.1.1.1/32"}},
		},
		{
			name: "duplicated list names from different extensions",
			files: map[string]string{
				"cn.txt":  "1.1.1.1\n",
				"cn.list": "2.2.2.2\n",
			},
			action:  lib.ActionAdd,
			wantErr: "found duplicated list CN",
		},
		{
			name: "duplicated list names from subdirectory",
			files: map[string]string{
				"cn.txt":     "1.1.1.1\n",
				"sub/CN.txt": "2.2.2.2\n",
			},
			action:  lib.ActionAdd,
			wantErr: "found duplicated list CN",
		},
		{
			name:    "filename with space",
			files:   map[string]string{"c n.txt": "1.1.1.1\n"},
			action:  lib.ActionAdd,
			wantErr: "filename c n.txt cannot be entry name, please remove special characters in it",
		},
		{
			name:    "filename with non-ASCII characters",
			files:   map[string]string{"中国.txt": "1.1.1.1\n"},
			action:  lib.ActionAdd,
			wantErr: "filename 中国.txt cannot be entry name",
		},
		{
			name:    "filename with symbol",
			files:   map[string]string{"cn@home.txt": "1.1.1.1\n"},
			action:  lib.ActionAdd,
			wantErr: "filename cn@home.txt cannot be entry name",
		},
		{
			name: "invalid content in one file",
			files: map[string]string{
				"cn.txt": "1.1.1.1\n",
				"us.txt": "1.1.1.1/33\n",
			},
			action:    lib.ActionAdd,
			wantErrIs: lib.ErrInvalidCIDR,
		},
		{
			name:      "nonexistent directory",
			files:     map[string]string{},
			subPath:   "missing",
			action:    lib.ActionAdd,
			wantErrIs: fs.ErrNotExist,
		},
		{
			name:    "remove from missing list",
			files:   map[string]string{"cn.txt": "1.1.1.1\n"},
			action:  lib.ActionRemove,
			wantErr: "entry CN not found",
		},
		{
			name:      "unknown action",
			files:     map[string]string{"cn.txt": "1.1.1.1\n"},
			action:    lib.Action("unknown"),
			wantErrIs: lib.ErrUnknownAction,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			for name, content := range tt.files {
				writeTestFile(t, dir, name, content)
			}

			args := maps.Clone(tt.args)
			if args == nil {
				args = make(map[string]any)
			}
			args["inputDir"] = filepath.Join(dir, tt.subPath)

			got, err := runInput(t, TypeTextIn, tt.action, args, tt.initial)
			assertInputResult(t, TypeTextIn, tt.action, got, err, tt.want, tt.wantErr, tt.wantErrIs)
		})
	}
}

func TestTextIn_Input_Remote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		status      int
		body        string
		args        map[string]any
		upperScheme bool
		action      lib.Action
		initial     map[string][]string
		want        map[string][]string
		wantErr     string
		wantErrIs   error
	}{
		{
			name:   "status 200",
			status: http.StatusOK,
			body:   "# remote list\n192.168.0.3/24\n192.168.1.5/24\n2001:db8::/33\n2001:db8:8000::/33\n",
			args:   map[string]any{"name": "cn"},
			action: lib.ActionAdd,
			want:   map[string][]string{"CN": {"192.168.0.0/23", "2001:db8::/32"}},
		},
		{
			name:        "upper-case scheme",
			status:      http.StatusOK,
			body:        "1.1.1.1\n",
			args:        map[string]any{"name": "cn"},
			upperScheme: true,
			action:      lib.ActionAdd,
			want:        map[string][]string{"CN": {"1.1.1.1/32"}},
		},
		{
			name:   "remote and ipOrCIDR are combined",
			status: http.StatusOK,
			body:   "1.1.1.0/25\n",
			args:   map[string]any{"name": "cn", "ipOrCIDR": []string{"1.1.1.128/25", "::ffff:2.2.2.2"}},
			action: lib.ActionAdd,
			want:   map[string][]string{"CN": {"1.1.1.0/24", "2.2.2.2/32"}},
		},
		{
			name:    "remove with onlyIPType",
			status:  http.StatusOK,
			body:    "10.0.0.0/8\n2001:db8::/33\n",
			args:    map[string]any{"name": "cn", "onlyIPType": "ipv6"},
			action:  lib.ActionRemove,
			initial: map[string][]string{"CN": {"10.0.0.0/8", "2001:db8::/32"}},
			want:    map[string][]string{"CN": {"10.0.0.0/8", "2001:db8:8000::/33"}},
		},
		{
			name:   "wantedList containing the list",
			status: http.StatusOK,
			body:   "1.1.1.1\n",
			args:   map[string]any{"name": "cn", "wantedList": []string{"cn"}},
			action: lib.ActionAdd,
			want:   map[string][]string{"CN": {"1.1.1.1/32"}},
		},
		{
			name:    "wantedList not containing the list",
			status:  http.StatusOK,
			body:    "1.1.1.1\n",
			args:    map[string]any{"name": "cn", "wantedList": []string{"us"}},
			action:  lib.ActionAdd,
			wantErr: "no entry is generated",
		},
		{
			name:    "status 404",
			status:  http.StatusNotFound,
			body:    "not found",
			args:    map[string]any{"name": "cn"},
			action:  lib.ActionAdd,
			wantErr: "http status code 404",
		},
		{
			name:    "status 500",
			status:  http.StatusInternalServerError,
			body:    "1.1.1.1\n",
			args:    map[string]any{"name": "cn"},
			action:  lib.ActionAdd,
			wantErr: "http status code 500",
		},
		{
			name:    "status 204 is not accepted",
			status:  http.StatusNoContent,
			args:    map[string]any{"name": "cn"},
			action:  lib.ActionAdd,
			wantErr: "http status code 204",
		},
		{
			name:      "invalid remote content",
			status:    http.StatusOK,
			body:      "1.1.1.1\n<html>\n",
			args:      map[string]any{"name": "cn"},
			action:    lib.ActionAdd,
			wantErrIs: lib.ErrInvalidIP,
		},
		{
			name:    "remove from missing list",
			status:  http.StatusOK,
			body:    "1.1.1.1\n",
			args:    map[string]any{"name": "cn"},
			action:  lib.ActionRemove,
			wantErr: "entry CN not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := newTestServer(t, tt.status, tt.body)
			uri := srv.URL + "/cn.txt"
			if tt.upperScheme {
				uri = "HTTP://" + strings.TrimPrefix(uri, "http://")
			}

			args := maps.Clone(tt.args)
			args["uri"] = uri

			got, err := runInput(t, TypeTextIn, tt.action, args, tt.initial)
			assertInputResult(t, TypeTextIn, tt.action, got, err, tt.want, tt.wantErr, tt.wantErrIs)
			if tt.wantErr != "" && strings.HasPrefix(tt.wantErr, "http status code") {
				assertErrContains(t, err, "failed to get remote file "+uri)
			}
		})
	}
}

func TestTextIn_Input_RemoteConnectionAborted(t *testing.T) {
	t.Parallel()

	args := map[string]any{"name": "cn", "uri": failingServerURL(t) + "/cn.txt"}
	got, err := runInput(t, TypeTextIn, lib.ActionAdd, args, nil)
	assertInputResult(t, TypeTextIn, lib.ActionAdd, got, err, nil, "[type text | action add]", nil)

	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		t.Fatalf("expected *url.Error, got %T: %v", errors.Unwrap(err), err)
	}
}

func TestTextIn_Input_HTTPSUntrustedCertificate(t *testing.T) {
	t.Parallel()

	srv := newTestTLSServer(t, "1.1.1.1\n")
	args := map[string]any{"name": "cn", "uri": srv.URL + "/cn.txt"}
	got, err := runInput(t, TypeTextIn, lib.ActionAdd, args, nil)
	assertInputResult(t, TypeTextIn, lib.ActionAdd, got, err, nil, "[type text | action add]", nil)

	var certErr x509.UnknownAuthorityError
	if !errors.As(err, &certErr) {
		t.Fatalf("expected x509.UnknownAuthorityError, got %v", err)
	}
}

// TestTextIn_Input_HTTPS must not run in parallel since it temporarily
// replaces http.DefaultTransport so that the test server certificate is
// trusted.
func TestTextIn_Input_HTTPS(t *testing.T) {
	srv := newTestTLSServer(t, "192.168.0.3/24\n192.168.1.5/24\n")

	prevTransport := http.DefaultTransport
	http.DefaultTransport = srv.Client().Transport
	t.Cleanup(func() { http.DefaultTransport = prevTransport })

	for _, scheme := range []string{"https://", "HTTPS://"} {
		t.Run(scheme, func(t *testing.T) {
			args := map[string]any{"name": "cn", "uri": scheme + strings.TrimPrefix(srv.URL, "https://") + "/cn.txt"}
			got, err := runInput(t, TypeTextIn, lib.ActionAdd, args, nil)
			assertInputResult(t, TypeTextIn, lib.ActionAdd, got, err, map[string][]string{"CN": {"192.168.0.0/23"}}, "", nil)
		})
	}
}

func TestTextIn_Input_MissingArguments(t *testing.T) {
	t.Parallel()

	// These converters bypass the validation of newTextIn on purpose to reach
	// the defensive default branch of Input.
	tests := []struct {
		name string
		ti   *text_in
	}{
		{name: "nothing", ti: &text_in{Type: TypeTextIn, Action: lib.ActionAdd}},
		{name: "name only", ti: &text_in{Type: TypeTextIn, Action: lib.ActionAdd, Name: "cn"}},
		{name: "uri only", ti: &text_in{Type: TypeTextIn, Action: lib.ActionAdd, URI: "./cn.txt"}},
		{name: "ipOrCIDR only", ti: &text_in{Type: TypeTextIn, Action: lib.ActionAdd, IPOrCIDR: []string{"1.1.1.1"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.ti.Input(lib.NewContainer())
			assertErrContains(t, err, "❌ [type text | action add] config missing argument inputDir or name or uri or ipOrCIDR")
			if got != nil {
				t.Errorf("expected nil container, got %#v", got)
			}
		})
	}
}

func TestTextIn_WalkLocalFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		filename  string
		listName  string
		want      map[string]bool
		existing  []string
		wantList  string // expected entry name, empty if no entry should be created
		wantErr   string
		wantErrIs error
	}{
		{name: "lower-case name with extension", filename: "cn.txt", wantList: "CN"},
		{name: "upper-case name with extension", filename: "CN.TXT", wantList: "CN"},
		{name: "multiple dots keep all but last extension", filename: "geo-ip_list.v2.txt", wantList: "GEO-IP_LIST.V2"},
		{name: "hidden file without extension", filename: ".hidden", wantList: ".HIDDEN"},
		{name: "hidden file with extension", filename: ".hidden.txt", wantList: ".HIDDEN"},
		{name: "no extension", filename: "noext", wantList: "NOEXT"},
		{name: "trailing dot", filename: "trailing.", wantList: "TRAILING"},
		{name: "explicit name overrides filename", filename: "c n@file.txt", listName: " us ", wantList: "US"},
		{name: "explicit name is not checked for special characters", filename: "cn.txt", listName: "my list", wantList: "MY LIST"},
		{name: "wanted list contains entry", filename: "cn.txt", want: map[string]bool{"CN": true}, wantList: "CN"},
		{name: "wanted list does not contain entry", filename: "cn.txt", want: map[string]bool{"US": true}},
		{name: "wanted list with explicit name", filename: "cn.txt", listName: "us", want: map[string]bool{"US": true}, wantList: "US"},
		{name: "space in filename", filename: "c n.txt", wantErr: "filename c n.txt cannot be entry name"},
		{name: "plus in filename", filename: "cn+1.txt", wantErr: "filename cn+1.txt cannot be entry name"},
		{name: "duplicated list", filename: "cn.txt", existing: []string{"CN"}, wantErr: "found duplicated list CN"},
		{name: "duplicated list with explicit name", filename: "a.txt", listName: "cn", existing: []string{"CN"}, wantErr: "found duplicated list CN"},
		{name: "duplicated but not wanted", filename: "cn.txt", existing: []string{"CN"}, want: map[string]bool{"US": true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := writeTestFile(t, t.TempDir(), tt.filename, "1.1.1.1\n")
			ti := &text_in{Type: TypeTextIn, Want: tt.want}
			entries := make(map[string]*lib.Entry)
			for _, name := range tt.existing {
				entries[name] = lib.NewEntry(name)
			}

			err := ti.walkLocalFile(path, tt.listName, entries)
			assertErr(t, err, tt.wantErr, tt.wantErrIs)
			if tt.wantErr != "" {
				return
			}

			wantNames := slices.Clone(tt.existing)
			if tt.wantList != "" {
				wantNames = append(wantNames, tt.wantList)
			}
			slices.Sort(wantNames)
			assertStrings(t, "entry names", sortedKeys(entries), wantNames)

			if tt.wantList != "" {
				entry := entries[tt.wantList]
				if entry.GetName() != tt.wantList {
					t.Errorf("entry name = %q, want %q", entry.GetName(), tt.wantList)
				}
				assertStrings(t, "CIDRs", entryCIDRs(t, entry), []string{"1.1.1.1/32"})
			}
		})
	}
}

func TestTextIn_WalkLocalFile_OpenError(t *testing.T) {
	t.Parallel()

	ti := &text_in{Type: TypeTextIn}
	entries := make(map[string]*lib.Entry)
	err := ti.walkLocalFile(filepath.Join(t.TempDir(), "missing.txt"), "cn", entries)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected %v, got %v", fs.ErrNotExist, err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no entry, got %v", sortedKeys(entries))
	}
}

func TestTextIn_WalkLocalFile_ScanError(t *testing.T) {
	t.Parallel()

	path := writeTestFile(t, t.TempDir(), "cn.txt", "1.1.1.1\ninvalid\n")
	ti := &text_in{Type: TypeTextIn}
	entries := make(map[string]*lib.Entry)
	err := ti.walkLocalFile(path, "", entries)
	if !errors.Is(err, lib.ErrInvalidIP) {
		t.Fatalf("expected %v, got %v", lib.ErrInvalidIP, err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no entry on error, got %v", sortedKeys(entries))
	}
}

func TestTextIn_WalkDir(t *testing.T) {
	t.Parallel()

	t.Run("walks nested directories", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		writeTestFile(t, dir, "cn.txt", "1.1.1.1\n")
		writeTestFile(t, dir, "a/b/c/us.txt", "2001:db8::/32\n")

		ti := &text_in{Type: TypeTextIn}
		entries := make(map[string]*lib.Entry)
		if err := ti.walkDir(dir, entries); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertStrings(t, "entry names", sortedKeys(entries), []string{"CN", "US"})
		assertStrings(t, "CIDRs of CN", entryCIDRs(t, entries["CN"]), []string{"1.1.1.1/32"})
		assertStrings(t, "CIDRs of US", entryCIDRs(t, entries["US"]), []string{"2001:db8::/32"})
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		t.Parallel()

		ti := &text_in{Type: TypeTextIn}
		err := ti.walkDir(filepath.Join(t.TempDir(), "missing"), make(map[string]*lib.Entry))
		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("expected %v, got %v", fs.ErrNotExist, err)
		}
	})

	t.Run("stops at first invalid file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		writeTestFile(t, dir, "a.txt", "1.1.1.1\n")
		writeTestFile(t, dir, "b c.txt", "2.2.2.2\n")

		ti := &text_in{Type: TypeTextIn}
		err := ti.walkDir(dir, make(map[string]*lib.Entry))
		assertErrContains(t, err, "filename b c.txt cannot be entry name")
	})
}

func TestTextIn_WalkRemoteFile(t *testing.T) {
	t.Parallel()

	t.Run("name is upper-cased", func(t *testing.T) {
		t.Parallel()

		srv := newTestServer(t, http.StatusOK, "1.1.1.1\n")
		ti := &text_in{Type: TypeTextIn}
		entries := make(map[string]*lib.Entry)
		if err := ti.walkRemoteFile(srv.URL, "cn", entries); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertStrings(t, "entry names", sortedKeys(entries), []string{"CN"})
		assertStrings(t, "CIDRs", entryCIDRs(t, entries["CN"]), []string{"1.1.1.1/32"})
	})

	t.Run("scan error does not create entry", func(t *testing.T) {
		t.Parallel()

		srv := newTestServer(t, http.StatusOK, "payload:\n  - IP-CIDR,1.1.1.1/33\n")
		ti := &text_in{Type: TypeClashRuleSetClassicalIn}
		entries := make(map[string]*lib.Entry)
		err := ti.walkRemoteFile(srv.URL, "cn", entries)
		if !errors.Is(err, lib.ErrInvalidCIDR) {
			t.Fatalf("expected %v, got %v", lib.ErrInvalidCIDR, err)
		}
		if len(entries) != 0 {
			t.Errorf("expected no entry, got %v", sortedKeys(entries))
		}
	})

	t.Run("invalid URL", func(t *testing.T) {
		t.Parallel()

		ti := &text_in{Type: TypeTextIn}
		err := ti.walkRemoteFile("http://[::1", "cn", make(map[string]*lib.Entry))
		var urlErr *url.Error
		if !errors.As(err, &urlErr) {
			t.Fatalf("expected *url.Error, got %v", err)
		}
	})
}

func TestTextIn_AppendIPOrCIDR(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ipOrCIDR []string
		existing map[string][]string
		want     map[string][]string
		wantErr  error
	}{
		{
			name:     "nil list does nothing",
			ipOrCIDR: nil,
			want:     map[string][]string{},
		},
		{
			name:     "empty list does not create entry",
			ipOrCIDR: []string{},
			existing: map[string][]string{"US": {"2.2.2.2"}},
			want:     map[string][]string{"US": {"2.2.2.2/32"}},
		},
		{
			name:     "creates new entry",
			ipOrCIDR: []string{" 1.1.1.1 ", "2001:db8::/32"},
			want:     map[string][]string{"CN": {"1.1.1.1/32", "2001:db8::/32"}},
		},
		{
			name:     "merges into existing entry",
			ipOrCIDR: []string{"192.168.1.5/24"},
			existing: map[string][]string{"CN": {"192.168.0.3/24"}, "US": {"2.2.2.2"}},
			want:     map[string][]string{"CN": {"192.168.0.0/23"}, "US": {"2.2.2.2/32"}},
		},
		{
			name:     "invalid value does not create entry",
			ipOrCIDR: []string{"1.1.1.1", "::ffff:1.2.3.0/95"},
			want:     map[string][]string{},
			wantErr:  lib.ErrInvalidCIDR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entries := make(map[string]*lib.Entry)
			for name, cidrs := range tt.existing {
				entries[name] = newTestEntry(t, name, cidrs...)
			}

			ti := &text_in{Type: TypeTextIn}
			err := ti.appendIPOrCIDR(tt.ipOrCIDR, "cn", entries)
			assertErr(t, err, "", tt.wantErr)

			assertStrings(t, "entry names", sortedKeys(entries), sortedKeys(tt.want))
			for name, wantCIDRs := range tt.want {
				assertStrings(t, "CIDRs of "+name, entryCIDRs(t, entries[name]), wantCIDRs)
			}
		})
	}
}
