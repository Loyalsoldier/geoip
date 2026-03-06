package lib

import (
	"net"
	"net/netip"
	"testing"
)

func TestNewEntry(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"us", "US"},
		{"  cn  ", "CN"},
		{"JP", "JP"},
		{"", ""},
	}
	for _, tt := range tests {
		e := NewEntry(tt.input)
		if e.GetName() != tt.expected {
			t.Errorf("NewEntry(%q).GetName() = %q, want %q", tt.input, e.GetName(), tt.expected)
		}
	}
}

func TestEntryHasBuilderAndSet(t *testing.T) {
	e := NewEntry("test")
	if e.hasIPv4Builder() {
		t.Error("new entry should not have ipv4 builder")
	}
	if e.hasIPv6Builder() {
		t.Error("new entry should not have ipv6 builder")
	}
	if e.hasIPv4Set() {
		t.Error("new entry should not have ipv4 set")
	}
	if e.hasIPv6Set() {
		t.Error("new entry should not have ipv6 set")
	}
}

func TestEntryAddPrefixString(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		wantErr bool
	}{
		{"ipv4 cidr", "1.0.0.0/24", false},
		{"ipv6 cidr", "2001:db8::/32", false},
		{"ipv4 address", "8.8.8.8", false},
		{"ipv6 address", "2001:db8::1", false},
		{"comment #", "# comment", true},
		{"comment //", "// comment", true},
		{"comment /*", "/* comment */", true},
		{"empty string", "", true},
		{"invalid string", "not-an-ip", true},
		{"invalid cidr", "999.999.999.999/24", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEntry("test")
			err := e.AddPrefix(tt.cidr)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddPrefix(%q) error = %v, wantErr %v", tt.cidr, err, tt.wantErr)
			}
		})
	}
}

func TestEntryAddPrefixNetIP(t *testing.T) {
	e := NewEntry("test")

	// net.IP IPv4
	ipv4 := net.ParseIP("1.2.3.4")
	if err := e.AddPrefix(ipv4); err != nil {
		t.Errorf("AddPrefix(net.IP IPv4) error = %v", err)
	}

	// net.IP IPv6
	ipv6 := net.ParseIP("2001:db8::1")
	if err := e.AddPrefix(ipv6); err != nil {
		t.Errorf("AddPrefix(net.IP IPv6) error = %v", err)
	}

	// Invalid net.IP
	invalidIP := net.IP{}
	if err := e.AddPrefix(invalidIP); err == nil {
		t.Error("AddPrefix(invalid net.IP) should return error")
	}
}

func TestEntryAddPrefixNetIPNet(t *testing.T) {
	e := NewEntry("test")

	// IPv4 net
	_, ipNet4, _ := net.ParseCIDR("10.0.0.0/8")
	if err := e.AddPrefix(ipNet4); err != nil {
		t.Errorf("AddPrefix(*net.IPNet IPv4) error = %v", err)
	}

	// IPv6 net
	_, ipNet6, _ := net.ParseCIDR("2001:db8::/32")
	if err := e.AddPrefix(ipNet6); err != nil {
		t.Errorf("AddPrefix(*net.IPNet IPv6) error = %v", err)
	}
}

func TestEntryAddPrefixNetipAddr(t *testing.T) {
	e := NewEntry("test")

	// netip.Addr IPv4
	addr4 := netip.MustParseAddr("1.2.3.4")
	if err := e.AddPrefix(addr4); err != nil {
		t.Errorf("AddPrefix(netip.Addr IPv4) error = %v", err)
	}

	// netip.Addr IPv6
	addr6 := netip.MustParseAddr("2001:db8::1")
	if err := e.AddPrefix(addr6); err != nil {
		t.Errorf("AddPrefix(netip.Addr IPv6) error = %v", err)
	}

	// *netip.Addr IPv4
	a4 := netip.MustParseAddr("5.6.7.8")
	if err := e.AddPrefix(&a4); err != nil {
		t.Errorf("AddPrefix(*netip.Addr IPv4) error = %v", err)
	}

	// *netip.Addr IPv6
	a6 := netip.MustParseAddr("2001:db8::2")
	if err := e.AddPrefix(&a6); err != nil {
		t.Errorf("AddPrefix(*netip.Addr IPv6) error = %v", err)
	}
}

