package lib

import (
	"testing"
)

func TestNewContainer(t *testing.T) {
	container := NewContainer()
	if container == nil {
		t.Error("NewContainer() should return non-nil container")
	}
	
	if container.Len() != 0 {
		t.Errorf("New container should have length 0, got %d", container.Len())
	}
}

func TestContainerAddAndGet(t *testing.T) {
	container := NewContainer()
	entry := NewEntry("test")
	
	// Add prefix to entry
	err := entry.AddPrefix("192.168.1.0/24")
	if err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	
	// Add entry to container
	err = container.Add(entry)
	if err != nil {
		t.Errorf("Container.Add() should not return error: %v", err)
	}
	
	// Check container length
	if container.Len() != 1 {
		t.Errorf("Container length should be 1 after adding entry, got %d", container.Len())
	}
	
	// Get entry from container
	retrievedEntry, found := container.GetEntry("test")
	if !found {
		t.Error("GetEntry() should find the added entry")
	}
	if retrievedEntry.GetName() != "TEST" {
		t.Errorf("Retrieved entry name should be 'TEST', got '%s'", retrievedEntry.GetName())
	}
	
	// Test case insensitive retrieval
	retrievedEntry, found = container.GetEntry("TEST")
	if !found {
		t.Error("GetEntry() should be case insensitive")
	}
	
	retrievedEntry, found = container.GetEntry("TeSt")
	if !found {
		t.Error("GetEntry() should be case insensitive")
	}
	
	// Test retrieval with extra spaces
	retrievedEntry, found = container.GetEntry("  test  ")
	if !found {
		t.Error("GetEntry() should handle names with spaces")
	}
}

func TestContainerGetEntryNotFound(t *testing.T) {
	container := NewContainer()
	
	// Try to get non-existent entry
	_, found := container.GetEntry("nonexistent")
	if found {
		t.Error("GetEntry() should return false for non-existent entry")
	}
	
	// Try with empty name
	_, found = container.GetEntry("")
	if found {
		t.Error("GetEntry() should return false for empty name")
	}
}

func TestContainerAddMultipleEntries(t *testing.T) {
	container := NewContainer()
	
	// Create and add multiple entries
	entries := []string{"entry1", "entry2", "entry3"}
	for _, name := range entries {
		entry := NewEntry(name)
		err := entry.AddPrefix("192.168.1.0/24")
		if err != nil {
			t.Fatalf("AddPrefix failed for %s: %v", name, err)
		}
		
		err = container.Add(entry)
		if err != nil {
			t.Errorf("Container.Add() failed for %s: %v", name, err)
		}
	}
	
	// Check container length
	if container.Len() != len(entries) {
		t.Errorf("Container length should be %d, got %d", len(entries), container.Len())
	}
	
	// Verify all entries can be retrieved
	for _, name := range entries {
		_, found := container.GetEntry(name)
		if !found {
			t.Errorf("Entry %s should be found in container", name)
		}
	}
}

func TestContainerAddDuplicateEntry(t *testing.T) {
	container := NewContainer()
	entry1 := NewEntry("test")
	entry2 := NewEntry("test")
	
	// Add prefixes to entries
	err := entry1.AddPrefix("192.168.1.0/24")
	if err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	err = entry2.AddPrefix("10.0.0.0/8")
	if err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	
	// Add first entry
	err = container.Add(entry1)
	if err != nil {
		t.Errorf("First Add() should not return error: %v", err)
	}
	
	// Add second entry with same name (should merge)
	err = container.Add(entry2)
	if err != nil {
		t.Errorf("Second Add() should not return error: %v", err)
	}
	
	// Container should still have length 1
	if container.Len() != 1 {
		t.Errorf("Container should still have length 1 after adding duplicate, got %d", container.Len())
	}
}

func TestContainerAddWithIgnoreOptions(t *testing.T) {
	container := NewContainer()
	entry := NewEntry("test")
	
	// Add both IPv4 and IPv6 prefixes
	err := entry.AddPrefix("192.168.1.0/24")
	if err != nil {
		t.Fatalf("AddPrefix IPv4 failed: %v", err)
	}
	err = entry.AddPrefix("2001:db8::/32")
	if err != nil {
		t.Fatalf("AddPrefix IPv6 failed: %v", err)
	}
	
	// Add with IgnoreIPv4 option
	err = container.Add(entry, IgnoreIPv4)
	if err != nil {
		t.Errorf("Add() with IgnoreIPv4 should not return error: %v", err)
	}
	
	// Create another entry with IgnoreIPv6 option
	entry2 := NewEntry("test2")
	err = entry2.AddPrefix("10.0.0.0/8")
	if err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	
	err = container.Add(entry2, IgnoreIPv6)
	if err != nil {
		t.Errorf("Add() with IgnoreIPv6 should not return error: %v", err)
	}
	
	if container.Len() != 2 {
		t.Errorf("Container should have length 2, got %d", container.Len())
	}
}

