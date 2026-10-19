package plaintext

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

// mixedCIDRs contains IPv4, IPv6 and IPv4-mapped IPv6 inputs.
// After normalization they become: 1.0.0.0/23, 8.8.8.8/32, 10.0.0.0/8, 2001:db8::/32, 2606:4700::1/128.
var mixedCIDRs = []string{
	"1.0.0.0/24",
	"1.0.1.0/24",
	"::ffff:8.8.8.8",
	"::ffff:10.0.0.0/104",
	"2001:db8::/32",
	"2606:4700::1",
}

func TestGetDefaultOutputDir(t *testing.T) {
	tests := []struct {
		iType string
		want  string
	}{
		{TypeTextOut, filepath.Join("output", "text")},
		{TypeClashRuleSetClassicalOut, filepath.Join("output", "clash", "classical")},
		{TypeClashRuleSetIPCIDROut, filepath.Join("output", "clash", "ipcidr")},
		{TypeSurgeRuleSetOut, filepath.Join("output", "surge")},
		{"unknown", filepath.Join("output", "text")},
	}
	for _, tt := range tests {
		t.Run(tt.iType, func(t *testing.T) {
			if got := getDefaultOutputDir(tt.iType); got != tt.want {
				t.Errorf("getDefaultOutputDir(%q) = %q, want %q", tt.iType, got, tt.want)
			}
		})
	}
}

