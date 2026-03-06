package lib

import (
	"net"
	"net/netip"
	"testing"
)

func TestNewEntry(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple name", "test", "TEST"},
		{"uppercase name", "TEST", "TEST"},
		{"lowercase name", "test", "TEST"},
		{"with spaces", "  test  ", "TEST"},
		{"mixed case", "TeSt", "TEST"},
		{"empty", "", ""},
		{"spaces only", "   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := NewEntry(tt.input)
			if entry.GetName() != tt.expected {
				t.Errorf("NewEntry(%q).GetName() = %q, want %q", tt.input, entry.GetName(), tt.expected)
			}
		})
	}
}

func TestEntry_GetName(t *testing.T) {
	entry := NewEntry("myentry")
	if entry.GetName() != "MYENTRY" {
		t.Errorf("Entry.GetName() = %q, want %q", entry.GetName(), "MYENTRY")
	}
}

func TestEntry_AddPrefix_String(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		wantErr bool
		errType error
	}{
		// Valid IPv4
		{"valid IPv4 CIDR", "192.168.1.0/24", false, nil},
		{"valid IPv4 address", "192.168.1.1", false, nil},
		{"valid IPv4 /32", "10.0.0.1/32", false, nil},

		// Valid IPv6
		{"valid IPv6 CIDR", "2001:db8::/32", false, nil},
		{"valid IPv6 address", "2001:db8::1", false, nil},
		{"valid IPv6 /128", "fe80::1/128", false, nil},

		// Comment lines and prefixes with comments
		{"IP with comment #", "192.168.1.0/24 # comment", false, nil},
		{"IP with comment //", "192.168.1.0/24 // comment", false, nil},
		{"IP with comment /*", "192.168.1.0/24 /* comment", false, nil},

		// Invalid inputs
		{"invalid CIDR", "192.168.1.0/33", true, ErrInvalidCIDR},
		{"invalid IP", "invalid", true, ErrInvalidIP},
		{"invalid format", "192.168.1", true, ErrInvalidIP},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := NewEntry("test")
			err := entry.AddPrefix(tt.prefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("Entry.AddPrefix(%q) error = %v, wantErr %v", tt.prefix, err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errType != nil && err != tt.errType {
				t.Errorf("Entry.AddPrefix(%q) error = %v, want %v", tt.prefix, err, tt.errType)
			}
		})
	}
}

// Separate test for comment-only lines since they have special handling
func TestEntry_AddPrefix_CommentLines(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{"comment with #", "# comment"},
		{"comment with //", "// comment"},
		{"comment with /*", "/* comment"},
		{"only spaces", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := NewEntry("test")
			err := entry.AddPrefix(tt.prefix)
			// Comment lines should return ErrInvalidIPType because add() is called with nil prefix
			if err != ErrInvalidIPType {
				t.Errorf("Entry.AddPrefix(%q) error = %v, want %v", tt.prefix, err, ErrInvalidIPType)
			}
		})
	}
}

func TestEntry_AddPrefix_NetIP(t *testing.T) {
	entry := NewEntry("test")

	// Test with net.IP (IPv4)
	ip := net.ParseIP("192.168.1.1")
	err := entry.AddPrefix(ip)
	if err != nil {
		t.Errorf("Entry.AddPrefix(net.IP) error = %v, want nil", err)
	}

	// Test with *net.IPNet
	_, ipnet, _ := net.ParseCIDR("10.0.0.0/8")
	err = entry.AddPrefix(ipnet)
	if err != nil {
		t.Errorf("Entry.AddPrefix(*net.IPNet) error = %v, want nil", err)
	}

	// Test with netip.Addr (IPv4)
	addr := netip.MustParseAddr("172.16.0.1")
	err = entry.AddPrefix(addr)
	if err != nil {
		t.Errorf("Entry.AddPrefix(netip.Addr) error = %v, want nil", err)
	}

	// Test with *netip.Addr
	addrPtr := netip.MustParseAddr("172.16.0.2")
	err = entry.AddPrefix(&addrPtr)
	if err != nil {
		t.Errorf("Entry.AddPrefix(*netip.Addr) error = %v, want nil", err)
	}

	// Test with netip.Prefix
	prefix := netip.MustParsePrefix("192.0.2.0/24")
	err = entry.AddPrefix(prefix)
	if err != nil {
		t.Errorf("Entry.AddPrefix(netip.Prefix) error = %v, want nil", err)
	}

	// Test with *netip.Prefix
	prefixPtr := netip.MustParsePrefix("198.51.100.0/24")
	err = entry.AddPrefix(&prefixPtr)
	if err != nil {
		t.Errorf("Entry.AddPrefix(*netip.Prefix) error = %v, want nil", err)
	}
}

