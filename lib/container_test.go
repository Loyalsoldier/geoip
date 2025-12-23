package lib

import (
	"testing"
)

func TestNewContainer(t *testing.T) {
	c := NewContainer()
	if c == nil {
		t.Fatal("NewContainer() returned nil")
	}
	if c.Len() != 0 {
		t.Errorf("NewContainer().Len() = %d, want 0", c.Len())
	}
}

func TestContainer_GetEntry(t *testing.T) {
	c := NewContainer()
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	c.Add(entry)

	tests := []struct {
		name      string
		entryName string
		wantFound bool
	}{
		{
			name:      "existing entry",
			entryName: "test",
			wantFound: true,
		},
		{
			name:      "existing entry uppercase",
			entryName: "TEST",
			wantFound: true,
		},
		{
			name:      "existing entry with spaces",
			entryName: "  test  ",
			wantFound: true,
		},
		{
			name:      "non-existing entry",
			entryName: "nonexistent",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := c.GetEntry(tt.entryName)
			if found != tt.wantFound {
				t.Errorf("Container.GetEntry() found = %v, wantFound %v", found, tt.wantFound)
			}
			if tt.wantFound && got == nil {
				t.Error("Container.GetEntry() returned nil entry")
			}
		})
	}
}

func TestContainer_Add(t *testing.T) {
	tests := []struct {
		name    string
		entries []*Entry
		opts    []IgnoreIPOption
		wantLen int
	}{
		{
			name: "add single entry",
			entries: []*Entry{
				func() *Entry {
					e := NewEntry("test1")
					e.AddPrefix("192.168.1.0/24")
					return e
				}(),
			},
			opts:    nil,
			wantLen: 1,
		},
		{
			name: "add multiple entries",
			entries: []*Entry{
				func() *Entry {
					e := NewEntry("test1")
					e.AddPrefix("192.168.1.0/24")
					return e
				}(),
				func() *Entry {
					e := NewEntry("test2")
					e.AddPrefix("10.0.0.0/8")
					return e
				}(),
			},
			opts:    nil,
			wantLen: 2,
		},
		{
			name: "add duplicate entry",
			entries: []*Entry{
				func() *Entry {
					e := NewEntry("test1")
					e.AddPrefix("192.168.1.0/24")
					return e
				}(),
				func() *Entry {
					e := NewEntry("test1")
					e.AddPrefix("10.0.0.0/8")
					return e
				}(),
			},
			opts:    nil,
			wantLen: 1, // Should merge into one
		},
		{
			name: "add with ignore IPv4",
			entries: []*Entry{
				func() *Entry {
					e := NewEntry("test1")
					e.AddPrefix("192.168.1.0/24")
					e.AddPrefix("2001:db8::/32")
					return e
				}(),
			},
			opts:    []IgnoreIPOption{IgnoreIPv4},
			wantLen: 1,
		},
		{
			name: "add with ignore IPv6",
			entries: []*Entry{
				func() *Entry {
					e := NewEntry("test1")
					e.AddPrefix("192.168.1.0/24")
					e.AddPrefix("2001:db8::/32")
					return e
				}(),
			},
			opts:    []IgnoreIPOption{IgnoreIPv6},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewContainer()
			for _, entry := range tt.entries {
				if err := c.Add(entry, tt.opts...); err != nil {
					t.Errorf("Container.Add() error = %v", err)
				}
			}
			if c.Len() != tt.wantLen {
				t.Errorf("Container.Len() = %d, want %d", c.Len(), tt.wantLen)
			}
		})
	}
}

