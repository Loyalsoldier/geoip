package main

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
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

func TestIsValidIPOrCIDR(t *testing.T) {
	tests := []struct {
		name   string
		search string
		want   bool
	}{
		{name: "empty", search: "", want: false},
		{name: "IPv4", search: "1.1.1.1", want: true},
		{name: "IPv4 zero", search: "0.0.0.0", want: true},
		{name: "IPv6", search: "2001:db8::1", want: true},
		{name: "IPv6 loopback", search: "::1", want: true},
		{name: "IPv4-mapped IPv6", search: "::ffff:1.2.3.4", want: true},
		{name: "IPv4 CIDR", search: "1.0.0.0/24", want: true},
		{name: "IPv4 CIDR with host bits", search: "1.0.0.1/24", want: true},
		{name: "IPv4 CIDR /0", search: "0.0.0.0/0", want: true},
		{name: "IPv4 CIDR /32", search: "1.1.1.1/32", want: true},
		{name: "IPv6 CIDR", search: "2001:db8::/32", want: true},
		{name: "IPv6 CIDR /128", search: "::1/128", want: true},
		{name: "IPv4-mapped IPv6 CIDR", search: "::ffff:1.2.3.0/120", want: true},
		{name: "IPv4-mapped IPv6 CIDR shorter than 96 bits", search: "::ffff:1.2.3.0/95", want: true},
		{name: "IPv4 CIDR prefix too long", search: "1.1.1.1/33", want: false},
		{name: "IPv6 CIDR prefix too long", search: "2001:db8::/129", want: false},
		{name: "negative prefix", search: "1.1.1.1/-1", want: false},
		{name: "empty prefix", search: "1.1.1.1/", want: false},
		{name: "non-numeric prefix", search: "1.1.1.1/abc", want: false},
		{name: "missing address", search: "/24", want: false},
		{name: "invalid IPv4 octet", search: "256.1.1.1", want: false},
		{name: "too few IPv4 octets", search: "1.1.1", want: false},
		{name: "invalid IPv6", search: "2001:db8:::1", want: false},
		{name: "garbage", search: "not-an-ip", want: false},
		{name: "surrounding spaces", search: " 1.1.1.1 ", want: false},
		{name: "IPv6 zone", search: "fe80::1%eth0", want: true},
		{name: "IPv6 zone in CIDR", search: "fe80::1%eth0/64", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidIPOrCIDR(tt.search); got != tt.want {
				t.Fatalf("isValidIPOrCIDR(%q) = %v, want %v", tt.search, got, tt.want)
			}
		})
	}
}

func TestSupportedInputFormats(t *testing.T) {
	wantSupported := map[string]bool{
		"v2raygeoipdat":         true,
		"mihomomrs":             true,
		"singboxsrs":            true,
		"maxmindmmdb":           true,
		"ipinfocountrymmdb":     true,
		"dbipcountrymmdb":       true,
		"clashruleset":          true,
		"clashrulesetclassical": true,
		"surgeruleset":          true,
		"text":                  true,
	}
	wantDisallowedDir := map[string]bool{
		"v2raygeoipdat":     true,
		"maxmindmmdb":       true,
		"ipinfocountrymmdb": true,
		"dbipcountrymmdb":   true,
	}

	if len(supportedInputFormats) != len(wantSupported) {
		t.Fatalf("len(supportedInputFormats) = %d, want %d: %v", len(supportedInputFormats), len(wantSupported), supportedInputFormats)
	}
	if !maps.Equal(supportedInputFormats, wantSupported) {
		t.Fatalf("supportedInputFormats = %v, want %v", supportedInputFormats, wantSupported)
	}

	if len(supportedInputFormatsDisallowedDirFlag) != len(wantDisallowedDir) {
		t.Fatalf("len(supportedInputFormatsDisallowedDirFlag) = %d, want %d: %v", len(supportedInputFormatsDisallowedDirFlag), len(wantDisallowedDir), supportedInputFormatsDisallowedDirFlag)
	}
	if !maps.Equal(supportedInputFormatsDisallowedDirFlag, wantDisallowedDir) {
		t.Fatalf("supportedInputFormatsDisallowedDirFlag = %v, want %v", supportedInputFormatsDisallowedDirFlag, wantDisallowedDir)
	}

	for format := range supportedInputFormatsDisallowedDirFlag {
		if !supportedInputFormats[format] {
			t.Errorf("format %q disallows the dir flag but is not supported", format)
		}
	}

	// Every supported format must be accepted by getInputForLookup.
	for _, format := range slices.Sorted(maps.Keys(supportedInputFormats)) {
		t.Run(format, func(t *testing.T) {
			input := getInputForLookup(format, "true", "dummy-uri", "")
			if got := strings.ToLower(input.GetType()); got != format {
				t.Fatalf("getInputForLookup(%q).GetType() = %q", format, input.GetType())
			}
		})
	}
}