func TestEntry_AddPrefix_IPv6(t *testing.T) {
	entry := NewEntry("test")

	// Test with net.IP (IPv6)
	ip := net.ParseIP("2001:db8::1")
	err := entry.AddPrefix(ip)
	if err != nil {
		t.Errorf("Entry.AddPrefix(net.IP IPv6) error = %v, want nil", err)
	}

	// Test with netip.Addr (IPv6)
	addr := netip.MustParseAddr("2001:db8::2")
	err = entry.AddPrefix(addr)
	if err != nil {
		t.Errorf("Entry.AddPrefix(netip.Addr IPv6) error = %v, want nil", err)
	}

	// Test with *netip.Addr (IPv6)
	addrPtr := netip.MustParseAddr("2001:db8::3")
	err = entry.AddPrefix(&addrPtr)
	if err != nil {
		t.Errorf("Entry.AddPrefix(*netip.Addr IPv6) error = %v, want nil", err)
	}

	// Test with netip.Prefix (IPv6)
	prefix := netip.MustParsePrefix("2001:db8::/32")
	err = entry.AddPrefix(prefix)
	if err != nil {
		t.Errorf("Entry.AddPrefix(netip.Prefix IPv6) error = %v, want nil", err)
	}

	// Test with *netip.Prefix (IPv6)
	prefixPtr := netip.MustParsePrefix("2001:db8:1::/48")
	err = entry.AddPrefix(&prefixPtr)
	if err != nil {
		t.Errorf("Entry.AddPrefix(*netip.Prefix IPv6) error = %v, want nil", err)
	}
}

func TestEntry_AddPrefix_IPv4In6(t *testing.T) {
	entry := NewEntry("test")

	// IPv4-mapped IPv6 address should be converted to IPv4
	prefix := netip.MustParsePrefix("::ffff:192.168.1.0/120")
	err := entry.AddPrefix(prefix)
	if err != nil {
		t.Errorf("Entry.AddPrefix(IPv4-in-IPv6) error = %v, want nil", err)
	}

	// Test with pointer
	prefixPtr := netip.MustParsePrefix("::ffff:10.0.0.0/104")
	err = entry.AddPrefix(&prefixPtr)
	if err != nil {
		t.Errorf("Entry.AddPrefix(*IPv4-in-IPv6) error = %v, want nil", err)
	}

	// Invalid IPv4-in-IPv6 prefix (bits < 96)
	invalidPrefix := netip.MustParsePrefix("::ffff:192.168.1.0/95")
	err = entry.AddPrefix(invalidPrefix)
	if err != ErrInvalidPrefix {
		t.Errorf("Entry.AddPrefix(invalid IPv4-in-IPv6) error = %v, want %v", err, ErrInvalidPrefix)
	}

	// Test with pointer
	err = entry.AddPrefix(&invalidPrefix)
	if err != ErrInvalidPrefix {
		t.Errorf("Entry.AddPrefix(*invalid IPv4-in-IPv6) error = %v, want %v", err, ErrInvalidPrefix)
	}

	// IPv4-mapped IPv6 CIDR string should NOT error - it gets converted
	err = entry.AddPrefix("::ffff:192.168.1.0/120")
	if err != nil {
		t.Errorf("Entry.AddPrefix(IPv4-mapped IPv6 string) error = %v, want nil", err)
	}
}

