package lib

import (
	"net/netip"
	"testing"

	"go4.org/netipx"
)

func TestNewContainerBasicOperations(t *testing.T) {
	c := NewContainer().(*container)
	if !c.isValid() {
		t.Fatalf("new container should be valid")
	}
	if c.Len() != 0 {
		t.Fatalf("expected len 0, got %d", c.Len())
	}
	if entry, ok := c.GetEntry("missing"); ok || entry != nil {
		t.Fatalf("expected missing entry")
	}

	invalid := &container{}
	if entry, ok := invalid.GetEntry("anything"); ok || entry != nil {
		t.Fatalf("expected invalid container to return nil entry")
	}
	if invalid.Len() != 0 {
		t.Fatalf("invalid container length should be 0")
	}
}

func TestContainerAddAndMerge(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("test")
	if err := entry.AddPrefix("10.0.0.0/24"); err != nil {
		t.Fatalf("AddPrefix() error = %v", err)
	}
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix() error = %v", err)
	}

	if err := c.Add(entry); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	got, ok := c.GetEntry("TEST")
	if !ok || got == nil {
		t.Fatalf("entry not found after add")
	}

	// merge with existing entry, should append new prefixes
	entry2 := NewEntry("test")
	if err := entry2.AddPrefix("192.0.2.0/24"); err != nil {
		t.Fatalf("AddPrefix() error = %v", err)
	}
	if err := c.Add(entry2); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	ipset, err := got.GetIPv4Set()
	if err != nil {
		t.Fatalf("GetIPv4Set() error = %v", err)
	}
	if !ipset.Contains(netip.MustParseAddr("10.0.0.1")) || !ipset.Contains(netip.MustParseAddr("192.0.2.1")) {
		t.Fatalf("merged IPv4 set missing prefixes")
	}
}

func TestContainerAddWithIgnore(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("mix")
	if err := entry.AddPrefix("10.1.0.0/16"); err != nil {
		t.Fatalf("AddPrefix() error = %v", err)
	}
	if err := entry.AddPrefix("2001:db8:1::/48"); err != nil {
		t.Fatalf("AddPrefix() error = %v", err)
	}

	if err := c.Add(entry, IgnoreIPv6); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	got, _ := c.GetEntry("mix")
	if got.hasIPv6Builder() {
		t.Fatalf("expected IPv6 builder to be nil when ignored")
	}
}

func TestContainerRemovePrefixAndEntry(t *testing.T) {
	c := NewContainer()
	entry := NewEntry("remove")
	_ = entry.AddPrefix("10.2.0.0/16")
	_ = entry.AddPrefix("2001:db8:2::/48")
	_ = c.Add(entry)

	// remove prefix
	removeEntry := NewEntry("remove")
	_ = removeEntry.AddPrefix("10.2.0.0/24")
	if err := c.Remove(removeEntry, CaseRemovePrefix); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	got, _ := c.GetEntry("remove")
	ipset, _ := got.GetIPv4Set()
	if ipset.Contains(netip.MustParseAddr("10.2.0.1")) {
		t.Fatalf("expected prefix to be removed")
	}

	// remove only IPv4 builder
	removeEntry2 := NewEntry("remove")
	_ = c.Remove(removeEntry2, CaseRemoveEntry, IgnoreIPv6)
	if got.hasIPv4Builder() {
		t.Fatalf("expected IPv4 builder cleared")
	}

	// remove the entry entirely
	if err := c.Remove(removeEntry2, CaseRemoveEntry); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if c.Len() != 0 {
		t.Fatalf("expected container empty after removal")
	}
}