func TestNewTextOut(t *testing.T) {
	tests := []struct {
		name  string
		iType string
		opts  []lib.OutputOption
		check func(t *testing.T, o *text_out)
	}{
		{
			name:  "defaults for text",
			iType: TypeTextOut,
			check: func(t *testing.T, o *text_out) {
				if o.OutputDir != defaultOutputDirForTextOut || o.OutputExt != ".txt" {
					t.Errorf("OutputDir/Ext = %q/%q", o.OutputDir, o.OutputExt)
				}
			},
		},
		{
			name:  "defaults for surge",
			iType: TypeSurgeRuleSetOut,
			opts:  []lib.OutputOption{nil},
			check: func(t *testing.T, o *text_out) {
				if o.OutputDir != defaultOutputDirForSurgeRuleSetOut {
					t.Errorf("OutputDir = %q", o.OutputDir)
				}
			},
		},
		{
			name:  "empty dir/ext fall back to defaults",
			iType: TypeClashRuleSetIPCIDROut,
			opts:  []lib.OutputOption{WithOutputDir("   "), WithOutputExtension("  ")},
			check: func(t *testing.T, o *text_out) {
				if o.OutputDir != defaultOutputDirForClashRuleSetIPCIDROut || o.OutputExt != ".txt" {
					t.Errorf("OutputDir/Ext = %q/%q", o.OutputDir, o.OutputExt)
				}
			},
		},
		{
			name:  "all options",
			iType: TypeClashRuleSetClassicalOut,
			opts: []lib.OutputOption{
				WithOutputDir("  /tmp/out  "),
				WithOutputExtension(" .yaml "),
				WithOutputWantedList([]string{"cn"}),
				WithOutputExcludedList([]string{"us"}),
				WithOutputOnlyIPType(lib.IPv4),
				WithAddPrefixInLine("pre"),
				WithAddSuffixInLine("suf"),
			},
			check: func(t *testing.T, o *text_out) {
				if o.OutputDir != "/tmp/out" || o.OutputExt != ".yaml" {
					t.Errorf("OutputDir/Ext = %q/%q", o.OutputDir, o.OutputExt)
				}
				compareStrings(t, "Want", o.Want, []string{"cn"})
				compareStrings(t, "Exclude", o.Exclude, []string{"us"})
				if o.OnlyIPType != lib.IPv4 || o.AddPrefixInLine != "pre" || o.AddSuffixInLine != "suf" {
					t.Errorf("OnlyIPType/prefix/suffix = %q/%q/%q", o.OnlyIPType, o.AddPrefixInLine, o.AddSuffixInLine)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := newTextOut(tt.iType, "desc", lib.ActionOutput, tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			o := conv.(*text_out)
			if o.Type != tt.iType || o.Description != "desc" || o.Action != lib.ActionOutput {
				t.Errorf("Type/Desc/Action = %q/%q/%q", o.Type, o.Description, o.Action)
			}
			tt.check(t, o)
		})
	}
}

func TestNewTextOutFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		iType   string
		data    string
		wantErr string
		check   func(t *testing.T, o *text_out)
	}{
		{
			name:  "empty data uses defaults",
			iType: TypeClashRuleSetClassicalOut,
			check: func(t *testing.T, o *text_out) {
				if o.OutputDir != defaultOutputDirForClashRuleSetClassicalOut || o.OutputExt != ".txt" {
					t.Errorf("OutputDir/Ext = %q/%q", o.OutputDir, o.OutputExt)
				}
			},
		},
		{
			name:    "invalid json",
			iType:   TypeTextOut,
			data:    "{",
			wantErr: "unexpected end of JSON input",
		},
		{
			name:  "full config",
			iType: TypeTextOut,
			data: `{"outputDir":"./out","outputExtension":".list","wantedList":["cn"],"excludedList":["us"],` +
				`"onlyIPType":"ipv6","addPrefixInLine":"- ","addSuffixInLine":",x"}`,
			check: func(t *testing.T, o *text_out) {
				if o.OutputDir != "./out" || o.OutputExt != ".list" {
					t.Errorf("OutputDir/Ext = %q/%q", o.OutputDir, o.OutputExt)
				}
				compareStrings(t, "Want", o.Want, []string{"cn"})
				compareStrings(t, "Exclude", o.Exclude, []string{"us"})
				if o.OnlyIPType != lib.IPv6 || o.AddPrefixInLine != "- " || o.AddSuffixInLine != ",x" {
					t.Errorf("OnlyIPType/prefix/suffix = %q/%q/%q", o.OnlyIPType, o.AddPrefixInLine, o.AddSuffixInLine)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := newTextOutFromBytes(tt.iType, "desc", lib.ActionOutput, []byte(tt.data))
			if checkErr(t, err, tt.wantErr) {
				return
			}
			tt.check(t, conv.(*text_out))
		})
	}
}

func TestTextOutMarshalBytes(t *testing.T) {
	tests := []struct {
		name    string
		out     *text_out
		cidrs   []string
		want    []string // expected lines
		wantErr string
	}{
		{
			name:  "text plain",
			out:   &text_out{Type: TypeTextOut},
			cidrs: mixedCIDRs,
			want:  []string{"1.0.0.0/23", "8.8.8.8/32", "10.0.0.0/8", "2001:db8::/32", "2606:4700::1/128"},
		},
		{
			name:  "text with prefix and suffix",
			out:   &text_out{Type: TypeTextOut, AddPrefixInLine: "IP-CIDR,", AddSuffixInLine: ",no-resolve"},
			cidrs: mixedCIDRs,
			want: []string{
				"IP-CIDR,1.0.0.0/23,no-resolve", "IP-CIDR,8.8.8.8/32,no-resolve", "IP-CIDR,10.0.0.0/8,no-resolve",
				"IP-CIDR,2001:db8::/32,no-resolve", "IP-CIDR,2606:4700::1/128,no-resolve",
			},
		},
		{
			name:  "text only ipv4",
			out:   &text_out{Type: TypeTextOut, OnlyIPType: lib.IPv4},
			cidrs: mixedCIDRs,
			want:  []string{"1.0.0.0/23", "8.8.8.8/32", "10.0.0.0/8"},
		},
		{
			name:  "text only ipv6",
			out:   &text_out{Type: TypeTextOut, OnlyIPType: lib.IPv6},
			cidrs: mixedCIDRs,
			want:  []string{"2001:db8::/32", "2606:4700::1/128"},
		},
		{
			name:  "clash classical",
			out:   &text_out{Type: TypeClashRuleSetClassicalOut, AddPrefixInLine: "ignored", AddSuffixInLine: "ignored"},
			cidrs: mixedCIDRs,
			want: []string{
				"payload:",
				"  - IP-CIDR,1.0.0.0/23", "  - IP-CIDR,8.8.8.8/32", "  - IP-CIDR,10.0.0.0/8",
				"  - IP-CIDR6,2001:db8::/32", "  - IP-CIDR6,2606:4700::1/128",
			},
		},
		{
			name:  "clash ipcidr",
			out:   &text_out{Type: TypeClashRuleSetIPCIDROut},
			cidrs: mixedCIDRs,
			want: []string{
				"payload:",
				"  - '1.0.0.0/23'", "  - '8.8.8.8/32'", "  - '10.0.0.0/8'",
				"  - '2001:db8::/32'", "  - '2606:4700::1/128'",
			},
		},
		{
			name:  "clash ipcidr only ipv6",
			out:   &text_out{Type: TypeClashRuleSetIPCIDROut, OnlyIPType: lib.IPv6},
			cidrs: mixedCIDRs,
			want:  []string{"payload:", "  - '2001:db8::/32'", "  - '2606:4700::1/128'"},
		},
		{
			name:  "surge without suffix",
			out:   &text_out{Type: TypeSurgeRuleSetOut},
			cidrs: mixedCIDRs,
			want: []string{
				"IP-CIDR,1.0.0.0/23", "IP-CIDR,8.8.8.8/32", "IP-CIDR,10.0.0.0/8",
				"IP-CIDR6,2001:db8::/32", "IP-CIDR6,2606:4700::1/128",
			},
		},
		{
			name:  "surge with suffix (prefix is ignored)",
			out:   &text_out{Type: TypeSurgeRuleSetOut, AddPrefixInLine: "ignored", AddSuffixInLine: ",no-resolve"},
			cidrs: mixedCIDRs,
			want: []string{
				"IP-CIDR,1.0.0.0/23,no-resolve", "IP-CIDR,8.8.8.8/32,no-resolve", "IP-CIDR,10.0.0.0/8,no-resolve",
				"IP-CIDR6,2001:db8::/32,no-resolve", "IP-CIDR6,2606:4700::1/128,no-resolve",
			},
		},
		{
			name:    "empty entry",
			out:     &text_out{Type: TypeTextOut},
			wantErr: "has no prefix",
		},
		{
			name:    "only ipv6 requested but entry has only ipv4",
			out:     &text_out{Type: TypeSurgeRuleSetOut, OnlyIPType: lib.IPv6},
			cidrs:   []string{"1.1.1.1", "::ffff:2.2.2.2"},
			wantErr: "has no prefix",
		},
		{
			name:    "unsupported type",
			out:     &text_out{Type: "unknown"},
			cidrs:   []string{"1.1.1.1"},
			wantErr: lib.ErrNotSupportedFormat.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := newEntryWith(t, "test", tt.cidrs...)
			data, err := tt.out.marshalBytes(entry)
			if checkErr(t, err, tt.wantErr) {
				if data != nil {
					t.Errorf("data should be nil on error")
				}
				return
			}
			if !bytes.HasSuffix(data, []byte("\n")) {
				t.Errorf("output should end with newline: %q", data)
			}
			lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
			compareStrings(t, "lines", lines, tt.want)
		})
	}
}

