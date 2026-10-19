package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/Loyalsoldier/geoip/plugin/maxmind"
	"github.com/Loyalsoldier/geoip/plugin/mihomo"
	"github.com/Loyalsoldier/geoip/plugin/plaintext"
	"github.com/Loyalsoldier/geoip/plugin/singbox"
	"github.com/Loyalsoldier/geoip/plugin/special"
	"github.com/Loyalsoldier/geoip/plugin/v2ray"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// lookupFixture creates:
//
//	dir/cn.txt : 1.0.0.0/24, 2001:db8::/32
//	dir/us.txt : 8.8.8.0/24, 1.0.0.0/16
//	single.txt : 1.0.0.0/24, 2001:db8::/32
//	clash.yaml / classical.yaml / surge.list with 1.0.0.0/24 and 2001:db8::/32
func lookupFixture(t *testing.T) (dir string, files map[string]string) {
	t.Helper()
	root := t.TempDir()
	dir = filepath.Join(root, "lists")
	writeFile(t, filepath.Join(dir, "cn.txt"), "1.0.0.0/24\n2001:db8::/32\n")
	writeFile(t, filepath.Join(dir, "us.txt"), "# US\n8.8.8.0/24\n1.0.0.0/16\n")

	files = map[string]string{
		plaintext.TypeTextIn:                  filepath.Join(root, "single.txt"),
		plaintext.TypeClashRuleSetIPCIDRIn:    filepath.Join(root, "clash.yaml"),
		plaintext.TypeClashRuleSetClassicalIn: filepath.Join(root, "classical.yaml"),
		plaintext.TypeSurgeRuleSetIn:          filepath.Join(root, "surge.list"),
	}
	writeFile(t, files[plaintext.TypeTextIn], "1.0.0.0/24\n2001:db8::/32\n")
	writeFile(t, files[plaintext.TypeClashRuleSetIPCIDRIn], "payload:\n  - '1.0.0.0/24'\n  - '2001:db8::/32'\n")
	writeFile(t, files[plaintext.TypeClashRuleSetClassicalIn], "payload:\n  - IP-CIDR,1.0.0.0/24,no-resolve\n  - IP-CIDR6,2001:db8::/32\n  - DOMAIN,example.com\n")
	writeFile(t, files[plaintext.TypeSurgeRuleSetIn], "IP-CIDR,1.0.0.0/24\nIP-CIDR6,2001:db8::/32,no-resolve\nDOMAIN,example.com\n")
	return dir, files
}

func TestLookupCmdOnceModeDir(t *testing.T) {
	dir, _ := lookupFixture(t)

	tests := []struct {
		name   string
		search string
		extra  []string
		want   string
	}{
		{"ipv4 in two lists", "1.0.0.1", nil, "cn,us"},
		{"ipv4 in one list", "8.8.8.8", nil, "us"},
		{"ipv4 in /16 only", "1.0.5.5", nil, "us"},
		{"ipv4 not found", "9.9.9.9", nil, "false"},
		{"ipv6 found", "2001:db8::1", nil, "cn"},
		{"ipv6 uppercase and spaces", "  2001:DB8::ABCD  ", nil, "cn"},
		{"ipv6 not found", "2001:db9::1", nil, "false"},
		{"ipv4 cidr contained in both", "1.0.0.0/25", nil, "cn,us"},
		{"ipv4 cidr contained in one", "1.0.0.0/20", nil, "us"},
		{"ipv4 cidr not contained", "1.0.0.0/8", nil, "false"},
		{"ipv6 cidr contained", "2001:db8:1::/48", nil, "cn"},
		{"ipv4-mapped ipv6 address", "::ffff:1.0.0.1", nil, "cn,us"},
		{"ipv4-mapped ipv6 cidr", "::ffff:1.0.0.0/120", nil, "cn,us"},
		{"invalid ip", "not-an-ip", nil, "false"},
		{"invalid cidr", "1.0.0.0/33", nil, "false"},
		{"searchlist filters", "1.0.0.1", []string{"-l", "cn"}, "cn"},
		{"searchlist uppercase", "1.0.0.1", []string{"--searchlist", "US"}, "us"},
		{"searchlist multiple", "1.0.0.1", []string{"-l", "cn,us"}, "cn,us"},
		{"searchlist excludes match", "8.8.8.8", []string{"-l", "cn"}, "false"},
		{"searchlist unknown list", "8.8.8.8", []string{"-l", "jp"}, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"lookup", "-f", "text", "-d", dir}, tt.extra...)
			args = append(args, tt.search)
			var err error
			out := captureStdout(t, func() { err = runCmd(t, args...) })
			if err != nil {
				t.Fatalf("Execute error: %v", err)
			}
			if out != tt.want+"\n" {
				t.Fatalf("output = %q, want %q", out, tt.want+"\n")
			}
		})
	}
}