func TestContainerRemoveBranches(t *testing.T) {
	c := NewContainer()
	entry := NewEntry("rb")
	_ = entry.AddPrefix("10.6.0.0/16")
	_ = entry.AddPrefix("2001:db8:6::/48")
	_ = c.Add(entry)

	r1 := NewEntry("rb")
	_ = r1.AddPrefix("2001:db8:6::/48")
	if err := c.Remove(r1, CaseRemovePrefix, IgnoreIPv4); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	entry.ipv6Set = nil
	if set, _ := entry.GetIPv6Set(); set.Contains(netip.MustParseAddr("2001:db8:6::1")) {
		t.Fatalf("expected ipv6 prefix removed")
	}

	r2 := NewEntry("rb")
	_ = r2.AddPrefix("10.6.0.0/16")
	if err := c.Remove(r2, CaseRemovePrefix, IgnoreIPv6); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	entry.ipv4Set = nil
	if set, _ := entry.GetIPv4Set(); set.Contains(netip.MustParseAddr("10.6.0.1")) {
		t.Fatalf("expected ipv4 prefix removed")
	}

	// Add a new IPv4 prefix and clear only IPv6 builder
	_ = entry.AddPrefix("10.6.1.0/24")
	if err := c.Remove(NewEntry("rb"), CaseRemoveEntry, IgnoreIPv4); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if entry.hasIPv6Builder() {
		t.Fatalf("expected ipv6 builder to be cleared")
	}

	// error from invalid builder
	bad := NewEntry("rb")
	bad.ipv4Builder = &netipx.IPSetBuilder{}
	bad.ipv4Builder.AddPrefix(netip.Prefix{})
	if err := c.Remove(bad, CaseRemovePrefix); err == nil {
		t.Fatalf("expected error from invalid builder")
	}

	badv6 := NewEntry("rb")
	badv6.ipv6Builder = &netipx.IPSetBuilder{}
	badv6.ipv6Builder.AddPrefix(netip.Prefix{})
	if err := c.Remove(badv6, CaseRemovePrefix); err == nil {
		t.Fatalf("expected error from invalid ipv6 builder")
	}

	// create missing builders during remove
	only6 := NewEntry("only6")
	_ = only6.AddPrefix("2001:db8:10::/48")
	_ = c.Add(only6)
	remove4 := NewEntry("only6")
	_ = remove4.AddPrefix("203.0.113.0/24")
	if err := c.Remove(remove4, CaseRemovePrefix, IgnoreIPv6); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	got, _ := c.GetEntry("only6")
	if !got.hasIPv4Builder() {
		t.Fatalf("expected ipv4 builder created during remove")
	}

	only4 := NewEntry("only4")
	_ = only4.AddPrefix("198.51.101.0/24")
	_ = c.Add(only4)
	remove6 := NewEntry("only4")
	_ = remove6.AddPrefix("2001:db8:11::/48")
	if err := c.Remove(remove6, CaseRemovePrefix, IgnoreIPv4); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	got2, _ := c.GetEntry("only4")
	if !got2.hasIPv6Builder() {
		t.Fatalf("expected ipv6 builder created during remove")
	}

	empty := &Entry{name: "EMPTY"}
	c.(*container).entries["EMPTY"] = empty
	removeEmpty := NewEntry("empty")
	_ = removeEmpty.AddPrefix("10.0.0.0/24")
	if err := c.Remove(removeEmpty, CaseRemovePrefix); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if got3, _ := c.GetEntry("empty"); !got3.hasIPv4Builder() || !got3.hasIPv6Builder() {
		t.Fatalf("expected builders created in default remove branch")
	}

	// unknown remove case with existing entry
	if err := c.Remove(NewEntry("only4"), CaseRemove(123)); err == nil {
		t.Fatalf("expected error for unknown remove case on existing entry")
	}
}
func TestContainerRemoveErrors(t *testing.T) {
	c := NewContainer()
	entry := NewEntry("missing")

	if err := c.Remove(entry, CaseRemoveEntry); err == nil {
		t.Fatalf("expected error when removing missing entry")
	}

	if err := c.Remove(entry, CaseRemove(99)); err == nil {
		t.Fatalf("expected error for unknown remove case")
	}
}

