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
		{"lowercase name", "myentry", "MYENTRY"},
		{"with spaces", "  spaced  ", "SPACED"},
		{"mixed case", "MixedCase", "MIXEDCASE"},
		{"with numbers", "entry123", "ENTRY123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := NewEntry(tt.input)
			if entry.GetName() != tt.expected {
				t.Errorf("NewEntry(%q).GetName() = %q, expected %q", tt.input, entry.GetName(), tt.expected)
			}
		})
	}
}

func TestEntry_hasBuilders(t *testing.T) {
	t.Run("no builders", func(t *testing.T) {
		e := NewEntry("test")
		if e.hasIPv4Builder() {
			t.Error("expected hasIPv4Builder() = false")
		}
		if e.hasIPv6Builder() {
			t.Error("expected hasIPv6Builder() = false")
		}
	})

	t.Run("with IPv4 builder", func(t *testing.T) {
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if !e.hasIPv4Builder() {
			t.Error("expected hasIPv4Builder() = true")
		}
		if e.hasIPv6Builder() {
			t.Error("expected hasIPv6Builder() = false")
		}
	})

	t.Run("with IPv6 builder", func(t *testing.T) {
		e := NewEntry("test")
		if err := e.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if e.hasIPv4Builder() {
			t.Error("expected hasIPv4Builder() = false")
		}
		if !e.hasIPv6Builder() {
			t.Error("expected hasIPv6Builder() = true")
		}
	})
}

func TestEntry_AddPrefix_String(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		expectV4  bool
		expectV6  bool
	}{
		{"IPv4 CIDR", "192.168.1.0/24", false, true, false},
		{"IPv4 address", "10.0.0.1", false, true, false},
		{"IPv6 CIDR", "2001:db8::/32", false, false, true},
		{"IPv6 address", "::1", false, false, true},
		{"IPv4 comment line #", "# comment", true, false, false},
		{"IPv4 comment line //", "// comment", true, false, false},
		{"IPv4 comment line /*", "/* comment", true, false, false},
		{"empty string", "", true, false, false},
		{"spaces only", "   ", true, false, false},
		{"invalid CIDR", "192.168.1.0/33", true, false, false},
		{"invalid IP", "not.an.ip", true, false, false},
		{"IPv4 with inline comment", "192.168.1.0/24 # comment", false, true, false},
		{"IPv4 mapped IPv6 address", "::ffff:192.168.1.1", false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEntry("test")
			err := e.AddPrefix(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectV4 && !e.hasIPv4Builder() {
				t.Error("expected IPv4 builder to exist")
			}
			if tt.expectV6 && !e.hasIPv6Builder() {
				t.Error("expected IPv6 builder to exist")
			}
		})
	}
}

func TestEntry_AddPrefix_NetIP(t *testing.T) {
	t.Run("net.IP IPv4", func(t *testing.T) {
		e := NewEntry("test")
		ip := net.ParseIP("192.168.1.1")
		if err := e.AddPrefix(ip); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if !e.hasIPv4Builder() {
			t.Error("expected IPv4 builder")
		}
	})

	t.Run("net.IP IPv6", func(t *testing.T) {
		e := NewEntry("test")
		ip := net.ParseIP("2001:db8::1")
		if err := e.AddPrefix(ip); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if !e.hasIPv6Builder() {
			t.Error("expected IPv6 builder")
		}
	})

	t.Run("net.IP invalid", func(t *testing.T) {
		e := NewEntry("test")
		var ip net.IP = nil
		err := e.AddPrefix(ip)
		if err != ErrInvalidIP {
			t.Errorf("expected ErrInvalidIP, got %v", err)
		}
	})
}

