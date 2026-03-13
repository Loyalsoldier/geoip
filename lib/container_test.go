package lib

import (
	"testing"
)

func TestNewContainer(t *testing.T) {
	c := NewContainer()
	if c == nil {
		t.Fatal("NewContainer returned nil")
	}
	if c.Len() != 0 {
		t.Errorf("NewContainer().Len() = %d, want 0", c.Len())
	}
}

func TestContainerAdd(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("test")
	if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}

	if err := c.Add(entry); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if c.Len() != 1 {
		t.Errorf("Container.Len() = %d, want 1", c.Len())
	}
}

func TestContainerAdd_ExistingEntry(t *testing.T) {
	c := NewContainer()

	// Add first entry
	entry1 := NewEntry("test")
	if err := entry1.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry1); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Add second entry with same name (should merge)
	entry2 := NewEntry("test")
	if err := entry2.AddPrefix("10.0.0.0/8"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry2); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Should still be 1 entry
	if c.Len() != 1 {
		t.Errorf("Container.Len() = %d, want 1", c.Len())
	}
}

func TestContainerAdd_IgnoreIPv4(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("test")
	if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}

	if err := c.Add(entry, IgnoreIPv4); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Get the entry and verify IPv4 was ignored
	e, found := c.GetEntry("test")
	if !found {
		t.Fatal("Entry not found")
	}

	_, err := e.GetIPv4Set()
	if err == nil {
		t.Error("IPv4 should be ignored")
	}
}

func TestContainerAdd_IgnoreIPv6(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("test")
	if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}

	if err := c.Add(entry, IgnoreIPv6); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Get the entry and verify IPv6 was ignored
	e, found := c.GetEntry("test")
	if !found {
		t.Fatal("Entry not found")
	}

	_, err := e.GetIPv6Set()
	if err == nil {
		t.Error("IPv6 should be ignored")
	}
}

func TestContainerAdd_ExistingEntryWithIPv6(t *testing.T) {
	c := NewContainer()

	// Add first entry with only IPv4
	entry1 := NewEntry("test")
	if err := entry1.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry1); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Add second entry with IPv6 (same name)
	entry2 := NewEntry("test")
	if err := entry2.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry2); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	e, found := c.GetEntry("test")
	if !found {
		t.Fatal("Entry not found")
	}

	// Now should have both IPv4 and IPv6
	_, err4 := e.GetIPv4Set()
	if err4 != nil {
		t.Errorf("GetIPv4Set failed: %v", err4)
	}
	_, err6 := e.GetIPv6Set()
	if err6 != nil {
		t.Errorf("GetIPv6Set failed: %v", err6)
	}
}

func TestContainerAdd_ExistingEntryIgnoreIPv4(t *testing.T) {
	c := NewContainer()

	// Add first entry
	entry1 := NewEntry("test")
	if err := entry1.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry1); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Add second entry with IgnoreIPv4 - only IPv6 should be added
	entry2 := NewEntry("test")
	if err := entry2.AddPrefix("10.0.0.0/8"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := entry2.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry2, IgnoreIPv4); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	e, found := c.GetEntry("test")
	if !found {
		t.Fatal("Entry not found")
	}

	// Should have IPv6 now
	_, err6 := e.GetIPv6Set()
	if err6 != nil {
		t.Errorf("GetIPv6Set failed: %v", err6)
	}
}

func TestContainerAdd_ExistingEntryIgnoreIPv6(t *testing.T) {
	c := NewContainer()

	// Add first entry with IPv6
	entry1 := NewEntry("test")
	if err := entry1.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry1); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Add second entry with IgnoreIPv6 - only IPv4 should be added
	entry2 := NewEntry("test")
	if err := entry2.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := entry2.AddPrefix("2002::/16"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry2, IgnoreIPv6); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	e, found := c.GetEntry("test")
	if !found {
		t.Fatal("Entry not found")
	}

	// Should have IPv4 now
	_, err4 := e.GetIPv4Set()
	if err4 != nil {
		t.Errorf("GetIPv4Set failed: %v", err4)
	}
}

