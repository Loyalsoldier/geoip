package lib

import (
	"net"
	"testing"
)

// Additional comprehensive coverage tests

func TestEntry_ProcessPrefix_NetIPNetEdgeCases(t *testing.T) {
	entry := NewEntry("test")

	// Test with *net.IPNet IPv6
	_, ipnet, _ := net.ParseCIDR("2001:db8::/32")
	err := entry.AddPrefix(ipnet)
	if err != nil {
		t.Errorf("Entry.AddPrefix(*net.IPNet IPv6) error = %v, want nil", err)
	}
}

func TestContainer_AddExistingWithNilBuilders(t *testing.T) {
	container := NewContainer()

	// Add first entry with IPv4
	entry1 := NewEntry("test")
	entry1.AddPrefix("192.168.1.0/24")
	container.Add(entry1)

	// Add second entry with IPv6 and default ignore option (no ignore)
	entry2 := NewEntry("test")
	entry2.AddPrefix("2001:db8::/32")
	err := container.Add(entry2, nil)
	if err != nil {
		t.Errorf("Container.Add() with nil option error = %v, want nil", err)
	}
}

func TestContainer_RemoveWithNilBuilders(t *testing.T) {
	container := NewContainer()

	// Add entry with IPv4
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	container.Add(entry)

	// Remove with entry that has no builders created
	removeEntry := NewEntry("test")
	removeEntry.AddPrefix("10.0.0.0/8")

	// Remove prefixes with no ignore option
	err := container.Remove(removeEntry, CaseRemovePrefix, nil)
	if err != nil {
		t.Errorf("Container.Remove() with nil option error = %v, want nil", err)
	}
}

func TestEntry_MarshalWithNilOption(t *testing.T) {
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")

	// Test with nil option
	prefixes, err := entry.MarshalPrefix(nil)
	if err != nil {
		t.Errorf("Entry.MarshalPrefix(nil) error = %v, want nil", err)
	}
	if len(prefixes) != 1 {
		t.Errorf("Entry.MarshalPrefix(nil) returned %d prefixes, want 1", len(prefixes))
	}

	ipranges, err := entry.MarshalIPRange(nil)
	if err != nil {
		t.Errorf("Entry.MarshalIPRange(nil) error = %v, want nil", err)
	}
	if len(ipranges) != 1 {
		t.Errorf("Entry.MarshalIPRange(nil) returned %d ranges, want 1", len(ipranges))
	}

	cidrs, err := entry.MarshalText(nil)
	if err != nil {
		t.Errorf("Entry.MarshalText(nil) error = %v, want nil", err)
	}
	if len(cidrs) != 1 {
		t.Errorf("Entry.MarshalText(nil) returned %d CIDRs, want 1", len(cidrs))
	}
}

func TestContainer_LookupIPv6Prefix(t *testing.T) {
	container := NewContainer()

	// Add IPv6 entry
	entry := NewEntry("entry1")
	entry.AddPrefix("2001:db8::/32")
	container.Add(entry)

	// Lookup IPv6 CIDR
	results, found, err := container.Lookup("2001:db8:1::/48")
	if err != nil {
		t.Errorf("Container.Lookup() IPv6 CIDR error = %v, want nil", err)
	}
	if !found {
		t.Error("Container.Lookup() IPv6 CIDR found = false, want true")
	}
	if len(results) != 1 {
		t.Errorf("Container.Lookup() IPv6 CIDR returned %d results, want 1", len(results))
	}
}

func TestEntry_RemovePrefixComment(t *testing.T) {
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")

	// Remove with comment (should still work)
	err := entry.RemovePrefix("192.168.1.0/24 // comment")
	if err != nil {
		t.Errorf("Entry.RemovePrefix() with comment error = %v, want nil", err)
	}
}

func TestEntry_GetIPv4SetBuildError(t *testing.T) {
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")

	// Get the set
	set, err := entry.GetIPv4Set()
	if err != nil {
		t.Errorf("Entry.GetIPv4Set() error = %v, want nil", err)
	}
	if set == nil {
		t.Error("Entry.GetIPv4Set() returned nil set")
	}

	// Get it again (should use cached version)
	set2, err := entry.GetIPv4Set()
	if err != nil {
		t.Errorf("Entry.GetIPv4Set() second call error = %v, want nil", err)
	}
	if set2 == nil {
		t.Error("Entry.GetIPv4Set() second call returned nil set")
	}
}