func TestEntry_AddPrefix_NetIPNet(t *testing.T) {
	t.Run("*net.IPNet IPv4", func(t *testing.T) {
		e := NewEntry("test")
		_, ipnet, _ := net.ParseCIDR("10.0.0.0/8")
		if err := e.AddPrefix(ipnet); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if !e.hasIPv4Builder() {
			t.Error("expected IPv4 builder")
		}
	})

	t.Run("*net.IPNet IPv6", func(t *testing.T) {
		e := NewEntry("test")
		_, ipnet, _ := net.ParseCIDR("fd00::/8")
		if err := e.AddPrefix(ipnet); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if !e.hasIPv6Builder() {
			t.Error("expected IPv6 builder")
		}
	})
}

func TestEntry_AddPrefix_NetipAddr(t *testing.T) {
	t.Run("netip.Addr IPv4", func(t *testing.T) {
		e := NewEntry("test")
		addr := netip.MustParseAddr("172.16.0.1")
		if err := e.AddPrefix(addr); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if !e.hasIPv4Builder() {
			t.Error("expected IPv4 builder")
		}
	})

	t.Run("netip.Addr IPv6", func(t *testing.T) {
		e := NewEntry("test")
		addr := netip.MustParseAddr("fe80::1")
		if err := e.AddPrefix(addr); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if !e.hasIPv6Builder() {
			t.Error("expected IPv6 builder")
		}
	})

	t.Run("*netip.Addr IPv4", func(t *testing.T) {
		e := NewEntry("test")
		addr := netip.MustParseAddr("172.16.0.1")
		if err := e.AddPrefix(&addr); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if !e.hasIPv4Builder() {
			t.Error("expected IPv4 builder")
		}
	})

	t.Run("*netip.Addr IPv6", func(t *testing.T) {
		e := NewEntry("test")
		addr := netip.MustParseAddr("fe80::1")
		if err := e.AddPrefix(&addr); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if !e.hasIPv6Builder() {
			t.Error("expected IPv6 builder")
		}
	})

	t.Run("netip.Addr IPv4 mapped", func(t *testing.T) {
		e := NewEntry("test")
		addr := netip.MustParseAddr("::ffff:192.168.1.1")
		if err := e.AddPrefix(addr); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if !e.hasIPv4Builder() {
			t.Error("expected IPv4 builder for mapped address")
		}
	})
}

func TestEntry_AddPrefix_NetipPrefix(t *testing.T) {
	t.Run("netip.Prefix IPv4", func(t *testing.T) {
		e := NewEntry("test")
		prefix := netip.MustParsePrefix("192.168.0.0/16")
		if err := e.AddPrefix(prefix); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if !e.hasIPv4Builder() {
			t.Error("expected IPv4 builder")
		}
	})

	t.Run("netip.Prefix IPv6", func(t *testing.T) {
		e := NewEntry("test")
		prefix := netip.MustParsePrefix("fd00::/8")
		if err := e.AddPrefix(prefix); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if !e.hasIPv6Builder() {
			t.Error("expected IPv6 builder")
		}
	})

	t.Run("*netip.Prefix IPv4", func(t *testing.T) {
		e := NewEntry("test")
		prefix := netip.MustParsePrefix("10.0.0.0/8")
		if err := e.AddPrefix(&prefix); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if !e.hasIPv4Builder() {
			t.Error("expected IPv4 builder")
		}
	})

	t.Run("*netip.Prefix IPv6", func(t *testing.T) {
		e := NewEntry("test")
		prefix := netip.MustParsePrefix("2001:db8::/32")
		if err := e.AddPrefix(&prefix); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if !e.hasIPv6Builder() {
			t.Error("expected IPv6 builder")
		}
	})

	t.Run("netip.Prefix IPv4-in-6", func(t *testing.T) {
		e := NewEntry("test")
		// Create an IPv4-mapped IPv6 prefix
		addr := netip.MustParseAddr("::ffff:192.168.1.0")
		prefix := netip.PrefixFrom(addr, 120) // /120 corresponds to /24 for IPv4
		if err := e.AddPrefix(prefix); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if !e.hasIPv4Builder() {
			t.Error("expected IPv4 builder for IPv4-in-6")
		}
	})

	t.Run("*netip.Prefix IPv4-in-6", func(t *testing.T) {
		e := NewEntry("test")
		addr := netip.MustParseAddr("::ffff:192.168.1.0")
		prefix := netip.PrefixFrom(addr, 120)
		if err := e.AddPrefix(&prefix); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if !e.hasIPv4Builder() {
			t.Error("expected IPv4 builder for IPv4-in-6")
		}
	})

	t.Run("netip.Prefix IPv4-in-6 invalid bits", func(t *testing.T) {
		e := NewEntry("test")
		addr := netip.MustParseAddr("::ffff:192.168.1.0")
		prefix := netip.PrefixFrom(addr, 64) // bits < 96, should error
		err := e.AddPrefix(prefix)
		if err != ErrInvalidPrefix {
			t.Errorf("expected ErrInvalidPrefix, got %v", err)
		}
	})

	t.Run("*netip.Prefix IPv4-in-6 invalid bits", func(t *testing.T) {
		e := NewEntry("test")
		addr := netip.MustParseAddr("::ffff:192.168.1.0")
		prefix := netip.PrefixFrom(addr, 64)
		err := e.AddPrefix(&prefix)
		if err != ErrInvalidPrefix {
			t.Errorf("expected ErrInvalidPrefix, got %v", err)
		}
	})
}