func TestContainerGetEntry(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("test")
	if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Test case insensitivity
	testCases := []string{"test", "TEST", "Test", " test ", "  TEST  "}
	for _, tc := range testCases {
		e, found := c.GetEntry(tc)
		if !found {
			t.Errorf("GetEntry(%q) not found", tc)
		}
		if e == nil {
			t.Errorf("GetEntry(%q) returned nil entry", tc)
		}
	}
}

func TestContainerGetEntry_NotFound(t *testing.T) {
	c := NewContainer()

	e, found := c.GetEntry("nonexistent")
	if found {
		t.Error("GetEntry for nonexistent entry should return false")
	}
	if e != nil {
		t.Error("GetEntry for nonexistent entry should return nil")
	}
}

func TestContainerLen(t *testing.T) {
	c := NewContainer()

	if c.Len() != 0 {
		t.Errorf("Empty container Len() = %d, want 0", c.Len())
	}

	// Add entries
	for i := 0; i < 5; i++ {
		entry := NewEntry("entry" + string(rune('0'+i)))
		if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(entry); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}

	if c.Len() != 5 {
		t.Errorf("Container.Len() = %d, want 5", c.Len())
	}
}

func TestContainerLoop(t *testing.T) {
	c := NewContainer()

	// Add entries
	names := []string{"entry1", "entry2", "entry3"}
	for _, name := range names {
		entry := NewEntry(name)
		if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(entry); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}

	// Loop through entries
	count := 0
	for range c.Loop() {
		count++
	}

	if count != 3 {
		t.Errorf("Loop iterated %d times, want 3", count)
	}
}

func TestContainerRemove_CaseRemovePrefix(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("test")
	if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := entry.AddPrefix("10.0.0.0/8"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Remove one prefix
	removeEntry := NewEntry("test")
	if err := removeEntry.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}

	if err := c.Remove(removeEntry, CaseRemovePrefix); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Entry should still exist
	e, found := c.GetEntry("test")
	if !found {
		t.Error("Entry should still exist after removing prefix")
	}
	if e == nil {
		t.Error("Entry should not be nil")
	}
}

func TestContainerRemove_CaseRemoveEntry(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("test")
	if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Remove entire entry
	removeEntry := NewEntry("test")
	if err := c.Remove(removeEntry, CaseRemoveEntry); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Entry should be gone
	_, found := c.GetEntry("test")
	if found {
		t.Error("Entry should be removed")
	}
}

func TestContainerRemove_NotFound(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("nonexistent")
	err := c.Remove(entry, CaseRemoveEntry)
	if err == nil {
		t.Error("Remove for nonexistent entry should return error")
	}
}

func TestContainerRemove_UnknownCase(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("test")
	if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	removeEntry := NewEntry("test")
	err := c.Remove(removeEntry, CaseRemove(999))
	if err == nil {
		t.Error("Remove with unknown case should return error")
	}
}

func TestContainerRemove_CaseRemovePrefixIgnoreIPv4(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("test")
	if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Remove only IPv6 prefix (ignore IPv4)
	removeEntry := NewEntry("test")
	if err := removeEntry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}

	if err := c.Remove(removeEntry, CaseRemovePrefix, IgnoreIPv4); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
}

func TestContainerRemove_CaseRemovePrefixIgnoreIPv6(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("test")
	if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Remove only IPv4 prefix (ignore IPv6)
	removeEntry := NewEntry("test")
	if err := removeEntry.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}

	if err := c.Remove(removeEntry, CaseRemovePrefix, IgnoreIPv6); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
}

func TestContainerRemove_CaseRemoveEntryIgnoreIPv4(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("test")
	if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Remove entry with IgnoreIPv4 - should only clear IPv6
	removeEntry := NewEntry("test")
	if err := c.Remove(removeEntry, CaseRemoveEntry, IgnoreIPv4); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Entry should still exist
	e, found := c.GetEntry("test")
	if !found {
		t.Error("Entry should still exist")
	}

	// IPv4 should still exist, IPv6 should be gone
	_, err4 := e.GetIPv4Set()
	if err4 != nil {
		t.Errorf("GetIPv4Set failed: %v", err4)
	}
}

func TestContainerRemove_CaseRemoveEntryIgnoreIPv6(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("test")
	if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Remove entry with IgnoreIPv6 - should only clear IPv4
	removeEntry := NewEntry("test")
	if err := c.Remove(removeEntry, CaseRemoveEntry, IgnoreIPv6); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Entry should still exist
	e, found := c.GetEntry("test")
	if !found {
		t.Error("Entry should still exist")
	}

	// IPv6 should still exist, IPv4 should be gone
	_, err6 := e.GetIPv6Set()
	if err6 != nil {
		t.Errorf("GetIPv6Set failed: %v", err6)
	}
}

