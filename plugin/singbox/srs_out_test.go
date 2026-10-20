package singbox

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/sagernet/sing-box/constant"
)

// srsOutSource covers merging of adjacent IPv4/IPv6 prefixes and lists that
// hold only one IP family.
var srsOutSource = map[string][]string{
	"CN":     cnInput,
	"US":     usInput,
	"V4ONLY": v4Input,
	"V6ONLY": {"2606:4700::/32"},
}

var srsOutMerged = map[string][]string{
	"CN":     cnCIDRs,
	"US":     usCIDRs,
	"V4ONLY": v4CIDRs,
	"V6ONLY": {"2606:4700::/32"},
}

func newSRSOut(t *testing.T, opts ...lib.OutputOption) *srs_out {
	t.Helper()
	conv, err := NewSRSOut(lib.ActionOutput, opts...)
	if err != nil {
		t.Fatalf("NewSRSOut: %v", err)
	}
	return conv.(*srs_out)
}

// readSRSDir reads every SRS file of dir back with the input converter.
func readSRSDir(t *testing.T, dir string) lib.Container {
	t.Helper()
	conv, err := NewSRSIn(lib.ActionAdd, WithInputDir(dir))
	if err != nil {
		t.Fatalf("NewSRSIn: %v", err)
	}
	container, err := conv.Input(lib.NewContainer())
	if err != nil {
		t.Fatalf("Input(%s): %v", dir, err)
	}
	return container
}

// requireDevFull skips the test unless /dev/full, whose writes always fail
// with ENOSPC, is available.
func requireDevFull(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("/dev/full is only used on linux")
	}
	if _, err := os.Stat("/dev/full"); err != nil {
		t.Skipf("/dev/full is not available: %v", err)
	}
}

func TestNewSRSOut(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		conv, err := NewSRSOut(lib.ActionOutput, nil)
		if err != nil {
			t.Fatalf("NewSRSOut: %v", err)
		}
		out := conv.(*srs_out)
		if out.OutputDir != defaultOutputDir || out.OutputDir != filepath.Join("output", "srs") {
			t.Errorf("OutputDir: got %q, want %q", out.OutputDir, defaultOutputDir)
		}
		if out.Want != nil || out.Exclude != nil || out.OnlyIPType != "" {
			t.Errorf("unexpected non-zero options: %+v", out)
		}
		if got := conv.GetType(); got != TypeSRSOut {
			t.Errorf("GetType: got %q, want %q", got, TypeSRSOut)
		}
		if got := conv.GetAction(); got != lib.ActionOutput {
			t.Errorf("GetAction: got %q, want %q", got, lib.ActionOutput)
		}
		if got := conv.GetDescription(); got != DescSRSOut {
			t.Errorf("GetDescription: got %q, want %q", got, DescSRSOut)
		}
	})

	t.Run("all options", func(t *testing.T) {
		t.Parallel()

		out := newSRSOut(t,
			WithOutputDir(" out/dir "),
			WithOutputWantedList([]string{"cn"}),
			WithOutputExcludedList([]string{"us"}),
			WithOutputOnlyIPType(lib.IPv4),
		)
		if out.OutputDir != "out/dir" {
			t.Errorf("OutputDir: got %q, want %q", out.OutputDir, "out/dir")
		}
		assertStrings(t, "Want", out.Want, []string{"cn"})
		assertStrings(t, "Exclude", out.Exclude, []string{"us"})
		if out.OnlyIPType != lib.IPv4 {
			t.Errorf("OnlyIPType: got %q, want %q", out.OnlyIPType, lib.IPv4)
		}
	})
}

func TestSRSOut_WithOutputDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dir  string
		want string
	}{
		{name: "empty uses default", dir: "", want: defaultOutputDir},
		{name: "blank uses default", dir: " \t\n", want: defaultOutputDir},
		{name: "value is trimmed", dir: "  custom  ", want: "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			out := &srs_out{}
			WithOutputDir(tt.dir)(out)
			if out.OutputDir != tt.want {
				t.Errorf("got %q, want %q", out.OutputDir, tt.want)
			}
		})
	}
}

func TestNewSRSOutFromBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		data      string
		wantErrAs any
		want      srs_out
	}{
		{name: "invalid json", data: "{", wantErrAs: new(*json.SyntaxError)},
		{name: "wrong field type", data: `{"wantedList": "cn"}`, wantErrAs: new(*json.UnmarshalTypeError)},
		{name: "empty data uses defaults", data: "", want: srs_out{OutputDir: defaultOutputDir}},
		{name: "blank output dir uses default", data: `{"outputDir": "  "}`, want: srs_out{OutputDir: defaultOutputDir}},
		{
			name: "all fields",
			data: `{"outputDir": " /x/y ", "wantedList": ["cn", "us"], "excludedList": ["jp"], "onlyIPType": "ipv6"}`,
			want: srs_out{OutputDir: "/x/y", Want: []string{"cn", "us"}, Exclude: []string{"jp"}, OnlyIPType: lib.IPv6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewSRSOutFromBytes(lib.ActionOutput, []byte(tt.data))
			if tt.wantErrAs != nil {
				assertErrorContains(t, err, "[type singboxSRS | action output]")
				if !errors.As(err, tt.wantErrAs) {
					t.Errorf("error %v is not %T", err, tt.wantErrAs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			out := conv.(*srs_out)
			if out.Type != TypeSRSOut || out.Action != lib.ActionOutput || out.Description != DescSRSOut {
				t.Errorf("unexpected identity: %+v", out)
			}
			if out.OutputDir != tt.want.OutputDir {
				t.Errorf("OutputDir: got %q, want %q", out.OutputDir, tt.want.OutputDir)
			}
			assertStrings(t, "Want", out.Want, tt.want.Want)
			assertStrings(t, "Exclude", out.Exclude, tt.want.Exclude)
			if out.OnlyIPType != tt.want.OnlyIPType {
				t.Errorf("OnlyIPType: got %q, want %q", out.OnlyIPType, tt.want.OnlyIPType)
			}
		})
	}
}

func TestSRSOut_filterAndSortList(t *testing.T) {
	t.Parallel()

	lists := map[string][]string{
		"US": {"8.8.8.0/24"},
		"CN": {"1.0.1.0/24"},
		"JP": {"1.0.16.0/20"},
		"EU": {"2.0.0.0/16"},
	}

	tests := []struct {
		name    string
		want    []string
		exclude []string
		result  []string
	}{
		{name: "all lists sorted", result: []string{"CN", "EU", "JP", "US"}},
		{name: "excluded lists are normalized", exclude: []string{" us ", "", "jp", "NOTEXIST"}, result: []string{"CN", "EU"}},
		{name: "wanted lists are normalized and sorted", want: []string{"us", " cn ", "", "notexist"}, result: []string{"CN", "NOTEXIST", "US"}},
		{name: "blank wanted lists mean no wanted list", want: []string{"", "  "}, exclude: []string{"eu"}, result: []string{"CN", "JP", "US"}},
		{name: "wanted minus excluded", want: []string{"us", "cn"}, exclude: []string{"CN"}, result: []string{"US"}},
		{name: "wanted list fully excluded", want: []string{"cn", "us"}, exclude: []string{"us", "CN"}, result: []string{}},
		{name: "all lists excluded", exclude: []string{"cn", "us", "jp", "eu"}, result: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			out := newSRSOut(t, WithOutputWantedList(tt.want), WithOutputExcludedList(tt.exclude))
			got := out.filterAndSortList(newTestContainer(t, lists))
			if got == nil {
				t.Fatal("got nil list, want non-nil")
			}
			assertStrings(t, "lists", got, tt.result)
		})
	}
}

func TestSRSOut_marshalRuleSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cidrs      []string
		onlyIPType lib.IPType
		want       []string
		wantErr    string
	}{
		{name: "both families", cidrs: cnInput, want: cnCIDRs},
		{name: "only ipv4", cidrs: cnInput, onlyIPType: lib.IPv4, want: []string{"192.168.0.0/23"}},
		{name: "only ipv6", cidrs: cnInput, onlyIPType: lib.IPv6, want: []string{"2001:db8::/32"}},
		{name: "ipv4-mapped prefixes are stored as ipv4", cidrs: []string{"::ffff:10.0.0.0/104", "2001:db8::1"}, want: []string{"10.0.0.0/8", "2001:db8::1/128"}},
		{
			name:  "ipv4-mapped prefixes are masked and merged",
			cidrs: []string{"::ffff:192.168.0.5/120", "::ffff:192.168.1.0/120", "::ffff:8.8.8.8/128"},
			want:  []string{"8.8.8.8/32", "192.168.0.0/23"},
		},
		{name: "whole ipv4-mapped space", cidrs: []string{"::ffff:0:0/96"}, onlyIPType: lib.IPv4, want: []string{"0.0.0.0/0"}},
		{name: "no prefix", cidrs: nil, wantErr: "TEST"},
		{name: "no prefix of the wanted ip type", cidrs: v4Input, onlyIPType: lib.IPv6, wantErr: "TEST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entry := lib.NewEntry("test")
			for _, cidr := range tt.cidrs {
				if err := entry.AddPrefix(cidr); err != nil {
					t.Fatalf("AddPrefix(%q): %v", cidr, err)
				}
			}

			ruleSet, err := newSRSOut(t, WithOutputOnlyIPType(tt.onlyIPType)).marshalRuleSet(entry)
			if tt.wantErr != "" {
				assertErrorContains(t, err, tt.wantErr)
				if ruleSet != nil {
					t.Errorf("got rule-set %+v, want nil", ruleSet)
				}
				return
			}
			if err != nil {
				t.Fatalf("marshalRuleSet: %v", err)
			}
			if len(ruleSet.Rules) != 1 {
				t.Fatalf("got %d rules, want 1", len(ruleSet.Rules))
			}
			rule := ruleSet.Rules[0]
			if rule.Type != constant.RuleTypeDefault {
				t.Errorf("rule type: got %q, want %q", rule.Type, constant.RuleTypeDefault)
			}
			assertStrings(t, "ip_cidr", rule.DefaultOptions.IPCIDR, tt.want)
		})
	}
}