func TestGetInputForLookup(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		nameArg  string
		uri      string
		dir      string
		wantType string
		wantDesc string
	}{
		{name: "maxmindMMDB with default uri", format: "maxmindMMDB", wantType: maxmind.TypeGeoLite2CountryMMDBIn, wantDesc: maxmind.DescGeoLite2CountryMMDBIn},
		{name: "maxmindMMDB with uri", format: "maxmindmmdb", nameArg: "true", uri: "country.mmdb", wantType: maxmind.TypeGeoLite2CountryMMDBIn, wantDesc: maxmind.DescGeoLite2CountryMMDBIn},
		{name: "dbipCountryMMDB", format: "DBIPCOUNTRYMMDB", uri: "dbip.mmdb", wantType: maxmind.TypeDBIPCountryMMDBIn, wantDesc: maxmind.DescDBIPCountryMMDBIn},
		{name: "ipinfoCountryMMDB", format: "ipinfoCountryMMDB", wantType: maxmind.TypeIPInfoCountryMMDBIn, wantDesc: maxmind.DescIPInfoCountryMMDBIn},
		{name: "mihomoMRS with uri", format: "mihomoMRS", nameArg: "true", uri: "cn.mrs", wantType: mihomo.TypeMRSIn, wantDesc: mihomo.DescMRSIn},
		{name: "mihomoMRS with dir", format: "mihomomrs", dir: "mrs", wantType: mihomo.TypeMRSIn, wantDesc: mihomo.DescMRSIn},
		{name: "singboxSRS with uri", format: "singboxSRS", nameArg: "true", uri: "cn.srs", wantType: singbox.TypeSRSIn, wantDesc: singbox.DescSRSIn},
		{name: "singboxSRS with dir", format: "SingboxSrs", dir: "srs", wantType: singbox.TypeSRSIn, wantDesc: singbox.DescSRSIn},
		{name: "v2rayGeoIPDat", format: "v2rayGeoIPDat", nameArg: "true", uri: "geoip.dat", wantType: v2ray.TypeGeoIPDatIn, wantDesc: v2ray.DescGeoIPDatIn},
		{name: "text with uri", format: "text", nameArg: "true", uri: "cn.txt", wantType: plaintext.TypeTextIn, wantDesc: plaintext.DescTextIn},
		{name: "text with dir", format: "TEXT", dir: "text", wantType: plaintext.TypeTextIn, wantDesc: plaintext.DescTextIn},
		{name: "clashRuleSet with uri", format: "clashRuleSet", nameArg: "true", uri: "cn.yaml", wantType: plaintext.TypeClashRuleSetIPCIDRIn, wantDesc: plaintext.DescClashRuleSetIPCIDRIn},
		{name: "clashRuleSet with dir", format: "clashruleset", dir: "clash", wantType: plaintext.TypeClashRuleSetIPCIDRIn, wantDesc: plaintext.DescClashRuleSetIPCIDRIn},
		{name: "clashRuleSetClassical", format: "clashRuleSetClassical", nameArg: "true", uri: "cn.yaml", wantType: plaintext.TypeClashRuleSetClassicalIn, wantDesc: plaintext.DescClashRuleSetClassicalIn},
		{name: "surgeRuleSet", format: "surgeRuleSet", dir: "surge", wantType: plaintext.TypeSurgeRuleSetIn, wantDesc: plaintext.DescSurgeRuleSetIn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := getInputForLookup(tt.format, tt.nameArg, tt.uri, tt.dir)
			if input == nil {
				t.Fatalf("getInputForLookup() = nil")
			}
			if input.GetType() != tt.wantType {
				t.Errorf("GetType() = %q, want %q", input.GetType(), tt.wantType)
			}
			if input.GetAction() != lib.ActionAdd {
				t.Errorf("GetAction() = %q, want %q", input.GetAction(), lib.ActionAdd)
			}
			if input.GetDescription() != tt.wantDesc {
				t.Errorf("GetDescription() = %q, want %q", input.GetDescription(), tt.wantDesc)
			}
		})
	}
}