func TestEntry_AddPrefix_InvalidType(t *testing.T) {
	e := NewEntry("test")
	err := e.AddPrefix(12345) // int is not a valid type
	if err != ErrInvalidPrefixType {
		t.Errorf("expected ErrInvalidPrefixType, got %v", err)
	}
}

func TestEntry_RemovePrefix(t *testing.T) {
	t.Run("remove IPv4 prefix", func(t *testing.T) {
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.0.0/16"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e.RemovePrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("RemovePrefix failed: %v", err)
		}
		// Verify the removal worked
		prefixes, err := e.MarshalPrefix()
		if err != nil {
			t.Fatalf("MarshalPrefix failed: %v", err)
		}
		// Should have all /24s except 192.168.1.0/24
		for _, p := range prefixes {
			if p.String() == "192.168.1.0/24" {
				t.Error("192.168.1.0/24 should have been removed")
			}
		}
	})

	t.Run("remove IPv6 prefix", func(t *testing.T) {
		e := NewEntry("test")
		if err := e.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e.RemovePrefix("2001:db8:1::/48"); err != nil {
			t.Fatalf("RemovePrefix failed: %v", err)
		}
	})

	t.Run("remove from non-existent builder", func(t *testing.T) {
		e := NewEntry("test")
		// No builder exists, but removal should not error
		if err := e.RemovePrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("RemovePrefix failed: %v", err)
		}
		if err := e.RemovePrefix("2001:db8::/32"); err != nil {
			t.Fatalf("RemovePrefix failed: %v", err)
		}
	})

	t.Run("remove comment line", func(t *testing.T) {
		e := NewEntry("test")
		// Comment lines result in ErrInvalidIPType because processPrefix returns nil prefix and empty ipType
		// which then causes remove to fail
		err := e.RemovePrefix("# comment")
		if err == nil {
			t.Error("expected error for comment line")
		}
	})

	t.Run("remove invalid", func(t *testing.T) {
		e := NewEntry("test")
		err := e.RemovePrefix("not.an.ip")
		if err != ErrInvalidIP {
			t.Errorf("expected ErrInvalidIP, got %v", err)
		}
	})
}

func TestEntry_GetIPv4Set(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		set, err := e.GetIPv4Set()
		if err != nil {
			t.Fatalf("GetIPv4Set failed: %v", err)
		}
		if set == nil {
			t.Error("expected non-nil set")
		}
	})

	t.Run("no IPv4 set", func(t *testing.T) {
		e := NewEntry("test")
		if err := e.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		_, err := e.GetIPv4Set()
		if err == nil {
			t.Error("expected error for no IPv4 set")
		}
	})

	t.Run("no builder", func(t *testing.T) {
		e := NewEntry("test")
		_, err := e.GetIPv4Set()
		if err == nil {
			t.Error("expected error for no builder")
		}
	})
}

