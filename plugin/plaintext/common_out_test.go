package plaintext

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

// assertTextOut compares every field of got with want.
func assertTextOut(t *testing.T, got, want *text_out) {
	t.Helper()

	if got.Type != want.Type {
		t.Errorf("Type = %q, want %q", got.Type, want.Type)
	}
	if got.Action != want.Action {
		t.Errorf("Action = %q, want %q", got.Action, want.Action)
	}
	if got.Description != want.Description {
		t.Errorf("Description = %q, want %q", got.Description, want.Description)
	}
	if got.OutputDir != want.OutputDir {
		t.Errorf("OutputDir = %q, want %q", got.OutputDir, want.OutputDir)
	}
	if got.OutputExt != want.OutputExt {
		t.Errorf("OutputExt = %q, want %q", got.OutputExt, want.OutputExt)
	}
	assertStrings(t, "Want", got.Want, want.Want)
	assertStrings(t, "Exclude", got.Exclude, want.Exclude)
	if got.OnlyIPType != want.OnlyIPType {
		t.Errorf("OnlyIPType = %q, want %q", got.OnlyIPType, want.OnlyIPType)
	}
	if got.AddPrefixInLine != want.AddPrefixInLine {
		t.Errorf("AddPrefixInLine = %q, want %q", got.AddPrefixInLine, want.AddPrefixInLine)
	}
	if got.AddSuffixInLine != want.AddSuffixInLine {
		t.Errorf("AddSuffixInLine = %q, want %q", got.AddSuffixInLine, want.AddSuffixInLine)
	}
}

func TestGetDefaultOutputDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		iType string
		want  string
	}{
		{iType: TypeTextOut, want: filepath.Join("output", "text")},
		{iType: TypeClashRuleSetClassicalOut, want: filepath.Join("output", "clash", "classical")},
		{iType: TypeClashRuleSetIPCIDROut, want: filepath.Join("output", "clash", "ipcidr")},
		{iType: TypeSurgeRuleSetOut, want: filepath.Join("output", "surge")},
		{iType: "unknown", want: filepath.Join("output", "text")},
		{iType: "", want: filepath.Join("output", "text")},
	}

	for _, tt := range tests {
		t.Run("type "+tt.iType, func(t *testing.T) {
			t.Parallel()

			if got := getDefaultOutputDir(tt.iType); got != tt.want {
				t.Errorf("getDefaultOutputDir(%q) = %q, want %q", tt.iType, got, tt.want)
			}
		})
	}
}

func TestNewTextOut(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		iType string
		opts  []lib.OutputOption
		want  *text_out
	}{
		{
			name:  "text defaults",
			iType: TypeTextOut,
			want:  &text_out{OutputDir: filepath.Join("output", "text"), OutputExt: ".txt"},
		},
		{
			name:  "clashRuleSetClassical defaults",
			iType: TypeClashRuleSetClassicalOut,
			want:  &text_out{OutputDir: filepath.Join("output", "clash", "classical"), OutputExt: ".txt"},
		},
		{
			name:  "clashRuleSet defaults",
			iType: TypeClashRuleSetIPCIDROut,
			want:  &text_out{OutputDir: filepath.Join("output", "clash", "ipcidr"), OutputExt: ".txt"},
		},
		{
			name:  "surgeRuleSet defaults",
			iType: TypeSurgeRuleSetOut,
			want:  &text_out{OutputDir: filepath.Join("output", "surge"), OutputExt: ".txt"},
		},
		{
			name:  "nil options are ignored",
			iType: TypeTextOut,
			opts:  []lib.OutputOption{nil, WithOutputExtension(".conf"), nil},
			want:  &text_out{OutputDir: filepath.Join("output", "text"), OutputExt: ".conf"},
		},
		{
			name:  "all options",
			iType: TypeTextOut,
			opts: []lib.OutputOption{
				WithOutputDir("  ./custom  "),
				WithOutputExtension("  .list  "),
				WithOutputWantedList([]string{"cn", "us"}),
				WithOutputExcludedList([]string{"jp"}),
				WithOutputOnlyIPType(lib.IPv4),
				WithAddPrefixInLine("IP-CIDR,"),
				WithAddSuffixInLine(",no-resolve"),
			},
			want: &text_out{
				OutputDir:       "./custom",
				OutputExt:       ".list",
				Want:            []string{"cn", "us"},
				Exclude:         []string{"jp"},
				OnlyIPType:      lib.IPv4,
				AddPrefixInLine: "IP-CIDR,",
				AddSuffixInLine: ",no-resolve",
			},
		},
		{
			name:  "empty output dir and extension fall back to defaults of the type",
			iType: TypeSurgeRuleSetOut,
			opts:  []lib.OutputOption{WithOutputDir("   "), WithOutputExtension("")},
			want:  &text_out{OutputDir: filepath.Join("output", "surge"), OutputExt: ".txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := newTextOut(tt.iType, "test description", lib.ActionOutput, tt.opts...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.want.Type = tt.iType
			tt.want.Action = lib.ActionOutput
			tt.want.Description = "test description"
			assertTextOut(t, asTextOut(t, conv), tt.want)
		})
	}
}

func TestNewTextOutFromBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		iType string
		data  string
		want  *text_out
	}{
		{
			name:  "empty data for text",
			iType: TypeTextOut,
			data:  "",
			want:  &text_out{OutputDir: filepath.Join("output", "text"), OutputExt: ".txt"},
		},
		{
			name:  "empty object for clashRuleSetClassical",
			iType: TypeClashRuleSetClassicalOut,
			data:  `{}`,
			want:  &text_out{OutputDir: filepath.Join("output", "clash", "classical"), OutputExt: ".txt"},
		},
		{
			name:  "null for clashRuleSet",
			iType: TypeClashRuleSetIPCIDROut,
			data:  `null`,
			want:  &text_out{OutputDir: filepath.Join("output", "clash", "ipcidr"), OutputExt: ".txt"},
		},
		{
			name:  "whitespace-only values for surgeRuleSet",
			iType: TypeSurgeRuleSetOut,
			data:  `{"outputDir":"   ","outputExtension":"  "}`,
			want:  &text_out{OutputDir: filepath.Join("output", "surge"), OutputExt: ".txt"},
		},
		{
			name:  "all args",
			iType: TypeTextOut,
			data: `{
				"outputDir": " ./text ",
				"outputExtension": " .conf ",
				"wantedList": ["cn", "us"],
				"excludedList": ["jp"],
				"onlyIPType": "ipv6",
				"addPrefixInLine": "HOST,",
				"addSuffixInLine": ";",
				"unknownField": true
			}`,
			want: &text_out{
				OutputDir:       "./text",
				OutputExt:       ".conf",
				Want:            []string{"cn", "us"},
				Exclude:         []string{"jp"},
				OnlyIPType:      lib.IPv6,
				AddPrefixInLine: "HOST,",
				AddSuffixInLine: ";",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			desc := outputDescriptions[tt.iType]
			conv, err := newTextOutFromBytes(tt.iType, desc, lib.ActionOutput, []byte(tt.data))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.want.Type = tt.iType
			tt.want.Action = lib.ActionOutput
			tt.want.Description = desc
			assertTextOut(t, asTextOut(t, conv), tt.want)
		})
	}
}

func TestNewTextOutFromBytes_InvalidJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		data      string
		checkType func(error) bool
	}{
		{
			name: "truncated json",
			data: `{"outputDir":`,
			checkType: func(err error) bool {
				var target *json.SyntaxError
				return errors.As(err, &target)
			},
		},
		{
			name: "wrong type for outputDir",
			data: `{"outputDir":1}`,
			checkType: func(err error) bool {
				var target *json.UnmarshalTypeError
				return errors.As(err, &target)
			},
		},
		{
			name: "wrong type for wantedList",
			data: `{"wantedList":"cn"}`,
			checkType: func(err error) bool {
				var target *json.UnmarshalTypeError
				return errors.As(err, &target)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := newTextOutFromBytes(TypeTextOut, DescTextOut, lib.ActionOutput, []byte(tt.data))
			assertErrContains(t, err, "[type text | action output]")
			if !tt.checkType(err) {
				t.Errorf("unexpected error type: %v", err)
			}
			if conv != nil {
				t.Errorf("expected nil converter on error, got %#v", conv)
			}
		})
	}
}

func TestWithOutputDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		iType string
		dir   string
		want  string
	}{
		{name: "custom dir is trimmed", iType: TypeTextOut, dir: "  ./out  ", want: "./out"},
		{name: "empty dir for text", iType: TypeTextOut, dir: "", want: filepath.Join("output", "text")},
		{name: "empty dir for clashRuleSetClassical", iType: TypeClashRuleSetClassicalOut, dir: "", want: filepath.Join("output", "clash", "classical")},
		{name: "blank dir for clashRuleSet", iType: TypeClashRuleSetIPCIDROut, dir: " \t ", want: filepath.Join("output", "clash", "ipcidr")},
		{name: "blank dir for surgeRuleSet", iType: TypeSurgeRuleSetOut, dir: "  ", want: filepath.Join("output", "surge")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			to := &text_out{Type: tt.iType, OutputDir: "previous"}
			WithOutputDir(tt.dir)(to)
			if to.OutputDir != tt.want {
				t.Errorf("OutputDir = %q, want %q", to.OutputDir, tt.want)
			}
		})
	}
}

func TestWithOutputExtension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ext  string
		want string
	}{
		{name: "empty", ext: "", want: ".txt"},
		{name: "blank", ext: "   ", want: ".txt"},
		{name: "custom", ext: ".yaml", want: ".yaml"},
		{name: "custom trimmed", ext: "  .conf  ", want: ".conf"},
		{name: "without leading dot", ext: "list", want: "list"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			to := &text_out{OutputExt: "previous"}
			WithOutputExtension(tt.ext)(to)
			if to.OutputExt != tt.want {
				t.Errorf("OutputExt = %q, want %q", to.OutputExt, tt.want)
			}
		})
	}
}

