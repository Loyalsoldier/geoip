package lib

import (
	"sort"
	"testing"
)

func TestNewContainer(t *testing.T) {
	c := NewContainer()
	if c == nil {
		t.Fatal("NewContainer returned nil")
	}
	if c.Len() != 0 {
		t.Errorf("expected Len() = 0, got %d", c.Len())
	}
}

func TestContainerGetEntry(t *testing.T) {
	c := NewContainer()

	// Entry not found
	_, found := c.GetEntry("US")
	if found {
		t.Error("expected entry not found")
	}

	// Add entry and retrieve
	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	val, found := c.GetEntry("US")
	if !found {
		t.Error("expected entry to be found")
	}
	if val.GetName() != "US" {
		t.Errorf("expected name US, got %q", val.GetName())
	}

	// Case insensitive with spaces
	val, found = c.GetEntry("  us  ")
	if !found {
		t.Error("expected entry to be found with spaces and lowercase")
	}
	if val.GetName() != "US" {
		t.Errorf("expected name US, got %q", val.GetName())
	}
}

func TestContainerGetEntryInvalid(t *testing.T) {
	c := &container{entries: nil}
	_, found := c.GetEntry("US")
	if found {
		t.Error("expected entry not found on invalid container")
	}
}

func TestContainerLen(t *testing.T) {
	c := NewContainer()
	if c.Len() != 0 {
		t.Errorf("expected Len() = 0, got %d", c.Len())
	}

	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}
	if c.Len() != 1 {
		t.Errorf("expected Len() = 1, got %d", c.Len())
	}
}

func TestContainerLenInvalid(t *testing.T) {
	c := &container{entries: nil}
	if c.Len() != 0 {
		t.Errorf("expected Len() = 0 on invalid container, got %d", c.Len())
	}
}

func TestContainerLoop(t *testing.T) {
	c := NewContainer()

	entry1 := NewEntry("us")
	if err := entry1.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	entry2 := NewEntry("cn")
	if err := entry2.AddPrefix("2.0.0.0/24"); err != nil {
		t.Fatal(err)
	}

	if err := c.Add(entry1); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry2); err != nil {
		t.Fatal(err)
	}

	names := make([]string, 0)
	for entry := range c.Loop() {
		names = append(names, entry.GetName())
	}

	sort.Strings(names)
	if len(names) != 2 {
		t.Errorf("expected 2 entries, got %d", len(names))
	}
	if names[0] != "CN" || names[1] != "US" {
		t.Errorf("unexpected names: %v", names)
	}
}

func TestContainerAdd(t *testing.T) {
	c := NewContainer()

	// Add new entry
	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	// Add to existing entry (merge)
	entry2 := NewEntry("us")
	if err := entry2.AddPrefix("2.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := entry2.AddPrefix("2001:db9::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry2); err != nil {
		t.Fatal(err)
	}

	if c.Len() != 1 {
		t.Errorf("expected 1 entry after merge, got %d", c.Len())
	}

	val, found := c.GetEntry("US")
	if !found {
		t.Fatal("entry US not found")
	}
	prefixes, err := val.MarshalPrefix()
	if err != nil {
		t.Fatal(err)
	}
	// 1.0.0.0/24, 2.0.0.0/24 are separate, 2001:db8::/32 and 2001:db9::/32 are adjacent
	// so they get merged into 2001:db8::/31, resulting in 3 prefixes total
	if len(prefixes) != 3 {
		t.Errorf("expected 3 prefixes after merge, got %d", len(prefixes))
	}
}

func TestContainerAddIgnoreIPv4(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}

	// Add with IgnoreIPv4 - should only have IPv6
	if err := c.Add(entry, IgnoreIPv4); err != nil {
		t.Fatal(err)
	}

	val, _ := c.GetEntry("US")
	if val.hasIPv4Builder() {
		t.Error("entry should not have IPv4 builder when ignoring IPv4")
	}
}

func TestContainerAddIgnoreIPv6(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}

	// Add with IgnoreIPv6 - should only have IPv4
	if err := c.Add(entry, IgnoreIPv6); err != nil {
		t.Fatal(err)
	}

	val, _ := c.GetEntry("US")
	if val.hasIPv6Builder() {
		t.Error("entry should not have IPv6 builder when ignoring IPv6")
	}
}