func TestContainerLookup_IPv4Address(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("test")
	if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Lookup IP address
	result, found, err := c.Lookup("192.168.1.100")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if !found {
		t.Error("IP should be found")
	}
	if len(result) != 1 || result[0] != "TEST" {
		t.Errorf("Lookup result = %v, want [TEST]", result)
	}
}

func TestContainerLookup_IPv4CIDR(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("test")
	if err := entry.AddPrefix("192.168.0.0/16"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Lookup CIDR
	result, found, err := c.Lookup("192.168.1.0/24")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if !found {
		t.Error("CIDR should be found")
	}
	if len(result) != 1 || result[0] != "TEST" {
		t.Errorf("Lookup result = %v, want [TEST]", result)
	}
}

func TestContainerLookup_IPv6Address(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("test")
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Lookup IPv6 address
	result, found, err := c.Lookup("2001:db8::1")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if !found {
		t.Error("IP should be found")
	}
	if len(result) != 1 || result[0] != "TEST" {
		t.Errorf("Lookup result = %v, want [TEST]", result)
	}
}

func TestContainerLookup_IPv6CIDR(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("test")
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Lookup IPv6 CIDR
	result, found, err := c.Lookup("2001:db8:1::/48")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if !found {
		t.Error("CIDR should be found")
	}
	if len(result) != 1 || result[0] != "TEST" {
		t.Errorf("Lookup result = %v, want [TEST]", result)
	}
}

func TestContainerLookup_NotFound(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("test")
	if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Lookup non-existing IP
	result, found, err := c.Lookup("10.0.0.1")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if found {
		t.Error("IP should not be found")
	}
	if len(result) != 0 {
		t.Errorf("Lookup result = %v, want empty", result)
	}
}

func TestContainerLookup_InvalidIP(t *testing.T) {
	c := NewContainer()

	_, _, err := c.Lookup("invalid")
	if err == nil {
		t.Error("Lookup with invalid IP should return error")
	}
}

func TestContainerLookup_InvalidCIDR(t *testing.T) {
	c := NewContainer()

	_, _, err := c.Lookup("192.168.1.0/33")
	if err == nil {
		t.Error("Lookup with invalid CIDR should return error")
	}
}

func TestContainerLookup_WithSearchList(t *testing.T) {
	c := NewContainer()

	// Add multiple entries
	entry1 := NewEntry("entry1")
	if err := entry1.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry1); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	entry2 := NewEntry("entry2")
	if err := entry2.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry2); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Lookup with search list
	result, found, err := c.Lookup("192.168.1.100", "entry1")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if !found {
		t.Error("IP should be found")
	}
	if len(result) != 1 || result[0] != "ENTRY1" {
		t.Errorf("Lookup result = %v, want [ENTRY1]", result)
	}
}

func TestContainerLookup_WithEmptySearchList(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("test")
	if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Lookup with empty strings in search list (should be ignored)
	result, found, err := c.Lookup("192.168.1.100", "", "  ")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if !found {
		t.Error("IP should be found")
	}
	if len(result) != 1 {
		t.Errorf("Lookup result = %v, want length 1", result)
	}
}

func TestContainerIsValid_NilEntries(t *testing.T) {
	// Test with invalid container (nil entries)
	c := &container{entries: nil}

	// GetEntry should return false
	_, found := c.GetEntry("test")
	if found {
		t.Error("GetEntry on invalid container should return false")
	}

	// Len should return 0
	if c.Len() != 0 {
		t.Errorf("Len on invalid container = %d, want 0", c.Len())
	}
}

func TestContainerRemove_CaseRemovePrefix_NoBuilders(t *testing.T) {
	c := NewContainer()

	// Add entry with no builders initially
	entry := NewEntry("test")
	if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Try to remove with an entry that has different IP type
	removeEntry := NewEntry("test")
	if err := removeEntry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}

	// This should not error but should create the builder for the existing entry
	if err := c.Remove(removeEntry, CaseRemovePrefix); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
}