func TestEntry_AddPrefix_InvalidTypes(t *testing.T) {
	entry := NewEntry("test")

	// Invalid type
	err := entry.AddPrefix(123)
	if err != ErrInvalidPrefixType {
		t.Errorf("Entry.AddPrefix(int) error = %v, want %v", err, ErrInvalidPrefixType)
	}

	// Invalid net.IP
	invalidIP := net.IP{}
	err = entry.AddPrefix(invalidIP)
	if err != ErrInvalidIP {
		t.Errorf("Entry.AddPrefix(invalid net.IP) error = %v, want %v", err, ErrInvalidIP)
	}
}

func TestEntry_RemovePrefix(t *testing.T) {
	entry := NewEntry("test")

	// Add some prefixes first
	entry.AddPrefix("192.168.1.0/24")
	entry.AddPrefix("10.0.0.0/8")

	// Remove a prefix
	err := entry.RemovePrefix("192.168.1.0/24")
	if err != nil {
		t.Errorf("Entry.RemovePrefix() error = %v, want nil", err)
	}

	// Remove with comment
	err = entry.RemovePrefix("10.0.0.0/8 # comment")
	if err != nil {
		t.Errorf("Entry.RemovePrefix() with comment error = %v, want nil", err)
	}

	// Remove invalid CIDR
	err = entry.RemovePrefix("invalid")
	if err != ErrInvalidIP {
		t.Errorf("Entry.RemovePrefix(invalid) error = %v, want %v", err, ErrInvalidIP)
	}
}

func TestEntry_GetIPv4Set(t *testing.T) {
	entry := NewEntry("test")

	// Should error when no IPv4 set
	_, err := entry.GetIPv4Set()
	if err == nil {
		t.Error("Entry.GetIPv4Set() on empty entry expected error, got nil")
	}

	// Add IPv4 prefix
	entry.AddPrefix("192.168.1.0/24")

	// Should succeed now
	ipset, err := entry.GetIPv4Set()
	if err != nil {
		t.Errorf("Entry.GetIPv4Set() after adding prefix error = %v, want nil", err)
	}
	if ipset == nil {
		t.Error("Entry.GetIPv4Set() returned nil IPSet")
	}

	// Verify the set contains our prefix
	addr := netip.MustParseAddr("192.168.1.1")
	if !ipset.Contains(addr) {
		t.Errorf("IPv4 set doesn't contain expected address %v", addr)
	}
}

func TestEntry_GetIPv6Set(t *testing.T) {
	entry := NewEntry("test")

	// Should error when no IPv6 set
	_, err := entry.GetIPv6Set()
	if err == nil {
		t.Error("Entry.GetIPv6Set() on empty entry expected error, got nil")
	}

	// Add IPv6 prefix
	entry.AddPrefix("2001:db8::/32")

	// Should succeed now
	ipset, err := entry.GetIPv6Set()
	if err != nil {
		t.Errorf("Entry.GetIPv6Set() after adding prefix error = %v, want nil", err)
	}
	if ipset == nil {
		t.Error("Entry.GetIPv6Set() returned nil IPSet")
	}

	// Verify the set contains our prefix
	addr := netip.MustParseAddr("2001:db8::1")
	if !ipset.Contains(addr) {
		t.Errorf("IPv6 set doesn't contain expected address %v", addr)
	}
}