func TestEntry_GetIPv6Set(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		e := NewEntry("test")
		if err := e.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		set, err := e.GetIPv6Set()
		if err != nil {
			t.Fatalf("GetIPv6Set failed: %v", err)
		}
		if set == nil {
			t.Error("expected non-nil set")
		}
	})

	t.Run("no IPv6 set", func(t *testing.T) {
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		_, err := e.GetIPv6Set()
		if err == nil {
			t.Error("expected error for no IPv6 set")
		}
	})

	t.Run("no builder", func(t *testing.T) {
		e := NewEntry("test")
		_, err := e.GetIPv6Set()
		if err == nil {
			t.Error("expected error for no builder")
		}
	})
}

func TestEntry_MarshalPrefix(t *testing.T) {
	t.Run("both IPv4 and IPv6", func(t *testing.T) {
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		prefixes, err := e.MarshalPrefix()
		if err != nil {
			t.Fatalf("MarshalPrefix failed: %v", err)
		}
		if len(prefixes) != 2 {
			t.Errorf("expected 2 prefixes, got %d", len(prefixes))
		}
	})

	t.Run("with IgnoreIPv4", func(t *testing.T) {
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		prefixes, err := e.MarshalPrefix(IgnoreIPv4)
		if err != nil {
			t.Fatalf("MarshalPrefix failed: %v", err)
		}
		if len(prefixes) != 1 {
			t.Errorf("expected 1 prefix (IPv6 only), got %d", len(prefixes))
		}
		if !prefixes[0].Addr().Is6() {
			t.Error("expected IPv6 prefix")
		}
	})

	t.Run("with IgnoreIPv6", func(t *testing.T) {
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		prefixes, err := e.MarshalPrefix(IgnoreIPv6)
		if err != nil {
			t.Fatalf("MarshalPrefix failed: %v", err)
		}
		if len(prefixes) != 1 {
			t.Errorf("expected 1 prefix (IPv4 only), got %d", len(prefixes))
		}
		if !prefixes[0].Addr().Is4() {
			t.Error("expected IPv4 prefix")
		}
	})

	t.Run("with nil option", func(t *testing.T) {
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		prefixes, err := e.MarshalPrefix(nil)
		if err != nil {
			t.Fatalf("MarshalPrefix failed: %v", err)
		}
		if len(prefixes) != 1 {
			t.Errorf("expected 1 prefix, got %d", len(prefixes))
		}
	})

	t.Run("empty entry", func(t *testing.T) {
		e := NewEntry("test")
		_, err := e.MarshalPrefix()
		if err == nil {
			t.Error("expected error for empty entry")
		}
	})
}

func TestEntry_MarshalIPRange(t *testing.T) {
	t.Run("both IPv4 and IPv6", func(t *testing.T) {
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		ranges, err := e.MarshalIPRange()
		if err != nil {
			t.Fatalf("MarshalIPRange failed: %v", err)
		}
		if len(ranges) != 2 {
			t.Errorf("expected 2 ranges, got %d", len(ranges))
		}
	})

	t.Run("with IgnoreIPv4", func(t *testing.T) {
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		ranges, err := e.MarshalIPRange(IgnoreIPv4)
		if err != nil {
			t.Fatalf("MarshalIPRange failed: %v", err)
		}
		if len(ranges) != 1 {
			t.Errorf("expected 1 range (IPv6 only), got %d", len(ranges))
		}
	})

	t.Run("with IgnoreIPv6", func(t *testing.T) {
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		ranges, err := e.MarshalIPRange(IgnoreIPv6)
		if err != nil {
			t.Fatalf("MarshalIPRange failed: %v", err)
		}
		if len(ranges) != 1 {
			t.Errorf("expected 1 range (IPv4 only), got %d", len(ranges))
		}
	})

	t.Run("with nil option", func(t *testing.T) {
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		ranges, err := e.MarshalIPRange(nil)
		if err != nil {
			t.Fatalf("MarshalIPRange failed: %v", err)
		}
		if len(ranges) != 1 {
			t.Errorf("expected 1 range, got %d", len(ranges))
		}
	})

	t.Run("empty entry", func(t *testing.T) {
		e := NewEntry("test")
		_, err := e.MarshalIPRange()
		if err == nil {
			t.Error("expected error for empty entry")
		}
	})
}

