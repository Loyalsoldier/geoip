package special

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

// Merged result of privateCIDRs: 224.0.0.0/4 + 240.0.0.0/4 (which includes
// 255.255.255.255/32) become 224.0.0.0/3, and ::/128 + ::1/128 become ::/127.
var (
	wantPrivateIPv4 = []string{
		"0.0.0.0/8",
		"10.0.0.0/8",
		"100.64.0.0/10",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"172.16.0.0/12",
		"192.0.0.0/24",
		"192.0.2.0/24",
		"192.88.99.0/24",
		"192.168.0.0/16",
		"198.18.0.0/15",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"224.0.0.0/3",
	}
	wantPrivateIPv6 = []string{
		"::/127",
		"fc00::/7",
		"fe80::/10",
		"ff00::/8",
	}
)

func TestNewPrivate(t *testing.T) {
	tests := []struct {
		name       string
		action     lib.Action
		opts       []lib.InputOption
		wantIPType lib.IPType
	}{
		{name: "add action without options", action: lib.ActionAdd},
		{name: "remove action with onlyIPType ipv4", action: lib.ActionRemove, opts: []lib.InputOption{WithPrivateOnlyIPType(lib.IPv4)}, wantIPType: lib.IPv4},
		{name: "add action with onlyIPType ipv6", action: lib.ActionAdd, opts: []lib.InputOption{WithPrivateOnlyIPType(lib.IPv6)}, wantIPType: lib.IPv6},
		{name: "nil options are ignored", action: lib.ActionAdd, opts: []lib.InputOption{nil, WithPrivateOnlyIPType(lib.IPv4), nil}, wantIPType: lib.IPv4},
		{name: "unknown action is accepted by constructor", action: lib.Action("unknown")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewPrivate(tt.action, tt.opts...)
			if err != nil {
				t.Fatalf("NewPrivate() error = %v", err)
			}
			p, ok := got.(*private)
			if !ok {
				t.Fatalf("NewPrivate() returned %T, want *private", got)
			}
			if p.GetType() != TypePrivate {
				t.Errorf("GetType() = %q, want %q", p.GetType(), TypePrivate)
			}
			if p.GetAction() != tt.action {
				t.Errorf("GetAction() = %q, want %q", p.GetAction(), tt.action)
			}
			if p.GetDescription() != DescPrivate {
				t.Errorf("GetDescription() = %q, want %q", p.GetDescription(), DescPrivate)
			}
			if p.OnlyIPType != tt.wantIPType {
				t.Errorf("OnlyIPType = %q, want %q", p.OnlyIPType, tt.wantIPType)
			}
		})
	}
}

func TestNewPrivateFromBytes(t *testing.T) {
	tests := []struct {
		name          string
		action        lib.Action
		data          string
		wantJSONError bool
		wantIPType    lib.IPType
	}{
		{name: "empty data", action: lib.ActionAdd, data: ""},
		{name: "empty JSON object", action: lib.ActionRemove, data: `{}`},
		{name: "onlyIPType ipv4", action: lib.ActionAdd, data: `{"onlyIPType": "ipv4"}`, wantIPType: lib.IPv4},
		{name: "onlyIPType ipv6", action: lib.ActionRemove, data: `{"onlyIPType": "ipv6"}`, wantIPType: lib.IPv6},
		{name: "invalid JSON", action: lib.ActionAdd, data: `{"onlyIPType":`, wantJSONError: true},
		{name: "wrong JSON type", action: lib.ActionAdd, data: `{"onlyIPType": 4}`, wantJSONError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewPrivateFromBytes(tt.action, []byte(tt.data))
			if tt.wantJSONError {
				if err == nil {
					t.Fatalf("NewPrivateFromBytes() error = nil, want JSON error")
				}
				if got != nil {
					t.Fatalf("NewPrivateFromBytes() = %v, want nil", got)
				}
				var syntaxErr *json.SyntaxError
				var typeErr *json.UnmarshalTypeError
				if !errors.As(err, &syntaxErr) && !errors.As(err, &typeErr) {
					t.Fatalf("error = %v, want wrapped JSON error", err)
				}
				wantPrefix := "❌ [type private | action " + string(tt.action) + "] "
				if !strings.HasPrefix(err.Error(), wantPrefix) {
					t.Fatalf("error = %q, want prefix %q", err.Error(), wantPrefix)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewPrivateFromBytes() error = %v", err)
			}
			p := got.(*private)
			if p.Type != TypePrivate || p.Action != tt.action || p.Description != DescPrivate {
				t.Errorf("got Type=%q Action=%q Description=%q", p.Type, p.Action, p.Description)
			}
			if p.OnlyIPType != tt.wantIPType {
				t.Errorf("OnlyIPType = %q, want %q", p.OnlyIPType, tt.wantIPType)
			}
		})
	}
}

