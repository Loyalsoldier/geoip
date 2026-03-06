package lib

import (
	"testing"
)

func TestNewContainer(t *testing.T) {
	container := NewContainer()
	if container == nil {
		t.Fatal("NewContainer() returned nil")
	}
	if container.Len() != 0 {
		t.Errorf("NewContainer().Len() = %d, want 0", container.Len())
	}
}

func TestContainer_GetEntry(t *testing.T) {
	container := NewContainer()

	// Get non-existent entry
	_, found := container.GetEntry("notfound")
	if found {
		t.Error("Container.GetEntry() found non-existent entry")
	}

	// Add an entry
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	container.Add(entry)

	// Get existing entry (case insensitive, trimmed)
	tests := []string{"test", "TEST", "  test  ", "TeSt"}
	for _, name := range tests {
		got, found := container.GetEntry(name)
		if !found {
			t.Errorf("Container.GetEntry(%q) not found, want found", name)
			continue
		}
		if got.GetName() != "TEST" {
			t.Errorf("Container.GetEntry(%q).GetName() = %q, want %q", name, got.GetName(), "TEST")
		}
	}
}

func TestContainer_Len(t *testing.T) {
	container := NewContainer()

	if container.Len() != 0 {
		t.Errorf("Empty container.Len() = %d, want 0", container.Len())
	}

	// Add entries
	entry1 := NewEntry("entry1")
	entry1.AddPrefix("192.168.1.0/24")
	container.Add(entry1)

	if container.Len() != 1 {
		t.Errorf("Container.Len() after 1 add = %d, want 1", container.Len())
	}

	entry2 := NewEntry("entry2")
	entry2.AddPrefix("10.0.0.0/8")
	container.Add(entry2)

	if container.Len() != 2 {
		t.Errorf("Container.Len() after 2 adds = %d, want 2", container.Len())
	}
}

func TestContainer_Loop(t *testing.T) {
	container := NewContainer()

	// Add multiple entries
	entry1 := NewEntry("entry1")
	entry1.AddPrefix("192.168.1.0/24")
	container.Add(entry1)

	entry2 := NewEntry("entry2")
	entry2.AddPrefix("10.0.0.0/8")
	container.Add(entry2)

	entry3 := NewEntry("entry3")
	entry3.AddPrefix("2001:db8::/32")
	container.Add(entry3)

	// Loop through entries
	count := 0
	names := make(map[string]bool)
	for entry := range container.Loop() {
		count++
		names[entry.GetName()] = true
	}

	if count != 3 {
		t.Errorf("Container.Loop() iterated %d times, want 3", count)
	}

	expectedNames := map[string]bool{"ENTRY1": true, "ENTRY2": true, "ENTRY3": true}
	for name := range expectedNames {
		if !names[name] {
			t.Errorf("Container.Loop() missing entry %q", name)
		}
	}
}

func TestContainer_Add_NewEntry(t *testing.T) {
	container := NewContainer()

	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")

	err := container.Add(entry)
	if err != nil {
		t.Errorf("Container.Add() error = %v, want nil", err)
	}

	if container.Len() != 1 {
		t.Errorf("Container.Len() = %d, want 1", container.Len())
	}

	// Verify entry exists
	got, found := container.GetEntry("test")
	if !found {
		t.Fatal("Container.GetEntry() not found after Add")
	}
	if got.GetName() != "TEST" {
		t.Errorf("Added entry name = %q, want %q", got.GetName(), "TEST")
	}
}

func TestContainer_Add_ExistingEntry(t *testing.T) {
	container := NewContainer()

	// Add first entry
	entry1 := NewEntry("test")
	entry1.AddPrefix("192.168.1.0/24")
	container.Add(entry1)

	// Add second entry with same name
	entry2 := NewEntry("test")
	entry2.AddPrefix("10.0.0.0/8")
	err := container.Add(entry2)
	if err != nil {
		t.Errorf("Container.Add() existing entry error = %v, want nil", err)
	}

	// Should still have only 1 entry
	if container.Len() != 1 {
		t.Errorf("Container.Len() = %d, want 1", container.Len())
	}

	// Verify both prefixes are in the entry
	got, _ := container.GetEntry("test")
	cidrs, err := got.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error = %v", err)
	}
	if len(cidrs) != 2 {
		t.Errorf("Entry has %d prefixes, want 2", len(cidrs))
	}
}

