package v2ray

import (
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"google.golang.org/protobuf/proto"
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

// readDat decodes a dat file into country code -> CIDR strings, preserving entry order.
func readDat(t *testing.T, path string) ([]string, map[string][]string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var list GeoIPList
	if err := proto.Unmarshal(content, &list); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	order := make([]string, 0, len(list.Entry))
	result := make(map[string][]string, len(list.Entry))
	for _, g := range list.Entry {
		order = append(order, g.CountryCode)
		for _, c := range g.Cidr {
			addr, ok := netip.AddrFromSlice(c.Ip)
			if !ok {
				t.Fatalf("invalid ip %v", c.Ip)
			}
			result[g.CountryCode] = append(result[g.CountryCode], netip.PrefixFrom(addr, int(c.Prefix)).String())
		}
	}
	return order, result
}

func TestRegisteredViaConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{
			name:   "valid input and output",
			config: `{"input":[{"type":"v2rayGeoIPDat","action":"add","args":{"uri":"geoip.dat"}}],"output":[{"type":"v2rayGeoIPDat","action":"output","args":{"outputDir":"out"}}]}`,
		},
		{
			name:    "invalid input args",
			config:  `{"input":[{"type":"v2rayGeoIPDat","action":"add","args":{}}]}`,
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

func TestNewGeoIPDatOut(t *testing.T) {
	tests := []struct {
		name string
		opts []lib.OutputOption
		want *geoip_dat_out
	}{
		{
			name: "defaults",
			opts: []lib.OutputOption{nil},
			want: &geoip_dat_out{OutputName: defaultOutputName, OutputDir: defaultOutputDir},
		},
		{
			name: "blank values fall back to defaults",
			opts: []lib.OutputOption{WithOutputName(" "), WithOutputDir(" ")},
			want: &geoip_dat_out{OutputName: defaultOutputName, OutputDir: defaultOutputDir},
		},
		{
			name: "all options",
			opts: []lib.OutputOption{
				WithOutputName(" my.dat "),
				WithOutputDir(" dir "),
				WithOutputWantedList([]string{"cn"}),
				WithOutputExcludedList([]string{"us"}),
				WithOneFilePerList(true),
				WithOutputOnlyIPType(lib.IPv4),
			},
			want: &geoip_dat_out{
				OutputName: "my.dat", OutputDir: "dir", Want: []string{"cn"}, Exclude: []string{"us"},
				OneFilePerList: true, OnlyIPType: lib.IPv4,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewGeoIPDatOut(lib.ActionOutput, tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			got := conv.(*geoip_dat_out)
			assertDatOut(t, got, tt.want)
			if got.GetType() != TypeGeoIPDatOut || got.GetAction() != lib.ActionOutput || got.GetDescription() != DescGeoIPDatOut {
				t.Fatal("unexpected getter results")
			}
		})
	}
}

func assertDatOut(t *testing.T, got, want *geoip_dat_out) {
	t.Helper()
	if got.OutputName != want.OutputName || got.OutputDir != want.OutputDir ||
		!slices.Equal(got.Want, want.Want) || !slices.Equal(got.Exclude, want.Exclude) ||
		got.OneFilePerList != want.OneFilePerList || got.OnlyIPType != want.OnlyIPType {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestNewGeoIPDatOutFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		want    *geoip_dat_out
	}{
		{name: "empty", data: "", want: &geoip_dat_out{OutputName: defaultOutputName, OutputDir: defaultOutputDir}},
		{name: "invalid json", data: "{", wantErr: true},
		{
			name: "full",
			data: `{"outputName":"a.dat","outputDir":"dir","wantedList":["cn"],"excludedList":["us"],"oneFilePerList":true,"onlyIPType":"ipv6"}`,
			want: &geoip_dat_out{
				OutputName: "a.dat", OutputDir: "dir", Want: []string{"cn"}, Exclude: []string{"us"},
				OneFilePerList: true, OnlyIPType: lib.IPv6,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewGeoIPDatOutFromBytes(lib.ActionOutput, []byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			assertDatOut(t, conv.(*geoip_dat_out), tt.want)
		})
	}
}

func TestGeoIPDatOut_filterAndSortList(t *testing.T) {
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
			g := &geoip_dat_out{Want: tt.want, Exclude: tt.exclude}
			got := g.filterAndSortList(c)
			if len(got) != len(tt.expect) || !slices.Equal(got, tt.expect) {
				t.Fatalf("got %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestGeoIPDatOut_sort(t *testing.T) {
	list := &GeoIPList{Entry: []*GeoIP{{CountryCode: "US"}, {CountryCode: "CN"}, {CountryCode: "JP"}}}
	(&geoip_dat_out{}).sort(list)
	got := make([]string, 0, len(list.Entry))
	for _, e := range list.Entry {
		got = append(got, e.CountryCode)
	}
	want := []string{"CN", "JP", "US"}
	if len(got) != len(want) || !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestGeoIPDatOut_Output(t *testing.T) {
	data := map[string][]string{
		"CN": {"1.0.0.0/24", "1.0.1.0/24", "2001:db8::/32"},
		"US": {"8.8.8.0/24"},
		"V6": {"2400::/16"},
	}

	type datFile struct {
		order []string
		lists map[string][]string
	}

	tests := []struct {
		name       string
		out        func(dir string) *geoip_dat_out
		setup      func(t *testing.T, dir string)
		wantErr    string
		wantFiles  map[string]datFile
		wantAbsent []string
	}{
		{
			name: "single file all entries sorted",
			out:  func(dir string) *geoip_dat_out { return &geoip_dat_out{OutputName: "geoip.dat", OutputDir: dir} },
			wantFiles: map[string]datFile{
				"geoip.dat": {
					order: []string{"CN", "US", "V6"},
					lists: map[string][]string{
						"CN": {"1.0.0.0/23", "2001:db8::/32"},
						"US": {"8.8.8.0/24"},
						"V6": {"2400::/16"},
					},
				},
			},
		},
		{
			name: "single file wanted with missing entry and only ipv6",
			out: func(dir string) *geoip_dat_out {
				return &geoip_dat_out{OutputName: "geoip.dat", OutputDir: dir, Want: []string{"cn", "missing"}, OnlyIPType: lib.IPv6}
			},
			wantFiles: map[string]datFile{
				"geoip.dat": {order: []string{"CN"}, lists: map[string][]string{"CN": {"2001:db8::/32"}}},
			},
		},
		{
			name: "one file per list",
			out: func(dir string) *geoip_dat_out {
				return &geoip_dat_out{OutputName: "geoip.dat", OutputDir: dir, OneFilePerList: true, Exclude: []string{"v6"}}
			},
			wantFiles: map[string]datFile{
				"cn.dat": {order: []string{"CN"}, lists: map[string][]string{"CN": {"1.0.0.0/23", "2001:db8::/32"}}},
				"us.dat": {order: []string{"US"}, lists: map[string][]string{"US": {"8.8.8.0/24"}}},
			},
			wantAbsent: []string{"geoip.dat", "v6.dat"},
		},
		{
			name: "nothing to output",
			out: func(dir string) *geoip_dat_out {
				return &geoip_dat_out{OutputName: "geoip.dat", OutputDir: dir, Want: []string{"missing"}}
			},
			wantAbsent: []string{"geoip.dat"},
		},
		{
			name: "entry without required ip type",
			out: func(dir string) *geoip_dat_out {
				return &geoip_dat_out{OutputName: "geoip.dat", OutputDir: dir, Want: []string{"v6"}, OnlyIPType: lib.IPv4}
			},
			wantErr: "has no prefix",
		},
		{
			name: "single file output dir is a file",
			out: func(dir string) *geoip_dat_out {
				return &geoip_dat_out{OutputName: "geoip.dat", OutputDir: filepath.Join(dir, "file")}
			},
			setup: func(t *testing.T, dir string) {
				if err := os.WriteFile(filepath.Join(dir, "file"), []byte("x"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: "not a directory",
		},
		{
			name: "one file per list output path is a directory",
			out: func(dir string) *geoip_dat_out {
				return &geoip_dat_out{OutputDir: dir, OneFilePerList: true, Want: []string{"cn"}}
			},
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, "cn.dat"), 0755); err != nil {
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
			g := tt.out(dir)
			g.Type, g.Action = TypeGeoIPDatOut, lib.ActionOutput
			err := g.Output(newTestContainer(t, data))
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
				order, lists := readDat(t, filepath.Join(dir, filename))
				if len(order) != len(want.order) || !slices.Equal(order, want.order) {
					t.Fatalf("%s: entry order got %v, want %v", filename, order, want.order)
				}
				for name, cidrs := range want.lists {
					assertCIDRs(t, lists[name], cidrs)
				}
			}
			for _, filename := range tt.wantAbsent {
				if _, err := os.Stat(filepath.Join(dir, filename)); !os.IsNotExist(err) {
					t.Fatalf("file %s should not exist", filename)
				}
			}
		})
	}
}