func TestEntry_MarshalPrefix(t *testing.T) {
	entry := NewEntry("test")

	// Should error when no prefixes
	_, err := entry.MarshalPrefix()
	if err == nil {
		t.Error("Entry.MarshalPrefix() on empty entry expected error, got nil")
	}

	// Add IPv4 and IPv6 prefixes
	entry.AddPrefix("192.168.1.0/24")
	entry.AddPrefix("2001:db8::/32")

	// Test without options
	prefixes, err := entry.MarshalPrefix()
	if err != nil {
		t.Errorf("Entry.MarshalPrefix() error = %v, want nil", err)
	}
	if len(prefixes) != 2 {
		t.Errorf("Entry.MarshalPrefix() returned %d prefixes, want 2", len(prefixes))
	}

	// Test with IgnoreIPv4
	prefixes, err = entry.MarshalPrefix(IgnoreIPv4)
	if err != nil {
		t.Errorf("Entry.MarshalPrefix(IgnoreIPv4) error = %v, want nil", err)
	}
	if len(prefixes) != 1 {
		t.Errorf("Entry.MarshalPrefix(IgnoreIPv4) returned %d prefixes, want 1", len(prefixes))
	}
	if !prefixes[0].Addr().Is6() {
		t.Error("Entry.MarshalPrefix(IgnoreIPv4) should return only IPv6 prefixes")
	}

	// Test with IgnoreIPv6
	prefixes, err = entry.MarshalPrefix(IgnoreIPv6)
	if err != nil {
		t.Errorf("Entry.MarshalPrefix(IgnoreIPv6) error = %v, want nil", err)
	}
	if len(prefixes) != 1 {
		t.Errorf("Entry.MarshalPrefix(IgnoreIPv6) returned %d prefixes, want 1", len(prefixes))
	}
	if !prefixes[0].Addr().Is4() {
		t.Error("Entry.MarshalPrefix(IgnoreIPv6) should return only IPv4 prefixes")
	}
}

func TestEntry_MarshalIPRange(t *testing.T) {
	entry := NewEntry("test")

	// Should error when no prefixes
	_, err := entry.MarshalIPRange()
	if err == nil {
		t.Error("Entry.MarshalIPRange() on empty entry expected error, got nil")
	}

	// Add IPv4 and IPv6 prefixes
	entry.AddPrefix("192.168.1.0/24")
	entry.AddPrefix("2001:db8::/32")

	// Test without options
	ipranges, err := entry.MarshalIPRange()
	if err != nil {
		t.Errorf("Entry.MarshalIPRange() error = %v, want nil", err)
	}
	if len(ipranges) != 2 {
		t.Errorf("Entry.MarshalIPRange() returned %d ranges, want 2", len(ipranges))
	}

	// Test with IgnoreIPv4
	ipranges, err = entry.MarshalIPRange(IgnoreIPv4)
	if err != nil {
		t.Errorf("Entry.MarshalIPRange(IgnoreIPv4) error = %v, want nil", err)
	}
	if len(ipranges) != 1 {
		t.Errorf("Entry.MarshalIPRange(IgnoreIPv4) returned %d ranges, want 1", len(ipranges))
	}

	// Test with IgnoreIPv6
	ipranges, err = entry.MarshalIPRange(IgnoreIPv6)
	if err != nil {
		t.Errorf("Entry.MarshalIPRange(IgnoreIPv6) error = %v, want nil", err)
	}
	if len(ipranges) != 1 {
		t.Errorf("Entry.MarshalIPRange(IgnoreIPv6) returned %d ranges, want 1", len(ipranges))
	}
}