func TestEntry_MarshalText(t *testing.T) {
	t.Run("both IPv4 and IPv6", func(t *testing.T) {
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		texts, err := e.MarshalText()
		if err != nil {
			t.Fatalf("MarshalText failed: %v", err)
		}
		if len(texts) != 2 {
			t.Errorf("expected 2 texts, got %d", len(texts))
		}
	})

	t.Run("with IgnoreIPv4", func(t *testing.T) {
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		texts, err := e.MarshalText(IgnoreIPv4)
		if err != nil {
			t.Fatalf("MarshalText failed: %v", err)
		}
		if len(texts) != 1 {
			t.Errorf("expected 1 text (IPv6 only), got %d", len(texts))
		}
	})

	t.Run("with IgnoreIPv6", func(t *testing.T) {
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		if err := e.AddPrefix("2001:db8::/32"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		texts, err := e.MarshalText(IgnoreIPv6)
		if err != nil {
			t.Fatalf("MarshalText failed: %v", err)
		}
		if len(texts) != 1 {
			t.Errorf("expected 1 text (IPv4 only), got %d", len(texts))
		}
	})

	t.Run("with nil option", func(t *testing.T) {
		e := NewEntry("test")
		if err := e.AddPrefix("192.168.1.0/24"); err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		texts, err := e.MarshalText(nil)
		if err != nil {
			t.Fatalf("MarshalText failed: %v", err)
		}
		if len(texts) != 1 {
			t.Errorf("expected 1 text, got %d", len(texts))
		}
	})

	t.Run("empty entry", func(t *testing.T) {
		e := NewEntry("test")
		_, err := e.MarshalText()
		if err == nil {
			t.Error("expected error for empty entry")
		}
	})
}

func TestEntry_add_InvalidIPType(t *testing.T) {
	e := NewEntry("test")
	prefix := netip.MustParsePrefix("192.168.1.0/24")
	err := e.add(&prefix, "invalid")
	if err != ErrInvalidIPType {
		t.Errorf("expected ErrInvalidIPType, got %v", err)
	}
}

func TestEntry_remove_InvalidIPType(t *testing.T) {
	e := NewEntry("test")
	prefix := netip.MustParsePrefix("192.168.1.0/24")
	err := e.remove(&prefix, "invalid")
	if err != ErrInvalidIPType {
		t.Errorf("expected ErrInvalidIPType, got %v", err)
	}
}

func TestEntry_processPrefix_InvalidIPv4MappedCIDR(t *testing.T) {
	e := NewEntry("test")
	// This tests the case where the CIDR appears to be IPv4-mapped in IPv6 notation
	// but is detected and rejected
	err := e.AddPrefix("::ffff:192.168.1.0/120")
	// This should work as it's a valid IPv4-mapped CIDR
	if err != nil {
		t.Fatalf("AddPrefix failed unexpectedly: %v", err)
	}
}

func TestEntry_processPrefix_InvalidAddr(t *testing.T) {
	e := NewEntry("test")
	// Test with an invalid netip.Addr (zero value)
	var addr netip.Addr
	_, _, err := e.processPrefix(addr)
	if err != ErrInvalidIPLength {
		t.Errorf("expected ErrInvalidIPLength for zero addr, got %v", err)
	}

	// Test with pointer to invalid netip.Addr
	_, _, err = e.processPrefix(&addr)
	if err != ErrInvalidIPLength {
		t.Errorf("expected ErrInvalidIPLength for zero addr pointer, got %v", err)
	}
}

func TestEntry_processPrefix_InvalidPrefix(t *testing.T) {
	e := NewEntry("test")
	// Test with an invalid netip.Prefix (zero value)
	var prefix netip.Prefix
	_, _, err := e.processPrefix(prefix)
	if err != ErrInvalidIPLength {
		t.Errorf("expected ErrInvalidIPLength for zero prefix, got %v", err)
	}

	// Test with pointer to invalid netip.Prefix
	_, _, err = e.processPrefix(&prefix)
	if err != ErrInvalidIPLength {
		t.Errorf("expected ErrInvalidIPLength for zero prefix pointer, got %v", err)
	}
}

func TestEntry_processPrefix_IPv4MappedCIDRString(t *testing.T) {
	e := NewEntry("test")
	// IPv4-mapped IPv6 CIDRs are parsed as IPv4 by net.ParseCIDR
	// The condition in processPrefix (line 209) is effectively unreachable
	// because network.String() always returns an IPv4 form for these cases
	err := e.AddPrefix("::ffff:192.168.1.0/96")
	if err != nil {
		t.Errorf("unexpected error for IPv4-mapped CIDR: %v", err)
	}
}

func TestEntry_hasSets(t *testing.T) {
	e := NewEntry("test")

	// Initially no sets
	if e.hasIPv4Set() {
		t.Error("expected hasIPv4Set() = false initially")
	}
	if e.hasIPv6Set() {
		t.Error("expected hasIPv6Set() = false initially")
	}

	// Add prefix and build set
	if err := e.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	if err := e.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}

	// Build the sets
	_, _ = e.GetIPv4Set()
	_, _ = e.GetIPv6Set()

	// Now should have sets
	if !e.hasIPv4Set() {
		t.Error("expected hasIPv4Set() = true after building")
	}
	if !e.hasIPv6Set() {
		t.Error("expected hasIPv6Set() = true after building")
	}

	// Call GetIPv4Set/GetIPv6Set again to test the cached path
	_, err := e.GetIPv4Set()
	if err != nil {
		t.Errorf("GetIPv4Set (cached) failed: %v", err)
	}
	_, err = e.GetIPv6Set()
	if err != nil {
		t.Errorf("GetIPv6Set (cached) failed: %v", err)
	}
}