func TestPrivateInput(t *testing.T) {
	existing := testEntry{name: "private", cidrs: []string{"1.1.1.1/32", "10.0.0.0/7", "2001:db8::/32", "fe80::/9"}}
	other := testEntry{name: "cn", cidrs: []string{"1.0.1.0/24", "2001:250::/35"}}

	tests := []struct {
		name        string
		action      lib.Action
		onlyIPType  lib.IPType
		entries     []testEntry
		wantErr     string
		wantErrIs   error
		wantLen     int
		wantEntries []testEntry
	}{
		{
			name:        "add to empty container",
			action:      lib.ActionAdd,
			wantLen:     1,
			wantEntries: []testEntry{{name: "private", cidrs: slices.Concat(wantPrivateIPv4, wantPrivateIPv6)}},
		},
		{
			name:        "add only IPv4",
			action:      lib.ActionAdd,
			onlyIPType:  lib.IPv4,
			wantLen:     1,
			wantEntries: []testEntry{{name: "private", cidrs: wantPrivateIPv4}},
		},
		{
			name:        "add only IPv6",
			action:      lib.ActionAdd,
			onlyIPType:  lib.IPv6,
			wantLen:     1,
			wantEntries: []testEntry{{name: "private", cidrs: wantPrivateIPv6}},
		},
		{
			name:    "add merges into existing entry and keeps other entries",
			action:  lib.ActionAdd,
			entries: []testEntry{{name: "private", cidrs: []string{"1.1.1.1/32", "2001:db8::/32"}}, other},
			wantLen: 2,
			wantEntries: []testEntry{
				{name: "private", cidrs: slices.Concat(
					[]string{"0.0.0.0/8", "1.1.1.1/32"}, wantPrivateIPv4[1:],
					wantPrivateIPv6[:1], []string{"2001:db8::/32"}, wantPrivateIPv6[1:],
				)},
				other,
			},
		},
		{
			name:       "add only IPv4 merges into existing entry",
			action:     lib.ActionAdd,
			onlyIPType: lib.IPv4,
			entries:    []testEntry{{name: "private", cidrs: []string{"2001:db8::/32"}}},
			wantLen:    1,
			wantEntries: []testEntry{
				{name: "private", cidrs: slices.Concat(wantPrivateIPv4, []string{"2001:db8::/32"})},
			},
		},
		{
			name:    "remove from existing entry",
			action:  lib.ActionRemove,
			entries: []testEntry{existing, other},
			wantLen: 2,
			wantEntries: []testEntry{
				{name: "private", cidrs: []string{"1.1.1.1/32", "11.0.0.0/8", "2001:db8::/32", "fec0::/10"}},
				other,
			},
		},
		{
			name:       "remove only IPv4",
			action:     lib.ActionRemove,
			onlyIPType: lib.IPv4,
			entries:    []testEntry{existing},
			wantLen:    1,
			wantEntries: []testEntry{
				{name: "private", cidrs: []string{"1.1.1.1/32", "11.0.0.0/8", "2001:db8::/32", "fe80::/9"}},
			},
		},
		{
			name:       "remove only IPv6",
			action:     lib.ActionRemove,
			onlyIPType: lib.IPv6,
			entries:    []testEntry{existing},
			wantLen:    1,
			wantEntries: []testEntry{
				{name: "private", cidrs: []string{"1.1.1.1/32", "10.0.0.0/7", "2001:db8::/32", "fec0::/10"}},
			},
		},
		{
			name:    "remove everything leaves an empty entry",
			action:  lib.ActionRemove,
			entries: []testEntry{{name: "private", cidrs: []string{"10.0.0.0/8", "fc00::/7"}}},
			wantLen: 1,
			wantEntries: []testEntry{
				{name: "private", cidrs: nil},
			},
		},
		{
			name:    "remove when entry is missing",
			action:  lib.ActionRemove,
			entries: []testEntry{other},
			wantErr: "❌ [type private | action remove] entry PRIVATE not found",
		},
		{
			name:      "unknown action",
			action:    lib.Action("unknown"),
			wantErr:   "❌ [type private | action unknown] unknown action",
			wantErrIs: lib.ErrUnknownAction,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := newTestContainer(t, tt.entries...)

			p, err := NewPrivate(tt.action, WithPrivateOnlyIPType(tt.onlyIPType))
			if err != nil {
				t.Fatalf("NewPrivate() error = %v", err)
			}

			got, err := p.Input(container)
			if tt.wantErr != "" {
				assertErrorString(t, err, tt.wantErr)
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("errors.Is(%v, %v) = false, want true", err, tt.wantErrIs)
				}
				if got != nil {
					t.Fatalf("Input() container = %v, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Input() error = %v", err)
			}
			if got != container {
				t.Fatalf("Input() returned a different container")
			}
			if got.Len() != tt.wantLen {
				t.Fatalf("container.Len() = %d, want %d", got.Len(), tt.wantLen)
			}
			for _, want := range tt.wantEntries {
				assertEntryCIDRs(t, got, want.name, want.cidrs)
			}
		})
	}
}