func TestTextOutMarshalHelpersInvalidCIDR(t *testing.T) {
	o := &text_out{}
	tests := []struct {
		name string
		fn   func(*bytes.Buffer, []string) error
	}{
		{"clash classical", o.marshalBytesForClashRuleSetClassicalOut},
		{"surge", o.marshalBytesForSurgeRuleSetOut},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := tt.fn(&buf, []string{"1.1.1.1/32", "not-a-cidr"}); err == nil {
				t.Fatal("expected error for invalid CIDR")
			}
		})
	}
}

func TestTextOutFilterAndSortList(t *testing.T) {
	container := newContainerWith(t, map[string][]string{
		"CN": {"1.0.0.0/24"},
		"US": {"2001:db8::/32"},
		"JP": {"::ffff:3.3.3.3"},
	})

	tests := []struct {
		name    string
		want    []string
		exclude []string
		expect  []string
	}{
		{name: "all, sorted", expect: []string{"CN", "JP", "US"}},
		{name: "exclude", exclude: []string{" us ", ""}, expect: []string{"CN", "JP"}},
		{name: "exclude everything", exclude: []string{"cn", "jp", "us"}, expect: []string{}},
		{name: "wanted list sorted and uppercased", want: []string{" us", "cn ", "", "  "}, expect: []string{"CN", "US"}},
		{name: "wanted minus excluded", want: []string{"cn", "us"}, exclude: []string{"US"}, expect: []string{"CN"}},
		{name: "wanted all excluded", want: []string{"cn"}, exclude: []string{"cn"}, expect: []string{}},
		{name: "wanted includes unknown list", want: []string{"zz", "cn"}, expect: []string{"CN", "ZZ"}},
		{name: "only blank wanted entries act as no wanted list", want: []string{"", " "}, expect: []string{"CN", "JP", "US"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &text_out{Type: TypeTextOut, Want: tt.want, Exclude: tt.exclude}
			got := o.filterAndSortList(container)
			if got == nil {
				t.Fatal("result should not be nil")
			}
			compareStrings(t, "list", got, tt.expect)
		})
	}
}

