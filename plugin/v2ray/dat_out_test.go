package v2ray

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

// datOutSource covers merging of adjacent IPv4/IPv6 prefixes and lists that
// hold only one IP family.
var datOutSource = map[string][]string{
	"CN":     {"192.168.0.3/24", "192.168.1.5/24", "2001:db8::/33", "2001:db8:8000::/33"},
	"US":     {"10.0.0.0/8", "11.0.0.0/8", "2400:cb00::/32"},
	"V4ONLY": {"1.1.1.1/24"},
	"V6ONLY": {"2606:4700::/32"},
}

var datOutMerged = map[string][]string{
	"CN":     {"192.168.0.0/23", "2001:db8::/32"},
	"US":     {"10.0.0.0/7", "2400:cb00::/32"},
	"V4ONLY": {"1.1.1.0/24"},
	"V6ONLY": {"2606:4700::/32"},
}

func pick(lists map[string][]string, names ...string) map[string][]string {
	out := make(map[string][]string, len(names))
	for _, name := range names {
		out[name] = lists[name]
	}
	return out
}

// readDat reads a dat file back with the input converter.
func readDat(t *testing.T, path string) lib.Container {
	t.Helper()
	conv, err := NewGeoIPDatIn(lib.ActionAdd, WithURI(path))
	if err != nil {
		t.Fatalf("NewGeoIPDatIn: %v", err)
	}
	container, err := conv.Input(lib.NewContainer())
	if err != nil {
		t.Fatalf("Input(%s): %v", path, err)
	}
	return container
}

func newDatOut(t *testing.T, opts ...lib.OutputOption) *geoip_dat_out {
	t.Helper()
	conv, err := NewGeoIPDatOut(lib.ActionOutput, opts...)
	if err != nil {
		t.Fatalf("NewGeoIPDatOut: %v", err)
	}
	return conv.(*geoip_dat_out)
}

func TestNewGeoIPDatOut(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		conv, err := NewGeoIPDatOut(lib.ActionOutput, nil)
		if err != nil {
			t.Fatalf("NewGeoIPDatOut: %v", err)
		}
		out := conv.(*geoip_dat_out)
		if out.OutputName != defaultOutputName || out.OutputName != "geoip.dat" {
			t.Errorf("OutputName: got %q, want %q", out.OutputName, defaultOutputName)
		}
		if out.OutputDir != defaultOutputDir || out.OutputDir != filepath.Join("output", "dat") {
			t.Errorf("OutputDir: got %q, want %q", out.OutputDir, defaultOutputDir)
		}
		if out.Want != nil || out.Exclude != nil || out.OneFilePerList || out.OnlyIPType != "" {
			t.Errorf("unexpected non-zero options: %+v", out)
		}
		if got := conv.GetType(); got != TypeGeoIPDatOut {
			t.Errorf("GetType: got %q, want %q", got, TypeGeoIPDatOut)
		}
		if got := conv.GetAction(); got != lib.ActionOutput {
			t.Errorf("GetAction: got %q, want %q", got, lib.ActionOutput)
		}
		if got := conv.GetDescription(); got != DescGeoIPDatOut {
			t.Errorf("GetDescription: got %q, want %q", got, DescGeoIPDatOut)
		}
	})

	t.Run("all options", func(t *testing.T) {
		t.Parallel()

		out := newDatOut(t,
			WithOutputName(" my.dat "),
			WithOutputDir(" out/dir "),
			WithOutputWantedList([]string{"cn"}),
			WithOutputExcludedList([]string{"us"}),
			WithOneFilePerList(true),
			WithOutputOnlyIPType(lib.IPv4),
		)
		if out.OutputName != "my.dat" {
			t.Errorf("OutputName: got %q, want %q", out.OutputName, "my.dat")
		}
		if out.OutputDir != "out/dir" {
			t.Errorf("OutputDir: got %q, want %q", out.OutputDir, "out/dir")
		}
		assertStrings(t, "Want", out.Want, []string{"cn"})
		assertStrings(t, "Exclude", out.Exclude, []string{"us"})
		if !out.OneFilePerList {
			t.Error("OneFilePerList: got false, want true")
		}
		if out.OnlyIPType != lib.IPv4 {
			t.Errorf("OnlyIPType: got %q, want %q", out.OnlyIPType, lib.IPv4)
		}
	})
}

