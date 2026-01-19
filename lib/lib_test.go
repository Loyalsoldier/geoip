package lib

import "testing"

func TestConstants(t *testing.T) {
	tests := []struct {
		name     string
		action   Action
		expected bool
	}{
		{"ActionAdd in registry", ActionAdd, true},
		{"ActionRemove in registry", ActionRemove, true},
		{"ActionOutput in registry", ActionOutput, true},
		{"Invalid action not in registry", Action("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ActionsRegistry[tt.action]; got != tt.expected {
				t.Errorf("ActionsRegistry[%s] = %v, want %v", tt.action, got, tt.expected)
			}
		})
	}
}

func TestActionConstants(t *testing.T) {
	if ActionAdd != "add" {
		t.Errorf("ActionAdd = %s, want 'add'", ActionAdd)
	}
	if ActionRemove != "remove" {
		t.Errorf("ActionRemove = %s, want 'remove'", ActionRemove)
	}
	if ActionOutput != "output" {
		t.Errorf("ActionOutput = %s, want 'output'", ActionOutput)
	}
}

func TestIPTypeConstants(t *testing.T) {
	if IPv4 != "ipv4" {
		t.Errorf("IPv4 = %s, want 'ipv4'", IPv4)
	}
	if IPv6 != "ipv6" {
		t.Errorf("IPv6 = %s, want 'ipv6'", IPv6)
	}
}

func TestCaseRemoveConstants(t *testing.T) {
	if CaseRemovePrefix != 0 {
		t.Errorf("CaseRemovePrefix = %d, want 0", CaseRemovePrefix)
	}
	if CaseRemoveEntry != 1 {
		t.Errorf("CaseRemoveEntry = %d, want 1", CaseRemoveEntry)
	}
}

func TestIgnoreIPv4(t *testing.T) {
	if got := IgnoreIPv4(); got != IPv4 {
		t.Errorf("IgnoreIPv4() = %v, want %v", got, IPv4)
	}
}

func TestIgnoreIPv6(t *testing.T) {
	if got := IgnoreIPv6(); got != IPv6 {
		t.Errorf("IgnoreIPv6() = %v, want %v", got, IPv6)
	}
}