func TestContainerRemove(t *testing.T) {
	container := NewContainer()
	entry := NewEntry("test")
	
	// Add prefix to entry
	err := entry.AddPrefix("192.168.1.0/24")
	if err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	
	// Add entry to container
	err = container.Add(entry)
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}
	
	// Remove entry with CaseRemovePrefix
	err = container.Remove(entry, CaseRemovePrefix)
	if err != nil {
		t.Errorf("Remove() with CaseRemovePrefix should not return error: %v", err)
	}
	
	// Remove entry with CaseRemoveEntry
	err = container.Remove(entry, CaseRemoveEntry)
	if err != nil {
		t.Errorf("Remove() with CaseRemoveEntry should not return error: %v", err)
	}
}

func TestContainerRemoveWithIgnoreOptions(t *testing.T) {
	container := NewContainer()
	entry := NewEntry("test")
	
	// Add both IPv4 and IPv6 prefixes
	err := entry.AddPrefix("192.168.1.0/24")
	if err != nil {
		t.Fatalf("AddPrefix IPv4 failed: %v", err)
	}
	err = entry.AddPrefix("2001:db8::/32")
	if err != nil {
		t.Fatalf("AddPrefix IPv6 failed: %v", err)
	}
	
	// Add entry to container
	err = container.Add(entry)
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}
	
	// Remove with ignore options
	err = container.Remove(entry, CaseRemovePrefix, IgnoreIPv4)
	if err != nil {
		t.Errorf("Remove() with IgnoreIPv4 should not return error: %v", err)
	}
	
	err = container.Remove(entry, CaseRemovePrefix, IgnoreIPv6)
	if err != nil {
		t.Errorf("Remove() with IgnoreIPv6 should not return error: %v", err)
	}
}

func TestContainerLoop(t *testing.T) {
	container := NewContainer()
	entryNames := []string{"entry1", "entry2", "entry3"}
	
	// Add multiple entries
	for _, name := range entryNames {
		entry := NewEntry(name)
		err := entry.AddPrefix("192.168.1.0/24")
		if err != nil {
			t.Fatalf("AddPrefix failed for %s: %v", name, err)
		}
		
		err = container.Add(entry)
		if err != nil {
			t.Fatalf("Add() failed for %s: %v", name, err)
		}
	}
	
	// Loop through entries
	count := 0
	foundNames := make(map[string]bool)
	
	for entry := range container.Loop() {
		count++
		foundNames[entry.GetName()] = true
		
		if entry == nil {
			t.Error("Loop() should not return nil entry")
		}
	}
	
	if count != len(entryNames) {
		t.Errorf("Loop() should iterate %d times, got %d", len(entryNames), count)
	}
	
	// Check that all expected names were found
	for _, name := range entryNames {
		expectedName := name // Will be converted to uppercase by NewEntry
		if name == "entry1" {
			expectedName = "ENTRY1"
		} else if name == "entry2" {
			expectedName = "ENTRY2"
		} else if name == "entry3" {
			expectedName = "ENTRY3"
		}
		
		if !foundNames[expectedName] {
			t.Errorf("Entry %s should be found in loop", expectedName)
		}
	}
}

func TestContainerLoopEmpty(t *testing.T) {
	container := NewContainer()
	
	// Loop through empty container
	count := 0
	for range container.Loop() {
		count++
	}
	
	if count != 0 {
		t.Errorf("Loop() on empty container should iterate 0 times, got %d", count)
	}
}

func TestContainerLookup(t *testing.T) {
	container := NewContainer()
	entry := NewEntry("test")
	
	// Add prefix to entry
	err := entry.AddPrefix("192.168.1.0/24")
	if err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	
	// Add entry to container
	err = container.Add(entry)
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}
	
	tests := []struct {
		name         string
		ipOrCidr     string
		searchList   []string
		expectFound  bool
		expectResult []string
		expectError  bool
	}{
		{
			name:         "IP in range",
			ipOrCidr:     "192.168.1.100",
			searchList:   []string{"test"},
			expectFound:  true,
			expectResult: []string{"TEST"},
			expectError:  false,
		},
		{
			name:         "IP not in range",
			ipOrCidr:     "10.0.0.1",
			searchList:   []string{"test"},
			expectFound:  false,
			expectResult: []string{},
			expectError:  false,
		},
		{
			name:         "Invalid IP",
			ipOrCidr:     "invalid-ip",
			searchList:   []string{"test"},
			expectFound:  false,
			expectResult: nil,
			expectError:  true,
		},
		{
			name:         "Empty search list",
			ipOrCidr:     "192.168.1.100",
			searchList:   []string{},
			expectFound:  true,
			expectResult: []string{"TEST"},
			expectError:  false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found, err := container.Lookup(tt.ipOrCidr, tt.searchList...)
			
			if tt.expectError && err == nil {
				t.Errorf("Lookup() should return error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Lookup() should not return error but got: %v", err)
			}
			if !tt.expectError && found != tt.expectFound {
				t.Errorf("Lookup() found = %v; want %v", found, tt.expectFound)
			}
			if !tt.expectError && tt.expectResult != nil && len(result) != len(tt.expectResult) {
				t.Errorf("Lookup() result length = %d; want %d", len(result), len(tt.expectResult))
			}
		})
	}
}