func TestGeoIPDatOut_WithOutputNameAndDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantName string
		wantDir  string
	}{
		{name: "empty uses defaults", input: "", wantName: defaultOutputName, wantDir: defaultOutputDir},
		{name: "blank uses defaults", input: " \t\n", wantName: defaultOutputName, wantDir: defaultOutputDir},
		{name: "value is trimmed", input: "  custom  ", wantName: "custom", wantDir: "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			out := &geoip_dat_out{}
			WithOutputName(tt.input)(out)
			WithOutputDir(tt.input)(out)
			if out.OutputName != tt.wantName {
				t.Errorf("OutputName: got %q, want %q", out.OutputName, tt.wantName)
			}
			if out.OutputDir != tt.wantDir {
				t.Errorf("OutputDir: got %q, want %q", out.OutputDir, tt.wantDir)
			}
		})
	}
}

func TestNewGeoIPDatOutFromBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		data      string
		wantErrAs any
		want      geoip_dat_out
	}{
		{name: "invalid json", data: "{", wantErrAs: new(*json.SyntaxError)},
		{name: "wrong field type", data: `{"oneFilePerList": "yes"}`, wantErrAs: new(*json.UnmarshalTypeError)},
		{
			name: "empty data uses defaults",
			data: "",
			want: geoip_dat_out{OutputName: defaultOutputName, OutputDir: defaultOutputDir},
		},
		{
			name: "blank values use defaults",
			data: `{"outputName": " ", "outputDir": ""}`,
			want: geoip_dat_out{OutputName: defaultOutputName, OutputDir: defaultOutputDir},
		},
		{
			name: "all fields",
			data: `{"outputName": " a.dat ", "outputDir": " /x/y ", "wantedList": ["cn", "us"], "excludedList": ["jp"], "oneFilePerList": true, "onlyIPType": "ipv4"}`,
			want: geoip_dat_out{
				OutputName:     "a.dat",
				OutputDir:      "/x/y",
				Want:           []string{"cn", "us"},
				Exclude:        []string{"jp"},
				OneFilePerList: true,
				OnlyIPType:     lib.IPv4,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewGeoIPDatOutFromBytes(lib.ActionOutput, []byte(tt.data))
			if tt.wantErrAs != nil {
				assertErrorContains(t, err, "[type v2rayGeoIPDat | action output]")
				if !errors.As(err, tt.wantErrAs) {
					t.Errorf("error %v is not %T", err, tt.wantErrAs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			out := conv.(*geoip_dat_out)
			if out.Type != TypeGeoIPDatOut || out.Action != lib.ActionOutput || out.Description != DescGeoIPDatOut {
				t.Errorf("unexpected identity: %+v", out)
			}
			if out.OutputName != tt.want.OutputName {
				t.Errorf("OutputName: got %q, want %q", out.OutputName, tt.want.OutputName)
			}
			if out.OutputDir != tt.want.OutputDir {
				t.Errorf("OutputDir: got %q, want %q", out.OutputDir, tt.want.OutputDir)
			}
			assertStrings(t, "Want", out.Want, tt.want.Want)
			assertStrings(t, "Exclude", out.Exclude, tt.want.Exclude)
			if out.OneFilePerList != tt.want.OneFilePerList {
				t.Errorf("OneFilePerList: got %v, want %v", out.OneFilePerList, tt.want.OneFilePerList)
			}
			if out.OnlyIPType != tt.want.OnlyIPType {
				t.Errorf("OnlyIPType: got %q, want %q", out.OnlyIPType, tt.want.OnlyIPType)
			}
		})
	}
}

func TestGeoIPDatOut_filterAndSortList(t *testing.T) {
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

			out := newDatOut(t, WithOutputWantedList(tt.want), WithOutputExcludedList(tt.exclude))
			got := out.filterAndSortList(newTestContainer(t, lists))
			if got == nil {
				t.Fatal("got nil list, want non-nil")
			}
			assertStrings(t, "lists", got, tt.result)
		})
	}
}