func TestContainerAddMergeIgnoreIPv4(t *testing.T) {
	c := NewContainer()

	// First add normally
	entry1 := NewEntry("us")
	if err := entry1.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry1); err != nil {
		t.Fatal(err)
	}

	// Add with IgnoreIPv4 to existing entry
	entry2 := NewEntry("us")
	if err := entry2.AddPrefix("2.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := entry2.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry2, IgnoreIPv4); err != nil {
		t.Fatal(err)
	}

	// Verify: original IPv4 should remain, IPv6 should be merged
	val, found := c.GetEntry("US")
	if !found {
		t.Fatal("entry US not found")
	}
	prefixes, err := val.MarshalPrefix()
	if err != nil {
		t.Fatal(err)
	}
	gotIPv4, gotIPv6 := 0, 0
	for _, p := range prefixes {
		if p.Addr().Is4() {
			gotIPv4++
			if p.String() != "1.0.0.0/24" {
				t.Errorf("expected original IPv4 prefix 1.0.0.0/24, got %s", p.String())
			}
		} else {
			gotIPv6++
			if p.String() != "2001:db8::/32" {
				t.Errorf("expected merged IPv6 prefix 2001:db8::/32, got %s", p.String())
			}
		}
	}
	if gotIPv4 != 1 {
		t.Errorf("expected 1 IPv4 prefix, got %d", gotIPv4)
	}
	if gotIPv6 != 1 {
		t.Errorf("expected 1 IPv6 prefix, got %d", gotIPv6)
	}
}

func TestContainerAddMergeIgnoreIPv6(t *testing.T) {
	c := NewContainer()

	// First add normally
	entry1 := NewEntry("us")
	if err := entry1.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry1); err != nil {
		t.Fatal(err)
	}

	// Add with IgnoreIPv6 to existing entry
	entry2 := NewEntry("us")
	if err := entry2.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := entry2.AddPrefix("2001:db9::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry2, IgnoreIPv6); err != nil {
		t.Fatal(err)
	}

	// Verify: IPv4 should be merged, original IPv6 should remain
	val, found := c.GetEntry("US")
	if !found {
		t.Fatal("entry US not found")
	}
	prefixes, err := val.MarshalPrefix()
	if err != nil {
		t.Fatal(err)
	}
	gotIPv4, gotIPv6 := 0, 0
	for _, p := range prefixes {
		if p.Addr().Is4() {
			gotIPv4++
			if p.String() != "1.0.0.0/24" {
				t.Errorf("expected merged IPv4 prefix 1.0.0.0/24, got %s", p.String())
			}
		} else {
			gotIPv6++
			if p.String() != "2001:db8::/32" {
				t.Errorf("expected original IPv6 prefix 2001:db8::/32, got %s", p.String())
			}
		}
	}
	if gotIPv4 != 1 {
		t.Errorf("expected 1 IPv4 prefix, got %d", gotIPv4)
	}
	if gotIPv6 != 1 {
		t.Errorf("expected 1 IPv6 prefix, got %d", gotIPv6)
	}
}

func TestContainerAddNilOption(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	// Pass nil option
	if err := c.Add(entry, nil); err != nil {
		t.Fatal(err)
	}
}