func TestGetInputForLookupReadsData(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "cn.txt", "1.0.1.0/24\n")
	writeFile(t, dir, "us.txt", "1.0.0.0/16\n")
	uri := writeFile(t, t.TempDir(), "list.txt", "2001:db8::/32\n")

	tests := []struct {
		name      string
		nameArg   string
		uri       string
		dir       string
		wantNames []string
	}{
		{name: "uri", nameArg: "true", uri: uri, wantNames: []string{"TRUE"}},
		{name: "dir", dir: dir, wantNames: []string{"CN", "US"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := getInputForLookup("text", tt.nameArg, tt.uri, tt.dir)
			container, err := input.Input(lib.NewContainer())
			if err != nil {
				t.Fatalf("Input() error = %v", err)
			}
			var names []string
			for entry := range container.Loop() {
				names = append(names, entry.GetName())
			}
			slices.Sort(names)
			if len(names) != len(tt.wantNames) {
				t.Fatalf("entry names = %q, want %q", names, tt.wantNames)
			}
			for i := range tt.wantNames {
				if names[i] != tt.wantNames[i] {
					t.Errorf("entry names[%d] = %q, want %q", i, names[i], tt.wantNames[i])
				}
			}
		})
	}
}

func TestGetOutputForLookup(t *testing.T) {
	tests := []struct {
		name       string
		search     string
		searchList []string
		wantOut    string
	}{
		{name: "IP without search list", search: "1.0.1.1", wantOut: "cn,us\n"},
		{name: "IP with search list", search: "1.0.1.1", searchList: []string{"us"}, wantOut: "us\n"},
		{name: "CIDR with multiple search lists", search: "1.0.1.0/24", searchList: []string{"cn", "jp"}, wantOut: "cn\n"},
		{name: "IP not found", search: "8.8.8.8", wantOut: "false\n"},
		{name: "search is trimmed", search: "  1.0.2.1  ", wantOut: "us\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := getOutputForLookup(tt.search, tt.searchList...)
			if output == nil {
				t.Fatalf("getOutputForLookup() = nil")
			}
			if output.GetType() != special.TypeLookup {
				t.Errorf("GetType() = %q, want %q", output.GetType(), special.TypeLookup)
			}
			if output.GetAction() != lib.ActionOutput {
				t.Errorf("GetAction() = %q, want %q", output.GetAction(), lib.ActionOutput)
			}
			if output.GetDescription() != special.DescLookup {
				t.Errorf("GetDescription() = %q, want %q", output.GetDescription(), special.DescLookup)
			}

			container := lib.NewContainer()
			for name, cidr := range map[string]string{"cn": "1.0.1.0/24", "us": "1.0.0.0/16"} {
				entry := lib.NewEntry(name)
				if err := entry.AddPrefix(cidr); err != nil {
					t.Fatalf("AddPrefix(%q) error = %v", cidr, err)
				}
				if err := container.Add(entry); err != nil {
					t.Fatalf("container.Add() error = %v", err)
				}
			}

			var outErr error
			stdout := captureStdout(t, func() {
				outErr = output.Output(container)
			})
			if outErr != nil {
				t.Fatalf("Output() error = %v", outErr)
			}
			if stdout != tt.wantOut {
				t.Fatalf("stdout = %q, want %q", stdout, tt.wantOut)
			}
		})
	}
}

