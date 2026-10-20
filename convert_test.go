package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// jsonString returns s encoded as a JSON string literal.
func jsonString(t *testing.T, s string) string {
	t.Helper()

	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal(%q) error = %v", s, err)
	}

	return string(b)
}

// assertDirFiles asserts that dir contains exactly the files in want (by
// name) with exactly the given contents.
func assertDirFiles(t *testing.T, dir string, want map[string]string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) error = %v", dir, err)
	}
	if len(entries) != len(want) {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("output files = %q, want %d files", names, len(want))
	}
	for _, e := range entries {
		wantContent, ok := want[e.Name()]
		if !ok {
			t.Errorf("unexpected output file %q", e.Name())
			continue
		}
		got, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("os.ReadFile(%q) error = %v", e.Name(), err)
		}
		assertLines(t, string(got), wantContent)
	}
}

func TestConvertCmd(t *testing.T) {
	tests := []struct {
		name string
		// config returns the config content; outDir and inDir are temp dirs.
		config    func(t *testing.T, outDir, inDir string) string
		inFiles   map[string]string
		useHTTP   bool
		args      func(configURI string) []string
		wantFiles map[string]string
	}{
		{
			name: "inline text input and test input to text output",
			config: func(t *testing.T, outDir, _ string) string {
				return `{
  "input": [
    {"type": "text", "action": "add", "args": {"name": "cn", "ipOrCIDR": ["1.0.2.0/23", "2001:250::/35", "1.0.1.0/24", "1.0.1.1"]}},
    {"type": "test", "action": "add"}
  ],
  "output": [
    {"type": "text", "action": "output", "args": {"outputDir": ` + jsonString(t, outDir) + `}}
  ]
}`
			},
			args: func(uri string) []string { return []string{"convert", "-c", uri} },
			wantFiles: map[string]string{
				"cn.txt":   "1.0.1.0/24\n1.0.2.0/23\n2001:250::/35\n",
				"test.txt": "127.0.0.0/8\n",
			},
		},
		{
			name: "JSON with comments, cutter, remove action, wanted list and onlyIPType",
			config: func(t *testing.T, outDir, _ string) string {
				return `// leading comment
{
  "input": [
    {"type": "text", "action": "add", "args": {"name": "cn", "ipOrCIDR": ["1.0.1.0/24", "2001:250::/35"]}},
    {"type": "text", "action": "add", "args": {"name": "us", "ipOrCIDR": ["3.0.0.0/8", "2600::/16"]}},
    {"type": "text", "action": "add", "args": {"name": "jp", "ipOrCIDR": ["1.0.16.0/20"]}},
    /* remove JP entirely */
    {"type": "cutter", "action": "remove", "args": {"wantedList": ["jp"]}},
    {"type": "text", "action": "remove", "args": {"name": "us", "ipOrCIDR": ["3.0.0.0/9"]}},
  ],
  "output": [
    {"type": "text", "action": "output", "args": {"outputDir": ` + jsonString(t, outDir) + `, "wantedList": ["cn", "us", "jp"], "onlyIPType": "ipv4"}},
  ],
}`
			},
			args: func(uri string) []string { return []string{"conv", "--config", uri} },
			wantFiles: map[string]string{
				"cn.txt": "1.0.1.0/24\n",
				"us.txt": "3.128.0.0/9\n",
			},
		},
		{
			name: "local file input via uri",
			config: func(t *testing.T, outDir, inDir string) string {
				return `{
  "input": [
    {"type": "text", "action": "add", "args": {"name": "local", "uri": ` + jsonString(t, filepath.Join(inDir, "local.txt")) + `}}
  ],
  "output": [
    {"type": "text", "args": {"outputDir": ` + jsonString(t, outDir) + `, "outputExtension": ".list"}}
  ]
}`
			},
			inFiles: map[string]string{
				"local.txt": "# comment\n10.0.0.1\n10.0.0.0/24\n\n2001:db8::/48 // comment\n",
			},
			args: func(uri string) []string { return []string{"convert", "--config=" + uri} },
			wantFiles: map[string]string{
				"local.list": "10.0.0.0/24\n2001:db8::/48\n",
			},
		},
		{
			name: "remote config over HTTP",
			config: func(t *testing.T, outDir, _ string) string {
				return `{
  "input": [{"type": "text", "action": "add", "args": {"name": "remote", "ipOrCIDR": ["192.168.0.3/24", "192.168.1.5/24"]}}],
  "output": [{"type": "text", "args": {"outputDir": ` + jsonString(t, outDir) + `}}]
}`
			},
			useHTTP: true,
			args:    func(uri string) []string { return []string{"convert", "-c", uri} },
			wantFiles: map[string]string{
				"remote.txt": "192.168.0.0/23\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logBuf := captureLog(t)
			outDir := filepath.Join(t.TempDir(), "out")
			inDir := t.TempDir()
			for name, content := range tt.inFiles {
				writeFile(t, inDir, name, content)
			}

			config := tt.config(t, outDir, inDir)
			configURI := writeFile(t, t.TempDir(), "config.json", config)
			if tt.useHTTP {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/config.json" {
						http.NotFound(w, r)
						return
					}
					_, _ = w.Write([]byte(config))
				}))
				t.Cleanup(srv.Close)
				configURI = srv.URL + "/config.json"
			}

			stdout, cobraOutput, err := executeCommand(t, tt.args(configURI)...)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if cobraOutput != "" {
				t.Errorf("cobra output = %q, want empty", cobraOutput)
			}
			if !strings.Contains(logBuf.String(), "Use config: "+configURI+"\n") {
				t.Errorf("log output = %q, want it to contain the used config", logBuf.String())
			}

			assertDirFiles(t, outDir, tt.wantFiles)
		})
	}
}

func TestConvertCmdFatal(t *testing.T) {
	if args, ok := subprocessArgs(t); ok {
		rootCmd.SetArgs(args)
		_ = rootCmd.Execute()
		return
	}

	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")
	invalidJSON := writeFile(t, dir, "invalid.json", `{"input": [`)
	unknownType := writeFile(t, dir, "unknown.json", `{"input": [{"type": "bogus", "action": "add"}]}`)
	noOutput := writeFile(t, dir, "no-output.json", `{"input": [{"type": "test", "action": "add"}]}`)
	missingInput := writeFile(t, dir, "missing-input.json", `{
  "input": [{"type": "text", "action": "add", "args": {"name": "x", "uri": `+jsonString(t, filepath.Join(dir, "missing.txt"))+`}}],
  "output": [{"type": "text", "args": {"outputDir": `+jsonString(t, outDir)+`}}]
}`)

	tests := []struct {
		name    string
		args    []string
		wantMsg string
	}{
		{name: "config file not found", args: []string{"convert", "-c", filepath.Join(dir, "missing.json")}, wantMsg: "no such file or directory"},
		{name: "invalid config JSON", args: []string{"convert", "-c", invalidJSON}, wantMsg: "unexpected end of JSON input"},
		{name: "unknown converter type", args: []string{"convert", "-c", unknownType}, wantMsg: "unknown config type"},
		{name: "config without output", args: []string{"convert", "-c", noOutput}, wantMsg: "input type and output type must be specified"},
		{name: "input fails at run time", args: []string{"convert", "-c", missingInput}, wantMsg: "missing.txt: no such file or directory"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stderr := runSubprocess(t, "TestConvertCmdFatal", tt.args, "")
			assertFatal(t, code, stderr, tt.wantMsg)
		})
	}

	if _, err := os.Stat(outDir); !os.IsNotExist(err) {
		t.Errorf("output dir %q should not be created on failure, stat error = %v", outDir, err)
	}
}