func TestContainerRemoveCaseRemovePrefix(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := entry.AddPrefix("2.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	// Remove prefix
	removeEntry := NewEntry("us")
	if err := removeEntry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := removeEntry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Remove(removeEntry, CaseRemovePrefix); err != nil {
		t.Errorf("Remove error = %v", err)
	}

	// Verify: only 2.0.0.0/24 should remain
	val, found := c.GetEntry("US")
	if !found {
		t.Fatal("entry US not found")
	}
	prefixes, err := val.MarshalPrefix()
	if err != nil {
		t.Fatal(err)
	}
	gotIPv4, gotIPv6 := 0, 0
	for _, p := range prefixes {
		if p.Addr().Is4() {
			gotIPv4++
			if p.String() != "2.0.0.0/24" {
				t.Errorf("expected remaining IPv4 prefix 2.0.0.0/24, got %s", p.String())
			}
		} else {
			gotIPv6++
		}
	}
	if gotIPv4 != 1 {
		t.Errorf("expected 1 IPv4 prefix, got %d", gotIPv4)
	}
	if gotIPv6 != 0 {
		t.Errorf("expected 0 IPv6 prefixes, got %d", gotIPv6)
	}
}

func TestContainerRemoveCaseRemovePrefixIgnoreIPv4(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	removeEntry := NewEntry("us")
	if err := removeEntry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := removeEntry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Remove(removeEntry, CaseRemovePrefix, IgnoreIPv4); err != nil {
		t.Errorf("Remove with IgnoreIPv4 error = %v", err)
	}

	// Verify: IPv4 should be untouched (1.0.0.0/24 remains), IPv6 should be removed
	val, found := c.GetEntry("US")
	if !found {
		t.Fatal("entry US not found")
	}
	prefixes, err := val.MarshalPrefix()
	if err != nil {
		t.Fatal(err)
	}
	gotIPv4, gotIPv6 := 0, 0
	for _, p := range prefixes {
		if p.Addr().Is4() {
			gotIPv4++
			if p.String() != "1.0.0.0/24" {
				t.Errorf("expected remaining IPv4 prefix 1.0.0.0/24, got %s", p.String())
			}
		} else {
			gotIPv6++
		}
	}
	if gotIPv4 != 1 {
		t.Errorf("expected 1 IPv4 prefix, got %d", gotIPv4)
	}
	if gotIPv6 != 0 {
		t.Errorf("expected 0 IPv6 prefixes, got %d", gotIPv6)
	}
}

func TestContainerRemoveCaseRemovePrefixIgnoreIPv6(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	removeEntry := NewEntry("us")
	if err := removeEntry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := removeEntry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Remove(removeEntry, CaseRemovePrefix, IgnoreIPv6); err != nil {
		t.Errorf("Remove with IgnoreIPv6 error = %v", err)
	}

	// Verify: IPv4 should be removed, IPv6 should be untouched (2001:db8::/32 remains)
	val, found := c.GetEntry("US")
	if !found {
		t.Fatal("entry US not found")
	}
	prefixes, err := val.MarshalPrefix()
	if err != nil {
		t.Fatal(err)
	}
	gotIPv4, gotIPv6 := 0, 0
	for _, p := range prefixes {
		if p.Addr().Is4() {
			gotIPv4++
		} else {
			gotIPv6++
			if p.String() != "2001:db8::/32" {
				t.Errorf("expected remaining IPv6 prefix 2001:db8::/32, got %s", p.String())
			}
		}
	}
	if gotIPv4 != 0 {
		t.Errorf("expected 0 IPv4 prefixes, got %d", gotIPv4)
	}
	if gotIPv6 != 1 {
		t.Errorf("expected 1 IPv6 prefix, got %d", gotIPv6)
	}
}

func TestContainerRemoveCaseRemoveEntry(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	// Remove entire entry
	removeEntry := NewEntry("us")
	if err := c.Remove(removeEntry, CaseRemoveEntry); err != nil {
		t.Errorf("Remove error = %v", err)
	}

	if c.Len() != 0 {
		t.Errorf("expected 0 entries after remove, got %d", c.Len())
	}
}

func TestContainerRemoveCaseRemoveEntryIgnoreIPv4(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	removeEntry := NewEntry("us")
	if err := c.Remove(removeEntry, CaseRemoveEntry, IgnoreIPv4); err != nil {
		t.Errorf("Remove error = %v", err)
	}

	// Entry should still exist (only ipv6 builder was set to nil)
	val, found := c.GetEntry("US")
	if !found {
		t.Error("entry should still exist after CaseRemoveEntry with IgnoreIPv4")
	}
	if val.hasIPv6Builder() {
		t.Error("ipv6Builder should be nil after CaseRemoveEntry with IgnoreIPv4")
	}
}

func TestContainerRemoveCaseRemoveEntryIgnoreIPv6(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	removeEntry := NewEntry("us")
	if err := c.Remove(removeEntry, CaseRemoveEntry, IgnoreIPv6); err != nil {
		t.Errorf("Remove error = %v", err)
	}

	val, found := c.GetEntry("US")
	if !found {
		t.Error("entry should still exist after CaseRemoveEntry with IgnoreIPv6")
	}
	if val.hasIPv4Builder() {
		t.Error("ipv4Builder should be nil after CaseRemoveEntry with IgnoreIPv6")
	}
}

func TestContainerRemoveNotFound(t *testing.T) {
	c := NewContainer()
	removeEntry := NewEntry("us")
	err := c.Remove(removeEntry, CaseRemovePrefix)
	if err == nil {
		t.Error("Remove on non-existent entry should return error")
	}
}

func TestContainerRemoveUnknownCase(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	removeEntry := NewEntry("us")
	err := c.Remove(removeEntry, CaseRemove(99))
	if err == nil {
		t.Error("Remove with unknown case should return error")
	}
}

func TestContainerRemoveNilOption(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	removeEntry := NewEntry("us")
	if err := removeEntry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := c.Remove(removeEntry, CaseRemovePrefix, nil); err != nil {
		t.Errorf("Remove with nil option error = %v", err)
	}
}

func TestContainerLookupIP(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	entry2 := NewEntry("cn")
	if err := entry2.AddPrefix("2.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := entry2.AddPrefix("2001:db9::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry2); err != nil {
		t.Fatal(err)
	}

	// Lookup IPv4
	result, found, err := c.Lookup("1.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Error("expected to find 1.0.0.1")
	}
	if len(result) != 1 || result[0] != "US" {
		t.Errorf("expected [US], got %v", result)
	}

	// Lookup IPv6
	result, found, err = c.Lookup("2001:db8::1")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Error("expected to find 2001:db8::1")
	}
	if len(result) != 1 || result[0] != "US" {
		t.Errorf("expected [US], got %v", result)
	}

	// Lookup not found
	_, found, err = c.Lookup("3.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("expected not to find 3.0.0.1")
	}

	// Lookup invalid IP
	_, _, err = c.Lookup("invalid")
	if err == nil {
		t.Error("expected error for invalid IP")
	}
}