func TestEntryAddPrefixNetipPrefix(t *testing.T) {
	e := NewEntry("test")

	// netip.Prefix IPv4
	p4 := netip.MustParsePrefix("10.0.0.0/8")
	if err := e.AddPrefix(p4); err != nil {
		t.Errorf("AddPrefix(netip.Prefix IPv4) error = %v", err)
	}

	// netip.Prefix IPv6
	p6 := netip.MustParsePrefix("2001:db8::/32")
	if err := e.AddPrefix(p6); err != nil {
		t.Errorf("AddPrefix(netip.Prefix IPv6) error = %v", err)
	}

	// *netip.Prefix IPv4
	pp4 := netip.MustParsePrefix("172.16.0.0/12")
	if err := e.AddPrefix(&pp4); err != nil {
		t.Errorf("AddPrefix(*netip.Prefix IPv4) error = %v", err)
	}

	// *netip.Prefix IPv6
	pp6 := netip.MustParsePrefix("fd00::/8")
	if err := e.AddPrefix(&pp6); err != nil {
		t.Errorf("AddPrefix(*netip.Prefix IPv6) error = %v", err)
	}
}

func TestEntryAddPrefixInvalidType(t *testing.T) {
	e := NewEntry("test")
	err := e.AddPrefix(12345)
	if err != ErrInvalidPrefixType {
		t.Errorf("AddPrefix(int) error = %v, want ErrInvalidPrefixType", err)
	}
}

func TestEntryRemovePrefix(t *testing.T) {
	e := NewEntry("test")

	// Add first
	if err := e.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := e.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}

	// Remove
	if err := e.RemovePrefix("1.0.0.0/25"); err != nil {
		t.Errorf("RemovePrefix error = %v", err)
	}
	if err := e.RemovePrefix("2001:db8::/48"); err != nil {
		t.Errorf("RemovePrefix error = %v", err)
	}

	// Remove with comment line - returns error because processPrefix returns ErrCommentLine
	// which is skipped, then remove() is called with nil prefix and empty IPType
	if err := e.RemovePrefix("# comment"); err == nil {
		t.Error("RemovePrefix comment line should return error")
	}

	// Remove invalid
	if err := e.RemovePrefix("invalid"); err == nil {
		t.Error("RemovePrefix(invalid) should return error")
	}
}

func TestEntryRemovePrefixNoBuilder(t *testing.T) {
	e := NewEntry("test")
	// Remove from entry without builders should not fail
	if err := e.RemovePrefix("1.0.0.0/24"); err != nil {
		t.Errorf("RemovePrefix without builder error = %v", err)
	}
	if err := e.RemovePrefix("2001:db8::/32"); err != nil {
		t.Errorf("RemovePrefix without builder error = %v", err)
	}
}

func TestEntryGetIPv4Set(t *testing.T) {
	e := NewEntry("test")

	// No builder -> error
	_, err := e.GetIPv4Set()
	if err == nil {
		t.Error("GetIPv4Set without builder should return error")
	}

	// With data
	if err := e.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	set, err := e.GetIPv4Set()
	if err != nil {
		t.Errorf("GetIPv4Set error = %v", err)
	}
	if set == nil {
		t.Error("GetIPv4Set returned nil set")
	}
}

func TestEntryGetIPv6Set(t *testing.T) {
	e := NewEntry("test")

	// No builder -> error
	_, err := e.GetIPv6Set()
	if err == nil {
		t.Error("GetIPv6Set without builder should return error")
	}

	// With data
	if err := e.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	set, err := e.GetIPv6Set()
	if err != nil {
		t.Errorf("GetIPv6Set error = %v", err)
	}
	if set == nil {
		t.Error("GetIPv6Set returned nil set")
	}
}