func TestGeoIPDatOut_Output_RoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		opts       []lib.OutputOption
		outputName string
		want       map[string][]string
	}{
		{
			name:       "all lists",
			outputName: "geoip.dat",
			want:       datOutMerged,
		},
		{
			name:       "custom output name",
			opts:       []lib.OutputOption{WithOutputName("custom.dat")},
			outputName: "custom.dat",
			want:       datOutMerged,
		},
		{
			name:       "wanted lists",
			opts:       []lib.OutputOption{WithOutputWantedList([]string{" us", "cn ", ""})},
			outputName: "geoip.dat",
			want:       pick(datOutMerged, "CN", "US"),
		},
		{
			name:       "excluded lists",
			opts:       []lib.OutputOption{WithOutputExcludedList([]string{"v4only", " V6ONLY "})},
			outputName: "geoip.dat",
			want:       pick(datOutMerged, "CN", "US"),
		},
		{
			name:       "wanted minus excluded",
			opts:       []lib.OutputOption{WithOutputWantedList([]string{"cn", "us"}), WithOutputExcludedList([]string{"us"})},
			outputName: "geoip.dat",
			want:       pick(datOutMerged, "CN"),
		},
		{
			name: "only ipv4",
			opts: []lib.OutputOption{
				WithOutputWantedList([]string{"cn", "us", "v4only"}),
				WithOutputOnlyIPType(lib.IPv4),
			},
			outputName: "geoip.dat",
			want: map[string][]string{
				"CN":     {"192.168.0.0/23"},
				"US":     {"10.0.0.0/7"},
				"V4ONLY": {"1.1.1.0/24"},
			},
		},
		{
			name: "only ipv6",
			opts: []lib.OutputOption{
				WithOutputExcludedList([]string{"v4only"}),
				WithOutputOnlyIPType(lib.IPv6),
			},
			outputName: "geoip.dat",
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
			out := newDatOut(t, append([]lib.OutputOption{WithOutputDir(dir)}, tt.opts...)...)
			if err := out.Output(newTestContainer(t, datOutSource)); err != nil {
				t.Fatalf("Output: %v", err)
			}
			assertStrings(t, "written files", dirFileNames(t, dir), []string{tt.outputName})

			path := filepath.Join(dir, tt.outputName)
			assertContainer(t, readDat(t, path), tt.want)

			// The raw file must be sorted by country code and keep IPv4
			// addresses as 4 bytes and IPv6 addresses as 16 bytes.
			list := readGeoIPList(t, path)
			if len(list.Entry) != len(tt.want) {
				t.Fatalf("got %d raw entries, want %d", len(list.Entry), len(tt.want))
			}
			codes := make([]string, 0, len(list.Entry))
			for _, geoip := range list.Entry {
				codes = append(codes, geoip.GetCountryCode())
				if len(geoip.GetCidr()) != len(tt.want[geoip.GetCountryCode()]) {
					t.Errorf("list %s: got %d raw CIDRs, want %d", geoip.GetCountryCode(), len(geoip.GetCidr()), len(tt.want[geoip.GetCountryCode()]))
				}
				for _, cidr := range geoip.GetCidr() {
					switch {
					case len(cidr.GetIp()) == 4 && cidr.GetPrefix() <= 32:
					case len(cidr.GetIp()) == 16 && cidr.GetPrefix() <= 128:
					default:
						t.Errorf("list %s: unexpected raw CIDR %v/%d", geoip.GetCountryCode(), cidr.GetIp(), cidr.GetPrefix())
					}
				}
			}
			if !slices.IsSorted(codes) {
				t.Errorf("raw entries are not sorted: %q", codes)
			}
		})
	}
}