// lookupTestData writes a list directory and single list files for the
// lookup command tests.
type lookupTestData struct {
	dir        string // cn.txt, us.txt, jp.txt
	textFile   string // plaintext list
	clashFile  string // clash ipcidr rule set
	classFile  string // clash classical rule set
	surgeFile  string // surge rule set
	missingURI string // does not exist
}

func newLookupTestData(t *testing.T) lookupTestData {
	t.Helper()

	dir := filepath.Join(t.TempDir(), "lists")
	files := t.TempDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	writeFile(t, dir, "cn.txt", "1.0.1.0/24\n2001:250::/35\n")
	writeFile(t, dir, "us.txt", "1.0.0.0/16\n2600::/16\n")
	writeFile(t, dir, "jp.txt", "1.0.16.0/20\n")

	return lookupTestData{
		dir:        dir,
		textFile:   writeFile(t, files, "list.txt", "# comment\n1.1.1.0/24\n2001:db8::/32\n"),
		clashFile:  writeFile(t, files, "clash.yaml", "payload:\n  - '1.1.1.0/24'\n  - '2001:db8::/32'\n"),
		classFile:  writeFile(t, files, "classical.yaml", "payload:\n  - IP-CIDR,1.1.1.0/24,no-resolve\n  - DOMAIN,example.com\n"),
		surgeFile:  writeFile(t, files, "surge.list", "IP-CIDR,1.1.1.0/24\nIP-CIDR6,2001:db8::/32\n"),
		missingURI: filepath.Join(files, "missing.txt"),
	}
}