func TestEntryMarshalPrefix(t *testing.T) {
	e := NewEntry("test")
	if err := e.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := e.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}

	// All prefixes
	prefixes, err := e.MarshalPrefix()
	if err != nil {
		t.Errorf("MarshalPrefix error = %v", err)
	}
	if len(prefixes) != 2 {
		t.Errorf("expected 2 prefixes, got %d", len(prefixes))
	}

	// Ignore IPv4
	prefixes, err = e.MarshalPrefix(IgnoreIPv4)
	if err != nil {
		t.Errorf("MarshalPrefix(IgnoreIPv4) error = %v", err)
	}
	for _, p := range prefixes {
		if p.Addr().Is4() {
			t.Error("should not contain IPv4 prefix when ignoring IPv4")
		}
	}

	// Ignore IPv6
	prefixes, err = e.MarshalPrefix(IgnoreIPv6)
	if err != nil {
		t.Errorf("MarshalPrefix(IgnoreIPv6) error = %v", err)
	}
	for _, p := range prefixes {
		if p.Addr().Is6() && !p.Addr().Is4In6() {
			t.Error("should not contain IPv6 prefix when ignoring IPv6")
		}
	}

	// With nil option
	prefixes, err = e.MarshalPrefix(nil)
	if err != nil {
		t.Errorf("MarshalPrefix(nil) error = %v", err)
	}
	if len(prefixes) != 2 {
		t.Errorf("expected 2 prefixes with nil option, got %d", len(prefixes))
	}
}

func TestEntryMarshalPrefixEmpty(t *testing.T) {
	e := NewEntry("test")
	_, err := e.MarshalPrefix()
	if err == nil {
		t.Error("MarshalPrefix on empty entry should return error")
	}
}

func TestEntryMarshalIPRange(t *testing.T) {
	e := NewEntry("test")
	if err := e.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := e.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}

	// All ranges
	ranges, err := e.MarshalIPRange()
	if err != nil {
		t.Errorf("MarshalIPRange error = %v", err)
	}
	if len(ranges) != 2 {
		t.Errorf("expected 2 ranges, got %d", len(ranges))
	}

	// Ignore IPv4
	ranges, err = e.MarshalIPRange(IgnoreIPv4)
	if err != nil {
		t.Errorf("MarshalIPRange(IgnoreIPv4) error = %v", err)
	}
	if len(ranges) != 1 {
		t.Errorf("expected 1 range, got %d", len(ranges))
	}

	// Ignore IPv6
	ranges, err = e.MarshalIPRange(IgnoreIPv6)
	if err != nil {
		t.Errorf("MarshalIPRange(IgnoreIPv6) error = %v", err)
	}
	if len(ranges) != 1 {
		t.Errorf("expected 1 range, got %d", len(ranges))
	}

	// With nil option
	ranges, err = e.MarshalIPRange(nil)
	if err != nil {
		t.Errorf("MarshalIPRange(nil) error = %v", err)
	}
	if len(ranges) != 2 {
		t.Errorf("expected 2 ranges with nil option, got %d", len(ranges))
	}
}

func TestEntryMarshalIPRangeEmpty(t *testing.T) {
	e := NewEntry("test")
	_, err := e.MarshalIPRange()
	if err == nil {
		t.Error("MarshalIPRange on empty entry should return error")
	}
}

func TestEntryMarshalText(t *testing.T) {
	e := NewEntry("test")
	if err := e.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := e.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}

	// All text
	text, err := e.MarshalText()
	if err != nil {
		t.Errorf("MarshalText error = %v", err)
	}
	if len(text) != 2 {
		t.Errorf("expected 2 text entries, got %d", len(text))
	}

	// Ignore IPv4
	text, err = e.MarshalText(IgnoreIPv4)
	if err != nil {
		t.Errorf("MarshalText(IgnoreIPv4) error = %v", err)
	}
	if len(text) != 1 {
		t.Errorf("expected 1 text entry, got %d", len(text))
	}

	// Ignore IPv6
	text, err = e.MarshalText(IgnoreIPv6)
	if err != nil {
		t.Errorf("MarshalText(IgnoreIPv6) error = %v", err)
	}
	if len(text) != 1 {
		t.Errorf("expected 1 text entry, got %d", len(text))
	}

	// With nil option
	text, err = e.MarshalText(nil)
	if err != nil {
		t.Errorf("MarshalText(nil) error = %v", err)
	}
	if len(text) != 2 {
		t.Errorf("expected 2 text entries with nil option, got %d", len(text))
	}
}

func TestEntryMarshalTextEmpty(t *testing.T) {
	e := NewEntry("test")
	_, err := e.MarshalText()
	if err == nil {
		t.Error("MarshalText on empty entry should return error")
	}
}

