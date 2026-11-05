package lib

import (
	"errors"
	"net/netip"
	"testing"

	"go4.org/netipx"
)

// mockInputConverter is a configurable InputConverter used across tests.
type mockInputConverter struct {
	typ    string
	action Action
	desc   string
	err    error
	fn     func(Container) (Container, error)
}

func (m *mockInputConverter) GetType() string        { return m.typ }
func (m *mockInputConverter) GetAction() Action      { return m.action }
func (m *mockInputConverter) GetDescription() string { return m.desc }
func (m *mockInputConverter) Input(c Container) (Container, error) {
	if m.fn != nil {
		return m.fn(c)
	}
	return c, m.err
}

// mockOutputConverter is a configurable OutputConverter used across tests.
type mockOutputConverter struct {
	typ    string
	action Action
	desc   string
	err    error
	called int
}

func (m *mockOutputConverter) GetType() string        { return m.typ }
func (m *mockOutputConverter) GetAction() Action      { return m.action }
func (m *mockOutputConverter) GetDescription() string { return m.desc }
func (m *mockOutputConverter) Output(Container) error {
	m.called++
	return m.err
}

// brokenBuilder returns an IPSetBuilder whose IPSet() call returns an error.
func brokenBuilder() *netipx.IPSetBuilder {
	b := new(netipx.IPSetBuilder)
	b.AddPrefix(netip.Prefix{})
	return b
}

// assertStrings compares both the length and the exact content of two string slices.
func assertStrings(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d mismatch: got %q, want %q (got %v, want %v)", i, got[i], want[i], got, want)
		}
	}
}

func assertErr(t *testing.T, err, want error, wantErr bool) {
	t.Helper()
	if wantErr != (err != nil) {
		t.Fatalf("error = %v, wantErr %v", err, wantErr)
	}
	if want != nil && !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}

func TestIgnoreIPOptions(t *testing.T) {
	if got := IgnoreIPv4(); got != IPv4 {
		t.Errorf("IgnoreIPv4() = %q, want %q", got, IPv4)
	}
	if got := IgnoreIPv6(); got != IPv6 {
		t.Errorf("IgnoreIPv6() = %q, want %q", got, IPv6)
	}
}

func TestActionsRegistry(t *testing.T) {
	tests := []struct {
		action Action
		want   bool
	}{
		{ActionAdd, true},
		{ActionRemove, true},
		{ActionOutput, true},
		{Action("unknown"), false},
		{Action(""), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			if got := ActionsRegistry[tt.action]; got != tt.want {
				t.Errorf("ActionsRegistry[%q] = %v, want %v", tt.action, got, tt.want)
			}
		})
	}
}
