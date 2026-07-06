package lib

import "testing"

func TestLookupSkipsEntriesWithoutRequestedIPFamilyIPv4(t *testing.T) {
	container := NewContainer()

	ipv6Only := NewEntry("ipv6_only")
	if err := ipv6Only.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix ipv6_only failed: %v", err)
	}
	if err := container.Add(ipv6Only); err != nil {
		t.Fatalf("Add ipv6_only failed: %v", err)
	}

	ipv4Only := NewEntry("ipv4_only")
	if err := ipv4Only.AddPrefix("1.1.1.0/24"); err != nil {
		t.Fatalf("AddPrefix ipv4_only failed: %v", err)
	}
	if err := container.Add(ipv4Only); err != nil {
		t.Fatalf("Add ipv4_only failed: %v", err)
	}

	lists, found, err := container.Lookup("1.1.1.1")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if !found {
		t.Fatal("expected to find IPv4 target")
	}
	if len(lists) != 1 || lists[0] != "IPV4_ONLY" {
		t.Fatalf("unexpected lookup lists: %v", lists)
	}
}

func TestLookupSkipsEntriesWithoutRequestedIPFamilyIPv6(t *testing.T) {
	container := NewContainer()

	ipv4Only := NewEntry("ipv4_only")
	if err := ipv4Only.AddPrefix("1.1.1.0/24"); err != nil {
		t.Fatalf("AddPrefix ipv4_only failed: %v", err)
	}
	if err := container.Add(ipv4Only); err != nil {
		t.Fatalf("Add ipv4_only failed: %v", err)
	}

	ipv6Only := NewEntry("ipv6_only")
	if err := ipv6Only.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix ipv6_only failed: %v", err)
	}
	if err := container.Add(ipv6Only); err != nil {
		t.Fatalf("Add ipv6_only failed: %v", err)
	}

	lists, found, err := container.Lookup("2001:db8::1")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if !found {
		t.Fatal("expected to find IPv6 target")
	}
	if len(lists) != 1 || lists[0] != "IPV6_ONLY" {
		t.Fatalf("unexpected lookup lists: %v", lists)
	}
}