func TestGeoIPDatOut_Output_RawContent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	out := newDatOut(t, WithOutputDir(dir))
	container := newTestContainer(t, map[string][]string{
		"CN": {"192.168.0.3/24", "192.168.1.5/24", "2001:db8::/33", "2001:db8:8000::/33"},
	})
	if err := out.Output(container); err != nil {
		t.Fatalf("Output: %v", err)
	}

	list := readGeoIPList(t, filepath.Join(dir, defaultOutputName))
	if len(list.GetEntry()) != 1 {
		t.Fatalf("got %d entries, want 1", len(list.GetEntry()))
	}
	geoip := list.GetEntry()[0]
	if geoip.GetCountryCode() != "CN" {
		t.Errorf("country code: got %q, want %q", geoip.GetCountryCode(), "CN")
	}
	want := []*CIDR{
		{Ip: []byte{192, 168, 0, 0}, Prefix: 23},
		{Ip: []byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, Prefix: 32},
	}
	if len(geoip.GetCidr()) != len(want) {
		t.Fatalf("got %d CIDRs, want %d", len(geoip.GetCidr()), len(want))
	}
	for i, cidr := range geoip.GetCidr() {
		if !bytes.Equal(cidr.GetIp(), want[i].Ip) || cidr.GetPrefix() != want[i].Prefix {
			t.Errorf("CIDR[%d]: got %v/%d, want %v/%d", i, cidr.GetIp(), cidr.GetPrefix(), want[i].Ip, want[i].Prefix)
		}
	}
}

func TestGeoIPDatOut_Output_Reproducible(t *testing.T) {
	t.Parallel()

	var outputs [][]byte
	for range 3 {
		dir := t.TempDir()
		if err := newDatOut(t, WithOutputDir(dir)).Output(newTestContainer(t, datOutSource)); err != nil {
			t.Fatalf("Output: %v", err)
		}
		data, err := os.ReadFile(filepath.Join(dir, defaultOutputName))
		if err != nil {
			t.Fatal(err)
		}
		outputs = append(outputs, data)
	}
	for i := 1; i < len(outputs); i++ {
		if !bytes.Equal(outputs[0], outputs[i]) {
			t.Fatalf("output %d differs from output 0", i)
		}
	}
}

func TestGeoIPDatOut_Output_OneFilePerList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		opts      []lib.OutputOption
		wantFiles []string
		want      map[string][]string
	}{
		{
			name:      "all lists, output name is ignored",
			opts:      []lib.OutputOption{WithOutputName("ignored.dat")},
			wantFiles: []string{"cn.dat", "us.dat", "v4only.dat", "v6only.dat"},
			want:      datOutMerged,
		},
		{
			name:      "wanted list with only ipv6",
			opts:      []lib.OutputOption{WithOutputWantedList([]string{"CN", "v6only"}), WithOutputOnlyIPType(lib.IPv6)},
			wantFiles: []string{"cn.dat", "v6only.dat"},
			want: map[string][]string{
				"CN":     {"2001:db8::/32"},
				"V6ONLY": {"2606:4700::/32"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			opts := append([]lib.OutputOption{WithOutputDir(dir), WithOneFilePerList(true)}, tt.opts...)
			if err := newDatOut(t, opts...).Output(newTestContainer(t, datOutSource)); err != nil {
				t.Fatalf("Output: %v", err)
			}
			assertStrings(t, "written files", dirFileNames(t, dir), tt.wantFiles)

			for name, cidrs := range tt.want {
				path := filepath.Join(dir, strings.ToLower(name)+".dat")
				if got := readGeoIPList(t, path); len(got.GetEntry()) != 1 {
					t.Fatalf("%s: got %d entries, want 1", path, len(got.GetEntry()))
				}
				assertContainer(t, readDat(t, path), map[string][]string{name: cidrs})
			}
		})
	}
}