func TestContainerAddErrorAndIgnoreBranches(t *testing.T) {
	c := NewContainer()
	valid := NewEntry("mix")
	_ = valid.AddPrefix("10.5.0.0/16")
	_ = valid.AddPrefix("2001:db8:5::/48")
	_ = c.Add(valid)

	// ignore IPv4 when adding a new entry
	entry := NewEntry("newone")
	_ = entry.AddPrefix("203.0.113.0/24")
	_ = entry.AddPrefix("2001:db8:6::/48")
	if err := c.Add(entry, IgnoreIPv4); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if got, _ := c.GetEntry("newone"); got.hasIPv4Builder() {
		t.Fatalf("expected IPv4 builder nil when ignored")
	}

	// found=true path with ignore IPv4 branch
	moreIPv6 := NewEntry("mix")
	_ = moreIPv6.AddPrefix("2001:db8:7::/48")
	if err := c.Add(moreIPv6, IgnoreIPv4); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	existing, _ := c.GetEntry("mix")
	ipv6set, _ := existing.GetIPv6Set()
	if !ipv6set.Contains(netip.MustParseAddr("2001:db8:7::1")) {
		t.Fatalf("expected IPv6 prefix merged")
	}

	// found=true path with ignore IPv6 branch
	moreIPv4 := NewEntry("mix")
	_ = moreIPv4.AddPrefix("10.5.1.0/24")
	if err := c.Add(moreIPv4, IgnoreIPv6); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	existing.ipv4Set = nil
	ipv4set, _ := existing.GetIPv4Set()
	if !ipv4set.Contains(netip.MustParseAddr("10.5.1.1")) {
		t.Fatalf("expected IPv4 prefix merged")
	}

	// error from building ipset in found=true path
	bad := NewEntry("mix")
	bad.ipv4Builder = &netipx.IPSetBuilder{}
	bad.ipv4Builder.AddPrefix(netip.Prefix{}) // invalid, accumulates error
	if err := c.Add(bad); err == nil {
		t.Fatalf("expected error from invalid builder")
	}

	bad6 := NewEntry("mix")
	bad6.ipv6Builder = &netipx.IPSetBuilder{}
	bad6.ipv6Builder.AddPrefix(netip.Prefix{})
	if err := c.Add(bad6); err == nil {
		t.Fatalf("expected error from invalid ipv6 builder")
	}
}

