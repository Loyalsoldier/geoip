package special

import (
	"errors"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestNewTest(t *testing.T) {
	tests := []struct {
		name     string
		action   lib.Action
		opts     []lib.InputOption
		wantDesc string
	}{
		{name: "add action", action: lib.ActionAdd, wantDesc: descTest},
		{name: "remove action", action: lib.ActionRemove, wantDesc: descTest},
		{name: "unknown action is accepted by constructor", action: lib.Action("unknown"), wantDesc: descTest},
		{
			name:   "options are applied and nil options ignored",
			action: lib.ActionAdd,
			opts: []lib.InputOption{
				nil,
				func(c lib.InputConverter) { c.(*test).Description = "custom" },
			},
			wantDesc: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTest(tt.action, tt.opts...)
			if err != nil {
				t.Fatalf("newTest() error = %v", err)
			}
			c, ok := got.(*test)
			if !ok {
				t.Fatalf("newTest() returned %T, want *test", got)
			}
			if c.GetType() != typeTest {
				t.Errorf("GetType() = %q, want %q", c.GetType(), typeTest)
			}
			if c.GetAction() != tt.action {
				t.Errorf("GetAction() = %q, want %q", c.GetAction(), tt.action)
			}
			if c.GetDescription() != tt.wantDesc {
				t.Errorf("GetDescription() = %q, want %q", c.GetDescription(), tt.wantDesc)
			}
		})
	}
}

func TestNewTestFromBytes(t *testing.T) {
	tests := []struct {
		name   string
		action lib.Action
		data   []byte
	}{
		{name: "nil data", action: lib.ActionAdd},
		{name: "valid JSON is ignored", action: lib.ActionRemove, data: []byte(`{"foo": "bar"}`)},
		{name: "invalid JSON is ignored", action: lib.ActionAdd, data: []byte(`{`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTestFromBytes(tt.action, tt.data)
			if err != nil {
				t.Fatalf("newTestFromBytes() error = %v", err)
			}
			c := got.(*test)
			if c.Type != typeTest || c.Action != tt.action || c.Description != descTest {
				t.Errorf("got Type=%q Action=%q Description=%q", c.Type, c.Action, c.Description)
			}
		})
	}
}

func TestTestInput(t *testing.T) {
	tests := []struct {
		name        string
		action      lib.Action
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
			wantEntries: []testEntry{{name: "test", cidrs: []string{"127.0.0.0/8"}}},
		},
		{
			name:    "add merges into existing entry",
			action:  lib.ActionAdd,
			entries: []testEntry{{name: "test", cidrs: []string{"1.1.1.1/32", "126.0.0.0/8", "::1"}}, {name: "cn", cidrs: []string{"1.0.1.0/24"}}},
			wantLen: 2,
			wantEntries: []testEntry{
				{name: "test", cidrs: []string{"1.1.1.1/32", "126.0.0.0/7", "::1/128"}},
				{name: "cn", cidrs: []string{"1.0.1.0/24"}},
			},
		},
		{
			name:        "remove from existing entry",
			action:      lib.ActionRemove,
			entries:     []testEntry{{name: "test", cidrs: []string{"126.0.0.0/7", "2001:db8::/32"}}},
			wantLen:     1,
			wantEntries: []testEntry{{name: "test", cidrs: []string{"126.0.0.0/8", "2001:db8::/32"}}},
		},
		{
			name:        "remove everything leaves an empty entry",
			action:      lib.ActionRemove,
			entries:     []testEntry{{name: "test", cidrs: []string{"127.0.0.1"}}},
			wantLen:     1,
			wantEntries: []testEntry{{name: "test", cidrs: nil}},
		},
		{
			name:    "remove when entry is missing",
			action:  lib.ActionRemove,
			entries: []testEntry{{name: "cn", cidrs: []string{"127.0.0.0/8"}}},
			wantErr: "❌ [type test | action remove] entry TEST not found",
		},
		{
			name:      "unknown action",
			action:    lib.Action("unknown"),
			wantErr:   "❌ [type test | action unknown] unknown action",
			wantErrIs: lib.ErrUnknownAction,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := newTestContainer(t, tt.entries...)

			c, err := newTest(tt.action)
			if err != nil {
				t.Fatalf("newTest() error = %v", err)
			}

			got, err := c.Input(container)
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

func TestTestConfigCreator(t *testing.T) {
	instance, err := lib.NewInstance()
	if err != nil {
		t.Fatalf("lib.NewInstance() error = %v", err)
	}
	config := `{"input": [{"type": "test", "action": "add", "args": {"ignored": true}}]}`
	if err := instance.InitConfigFromBytes([]byte(config)); err != nil {
		t.Fatalf("InitConfigFromBytes() error = %v", err)
	}

	container := lib.NewContainer()
	if err := instance.RunInput(container); err != nil {
		t.Fatalf("RunInput() error = %v", err)
	}
	if container.Len() != 1 {
		t.Fatalf("container.Len() = %d, want 1", container.Len())
	}
	assertEntryCIDRs(t, container, "test", []string{"127.0.0.0/8"})
}

func TestTestInputErrors(t *testing.T) {
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
			wantErr:   "❌ [type test | action add] fake container failure",
			wantErrIs: errFakeContainer,
		},
		{
			name:       "container remove fails",
			action:     lib.ActionRemove,
			failRemove: true,
			wantErr:    "❌ [type test | action remove] fake container failure",
			wantErrIs:  errFakeContainer,
		},
		{
			name:      "invalid built-in CIDR",
			action:    lib.ActionAdd,
			cidrs:     []string{"not-an-ip"},
			wantErr:   "❌ [type test | action add] invalid IP address",
			wantErrIs: lib.ErrInvalidIP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cidrs != nil {
				orig := testCIDRs
				testCIDRs = tt.cidrs
				t.Cleanup(func() { testCIDRs = orig })
			}

			inner := newTestContainer(t, testEntry{name: "test", cidrs: []string{"127.0.0.0/8"}})
			container := &failingContainer{Container: inner, failAdd: tt.failAdd, failRemove: tt.failRemove}

			c, err := newTest(tt.action)
			if err != nil {
				t.Fatalf("newTest() error = %v", err)
			}

			got, err := c.Input(container)
			assertErrorString(t, err, tt.wantErr)
			if !errors.Is(err, tt.wantErrIs) {
				t.Errorf("errors.Is(%v, %v) = false, want true", err, tt.wantErrIs)
			}
			if got != nil {
				t.Fatalf("Input() container = %v, want nil", got)
			}
			assertEntryCIDRs(t, inner, "test", []string{"127.0.0.0/8"})
		})
	}
}