func TestSRSOut_Output_RoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		opts      []lib.OutputOption
		wantFiles []string
		want      map[string][]string
	}{
		{
			name:      "all lists",
			wantFiles: []string{"cn.srs", "us.srs", "v4only.srs", "v6only.srs"},
			want:      srsOutMerged,
		},
		{
			name:      "wanted lists",
			opts:      []lib.OutputOption{WithOutputWantedList([]string{" us", "cn ", ""})},
			wantFiles: []string{"cn.srs", "us.srs"},
			want:      pick(srsOutMerged, "CN", "US"),
		},
		{
			name:      "excluded lists",
			opts:      []lib.OutputOption{WithOutputExcludedList([]string{"v4only", " V6ONLY "})},
			wantFiles: []string{"cn.srs", "us.srs"},
			want:      pick(srsOutMerged, "CN", "US"),
		},
		{
			name:      "wanted minus excluded",
			opts:      []lib.OutputOption{WithOutputWantedList([]string{"cn", "us"}), WithOutputExcludedList([]string{"us"})},
			wantFiles: []string{"cn.srs"},
			want:      pick(srsOutMerged, "CN"),
		},
		{
			name:      "only ipv4",
			opts:      []lib.OutputOption{WithOutputExcludedList([]string{"v6only"}), WithOutputOnlyIPType(lib.IPv4)},
			wantFiles: []string{"cn.srs", "us.srs", "v4only.srs"},
			want: map[string][]string{
				"CN":     {"192.168.0.0/23"},
				"US":     {"10.0.0.0/7"},
				"V4ONLY": v4CIDRs,
			},
		},
		{
			name:      "only ipv6",
			opts:      []lib.OutputOption{WithOutputWantedList([]string{"cn", "us", "v6only"}), WithOutputOnlyIPType(lib.IPv6)},
			wantFiles: []string{"cn.srs", "us.srs", "v6only.srs"},
			want: map[string][]string{
				"CN":     {"2001:db8::/32"},
				"US":     {"2400:cb00::/32"},
				"V6ONLY": {"2606:4700::/32"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			out := newSRSOut(t, append([]lib.OutputOption{WithOutputDir(dir)}, tt.opts...)...)
			if err := out.Output(newTestContainer(t, srsOutSource)); err != nil {
				t.Fatalf("Output: %v", err)
			}
			assertStrings(t, "written files", dirFileNames(t, dir), tt.wantFiles)
			assertContainer(t, readSRSDir(t, dir), tt.want)
		})
	}
}

