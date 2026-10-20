package mihomo

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"go4.org/netipx"
)

// mrsOutSource covers merging of adjacent IPv4/IPv6 prefixes and lists that
// hold only one IP family.
var mrsOutSource = map[string][]string{
	"CN":     {"192.168.0.3/24", "192.168.1.5/24", "2001:db8::/33", "2001:db8:8000::/33"},
	"US":     {"10.0.0.0/8", "11.0.0.0/8", "2400:cb00::/32"},
	"V4ONLY": {"1.1.1.1/24"},
	"V6ONLY": {"2606:4700::/32"},
}

var mrsOutMerged = map[string][]string{
	"CN":     {"192.168.0.0/23", "2001:db8::/32"},
	"US":     {"10.0.0.0/7", "2400:cb00::/32"},
	"V4ONLY": {"1.1.1.0/24"},
	"V6ONLY": {"2606:4700::/32"},
}

func newMRSOut(t *testing.T, opts ...lib.OutputOption) *mrs_out {
	t.Helper()
	conv, err := NewMRSOut(lib.ActionOutput, opts...)
	if err != nil {
		t.Fatalf("NewMRSOut: %v", err)
	}
	return conv.(*mrs_out)
}

// readMRSDir reads every MRS file of dir back with the input converter.
func readMRSDir(t *testing.T, dir string) lib.Container {
	t.Helper()
	conv, err := NewMRSIn(lib.ActionAdd, WithInputDir(dir))
	if err != nil {
		t.Fatalf("NewMRSIn: %v", err)
	}
	container, err := conv.Input(lib.NewContainer())
	if err != nil {
		t.Fatalf("Input(%s): %v", dir, err)
	}
	return container
}

// manyRanges returns n disjoint single-address IPv4 ranges, enough to make
// the zstd encoder flush a block (and hit the writer) while ranges are written.
func manyRanges(n int) []netipx.IPRange {
	ranges := make([]netipx.IPRange, 0, n)
	addr := netip.MustParseAddr("10.0.0.0")
	for range n {
		ranges = append(ranges, netipx.IPRangeFrom(addr, addr))
		addr = addr.Next().Next()
	}
	return ranges
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

func TestNewMRSOut(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		conv, err := NewMRSOut(lib.ActionOutput, nil)
		if err != nil {
			t.Fatalf("NewMRSOut: %v", err)
		}
		out := conv.(*mrs_out)
		if out.OutputDir != defaultOutputDir || out.OutputDir != filepath.Join("output", "mrs") {
			t.Errorf("OutputDir: got %q, want %q", out.OutputDir, defaultOutputDir)
		}
		if out.Want != nil || out.Exclude != nil || out.OnlyIPType != "" {
			t.Errorf("unexpected non-zero options: %+v", out)
		}
		if got := conv.GetType(); got != TypeMRSOut {
			t.Errorf("GetType: got %q, want %q", got, TypeMRSOut)
		}
		if got := conv.GetAction(); got != lib.ActionOutput {
			t.Errorf("GetAction: got %q, want %q", got, lib.ActionOutput)
		}
		if got := conv.GetDescription(); got != DescMRSOut {
			t.Errorf("GetDescription: got %q, want %q", got, DescMRSOut)
		}
	})

	t.Run("all options", func(t *testing.T) {
		t.Parallel()

		out := newMRSOut(t,
			WithOutputDir(" out/dir "),
			WithOutputWantedList([]string{"cn"}),
			WithOutputExcludedList([]string{"us"}),
			WithOutputOnlyIPType(lib.IPv6),
		)
		if out.OutputDir != "out/dir" {
			t.Errorf("OutputDir: got %q, want %q", out.OutputDir, "out/dir")
		}
		assertStrings(t, "Want", out.Want, []string{"cn"})
		assertStrings(t, "Exclude", out.Exclude, []string{"us"})
		if out.OnlyIPType != lib.IPv6 {
			t.Errorf("OnlyIPType: got %q, want %q", out.OnlyIPType, lib.IPv6)
		}
	})
}

func TestMRSOut_WithOutputDir(t *testing.T) {
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

			out := &mrs_out{}
			WithOutputDir(tt.dir)(out)
			if out.OutputDir != tt.want {
				t.Errorf("got %q, want %q", out.OutputDir, tt.want)
			}
		})
	}
}

func TestNewMRSOutFromBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		data      string
		wantErrAs any
		want      mrs_out
	}{
		{name: "invalid json", data: "{", wantErrAs: new(*json.SyntaxError)},
		{name: "wrong field type", data: `{"excludedList": "cn"}`, wantErrAs: new(*json.UnmarshalTypeError)},
		{name: "empty data uses defaults", data: "", want: mrs_out{OutputDir: defaultOutputDir}},
		{name: "blank output dir uses default", data: `{"outputDir": "  "}`, want: mrs_out{OutputDir: defaultOutputDir}},
		{
			name: "all fields",
			data: `{"outputDir": " /x/y ", "wantedList": ["cn", "us"], "excludedList": ["jp"], "onlyIPType": "ipv4"}`,
			want: mrs_out{OutputDir: "/x/y", Want: []string{"cn", "us"}, Exclude: []string{"jp"}, OnlyIPType: lib.IPv4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewMRSOutFromBytes(lib.ActionOutput, []byte(tt.data))
			if tt.wantErrAs != nil {
				assertErrorContains(t, err, "[type mihomoMRS | action output]")
				if !errors.As(err, tt.wantErrAs) {
					t.Errorf("error %v is not %T", err, tt.wantErrAs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			out := conv.(*mrs_out)
			if out.Type != TypeMRSOut || out.Action != lib.ActionOutput || out.Description != DescMRSOut {
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

func TestMRSOut_filterAndSortList(t *testing.T) {
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

			out := newMRSOut(t, WithOutputWantedList(tt.want), WithOutputExcludedList(tt.exclude))
			got := out.filterAndSortList(newTestContainer(t, lists))
			if got == nil {
				t.Fatal("got nil list, want non-nil")
			}
			assertStrings(t, "lists", got, tt.result)
		})
	}
}

func TestMRSOut_Output_RoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		opts      []lib.OutputOption
		wantFiles []string
		want      map[string][]string
	}{
		{
			name:      "all lists",
			wantFiles: []string{"cn.mrs", "us.mrs", "v4only.mrs", "v6only.mrs"},
			want:      mrsOutMerged,
		},
		{
			name:      "wanted lists",
			opts:      []lib.OutputOption{WithOutputWantedList([]string{" us", "cn ", ""})},
			wantFiles: []string{"cn.mrs", "us.mrs"},
			want:      pick(mrsOutMerged, "CN", "US"),
		},
		{
			name:      "excluded lists",
			opts:      []lib.OutputOption{WithOutputExcludedList([]string{"v4only", " V6ONLY "})},
			wantFiles: []string{"cn.mrs", "us.mrs"},
			want:      pick(mrsOutMerged, "CN", "US"),
		},
		{
			name:      "wanted minus excluded",
			opts:      []lib.OutputOption{WithOutputWantedList([]string{"cn", "us"}), WithOutputExcludedList([]string{"us"})},
			wantFiles: []string{"cn.mrs"},
			want:      pick(mrsOutMerged, "CN"),
		},
		{
			name:      "only ipv4",
			opts:      []lib.OutputOption{WithOutputExcludedList([]string{"v6only"}), WithOutputOnlyIPType(lib.IPv4)},
			wantFiles: []string{"cn.mrs", "us.mrs", "v4only.mrs"},
			want: map[string][]string{
				"CN":     {"192.168.0.0/23"},
				"US":     {"10.0.0.0/7"},
				"V4ONLY": {"1.1.1.0/24"},
			},
		},
		{
			name:      "only ipv6",
			opts:      []lib.OutputOption{WithOutputWantedList([]string{"cn", "us", "v6only"}), WithOutputOnlyIPType(lib.IPv6)},
			wantFiles: []string{"cn.mrs", "us.mrs", "v6only.mrs"},
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
			out := newMRSOut(t, append([]lib.OutputOption{WithOutputDir(dir)}, tt.opts...)...)
			if err := out.Output(newTestContainer(t, mrsOutSource)); err != nil {
				t.Fatalf("Output: %v", err)
			}
			assertStrings(t, "written files", dirFileNames(t, dir), tt.wantFiles)
			assertContainer(t, readMRSDir(t, dir), tt.want)
		})
	}
}

func TestMRSOut_Output_RawContent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := newMRSOut(t, WithOutputDir(dir)).Output(newTestContainer(t, pick(mrsOutSource, "CN"))); err != nil {
		t.Fatalf("Output: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "cn.mrs"))
	if err != nil {
		t.Fatal(err)
	}

	decoded := decodeMRS(t, data)
	want := [][2]netip.Addr{
		// IPv4 is stored as IPv4-mapped IPv6.
		{netip.MustParseAddr("::ffff:192.168.0.0"), netip.MustParseAddr("::ffff:192.168.1.255")},
		{netip.MustParseAddr("2001:db8::"), netip.MustParseAddr("2001:db8:ffff:ffff:ffff:ffff:ffff:ffff")},
	}
	if decoded.count != int64(len(want)) {
		t.Errorf("count: got %d, want %d", decoded.count, len(want))
	}
	if len(decoded.ranges) != len(want) {
		t.Fatalf("got %d ranges, want %d", len(decoded.ranges), len(want))
	}
	for i := range want {
		if decoded.ranges[i] != want[i] {
			t.Errorf("range[%d]: got %v, want %v", i, decoded.ranges[i], want[i])
		}
	}
}

func TestMRSOut_Output_NothingWritten(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		lists map[string][]string
		opts  []lib.OutputOption
	}{
		{name: "empty container", lists: nil},
		{name: "all lists excluded", lists: mrsOutSource, opts: []lib.OutputOption{WithOutputExcludedList([]string{"cn", "us", "v4only", "v6only"})}},
		{name: "wanted list fully excluded", lists: mrsOutSource, opts: []lib.OutputOption{WithOutputWantedList([]string{"cn"}), WithOutputExcludedList([]string{"CN"})}},
		{name: "wanted list not in container", lists: mrsOutSource, opts: []lib.OutputOption{WithOutputWantedList([]string{"notexist"})}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := filepath.Join(t.TempDir(), "out")
			opts := append([]lib.OutputOption{WithOutputDir(dir)}, tt.opts...)
			if err := newMRSOut(t, opts...).Output(newTestContainer(t, tt.lists)); err != nil {
				t.Fatalf("Output: %v", err)
			}
			if _, err := os.Stat(dir); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("output dir should not be created, stat error: %v", err)
			}
		})
	}
}

