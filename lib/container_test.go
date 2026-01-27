package lib

import (
	"fmt"
	"testing"
)

func TestNewContainer(t *testing.T) {
	c := NewContainer()
	if c == nil {
		t.Fatal("NewContainer() returned nil")
	}
	if c.Len() != 0 {
		t.Errorf("NewContainer().Len() = %d, expected 0", c.Len())
	}
}

func TestContainer_GetEntry(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		c := NewContainer()
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
		entry, ok := c.GetEntry("test")
		if !ok {
			t.Error("expected entry to be found")
		}
		if entry == nil {
			t.Error("expected non-nil entry")
		}
		if entry.GetName() != "TEST" {
			t.Errorf("entry name = %q, expected %q", entry.GetName(), "TEST")
		}
	})

	t.Run("found case insensitive", func(t *testing.T) {
		c := NewContainer()
		e := NewEntry("MyEntry")
		if err := e.AddPrefix("10.0.0.0/8"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
		// Test various cases
		for _, name := range []string{"myentry", "MYENTRY", "MyEntry", "  myentry  "} {
			entry, ok := c.GetEntry(name)
			if !ok {
				t.Errorf("expected entry to be found for %q", name)
			}
			if entry.GetName() != "MYENTRY" {
				t.Errorf("entry name = %q, expected %q for input %q", entry.GetName(), "MYENTRY", name)
			}
		}
	})

	t.Run("not found", func(t *testing.T) {
		c := NewContainer()
		entry, ok := c.GetEntry("nonexistent")
		if ok {
			t.Error("expected entry not to be found")
		}
		if entry != nil {
			t.Error("expected nil entry")
		}
	})

	t.Run("nil map", func(t *testing.T) {
		c := &container{entries: nil}
		entry, ok := c.GetEntry("test")
		if ok {
			t.Error("expected entry not to be found for nil map")
		}
		if entry != nil {
			t.Error("expected nil entry for nil map")
		}
	})
}

func TestContainer_Len(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		c := NewContainer()
		if c.Len() != 0 {
			t.Errorf("Len() = %d, expected 0", c.Len())
		}
	})

	t.Run("with entries", func(t *testing.T) {
		c := NewContainer()
		for i := 0; i < 5; i++ {
			e := NewEntry(fmt.Sprintf("test%c", 'A'+i))
			if err := e.AddPrefix(fmt.Sprintf("192.168.%d.0/24", i)); err != nil {
				t.Fatalf("AddPrefix failed: %v", err)
			}
			if err := c.Add(e); err != nil {
				t.Fatalf("Add failed: %v", err)
			}
		}
		if c.Len() != 5 {
			t.Errorf("Len() = %d, expected 5", c.Len())
		}
	})

	t.Run("nil map", func(t *testing.T) {
		c := &container{entries: nil}
		if c.Len() != 0 {
			t.Errorf("Len() = %d, expected 0 for nil map", c.Len())
		}
	})
}

