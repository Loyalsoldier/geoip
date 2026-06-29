package lib

import (
	"testing"
)

// Additional tests for edge cases to improve coverage

func TestContainer_InvalidContainer(t *testing.T) {
	// Test with nil entries map
	c := &container{entries: nil}

	if c.isValid() {
		t.Error("container with nil entries should not be valid")
	}

	if c.Len() != 0 {
		t.Errorf("invalid container.Len() = %d, want 0", c.Len())
	}

	_, found := c.GetEntry("test")
	if found {
		t.Error("invalid container.GetEntry() should not find entries")
	}
}

func TestEntry_BuildIPSetErrors(t *testing.T) {
	entry := NewEntry("test")

	// Add IPv4 prefix
	entry.AddPrefix("192.168.1.0/24")

	// Build the set (this should succeed)
	err := entry.buildIPSet()
	if err != nil {
		t.Errorf("Entry.buildIPSet() error = %v, want nil", err)
	}

	// Build again (should be cached and succeed)
	err = entry.buildIPSet()
	if err != nil {
		t.Errorf("Entry.buildIPSet() second call error = %v, want nil", err)
	}

	// Test with IPv6
	entry2 := NewEntry("test2")
	entry2.AddPrefix("2001:db8::/32")

	err = entry2.buildIPSet()
	if err != nil {
		t.Errorf("Entry.buildIPSet() for IPv6 error = %v, want nil", err)
	}

	// Build again for IPv6 (should be cached)
	err = entry2.buildIPSet()
	if err != nil {
		t.Errorf("Entry.buildIPSet() for IPv6 second call error = %v, want nil", err)
	}
}

func TestEntry_RemoveIPv6(t *testing.T) {
	entry := NewEntry("test")

	// Add IPv6 prefix
	entry.AddPrefix("2001:db8::/32")

	// Remove it
	err := entry.RemovePrefix("2001:db8::/32")
	if err != nil {
		t.Errorf("Entry.RemovePrefix() IPv6 error = %v, want nil", err)
	}
}

func TestContainer_Add_ExistingEntryWithoutBuilders(t *testing.T) {
	container := NewContainer()

	// Add entry without creating builders (empty entry)
	entry1 := NewEntry("test")
	container.Add(entry1)

	// Add another entry with prefixes
	entry2 := NewEntry("test")
	entry2.AddPrefix("192.168.1.0/24")
	entry2.AddPrefix("2001:db8::/32")

	err := container.Add(entry2)
	if err != nil {
		t.Errorf("Container.Add() error = %v, want nil", err)
	}
}

func TestWantedListExtended_EmptyJSON(t *testing.T) {
	var w WantedListExtended
	err := w.UnmarshalJSON([]byte(""))
	if err != nil {
		t.Errorf("WantedListExtended.UnmarshalJSON() with empty data error = %v, want nil", err)
	}
}

func TestContainer_RemoveWithErrorInBuilder(t *testing.T) {
	container := NewContainer()

	// Add entry with both IPv4 and IPv6
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	entry.AddPrefix("2001:db8::/32")
	container.Add(entry)

	// Try removing with a malformed entry (empty, will cause issues building sets)
	removeEntry := NewEntry("test")
	// Don't add any prefixes, this should work but remove nothing
	err := container.Remove(removeEntry, CaseRemovePrefix)
	if err != nil {
		t.Errorf("Container.Remove() with empty entry error = %v, want nil", err)
	}
}

func TestEntry_ProcessPrefix_EdgeCases(t *testing.T) {
	entry := NewEntry("test")

	// Test string with only comment marker and content
	err := entry.AddPrefix("//")
	if err != ErrInvalidIPType {
		t.Errorf("Entry.AddPrefix('//') error = %v, want %v", err, ErrInvalidIPType)
	}

	err = entry.AddPrefix("#")
	if err != ErrInvalidIPType {
		t.Errorf("Entry.AddPrefix('#') error = %v, want %v", err, ErrInvalidIPType)
	}

	err = entry.AddPrefix("/*")
	if err != ErrInvalidIPType {
		t.Errorf("Entry.AddPrefix('/*') error = %v, want %v", err, ErrInvalidIPType)
	}
}

func TestContainer_Lookup_EmptyResults(t *testing.T) {
	container := NewContainer()

	// Add entry
	entry := NewEntry("entry1")
	entry.AddPrefix("192.168.1.0/24")
	container.Add(entry)

	// Lookup with search list that doesn't match
	results, found, err := container.Lookup("192.168.1.1", "nonexistent")
	if err != nil {
		t.Errorf("Container.Lookup() error = %v, want nil", err)
	}
	if found {
		t.Error("Container.Lookup() found = true, want false")
	}
	if len(results) != 0 {
		t.Errorf("Container.Lookup() returned %d results, want 0", len(results))
	}
}

func TestEntry_MarshalPrefixOnlyIPv4(t *testing.T) {
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")

	// Marshal with IgnoreIPv6 (should return IPv4 only)
	prefixes, err := entry.MarshalPrefix(IgnoreIPv6)
	if err != nil {
		t.Errorf("Entry.MarshalPrefix(IgnoreIPv6) error = %v, want nil", err)
	}
	if len(prefixes) != 1 {
		t.Errorf("Entry.MarshalPrefix(IgnoreIPv6) returned %d prefixes, want 1", len(prefixes))
	}

	// Test MarshalIPRange with only IPv4
	ipranges, err := entry.MarshalIPRange(IgnoreIPv6)
	if err != nil {
		t.Errorf("Entry.MarshalIPRange(IgnoreIPv6) error = %v, want nil", err)
	}
	if len(ipranges) != 1 {
		t.Errorf("Entry.MarshalIPRange(IgnoreIPv6) returned %d ranges, want 1", len(ipranges))
	}

	// Test MarshalText with only IPv4
	cidrs, err := entry.MarshalText(IgnoreIPv6)
	if err != nil {
		t.Errorf("Entry.MarshalText(IgnoreIPv6) error = %v, want nil", err)
	}
	if len(cidrs) != 1 {
		t.Errorf("Entry.MarshalText(IgnoreIPv6) returned %d CIDRs, want 1", len(cidrs))
	}
}

func TestEntry_MarshalPrefixOnlyIPv6(t *testing.T) {
	entry := NewEntry("test")
	entry.AddPrefix("2001:db8::/32")

	// Marshal with IgnoreIPv4 (should return IPv6 only)
	prefixes, err := entry.MarshalPrefix(IgnoreIPv4)
	if err != nil {
		t.Errorf("Entry.MarshalPrefix(IgnoreIPv4) error = %v, want nil", err)
	}
	if len(prefixes) != 1 {
		t.Errorf("Entry.MarshalPrefix(IgnoreIPv4) returned %d prefixes, want 1", len(prefixes))
	}

	// Test MarshalIPRange with only IPv6
	ipranges, err := entry.MarshalIPRange(IgnoreIPv4)
	if err != nil {
		t.Errorf("Entry.MarshalIPRange(IgnoreIPv4) error = %v, want nil", err)
	}
	if len(ipranges) != 1 {
		t.Errorf("Entry.MarshalIPRange(IgnoreIPv4) returned %d ranges, want 1", len(ipranges))
	}

	// Test MarshalText with only IPv6
	cidrs, err := entry.MarshalText(IgnoreIPv4)
	if err != nil {
		t.Errorf("Entry.MarshalText(IgnoreIPv4) error = %v, want nil", err)
	}
	if len(cidrs) != 1 {
		t.Errorf("Entry.MarshalText(IgnoreIPv4) returned %d CIDRs, want 1", len(cidrs))
	}
}
