package special

import (
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestNewStdout(t *testing.T) {
	got, err := NewStdout(lib.ActionOutput, nil,
		WithStdoutWantedList([]string{"cn"}),
		WithStdoutExcludedList([]string{"us"}),
		WithStdoutOnlyIPType(lib.IPv4))
	if err != nil {
		t.Fatal(err)
	}
	s := got.(*stdout)
	if s.GetType() != TypeStdout || s.GetAction() != lib.ActionOutput || s.GetDescription() != DescStdout {
		t.Fatalf("unexpected metadata: %+v", s)
	}
	if !slices.Equal(s.Want, []string{"cn"}) || !slices.Equal(s.Exclude, []string{"us"}) || s.OnlyIPType != lib.IPv4 {
		t.Fatalf("got %+v", s)
	}
}

func TestNewStdoutFromBytes(t *testing.T) {
	tests := []struct {
		name        string
		data        string
		wantErr     bool
		wantWant    []string
		wantExclude []string
		wantIP      lib.IPType
	}{
		{"empty", ``, false, nil, nil, ""},
		{"valid", `{"wantedList":["cn"],"excludedList":["us"],"onlyIPType":"ipv6"}`, false, []string{"cn"}, []string{"us"}, lib.IPv6},
		{"invalid json", `{`, true, nil, nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewStdoutFromBytes(lib.ActionOutput, []byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			s := got.(*stdout)
			if !slices.Equal(s.Want, tt.wantWant) || !slices.Equal(s.Exclude, tt.wantExclude) || s.OnlyIPType != tt.wantIP {
				t.Fatalf("got %+v", s)
			}
		})
	}
}

func TestStdoutFilterAndSortList(t *testing.T) {
	c := newContainerWith(t, map[string][]string{
		"US": {"8.8.8.0/24"}, "CN": {"1.0.0.0/24"}, "JP": {"9.9.9.0/24"},
	})
	tests := []struct {
		name    string
		want    []string
		exclude []string
		expect  []string
	}{
		{"all sorted", nil, nil, []string{"CN", "JP", "US"}},
		{"exclude", nil, []string{" us ", ""}, []string{"CN", "JP"}},
		{"wanted sorted", []string{"us", " cn", "", "notexist"}, nil, []string{"CN", "NOTEXIST", "US"}},
		{"wanted minus excluded", []string{"us", "cn"}, []string{"cn"}, []string{"US"}},
		{"wanted all excluded", []string{"cn"}, []string{"cn"}, []string{}},
		{"blank wanted falls back to all", []string{" "}, []string{"jp"}, []string{"CN", "US"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stdout{Want: tt.want, Exclude: tt.exclude}
			assertCIDRs(t, s.filterAndSortList(c), tt.expect)
		})
	}
}

func TestStdoutGenerateCIDRList(t *testing.T) {
	tests := []struct {
		name    string
		cidrs   []string
		onlyIP  lib.IPType
		want    []string
		wantErr bool
	}{
		{"all", []string{"1.0.0.0/24", "1.0.1.0/24", "2001:db8::/32"}, "", []string{"1.0.0.0/23", "2001:db8::/32"}, false},
		{"ipv4", []string{"1.0.0.0/24", "2001:db8::/32"}, lib.IPv4, []string{"1.0.0.0/24"}, false},
		{"ipv6", []string{"1.0.0.0/24", "2001:db8::/32"}, lib.IPv6, []string{"2001:db8::/32"}, false},
		{"empty entry", nil, "", nil, true},
		{"ipv4 only but no ipv4", []string{"2001:db8::/32"}, lib.IPv4, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := lib.NewEntry("test")
			for _, cidr := range tt.cidrs {
				if err := entry.AddPrefix(cidr); err != nil {
					t.Fatal(err)
				}
			}
			s := &stdout{OnlyIPType: tt.onlyIP}
			got, err := s.generateCIDRList(entry)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			assertCIDRs(t, got, tt.want)
		})
	}
}

func TestStdoutOutput(t *testing.T) {
	data := map[string][]string{
		"CN": {"1.0.0.0/24", "2001:db8::/32"},
		"US": {"8.8.8.0/24"},
		"JP": {"2400::/32"},
	}
	tests := []struct {
		name    string
		want    []string
		exclude []string
		onlyIP  lib.IPType
		expect  []string
	}{
		{"all", nil, nil, "", []string{"1.0.0.0/24", "2001:db8::/32", "2400::/32", "8.8.8.0/24"}},
		{"wanted with missing", []string{"us", "notexist"}, nil, "", []string{"8.8.8.0/24"}},
		{"excluded", nil, []string{"cn"}, "", []string{"2400::/32", "8.8.8.0/24"}},
		{"ipv4 only skips empty entries", nil, nil, lib.IPv4, []string{"1.0.0.0/24", "8.8.8.0/24"}},
		{"ipv6 only", nil, nil, lib.IPv6, []string{"2001:db8::/32", "2400::/32"}},
		{"nothing", []string{"cn"}, []string{"cn"}, "", []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, _ := NewStdout(lib.ActionOutput,
				WithStdoutWantedList(tt.want),
				WithStdoutExcludedList(tt.exclude),
				WithStdoutOnlyIPType(tt.onlyIP))
			c := newContainerWith(t, data)
			var outErr error
			out := captureStdout(t, func() { outErr = conv.Output(c) })
			if outErr != nil {
				t.Fatal(outErr)
			}
			got := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
			if out == "" {
				got = []string{}
			}
			assertCIDRs(t, got, tt.expect)
		})
	}
}