func TestContainer_Loop(t *testing.T) {
	c := NewContainer()

	// Add some entries
	names := []string{"A", "B", "C"}
	for _, name := range names {
		e := NewEntry(name)
		if err := e.AddPrefix("10.0.0.0/8"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}

	// Collect entries from channel
	collected := make(map[string]bool)
	for entry := range c.Loop() {
		collected[entry.GetName()] = true
	}

	if len(collected) != len(names) {
		t.Errorf("collected %d entries, expected %d", len(collected), len(names))
	}
	for _, name := range names {
		if !collected[name] {
			t.Errorf("entry %q not found in loop", name)
		}
	}
}

func TestContainer_Add(t *testing.T) {
	t.Run("new entry", func(t *testing.T) {
		c := NewContainer()
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
		if c.Len() != 1 {
			t.Errorf("Len() = %d, expected 1", c.Len())
		}
	})

	t.Run("merge existing entry", func(t *testing.T) {
		c := NewContainer()

		e1 := NewEntry("test")
		if err := e1.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e1); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		e2 := NewEntry("test")
		if err := e2.AddPrefix("192.168.2.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e2); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		if c.Len() != 1 {
			t.Errorf("Len() = %d, expected 1 (merged)", c.Len())
		}

		entry, _ := c.GetEntry("test")
		prefixes, err := entry.MarshalPrefix()
		if err != nil {
			t.Fatalf("MarshalPrefix failed: %v", err)
		}
		if len(prefixes) != 2 {
			t.Errorf("expected 2 prefixes after merge, got %d", len(prefixes))
		}
	})

	t.Run("add with IgnoreIPv4", func(t *testing.T) {
		c := NewContainer()
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e, IgnoreIPv4); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		entry, _ := c.GetEntry("test")
		// Only IPv6 should be present
		_, err := entry.GetIPv4Set()
		if err == nil {
			t.Error("expected no IPv4 set when IgnoreIPv4 used")
		}
		_, err = entry.GetIPv6Set()
		if err != nil {
			t.Errorf("expected IPv6 set: %v", err)
		}
	})

	t.Run("add with IgnoreIPv6", func(t *testing.T) {
		c := NewContainer()
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e, IgnoreIPv6); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		entry, _ := c.GetEntry("test")
		// Only IPv4 should be present
		_, err := entry.GetIPv4Set()
		if err != nil {
			t.Errorf("expected IPv4 set: %v", err)
		}
		_, err = entry.GetIPv6Set()
		if err == nil {
			t.Error("expected no IPv6 set when IgnoreIPv6 used")
		}
	})

	t.Run("merge with IgnoreIPv4", func(t *testing.T) {
		c := NewContainer()

		e1 := NewEntry("test")
		if err := e1.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e1); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		e2 := NewEntry("test")
		if err := e2.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e2, IgnoreIPv4); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		entry, _ := c.GetEntry("test")
		// Both should be present - original IPv4 and new IPv6
		_, err := entry.GetIPv4Set()
		if err != nil {
			t.Errorf("expected IPv4 set: %v", err)
		}
		_, err = entry.GetIPv6Set()
		if err != nil {
			t.Errorf("expected IPv6 set: %v", err)
		}
	})

	t.Run("merge with IgnoreIPv6", func(t *testing.T) {
		c := NewContainer()

		e1 := NewEntry("test")
		if err := e1.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e1); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		e2 := NewEntry("test")
		if err := e2.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e2, IgnoreIPv6); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		entry, _ := c.GetEntry("test")
		// Both should be present - original IPv6 and new IPv4
		_, err := entry.GetIPv4Set()
		if err != nil {
			t.Errorf("expected IPv4 set: %v", err)
		}
		_, err = entry.GetIPv6Set()
		if err != nil {
			t.Errorf("expected IPv6 set: %v", err)
		}
	})

	t.Run("add with nil option", func(t *testing.T) {
		c := NewContainer()
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e, nil); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
		if c.Len() != 1 {
			t.Errorf("Len() = %d, expected 1", c.Len())
		}
	})

	t.Run("merge existing without IPv4 builder", func(t *testing.T) {
		c := NewContainer()

		e1 := NewEntry("test")
		if err := e1.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e1); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		e2 := NewEntry("test")
		if err := e2.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e2); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		entry, _ := c.GetEntry("test")
		prefixes, err := entry.MarshalPrefix()
		if err != nil {
			t.Fatalf("MarshalPrefix failed: %v", err)
		}
		if len(prefixes) != 2 {
			t.Errorf("expected 2 prefixes, got %d", len(prefixes))
		}
	})

	t.Run("merge existing without IPv6 builder", func(t *testing.T) {
		c := NewContainer()

		e1 := NewEntry("test")
		if err := e1.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e1); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		e2 := NewEntry("test")
		if err := e2.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e2); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		entry, _ := c.GetEntry("test")
		prefixes, err := entry.MarshalPrefix()
		if err != nil {
			t.Fatalf("MarshalPrefix failed: %v", err)
		}
		if len(prefixes) != 2 {
			t.Errorf("expected 2 prefixes, got %d", len(prefixes))
		}
	})
}

