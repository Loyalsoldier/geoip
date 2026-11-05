package mihomo

import (
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"go4.org/netipx"
)

func newTestContainer(t *testing.T, data map[string][]string) lib.Container {
	t.Helper()
	c := lib.NewContainer()
	for name, cidrs := range data {
		e := lib.NewEntry(name)
		for _, cidr := range cidrs {
			if err := e.AddPrefix(cidr); err != nil {
				t.Fatalf("AddPrefix(%s): %v", cidr, err)
			}
		}
		if err := c.Add(e); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}
	return c
}

func TestRegisteredViaConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{
			name:   "valid input and output",
			config: `{"input":[{"type":"mihomoMRS","action":"add","args":{"name":"cn","uri":"cn.mrs"}}],"output":[{"type":"mihomoMRS","action":"output","args":{"outputDir":"out"}}]}`,
		},
		{
			name:    "invalid input args",
			config:  `{"input":[{"type":"mihomoMRS","action":"add","args":{}}]}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance, err := lib.NewInstance()
			if err != nil {
				t.Fatal(err)
			}
			err = instance.InitConfigFromBytes([]byte(tt.config))
			if (err != nil) != tt.wantErr {
				t.Fatalf("wantErr %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestNewMRSOut(t *testing.T) {
	tests := []struct {
		name string
		opts []lib.OutputOption
		want *mrs_out
	}{
		{
			name: "defaults",
			opts: []lib.OutputOption{nil},
			want: &mrs_out{Type: TypeMRSOut, Action: lib.ActionOutput, Description: DescMRSOut, OutputDir: defaultOutputDir},
		},
		{
			name: "empty output dir falls back to default",
			opts: []lib.OutputOption{WithOutputDir("  ")},
			want: &mrs_out{Type: TypeMRSOut, Action: lib.ActionOutput, Description: DescMRSOut, OutputDir: defaultOutputDir},
		},
		{
			name: "all options",
			opts: []lib.OutputOption{
				WithOutputDir(" dir "),
				WithOutputWantedList([]string{"cn"}),
				WithOutputExcludedList([]string{"us"}),
				WithOutputOnlyIPType(lib.IPv4),
			},
			want: &mrs_out{
				Type: TypeMRSOut, Action: lib.ActionOutput, Description: DescMRSOut, OutputDir: "dir",
				Want: []string{"cn"}, Exclude: []string{"us"}, OnlyIPType: lib.IPv4,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewMRSOut(lib.ActionOutput, tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			got := conv.(*mrs_out)
			if got.Type != tt.want.Type || got.Action != tt.want.Action || got.Description != tt.want.Description ||
				got.OutputDir != tt.want.OutputDir || !slices.Equal(got.Want, tt.want.Want) ||
				!slices.Equal(got.Exclude, tt.want.Exclude) || got.OnlyIPType != tt.want.OnlyIPType {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
			if got.GetType() != TypeMRSOut || got.GetAction() != lib.ActionOutput || got.GetDescription() != DescMRSOut {
				t.Fatal("unexpected getter results")
			}
		})
	}
}

func TestNewMRSOutFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		want    *mrs_out
	}{
		{name: "empty", data: "", want: &mrs_out{OutputDir: defaultOutputDir}},
		{name: "invalid json", data: "{", wantErr: true},
		{
			name: "full",
			data: `{"outputDir":"dir","wantedList":["cn"],"excludedList":["us"],"onlyIPType":"ipv6"}`,
			want: &mrs_out{OutputDir: "dir", Want: []string{"cn"}, Exclude: []string{"us"}, OnlyIPType: lib.IPv6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewMRSOutFromBytes(lib.ActionOutput, []byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got := conv.(*mrs_out)
			if got.OutputDir != tt.want.OutputDir || !slices.Equal(got.Want, tt.want.Want) ||
				!slices.Equal(got.Exclude, tt.want.Exclude) || got.OnlyIPType != tt.want.OnlyIPType {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestMRSOut_filterAndSortList(t *testing.T) {
	c := newTestContainer(t, map[string][]string{
		"CN": {"1.0.0.0/24"},
		"US": {"8.8.8.0/24"},
		"JP": {"2.0.0.0/24"},
	})

	tests := []struct {
		name    string
		want    []string
		exclude []string
		expect  []string
	}{
		{name: "all sorted", expect: []string{"CN", "JP", "US"}},
		{name: "exclude", exclude: []string{" us ", ""}, expect: []string{"CN", "JP"}},
		{name: "wanted sorted", want: []string{"us", " cn ", ""}, expect: []string{"CN", "US"}},
		{name: "wanted includes missing", want: []string{"xx"}, expect: []string{"XX"}},
		{name: "wanted all excluded", want: []string{"cn"}, exclude: []string{"CN"}, expect: []string{}},
		{name: "wanted only blanks", want: []string{" ", ""}, expect: []string{"CN", "JP", "US"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mrs_out{Want: tt.want, Exclude: tt.exclude}
			got := m.filterAndSortList(c)
			if len(got) != len(tt.expect) || !slices.Equal(got, tt.expect) {
				t.Fatalf("got %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestMRSOut_Output(t *testing.T) {
	data := map[string][]string{
		"CN": {"1.0.0.0/24", "1.0.1.0/24", "2001:db8::/32"},
		"US": {"8.8.8.0/24"},
		"V6": {"2400::/16"},
	}

	tests := []struct {
		name       string
		out        func(dir string) *mrs_out
		setup      func(t *testing.T, dir string)
		wantErr    string
		wantFiles  map[string][]string
		wantAbsent []string
	}{
		{
			name: "all entries",
			out:  func(dir string) *mrs_out { return &mrs_out{OutputDir: dir} },
			wantFiles: map[string][]string{
				"cn.mrs": {"1.0.0.0/23", "2001:db8::/32"},
				"us.mrs": {"8.8.8.0/24"},
				"v6.mrs": {"2400::/16"},
			},
		},
		{
			name:       "wanted list with missing entry",
			out:        func(dir string) *mrs_out { return &mrs_out{OutputDir: dir, Want: []string{"cn", "missing"}} },
			wantFiles:  map[string][]string{"cn.mrs": {"1.0.0.0/23", "2001:db8::/32"}},
			wantAbsent: []string{"us.mrs", "v6.mrs", "missing.mrs"},
		},
		{
			name:      "only ipv4",
			out:       func(dir string) *mrs_out { return &mrs_out{OutputDir: dir, Want: []string{"cn"}, OnlyIPType: lib.IPv4} },
			wantFiles: map[string][]string{"cn.mrs": {"1.0.0.0/23"}},
		},
		{
			name:    "entry without required ip type",
			out:     func(dir string) *mrs_out { return &mrs_out{OutputDir: dir, Want: []string{"v6"}, OnlyIPType: lib.IPv4} },
			wantErr: "has no prefix",
		},
		{
			name: "output dir is a file",
			out:  func(dir string) *mrs_out { return &mrs_out{OutputDir: filepath.Join(dir, "file")} },
			setup: func(t *testing.T, dir string) {
				writeFile(t, filepath.Join(dir, "file"), []byte("x"))
			},
			wantErr: "not a directory",
		},
		{
			name: "output file path is a directory",
			out:  func(dir string) *mrs_out { return &mrs_out{OutputDir: dir, Want: []string{"cn"}} },
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, "cn.mrs"), 0755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: "is a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.setup != nil {
				tt.setup(t, dir)
			}
			m := tt.out(dir)
			m.Type, m.Action = TypeMRSOut, lib.ActionOutput
			err := m.Output(newTestContainer(t, data))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for filename, want := range tt.wantFiles {
				content, err := os.ReadFile(filepath.Join(dir, filename))
				if err != nil {
					t.Fatalf("read %s: %v", filename, err)
				}
				entry := lib.NewEntry("X")
				if err := (&mrs_in{}).parseMRS(content, entry); err != nil {
					t.Fatalf("parse %s: %v", filename, err)
				}
				got, _ := entry.MarshalText()
				assertCIDRs(t, got, want)
			}
			for _, filename := range tt.wantAbsent {
				if _, err := os.Stat(filepath.Join(dir, filename)); !os.IsNotExist(err) {
					t.Fatalf("file %s should not exist", filename)
				}
			}
		})
	}
}

func TestMRSRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		cidrs []string
		want  []string
	}{
		{name: "single ipv4", cidrs: []string{"1.2.3.4/32"}, want: []string{"1.2.3.4/32"}},
		{name: "single ipv6", cidrs: []string{"::1/128"}, want: []string{"::1/128"}},
		{name: "merge adjacent", cidrs: []string{"10.0.0.0/9", "10.128.0.0/9"}, want: []string{"10.0.0.0/8"}},
		{name: "full ranges", cidrs: []string{"0.0.0.0/0", "::/0"}, want: []string{"0.0.0.0/0", "::/0"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := lib.NewEntry("X")
			if err := (&mrs_in{}).parseMRS(buildMRS(t, tt.cidrs...), entry); err != nil {
				t.Fatal(err)
			}
			got, _ := entry.MarshalText()
			assertCIDRs(t, got, tt.want)
		})
	}
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("write error") }

func TestMRSOut_convertToMrsWriteError(t *testing.T) {
	// enough ranges to force the zstd encoder to flush to the failing writer
	ranges := make([]netipx.IPRange, 0, 100000)
	addr := netip.MustParseAddr("2001:db8::")
	for i := 0; i < 100000; i++ {
		ranges = append(ranges, netipx.IPRangeFrom(addr, addr))
		addr = addr.Next().Next()
	}
	if err := (&mrs_out{}).convertToMrs(ranges, errWriter{}); err == nil {
		t.Fatal("expected write error")
	}
}