func TestPrivateInputIsRepeatable(t *testing.T) {
	p, err := NewPrivate(lib.ActionAdd)
	if err != nil {
		t.Fatalf("NewPrivate() error = %v", err)
	}

	for i := range 2 {
		container := lib.NewContainer()
		if _, err := p.Input(container); err != nil {
			t.Fatalf("run %d: Input() error = %v", i, err)
		}
		assertEntryCIDRs(t, container, "private", slices.Concat(wantPrivateIPv4, wantPrivateIPv6))
	}
}

func TestPrivateConfigCreator(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		wantInitErr bool
		want        []string
	}{
		{
			name:   "add without args",
			config: `{"input": [{"type": "private", "action": "add"}]}`,
			want:   slices.Concat(wantPrivateIPv4, wantPrivateIPv6),
		},
		{
			name:   "add with onlyIPType",
			config: `{"input": [{"type": "PRIVATE", "action": "add", "args": {"onlyIPType": "ipv6"}}]}`,
			want:   wantPrivateIPv6,
		},
		{
			name:        "invalid args",
			config:      `{"input": [{"type": "private", "action": "add", "args": {"onlyIPType": true}}]}`,
			wantInitErr: true,
		},
		{
			name:        "invalid action",
			config:      `{"input": [{"type": "private", "action": "bogus"}]}`,
			wantInitErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance, err := lib.NewInstance()
			if err != nil {
				t.Fatalf("lib.NewInstance() error = %v", err)
			}
			err = instance.InitConfigFromBytes([]byte(tt.config))
			if tt.wantInitErr {
				if err == nil {
					t.Fatalf("InitConfigFromBytes() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("InitConfigFromBytes() error = %v", err)
			}

			container := lib.NewContainer()
			if err := instance.RunInput(container); err != nil {
				t.Fatalf("RunInput() error = %v", err)
			}
			assertEntryCIDRs(t, container, "private", tt.want)
		})
	}
}

func TestPrivateInputErrors(t *testing.T) {
	tests := []struct {
		name       string
		action     lib.Action
		cidrs      []string
		failAdd    bool
		failRemove bool
		wantErr    string
		wantErrIs  error
	}{
		{
			name:      "container add fails",
			action:    lib.ActionAdd,
			failAdd:   true,
			wantErr:   "❌ [type private | action add] fake container failure",
			wantErrIs: errFakeContainer,
		},
		{
			name:       "container remove fails",
			action:     lib.ActionRemove,
			failRemove: true,
			wantErr:    "❌ [type private | action remove] fake container failure",
			wantErrIs:  errFakeContainer,
		},
		{
			name:      "invalid built-in CIDR",
			action:    lib.ActionAdd,
			cidrs:     []string{"10.0.0.0/8", "not-a-cidr/8"},
			wantErr:   "❌ [type private | action add] invalid CIDR",
			wantErrIs: lib.ErrInvalidCIDR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cidrs != nil {
				orig := privateCIDRs
				privateCIDRs = tt.cidrs
				t.Cleanup(func() { privateCIDRs = orig })
			}

			inner := newTestContainer(t, testEntry{name: "private", cidrs: []string{"10.0.0.0/8"}})
			container := &failingContainer{Container: inner, failAdd: tt.failAdd, failRemove: tt.failRemove}

			p, err := NewPrivate(tt.action)
			if err != nil {
				t.Fatalf("NewPrivate() error = %v", err)
			}

			got, err := p.Input(container)
			assertErrorString(t, err, tt.wantErr)
			if !errors.Is(err, tt.wantErrIs) {
				t.Errorf("errors.Is(%v, %v) = false, want true", err, tt.wantErrIs)
			}
			if got != nil {
				t.Fatalf("Input() container = %v, want nil", got)
			}
			assertEntryCIDRs(t, inner, "private", []string{"10.0.0.0/8"})
		})
	}
}