func TestTextOutOutput(t *testing.T) {
	lists := map[string][]string{
		"CN":    mixedCIDRs,
		"US":    {"2.0.0.0/24", "2001:db9::/32"},
		"EMPTY": nil,
	}

	type fileWant map[string][]string // filename -> lines

	tests := []struct {
		name      string
		newFn     func(lib.Action, ...lib.OutputOption) (lib.OutputConverter, error)
		opts      []lib.OutputOption
		files     fileWant
		wantErr   string
		setupDir  func(t *testing.T, dir string) string // returns OutputDir
		wantNoDir bool
	}{
		{
			name:  "text wanted lists with missing entry skipped",
			newFn: NewTextOut,
			opts:  []lib.OutputOption{WithOutputWantedList([]string{"cn", "us", "missing"})},
			files: fileWant{
				"cn.txt": {"1.0.0.0/23", "8.8.8.8/32", "10.0.0.0/8", "2001:db8::/32", "2606:4700::1/128"},
				"us.txt": {"2.0.0.0/24", "2001:db9::/32"},
			},
		},
		{
			name:  "text only ipv4 with custom extension",
			newFn: NewTextOut,
			opts: []lib.OutputOption{
				WithOutputWantedList([]string{"cn"}),
				WithOutputOnlyIPType(lib.IPv4),
				WithOutputExtension(".list"),
			},
			files: fileWant{"cn.list": {"1.0.0.0/23", "8.8.8.8/32", "10.0.0.0/8"}},
		},
		{
			name:  "clash classical",
			newFn: NewClashRuleSetClassicalOut,
			opts:  []lib.OutputOption{WithOutputExcludedList([]string{"empty", "us"}), WithOutputExtension(".yaml")},
			files: fileWant{"cn.yaml": {
				"payload:",
				"  - IP-CIDR,1.0.0.0/23", "  - IP-CIDR,8.8.8.8/32", "  - IP-CIDR,10.0.0.0/8",
				"  - IP-CIDR6,2001:db8::/32", "  - IP-CIDR6,2606:4700::1/128",
			}},
		},
		{
			name:  "clash ipcidr only ipv6",
			newFn: NewClashRuleSetIPCIDROut,
			opts:  []lib.OutputOption{WithOutputWantedList([]string{"us"}), WithOutputOnlyIPType(lib.IPv6)},
			files: fileWant{"us.txt": {"payload:", "  - '2001:db9::/32'"}},
		},
		{
			name:  "surge",
			newFn: NewSurgeRuleSetOut,
			opts:  []lib.OutputOption{WithOutputWantedList([]string{"us"}), WithAddSuffixInLine(",no-resolve")},
			files: fileWant{"us.txt": {"IP-CIDR,2.0.0.0/24,no-resolve", "IP-CIDR6,2001:db9::/32,no-resolve"}},
		},
		{
			name:      "nothing wanted writes nothing",
			newFn:     NewTextOut,
			opts:      []lib.OutputOption{WithOutputWantedList([]string{"missing"})},
			files:     fileWant{},
			wantNoDir: true,
		},
		{
			name:    "empty entry produces marshal error",
			newFn:   NewTextOut,
			opts:    []lib.OutputOption{WithOutputWantedList([]string{"empty"})},
			wantErr: "entry EMPTY has no prefix",
		},
		{
			name:  "output dir cannot be created",
			newFn: NewTextOut,
			opts:  []lib.OutputOption{WithOutputWantedList([]string{"cn"})},
			setupDir: func(t *testing.T, dir string) string {
				blocker := filepath.Join(dir, "blocker")
				writeTestFile(t, blocker, "x")
				return filepath.Join(blocker, "sub")
			},
			wantErr: "not a directory",
		},
		{
			name:  "output file cannot be written",
			newFn: NewTextOut,
			opts:  []lib.OutputOption{WithOutputWantedList([]string{"cn"})},
			setupDir: func(t *testing.T, dir string) string {
				out := filepath.Join(dir, "out")
				if err := os.MkdirAll(filepath.Join(out, "cn.txt"), 0755); err != nil {
					t.Fatal(err)
				}
				return out
			},
			wantErr: "is a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			outDir := filepath.Join(base, "out")
			if tt.setupDir != nil {
				outDir = tt.setupDir(t, base)
			}
			opts := append([]lib.OutputOption{WithOutputDir(outDir)}, tt.opts...)
			conv, err := tt.newFn(lib.ActionOutput, opts...)
			if err != nil {
				t.Fatal(err)
			}

			container := newContainerWith(t, lists)
			err = conv.Output(container)
			if checkErr(t, err, tt.wantErr) {
				return
			}

			entries, readErr := os.ReadDir(outDir)
			if tt.wantNoDir {
				if !errors.Is(readErr, os.ErrNotExist) {
					t.Fatalf("output dir should not exist, ReadDir err = %v, entries = %v", readErr, entries)
				}
				return
			}
			if readErr != nil {
				t.Fatal(readErr)
			}
			if len(entries) != len(tt.files) {
				t.Fatalf("file count = %d, want %d", len(entries), len(tt.files))
			}
			for filename, wantLines := range tt.files {
				data, err := os.ReadFile(filepath.Join(outDir, filename))
				if err != nil {
					t.Fatalf("read %s: %v", filename, err)
				}
				lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
				compareStrings(t, filename, lines, wantLines)
			}
		})
	}
}