func TestSRSOut_Output_RawContent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := newSRSOut(t, WithOutputDir(dir)).Output(newTestContainer(t, pick(srsOutSource, "CN"))); err != nil {
		t.Fatalf("Output: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "cn.srs"))
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte{'S', 'R', 'S', constant.RuleSetVersion1}; !bytes.HasPrefix(data, want) {
		t.Fatalf("header: got %v, want %v", data[:min(len(data), len(want))], want)
	}

	ruleSet := readSRS(t, data)
	if ruleSet.Version != constant.RuleSetVersion1 {
		t.Errorf("version: got %d, want %d", ruleSet.Version, constant.RuleSetVersion1)
	}
	if len(ruleSet.Options.Rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(ruleSet.Options.Rules))
	}
	rule := ruleSet.Options.Rules[0]
	if rule.Type != constant.RuleTypeDefault {
		t.Errorf("rule type: got %q, want %q", rule.Type, constant.RuleTypeDefault)
	}
	if rule.DefaultOptions.Invert {
		t.Error("rule is inverted")
	}
	assertStrings(t, "ip_cidr", rule.DefaultOptions.IPCIDR, cnCIDRs)

	// The output must be reproducible.
	dir2 := t.TempDir()
	if err := newSRSOut(t, WithOutputDir(dir2)).Output(newTestContainer(t, pick(srsOutSource, "CN"))); err != nil {
		t.Fatalf("Output: %v", err)
	}
	data2, err := os.ReadFile(filepath.Join(dir2, "cn.srs"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, data2) {
		t.Error("two outputs of the same list differ")
	}
}

func TestSRSOut_Output_OverwritesExistingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "cn.srs"), bytes.Repeat([]byte("stale content "), 100))
	if err := newSRSOut(t, WithOutputDir(dir)).Output(newTestContainer(t, pick(srsOutSource, "CN"))); err != nil {
		t.Fatalf("Output: %v", err)
	}
	assertContainer(t, readSRSDir(t, dir), pick(srsOutMerged, "CN"))
}

func TestSRSOut_Output_NothingWritten(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		lists map[string][]string
		opts  []lib.OutputOption
	}{
		{name: "empty container", lists: nil},
		{name: "all lists excluded", lists: srsOutSource, opts: []lib.OutputOption{WithOutputExcludedList([]string{"cn", "us", "v4only", "v6only"})}},
		{name: "wanted list fully excluded", lists: srsOutSource, opts: []lib.OutputOption{WithOutputWantedList([]string{"cn"}), WithOutputExcludedList([]string{"CN"})}},
		{name: "wanted list not in container", lists: srsOutSource, opts: []lib.OutputOption{WithOutputWantedList([]string{"notexist"})}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := filepath.Join(t.TempDir(), "out")
			opts := append([]lib.OutputOption{WithOutputDir(dir)}, tt.opts...)
			if err := newSRSOut(t, opts...).Output(newTestContainer(t, tt.lists)); err != nil {
				t.Fatalf("Output: %v", err)
			}
			if _, err := os.Stat(dir); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("output dir should not be created, stat error: %v", err)
			}
		})
	}
}

// TestSRSOut_Output_EntryNotFound captures the log output, so it must not run
// in parallel.
func TestSRSOut_Output_EntryNotFound(t *testing.T) {
	logs := captureLog(t)

	dir := t.TempDir()
	out := newSRSOut(t, WithOutputDir(dir), WithOutputWantedList([]string{"cn", "notexist"}))
	if err := out.Output(newTestContainer(t, srsOutSource)); err != nil {
		t.Fatalf("Output: %v", err)
	}
	if !bytes.Contains(logs.Bytes(), []byte("entry NOTEXIST not found")) {
		t.Errorf("log %q does not report the missing list", logs.String())
	}
	if !bytes.Contains(logs.Bytes(), []byte("cn.srs --> "+dir)) {
		t.Errorf("log %q does not report the written file", logs.String())
	}
	assertStrings(t, "written files", dirFileNames(t, dir), []string{"cn.srs"})
	assertContainer(t, readSRSDir(t, dir), pick(srsOutMerged, "CN"))
}