func TestContainer_Add_WithIgnoreIPv4(t *testing.T) {
	container := NewContainer()

	// Add entry with both IPv4 and IPv6
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	entry.AddPrefix("2001:db8::/32")
	container.Add(entry, IgnoreIPv4)

	// Should only have IPv6
	got, _ := container.GetEntry("test")
	_, err := got.GetIPv4Set()
	if err == nil {
		t.Error("Entry should not have IPv4 set when added with IgnoreIPv4")
	}

	_, err = got.GetIPv6Set()
	if err != nil {
		t.Errorf("Entry.GetIPv6Set() error = %v, want nil", err)
	}
}

func TestContainer_Add_WithIgnoreIPv6(t *testing.T) {
	container := NewContainer()

	// Add entry with both IPv4 and IPv6
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	entry.AddPrefix("2001:db8::/32")
	container.Add(entry, IgnoreIPv6)

	// Should only have IPv4
	got, _ := container.GetEntry("test")
	_, err := got.GetIPv4Set()
	if err != nil {
		t.Errorf("Entry.GetIPv4Set() error = %v, want nil", err)
	}

	_, err = got.GetIPv6Set()
	if err == nil {
		t.Error("Entry should not have IPv6 set when added with IgnoreIPv6")
	}
}

func TestContainer_Add_ExistingWithIgnoreOptions(t *testing.T) {
	container := NewContainer()

	// Add first entry with IPv4
	entry1 := NewEntry("test")
	entry1.AddPrefix("192.168.1.0/24")
	container.Add(entry1)

	// Add second entry with IPv6, ignoring IPv4
	entry2 := NewEntry("test")
	entry2.AddPrefix("2001:db8::/32")
	container.Add(entry2, IgnoreIPv4)

	// Should have only IPv6 now
	got, _ := container.GetEntry("test")
	_, err := got.GetIPv6Set()
	if err != nil {
		t.Errorf("Entry.GetIPv6Set() error = %v, want nil", err)
	}
}

func TestContainer_Remove_NotFound(t *testing.T) {
	container := NewContainer()

	entry := NewEntry("notfound")
	err := container.Remove(entry, CaseRemoveEntry)
	if err == nil {
		t.Error("Container.Remove() on non-existent entry expected error, got nil")
	}
}

func TestContainer_Remove_CaseRemoveEntry(t *testing.T) {
	container := NewContainer()

	// Add entry
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	container.Add(entry)

	// Remove entire entry
	err := container.Remove(entry, CaseRemoveEntry)
	if err != nil {
		t.Errorf("Container.Remove() error = %v, want nil", err)
	}

	// Should not be found anymore
	_, found := container.GetEntry("test")
	if found {
		t.Error("Entry still found after CaseRemoveEntry")
	}

	if container.Len() != 0 {
		t.Errorf("Container.Len() = %d, want 0 after remove", container.Len())
	}
}

func TestContainer_Remove_CaseRemovePrefix(t *testing.T) {
	container := NewContainer()

	// Add entry with multiple prefixes
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	entry.AddPrefix("10.0.0.0/8")
	container.Add(entry)

	// Remove one prefix
	removeEntry := NewEntry("test")
	removeEntry.AddPrefix("192.168.1.0/24")
	err := container.Remove(removeEntry, CaseRemovePrefix)
	if err != nil {
		t.Errorf("Container.Remove() error = %v, want nil", err)
	}

	// Entry should still exist
	got, found := container.GetEntry("test")
	if !found {
		t.Fatal("Entry not found after CaseRemovePrefix")
	}

	// Should have only one prefix left
	cidrs, err := got.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error = %v", err)
	}
	if len(cidrs) != 1 {
		t.Errorf("Entry has %d prefixes, want 1 after removal", len(cidrs))
	}
}

func TestContainer_Remove_WithIgnoreIPv4(t *testing.T) {
	container := NewContainer()

	// Add entry with both IPv4 and IPv6
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	entry.AddPrefix("2001:db8::/32")
	container.Add(entry)

	// Remove IPv6 only (ignoring IPv4)
	removeEntry := NewEntry("test")
	removeEntry.AddPrefix("2001:db8::/32")
	err := container.Remove(removeEntry, CaseRemovePrefix, IgnoreIPv4)
	if err != nil {
		t.Errorf("Container.Remove() error = %v, want nil", err)
	}

	// IPv4 should still exist
	got, _ := container.GetEntry("test")
	_, err = got.GetIPv4Set()
	if err != nil {
		t.Errorf("IPv4 set should still exist after removing IPv6 with IgnoreIPv4")
	}
}

