package lib

import (
	"testing"
)

func TestConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"ActionAdd", ActionAdd, Action("add")},
		{"ActionRemove", ActionRemove, Action("remove")},
		{"ActionOutput", ActionOutput, Action("output")},
		{"IPv4", IPv4, IPType("ipv4")},
		{"IPv6", IPv6, IPType("ipv6")},
		{"CaseRemovePrefix", CaseRemovePrefix, CaseRemove(0)},
		{"CaseRemoveEntry", CaseRemoveEntry, CaseRemove(1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("got %v, expected %v", tt.got, tt.expected)
			}
		})
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
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			got := ActionsRegistry[tt.action]
			if got != tt.expected {
				t.Errorf("ActionsRegistry[%s] = %v, expected %v", tt.action, got, tt.expected)
			}
		})
	}
}

func TestIgnoreIPv4(t *testing.T) {
	result := IgnoreIPv4()
	if result != IPv4 {
		t.Errorf("IgnoreIPv4() = %v, expected %v", result, IPv4)
	}
}

func TestIgnoreIPv6(t *testing.T) {
	result := IgnoreIPv6()
	if result != IPv6 {
		t.Errorf("IgnoreIPv6() = %v, expected %v", result, IPv6)
	}
}
