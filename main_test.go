package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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

const (
	subprocessEnv     = "GEOIP_MAIN_TEST_ARGS"
	subprocessFuncEnv = "GEOIP_MAIN_TEST_FUNC"
)

// fatalFuncs are helper calls expected to terminate the process via log.Fatal.
var fatalFuncs = map[string]func(){
	"lookupInputUnsupportedFormat": func() { getInputForLookup("unknown", "", "", "") },
	"lookupInputInvalidArgs":       func() { getInputForLookup(plaintext.TypeTextIn, "", "", "") },
	"lookupOutputEmptySearch":      func() { getOutputForLookup("") },
}

// TestMain runs the real main() in a subprocess when subprocessEnv is set,
// which makes it possible to test code paths ending with log.Fatal.
func TestMain(m *testing.M) {
	if name, ok := os.LookupEnv(subprocessFuncEnv); ok {
		fatalFuncs[name]()
		os.Exit(0)
	}
	if raw, ok := os.LookupEnv(subprocessEnv); ok {
		var args []string
		if err := json.Unmarshal([]byte(raw), &args); err != nil {
			panic(err)
		}
		os.Args = append([]string{"geoip"}, args...)
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

type runResult struct {
	stdout   string
	stderr   string
	exitCode int
}

// runMain executes the geoip CLI with args in a subprocess and stdin as input.
func runMain(t *testing.T, stdin string, args ...string) runResult {
	t.Helper()

	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	return runSubprocess(t, stdin, subprocessEnv+"="+string(raw))
}

// runSubprocess re-executes the test binary with extra env.
func runSubprocess(t *testing.T, stdin string, env string) runResult {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), env)
	// Propagate coverage directory so subprocess coverage is merged.
	if f := flag.Lookup("test.gocoverdir"); f != nil && f.Value.String() != "" {
		cmd.Env = append(cmd.Env, "GOCOVERDIR="+f.Value.String())
	}
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	res := runResult{}
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("failed to run subprocess: %v", err)
		}
		res.exitCode = exitErr.ExitCode()
	}
	res.stdout = stdout.String()
	res.stderr = stderr.String()
	return res
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func TestIsValidIPOrCIDR(t *testing.T) {
	tests := []struct {
		name   string
		search string
		want   bool
	}{
		{"empty", "", false},
		{"ipv4", "1.1.1.1", true},
		{"ipv6", "2001:db8::1", true},
		{"ipv4 cidr", "1.0.0.0/24", true},
		{"ipv6 cidr", "2001:db8::/32", true},
		{"invalid ip", "1.1.1.256", false},
		{"invalid cidr prefix", "1.0.0.0/33", false},
		{"invalid cidr addr", "abc/24", false},
		{"text", "hello", false},
		{"with spaces", " 1.1.1.1 ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidIPOrCIDR(tt.search); got != tt.want {
				t.Errorf("isValidIPOrCIDR(%q) = %v, want %v", tt.search, got, tt.want)
			}
		})
	}
}

