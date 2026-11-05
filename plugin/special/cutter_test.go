package special

import (
	"errors"
	"maps"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestNewCutter(t *testing.T) {
	tests := []struct {
		name       string
		action     lib.Action
		opts       []lib.InputOption
		wantErr    bool
		wantWant   map[string]bool
		wantIPType lib.IPType
	}{
		{"valid", lib.ActionRemove, []lib.InputOption{WithCutterWantedList([]string{" cn ", "", "us"}), WithCutterOnlyIPType(lib.IPv4)}, false, map[string]bool{"CN": true, "US": true}, lib.IPv4},
		{"nil option ignored", lib.ActionRemove, []lib.InputOption{nil, WithCutterWantedList([]string{"cn"})}, false, map[string]bool{"CN": true}, ""},
		{"add action", lib.ActionAdd, []lib.InputOption{WithCutterWantedList([]string{"cn"})}, true, nil, ""},
		{"output action", lib.ActionOutput, []lib.InputOption{WithCutterWantedList([]string{"cn"})}, true, nil, ""},
		{"no wanted list", lib.ActionRemove, nil, true, nil, ""},
		{"blank wanted list", lib.ActionRemove, []lib.InputOption{WithCutterWantedList([]string{" ", ""})}, true, nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCutter(tt.action, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if got != nil {
					t.Fatal("expected nil converter on error")
				}
				return
			}
			c := got.(*cutter)
			if c.GetType() != TypeCutter || c.GetAction() != tt.action || c.GetDescription() != DescCutter {
				t.Fatalf("unexpected metadata: %+v", c)
			}
			if !maps.Equal(c.Want, tt.wantWant) {
				t.Fatalf("Want = %v, want %v", c.Want, tt.wantWant)
			}
			if c.OnlyIPType != tt.wantIPType {
				t.Fatalf("OnlyIPType = %q, want %q", c.OnlyIPType, tt.wantIPType)
			}
		})
	}
}

func TestNewCutterFromBytes(t *testing.T) {
	tests := []struct {
		name       string
		action     lib.Action
		data       string
		wantErr    bool
		wantWant   map[string]bool
		wantIPType lib.IPType
	}{
		{"valid", lib.ActionRemove, `{"wantedList":["cn","us"],"onlyIPType":"ipv6"}`, false, map[string]bool{"CN": true, "US": true}, lib.IPv6},
		{"invalid json", lib.ActionRemove, `{invalid`, true, nil, ""},
		{"empty data", lib.ActionRemove, ``, true, nil, ""},
		{"wrong action", lib.ActionAdd, `{"wantedList":["cn"]}`, true, nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCutterFromBytes(tt.action, []byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			c := got.(*cutter)
			if !maps.Equal(c.Want, tt.wantWant) || c.OnlyIPType != tt.wantIPType {
				t.Fatalf("got %+v", c)
			}
		})
	}
}

func TestCutterInput(t *testing.T) {
	data := map[string][]string{
		"CN": {"1.0.0.0/24", "2001:db8::/32"},
		"US": {"8.8.8.0/24", "2400::/32"},
		"JP": {"9.9.9.0/24"},
	}
	tests := []struct {
		name      string
		want      []string
		onlyIP    lib.IPType
		wantNames []string
		wantCIDRs map[string][]string
	}{
		{
			name:      "remove whole entry",
			want:      []string{"cn"},
			wantNames: []string{"JP", "US"},
			wantCIDRs: map[string][]string{"US": {"8.8.8.0/24", "2400::/32"}, "JP": {"9.9.9.0/24"}},
		},
		{
			name:      "remove multiple entries",
			want:      []string{"cn", "us", "notexist"},
			wantNames: []string{"JP"},
			wantCIDRs: map[string][]string{"JP": {"9.9.9.0/24"}},
		},
		{
			name:      "remove only ipv4",
			want:      []string{"cn"},
			onlyIP:    lib.IPv4,
			wantNames: []string{"CN", "JP", "US"},
			wantCIDRs: map[string][]string{"CN": {"2001:db8::/32"}, "US": {"8.8.8.0/24", "2400::/32"}},
		},
		{
			name:      "remove only ipv6",
			want:      []string{"cn"},
			onlyIP:    lib.IPv6,
			wantNames: []string{"CN", "JP", "US"},
			wantCIDRs: map[string][]string{"CN": {"1.0.0.0/24"}, "JP": {"9.9.9.0/24"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewCutter(lib.ActionRemove, WithCutterWantedList(tt.want), WithCutterOnlyIPType(tt.onlyIP))
			if err != nil {
				t.Fatal(err)
			}
			c, err := conv.Input(newContainerWith(t, data))
			if err != nil {
				t.Fatal(err)
			}
			assertCIDRs(t, entryNames(c), tt.wantNames)
			for name, want := range tt.wantCIDRs {
				assertCIDRs(t, entryCIDRs(t, c, name), want)
			}
		})
	}
}

func TestCutterInputError(t *testing.T) {
	conv, err := NewCutter(lib.ActionRemove, WithCutterWantedList([]string{"cn"}))
	if err != nil {
		t.Fatal(err)
	}
	mc := &mockContainer{Container: newContainerWith(t, map[string][]string{"CN": {"1.0.0.0/24"}}), removeErr: errMock}
	got, err := conv.Input(mc)
	if !errors.Is(err, errMock) || got != nil {
		t.Fatalf("got (%v, %v), want mock error", got, err)
	}
}