func TestSRSOut_Output_Errors(t *testing.T) {
	t.Parallel()

	emptyContainer := func(t *testing.T) lib.Container {
		t.Helper()
		container := lib.NewContainer()
		if err := container.Add(lib.NewEntry("empty")); err != nil {
			t.Fatal(err)
		}
		return container
	}
	sourceContainer := func(t *testing.T) lib.Container {
		t.Helper()
		return newTestContainer(t, srsOutSource)
	}

	tests := []struct {
		name      string
		container func(*testing.T) lib.Container
		// setup prepares the output dir and returns the options to use.
		setup       func(t *testing.T, dir string) []lib.OutputOption
		wantErr     string
		wantPathErr bool
	}{
		{
			name:      "list without cidr",
			container: emptyContainer,
			setup: func(t *testing.T, dir string) []lib.OutputOption {
				return []lib.OutputOption{WithOutputDir(dir)}
			},
			wantErr: "EMPTY",
		},
		{
			name:      "list without cidr of the wanted ip type",
			container: sourceContainer,
			setup: func(t *testing.T, dir string) []lib.OutputOption {
				return []lib.OutputOption{WithOutputDir(dir), WithOutputWantedList([]string{"v4only"}), WithOutputOnlyIPType(lib.IPv6)}
			},
			wantErr: "V4ONLY",
		},
		{
			name:      "output dir is a regular file",
			container: sourceContainer,
			setup: func(t *testing.T, dir string) []lib.OutputOption {
				file := writeFile(t, filepath.Join(dir, "file"), []byte("x"))
				return []lib.OutputOption{WithOutputDir(filepath.Join(file, "sub"))}
			},
			wantPathErr: true,
		},
		{
			name:      "output file is a directory",
			container: sourceContainer,
			setup: func(t *testing.T, dir string) []lib.OutputOption {
				if err := os.Mkdir(filepath.Join(dir, "us.srs"), 0o755); err != nil {
					t.Fatal(err)
				}
				return []lib.OutputOption{WithOutputDir(dir)}
			},
			wantPathErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := tt.setup(t, t.TempDir())
			err := newSRSOut(t, opts...).Output(tt.container(t))
			assertErrorContains(t, err, "[type singboxSRS | action output]", tt.wantErr)
			var pathErr *fs.PathError
			if tt.wantPathErr && !errors.As(err, &pathErr) {
				t.Errorf("error %v is not a *fs.PathError", err)
			}
		})
	}

	t.Run("disk full", func(t *testing.T) {
		t.Parallel()
		requireDevFull(t)

		dir := t.TempDir()
		if err := os.Symlink("/dev/full", filepath.Join(dir, "cn.srs")); err != nil {
			t.Skipf("cannot create symlink: %v", err)
		}
		err := newSRSOut(t, WithOutputDir(dir)).Output(newTestContainer(t, pick(srsOutSource, "CN")))
		assertErrorContains(t, err, "[type singboxSRS | action output]")
	})
}

// TestSRSOut_Registration checks the converter is reachable through the lib
// config registry set up in init.
func TestSRSOut_Registration(t *testing.T) {
	t.Parallel()

	if err := lib.RegisterOutputConverter(TypeSRSOut, &srs_out{}); !errors.Is(err, lib.ErrDuplicatedConverter) {
		t.Errorf("RegisterOutputConverter: got %v, want %v", err, lib.ErrDuplicatedConverter)
	}
	if err := lib.RegisterOutputConfigCreator(TypeSRSOut, nil); err == nil {
		t.Error("RegisterOutputConfigCreator: got nil error for an already registered type")
	}

	t.Run("valid args", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		args, err := json.Marshal(map[string]any{"outputDir": dir, "wantedList": []string{"cn"}, "onlyIPType": "ipv6"})
		if err != nil {
			t.Fatal(err)
		}
		instance, err := lib.NewInstance()
		if err != nil {
			t.Fatal(err)
		}
		if err := instance.InitConfigFromBytes([]byte(`{"output": [{"type": "singboxSRS", "action": "output", "args": ` + string(args) + `}]}`)); err != nil {
			t.Fatalf("InitConfigFromBytes: %v", err)
		}
		if err := instance.RunOutput(newTestContainer(t, srsOutSource)); err != nil {
			t.Fatalf("RunOutput: %v", err)
		}
		assertContainer(t, readSRSDir(t, dir), map[string][]string{"CN": {"2001:db8::/32"}})
	})

	t.Run("invalid args", func(t *testing.T) {
		t.Parallel()

		instance, err := lib.NewInstance()
		if err != nil {
			t.Fatal(err)
		}
		if err := instance.InitConfigFromBytes([]byte(`{"output": [{"type": "singboxSRS", "args": {"outputDir": 1}}]}`)); err == nil {
			t.Fatal("got nil error, want error")
		}
	})
}
