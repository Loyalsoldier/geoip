package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readLines(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return strings.Fields(string(b))
}

func TestConvertCmd(t *testing.T) {
	type wantFile struct {
		name  string
		lines []string
	}
	tests := []struct {
		name    string
		cmd     string
		flag    string
		config  string // %[1]q is replaced by the output directory
		want    []wantFile
		missing []string
	}{
		{
			name: "text to text merges CIDRs",
			cmd:  "convert", flag: "-c",
			config: `{
				"input": [{"type": "text", "action": "add", "args": {"name": "cn", "ipOrCIDR": ["1.0.0.0/24", "1.0.1.0/24", "2001:db8::/32", "10.0.0.1"]}}],
				"output": [{"type": "text", "action": "output", "args": {"outputDir": %[1]q}}]
			}`,
			want: []wantFile{{"cn.txt", []string{"1.0.0.0/23", "10.0.0.1/32", "2001:db8::/32"}}},
		},
		{
			name: "alias conv, long flag, onlyIPType ipv4, default action",
			cmd:  "conv", flag: "--config",
			config: `{
				"input": [{"type": "text", "action": "add", "args": {"name": "cn", "ipOrCIDR": ["1.0.0.0/24", "2001:db8::/32"]}}],
				"output": [{"type": "text", "args": {"outputDir": %[1]q, "onlyIPType": "ipv4"}}]
			}`,
			want: []wantFile{{"cn.txt", []string{"1.0.0.0/24"}}},
		},
		{
			name: "add then remove",
			cmd:  "convert", flag: "-c",
			config: `{
				"input": [
					{"type": "text", "action": "add", "args": {"name": "cn", "ipOrCIDR": ["1.0.0.0/23"]}},
					{"type": "text", "action": "remove", "args": {"name": "cn", "ipOrCIDR": ["1.0.1.0/24"]}}
				],
				"output": [{"type": "text", "action": "output", "args": {"outputDir": %[1]q}}]
			}`,
			want: []wantFile{{"cn.txt", []string{"1.0.0.0/24"}}},
		},
		{
			name: "hujson comments and trailing commas, wantedList",
			cmd:  "convert", flag: "-c",
			config: `{
				// comment
				"input": [
					{"type": "text", "action": "add", "args": {"name": "cn", "ipOrCIDR": ["1.0.0.0/24"],},},
					{"type": "text", "action": "add", "args": {"name": "us", "ipOrCIDR": ["8.8.8.0/24", "8.8.9.0/24", "2001:db8::/48"],},},
				],
				/* block comment */
				"output": [{"type": "text", "action": "output", "args": {"outputDir": %[1]q, "wantedList": ["us"],},},],
			}`,
			want:    []wantFile{{"us.txt", []string{"8.8.8.0/23", "2001:db8::/48"}}},
			missing: []string{"cn.txt"},
		},
		{
			name: "multiple lists with prefix/suffix and excludedList",
			cmd:  "convert", flag: "-c",
			config: `{
				"input": [
					{"type": "text", "action": "add", "args": {"name": "a", "ipOrCIDR": ["192.168.0.0/24"]}},
					{"type": "text", "action": "add", "args": {"name": "b", "ipOrCIDR": ["10.0.0.0/8", "fd00::/8"]}},
					{"type": "text", "action": "add", "args": {"name": "c", "ipOrCIDR": ["172.16.0.0/12"]}}
				],
				"output": [{"type": "text", "action": "output", "args": {"outputDir": %[1]q, "excludedList": ["c"], "addPrefixInLine": "P:", "addSuffixInLine": ";S"}}]
			}`,
			want: []wantFile{
				{"a.txt", []string{"P:192.168.0.0/24;S"}},
				{"b.txt", []string{"P:10.0.0.0/8;S", "P:fd00::/8;S"}},
			},
			missing: []string{"c.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			outDir := filepath.Join(tmp, "out")
			cfgPath := filepath.Join(tmp, "config.json")
			writeFile(t, cfgPath, fmt.Sprintf(tt.config, outDir))

			if err := runCmd(t, tt.cmd, tt.flag, cfgPath); err != nil {
				t.Fatalf("Execute error: %v", err)
			}

			for _, wf := range tt.want {
				got := readLines(t, filepath.Join(outDir, wf.name))
				if len(got) != len(wf.lines) {
					t.Fatalf("%s: got %d lines %v, want %d %v", wf.name, len(got), got, len(wf.lines), wf.lines)
				}
				for i := range got {
					if got[i] != wf.lines[i] {
						t.Errorf("%s line %d = %q, want %q", wf.name, i, got[i], wf.lines[i])
					}
				}
			}
			for _, m := range tt.missing {
				if _, err := os.Stat(filepath.Join(outDir, m)); !os.IsNotExist(err) {
					t.Errorf("%s should not exist (stat err = %v)", m, err)
				}
			}
			entries, err := os.ReadDir(outDir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != len(tt.want) {
				t.Errorf("output dir has %d files, want %d", len(entries), len(tt.want))
			}
		})
	}
	// NOTE: missing/invalid config files, unknown types and input/output
	// errors all hit log.Fatal and are not tested.
}

func TestConvertCmdConfigFlagDefault(t *testing.T) {
	f := convertCmd.PersistentFlags().Lookup("config")
	if f == nil {
		t.Fatal("config flag not registered")
	}
	if f.DefValue != "config.json" {
		t.Errorf("default = %q, want config.json", f.DefValue)
	}
	if f.Shorthand != "c" {
		t.Errorf("shorthand = %q, want c", f.Shorthand)
	}
}
