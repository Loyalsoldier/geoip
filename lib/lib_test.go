package lib

import (
	"testing"
)

func TestConstants(t *testing.T) {
	// Test Action constants
	if ActionAdd != "add" {
		t.Errorf("ActionAdd = %q, want %q", ActionAdd, "add")
	}
	if ActionRemove != "remove" {
		t.Errorf("ActionRemove = %q, want %q", ActionRemove, "remove")
	}
	if ActionOutput != "output" {
		t.Errorf("ActionOutput = %q, want %q", ActionOutput, "output")
	}

	// Test IPType constants
	if IPv4 != "ipv4" {
		t.Errorf("IPv4 = %q, want %q", IPv4, "ipv4")
	}
	if IPv6 != "ipv6" {
		t.Errorf("IPv6 = %q, want %q", IPv6, "ipv6")
	}

	// Test CaseRemove constants
	if CaseRemovePrefix != 0 {
		t.Errorf("CaseRemovePrefix = %d, want %d", CaseRemovePrefix, 0)
	}
	if CaseRemoveEntry != 1 {
		t.Errorf("CaseRemoveEntry = %d, want %d", CaseRemoveEntry, 1)
	}
}

func TestActionsRegistry(t *testing.T) {
	tests := []struct {
		action   Action
		expected bool
	}{
		{ActionAdd, true},
		{ActionRemove, true},
		{ActionOutput, true},
		{Action("invalid"), false},
		{Action(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			got := ActionsRegistry[tt.action]
			if got != tt.expected {
				t.Errorf("ActionsRegistry[%q] = %v, want %v", tt.action, got, tt.expected)
			}
		})
	}
}

func TestIgnoreIPv4(t *testing.T) {
	result := IgnoreIPv4()
	if result != IPv4 {
		t.Errorf("IgnoreIPv4() = %q, want %q", result, IPv4)
	}
}

func TestIgnoreIPv6(t *testing.T) {
	result := IgnoreIPv6()
	if result != IPv6 {
		t.Errorf("IgnoreIPv6() = %q, want %q", result, IPv6)
	}
}

func TestIgnoreIPOption(t *testing.T) {
	// Test that IgnoreIPv4 returns correct IPType when called
	opt := IgnoreIPv4
	result := opt()
	if result != IPv4 {
		t.Errorf("IgnoreIPv4() = %q, want %q", result, IPv4)
	}

	// Test that IgnoreIPv6 returns correct IPType when called
	opt2 := IgnoreIPv6
	result2 := opt2()
	if result2 != IPv6 {
		t.Errorf("IgnoreIPv6() = %q, want %q", result2, IPv6)
	}
}