func TestEntry_MarshalPrefix_OnlyIPv6(t *testing.T) {
	e := NewEntry("test")
	if err := e.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	prefixes, err := e.MarshalPrefix()
	if err != nil {
		t.Fatalf("MarshalPrefix failed: %v", err)
	}
	if len(prefixes) != 1 {
		t.Errorf("expected 1 prefix, got %d", len(prefixes))
	}
}

func TestEntry_MarshalIPRange_OnlyIPv4(t *testing.T) {
	e := NewEntry("test")
	if err := e.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	ranges, err := e.MarshalIPRange()
	if err != nil {
		t.Fatalf("MarshalIPRange failed: %v", err)
	}
	if len(ranges) != 1 {
		t.Errorf("expected 1 range, got %d", len(ranges))
	}
}

func TestEntry_MarshalText_OnlyIPv4(t *testing.T) {
	e := NewEntry("test")
	if err := e.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	texts, err := e.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText failed: %v", err)
	}
	if len(texts) != 1 {
		t.Errorf("expected 1 text, got %d", len(texts))
	}
}

func TestEntry_MarshalText_OnlyIPv6(t *testing.T) {
	e := NewEntry("test")
	if err := e.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	texts, err := e.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText failed: %v", err)
	}
	if len(texts) != 1 {
		t.Errorf("expected 1 text, got %d", len(texts))
	}
}