func TestEntryAddInvalidIPType(t *testing.T) {
	e := NewEntry("test")
	prefix := netip.MustParsePrefix("1.0.0.0/24")
	err := e.add(&prefix, IPType("invalid"))
	if err != ErrInvalidIPType {
		t.Errorf("add with invalid IPType error = %v, want ErrInvalidIPType", err)
	}
}

func TestEntryRemoveInvalidIPType(t *testing.T) {
	e := NewEntry("test")
	prefix := netip.MustParsePrefix("1.0.0.0/24")
	err := e.remove(&prefix, IPType("invalid"))
	if err != ErrInvalidIPType {
		t.Errorf("remove with invalid IPType error = %v, want ErrInvalidIPType", err)
	}
}

func TestProcessPrefixStringCIDRWithComment(t *testing.T) {
	e := NewEntry("test")

	// CIDR with trailing comment
	prefix, ipType, err := e.processPrefix("10.0.0.0/8 # comment")
	if err != nil {
		t.Errorf("processPrefix CIDR with comment error = %v", err)
	}
	if ipType != IPv4 {
		t.Errorf("expected IPv4, got %q", ipType)
	}
	if prefix == nil {
		t.Error("prefix should not be nil")
	}

	// IP with trailing comment
	prefix, ipType, err = e.processPrefix("10.0.0.1 // comment")
	if err != nil {
		t.Errorf("processPrefix IP with comment error = %v", err)
	}
	if ipType != IPv4 {
		t.Errorf("expected IPv4, got %q", ipType)
	}
	if prefix == nil {
		t.Error("prefix should not be nil")
	}
}

func TestProcessPrefixIPv4MappedIPv6(t *testing.T) {
	e := NewEntry("test")

	// netip.Addr - IPv4-mapped IPv6
	mapped := netip.MustParseAddr("::ffff:1.2.3.4")
	prefix, ipType, err := e.processPrefix(mapped)
	if err != nil {
		t.Errorf("processPrefix mapped addr error = %v", err)
	}
	if ipType != IPv4 {
		t.Errorf("expected IPv4 for mapped address, got %q", ipType)
	}
	if prefix == nil {
		t.Error("prefix should not be nil")
	}

	// *netip.Addr - IPv4-mapped IPv6
	mapped2 := netip.MustParseAddr("::ffff:5.6.7.8")
	prefix, ipType, err = e.processPrefix(&mapped2)
	if err != nil {
		t.Errorf("processPrefix *mapped addr error = %v", err)
	}
	if ipType != IPv4 {
		t.Errorf("expected IPv4 for *mapped address, got %q", ipType)
	}
	if prefix == nil {
		t.Error("prefix should not be nil")
	}
}

func TestProcessPrefixNetipPrefixIs4In6(t *testing.T) {
	e := NewEntry("test")

	// netip.Prefix with IPv4-in-IPv6 address
	p := netip.MustParsePrefix("::ffff:10.0.0.0/104")
	prefix, ipType, err := e.processPrefix(p)
	if err != nil {
		t.Errorf("processPrefix 4in6 prefix error = %v", err)
	}
	if ipType != IPv4 {
		t.Errorf("expected IPv4, got %q", ipType)
	}
	if prefix == nil {
		t.Error("prefix should not be nil")
	}

	// *netip.Prefix with IPv4-in-IPv6 address
	pp := netip.MustParsePrefix("::ffff:10.0.0.0/104")
	prefix, ipType, err = e.processPrefix(&pp)
	if err != nil {
		t.Errorf("processPrefix *4in6 prefix error = %v", err)
	}
	if ipType != IPv4 {
		t.Errorf("expected IPv4, got %q", ipType)
	}
	if prefix == nil {
		t.Error("prefix should not be nil")
	}
}

func TestProcessPrefixNetipPrefixIs4In6InvalidBits(t *testing.T) {
	e := NewEntry("test")

	// netip.Prefix with IPv4-in-IPv6 address and bits < 96
	p := netip.MustParsePrefix("::ffff:0.0.0.0/80")
	_, _, err := e.processPrefix(p)
	if err != ErrInvalidPrefix {
		t.Errorf("expected ErrInvalidPrefix for 4in6 with bits < 96, got %v", err)
	}

	// *netip.Prefix with IPv4-in-IPv6 address and bits < 96
	pp := netip.MustParsePrefix("::ffff:0.0.0.0/80")
	_, _, err = e.processPrefix(&pp)
	if err != ErrInvalidPrefix {
		t.Errorf("expected ErrInvalidPrefix for *4in6 with bits < 96, got %v", err)
	}
}