func TestLookupCmdOnceModeURI(t *testing.T) {
	_, files := lookupFixture(t)

	type tc struct {
		name   string
		format string
		search string
		want   string
	}
	var tests []tc
	for _, format := range []string{
		plaintext.TypeTextIn,
		plaintext.TypeClashRuleSetIPCIDRIn,
		plaintext.TypeClashRuleSetClassicalIn,
		plaintext.TypeSurgeRuleSetIn,
	} {
		tests = append(tests,
			tc{format + " ipv4 found", format, "1.0.0.1", "true"},
			tc{format + " ipv6 found", format, "2001:db8::1", "true"},
			tc{format + " not found", format, "8.8.8.8", "false"},
		)
	}
	// format is case-insensitive and trimmed
	tests = append(tests, tc{"format upper-case", " TEXT ", "1.0.0.1", "true"})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri := files[strings.ToLower(strings.TrimSpace(tt.format))]
			if uri == "" {
				uri = files[strings.TrimSpace(tt.format)]
			}
			var err error
			out := captureStdout(t, func() {
				err = runCmd(t, "find", "--format", tt.format, "--uri", uri, tt.search)
			})
			if err != nil {
				t.Fatalf("Execute error: %v", err)
			}
			if out != tt.want+"\n" {
				t.Fatalf("output = %q, want %q", out, tt.want+"\n")
			}
		})
	}
}

// An invalid search argument short-circuits before any input is loaded, so a
// non-existent URI never causes log.Fatal.
func TestLookupCmdInvalidSearchSkipsInput(t *testing.T) {
	tests := []string{"", "  ", "abc", "1.2.3", "1.2.3.4/", "::g"}
	for _, search := range tests {
		t.Run("search="+search, func(t *testing.T) {
			var err error
			out := captureStdout(t, func() {
				err = runCmd(t, "lookup", "-f", "text", "-u", filepath.Join(t.TempDir(), "missing.txt"), search)
			})
			if err != nil {
				t.Fatalf("Execute error: %v", err)
			}
			if out != "false\n" {
				t.Fatalf("output = %q, want %q", out, "false\n")
			}
		})
	}
}

