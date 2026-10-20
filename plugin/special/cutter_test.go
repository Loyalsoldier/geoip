package special

import (
	"encoding/json"
	"errors"
	"maps"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestNewCutter(t *testing.T) {
	tests := []struct {
		name       string
		action     lib.Action
		opts       []lib.InputOption
		wantErr    string
		wantWant   map[string]bool
		wantIPType lib.IPType
	}{
		{
			name:   "remove action with normalized wanted list",
			action: lib.ActionRemove,
			opts: []lib.InputOption{
				WithCutterWantedList([]string{" cn ", "Us", "", "   ", "cn"}),
			},
			wantWant: map[string]bool{"CN": true, "US": true},
		},
		{
			name:   "remove action with onlyIPType ipv4",
			action: lib.ActionRemove,
			opts: []lib.InputOption{
				WithCutterWantedList([]string{"cn"}),
				WithCutterOnlyIPType(lib.IPv4),
			},
			wantWant:   map[string]bool{"CN": true},
			wantIPType: lib.IPv4,
		},
		{
			name:   "remove action with onlyIPType ipv6",
			action: lib.ActionRemove,
			opts: []lib.InputOption{
				WithCutterOnlyIPType(lib.IPv6),
				WithCutterWantedList([]string{"jp"}),
			},
			wantWant:   map[string]bool{"JP": true},
			wantIPType: lib.IPv6,
		},
		{
			name:     "nil options are ignored",
			action:   lib.ActionRemove,
			opts:     []lib.InputOption{nil, WithCutterWantedList([]string{"cn"}), nil},
			wantWant: map[string]bool{"CN": true},
		},
		{
			name:    "add action is rejected",
			action:  lib.ActionAdd,
			opts:    []lib.InputOption{WithCutterWantedList([]string{"cn"})},
			wantErr: "❌ [type cutter | action add] only supports `remove` action",
		},
		{
			name:    "output action is rejected",
			action:  lib.ActionOutput,
			opts:    []lib.InputOption{WithCutterWantedList([]string{"cn"})},
			wantErr: "❌ [type cutter | action output] only supports `remove` action",
		},
		{
			name:    "unknown action is rejected",
			action:  lib.Action("unknown"),
			opts:    []lib.InputOption{WithCutterWantedList([]string{"cn"})},
			wantErr: "❌ [type cutter | action unknown] only supports `remove` action",
		},
		{
			name:    "missing wanted list",
			action:  lib.ActionRemove,
			wantErr: "❌ [type cutter | action remove] wantedList must be specified",
		},
		{
			name:    "blank wanted list",
			action:  lib.ActionRemove,
			opts:    []lib.InputOption{WithCutterWantedList([]string{"", "  "})},
			wantErr: "❌ [type cutter | action remove] wantedList must be specified",
		},
		{
			name:    "action is checked before wanted list",
			action:  lib.ActionAdd,
			wantErr: "❌ [type cutter | action add] only supports `remove` action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCutter(tt.action, tt.opts...)
			if tt.wantErr != "" {
				assertErrorString(t, err, tt.wantErr)
				if got != nil {
					t.Fatalf("NewCutter() = %v, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewCutter() error = %v", err)
			}

			c, ok := got.(*cutter)
			if !ok {
				t.Fatalf("NewCutter() returned %T, want *cutter", got)
			}
			if c.GetType() != TypeCutter {
				t.Errorf("GetType() = %q, want %q", c.GetType(), TypeCutter)
			}
			if c.GetAction() != tt.action {
				t.Errorf("GetAction() = %q, want %q", c.GetAction(), tt.action)
			}
			if c.GetDescription() != DescCutter {
				t.Errorf("GetDescription() = %q, want %q", c.GetDescription(), DescCutter)
			}
			if !maps.Equal(c.Want, tt.wantWant) {
				t.Errorf("Want = %v, want %v", c.Want, tt.wantWant)
			}
			if c.OnlyIPType != tt.wantIPType {
				t.Errorf("OnlyIPType = %q, want %q", c.OnlyIPType, tt.wantIPType)
			}
		})
	}
}

func TestNewCutterFromBytes(t *testing.T) {
	tests := []struct {
		name          string
		action        lib.Action
		data          string
		wantErr       string
		wantJSONError bool
		wantWant      map[string]bool
		wantIPType    lib.IPType
	}{
		{
			name:       "valid JSON with all fields",
			action:     lib.ActionRemove,
			data:       `{"wantedList": ["cn", " us "], "onlyIPType": "ipv6"}`,
			wantWant:   map[string]bool{"CN": true, "US": true},
			wantIPType: lib.IPv6,
		},
		{
			name:     "valid JSON without onlyIPType",
			action:   lib.ActionRemove,
			data:     `{"wantedList": ["jp"]}`,
			wantWant: map[string]bool{"JP": true},
		},
		{
			name:    "empty data",
			action:  lib.ActionRemove,
			data:    "",
			wantErr: "❌ [type cutter | action remove] wantedList must be specified",
		},
		{
			name:    "empty JSON object",
			action:  lib.ActionRemove,
			data:    `{}`,
			wantErr: "❌ [type cutter | action remove] wantedList must be specified",
		},
		{
			name:    "valid JSON with add action",
			action:  lib.ActionAdd,
			data:    `{"wantedList": ["cn"]}`,
			wantErr: "❌ [type cutter | action add] only supports `remove` action",
		},
		{
			name:          "invalid JSON",
			action:        lib.ActionRemove,
			data:          `{"wantedList": [`,
			wantJSONError: true,
		},
		{
			name:          "wrong JSON type",
			action:        lib.ActionRemove,
			data:          `{"wantedList": "cn"}`,
			wantJSONError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCutterFromBytes(tt.action, []byte(tt.data))
			switch {
			case tt.wantJSONError:
				if err == nil {
					t.Fatalf("NewCutterFromBytes() error = nil, want JSON error")
				}
				if got != nil {
					t.Fatalf("NewCutterFromBytes() = %v, want nil", got)
				}
				var syntaxErr *json.SyntaxError
				var typeErr *json.UnmarshalTypeError
				if !errors.As(err, &syntaxErr) && !errors.As(err, &typeErr) {
					t.Fatalf("error = %v, want wrapped JSON error", err)
				}
				wantPrefix := "❌ [type cutter | action " + string(tt.action) + "] "
				if !strings.HasPrefix(err.Error(), wantPrefix) {
					t.Fatalf("error = %q, want prefix %q", err.Error(), wantPrefix)
				}
				return
			case tt.wantErr != "":
				assertErrorString(t, err, tt.wantErr)
				if got != nil {
					t.Fatalf("NewCutterFromBytes() = %v, want nil", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("NewCutterFromBytes() error = %v", err)
			}
			c := got.(*cutter)
			if c.Type != TypeCutter || c.Action != tt.action || c.Description != DescCutter {
				t.Errorf("got Type=%q Action=%q Description=%q", c.Type, c.Action, c.Description)
			}
			if !maps.Equal(c.Want, tt.wantWant) {
				t.Errorf("Want = %v, want %v", c.Want, tt.wantWant)
			}
			if c.OnlyIPType != tt.wantIPType {
				t.Errorf("OnlyIPType = %q, want %q", c.OnlyIPType, tt.wantIPType)
			}
		})
	}
}

func TestCutterInput(t *testing.T) {
	initial := []testEntry{
		{name: "cn", cidrs: []string{"1.0.1.0/24", "2001:250::/35"}},
		{name: "us", cidrs: []string{"3.0.0.0/8", "2600::/16"}},
		{name: "jp", cidrs: []string{"1.0.16.0/20", "2001:200::/37"}},
		{name: "v6only", cidrs: []string{"2001:db8::/32"}},
	}

	tests := []struct {
		name        string
		entries     []testEntry
		wantedList  []string
		onlyIPType  lib.IPType
		wantLen     int
		wantEntries []testEntry
		wantAbsent  []string
	}{
		{
			name:       "remove whole entries",
			entries:    initial,
			wantedList: []string{"cn", "us"},
			wantLen:    2,
			wantEntries: []testEntry{
				{name: "jp", cidrs: []string{"1.0.16.0/20", "2001:200::/37"}},
				{name: "v6only", cidrs: []string{"2001:db8::/32"}},
			},
			wantAbsent: []string{"cn", "us"},
		},
		{
			name:       "wanted list is case and space insensitive",
			entries:    initial,
			wantedList: []string{" Jp ", "V6ONLY"},
			wantLen:    2,
			wantEntries: []testEntry{
				{name: "cn", cidrs: []string{"1.0.1.0/24", "2001:250::/35"}},
				{name: "us", cidrs: []string{"3.0.0.0/8", "2600::/16"}},
			},
			wantAbsent: []string{"jp", "v6only"},
		},
		{
			name:       "onlyIPType ipv4 removes only the IPv4 part",
			entries:    initial,
			wantedList: []string{"cn", "us", "v6only"},
			onlyIPType: lib.IPv4,
			wantLen:    4,
			wantEntries: []testEntry{
				{name: "cn", cidrs: []string{"2001:250::/35"}},
				{name: "us", cidrs: []string{"2600::/16"}},
				{name: "jp", cidrs: []string{"1.0.16.0/20", "2001:200::/37"}},
				{name: "v6only", cidrs: []string{"2001:db8::/32"}},
			},
		},
		{
			name:       "onlyIPType ipv6 removes only the IPv6 part",
			entries:    initial,
			wantedList: []string{"cn", "us", "v6only"},
			onlyIPType: lib.IPv6,
			wantLen:    4,
			wantEntries: []testEntry{
				{name: "cn", cidrs: []string{"1.0.1.0/24"}},
				{name: "us", cidrs: []string{"3.0.0.0/8"}},
				{name: "jp", cidrs: []string{"1.0.16.0/20", "2001:200::/37"}},
				{name: "v6only", cidrs: nil}, // entry kept but now empty
			},
		},
		{
			name:       "wanted entries missing from container",
			entries:    initial,
			wantedList: []string{"xx", "yy"},
			wantLen:    4,
			wantEntries: []testEntry{
				{name: "cn", cidrs: []string{"1.0.1.0/24", "2001:250::/35"}},
				{name: "us", cidrs: []string{"3.0.0.0/8", "2600::/16"}},
				{name: "jp", cidrs: []string{"1.0.16.0/20", "2001:200::/37"}},
				{name: "v6only", cidrs: []string{"2001:db8::/32"}},
			},
		},
		{
			name:       "empty container",
			wantedList: []string{"cn"},
			wantLen:    0,
			wantAbsent: []string{"cn"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := newTestContainer(t, tt.entries...)

			c, err := NewCutter(lib.ActionRemove, WithCutterWantedList(tt.wantedList), WithCutterOnlyIPType(tt.onlyIPType))
			if err != nil {
				t.Fatalf("NewCutter() error = %v", err)
			}

			got, err := c.Input(container)
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
			for _, name := range tt.wantAbsent {
				assertEntryAbsent(t, got, name)
			}
		})
	}
}

func TestCutterConfigCreator(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		wantInitErr bool
		wantEntries []testEntry
		wantAbsent  []string
	}{
		{
			name:        "valid config",
			config:      `{"input": [{"type": "cutter", "action": "remove", "args": {"wantedList": ["cn"], "onlyIPType": "ipv4"}}]}`,
			wantEntries: []testEntry{{name: "cn", cidrs: []string{"2001:250::/35"}}, {name: "us", cidrs: []string{"3.0.0.0/8"}}},
		},
		{
			name:        "valid config removing whole entry",
			config:      `{"input": [{"type": "Cutter", "action": "remove", "args": {"wantedList": ["us"]}}]}`,
			wantEntries: []testEntry{{name: "cn", cidrs: []string{"1.0.1.0/24", "2001:250::/35"}}},
			wantAbsent:  []string{"us"},
		},
		{
			name:        "add action is rejected",
			config:      `{"input": [{"type": "cutter", "action": "add", "args": {"wantedList": ["cn"]}}]}`,
			wantInitErr: true,
		},
		{
			name:        "missing args",
			config:      `{"input": [{"type": "cutter", "action": "remove"}]}`,
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

			container := newTestContainer(t,
				testEntry{name: "cn", cidrs: []string{"1.0.1.0/24", "2001:250::/35"}},
				testEntry{name: "us", cidrs: []string{"3.0.0.0/8"}},
			)
			if err := instance.RunInput(container); err != nil {
				t.Fatalf("RunInput() error = %v", err)
			}
			for _, want := range tt.wantEntries {
				assertEntryCIDRs(t, container, want.name, want.cidrs)
			}
			for _, name := range tt.wantAbsent {
				assertEntryAbsent(t, container, name)
			}
		})
	}
}

func TestCutterInputContainerError(t *testing.T) {
	inner := newTestContainer(t, testEntry{name: "cn", cidrs: []string{"1.0.1.0/24"}})
	container := &failingContainer{Container: inner, failRemove: true}

	c, err := NewCutter(lib.ActionRemove, WithCutterWantedList([]string{"cn"}))
	if err != nil {
		t.Fatalf("NewCutter() error = %v", err)
	}

	got, err := c.Input(container)
	assertErrorString(t, err, "❌ [type cutter | action remove] fake container failure")
	if !errors.Is(err, errFakeContainer) {
		t.Errorf("errors.Is(%v, errFakeContainer) = false, want true", err)
	}
	if got != nil {
		t.Fatalf("Input() container = %v, want nil", got)
	}
	assertEntryCIDRs(t, inner, "cn", []string{"1.0.1.0/24"})
}