func TestGeoIPDatOut_Output_NothingWritten(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		lists map[string][]string
		opts  []lib.OutputOption
	}{
		{name: "empty container", lists: nil},
		{name: "all lists excluded", lists: datOutSource, opts: []lib.OutputOption{WithOutputExcludedList([]string{"cn", "us", "v4only", "v6only"})}},
		{name: "wanted list fully excluded", lists: datOutSource, opts: []lib.OutputOption{WithOutputWantedList([]string{"cn"}), WithOutputExcludedList([]string{"CN"})}},
		{name: "wanted list not in container", lists: datOutSource, opts: []lib.OutputOption{WithOutputWantedList([]string{"notexist"})}},
		{name: "one file per list, wanted list not in container", lists: datOutSource, opts: []lib.OutputOption{WithOneFilePerList(true), WithOutputWantedList([]string{"notexist"})}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := filepath.Join(t.TempDir(), "out")
			opts := append([]lib.OutputOption{WithOutputDir(dir)}, tt.opts...)
			if err := newDatOut(t, opts...).Output(newTestContainer(t, tt.lists)); err != nil {
				t.Fatalf("Output: %v", err)
			}
			if _, err := os.Stat(dir); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("output dir should not be created, stat error: %v", err)
			}
		})
	}
}

// TestGeoIPDatOut_Output_EntryNotFound captures the log output, so it must
// not run in parallel.
func TestGeoIPDatOut_Output_EntryNotFound(t *testing.T) {
	logs := captureLog(t)

	dir := t.TempDir()
	out := newDatOut(t, WithOutputDir(dir), WithOutputWantedList([]string{"cn", "notexist"}))
	if err := out.Output(newTestContainer(t, datOutSource)); err != nil {
		t.Fatalf("Output: %v", err)
	}
	if !bytes.Contains(logs.Bytes(), []byte("entry NOTEXIST not found")) {
		t.Errorf("log %q does not report the missing list", logs.String())
	}
	assertStrings(t, "written files", dirFileNames(t, dir), []string{defaultOutputName})
	assertContainer(t, readDat(t, filepath.Join(dir, defaultOutputName)), pick(datOutMerged, "CN"))
}

func TestGeoIPDatOut_Output_Errors(t *testing.T) {
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
		return newTestContainer(t, datOutSource)
	}

	tests := []struct {
		name      string
		container func(*testing.T) lib.Container
		// setup prepares the output dir and returns the options to use.
		setup     func(t *testing.T, dir string) []lib.OutputOption
		wantErr   string
		wantErrAs bool
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
			wantErrAs: true,
		},
		{
			name:      "output file is a directory",
			container: sourceContainer,
			setup: func(t *testing.T, dir string) []lib.OutputOption {
				if err := os.Mkdir(filepath.Join(dir, defaultOutputName), 0o755); err != nil {
					t.Fatal(err)
				}
				return []lib.OutputOption{WithOutputDir(dir)}
			},
			wantErrAs: true,
		},
		{
			name:      "one file per list, output file is a directory",
			container: sourceContainer,
			setup: func(t *testing.T, dir string) []lib.OutputOption {
				if err := os.Mkdir(filepath.Join(dir, "us.dat"), 0o755); err != nil {
					t.Fatal(err)
				}
				return []lib.OutputOption{WithOutputDir(dir), WithOneFilePerList(true)}
			},
			wantErrAs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := tt.setup(t, t.TempDir())
			err := newDatOut(t, opts...).Output(tt.container(t))
			assertErrorContains(t, err, "[type v2rayGeoIPDat | action output]", tt.wantErr)
			var pathErr *fs.PathError
			if tt.wantErrAs && !errors.As(err, &pathErr) {
				t.Errorf("error %v is not a *fs.PathError", err)
			}
		})
	}
}