func TestTextOutGetters(t *testing.T) {
	tests := []struct {
		name     string
		newFn    func(lib.Action, ...lib.OutputOption) (lib.OutputConverter, error)
		fromFn   func(lib.Action, []byte) (lib.OutputConverter, error)
		wantType string
		wantDesc string
		wantDir  string
	}{
		{"text", NewTextOut, NewTextOutFromBytes, TypeTextOut, DescTextOut, defaultOutputDirForTextOut},
		{"clash classical", NewClashRuleSetClassicalOut, NewClashRuleSetClassicalOutFromBytes, TypeClashRuleSetClassicalOut, DescClashRuleSetClassicalOut, defaultOutputDirForClashRuleSetClassicalOut},
		{"clash ipcidr", NewClashRuleSetIPCIDROut, NewClashRuleSetIPCIDROutFromBytes, TypeClashRuleSetIPCIDROut, DescClashRuleSetIPCIDROut, defaultOutputDirForClashRuleSetIPCIDROut},
		{"surge", NewSurgeRuleSetOut, NewSurgeRuleSetOutFromBytes, TypeSurgeRuleSetOut, DescSurgeRuleSetOut, defaultOutputDirForSurgeRuleSetOut},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c1, err := tt.newFn(lib.ActionOutput)
			if err != nil {
				t.Fatal(err)
			}
			c2, err := tt.fromFn(lib.ActionOutput, []byte(`{"outputDir":""}`))
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range []lib.OutputConverter{c1, c2} {
				if c.GetType() != tt.wantType {
					t.Errorf("GetType = %q, want %q", c.GetType(), tt.wantType)
				}
				if c.GetAction() != lib.ActionOutput {
					t.Errorf("GetAction = %q", c.GetAction())
				}
				if c.GetDescription() != tt.wantDesc {
					t.Errorf("GetDescription = %q, want %q", c.GetDescription(), tt.wantDesc)
				}
				if dir := c.(*text_out).OutputDir; dir != tt.wantDir {
					t.Errorf("OutputDir = %q, want %q", dir, tt.wantDir)
				}
			}
			if _, err := tt.fromFn(lib.ActionOutput, []byte("[")); err == nil {
				t.Error("expected error for invalid JSON bytes")
			}
		})
	}
}

func TestTextOutRegisteredConfigCreators(t *testing.T) {
	tests := []struct {
		name  string
		iType string
	}{
		{"text", TypeTextOut},
		{"clash classical", TypeClashRuleSetClassicalOut},
		{"clash ipcidr", TypeClashRuleSetIPCIDROut},
		{"surge", TypeSurgeRuleSetOut},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst, err := lib.NewInstance()
			if err != nil {
				t.Fatal(err)
			}
			cfg := `{"input":[{"type":"text","action":"add","args":{"name":"cn","ipOrCIDR":["1.1.1.1","::ffff:2.2.2.2","2001:db8::/32"]}}],` +
				`"output":[{"type":"` + tt.iType + `","action":"output","args":{"outputDir":"` + filepath.ToSlash(t.TempDir()) + `"}}]}`
			if err := inst.InitConfigFromBytes([]byte(cfg)); err != nil {
				t.Fatalf("InitConfigFromBytes: %v", err)
			}
			if err := inst.Run(); err != nil {
				t.Fatalf("Run: %v", err)
			}
		})
	}
}

func TestTextOutWriteFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "a", "b")
	o := &text_out{Type: TypeTextOut, OutputDir: dir}
	if err := o.writeFile("x.txt", []byte("1.1.1.1/32\n")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "x.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "1.1.1.1/32\n" {
		t.Errorf("content = %q", data)
	}
}
