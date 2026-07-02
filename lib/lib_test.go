package lib

import (
	"testing"
)

func TestActionConstants(t *testing.T) {
	tests := []struct {
		action Action
		want   string
	}{
		{ActionAdd, "add"},
		{ActionRemove, "remove"},
		{ActionOutput, "output"},
	}

	for _, tt := range tests {
		if string(tt.action) != tt.want {
			t.Errorf("Action constant = %s, want %s", tt.action, tt.want)
		}
	}
}

func TestIPTypeConstants(t *testing.T) {
	tests := []struct {
		ipType IPType
		want   string
	}{
		{IPv4, "ipv4"},
		{IPv6, "ipv6"},
	}

	for _, tt := range tests {
		if string(tt.ipType) != tt.want {
			t.Errorf("IPType constant = %s, want %s", tt.ipType, tt.want)
		}
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

func TestActionsRegistry(t *testing.T) {
	if !ActionsRegistry[ActionAdd] {
		t.Error("ActionAdd should be registered")
	}
	if !ActionsRegistry[ActionRemove] {
		t.Error("ActionRemove should be registered")
	}
	if !ActionsRegistry[ActionOutput] {
		t.Error("ActionOutput should be registered")
	}
	if ActionsRegistry["unknown"] {
		t.Error("unknown action should not be registered")
	}
}

func TestIgnoreIPv4(t *testing.T) {
	ipType := IgnoreIPv4()
	if ipType != IPv4 {
		t.Errorf("IgnoreIPv4() = %s, want %s", ipType, IPv4)
	}
}

func TestIgnoreIPv6(t *testing.T) {
	ipType := IgnoreIPv6()
	if ipType != IPv6 {
		t.Errorf("IgnoreIPv6() = %s, want %s", ipType, IPv6)
	}
}

func TestIgnoreIPOption(t *testing.T) {
	// Test that IgnoreIPOption functions return correct types
	var opt4 IgnoreIPOption = IgnoreIPv4
	var opt6 IgnoreIPOption = IgnoreIPv6

	if opt4() != IPv4 {
		t.Errorf("IgnoreIPv4 option returned %s, want %s", opt4(), IPv4)
	}
	if opt6() != IPv6 {
		t.Errorf("IgnoreIPv6 option returned %s, want %s", opt6(), IPv6)
	}
}