func TestLookupCmdOnce(t *testing.T) {
	data := newLookupTestData(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/list.txt" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("1.1.1.0/24\n2001:db8::/32\n"))
	}))
	t.Cleanup(srv.Close)

	tests := []struct {
		name    string
		args    []string
		wantOut string
	}{
		{name: "uri IP found", args: []string{"lookup", "-f", "text", "-u", data.textFile, "1.1.1.1"}, wantOut: "true\n"},
		{name: "uri IP not found", args: []string{"lookup", "-f", "text", "-u", data.textFile, "8.8.8.8"}, wantOut: "false\n"},
		{name: "uri CIDR found", args: []string{"lookup", "-f", "text", "-u", data.textFile, "1.1.1.128/25"}, wantOut: "true\n"},
		{name: "uri CIDR not found", args: []string{"lookup", "-f", "text", "-u", data.textFile, "1.1.0.0/23"}, wantOut: "false\n"},
		{name: "uri IPv6 found with uppercase and spaces", args: []string{"lookup", "-f", "text", "-u", data.textFile, " 2001:DB8::1 "}, wantOut: "true\n"},
		{name: "uri IPv4-mapped IPv6", args: []string{"lookup", "-f", "text", "-u", data.textFile, "::ffff:1.1.1.1"}, wantOut: "true\n"},
		{name: "format is trimmed and case insensitive", args: []string{"lookup", "--format", " TEXT ", "--uri", data.textFile, "1.1.1.1"}, wantOut: "true\n"},
		{name: "alias find", args: []string{"find", "-f", "text", "-u", data.textFile, "1.1.1.1"}, wantOut: "true\n"},
		{name: "remote uri over HTTP", args: []string{"lookup", "-f", "text", "-u", srv.URL + "/list.txt", "2001:db8::1"}, wantOut: "true\n"},
		{name: "clashRuleSet", args: []string{"lookup", "-f", "clashRuleSet", "-u", data.clashFile, "1.1.1.1"}, wantOut: "true\n"},
		{name: "clashRuleSetClassical found", args: []string{"lookup", "-f", "clashRuleSetClassical", "-u", data.classFile, "1.1.1.1"}, wantOut: "true\n"},
		{name: "clashRuleSetClassical not found", args: []string{"lookup", "-f", "clashRuleSetClassical", "-u", data.classFile, "1.1.2.1"}, wantOut: "false\n"},
		{name: "surgeRuleSet", args: []string{"lookup", "-f", "surgeRuleSet", "-u", data.surgeFile, "2001:db8::/48"}, wantOut: "true\n"},
		{name: "dir IP in multiple lists", args: []string{"lookup", "-f", "text", "-d", data.dir, "1.0.1.1"}, wantOut: "cn,us\n"},
		{name: "dir IP in overlapping lists", args: []string{"lookup", "-f", "text", "-d", data.dir, "1.0.16.1"}, wantOut: "jp,us\n"},
		{name: "dir IPv6", args: []string{"lookup", "-f", "text", "--dir", data.dir, "2600::1"}, wantOut: "us\n"},
		{name: "dir not found", args: []string{"lookup", "-f", "text", "-d", data.dir, "8.8.8.8"}, wantOut: "false\n"},
		{name: "dir with search list", args: []string{"lookup", "-f", "text", "-d", data.dir, "-l", "cn", "1.0.1.1"}, wantOut: "cn\n"},
		{name: "dir with comma separated search list", args: []string{"lookup", "-f", "text", "-d", data.dir, "--searchlist=jp,us", "1.0.1.1"}, wantOut: "us\n"},
		{name: "dir with repeated search list flags", args: []string{"lookup", "-f", "text", "-d", data.dir, "-l", "jp", "-l", "cn", "1.0.1.0/24"}, wantOut: "cn\n"},
		{name: "dir with search list without match", args: []string{"lookup", "-f", "text", "-d", data.dir, "-l", "jp", "1.0.1.1"}, wantOut: "false\n"},
		// Invalid search args print "false" before any input is read, so a
		// missing input file does not matter.
		{name: "invalid IP prints false", args: []string{"lookup", "-f", "text", "-u", data.missingURI, "not-an-ip"}, wantOut: "false\n"},
		{name: "invalid CIDR prints false", args: []string{"lookup", "-f", "text", "-u", data.missingURI, "1.1.1.1/33"}, wantOut: "false\n"},
		{name: "blank search prints false", args: []string{"lookup", "-f", "text", "-u", data.missingURI, "  "}, wantOut: "false\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, cobraOutput, err := executeCommand(t, tt.args...)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if cobraOutput != "" {
				t.Errorf("cobra output = %q, want empty", cobraOutput)
			}
			if stdout != tt.wantOut {
				t.Fatalf("stdout = %q, want %q", stdout, tt.wantOut)
			}
		})
	}
}