func TestEntry_GetIPv6SetBuildError(t *testing.T) {
	entry := NewEntry("test")
	entry.AddPrefix("2001:db8::/32")

	// Get the set
	set, err := entry.GetIPv6Set()
	if err != nil {
		t.Errorf("Entry.GetIPv6Set() error = %v, want nil", err)
	}
	if set == nil {
		t.Error("Entry.GetIPv6Set() returned nil set")
	}

	// Get it again (should use cached version)
	set2, err := entry.GetIPv6Set()
	if err != nil {
		t.Errorf("Entry.GetIPv6Set() second call error = %v, want nil", err)
	}
	if set2 == nil {
		t.Error("Entry.GetIPv6Set() second call returned nil set")
	}
}

func TestContainer_RemoveEntryWithIgnoreOptions_EdgeCases(t *testing.T) {
	container := NewContainer()

	// Add entry with only IPv4
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	container.Add(entry)

	// Remove with CaseRemoveEntry and IgnoreIPv4 (should remove IPv6 builder, but there is none)
	err := container.Remove(entry, CaseRemoveEntry, IgnoreIPv4)
	if err != nil {
		t.Errorf("Container.Remove() CaseRemoveEntry with IgnoreIPv4 error = %v, want nil", err)
	}
}

func TestContainer_RemoveEntryWithIgnoreOptions_IPv6Only(t *testing.T) {
	container := NewContainer()

	// Add entry with only IPv6
	entry := NewEntry("test")
	entry.AddPrefix("2001:db8::/32")
	container.Add(entry)

	// Remove with CaseRemoveEntry and IgnoreIPv6 (should remove IPv4 builder, but there is none)
	err := container.Remove(entry, CaseRemoveEntry, IgnoreIPv6)
	if err != nil {
		t.Errorf("Container.Remove() CaseRemoveEntry with IgnoreIPv6 error = %v, want nil", err)
	}
}

func TestContainer_RemovePrefixWithIgnoreOptions_EdgeCases(t *testing.T) {
	container := NewContainer()

	// Add entry with only IPv4
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	container.Add(entry)

	// Remove prefix with IgnoreIPv4 (should only remove IPv6, but there is none)
	removeEntry := NewEntry("test")
	removeEntry.AddPrefix("2001:db8::/32")
	err := container.Remove(removeEntry, CaseRemovePrefix, IgnoreIPv4)
	if err != nil {
		t.Errorf("Container.Remove() CaseRemovePrefix with IgnoreIPv4 error = %v, want nil", err)
	}
}

func TestContainer_RemovePrefixWithIgnoreOptions_IPv6Only(t *testing.T) {
	container := NewContainer()

	// Add entry with only IPv6
	entry := NewEntry("test")
	entry.AddPrefix("2001:db8::/32")
	container.Add(entry)

	// Remove prefix with IgnoreIPv6 (should only remove IPv4, but there is none)
	removeEntry := NewEntry("test")
	removeEntry.AddPrefix("192.168.1.0/24")
	err := container.Remove(removeEntry, CaseRemovePrefix, IgnoreIPv6)
	if err != nil {
		t.Errorf("Container.Remove() CaseRemovePrefix with IgnoreIPv6 error = %v, want nil", err)
	}
}

func TestContainer_AddExistingWithIgnoreOptionsAndNilBuilders(t *testing.T) {
	container := NewContainer()

	// Add entry with only IPv6
	entry1 := NewEntry("test")
	entry1.AddPrefix("2001:db8::/32")
	container.Add(entry1)

	// Add with IgnoreIPv6 (should add IPv4, but entry2 has none)
	entry2 := NewEntry("test")
	entry2.AddPrefix("192.168.1.0/24")
	err := container.Add(entry2, IgnoreIPv6)
	if err != nil {
		t.Errorf("Container.Add() with IgnoreIPv6 error = %v, want nil", err)
	}
}

func TestContainer_AddExistingWithIgnoreIPv4AndNilBuilders(t *testing.T) {
	container := NewContainer()

	// Add entry with only IPv4
	entry1 := NewEntry("test")
	entry1.AddPrefix("192.168.1.0/24")
	container.Add(entry1)

	// Add with IgnoreIPv4 (should add IPv6, but entry2 has none)
	entry2 := NewEntry("test")
	entry2.AddPrefix("2001:db8::/32")
	err := container.Add(entry2, IgnoreIPv4)
	if err != nil {
		t.Errorf("Container.Add() with IgnoreIPv4 error = %v, want nil", err)
	}
}