func TestContainer_Remove(t *testing.T) {
	t.Run("remove prefix", func(t *testing.T) {
		c := NewContainer()

		e1 := NewEntry("test")
		if err := e1.AddPrefix("192.168.0.0/16"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e1); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		e2 := NewEntry("test")
		if err := e2.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Remove(e2, CaseRemovePrefix); err != nil {
			t.Fatalf("Remove failed: %v", err)
		}

		entry, _ := c.GetEntry("test")
		prefixes, err := entry.MarshalPrefix()
		if err != nil {
			t.Fatalf("MarshalPrefix failed: %v", err)
		}
		// 192.168.1.0/24 should be removed from 192.168.0.0/16
		for _, p := range prefixes {
			if p.String() == "192.168.1.0/24" {
				t.Error("expected 192.168.1.0/24 to be removed")
			}
		}
	})

	t.Run("remove entry", func(t *testing.T) {
		c := NewContainer()

		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		if err := c.Remove(e, CaseRemoveEntry); err != nil {
			t.Fatalf("Remove failed: %v", err)
		}

		if c.Len() != 0 {
			t.Errorf("expected container to be empty after remove, got %d", c.Len())
		}
	})

	t.Run("remove entry not found", func(t *testing.T) {
		c := NewContainer()
		e := NewEntry("nonexistent")
		err := c.Remove(e, CaseRemoveEntry)
		if err == nil {
			t.Error("expected error for removing nonexistent entry")
		}
	})

	t.Run("unknown remove case", func(t *testing.T) {
		c := NewContainer()

		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		err := c.Remove(e, CaseRemove(999))
		if err == nil {
			t.Error("expected error for unknown remove case")
		}
	})

	t.Run("remove prefix with IgnoreIPv4", func(t *testing.T) {
		c := NewContainer()

		e1 := NewEntry("test")
		if err := e1.AddPrefix("192.168.0.0/16"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e1.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e1); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		e2 := NewEntry("test")
		if err := e2.AddPrefix("2001:db8:1::/48"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Remove(e2, CaseRemovePrefix, IgnoreIPv4); err != nil {
			t.Fatalf("Remove failed: %v", err)
		}
	})

	t.Run("remove prefix with IgnoreIPv6", func(t *testing.T) {
		c := NewContainer()

		e1 := NewEntry("test")
		if err := e1.AddPrefix("192.168.0.0/16"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e1.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e1); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		e2 := NewEntry("test")
		if err := e2.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Remove(e2, CaseRemovePrefix, IgnoreIPv6); err != nil {
			t.Fatalf("Remove failed: %v", err)
		}
	})

	t.Run("remove entry with IgnoreIPv4", func(t *testing.T) {
		c := NewContainer()

		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		if err := c.Remove(e, CaseRemoveEntry, IgnoreIPv4); err != nil {
			t.Fatalf("Remove failed: %v", err)
		}

		entry, _ := c.GetEntry("test")
		// IPv4 should remain, IPv6 should be removed
		_, err := entry.GetIPv4Set()
		if err != nil {
			t.Errorf("expected IPv4 set to remain: %v", err)
		}
	})

	t.Run("remove entry with IgnoreIPv6", func(t *testing.T) {
		c := NewContainer()

		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		if err := c.Remove(e, CaseRemoveEntry, IgnoreIPv6); err != nil {
			t.Fatalf("Remove failed: %v", err)
		}

		entry, _ := c.GetEntry("test")
		// IPv6 should remain, IPv4 should be removed
		_, err := entry.GetIPv6Set()
		if err != nil {
			t.Errorf("expected IPv6 set to remain: %v", err)
		}
	})

	t.Run("remove prefix without existing builder", func(t *testing.T) {
		c := NewContainer()

		e1 := NewEntry("test")
		if err := e1.AddPrefix("192.168.0.0/16"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e1); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		e2 := NewEntry("test")
		if err := e2.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		// This should create a new IPv6 builder just to remove from it
		if err := c.Remove(e2, CaseRemovePrefix); err != nil {
			t.Fatalf("Remove failed: %v", err)
		}
	})

	t.Run("remove with nil option", func(t *testing.T) {
		c := NewContainer()

		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		if err := c.Remove(e, CaseRemoveEntry, nil); err != nil {
			t.Fatalf("Remove failed: %v", err)
		}
	})
}

func TestContainer_Lookup(t *testing.T) {
	setup := func() Container {
		c := NewContainer()

		e1 := NewEntry("US")
		if err := e1.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e1.AddPrefix("2001:db8:1::/48"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e1); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		e2 := NewEntry("CN")
		if err := e2.AddPrefix("10.0.0.0/8"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e2.AddPrefix("2001:db8:2::/48"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e2); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		return c
	}

	t.Run("lookup IPv4 address found", func(t *testing.T) {
		c := setup()
		results, found, err := c.Lookup("192.168.1.100")
		if err != nil {
			t.Fatalf("Lookup failed: %v", err)
		}
		if !found {
			t.Error("expected to find IP")
		}
		if len(results) != 1 || results[0] != "US" {
			t.Errorf("expected [US], got %v", results)
		}
	})

	t.Run("lookup IPv4 CIDR found", func(t *testing.T) {
		c := setup()
		results, found, err := c.Lookup("192.168.1.0/25")
		if err != nil {
			t.Fatalf("Lookup failed: %v", err)
		}
		if !found {
			t.Error("expected to find CIDR")
		}
		if len(results) != 1 || results[0] != "US" {
			t.Errorf("expected [US], got %v", results)
		}
	})

	t.Run("lookup IPv6 address found", func(t *testing.T) {
		c := setup()
		results, found, err := c.Lookup("2001:db8:1::1")
		if err != nil {
			t.Fatalf("Lookup failed: %v", err)
		}
		if !found {
			t.Error("expected to find IP")
		}
		if len(results) != 1 || results[0] != "US" {
			t.Errorf("expected [US], got %v", results)
		}
	})

	t.Run("lookup IPv6 CIDR found", func(t *testing.T) {
		c := setup()
		results, found, err := c.Lookup("2001:db8:2::/64")
		if err != nil {
			t.Fatalf("Lookup failed: %v", err)
		}
		if !found {
			t.Error("expected to find CIDR")
		}
		if len(results) != 1 || results[0] != "CN" {
			t.Errorf("expected [CN], got %v", results)
		}
	})

	t.Run("lookup not found", func(t *testing.T) {
		c := setup()
		results, found, err := c.Lookup("172.16.0.1")
		if err != nil {
			t.Fatalf("Lookup failed: %v", err)
		}
		if found {
			t.Error("expected not to find IP")
		}
		if len(results) != 0 {
			t.Errorf("expected empty results, got %v", results)
		}
	})

	t.Run("lookup with search list", func(t *testing.T) {
		c := setup()
		results, found, err := c.Lookup("192.168.1.100", "US")
		if err != nil {
			t.Fatalf("Lookup failed: %v", err)
		}
		if !found {
			t.Error("expected to find IP with search list")
		}
		if len(results) != 1 || results[0] != "US" {
			t.Errorf("expected [US], got %v", results)
		}
	})

	t.Run("lookup with search list not in list", func(t *testing.T) {
		c := setup()
		results, found, err := c.Lookup("192.168.1.100", "CN")
		if err != nil {
			t.Fatalf("Lookup failed: %v", err)
		}
		if found {
			t.Error("expected not to find IP when searching in wrong list")
		}
		if len(results) != 0 {
			t.Errorf("expected empty results, got %v", results)
		}
	})

	t.Run("lookup invalid IP", func(t *testing.T) {
		c := setup()
		_, _, err := c.Lookup("not.an.ip")
		if err == nil {
			t.Error("expected error for invalid IP")
		}
	})

	t.Run("lookup invalid CIDR", func(t *testing.T) {
		c := setup()
		_, _, err := c.Lookup("192.168.1.0/33")
		if err == nil {
			t.Error("expected error for invalid CIDR")
		}
	})

	t.Run("lookup with empty search list entries", func(t *testing.T) {
		c := setup()
		results, found, err := c.Lookup("192.168.1.100", "", "  ", "US")
		if err != nil {
			t.Fatalf("Lookup failed: %v", err)
		}
		if !found {
			t.Error("expected to find IP")
		}
		if len(results) != 1 {
			t.Errorf("expected 1 result, got %d", len(results))
		}
	})

	t.Run("lookup IPv4 mapped address", func(t *testing.T) {
		c := setup()
		// Lookup using IPv4-mapped IPv6 notation - should be unmapped to IPv4
		results, found, err := c.Lookup("::ffff:192.168.1.100")
		if err != nil {
			t.Fatalf("Lookup failed: %v", err)
		}
		if !found {
			t.Error("expected to find IPv4-mapped address")
		}
		if len(results) != 1 || results[0] != "US" {
			t.Errorf("expected [US], got %v", results)
		}
	})
}

func TestContainer_isValid(t *testing.T) {
	t.Run("valid container", func(t *testing.T) {
		c := NewContainer().(*container)
		if !c.isValid() {
			t.Error("expected valid container")
		}
	})

	t.Run("nil entries", func(t *testing.T) {
		c := &container{entries: nil}
		if c.isValid() {
			t.Error("expected invalid container with nil entries")
		}
	})
}

func TestContainer_Add_MergeWithExistingBuilders(t *testing.T) {
	// Test merging when existing entry has both builders
	t.Run("merge both types when existing has both", func(t *testing.T) {
		c := NewContainer()

		e1 := NewEntry("test")
		if err := e1.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e1.AddPrefix("2001:db8:1::/48"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e1); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		e2 := NewEntry("test")
		if err := e2.AddPrefix("10.0.0.0/8"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e2.AddPrefix("2001:db8:2::/48"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := c.Add(e2); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		entry, _ := c.GetEntry("test")
		prefixes, err := entry.MarshalPrefix()
		if err != nil {
			t.Fatalf("MarshalPrefix failed: %v", err)
		}
		if len(prefixes) != 4 {
			t.Errorf("expected 4 prefixes after merge, got %d", len(prefixes))
		}
	})
}

func TestContainer_Remove_PrefixWithIPv6Only(t *testing.T) {
	c := NewContainer()

	e1 := NewEntry("test")
	if err := e1.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(e1); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	e2 := NewEntry("test")
	if err := e2.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	// Try to remove IPv4 from an entry that only has IPv6
	// This should create a new IPv4 builder just to remove from it
	if err := c.Remove(e2, CaseRemovePrefix); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
}

func TestContainer_Lookup_IPv6Only(t *testing.T) {
	c := NewContainer()

	e := NewEntry("test")
	if err := e.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(e); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Lookup IPv4 address - entry has no IPv4, should return error from GetIPv4Set
	_, _, err := c.Lookup("192.168.1.1")
	if err == nil {
		t.Error("expected error when looking up IPv4 in IPv6-only entry")
	}
}

func TestContainer_Lookup_IPv4Only(t *testing.T) {
	c := NewContainer()

	e := NewEntry("test")
	if err := e.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(e); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Lookup IPv6 address - entry has no IPv6, should return error from GetIPv6Set
	_, _, err := c.Lookup("2001:db8::1")
	if err == nil {
		t.Error("expected error when looking up IPv6 in IPv4-only entry")
	}
}

func TestContainer_Remove_CaseRemovePrefix_IPv4Only(t *testing.T) {
	c := NewContainer()

	// Create entry with only IPv4
	e1 := NewEntry("test")
	if err := e1.AddPrefix("192.168.0.0/16"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(e1); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Create entry with both IPv4 and IPv6 to remove
	e2 := NewEntry("test")
	if err := e2.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := e2.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}

	// Remove with no ignore option - should create missing IPv6 builder
	if err := c.Remove(e2, CaseRemovePrefix); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
}

func TestContainer_Remove_CaseRemovePrefix_IPv6Only(t *testing.T) {
	c := NewContainer()

	// Create entry with only IPv6
	e1 := NewEntry("test")
	if err := e1.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(e1); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Create entry with both IPv4 and IPv6 to remove
	e2 := NewEntry("test")
	if err := e2.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := e2.AddPrefix("2001:db8:1::/48"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}

	// Remove with no ignore option - should create missing IPv4 builder
	if err := c.Remove(e2, CaseRemovePrefix); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
}

func TestContainer_Remove_CaseRemovePrefix_IgnoreIPv4_NoIPv6Builder(t *testing.T) {
	c := NewContainer()

	// Create entry with only IPv4
	e1 := NewEntry("test")
	if err := e1.AddPrefix("192.168.0.0/16"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(e1); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Create entry with IPv6 to remove
	e2 := NewEntry("test")
	if err := e2.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}

	// Remove with IgnoreIPv4 - should create IPv6 builder on val to remove from
	if err := c.Remove(e2, CaseRemovePrefix, IgnoreIPv4); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
}

func TestContainer_Remove_CaseRemovePrefix_IgnoreIPv6_NoIPv4Builder(t *testing.T) {
	c := NewContainer()

	// Create entry with only IPv6
	e1 := NewEntry("test")
	if err := e1.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := c.Add(e1); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Create entry with IPv4 to remove
	e2 := NewEntry("test")
	if err := e2.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}

	// Remove with IgnoreIPv6 - should create IPv4 builder on val to remove from
	if err := c.Remove(e2, CaseRemovePrefix, IgnoreIPv6); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
}