func TestLookupCmdREPL(t *testing.T) {
	data := newLookupTestData(t)
	const prompt = "Enter IP or CIDR (type \"exit\" to quit):\n>> "

	tests := []struct {
		name    string
		args    []string
		input   string
		wantOut string
	}{
		{
			name:  "blank, invalid, found, not found and exit",
			args:  []string{"lookup", "-f", "text", "-u", data.textFile},
			input: "\n   \ninvalid\n1.1.1.1\n1.1.1.0/24\n8.8.8.8\n2001:DB8::1\nexit\n1.1.1.1\n",
			wantOut: prompt +
				"\n>> " + // blank line
				"\n>> " + // whitespace-only line
				"false\n\n>> " + // invalid
				"true\n\n>> " + // 1.1.1.1
				"true\n\n>> " + // 1.1.1.0/24
				"false\n\n>> " + // 8.8.8.8
				"true\n\n>> ", // 2001:DB8::1, then exit
		},
		{
			name:    "quoted exit",
			args:    []string{"lookup", "-f", "text", "-u", data.textFile},
			input:   "1.1.1.1\n\"exit\"\n8.8.8.8\n",
			wantOut: prompt + "true\n\n>> ",
		},
		{
			name:    "uppercase exit with spaces",
			args:    []string{"lookup", "-f", "text", "-u", data.textFile},
			input:   "  EXIT  \n1.1.1.1\n",
			wantOut: prompt,
		},
		{
			name:    "end of input without exit",
			args:    []string{"lookup", "-f", "text", "-u", data.textFile},
			input:   "1.1.1.1\n8.8.8.8",
			wantOut: prompt + "true\n\n>> false\n\n>> ",
		},
		{
			name:    "empty input",
			args:    []string{"find", "-f", "text", "-u", data.textFile},
			input:   "",
			wantOut: prompt,
		},
		{
			name:    "dir with search list",
			args:    []string{"lookup", "-f", "text", "-d", data.dir, "-l", "cn,jp"},
			input:   "1.0.1.1\n1.0.16.1\n1.0.2.1\n2001:250::1\nexit\n",
			wantOut: prompt + "cn\n\n>> jp\n\n>> false\n\n>> cn\n\n>> ",
		},
		{
			name:    "dir without search list",
			args:    []string{"lookup", "-f", "text", "-d", data.dir},
			input:   "1.0.1.1\n::ffff:1.0.16.0/120\nexit\n",
			wantOut: prompt + "cn,us\n\n>> jp,us\n\n>> ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setStdin(t, tt.input)

			stdout, cobraOutput, err := executeCommand(t, tt.args...)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if cobraOutput != "" {
				t.Errorf("cobra output = %q, want empty", cobraOutput)
			}
			assertLines(t, stdout, tt.wantOut)
		})
	}
}

func TestLookupCmdArgsErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "missing format", args: []string{"lookup", "-u", "x", "1.1.1.1"}, wantErr: `required flag(s) "format" not set`},
		{name: "missing uri and dir", args: []string{"lookup", "-f", "text", "1.1.1.1"}, wantErr: "at least one of the flags in the group [uri dir] is required"},
		{name: "both uri and dir", args: []string{"lookup", "-f", "text", "-u", "x", "-d", "y", "1.1.1.1"}, wantErr: "if any flags in the group [uri dir] are set none of the others can be; [dir uri] were all set"},
		{name: "too many args", args: []string{"lookup", "-f", "text", "-u", "x", "1.1.1.1", "2.2.2.2"}, wantErr: "accepts between 0 and 1 arg(s), received 2"},
		{name: "unknown flag", args: []string{"lookup", "--bogus"}, wantErr: "unknown flag: --bogus"},
		{name: "flag without value", args: []string{"lookup", "-f"}, wantErr: "flag needs an argument: 'f' in -f"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, cobraOutput, err := executeCommand(t, tt.args...)
			if err == nil {
				t.Fatalf("Execute() error = nil, want %q", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("Execute() error = %q, want %q", err.Error(), tt.wantErr)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if !strings.Contains(cobraOutput, "Error: "+tt.wantErr) {
				t.Errorf("cobra output = %q, want it to contain the error", cobraOutput)
			}
		})
	}
}