func TestGetInputForLookup(t *testing.T) {
	dir := t.TempDir()
	uri := writeFile(t, dir, "test.txt", "1.0.0.0/24\n")

	tests := []struct {
		name     string
		format   string
		listName string
		uri      string
		dir      string
		wantType string
	}{
		{"maxmind mmdb", maxmind.TypeGeoLite2CountryMMDBIn, "", "file.mmdb", "", maxmind.TypeGeoLite2CountryMMDBIn},
		{"dbip mmdb", maxmind.TypeDBIPCountryMMDBIn, "", "file.mmdb", "", maxmind.TypeDBIPCountryMMDBIn},
		{"ipinfo mmdb", maxmind.TypeIPInfoCountryMMDBIn, "", "file.mmdb", "", maxmind.TypeIPInfoCountryMMDBIn},
		{"mihomo mrs uri", mihomo.TypeMRSIn, "true", "file.mrs", "", mihomo.TypeMRSIn},
		{"mihomo mrs dir", mihomo.TypeMRSIn, "", "", dir, mihomo.TypeMRSIn},
		{"singbox srs", singbox.TypeSRSIn, "true", "file.srs", "", singbox.TypeSRSIn},
		{"v2ray dat", v2ray.TypeGeoIPDatIn, "", "geoip.dat", "", v2ray.TypeGeoIPDatIn},
		{"text", plaintext.TypeTextIn, "true", uri, "", plaintext.TypeTextIn},
		{"text dir", plaintext.TypeTextIn, "", "", dir, plaintext.TypeTextIn},
		{"clash ipcidr", plaintext.TypeClashRuleSetIPCIDRIn, "true", uri, "", plaintext.TypeClashRuleSetIPCIDRIn},
		{"clash classical", plaintext.TypeClashRuleSetClassicalIn, "true", uri, "", plaintext.TypeClashRuleSetClassicalIn},
		{"surge", plaintext.TypeSurgeRuleSetIn, "true", uri, "", plaintext.TypeSurgeRuleSetIn},
		{"case insensitive", strings.ToUpper(plaintext.TypeTextIn), "true", uri, "", plaintext.TypeTextIn},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := getInputForLookup(tt.format, tt.listName, tt.uri, tt.dir)
			if input == nil {
				t.Fatal("got nil input")
			}
			if got := input.GetType(); got != tt.wantType {
				t.Errorf("GetType() = %q, want %q", got, tt.wantType)
			}
			if got := input.GetAction(); got != lib.ActionAdd {
				t.Errorf("GetAction() = %q, want %q", got, lib.ActionAdd)
			}
		})
	}
}

func TestGetOutputForLookup(t *testing.T) {
	tests := []struct {
		name       string
		search     string
		searchList []string
	}{
		{"no search list", "1.1.1.1", nil},
		{"with search list", "1.0.0.0/24", []string{"cn", "us"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := getOutputForLookup(tt.search, tt.searchList...)
			if output == nil {
				t.Fatal("got nil output")
			}
			if got := output.GetType(); got != special.TypeLookup {
				t.Errorf("GetType() = %q, want %q", got, special.TypeLookup)
			}
			if got := output.GetAction(); got != lib.ActionOutput {
				t.Errorf("GetAction() = %q, want %q", got, lib.ActionOutput)
			}
		})
	}
}

func TestGetInputForMerge(t *testing.T) {
	input := getInputForMerge()
	if got := input.GetType(); got != special.TypeStdin {
		t.Errorf("GetType() = %q, want %q", got, special.TypeStdin)
	}
	if got := input.GetAction(); got != lib.ActionAdd {
		t.Errorf("GetAction() = %q, want %q", got, lib.ActionAdd)
	}
}

func TestGetOutputForMerge(t *testing.T) {
	for _, otype := range []string{"", "ipv4", "ipv6", "other"} {
		t.Run("otype="+otype, func(t *testing.T) {
			output := getOutputForMerge(otype)
			if got := output.GetType(); got != special.TypeStdout {
				t.Errorf("GetType() = %q, want %q", got, special.TypeStdout)
			}
			if got := output.GetAction(); got != lib.ActionOutput {
				t.Errorf("GetAction() = %q, want %q", got, lib.ActionOutput)
			}
		})
	}
}

func TestHelperFatal(t *testing.T) {
	tests := []struct {
		name        string
		wantContain string
	}{
		{"lookupInputUnsupportedFormat", "unsupported input format"},
		{"lookupInputInvalidArgs", "missing inputDir or name"},
		{"lookupOutputEmptySearch", "search must be specified"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := runSubprocess(t, "", subprocessFuncEnv+"="+tt.name)
			if res.exitCode == 0 {
				t.Fatalf("expected non-zero exit code, stderr: %s", res.stderr)
			}
			if !strings.Contains(res.stderr, tt.wantContain) {
				t.Errorf("stderr = %q, want to contain %q", res.stderr, tt.wantContain)
			}
		})
	}
}