func TestContainer_Remove_WithIgnoreIPv6(t *testing.T) {
	container := NewContainer()

	// Add entry with both IPv4 and IPv6
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	entry.AddPrefix("2001:db8::/32")
	container.Add(entry)

	// Remove IPv4 only (ignoring IPv6)
	removeEntry := NewEntry("test")
	removeEntry.AddPrefix("192.168.1.0/24")
	err := container.Remove(removeEntry, CaseRemovePrefix, IgnoreIPv6)
	if err != nil {
		t.Errorf("Container.Remove() error = %v, want nil", err)
	}

	// IPv6 should still exist
	got, _ := container.GetEntry("test")
	_, err = got.GetIPv6Set()
	if err != nil {
		t.Errorf("IPv6 set should still exist after removing IPv4 with IgnoreIPv6")
	}
}

func TestContainer_Remove_CaseRemoveEntry_WithIgnoreIPv4(t *testing.T) {
	container := NewContainer()

	// Add entry with both IPv4 and IPv6
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	entry.AddPrefix("2001:db8::/32")
	container.Add(entry)

	// Remove IPv6 only (CaseRemoveEntry with IgnoreIPv4)
	err := container.Remove(entry, CaseRemoveEntry, IgnoreIPv4)
	if err != nil {
		t.Errorf("Container.Remove() error = %v, want nil", err)
	}

	// Entry should still exist but only with IPv4
	got, found := container.GetEntry("test")
	if !found {
		t.Fatal("Entry should still exist")
	}

	_, err = got.GetIPv4Set()
	if err != nil {
		t.Errorf("IPv4 set should still exist")
	}

	_, err = got.GetIPv6Set()
	if err == nil {
		t.Error("IPv6 set should not exist")
	}
}

func TestContainer_Remove_CaseRemoveEntry_WithIgnoreIPv6(t *testing.T) {
	container := NewContainer()

	// Add entry with both IPv4 and IPv6
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	entry.AddPrefix("2001:db8::/32")
	container.Add(entry)

	// Remove IPv4 only (CaseRemoveEntry with IgnoreIPv6)
	err := container.Remove(entry, CaseRemoveEntry, IgnoreIPv6)
	if err != nil {
		t.Errorf("Container.Remove() error = %v, want nil", err)
	}

	// Entry should still exist but only with IPv6
	got, found := container.GetEntry("test")
	if !found {
		t.Fatal("Entry should still exist")
	}

	_, err = got.GetIPv6Set()
	if err != nil {
		t.Errorf("IPv6 set should still exist")
	}

	_, err = got.GetIPv4Set()
	if err == nil {
		t.Error("IPv4 set should not exist")
	}
}

func TestContainer_Remove_InvalidCase(t *testing.T) {
	container := NewContainer()

	// Add entry
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	container.Add(entry)

	// Try to remove with invalid case
	err := container.Remove(entry, CaseRemove(999))
	if err == nil {
		t.Error("Container.Remove() with invalid case expected error, got nil")
	}
}

func TestContainer_Lookup_IPv4(t *testing.T) {
	container := NewContainer()

	// Add entries
	entry1 := NewEntry("entry1")
	entry1.AddPrefix("192.168.1.0/24")
	container.Add(entry1)

	entry2 := NewEntry("entry2")
	entry2.AddPrefix("10.0.0.0/8")
	container.Add(entry2)

	// Lookup IPv4 address
	results, found, err := container.Lookup("192.168.1.1")
	if err != nil {
		t.Errorf("Container.Lookup() error = %v, want nil", err)
	}
	if !found {
		t.Error("Container.Lookup() found = false, want true")
	}
	if len(results) != 1 {
		t.Errorf("Container.Lookup() returned %d results, want 1", len(results))
	}
	if len(results) > 0 && results[0] != "ENTRY1" {
		t.Errorf("Container.Lookup() result = %q, want %q", results[0], "ENTRY1")
	}

	// Lookup IPv4 address in second entry
	results, found, err = container.Lookup("10.1.2.3")
	if err != nil {
		t.Errorf("Container.Lookup() error = %v, want nil", err)
	}
	if !found {
		t.Error("Container.Lookup() found = false, want true")
	}
	if len(results) != 1 {
		t.Errorf("Container.Lookup() returned %d results, want 1", len(results))
	}
	if len(results) > 0 && results[0] != "ENTRY2" {
		t.Errorf("Container.Lookup() result = %q, want %q", results[0], "ENTRY2")
	}
}

func TestContainer_Lookup_IPv6(t *testing.T) {
	container := NewContainer()

	// Add entry
	entry := NewEntry("entry1")
	entry.AddPrefix("2001:db8::/32")
	container.Add(entry)

	// Lookup IPv6 address
	results, found, err := container.Lookup("2001:db8::1")
	if err != nil {
		t.Errorf("Container.Lookup() error = %v, want nil", err)
	}
	if !found {
		t.Error("Container.Lookup() found = false, want true")
	}
	if len(results) != 1 {
		t.Errorf("Container.Lookup() returned %d results, want 1", len(results))
	}
	if len(results) > 0 && results[0] != "ENTRY1" {
		t.Errorf("Container.Lookup() result = %q, want %q", results[0], "ENTRY1")
	}
}