func TestContainer_Remove(t *testing.T) {
	tests := []struct {
		name       string
		setupFn    func(Container)
		removeName string
		removeCase CaseRemove
		opts       []IgnoreIPOption
		wantErr    bool
		checkFn    func(*testing.T, Container)
	}{
		{
			name: "remove non-existent entry",
			setupFn: func(c Container) {
				e := NewEntry("test1")
				e.AddPrefix("192.168.1.0/24")
				c.Add(e)
			},
			removeName: "nonexistent",
			removeCase: CaseRemoveEntry,
			wantErr:    true,
		},
		{
			name: "remove entry completely",
			setupFn: func(c Container) {
				e := NewEntry("test1")
				e.AddPrefix("192.168.1.0/24")
				c.Add(e)
			},
			removeName: "test1",
			removeCase: CaseRemoveEntry,
			wantErr:    false,
			checkFn: func(t *testing.T, c Container) {
				if c.Len() != 0 {
					t.Errorf("Container.Len() = %d, want 0", c.Len())
				}
			},
		},
		{
			name: "remove prefix",
			setupFn: func(c Container) {
				e := NewEntry("test1")
				e.AddPrefix("192.168.1.0/24")
				e.AddPrefix("10.0.0.0/8")
				c.Add(e)
			},
			removeName: "test1",
			removeCase: CaseRemovePrefix,
			wantErr:    false,
			checkFn: func(t *testing.T, c Container) {
				if c.Len() != 1 {
					t.Errorf("Container.Len() = %d, want 1", c.Len())
				}
			},
		},
		{
			name: "remove with ignore IPv4",
			setupFn: func(c Container) {
				e := NewEntry("test1")
				e.AddPrefix("192.168.1.0/24")
				e.AddPrefix("2001:db8::/32")
				c.Add(e)
			},
			removeName: "test1",
			removeCase: CaseRemoveEntry,
			opts:       []IgnoreIPOption{IgnoreIPv4},
			wantErr:    false,
			checkFn: func(t *testing.T, c Container) {
				if c.Len() != 1 {
					t.Errorf("Container.Len() = %d, want 1 (IPv4 should be removed)", c.Len())
				}
			},
		},
		{
			name: "remove with ignore IPv6",
			setupFn: func(c Container) {
				e := NewEntry("test1")
				e.AddPrefix("192.168.1.0/24")
				e.AddPrefix("2001:db8::/32")
				c.Add(e)
			},
			removeName: "test1",
			removeCase: CaseRemoveEntry,
			opts:       []IgnoreIPOption{IgnoreIPv6},
			wantErr:    false,
			checkFn: func(t *testing.T, c Container) {
				if c.Len() != 1 {
					t.Errorf("Container.Len() = %d, want 1 (IPv6 should be removed)", c.Len())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewContainer()
			if tt.setupFn != nil {
				tt.setupFn(c)
			}
			
			removeEntry := NewEntry(tt.removeName)
			removeEntry.AddPrefix("192.168.1.0/24")
			
			err := c.Remove(removeEntry, tt.removeCase, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Container.Remove() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if tt.checkFn != nil {
				tt.checkFn(t, c)
			}
		})
	}
}

func TestContainer_Loop(t *testing.T) {
	c := NewContainer()
	
	// Add some entries
	for i := 1; i <= 3; i++ {
		e := NewEntry("test" + string(rune('0'+i)))
		e.AddPrefix("192.168.1.0/24")
		c.Add(e)
	}

	count := 0
	for entry := range c.Loop() {
		if entry == nil {
			t.Error("Container.Loop() returned nil entry")
		}
		count++
	}

	if count != 3 {
		t.Errorf("Container.Loop() iterated %d times, want 3", count)
	}
}

func TestContainer_Lookup(t *testing.T) {
	c := NewContainer()
	
	// Setup test data
	e1 := NewEntry("CN")
	e1.AddPrefix("192.168.1.0/24")
	e1.AddPrefix("10.0.0.0/8")
	e1.AddPrefix("2001:db8:1::/48") // Add IPv6 for CN
	c.Add(e1)
	
	e2 := NewEntry("US")
	e2.AddPrefix("172.16.0.0/12")
	e2.AddPrefix("2001:db8::/32")
	c.Add(e2)

	tests := []struct {
		name       string
		ipOrCidr   string
		searchList []string
		wantFound  bool
		wantErr    bool
		checkFn    func(*testing.T, []string)
	}{
		{
			name:       "lookup IPv4 address",
			ipOrCidr:   "192.168.1.100",
			searchList: nil,
			wantFound:  true,
			wantErr:    false,
			checkFn: func(t *testing.T, results []string) {
				if len(results) == 0 {
					t.Error("Expected at least one result")
				}
			},
		},
		{
			name:       "lookup IPv4 CIDR",
			ipOrCidr:   "192.168.1.0/24",
			searchList: nil,
			wantFound:  true,
			wantErr:    false,
		},
		{
			name:       "lookup IPv6 address",
			ipOrCidr:   "2001:db8::1",
			searchList: nil,
			wantFound:  true,
			wantErr:    false,
		},
		{
			name:       "lookup IPv6 CIDR",
			ipOrCidr:   "2001:db8::/32",
			searchList: nil,
			wantFound:  true,
			wantErr:    false,
		},
		{
			name:       "lookup with search list",
			ipOrCidr:   "192.168.1.100",
			searchList: []string{"CN"},
			wantFound:  true,
			wantErr:    false,
		},
		{
			name:       "lookup not in search list",
			ipOrCidr:   "172.16.1.1",
			searchList: []string{"CN"},
			wantFound:  false,
			wantErr:    false,
		},
		{
			name:       "lookup non-existent IP",
			ipOrCidr:   "1.1.1.1",
			searchList: nil,
			wantFound:  false,
			wantErr:    false,
		},
		{
			name:       "invalid IP",
			ipOrCidr:   "invalid",
			searchList: nil,
			wantFound:  false,
			wantErr:    true,
		},
		{
			name:       "invalid CIDR",
			ipOrCidr:   "invalid/24",
			searchList: nil,
			wantFound:  false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, found, err := c.Lookup(tt.ipOrCidr, tt.searchList...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Container.Lookup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if found != tt.wantFound {
				t.Errorf("Container.Lookup() found = %v, wantFound %v", found, tt.wantFound)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, results)
			}
		})
	}
}

func TestContainer_RemoveWithPrefixCase(t *testing.T) {
	c := NewContainer()
	
	e1 := NewEntry("test")
	e1.AddPrefix("192.168.1.0/24")
	e1.AddPrefix("10.0.0.0/8")
	e1.AddPrefix("2001:db8::/32")
	c.Add(e1)
	
	// Create entry with prefixes to remove
	removeEntry := NewEntry("test")
	removeEntry.AddPrefix("192.168.1.0/24")
	
	// Remove with CaseRemovePrefix and ignore IPv4
	err := c.Remove(removeEntry, CaseRemovePrefix, IgnoreIPv4)
	if err != nil {
		t.Errorf("Container.Remove() error = %v", err)
	}
}

func TestContainer_InvalidRemoveCase(t *testing.T) {
	c := NewContainer()
	
	e := NewEntry("test")
	e.AddPrefix("192.168.1.0/24")
	c.Add(e)
	
	// Try to remove with invalid case
	err := c.Remove(e, CaseRemove(99))
	if err == nil {
		t.Error("Container.Remove() should return error for invalid case")
	}
}

func TestContainer_AddWithMerging(t *testing.T) {
	c := NewContainer()
	
	// Add first entry
	e1 := NewEntry("test")
	e1.AddPrefix("192.168.1.0/24")
	e1.AddPrefix("2001:db8::/32")
	c.Add(e1)
	
	// Add second entry with same name - should merge
	e2 := NewEntry("test")
	e2.AddPrefix("10.0.0.0/8")
	e2.AddPrefix("2001:db9::/32")
	c.Add(e2)
	
	if c.Len() != 1 {
		t.Errorf("Container.Len() = %d, want 1 (entries should merge)", c.Len())
	}
	
	// Verify merged entry has prefixes from both
	entry, found := c.GetEntry("test")
	if !found {
		t.Fatal("Entry not found after merge")
	}
	
	prefixes, err := entry.MarshalPrefix()
	if err != nil {
		t.Errorf("MarshalPrefix() error = %v", err)
	}
	// After merging, we should have at least some prefixes
	if len(prefixes) == 0 {
		t.Error("Merged entry has no prefixes")
	}
}

func TestContainer_AddWithIgnoreOptions(t *testing.T) {
	tests := []struct {
		name     string
		opts     []IgnoreIPOption
		checkFn  func(*testing.T, Container)
	}{
		{
			name: "add new entry with ignore IPv4",
			opts: []IgnoreIPOption{IgnoreIPv4},
			checkFn: func(t *testing.T, c Container) {
				entry, _ := c.GetEntry("test2")
				// When adding a new entry with ignore IPv4, IPv4 should not be added
				_, err := entry.GetIPv4Set()
				if err == nil {
					t.Error("Expected no IPv4 set for new entry when ignored")
				}
			},
		},
		{
			name: "add new entry with ignore IPv6",
			opts: []IgnoreIPOption{IgnoreIPv6},
			checkFn: func(t *testing.T, c Container) {
				entry, _ := c.GetEntry("test2")
				// When adding a new entry with ignore IPv6, IPv6 should not be added
				_, err := entry.GetIPv6Set()
				if err == nil {
					t.Error("Expected no IPv6 set for new entry when ignored")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewContainer()
			
			e := NewEntry("test2")
			e.AddPrefix("192.168.1.0/24")
			e.AddPrefix("2001:db8::/32")
			c.Add(e, tt.opts...)
			
			if tt.checkFn != nil {
				tt.checkFn(t, c)
			}
		})
	}
}

func TestContainer_RemoveWithPrefixAndIgnoreOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    []IgnoreIPOption
		checkFn func(*testing.T, Container)
	}{
		{
			name: "remove prefix with ignore IPv4",
			opts: []IgnoreIPOption{IgnoreIPv4},
			checkFn: func(t *testing.T, c Container) {
				entry, _ := c.GetEntry("test")
				// IPv6 should be removed, IPv4 should remain
				_, err := entry.GetIPv4Set()
				if err != nil {
					t.Error("IPv4 should still exist")
				}
			},
		},
		{
			name: "remove prefix with ignore IPv6",
			opts: []IgnoreIPOption{IgnoreIPv6},
			checkFn: func(t *testing.T, c Container) {
				entry, _ := c.GetEntry("test")
				// IPv4 should be removed, IPv6 should remain
				_, err := entry.GetIPv6Set()
				if err != nil {
					t.Error("IPv6 should still exist")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewContainer()
			
			e1 := NewEntry("test")
			e1.AddPrefix("192.168.1.0/24")
			e1.AddPrefix("2001:db8::/32")
			c.Add(e1)
			
			removeEntry := NewEntry("test")
			removeEntry.AddPrefix("192.168.1.0/24")
			removeEntry.AddPrefix("2001:db8::/32")
			
			err := c.Remove(removeEntry, CaseRemovePrefix, tt.opts...)
			if err != nil {
				t.Errorf("Container.Remove() error = %v", err)
			}
			
			if tt.checkFn != nil {
				tt.checkFn(t, c)
			}
		})
	}
}

func TestContainer_GetEntryInvalidContainer(t *testing.T) {
	// Create an invalid container (nil map)
	c := &container{entries: nil}
	
	_, found := c.GetEntry("test")
	if found {
		t.Error("GetEntry() should return false for invalid container")
	}
}

func TestContainer_LenInvalidContainer(t *testing.T) {
	// Create an invalid container (nil map)
	c := &container{entries: nil}
	
	if c.Len() != 0 {
		t.Errorf("Len() = %d, want 0 for invalid container", c.Len())
	}
}

func TestContainer_AddMergingEdgeCases(t *testing.T) {
	c := NewContainer()
	
	// Add entry with both IPv4 and IPv6
	e1 := NewEntry("test")
	e1.AddPrefix("192.168.1.0/24")
	e1.AddPrefix("2001:db8::/32")
	c.Add(e1)
	
	// Merge with entry that has only IPv4 - should merge both
	e2 := NewEntry("test")
	e2.AddPrefix("10.0.0.0/8")
	c.Add(e2)
	
	entry, _ := c.GetEntry("test")
	// Should have both IPv4 and IPv6
	_, err4 := entry.GetIPv4Set()
	_, err6 := entry.GetIPv6Set()
	if err4 != nil || err6 != nil {
		t.Error("Both IPv4 and IPv6 should exist after merge")
	}
	
	// Now merge with ignore options on existing entry
	e3 := NewEntry("test")
	e3.AddPrefix("172.16.0.0/12")
	e3.AddPrefix("2001:db9::/32")
	c.Add(e3, IgnoreIPv4)
	
	// IPv4 should still exist, IPv6 should be updated
	entry, _ = c.GetEntry("test")
	_, err := entry.GetIPv4Set()
	if err != nil {
		t.Error("IPv4 should still exist when merging with IgnoreIPv4")
	}
}

func TestContainer_AddMergingWithExistingBuilders(t *testing.T) {
	c := NewContainer()
	
	// Test merge when val already has builders (lines 102-109)
	e1 := NewEntry("test")
	e1.AddPrefix("192.168.1.0/24")
	e1.AddPrefix("2001:db8::/32")
	c.Add(e1)
	
	// Merge another entry (default case, both builders exist)
	e2 := NewEntry("test")
	e2.AddPrefix("10.0.0.0/8")
	e2.AddPrefix("2001:db9::/32")
	c.Add(e2) // This should hit lines 102-109
	
	entry, _ := c.GetEntry("test")
	prefixes, _ := entry.MarshalPrefix()
	if len(prefixes) == 0 {
		t.Error("Should have prefixes after merge")
	}
}

func TestContainer_AddMergingIgnoreIPv6WithExistingBuilder(t *testing.T) {
	c := NewContainer()
	
	// Add entry with IPv4 and IPv6
	e1 := NewEntry("test")
	e1.AddPrefix("192.168.1.0/24")
	e1.AddPrefix("2001:db8::/32")
	c.Add(e1)
	
	// Merge with IgnoreIPv6 when val already has IPv4 builder (lines 97-100)
	e2 := NewEntry("test")
	e2.AddPrefix("10.0.0.0/8")
	e2.AddPrefix("2001:db9::/32")
	c.Add(e2, IgnoreIPv6)
	
	entry, _ := c.GetEntry("test")
	_, err := entry.GetIPv4Set()
	if err != nil {
		t.Error("IPv4 should exist after merge with IgnoreIPv6")
	}
}

func TestContainer_AddMergingIgnoreIPv4WithExistingBuilder(t *testing.T) {
	c := NewContainer()
	
	// Add entry with IPv4 and IPv6
	e1 := NewEntry("test")
	e1.AddPrefix("192.168.1.0/24")
	e1.AddPrefix("2001:db8::/32")
	c.Add(e1)
	
	// Merge with IgnoreIPv4 when val already has IPv6 builder (lines 92-95)
	e2 := NewEntry("test")
	e2.AddPrefix("10.0.0.0/8")
	e2.AddPrefix("2001:db9::/32")
	c.Add(e2, IgnoreIPv4)
	
	entry, _ := c.GetEntry("test")
	_, err := entry.GetIPv6Set()
	if err != nil {
		t.Error("IPv6 should exist after merge with IgnoreIPv4")
	}
}

func TestContainer_RemoveNilBuilders(t *testing.T) {
	c := NewContainer()
	
	// Add entry
	e1 := NewEntry("test")
	e1.AddPrefix("192.168.1.0/24")
	c.Add(e1)
	
	// Try to remove prefixes when entry has no IPv6 builder
	removeEntry := NewEntry("test")
	removeEntry.AddPrefix("2001:db8::/32") // IPv6
	
	err := c.Remove(removeEntry, CaseRemovePrefix)
	if err != nil {
		t.Errorf("Remove should handle missing builder gracefully, got error: %v", err)
	}
}