func TestLookupCmdFatal(t *testing.T) {
	if args, ok := subprocessArgs(t); ok {
		rootCmd.SetArgs(args)
		_ = rootCmd.Execute()
		return
	}

	data := newLookupTestData(t)

	tests := []struct {
		name    string
		args    []string
		stdin   string
		wantMsg string
	}{
		{name: "unsupported format", args: []string{"lookup", "-f", "json", "-u", data.textFile, "1.1.1.1"}, wantMsg: "unsupported input format"},
		{name: "empty format", args: []string{"lookup", "-f", "", "-u", data.textFile, "1.1.1.1"}, wantMsg: "unsupported input format"},
		{name: "dir with single-file format", args: []string{"lookup", "-f", "v2rayGeoIPDat", "-d", data.dir, "1.1.1.1"}, wantMsg: `format v2raygeoipdat does not support "dir" flag, please use "uri" flag instead`},
		{name: "dir with mmdb format", args: []string{"lookup", "-f", "maxmindMMDB", "-d", data.dir}, wantMsg: `format maxmindmmdb does not support "dir" flag, please use "uri" flag instead`},
		{name: "empty uri with single-file format", args: []string{"lookup", "-f", "dbipCountryMMDB", "-u", "", "1.1.1.1"}, wantMsg: `format dbipcountrymmdb requires a non-empty "uri" flag`},
		{name: "empty uri and dir for text", args: []string{"lookup", "-f", "text", "-d", "", "1.1.1.1"}, wantMsg: "missing inputDir or name"},
		{name: "once mode input file missing", args: []string{"lookup", "-f", "text", "-u", data.missingURI, "1.1.1.1"}, wantMsg: missingFileError(t, data.missingURI)},
		{name: "once mode invalid input file", args: []string{"lookup", "-f", "v2rayGeoIPDat", "-u", data.textFile, "1.1.1.1"}, wantMsg: "[type v2rayGeoIPDat | action add]"},
		{name: "once mode lookup error", args: []string{"lookup", "-f", "text", "-u", data.textFile, "::ffff:1.1.1.0/95"}, wantMsg: "invalid CIDR"},
		{name: "REPL input file missing", args: []string{"lookup", "-f", "text", "-u", data.missingURI}, stdin: "1.1.1.1\n", wantMsg: missingFileError(t, data.missingURI)},
		{name: "REPL lookup error", args: []string{"lookup", "-f", "text", "-u", data.textFile}, stdin: "1.1.1.1\n::ffff:1.1.1.0/95\n", wantMsg: "invalid CIDR"},
		{name: "REPL line too long", args: []string{"lookup", "-f", "text", "-u", data.textFile}, stdin: strings.Repeat("1", 70*1024) + "\n", wantMsg: "token too long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stderr := runSubprocess(t, "TestLookupCmdFatal", tt.args, tt.stdin)
			assertFatal(t, code, stderr, tt.wantMsg)
		})
	}
}

func TestGetInputForLookupFatal(t *testing.T) {
	if args, ok := subprocessArgs(t); ok {
		getInputForLookup(args[0], args[1], args[2], args[3])
		return
	}

	tests := []struct {
		name    string
		args    []string // format, name, uri, dir
		wantMsg string
	}{
		{name: "unsupported format", args: []string{"bogus", "true", "x", ""}, wantMsg: "unsupported input format"},
		{name: "unsupported json format", args: []string{"json", "true", "x", ""}, wantMsg: "unsupported input format"},
		{name: "v2ray without uri", args: []string{"v2rayGeoIPDat", "", "", ""}, wantMsg: "uri must be specified"},
		{name: "mihomo name without uri", args: []string{"mihomoMRS", "true", "", ""}, wantMsg: "name and uri must be specified together"},
		{name: "singbox without anything", args: []string{"singboxSRS", "", "", ""}, wantMsg: "missing name or uri or inputDir"},
		{name: "text without anything", args: []string{"text", "", "", ""}, wantMsg: "missing inputDir or name"},
		{name: "clashRuleSet with dir and uri", args: []string{"clashRuleSet", "true", "x", "dir"}, wantMsg: "inputDir is not allowed to be used with name or uri or ipOrCIDR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stderr := runSubprocess(t, "TestGetInputForLookupFatal", tt.args, "")
			assertFatal(t, code, stderr, tt.wantMsg)
		})
	}
}

func TestGetOutputForLookupFatal(t *testing.T) {
	if args, ok := subprocessArgs(t); ok {
		getOutputForLookup(args[0], args[1:]...)
		return
	}

	tests := []struct {
		name string
		args []string // search, searchList...
	}{
		{name: "empty search", args: []string{""}},
		{name: "blank search with search list", args: []string{"   ", "cn"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stderr := runSubprocess(t, "TestGetOutputForLookupFatal", tt.args, "")
			assertFatal(t, code, stderr, "search must be specified")
		})
	}
}