func TestTextOut_MarshalBytes(t *testing.T) {
	t.Parallel()

	// Merged result: 1.2.3.4/32, 192.168.0.0/23, 2001:db8::/32
	mixed := []string{
		"192.168.0.3/24",
		"192.168.1.5/24",
		"2001:db8::/33",
		"2001:db8:8000::/33",
		"::ffff:1.2.3.4",
	}

	tests := []struct {
		name       string
		iType      string
		cidrs      []string
		onlyIPType lib.IPType
		prefix     string
		suffix     string
		want       string
		wantErr    string
		wantErrIs  error
	}{
		{
			name:  "text",
			iType: TypeTextOut,
			cidrs: mixed,
			want:  "1.2.3.4/32\n192.168.0.0/23\n2001:db8::/32\n",
		},
		{
			name:   "text with prefix",
			iType:  TypeTextOut,
			cidrs:  mixed,
			prefix: "HOST,",
			want:   "HOST,1.2.3.4/32\nHOST,192.168.0.0/23\nHOST,2001:db8::/32\n",
		},
		{
			name:   "text with suffix",
			iType:  TypeTextOut,
			cidrs:  mixed,
			suffix: ";",
			want:   "1.2.3.4/32;\n192.168.0.0/23;\n2001:db8::/32;\n",
		},
		{
			name:   "text with prefix and suffix",
			iType:  TypeTextOut,
			cidrs:  mixed,
			prefix: "IP-CIDR,",
			suffix: ",no-resolve",
			want:   "IP-CIDR,1.2.3.4/32,no-resolve\nIP-CIDR,192.168.0.0/23,no-resolve\nIP-CIDR,2001:db8::/32,no-resolve\n",
		},
		{
			name:       "text only IPv4",
			iType:      TypeTextOut,
			cidrs:      mixed,
			onlyIPType: lib.IPv4,
			want:       "1.2.3.4/32\n192.168.0.0/23\n",
		},
		{
			name:       "text only IPv6",
			iType:      TypeTextOut,
			cidrs:      mixed,
			onlyIPType: lib.IPv6,
			want:       "2001:db8::/32\n",
		},
		{
			name:       "unknown onlyIPType does not filter",
			iType:      TypeTextOut,
			cidrs:      mixed,
			onlyIPType: lib.IPType("ipv5"),
			want:       "1.2.3.4/32\n192.168.0.0/23\n2001:db8::/32\n",
		},
		{
			name:  "clashRuleSetClassical",
			iType: TypeClashRuleSetClassicalOut,
			cidrs: mixed,
			want:  "payload:\n  - IP-CIDR,1.2.3.4/32\n  - IP-CIDR,192.168.0.0/23\n  - IP-CIDR6,2001:db8::/32\n",
		},
		{
			name:   "clashRuleSetClassical ignores prefix and suffix",
			iType:  TypeClashRuleSetClassicalOut,
			cidrs:  mixed,
			prefix: "HOST,",
			suffix: ",no-resolve",
			want:   "payload:\n  - IP-CIDR,1.2.3.4/32\n  - IP-CIDR,192.168.0.0/23\n  - IP-CIDR6,2001:db8::/32\n",
		},
		{
			name:       "clashRuleSetClassical only IPv6",
			iType:      TypeClashRuleSetClassicalOut,
			cidrs:      mixed,
			onlyIPType: lib.IPv6,
			want:       "payload:\n  - IP-CIDR6,2001:db8::/32\n",
		},
		{
			name:  "clashRuleSet",
			iType: TypeClashRuleSetIPCIDROut,
			cidrs: mixed,
			want:  "payload:\n  - '1.2.3.4/32'\n  - '192.168.0.0/23'\n  - '2001:db8::/32'\n",
		},
		{
			name:   "clashRuleSet ignores prefix and suffix",
			iType:  TypeClashRuleSetIPCIDROut,
			cidrs:  mixed,
			prefix: "HOST,",
			suffix: ",no-resolve",
			want:   "payload:\n  - '1.2.3.4/32'\n  - '192.168.0.0/23'\n  - '2001:db8::/32'\n",
		},
		{
			name:       "clashRuleSet only IPv4",
			iType:      TypeClashRuleSetIPCIDROut,
			cidrs:      mixed,
			onlyIPType: lib.IPv4,
			want:       "payload:\n  - '1.2.3.4/32'\n  - '192.168.0.0/23'\n",
		},
		{
			name:  "surgeRuleSet",
			iType: TypeSurgeRuleSetOut,
			cidrs: mixed,
			want:  "IP-CIDR,1.2.3.4/32\nIP-CIDR,192.168.0.0/23\nIP-CIDR6,2001:db8::/32\n",
		},
		{
			name:   "surgeRuleSet with suffix ignores prefix",
			iType:  TypeSurgeRuleSetOut,
			cidrs:  mixed,
			prefix: "HOST,",
			suffix: ",no-resolve",
			want:   "IP-CIDR,1.2.3.4/32,no-resolve\nIP-CIDR,192.168.0.0/23,no-resolve\nIP-CIDR6,2001:db8::/32,no-resolve\n",
		},
		{
			name:       "surgeRuleSet only IPv4",
			iType:      TypeSurgeRuleSetOut,
			cidrs:      mixed,
			onlyIPType: lib.IPv4,
			want:       "IP-CIDR,1.2.3.4/32\nIP-CIDR,192.168.0.0/23\n",
		},
		{
			name:       "only IPv4 requested but entry has only IPv6",
			iType:      TypeTextOut,
			cidrs:      []string{"2001:db8::/32"},
			onlyIPType: lib.IPv4,
			wantErr:    "has no prefix",
		},
		{
			name:       "only IPv6 requested but entry has only IPv4",
			iType:      TypeSurgeRuleSetOut,
			cidrs:      []string{"1.1.1.1"},
			onlyIPType: lib.IPv6,
			wantErr:    "has no prefix",
		},
		{
			name:    "empty entry",
			iType:   TypeClashRuleSetIPCIDROut,
			cidrs:   nil,
			wantErr: "has no prefix",
		},
		{
			name:      "unsupported type",
			iType:     "unknown",
			cidrs:     mixed,
			wantErrIs: lib.ErrNotSupportedFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			to := &text_out{
				Type:            tt.iType,
				OnlyIPType:      tt.onlyIPType,
				AddPrefixInLine: tt.prefix,
				AddSuffixInLine: tt.suffix,
			}
			got, err := to.marshalBytes(newTestEntry(t, "test", tt.cidrs...))
			assertErr(t, err, tt.wantErr, tt.wantErrIs)
			if tt.wantErr != "" || tt.wantErrIs != nil {
				if got != nil {
					t.Errorf("expected nil data on error, got %q", got)
				}
				return
			}
			if string(got) != tt.want {
				t.Errorf("marshalBytes() mismatch:\ngot:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestTextOut_MarshalBytesForFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		iType   string
		cidrs   []string
		want    string
		wantErr bool
	}{
		{name: "text with no CIDR", iType: TypeTextOut, cidrs: nil, want: ""},
		{name: "clashRuleSetClassical with no CIDR", iType: TypeClashRuleSetClassicalOut, cidrs: nil, want: "payload:\n"},
		{name: "clashRuleSet with no CIDR", iType: TypeClashRuleSetIPCIDROut, cidrs: nil, want: "payload:\n"},
		{name: "surgeRuleSet with no CIDR", iType: TypeSurgeRuleSetOut, cidrs: nil, want: ""},
		{name: "clashRuleSetClassical with invalid CIDR", iType: TypeClashRuleSetClassicalOut, cidrs: []string{"1.1.1.1/32", "1.1.1.1"}, wantErr: true},
		{name: "clashRuleSetClassical with garbage", iType: TypeClashRuleSetClassicalOut, cidrs: []string{"garbage"}, wantErr: true},
		{name: "surgeRuleSet with invalid CIDR", iType: TypeSurgeRuleSetOut, cidrs: []string{"2001:db8::/32", "2001:db8::/129"}, wantErr: true},
		{name: "surgeRuleSet with garbage", iType: TypeSurgeRuleSetOut, cidrs: []string{"garbage"}, wantErr: true},
		{
			name:  "text keeps CIDRs verbatim",
			iType: TypeTextOut,
			cidrs: []string{"anything", "1.1.1.1"},
			want:  "anything\n1.1.1.1\n",
		},
		{
			name:  "clashRuleSet keeps CIDRs verbatim",
			iType: TypeClashRuleSetIPCIDROut,
			cidrs: []string{"anything"},
			want:  "payload:\n  - 'anything'\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			to := &text_out{Type: tt.iType}
			var buf bytes.Buffer
			var err error
			switch tt.iType {
			case TypeTextOut:
				err = to.marshalBytesForTextOut(&buf, tt.cidrs)
			case TypeClashRuleSetClassicalOut:
				err = to.marshalBytesForClashRuleSetClassicalOut(&buf, tt.cidrs)
			case TypeClashRuleSetIPCIDROut:
				err = to.marshalBytesForClashRuleSetIPCIDROut(&buf, tt.cidrs)
			case TypeSurgeRuleSetOut:
				err = to.marshalBytesForSurgeRuleSetOut(&buf, tt.cidrs)
			default:
				t.Fatalf("unexpected type %q", tt.iType)
			}

			if tt.wantErr {
				var parseErr *net.ParseError
				if !errors.As(err, &parseErr) {
					t.Fatalf("expected *net.ParseError, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("output mismatch:\ngot:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestTextOut_WriteFile(t *testing.T) {
	t.Parallel()

	t.Run("creates nested output dir", func(t *testing.T) {
		t.Parallel()

		dir := filepath.Join(t.TempDir(), "nested", "output")
		to := &text_out{Type: TypeTextOut, OutputDir: dir}
		if err := to.writeFile("cn.txt", []byte("1.1.1.1/32\n")); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertFiles(t, readDirFiles(t, dir), map[string]string{"cn.txt": "1.1.1.1/32\n"})
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		writeTestFile(t, dir, "cn.txt", "old content that is longer than the new one\n")
		to := &text_out{Type: TypeTextOut, OutputDir: dir}
		if err := to.writeFile("cn.txt", []byte("new\n")); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertFiles(t, readDirFiles(t, dir), map[string]string{"cn.txt": "new\n"})
	})

	t.Run("writes empty file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		to := &text_out{Type: TypeTextOut, OutputDir: dir}
		if err := to.writeFile("empty.txt", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertFiles(t, readDirFiles(t, dir), map[string]string{"empty.txt": ""})
	})

	t.Run("output dir is a regular file", func(t *testing.T) {
		t.Parallel()

		file := writeTestFile(t, t.TempDir(), "file", "content")
		to := &text_out{Type: TypeTextOut, OutputDir: filepath.Join(file, "sub")}
		if err := to.writeFile("cn.txt", []byte("1.1.1.1/32\n")); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("target file is a directory", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "cn.txt"), 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		to := &text_out{Type: TypeTextOut, OutputDir: dir}
		if err := to.writeFile("cn.txt", []byte("1.1.1.1/32\n")); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

// TestTextOut_WriteFile_Log must not run in parallel since it captures the
// output of the global logger.
func TestTextOut_WriteFile_Log(t *testing.T) {
	logs := captureLog(t)

	dir := t.TempDir()
	to := &text_out{Type: TypeSurgeRuleSetOut, OutputDir: dir}
	if err := to.writeFile("cn.txt", []byte("IP-CIDR,1.1.1.1/32\n")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "✅ [" + TypeSurgeRuleSetOut + "] cn.txt --> " + dir + "\n"
	if got := logs.String(); got != want {
		t.Errorf("log output = %q, want %q", got, want)
	}
}