func TestContainerLookupCIDR(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/16"); err != nil {
		t.Fatal(err)
	}
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	// Lookup IPv4 CIDR
	result, found, err := c.Lookup("1.0.0.0/24")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Error("expected to find 1.0.0.0/24")
	}
	if len(result) != 1 || result[0] != "US" {
		t.Errorf("expected [US], got %v", result)
	}

	// Lookup IPv6 CIDR
	result, found, err = c.Lookup("2001:db8::/48")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Error("expected to find 2001:db8::/48")
	}
	if len(result) != 1 || result[0] != "US" {
		t.Errorf("expected [US], got %v", result)
	}

	// Lookup not found CIDR
	_, found, err = c.Lookup("3.0.0.0/24")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("expected not to find 3.0.0.0/24")
	}

	// Lookup invalid CIDR
	_, _, err = c.Lookup("invalid/24")
	if err == nil {
		t.Error("expected error for invalid CIDR")
	}
}

func TestContainerLookupWithSearchList(t *testing.T) {
	c := NewContainer()

	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	entry2 := NewEntry("cn")
	if err := entry2.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry2); err != nil {
		t.Fatal(err)
	}

	// Search with specific list
	result, found, err := c.Lookup("1.0.0.1", "us")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Error("expected to find 1.0.0.1 in US")
	}
	if len(result) != 1 || result[0] != "US" {
		t.Errorf("expected [US], got %v", result)
	}

	// Search with empty string in list (should be skipped)
	result, found, err = c.Lookup("1.0.0.1", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Error("expected to find 1.0.0.1 with empty search list")
	}
	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}

	// Search with non-existent entry
	_, found, err = c.Lookup("1.0.0.1", "jp")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("expected not to find 1.0.0.1 in JP")
	}
}

func TestContainerRemoveCaseRemovePrefixNoBuilderOnExisting(t *testing.T) {
	c := NewContainer()

	// Add entry with only IPv4
	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	// Try to remove IPv6 from entry that only has IPv4 - should initialize builder
	removeEntry := NewEntry("us")
	if err := removeEntry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Remove(removeEntry, CaseRemovePrefix); err != nil {
		t.Errorf("Remove error = %v", err)
	}
}

func TestContainerAddMergeExistingWithoutBuilders(t *testing.T) {
	c := NewContainer()

	// Add entry with only IPv4
	entry1 := NewEntry("us")
	if err := entry1.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry1); err != nil {
		t.Fatal(err)
	}

	// Add entry with only IPv6 to merge
	entry2 := NewEntry("us")
	if err := entry2.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry2); err != nil {
		t.Fatal(err)
	}

	// Entry should now have both
	val, _ := c.GetEntry("US")
	if !val.hasIPv4Builder() {
		t.Error("entry should have IPv4 builder")
	}
	if !val.hasIPv6Builder() {
		t.Error("entry should have IPv6 builder")
	}
}

