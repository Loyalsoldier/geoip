package lib

import (
	"testing"
)

func TestConstants(t *testing.T) {
	if ActionAdd != "add" {
		t.Errorf("expected ActionAdd to be 'add', got %q", ActionAdd)
	}
	if ActionRemove != "remove" {
		t.Errorf("expected ActionRemove to be 'remove', got %q", ActionRemove)
	}
	if ActionOutput != "output" {
		t.Errorf("expected ActionOutput to be 'output', got %q", ActionOutput)
	}
	if IPv4 != "ipv4" {
		t.Errorf("expected IPv4 to be 'ipv4', got %q", IPv4)
	}
	if IPv6 != "ipv6" {
		t.Errorf("expected IPv6 to be 'ipv6', got %q", IPv6)
	}
	if CaseRemovePrefix != 0 {
		t.Errorf("expected CaseRemovePrefix to be 0, got %d", CaseRemovePrefix)
	}
	if CaseRemoveEntry != 1 {
		t.Errorf("expected CaseRemoveEntry to be 1, got %d", CaseRemoveEntry)
	}
}

func TestActionsRegistry(t *testing.T) {
	expected := map[Action]bool{
		ActionAdd:    true,
		ActionRemove: true,
		ActionOutput: true,
	}
	for action, val := range expected {
		if ActionsRegistry[action] != val {
			t.Errorf("expected ActionsRegistry[%q] to be %v", action, val)
		}
	}
	if len(ActionsRegistry) != len(expected) {
		t.Errorf("expected ActionsRegistry to have %d entries, got %d", len(expected), len(ActionsRegistry))
	}
}

func TestIgnoreIPv4(t *testing.T) {
	result := IgnoreIPv4()
	if result != IPv4 {
		t.Errorf("expected IgnoreIPv4() to return IPv4, got %q", result)
	}
}

func TestIgnoreIPv6(t *testing.T) {
	result := IgnoreIPv6()
	if result != IPv6 {
		t.Errorf("expected IgnoreIPv6() to return IPv6, got %q", result)
	}
}

func TestIgnoreIPOptionType(t *testing.T) {
	var opt IgnoreIPOption = IgnoreIPv4
	if opt() != IPv4 {
		t.Error("IgnoreIPOption function should return IPv4")
	}
	opt = IgnoreIPv6
	if opt() != IPv6 {
		t.Error("IgnoreIPOption function should return IPv6")
	}
}