func TestProcessPrefixStringCIDRIPv6(t *testing.T) {
	e := NewEntry("test")
	// Test normal IPv6 CIDR
	prefix, ipType, err := e.processPrefix("fe80::/10")
	if err != nil {
		t.Errorf("processPrefix IPv6 CIDR error = %v", err)
	}
	if ipType != IPv6 {
		t.Errorf("expected IPv6, got %q", ipType)
	}
	if prefix == nil {
		t.Error("prefix should not be nil")
	}
}

func TestProcessPrefixNetIPIPv4(t *testing.T) {
	e := NewEntry("test")

	// net.IP with IPv4 that is 16-byte form (IPv4-mapped IPv6)
	ip := net.ParseIP("1.2.3.4")
	prefix, ipType, err := e.processPrefix(ip)
	if err != nil {
		t.Errorf("processPrefix(net.IP v4) error = %v", err)
	}
	if ipType != IPv4 {
		t.Errorf("expected IPv4, got %q", ipType)
	}
	if prefix == nil {
		t.Error("prefix should not be nil")
	}
}

func TestEntryBuildIPSetIdempotent(t *testing.T) {
	e := NewEntry("test")
	if err := e.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := e.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}

	// Build once
	if err := e.buildIPSet(); err != nil {
		t.Fatal(err)
	}
	set4 := e.ipv4Set
	set6 := e.ipv6Set

	// Build again - should reuse
	if err := e.buildIPSet(); err != nil {
		t.Fatal(err)
	}
	if e.ipv4Set != set4 {
		t.Error("ipv4Set should be reused on second build")
	}
	if e.ipv6Set != set6 {
		t.Error("ipv6Set should be reused on second build")
	}
}

func TestEntryMarshalPrefixOnlyIPv4(t *testing.T) {
	e := NewEntry("test")
	if err := e.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}

	prefixes, err := e.MarshalPrefix()
	if err != nil {
		t.Errorf("MarshalPrefix error = %v", err)
	}
	if len(prefixes) != 1 {
		t.Errorf("expected 1 prefix, got %d", len(prefixes))
	}

	// Ignore IPv4 -> should fail because only IPv4 exists
	_, err = e.MarshalPrefix(IgnoreIPv4)
	if err == nil {
		t.Error("MarshalPrefix(IgnoreIPv4) should error when only IPv4 data exists")
	}
}

func TestEntryMarshalPrefixOnlyIPv6(t *testing.T) {
	e := NewEntry("test")
	if err := e.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}

	prefixes, err := e.MarshalPrefix()
	if err != nil {
		t.Errorf("MarshalPrefix error = %v", err)
	}
	if len(prefixes) != 1 {
		t.Errorf("expected 1 prefix, got %d", len(prefixes))
	}

	// Ignore IPv6 -> should fail because only IPv6 exists
	_, err = e.MarshalPrefix(IgnoreIPv6)
	if err == nil {
		t.Error("MarshalPrefix(IgnoreIPv6) should error when only IPv6 data exists")
	}
}

func TestEntryMarshalIPRangeOnlyIPv4(t *testing.T) {
	e := NewEntry("test")
	if err := e.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}

	// Ignore IPv4 -> should fail
	_, err := e.MarshalIPRange(IgnoreIPv4)
	if err == nil {
		t.Error("MarshalIPRange(IgnoreIPv4) should error when only IPv4 data exists")
	}
}

func TestEntryMarshalTextOnlyIPv4(t *testing.T) {
	e := NewEntry("test")
	if err := e.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}

	// Ignore IPv4 -> should fail
	_, err := e.MarshalText(IgnoreIPv4)
	if err == nil {
		t.Error("MarshalText(IgnoreIPv4) should error when only IPv4 data exists")
	}
}

func TestProcessPrefixInvalidNetIPNet(t *testing.T) {
	e := NewEntry("test")
	// Create an invalid *net.IPNet
	invalidIPNet := &net.IPNet{
		IP:   nil,
		Mask: nil,
	}
	_, _, err := e.processPrefix(invalidIPNet)
	if err == nil {
		t.Error("processPrefix with invalid *net.IPNet should return error")
	}
}