func TestContainerAddMergeExistingOnlyIPv6DefaultIgnore(t *testing.T) {
	c := NewContainer()

	// First entry has only IPv6
	entry1 := NewEntry("us")
	if err := entry1.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry1); err != nil {
		t.Fatal(err)
	}

	// Add new entry with both IPv4 and IPv6 - existing lacks IPv4 builder
	entry2 := NewEntry("us")
	if err := entry2.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := entry2.AddPrefix("2001:db9::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry2); err != nil {
		t.Fatal(err)
	}

	// val should now have both builders
	val, _ := c.GetEntry("US")
	if !val.hasIPv4Builder() {
		t.Error("entry should have IPv4 builder after merge")
	}
	if !val.hasIPv6Builder() {
		t.Error("entry should have IPv6 builder after merge")
	}
}

func TestContainerRemoveCaseRemovePrefixNoIPv6BuilderIgnoreIPv4(t *testing.T) {
	c := NewContainer()

	// Add entry with only IPv4
	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	// Remove with IgnoreIPv4 from entry that only has IPv4 (no IPv6 builder on val)
	removeEntry := NewEntry("us")
	if err := removeEntry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Remove(removeEntry, CaseRemovePrefix, IgnoreIPv4); err != nil {
		t.Errorf("Remove error = %v", err)
	}
}

func TestContainerRemoveCaseRemovePrefixNoIPv4BuilderIgnoreIPv6(t *testing.T) {
	c := NewContainer()

	// Add entry with only IPv6
	entry := NewEntry("us")
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	// Remove with IgnoreIPv6 from entry that only has IPv6 (no IPv4 builder on val)
	removeEntry := NewEntry("us")
	if err := removeEntry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := c.Remove(removeEntry, CaseRemovePrefix, IgnoreIPv6); err != nil {
		t.Errorf("Remove error = %v", err)
	}
}

func TestContainerRemoveCaseRemovePrefixNoBuilderDefault(t *testing.T) {
	c := NewContainer()

	// Add entry with only IPv6 (no IPv4 builder on val)
	entry := NewEntry("us")
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	// Remove default (no ignore) from entry that lacks IPv4 builder
	removeEntry := NewEntry("us")
	if err := removeEntry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := removeEntry.AddPrefix("2001:db8::/48"); err != nil {
		t.Fatal(err)
	}
	if err := c.Remove(removeEntry, CaseRemovePrefix); err != nil {
		t.Errorf("Remove error = %v", err)
	}
}

func TestContainerRemoveCaseRemovePrefixOnlyIPv4NoIPv6Builder(t *testing.T) {
	c := NewContainer()

	// Add entry with only IPv4 (no IPv6 builder on val)
	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	// Remove default from entry that lacks IPv6 builder
	removeEntry := NewEntry("us")
	if err := removeEntry.AddPrefix("1.0.0.0/25"); err != nil {
		t.Fatal(err)
	}
	if err := removeEntry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Remove(removeEntry, CaseRemovePrefix); err != nil {
		t.Errorf("Remove error = %v", err)
	}
}

func TestContainerLookupIPv6NotFoundInEntry(t *testing.T) {
	c := NewContainer()

	// Add entry with both IPv4 and IPv6 so lookup doesn't fail
	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	// Lookup IPv6 not found in any entry
	_, found, err := c.Lookup("fd00::1")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("expected not to find fd00::1")
	}
}

func TestContainerLookupIPv4ErrorEntryMissingIPSet(t *testing.T) {
	c := NewContainer()

	// Add entry with only IPv6 (no IPv4 data)
	entry := NewEntry("us")
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	// Lookup IPv4 address - will fail because US entry has no IPv4 set
	_, _, err := c.Lookup("1.0.0.1")
	if err == nil {
		t.Error("expected error when looking up IPv4 in entry with no IPv4 data")
	}
}

func TestContainerLookupIPv6ErrorEntryMissingIPSet(t *testing.T) {
	c := NewContainer()

	// Add entry with only IPv4 (no IPv6 data)
	entry := NewEntry("us")
	if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(entry); err != nil {
		t.Fatal(err)
	}

	// Lookup IPv6 address - will fail because US entry has no IPv6 set
	_, _, err := c.Lookup("2001:db8::1")
	if err == nil {
		t.Error("expected error when looking up IPv6 in entry with no IPv6 data")
	}

	// Similarly for CIDR lookup
	_, _, err = c.Lookup("2001:db8::/32")
	if err == nil {
		t.Error("expected error when looking up IPv6 CIDR in entry with no IPv6 data")
	}
}