// TestMRSOut_Output_EntryNotFound captures the log output, so it must not run
// in parallel.
func TestMRSOut_Output_EntryNotFound(t *testing.T) {
	logs := captureLog(t)

	dir := t.TempDir()
	out := newMRSOut(t, WithOutputDir(dir), WithOutputWantedList([]string{"cn", "notexist"}))
	if err := out.Output(newTestContainer(t, mrsOutSource)); err != nil {
		t.Fatalf("Output: %v", err)
	}
	if !bytes.Contains(logs.Bytes(), []byte("entry NOTEXIST not found")) {
		t.Errorf("log %q does not report the missing list", logs.String())
	}
	if !bytes.Contains(logs.Bytes(), []byte("cn.mrs --> "+dir)) {
		t.Errorf("log %q does not report the written file", logs.String())
	}
	assertStrings(t, "written files", dirFileNames(t, dir), []string{"cn.mrs"})
	assertContainer(t, readMRSDir(t, dir), pick(mrsOutMerged, "CN"))
}

func TestMRSOut_Output_Errors(t *testing.T) {
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
		return newTestContainer(t, mrsOutSource)
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
				return []lib.OutputOption{WithOutputDir(dir), WithOutputWantedList([]string{"v6only"}), WithOutputOnlyIPType(lib.IPv4)}
			},
			wantErr: "V6ONLY",
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
				if err := os.Mkdir(filepath.Join(dir, "us.mrs"), 0o755); err != nil {
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
			err := newMRSOut(t, opts...).Output(tt.container(t))
			assertErrorContains(t, err, "[type mihomoMRS | action output]", tt.wantErr)
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
		if err := os.Symlink("/dev/full", filepath.Join(dir, "big.mrs")); err != nil {
			t.Skipf("cannot create symlink: %v", err)
		}
		entry := lib.NewEntry("big")
		for _, r := range manyRanges(5000) {
			if err := entry.AddPrefix(r.From()); err != nil {
				t.Fatal(err)
			}
		}
		container := lib.NewContainer()
		if err := container.Add(entry); err != nil {
			t.Fatal(err)
		}

		err := newMRSOut(t, WithOutputDir(dir)).Output(container)
		assertErrorContains(t, err, "[type mihomoMRS | action output]")
	})
}