func TestContainer_Lookup_CIDR(t *testing.T) {
	container := NewContainer()

	// Add entry
	entry := NewEntry("entry1")
	entry.AddPrefix("192.168.0.0/16")
	container.Add(entry)

	// Lookup CIDR
	results, found, err := container.Lookup("192.168.1.0/24")
	if err != nil {
		t.Errorf("Container.Lookup() error = %v, want nil", err)
	}
	if !found {
		t.Error("Container.Lookup() found = false, want true")
	}
	if len(results) != 1 {
		t.Errorf("Container.Lookup() returned %d results, want 1", len(results))
	}
}

func TestContainer_Lookup_NotFound(t *testing.T) {
	container := NewContainer()

	// Add entry
	entry := NewEntry("entry1")
	entry.AddPrefix("192.168.1.0/24")
	container.Add(entry)

	// Lookup non-matching address
	results, found, err := container.Lookup("10.0.0.1")
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

func TestContainer_Lookup_WithSearchList(t *testing.T) {
	container := NewContainer()

	// Add multiple entries
	entry1 := NewEntry("entry1")
	entry1.AddPrefix("192.168.1.0/24")
	container.Add(entry1)

	entry2 := NewEntry("entry2")
	entry2.AddPrefix("192.168.1.0/24")
	container.Add(entry2)

	entry3 := NewEntry("entry3")
	entry3.AddPrefix("192.168.1.0/24")
	container.Add(entry3)

	// Lookup with search list
	results, found, err := container.Lookup("192.168.1.1", "entry1", "entry3")
	if err != nil {
		t.Errorf("Container.Lookup() error = %v, want nil", err)
	}
	if !found {
		t.Error("Container.Lookup() found = false, want true")
	}
	if len(results) != 2 {
		t.Errorf("Container.Lookup() returned %d results, want 2", len(results))
	}

	// Verify results contain only searched entries
	resultMap := make(map[string]bool)
	for _, r := range results {
		resultMap[r] = true
	}
	if !resultMap["ENTRY1"] || !resultMap["ENTRY3"] {
		t.Errorf("Container.Lookup() results = %v, want ENTRY1 and ENTRY3", results)
	}
	if resultMap["ENTRY2"] {
		t.Error("Container.Lookup() should not include ENTRY2")
	}
}

func TestContainer_Lookup_InvalidIP(t *testing.T) {
	container := NewContainer()

	// Lookup invalid IP
	_, _, err := container.Lookup("invalid")
	if err == nil {
		t.Error("Container.Lookup() with invalid IP expected error, got nil")
	}
}

func TestContainer_Lookup_InvalidCIDR(t *testing.T) {
	container := NewContainer()

	// Lookup invalid CIDR
	_, _, err := container.Lookup("192.168.1.0/33")
	if err == nil {
		t.Error("Container.Lookup() with invalid CIDR expected error, got nil")
	}
}

func TestContainer_Lookup_SearchListCaseInsensitive(t *testing.T) {
	container := NewContainer()

	// Add entry
	entry := NewEntry("MyEntry")
	entry.AddPrefix("192.168.1.0/24")
	container.Add(entry)

	// Lookup with different case
	results, found, err := container.Lookup("192.168.1.1", "myentry", "MYENTRY", "  MyEntry  ")
	if err != nil {
		t.Errorf("Container.Lookup() error = %v, want nil", err)
	}
	if !found {
		t.Error("Container.Lookup() found = false, want true")
	}
	if len(results) != 1 {
		t.Errorf("Container.Lookup() returned %d results, want 1", len(results))
	}
}

func TestContainer_Lookup_EmptySearchListEntries(t *testing.T) {
	container := NewContainer()

	// Add entry
	entry := NewEntry("entry1")
	entry.AddPrefix("192.168.1.0/24")
	container.Add(entry)

	// Lookup with empty/whitespace search list entries (should be ignored)
	results, found, err := container.Lookup("192.168.1.1", "", "  ", "entry1")
	if err != nil {
		t.Errorf("Container.Lookup() error = %v, want nil", err)
	}
	if !found {
		t.Error("Container.Lookup() found = false, want true")
	}
	if len(results) != 1 {
		t.Errorf("Container.Lookup() returned %d results, want 1", len(results))
	}
}
