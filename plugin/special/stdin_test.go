package special

import (
	"bufio"
	"errors"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestNewStdin(t *testing.T) {
	tests := []struct {
		name     string
		opts     []lib.InputOption
		wantErr  bool
		wantName string
		wantIP   lib.IPType
	}{
		{"valid", []lib.InputOption{nil, WithStdinName(" cn "), WithStdinOnlyIPType(lib.IPv4)}, false, "cn", lib.IPv4},
		{"no name", nil, true, "", ""},
		{"blank name", []lib.InputOption{WithStdinName("  ")}, true, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewStdin(lib.ActionAdd, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			s := got.(*stdin)
			if s.GetType() != TypeStdin || s.GetAction() != lib.ActionAdd || s.GetDescription() != DescStdin {
				t.Fatalf("unexpected metadata: %+v", s)
			}
			if s.Name != tt.wantName || s.OnlyIPType != tt.wantIP {
				t.Fatalf("got %+v", s)
			}
		})
	}
}

func TestNewStdinFromBytes(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		wantErr  bool
		wantName string
		wantIP   lib.IPType
	}{
		{"valid", `{"name":"cn","onlyIPType":"ipv6"}`, false, "cn", lib.IPv6},
		{"invalid json", `{`, true, "", ""},
		{"empty", ``, true, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewStdinFromBytes(lib.ActionAdd, []byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			s := got.(*stdin)
			if s.Name != tt.wantName || s.OnlyIPType != tt.wantIP {
				t.Fatalf("got %+v", s)
			}
		})
	}
}

func TestStdinInput(t *testing.T) {
	input := strings.Join([]string{
		"1.0.0.0/24",
		"",
		"  # full comment",
		"// another comment",
		"/* block comment",
		"1.0.1.0/24 # trailing comment",
		"2.2.2.2 // trailing",
		"invalid-line",
		"2001:db8::/32 /* trailing */",
		"   ",
	}, "\n")

	tests := []struct {
		name    string
		action  lib.Action
		onlyIP  lib.IPType
		initial map[string][]string
		want    []string
	}{
		{"add all", lib.ActionAdd, "", nil, []string{"1.0.0.0/23", "2.2.2.2/32", "2001:db8::/32"}},
		{"add ipv4", lib.ActionAdd, lib.IPv4, nil, []string{"1.0.0.0/23", "2.2.2.2/32"}},
		{"add ipv6", lib.ActionAdd, lib.IPv6, nil, []string{"2001:db8::/32"}},
		{
			"remove all", lib.ActionRemove, "",
			map[string][]string{"cn": {"1.0.0.0/22", "2001:db8::/31"}},
			[]string{"1.0.2.0/23", "2001:db9::/32"},
		},
		{
			"remove ipv4", lib.ActionRemove, lib.IPv4,
			map[string][]string{"cn": {"1.0.0.0/22", "2001:db8::/31"}},
			[]string{"1.0.2.0/23", "2001:db8::/31"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withStdin(t, input)
			conv, err := NewStdin(tt.action, WithStdinName("cn"), WithStdinOnlyIPType(tt.onlyIP))
			if err != nil {
				t.Fatal(err)
			}
			c, err := conv.Input(newContainerWith(t, tt.initial))
			if err != nil {
				t.Fatal(err)
			}
			assertCIDRs(t, entryCIDRs(t, c, "cn"), tt.want)
		})
	}
}

func TestStdinInputErrors(t *testing.T) {
	tests := []struct {
		name      string
		action    lib.Action
		input     string
		container lib.Container
		wantIs    error
	}{
		{"unknown action", lib.ActionOutput, "1.1.1.1", lib.NewContainer(), lib.ErrUnknownAction},
		{"add error", lib.ActionAdd, "1.1.1.1", &mockContainer{Container: lib.NewContainer(), addErr: errMock}, errMock},
		{"remove error", lib.ActionRemove, "1.1.1.1", &mockContainer{Container: lib.NewContainer(), removeErr: errMock}, errMock},
		{"remove entry not found", lib.ActionRemove, "1.1.1.1", lib.NewContainer(), nil},
		{"scanner error", lib.ActionAdd, strings.Repeat("a", bufio.MaxScanTokenSize+1), lib.NewContainer(), bufio.ErrTooLong},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withStdin(t, tt.input)
			conv, _ := NewStdin(tt.action, WithStdinName("cn"))
			got, err := conv.Input(tt.container)
			if err == nil || got != nil {
				t.Fatalf("got (%v, %v), want error", got, err)
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Fatalf("err = %v, want %v", err, tt.wantIs)
			}
		})
	}
}