func TestRootCommands(t *testing.T) {
	want := []string{"convert", "list", "lookup", "merge"}
	var got []string
	for _, c := range rootCmd.Commands() {
		if slices.Contains(want, c.Name()) {
			got = append(got, c.Name())
		}
	}
	slices.Sort(got)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("registered commands = %v, want %v", got, want)
	}
}

func TestMainCLI(t *testing.T) {
	dir := t.TempDir()
	cnFile := writeFile(t, dir, "cn.txt", "1.0.0.0/24\n2001:db8::/32\n")
	writeFile(t, dir, "us.txt", "8.8.8.0/24\n1.0.0.0/16\n")
	listDir := filepath.Join(dir, "lists")
	if err := os.Mkdir(listDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, listDir, "cn.txt", "1.0.0.0/24\n2001:db8::/32\n")
	writeFile(t, listDir, "us.txt", "8.8.8.0/24\n1.0.0.0/16\n")

	config := map[string]any{
		"input": []any{
			map[string]any{"type": "text", "action": "add", "args": map[string]any{"name": "cn", "uri": cnFile}},
		},
		"output": []any{
			map[string]any{"type": "stdout", "action": "output"},
		},
	}
	raw, _ := json.Marshal(config)
	goodConfig := writeFile(t, dir, "config.json", string(raw))
	badConfig := writeFile(t, dir, "bad.json", "{invalid")
	runErrConfig := writeFile(t, dir, "runerr.json", `{"input":[{"type":"text","action":"add","args":{"name":"x","uri":"`+filepath.Join(dir, "missing.txt")+`"}}],"output":[{"type":"stdout","action":"output"}]}`)
	mmdbFile := writeFile(t, dir, "fake.mmdb", "not a mmdb")

	tests := []struct {
		name        string
		args        []string
		stdin       string
		wantFail    bool
		wantStdout  []string // exact non-empty stdout lines; nil means not checked
		wantContain string   // substring expected in stdout+stderr
	}{
		// root
		{name: "help", args: []string{"--help"}, wantContain: "geoip is a convenient tool"},
		{name: "unknown command", args: []string{"nope"}, wantFail: true, wantContain: "unknown command"},

		// list
		{name: "list", args: []string{"list"}, wantContain: "All available input formats"},

		// convert
		{name: "convert", args: []string{"convert", "-c", goodConfig}, wantStdout: []string{"1.0.0.0/24", "2001:db8::/32"}},
		{name: "convert bad config", args: []string{"convert", "-c", badConfig}, wantFail: true},
		{name: "convert run error", args: []string{"convert", "-c", runErrConfig}, wantFail: true},

		// merge
		{name: "merge all", args: []string{"merge"}, stdin: "1.0.0.0/25\n1.0.0.128/25\n2001:db8::/33\n2001:db8:8000::/33\n", wantStdout: []string{"1.0.0.0/24", "2001:db8::/32"}},
		{name: "merge ipv4", args: []string{"merge", "-t", "ipv4"}, stdin: "1.0.0.0/25\n1.0.0.128/25\n2001:db8::/32\n", wantStdout: []string{"1.0.0.0/24"}},
		{name: "merge ipv6", args: []string{"merge", "-t", " IPv6 "}, stdin: "1.0.0.0/24\n2001:db8::/33\n2001:db8:8000::/33\n", wantStdout: []string{"2001:db8::/32"}},
		{name: "merge invalid type", args: []string{"merge", "-t", "ipv5"}, wantFail: true, wantContain: "invalid argument onlyiptype"},
		{name: "merge skips invalid lines", args: []string{"merge"}, stdin: "not-an-ip\n# comment\n1.0.0.1 // c\n", wantStdout: []string{"1.0.0.1/32"}},
		{name: "merge empty input", args: []string{"merge"}, stdin: "", wantStdout: []string{}},
		{name: "merge run error", args: []string{"merge"}, stdin: strings.Repeat("a", 70*1024), wantFail: true, wantContain: "token too long"},

		// lookup once mode
		{name: "lookup uri found", args: []string{"lookup", "-f", "text", "-u", cnFile, "1.0.0.1"}, wantStdout: []string{"true"}},
		{name: "lookup uri not found", args: []string{"lookup", "-f", "text", "-u", cnFile, "8.8.8.8"}, wantStdout: []string{"false"}},
		{name: "lookup invalid search", args: []string{"lookup", "-f", "text", "-u", cnFile, "abc"}, wantStdout: []string{"false"}},
		{name: "lookup dir", args: []string{"lookup", "-f", "text", "-d", listDir, "1.0.0.1"}, wantStdout: []string{"cn,us"}},
		{name: "lookup dir searchlist", args: []string{"lookup", "-f", "text", "-d", listDir, "-l", "us", "1.0.0.1"}, wantStdout: []string{"us"}},
		{name: "lookup dir cidr", args: []string{"lookup", "-f", "text", "-d", listDir, "8.8.8.0/25"}, wantStdout: []string{"us"}},
		{name: "lookup unsupported format", args: []string{"lookup", "-f", "foo", "-u", cnFile, "1.1.1.1"}, wantFail: true, wantContain: "unsupported input format"},
		{name: "lookup dir disallowed", args: []string{"lookup", "-f", "maxmindMMDB", "-d", listDir, "1.1.1.1"}, wantFail: true, wantContain: "does not support"},
		{name: "lookup empty uri", args: []string{"lookup", "-f", "maxmindMMDB", "-u", "", "1.1.1.1"}, wantFail: true, wantContain: "requires a non-empty"},
		{name: "lookup run error", args: []string{"lookup", "-f", "maxmindMMDB", "-u", mmdbFile, "1.1.1.1"}, wantFail: true},
		{name: "lookup missing format", args: []string{"lookup", "-u", cnFile}, wantFail: true},
		{name: "lookup too many args", args: []string{"lookup", "-f", "text", "-u", cnFile, "a", "b"}, wantFail: true},

		// lookup REPL mode
		{name: "repl", args: []string{"lookup", "-f", "text", "-d", listDir}, stdin: "1.0.0.1\n\nabc\n8.8.8.8\n9.9.9.9\nexit\n1.0.0.1\n", wantStdout: []string{
			`Enter IP or CIDR (type "exit" to quit):`, ">> cn,us", ">>", ">> false", ">> us", ">> false", ">>",
		}},
		{name: "repl quoted exit", args: []string{"lookup", "-f", "text", "-u", cnFile}, stdin: "1.0.0.1\n\"exit\"\n", wantStdout: []string{
			`Enter IP or CIDR (type "exit" to quit):`, ">> true", ">>",
		}},
		{name: "repl eof", args: []string{"lookup", "-f", "text", "-u", cnFile}, stdin: "8.8.8.8", wantStdout: []string{
			`Enter IP or CIDR (type "exit" to quit):`, ">> false", ">>",
		}},
		{name: "repl input error", args: []string{"lookup", "-f", "maxmindMMDB", "-u", mmdbFile}, wantFail: true},
		{name: "repl scanner error", args: []string{"lookup", "-f", "text", "-u", cnFile}, stdin: strings.Repeat("a", 70*1024), wantFail: true, wantContain: "token too long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := runMain(t, tt.stdin, tt.args...)
			if failed := res.exitCode != 0; failed != tt.wantFail {
				t.Fatalf("exit code = %d, wantFail %v\nstdout: %s\nstderr: %s", res.exitCode, tt.wantFail, res.stdout, res.stderr)
			}
			if tt.wantStdout != nil {
				got := nonEmptyLines(res.stdout)
				if len(got) != len(tt.wantStdout) {
					t.Fatalf("stdout lines length = %d, want %d\ngot: %q\nwant: %q", len(got), len(tt.wantStdout), got, tt.wantStdout)
				}
				if !slices.Equal(got, tt.wantStdout) {
					t.Errorf("stdout lines = %q, want %q", got, tt.wantStdout)
				}
			}
			if tt.wantContain != "" && !strings.Contains(res.stdout+res.stderr, tt.wantContain) {
				t.Errorf("output does not contain %q\nstdout: %s\nstderr: %s", tt.wantContain, res.stdout, res.stderr)
			}
		})
	}
}
