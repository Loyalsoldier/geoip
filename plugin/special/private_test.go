package special

import (
	"errors"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

var (
	privateIPv4Want = []string{
		"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16",
		"172.16.0.0/12", "192.0.0.0/24", "192.0.2.0/24", "192.31.196.0/24", "192.52.193.0/24",
		"192.88.99.0/24", "192.168.0.0/16", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24",
		"224.0.0.0/3",
	}
	privateIPv6Want = []string{
		"::/127", "64:ff9b::/96", "64:ff9b:1::/48", "100::/64", "2001::/23", "2001:db8::/32",
		"2620:4f:8000::/48", "3fff::/20", "5f00::/16", "fc00::/7", "fe80::/10", "ff00::/8",
	}
)

func TestNewPrivate(t *testing.T) {
	tests := []struct {
		name   string
		action lib.Action
		opts   []lib.InputOption
		want   lib.IPType
	}{
		{"add", lib.ActionAdd, nil, ""},
		{"remove ipv4", lib.ActionRemove, []lib.InputOption{nil, WithPrivateOnlyIPType(lib.IPv4)}, lib.IPv4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewPrivate(tt.action, tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			p := got.(*private)
			if p.GetType() != TypePrivate || p.GetAction() != tt.action || p.GetDescription() != DescPrivate || p.OnlyIPType != tt.want {
				t.Fatalf("got %+v", p)
			}
		})
	}
}

func TestNewPrivateFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		want    lib.IPType
	}{
		{"empty", ``, false, ""},
		{"ipv6", `{"onlyIPType":"ipv6"}`, false, lib.IPv6},
		{"invalid json", `{`, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewPrivateFromBytes(lib.ActionAdd, []byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got.(*private).OnlyIPType != tt.want {
				t.Fatalf("got %+v", got)
			}
		})
	}
}

func TestPrivateInput(t *testing.T) {
	all := append(append([]string{}, privateIPv4Want...), privateIPv6Want...)
	tests := []struct {
		name    string
		action  lib.Action
		onlyIP  lib.IPType
		initial map[string][]string
		want    []string
	}{
		{"add all", lib.ActionAdd, "", nil, all},
		{"add ipv4 only", lib.ActionAdd, lib.IPv4, nil, privateIPv4Want},
		{"add ipv6 only", lib.ActionAdd, lib.IPv6, nil, privateIPv6Want},
		{
			"remove all", lib.ActionRemove, "",
			map[string][]string{"private": {"10.0.0.0/8", "1.1.1.0/24", "fc00::/7", "2400::/32"}},
			[]string{"1.1.1.0/24", "2400::/32"},
		},
		{
			"remove ipv4 only", lib.ActionRemove, lib.IPv4,
			map[string][]string{"private": {"10.0.0.0/8", "1.1.1.0/24", "fc00::/7"}},
			[]string{"1.1.1.0/24", "fc00::/7"},
		},
		{
			"remove ipv6 only", lib.ActionRemove, lib.IPv6,
			map[string][]string{"private": {"10.0.0.0/8", "fc00::/7", "2400::/32"}},
			[]string{"10.0.0.0/8", "2400::/32"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, _ := NewPrivate(tt.action, WithPrivateOnlyIPType(tt.onlyIP))
			c, err := conv.Input(newContainerWith(t, tt.initial))
			if err != nil {
				t.Fatal(err)
			}
			assertCIDRs(t, entryCIDRs(t, c, entryNamePrivate), tt.want)
		})
	}
}

func TestPrivateInputErrors(t *testing.T) {
	tests := []struct {
		name      string
		action    lib.Action
		container lib.Container
		badCIDR   bool
		wantIs    error
	}{
		{"unknown action", lib.ActionOutput, lib.NewContainer(), false, lib.ErrUnknownAction},
		{"add error", lib.ActionAdd, &mockContainer{Container: lib.NewContainer(), addErr: errMock}, false, errMock},
		{"remove error", lib.ActionRemove, &mockContainer{Container: lib.NewContainer(), removeErr: errMock}, false, errMock},
		{"remove entry not found", lib.ActionRemove, lib.NewContainer(), false, nil},
		{"invalid cidr", lib.ActionAdd, lib.NewContainer(), true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.badCIDR {
				orig := privateCIDRs
				privateCIDRs = []string{"invalid"}
				t.Cleanup(func() { privateCIDRs = orig })
			}
			conv, _ := NewPrivate(tt.action)
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