func TestEntry_MarshalText(t *testing.T) {
	entry := NewEntry("test")

	// Should error when no prefixes
	_, err := entry.MarshalText()
	if err == nil {
		t.Error("Entry.MarshalText() on empty entry expected error, got nil")
	}

	// Add IPv4 and IPv6 prefixes
	entry.AddPrefix("192.168.1.0/24")
	entry.AddPrefix("2001:db8::/32")

	// Test without options
	cidrs, err := entry.MarshalText()
	if err != nil {
		t.Errorf("Entry.MarshalText() error = %v, want nil", err)
	}
	if len(cidrs) != 2 {
		t.Errorf("Entry.MarshalText() returned %d CIDRs, want 2", len(cidrs))
	}

	// Test with IgnoreIPv4
	cidrs, err = entry.MarshalText(IgnoreIPv4)
	if err != nil {
		t.Errorf("Entry.MarshalText(IgnoreIPv4) error = %v, want nil", err)
	}
	if len(cidrs) != 1 {
		t.Errorf("Entry.MarshalText(IgnoreIPv4) returned %d CIDRs, want 1", len(cidrs))
	}

	// Test with IgnoreIPv6
	cidrs, err = entry.MarshalText(IgnoreIPv6)
	if err != nil {
		t.Errorf("Entry.MarshalText(IgnoreIPv6) error = %v, want nil", err)
	}
	if len(cidrs) != 1 {
		t.Errorf("Entry.MarshalText(IgnoreIPv6) returned %d CIDRs, want 1", len(cidrs))
	}

	// Verify CIDRs are strings in correct format
	for _, cidr := range cidrs {
		_, err := netip.ParsePrefix(cidr)
		if err != nil {
			t.Errorf("Entry.MarshalText() returned invalid CIDR %q: %v", cidr, err)
		}
	}
}

func TestEntry_MultipleNilOptions(t *testing.T) {
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")

	// Test with multiple nil options
	prefixes, err := entry.MarshalPrefix(nil, nil, nil)
	if err != nil {
		t.Errorf("Entry.MarshalPrefix(nil, nil, nil) error = %v, want nil", err)
	}
	if len(prefixes) != 1 {
		t.Errorf("Entry.MarshalPrefix(nil, nil, nil) returned %d prefixes, want 1", len(prefixes))
	}
}

func TestEntry_InvalidPrefixInBuilder(t *testing.T) {
	entry := NewEntry("test")

	// Test invalid IP type in add/remove
	prefix := netip.MustParsePrefix("192.168.1.0/24")

	err := entry.add(&prefix, IPType("invalid"))
	if err != ErrInvalidIPType {
		t.Errorf("Entry.add() with invalid IPType error = %v, want %v", err, ErrInvalidIPType)
	}

	err = entry.remove(&prefix, IPType("invalid"))
	if err != ErrInvalidIPType {
		t.Errorf("Entry.remove() with invalid IPType error = %v, want %v", err, ErrInvalidIPType)
	}
}

func TestEntry_BuildIPSetError(t *testing.T) {
	entry := NewEntry("test")

	// Add a valid prefix to create a builder
	entry.AddPrefix("192.168.1.0/24")

	// Get the IPv4 set (should succeed)
	_, err := entry.GetIPv4Set()
	if err != nil {
		t.Errorf("Entry.GetIPv4Set() error = %v, want nil", err)
	}

	// Calling again should still work (cached)
	_, err = entry.GetIPv4Set()
	if err != nil {
		t.Errorf("Entry.GetIPv4Set() second call error = %v, want nil", err)
	}
}

func TestEntry_OnlyIPv4(t *testing.T) {
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")

	// Should succeed
	_, err := entry.GetIPv4Set()
	if err != nil {
		t.Errorf("Entry.GetIPv4Set() error = %v, want nil", err)
	}

	// IPv6 should fail
	_, err = entry.GetIPv6Set()
	if err == nil {
		t.Error("Entry.GetIPv6Set() on IPv4-only entry expected error, got nil")
	}
}

func TestEntry_OnlyIPv6(t *testing.T) {
	entry := NewEntry("test")
	entry.AddPrefix("2001:db8::/32")

	// Should succeed
	_, err := entry.GetIPv6Set()
	if err != nil {
		t.Errorf("Entry.GetIPv6Set() error = %v, want nil", err)
	}

	// IPv4 should fail
	_, err = entry.GetIPv4Set()
	if err == nil {
		t.Error("Entry.GetIPv4Set() on IPv6-only entry expected error, got nil")
	}
}

func TestEntry_RemoveFromEmptyBuilder(t *testing.T) {
	entry := NewEntry("test")

	// Remove from empty should not error
	err := entry.RemovePrefix("192.168.1.0/24")
	if err != nil {
		t.Errorf("Entry.RemovePrefix() on empty entry error = %v, want nil", err)
	}
}