func TestMRSOut_convertToMrs(t *testing.T) {
	t.Parallel()

	t.Run("encodes ranges", func(t *testing.T) {
		t.Parallel()

		ranges := []netipx.IPRange{
			netipx.IPRangeFrom(netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("1.1.1.10")),
			netipx.IPRangeFrom(netip.MustParseAddr("2001:db8::"), netip.MustParseAddr("2001:db8::ff")),
		}
		var buf bytes.Buffer
		if err := (&mrs_out{}).convertToMrs(ranges, &buf); err != nil {
			t.Fatalf("convertToMrs: %v", err)
		}

		decoded := decodeMRS(t, buf.Bytes())
		if decoded.count != int64(len(ranges)) {
			t.Errorf("count: got %d, want %d", decoded.count, len(ranges))
		}
		if len(decoded.ranges) != len(ranges) {
			t.Fatalf("got %d ranges, want %d", len(decoded.ranges), len(ranges))
		}
		for i, r := range ranges {
			if decoded.ranges[i][0].Unmap() != r.From() || decoded.ranges[i][1].Unmap() != r.To() {
				t.Errorf("range[%d]: got %v, want %v-%v", i, decoded.ranges[i], r.From(), r.To())
			}
		}

		entry := lib.NewEntry("test")
		if err := (&mrs_in{}).parseMRS(buf.Bytes(), entry); err != nil {
			t.Fatalf("parseMRS: %v", err)
		}
		assertStrings(t, "CIDRs", entryText(entry), []string{"1.1.1.1/32", "1.1.1.2/31", "1.1.1.4/30", "1.1.1.8/31", "1.1.1.10/32", "2001:db8::/120"})
	})

	t.Run("many ranges", func(t *testing.T) {
		t.Parallel()

		ranges := manyRanges(5000)
		var buf bytes.Buffer
		if err := (&mrs_out{}).convertToMrs(ranges, &buf); err != nil {
			t.Fatalf("convertToMrs: %v", err)
		}
		decoded := decodeMRS(t, buf.Bytes())
		if len(decoded.ranges) != len(ranges) {
			t.Fatalf("got %d ranges, want %d", len(decoded.ranges), len(ranges))
		}
		for i, r := range ranges {
			if decoded.ranges[i][0].Unmap() != r.From() || decoded.ranges[i][1].Unmap() != r.To() {
				t.Fatalf("range[%d]: got %v, want %v-%v", i, decoded.ranges[i], r.From(), r.To())
			}
		}
	})

	t.Run("writer failure", func(t *testing.T) {
		t.Parallel()

		// The encoder buffers small outputs until Close, so enough ranges are
		// needed to surface the writer error while ranges are encoded.
		err := (&mrs_out{}).convertToMrs(manyRanges(5000), errWriter{})
		assertErrorContains(t, err, "forced write failure")
	})
}

// TestMRSOut_Registration checks the converter is reachable through the lib
// config registry set up in init.
func TestMRSOut_Registration(t *testing.T) {
	t.Parallel()

	if err := lib.RegisterOutputConverter(TypeMRSOut, &mrs_out{}); !errors.Is(err, lib.ErrDuplicatedConverter) {
		t.Errorf("RegisterOutputConverter: got %v, want %v", err, lib.ErrDuplicatedConverter)
	}
	if err := lib.RegisterOutputConfigCreator(TypeMRSOut, nil); err == nil {
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
		if err := instance.InitConfigFromBytes([]byte(`{"output": [{"type": "mihomoMRS", "action": "output", "args": ` + string(args) + `}]}`)); err != nil {
			t.Fatalf("InitConfigFromBytes: %v", err)
		}
		if err := instance.RunOutput(newTestContainer(t, mrsOutSource)); err != nil {
			t.Fatalf("RunOutput: %v", err)
		}
		assertContainer(t, readMRSDir(t, dir), map[string][]string{"CN": {"2001:db8::/32"}})
	})

	t.Run("invalid args", func(t *testing.T) {
		t.Parallel()

		instance, err := lib.NewInstance()
		if err != nil {
			t.Fatal(err)
		}
		if err := instance.InitConfigFromBytes([]byte(`{"output": [{"type": "mihomoMRS", "args": {"outputDir": 1}}]}`)); err == nil {
			t.Fatal("got nil error, want error")
		}
	})
}