func TestGeoIPDatOut_generateGeoIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		onlyIPType lib.IPType
		want       []string
		wantErr    bool
	}{
		{name: "both families", want: []string{"192.168.0.0/23", "2001:db8::/32"}},
		{name: "only ipv4", onlyIPType: lib.IPv4, want: []string{"192.168.0.0/23"}},
		{name: "only ipv6", onlyIPType: lib.IPv6, want: []string{"2001:db8::/32"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			container := newTestContainer(t, pick(datOutSource, "CN"))
			entry, _ := container.GetEntry("CN")
			geoip, err := newDatOut(t, WithOutputOnlyIPType(tt.onlyIPType)).generateGeoIP(entry)
			if err != nil {
				t.Fatalf("generateGeoIP: %v", err)
			}
			if geoip.GetCountryCode() != "CN" {
				t.Errorf("country code: got %q, want CN", geoip.GetCountryCode())
			}
			got := make([]string, 0, len(geoip.GetCidr()))
			for _, cidr := range geoip.GetCidr() {
				got = append(got, cidrString(t, cidr))
			}
			assertStrings(t, "CIDRs", got, tt.want)
		})
	}

	t.Run("entry without prefix", func(t *testing.T) {
		t.Parallel()

		geoip, err := newDatOut(t).generateGeoIP(lib.NewEntry("empty"))
		assertErrorContains(t, err, "EMPTY")
		if geoip != nil {
			t.Errorf("got %v, want nil", geoip)
		}
	})
}

// cidrString formats a raw v2ray CIDR without going through lib.
func cidrString(t *testing.T, cidr *CIDR) string {
	t.Helper()
	addr, ok := netip.AddrFromSlice(cidr.GetIp())
	if !ok {
		t.Fatalf("invalid raw IP %v", cidr.GetIp())
	}
	return netip.PrefixFrom(addr, int(cidr.GetPrefix())).String()
}

func TestGeoIPDatOut_sort(t *testing.T) {
	t.Parallel()

	first, second := geoIP("CN", rawCIDR("1.0.0.0", 8)), geoIP("CN", rawCIDR("2.0.0.0", 8))
	list := &GeoIPList{Entry: []*GeoIP{geoIP("US"), first, geoIP("AU"), second, geoIP("JP")}}
	newDatOut(t).sort(list)

	got := make([]string, 0, len(list.Entry))
	for _, geoip := range list.Entry {
		got = append(got, geoip.GetCountryCode())
	}
	assertStrings(t, "sorted codes", got, []string{"AU", "CN", "CN", "JP", "US"})
	if list.Entry[1] != first || list.Entry[2] != second {
		t.Error("sort is not stable for equal country codes")
	}
}

// TestGeoIPDatOut_Registration checks the converter is reachable through the
// lib config registry set up in init.
func TestGeoIPDatOut_Registration(t *testing.T) {
	t.Parallel()

	if err := lib.RegisterOutputConverter(TypeGeoIPDatOut, &geoip_dat_out{}); !errors.Is(err, lib.ErrDuplicatedConverter) {
		t.Errorf("RegisterOutputConverter: got %v, want %v", err, lib.ErrDuplicatedConverter)
	}
	if err := lib.RegisterOutputConfigCreator(TypeGeoIPDatOut, nil); err == nil {
		t.Error("RegisterOutputConfigCreator: got nil error for an already registered type")
	}

	t.Run("valid args", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		args, err := json.Marshal(map[string]any{"outputDir": dir, "outputName": "out.dat", "wantedList": []string{"cn"}})
		if err != nil {
			t.Fatal(err)
		}
		instance, err := lib.NewInstance()
		if err != nil {
			t.Fatal(err)
		}
		if err := instance.InitConfigFromBytes([]byte(`{"output": [{"type": "v2rayGeoIPDat", "action": "output", "args": ` + string(args) + `}]}`)); err != nil {
			t.Fatalf("InitConfigFromBytes: %v", err)
		}
		if err := instance.RunOutput(newTestContainer(t, datOutSource)); err != nil {
			t.Fatalf("RunOutput: %v", err)
		}
		assertContainer(t, readDat(t, filepath.Join(dir, "out.dat")), pick(datOutMerged, "CN"))
	})

	t.Run("invalid args", func(t *testing.T) {
		t.Parallel()

		instance, err := lib.NewInstance()
		if err != nil {
			t.Fatal(err)
		}
		if err := instance.InitConfigFromBytes([]byte(`{"output": [{"type": "v2rayGeoIPDat", "args": {"outputDir": 1}}]}`)); err == nil {
			t.Fatal("got nil error, want error")
		}
	})
}