func TestLookupCmdREPL(t *testing.T) {
	dir, _ := lookupFixture(t)
	const header = "Enter IP or CIDR (type \"exit\" to quit):\n>> "

	tests := []struct {
		name  string
		extra []string
		input string
		want  string
	}{
		{
			name:  "mixed queries then exit",
			input: "\n1.0.0.1\nabc\n8.8.8.8\nexit\n9.9.9.9\n",
			want:  header + "\n>> " + "cn,us\n\n>> " + "false\n\n>> " + "us\n\n>> ",
		},
		{
			name:  "quoted exit, case-insensitive",
			input: "2001:db8::1\n\"EXIT\"\n1.0.0.1\n",
			want:  header + "cn\n\n>> ",
		},
		{
			name:  "EOF without exit",
			input: "  1.0.0.0/24  \n9.9.9.9",
			want:  header + "cn,us\n\n>> " + "false\n\n>> ",
		},
		{
			name:  "immediate EOF",
			input: "",
			want:  header,
		},
		{
			name:  "searchlist applies in REPL",
			extra: []string{"-l", "us"},
			input: "1.0.0.1\n2001:db8::1\nexit\n",
			want:  header + "us\n\n>> " + "false\n\n>> ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withStdin(t, tt.input)
			args := append([]string{"lookup", "-f", "text", "-d", dir}, tt.extra...)
			var err error
			out := captureStdout(t, func() { err = runCmd(t, args...) })
			if err != nil {
				t.Fatalf("Execute error: %v", err)
			}
			if out != tt.want {
				t.Fatalf("output = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestLookupCmdFlagErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"missing format", []string{"lookup", "-u", "x", "1.1.1.1"}, `required flag(s) "format" not set`},
		{"missing uri and dir", []string{"lookup", "-f", "text", "1.1.1.1"}, "at least one of the flags in the group [uri dir] is required"},
		{"uri and dir together", []string{"lookup", "-f", "text", "-u", "x", "-d", "y", "1.1.1.1"}, "none of the others can be"},
		{"too many args", []string{"lookup", "-f", "text", "-u", "x", "1.1.1.1", "2.2.2.2"}, "accepts between 0 and 1 arg(s)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runCmd(t, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
	// NOTE: unsupported formats, "dir" with single-file formats, empty uri with
	// single-file formats, and unreadable inputs all hit log.Fatal(f) and
	// are therefore not exercised.
}

func TestIsValidIPOrCIDR(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"1.2.3.4", true},
		{"255.255.255.255", true},
		{"1.2.3.256", false},
		{"1.2.3", false},
		{"01.2.3.4", false},
		{"1.2.3.4/0", true},
		{"1.2.3.4/32", true},
		{"1.2.3.4/33", false},
		{"1.2.3.4/", false},
		{"/24", false},
		{"::", true},
		{"2001:db8::1", true},
		{"2001:db8::/32", true},
		{"2001:db8::/129", false},
		{"::ffff:1.2.3.4", true},
		{"::ffff:1.2.3.4/120", true},
		{"fe80::1%eth0", true},
		{" 1.2.3.4", false}, // caller is responsible for trimming
		{"abc", false},
		{"1.2.3.4/24/1", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := isValidIPOrCIDR(tt.in); got != tt.want {
				t.Errorf("isValidIPOrCIDR(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// TestIsValidIPOrCIDRAcceptsLookupRejectedCIDR documents a bug: an
// IPv4-mapped IPv6 CIDR with fewer than 96 bits passes isValidIPOrCIDR but
// is rejected by Container.Lookup with ErrInvalidCIDR. In lookupCmd this
// makes instance.Run / RunOutput return an error, which is handled by
// log.Fatal, so `geoip lookup ... ::ffff:1.2.3.4/64` crashes the process
// (and terminates the REPL) instead of printing "false".
func TestIsValidIPOrCIDRAcceptsLookupRejectedCIDR(t *testing.T) {
	tests := []string{"::ffff:1.2.3.4/64", "::ffff:1.2.3.4/0", "::ffff:1.2.3.4/95"}
	for _, search := range tests {
		t.Run(search, func(t *testing.T) {
			if !isValidIPOrCIDR(search) {
				t.Fatalf("isValidIPOrCIDR(%q) = false; bug no longer reproduces", search)
			}
			_, _, err := lib.NewContainer().Lookup(search)
			if !errors.Is(err, lib.ErrInvalidCIDR) {
				t.Fatalf("Lookup(%q) err = %v, want ErrInvalidCIDR", search, err)
			}
		})
	}
}

func TestSupportedInputFormatMaps(t *testing.T) {
	wantSupported := []string{
		v2ray.TypeGeoIPDatIn, mihomo.TypeMRSIn, singbox.TypeSRSIn,
		maxmind.TypeGeoLite2CountryMMDBIn, maxmind.TypeIPInfoCountryMMDBIn, maxmind.TypeDBIPCountryMMDBIn,
		plaintext.TypeClashRuleSetIPCIDRIn, plaintext.TypeClashRuleSetClassicalIn,
		plaintext.TypeSurgeRuleSetIn, plaintext.TypeTextIn,
	}
	wantNoDir := []string{
		v2ray.TypeGeoIPDatIn, maxmind.TypeGeoLite2CountryMMDBIn,
		maxmind.TypeIPInfoCountryMMDBIn, maxmind.TypeDBIPCountryMMDBIn,
	}

	t.Run("supported", func(t *testing.T) {
		if len(supportedInputFormats) != len(wantSupported) {
			t.Fatalf("len = %d, want %d", len(supportedInputFormats), len(wantSupported))
		}
		for _, f := range wantSupported {
			if !supportedInputFormats[strings.ToLower(f)] {
				t.Errorf("%s not supported", f)
			}
		}
	})
	t.Run("disallowed dir", func(t *testing.T) {
		if len(supportedInputFormatsDisallowedDirFlag) != len(wantNoDir) {
			t.Fatalf("len = %d, want %d", len(supportedInputFormatsDisallowedDirFlag), len(wantNoDir))
		}
		for _, f := range wantNoDir {
			k := strings.ToLower(f)
			if !supportedInputFormatsDisallowedDirFlag[k] {
				t.Errorf("%s should disallow dir", f)
			}
			if !supportedInputFormats[k] {
				t.Errorf("%s disallows dir but is not supported", f)
			}
		}
	})
	t.Run("keys are lowercase", func(t *testing.T) {
		for k := range supportedInputFormats {
			if k != strings.ToLower(k) {
				t.Errorf("key %q not lowercase", k)
			}
		}
	})
}

func TestGetInputForLookup(t *testing.T) {
	tests := []struct {
		format   string
		name     string
		uri      string
		dir      string
		wantType string
		// fields expected to be present with these values
		fields map[string]string
	}{
		{maxmind.TypeGeoLite2CountryMMDBIn, "", "/x.mmdb", "", maxmind.TypeGeoLite2CountryMMDBIn, map[string]string{"URI": "/x.mmdb"}},
		{strings.ToUpper(maxmind.TypeDBIPCountryMMDBIn), "", "/x.mmdb", "", maxmind.TypeDBIPCountryMMDBIn, map[string]string{"URI": "/x.mmdb"}},
		{maxmind.TypeIPInfoCountryMMDBIn, "", "/x.mmdb", "", maxmind.TypeIPInfoCountryMMDBIn, map[string]string{"URI": "/x.mmdb"}},
		{v2ray.TypeGeoIPDatIn, "", "/x.dat", "", v2ray.TypeGeoIPDatIn, map[string]string{"URI": "/x.dat"}},
		{mihomo.TypeMRSIn, "true", "/x.mrs", "", mihomo.TypeMRSIn, map[string]string{"Name": "true", "URI": "/x.mrs", "InputDir": ""}},
		{mihomo.TypeMRSIn, "", "", "/d", mihomo.TypeMRSIn, map[string]string{"Name": "", "URI": "", "InputDir": "/d"}},
		{singbox.TypeSRSIn, "true", "/x.srs", "", singbox.TypeSRSIn, map[string]string{"Name": "true", "URI": "/x.srs"}},
		{singbox.TypeSRSIn, "", "", "/d", singbox.TypeSRSIn, map[string]string{"InputDir": "/d"}},
		{plaintext.TypeTextIn, "true", "/x.txt", "", plaintext.TypeTextIn, map[string]string{"Name": "true", "URI": "/x.txt"}},
		{plaintext.TypeTextIn, "", "", "/d", plaintext.TypeTextIn, map[string]string{"InputDir": "/d"}},
		{plaintext.TypeClashRuleSetIPCIDRIn, "true", "/x.yaml", "", plaintext.TypeClashRuleSetIPCIDRIn, map[string]string{"URI": "/x.yaml"}},
		{plaintext.TypeClashRuleSetClassicalIn, "", "", "/d", plaintext.TypeClashRuleSetClassicalIn, map[string]string{"InputDir": "/d"}},
		{plaintext.TypeSurgeRuleSetIn, "true", "/x.list", "", plaintext.TypeSurgeRuleSetIn, map[string]string{"Name": "true", "URI": "/x.list"}},
	}
	for _, tt := range tests {
		t.Run(tt.format+"_"+tt.uri+tt.dir, func(t *testing.T) {
			in := getInputForLookup(tt.format, tt.name, tt.uri, tt.dir)
			if in == nil {
				t.Fatal("nil input")
			}
			if got := in.GetType(); got != tt.wantType {
				t.Errorf("type = %q, want %q", got, tt.wantType)
			}
			if got := in.GetAction(); got != lib.ActionAdd {
				t.Errorf("action = %q, want %q", got, lib.ActionAdd)
			}
			for k, v := range tt.fields {
				if got := fieldString(t, in, k); got != v {
					t.Errorf("%s = %q, want %q", k, got, v)
				}
			}
		})
	}
	// NOTE: unknown formats and constructor errors reach log.Fatal; not tested.
}

func TestGetOutputForLookup(t *testing.T) {
	tests := []struct {
		name       string
		search     string
		searchList []string
		wantSearch string
		wantList   []string
	}{
		{"no list", "1.1.1.1", nil, "1.1.1.1", []string{}},
		{"trimmed", "  1.1.1.0/24 ", nil, "1.1.1.0/24", []string{}},
		{"with list", "2001:db8::1", []string{"cn", "US"}, "2001:db8::1", []string{"cn", "US"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := getOutputForLookup(tt.search, tt.searchList...)
			if out == nil {
				t.Fatal("nil output")
			}
			if got := out.GetType(); got != special.TypeLookup {
				t.Errorf("type = %q, want %q", got, special.TypeLookup)
			}
			if got := out.GetAction(); got != lib.ActionOutput {
				t.Errorf("action = %q, want %q", got, lib.ActionOutput)
			}
			if got := fieldString(t, out, "Search"); got != tt.wantSearch {
				t.Errorf("Search = %q, want %q", got, tt.wantSearch)
			}
			got := fieldStrings(t, out, "SearchList")
			if len(got) != len(tt.wantList) {
				t.Fatalf("SearchList len = %d (%v), want %d (%v)", len(got), got, len(tt.wantList), tt.wantList)
			}
			for i := range got {
				if got[i] != tt.wantList[i] {
					t.Errorf("SearchList[%d] = %q, want %q", i, got[i], tt.wantList[i])
				}
			}
		})
	}
	// NOTE: an empty search makes special.NewLookup fail -> log.Fatal; not tested.
}