func TestContainerAddCreatesMissingBuilders(t *testing.T) {
	c := NewContainer()

	partial := NewEntry("partial")
	_ = partial.AddPrefix("10.7.0.0/16")
	_ = c.Add(partial)

	addIPv6 := NewEntry("partial")
	_ = addIPv6.AddPrefix("2001:db8:7::/48")
	if err := c.Add(addIPv6, IgnoreIPv4); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if got, _ := c.GetEntry("partial"); !got.hasIPv6Builder() {
		t.Fatalf("expected ipv6 builder created")
	}

	partial2 := NewEntry("partial2")
	_ = partial2.AddPrefix("2001:db8:8::/48")
	_ = c.Add(partial2)

	addIPv4 := NewEntry("partial2")
	_ = addIPv4.AddPrefix("198.18.0.0/16")
	if err := c.Add(addIPv4, IgnoreIPv6); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if got, _ := c.GetEntry("partial2"); !got.hasIPv4Builder() {
		t.Fatalf("expected ipv4 builder created")
	}

	partial3 := &Entry{name: "PARTIAL3"}
	c.(*container).entries["PARTIAL3"] = partial3
	addBoth := NewEntry("partial3")
	_ = addBoth.AddPrefix("198.19.0.0/16")
	_ = addBoth.AddPrefix("2001:db8:9::/48")
	if err := c.Add(addBoth); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if got, _ := c.GetEntry("partial3"); !got.hasIPv4Builder() || !got.hasIPv6Builder() {
		t.Fatalf("expected both builders created")
	}
}
func TestContainerLookup(t *testing.T) {
	c := NewContainer()
	entry := NewEntry("zoneA")
	_ = entry.AddPrefix("10.3.0.0/16")
	_ = entry.AddPrefix("2001:db8:a::/48")
	_ = c.Add(entry)

	entry2 := NewEntry("zoneB")
	_ = entry2.AddPrefix("2001:db8:3::/48")
	_ = entry2.AddPrefix("198.51.100.0/24")
	_ = c.Add(entry2)

	t.Run("match ipv4", func(t *testing.T) {
		names, ok, err := c.Lookup("10.3.5.1")
		if err != nil {
			t.Fatalf("Lookup() error = %v", err)
		}
		if !ok || len(names) != 1 || names[0] != "ZONEA" {
			t.Fatalf("Lookup() got %v %v", names, ok)
		}
	})

	t.Run("match ipv6 prefix", func(t *testing.T) {
		names, ok, err := c.Lookup("2001:db8:3::/48")
		if err != nil {
			t.Fatalf("Lookup() error = %v", err)
		}
		if !ok || len(names) != 1 || names[0] != "ZONEB" {
			t.Fatalf("Lookup() got %v %v", names, ok)
		}
	})

	t.Run("match ipv4 prefix", func(t *testing.T) {
		names, ok, err := c.Lookup("10.3.0.0/16")
		if err != nil {
			t.Fatalf("Lookup() error = %v", err)
		}
		if !ok || len(names) != 1 || names[0] != "ZONEA" {
			t.Fatalf("Lookup() got %v %v", names, ok)
		}
	})

	t.Run("search list filters", func(t *testing.T) {
		names, ok, err := c.Lookup("10.3.5.1", "zoneB")
		if err != nil {
			t.Fatalf("Lookup() error = %v", err)
		}
		if ok || len(names) != 0 {
			t.Fatalf("expected no results when filtered out, got %v", names)
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		if _, _, err := c.Lookup("not-an-ip"); err == nil {
			t.Fatalf("expected error for invalid input")
		}
	})

	t.Run("ipv6 address lookup", func(t *testing.T) {
		names, ok, err := c.Lookup("2001:db8:a::1")
		if err != nil || !ok || len(names) != 1 {
			t.Fatalf("Lookup() ipv6 = %v %v %v", names, ok, err)
		}
	})

	t.Run("invalid prefix string", func(t *testing.T) {
		if _, _, err := c.Lookup("bad/64"); err == nil {
			t.Fatalf("expected error for invalid prefix")
		}
	})

	t.Run("entry lookup error", func(t *testing.T) {
		c2 := NewContainer()
		e := NewEntry("only6")
		_ = e.AddPrefix("2001:db8::/32")
		_ = c2.Add(e)
		if _, _, err := c2.Lookup("192.0.2.1"); err == nil {
			t.Fatalf("expected error when IPv4 set missing")
		}
	})
}

func TestContainerLoopChannel(t *testing.T) {
	c := NewContainer()
	entry := NewEntry("loop")
	_ = entry.AddPrefix("10.4.0.0/16")
	_ = c.Add(entry)

	count := 0
	for range c.(*container).Loop() {
		count++
	}
	if count != 1 {
		t.Fatalf("expected to loop over 1 entry, got %d", count)
	}
}

func TestContainerInternalLookup(t *testing.T) {
	c := &container{
		entries: map[string]*Entry{},
	}
	e := NewEntry("inner")
	_ = e.AddPrefix("203.0.113.0/24")
	_ = c.Add(e)

	prefix := netip.MustParsePrefix("203.0.113.0/24")
	names, ok, err := c.lookup(prefix, IPv4)
	if err != nil {
		t.Fatalf("lookup() error = %v", err)
	}
	if !ok || len(names) != 1 || names[0] != "INNER" {
		t.Fatalf("lookup() got %v %v", names, ok)
	}
}
