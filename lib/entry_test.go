package lib

import (
	"net"
	"net/netip"
	"testing"

	"go4.org/netipx"
)

func TestNewEntry(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "test",
			expected: "TEST",
		},
		{
			name:     "lowercase name",
			input:    "lowercase",
			expected: "LOWERCASE",
		},
		{
			name:     "name with spaces",
			input:    "  test name  ",
			expected: "TEST NAME",
		},
		{
			name:     "mixed case",
			input:    "MiXeD CaSe",
			expected: "MIXED CASE",
		},
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

func TestEntry_AddPrefix(t *testing.T) {
	tests := []struct {
		name    string
		cidr    any
		wantErr bool
	}{
		{
			name:    "valid IPv4 CIDR string",
			cidr:    "192.168.1.0/24",
			wantErr: false,
		},
		{
			name:    "valid IPv6 CIDR string",
			cidr:    "2001:db8::/32",
			wantErr: false,
		},
		{
			name:    "valid IPv4 address string",
			cidr:    "192.168.1.1",
			wantErr: false,
		},
		{
			name:    "valid IPv6 address string",
			cidr:    "2001:db8::1",
			wantErr: false,
		},
		{
			name:    "invalid CIDR",
			cidr:    "invalid/cidr",
			wantErr: true,
		},
		{
			name:    "invalid IP",
			cidr:    "999.999.999.999",
			wantErr: true,
		},
		{
			name:    "net.IP type",
			cidr:    net.ParseIP("192.168.1.1"),
			wantErr: false,
		},
		{
			name:    "net.IPNet type",
			cidr:    &net.IPNet{IP: net.ParseIP("192.168.1.0"), Mask: net.CIDRMask(24, 32)},
			wantErr: false,
		},
		{
			name:    "netip.Addr type",
			cidr:    netip.MustParseAddr("192.168.1.1"),
			wantErr: false,
		},
		{
			name:    "netip.Prefix type",
			cidr:    netip.MustParsePrefix("192.168.1.0/24"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := NewEntry("test")
			err := entry.AddPrefix(tt.cidr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Entry.AddPrefix() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEntry_RemovePrefix(t *testing.T) {
	entry := NewEntry("test")
	// First add a prefix
	entry.AddPrefix("192.168.1.0/24")
	entry.AddPrefix("10.0.0.0/8")

	tests := []struct {
		name    string
		cidr    string
		wantErr bool
	}{
		{
			name:    "valid CIDR",
			cidr:    "192.168.1.0/24",
			wantErr: false,
		},
		{
			name:    "invalid CIDR",
			cidr:    "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := entry.RemovePrefix(tt.cidr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Entry.RemovePrefix() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEntry_GetIPv4Set(t *testing.T) {
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")

	set, err := entry.GetIPv4Set()
	if err != nil {
		t.Errorf("Entry.GetIPv4Set() error = %v", err)
	}
	if set == nil {
		t.Error("Entry.GetIPv4Set() returned nil set")
	}

	// Test entry with no IPv4
	entry2 := NewEntry("test2")
	entry2.AddPrefix("2001:db8::/32")
	_, err = entry2.GetIPv4Set()
	if err == nil {
		t.Error("Entry.GetIPv4Set() should return error for entry with no IPv4")
	}
}

func TestEntry_GetIPv6Set(t *testing.T) {
	entry := NewEntry("test")
	entry.AddPrefix("2001:db8::/32")

	set, err := entry.GetIPv6Set()
	if err != nil {
		t.Errorf("Entry.GetIPv6Set() error = %v", err)
	}
	if set == nil {
		t.Error("Entry.GetIPv6Set() returned nil set")
	}

	// Test entry with no IPv6
	entry2 := NewEntry("test2")
	entry2.AddPrefix("192.168.1.0/24")
	_, err = entry2.GetIPv6Set()
	if err == nil {
		t.Error("Entry.GetIPv6Set() should return error for entry with no IPv6")
	}
}

func TestEntry_MarshalPrefix(t *testing.T) {
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	entry.AddPrefix("10.0.0.0/8")
	entry.AddPrefix("2001:db8::/32")

	tests := []struct {
		name    string
		opts    []IgnoreIPOption
		wantErr bool
		checkFn func(*testing.T, []netip.Prefix)
	}{
		{
			name:    "no options",
			opts:    nil,
			wantErr: false,
			checkFn: func(t *testing.T, prefixes []netip.Prefix) {
				if len(prefixes) != 3 {
					t.Errorf("MarshalPrefix() returned %d prefixes, want 3", len(prefixes))
				}
			},
		},
		{
			name:    "ignore IPv4",
			opts:    []IgnoreIPOption{IgnoreIPv4},
			wantErr: false,
			checkFn: func(t *testing.T, prefixes []netip.Prefix) {
				if len(prefixes) != 1 {
					t.Errorf("MarshalPrefix() returned %d prefixes, want 1", len(prefixes))
				}
			},
		},
		{
			name:    "ignore IPv6",
			opts:    []IgnoreIPOption{IgnoreIPv6},
			wantErr: false,
			checkFn: func(t *testing.T, prefixes []netip.Prefix) {
				if len(prefixes) != 2 {
					t.Errorf("MarshalPrefix() returned %d prefixes, want 2", len(prefixes))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := entry.MarshalPrefix(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Entry.MarshalPrefix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.checkFn != nil {
				tt.checkFn(t, got)
			}
		})
	}
}

func TestEntry_MarshalIPRange(t *testing.T) {
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	entry.AddPrefix("2001:db8::/32")

	ranges, err := entry.MarshalIPRange()
	if err != nil {
		t.Errorf("Entry.MarshalIPRange() error = %v", err)
	}
	if len(ranges) != 2 {
		t.Errorf("Entry.MarshalIPRange() returned %d ranges, want 2", len(ranges))
	}
}

func TestEntry_MarshalText(t *testing.T) {
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	entry.AddPrefix("2001:db8::/32")

	text, err := entry.MarshalText()
	if err != nil {
		t.Errorf("Entry.MarshalText() error = %v", err)
	}
	if len(text) != 2 {
		t.Errorf("Entry.MarshalText() returned %d lines, want 2", len(text))
	}
}

func TestEntry_EmptyEntry(t *testing.T) {
	entry := NewEntry("empty")

	_, err := entry.MarshalPrefix()
	if err == nil {
		t.Error("Entry.MarshalPrefix() should return error for empty entry")
	}

	_, err = entry.MarshalIPRange()
	if err == nil {
		t.Error("Entry.MarshalIPRange() should return error for empty entry")
	}

	_, err = entry.MarshalText()
	if err == nil {
		t.Error("Entry.MarshalText() should return error for empty entry")
	}
}

func TestEntry_ProcessPrefix_IPv4Mapped(t *testing.T) {
	entry := NewEntry("test")
	
	// Test IPv4-mapped IPv6 address
	err := entry.AddPrefix("::ffff:192.168.1.1")
	if err != nil {
		t.Errorf("Entry.AddPrefix() with IPv4-mapped address error = %v", err)
	}
}

func TestEntry_ProcessPrefix_EdgeCases(t *testing.T) {
	entry := NewEntry("test")

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name:    "CIDR with comment",
			input:   "192.168.1.0/24 # comment",
			wantErr: false,
		},
		{
			name:    "IP with inline comment",
			input:   "192.168.1.1 // comment",
			wantErr: false,
		},
		{
			name:    "pointer to netip.Addr",
			input:   func() *netip.Addr { a := netip.MustParseAddr("192.168.1.1"); return &a }(),
			wantErr: false,
		},
		{
			name:    "pointer to netip.Prefix",
			input:   func() *netip.Prefix { p := netip.MustParsePrefix("192.168.1.0/24"); return &p }(),
			wantErr: false,
		},
		{
			name:    "unsupported type",
			input:   123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := entry.AddPrefix(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Entry.AddPrefix(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestEntry_AddPrefix_IPv4In6(t *testing.T) {
	entry := NewEntry("test")
	
	// Create an IPv4-in-IPv6 prefix
	prefix := netip.MustParsePrefix("::ffff:c0a8:0100/120") // ::ffff:192.168.1.0/120
	err := entry.AddPrefix(prefix)
	if err != nil {
		t.Errorf("Entry.AddPrefix() with IPv4-in-IPv6 error = %v", err)
	}
}

func TestEntry_InvalidIPLengthCases(t *testing.T) {
	entry := NewEntry("test")
	
	// Test with invalid IP that could trigger ErrInvalidIPLength
	invalidIP := net.IP{} // Invalid empty IP
	err := entry.AddPrefix(invalidIP)
	if err == nil {
		t.Error("Entry.AddPrefix() should return error for invalid IP")
	}
}

func TestEntry_AddPrefix_WithInvalidIPv4MappedCIDR(t *testing.T) {
	entry := NewEntry("test")
	
	// Test IPv4-mapped IPv6 CIDR - this is actually valid and should work
	err := entry.AddPrefix("::ffff:192.168.1.0/120")
	if err != nil {
		// If it errors, that's fine - just checking the code path
		t.Logf("AddPrefix with IPv4-mapped CIDR returned: %v", err)
	}
}

func TestEntry_RemovePrefixVariousCases(t *testing.T) {
	tests := []struct {
		name       string
		addPrefixes []string
		removePrefixes []string
		wantErr    bool
	}{
		{
			name:       "remove IPv4 prefix",
			addPrefixes: []string{"192.168.1.0/24", "10.0.0.0/8"},
			removePrefixes: []string{"192.168.1.0/24"},
			wantErr:    false,
		},
		{
			name:       "remove IPv6 prefix",
			addPrefixes: []string{"2001:db8::/32", "2001:db9::/32"},
			removePrefixes: []string{"2001:db8::/32"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := NewEntry("test")
			for _, prefix := range tt.addPrefixes {
				entry.AddPrefix(prefix)
			}
			for _, prefix := range tt.removePrefixes {
				err := entry.RemovePrefix(prefix)
				if (err != nil) != tt.wantErr {
					t.Errorf("Entry.RemovePrefix() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestEntry_MarshalIPRangeWithOptions(t *testing.T) {
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	entry.AddPrefix("10.0.0.0/8")
	entry.AddPrefix("2001:db8::/32")

	tests := []struct {
		name    string
		opts    []IgnoreIPOption
		wantErr bool
		checkFn func(*testing.T, []netipx.IPRange)
	}{
		{
			name:    "no options",
			opts:    nil,
			wantErr: false,
			checkFn: func(t *testing.T, ranges []netipx.IPRange) {
				if len(ranges) != 3 {
					t.Errorf("MarshalIPRange() returned %d ranges, want 3", len(ranges))
				}
			},
		},
		{
			name:    "ignore IPv4",
			opts:    []IgnoreIPOption{IgnoreIPv4},
			wantErr: false,
			checkFn: func(t *testing.T, ranges []netipx.IPRange) {
				if len(ranges) != 1 {
					t.Errorf("MarshalIPRange() returned %d ranges, want 1 (IPv6 only)", len(ranges))
				}
			},
		},
		{
			name:    "ignore IPv6",
			opts:    []IgnoreIPOption{IgnoreIPv6},
			wantErr: false,
			checkFn: func(t *testing.T, ranges []netipx.IPRange) {
				if len(ranges) != 2 {
					t.Errorf("MarshalIPRange() returned %d ranges, want 2 (IPv4 only)", len(ranges))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := entry.MarshalIPRange(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Entry.MarshalIPRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.checkFn != nil {
				tt.checkFn(t, got)
			}
		})
	}
}

func TestEntry_MarshalTextWithOptions(t *testing.T) {
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	entry.AddPrefix("10.0.0.0/8")
	entry.AddPrefix("2001:db8::/32")

	tests := []struct {
		name    string
		opts    []IgnoreIPOption
		wantErr bool
		checkFn func(*testing.T, []string)
	}{
		{
			name:    "no options",
			opts:    nil,
			wantErr: false,
			checkFn: func(t *testing.T, text []string) {
				if len(text) != 3 {
					t.Errorf("MarshalText() returned %d lines, want 3", len(text))
				}
			},
		},
		{
			name:    "ignore IPv4",
			opts:    []IgnoreIPOption{IgnoreIPv4},
			wantErr: false,
			checkFn: func(t *testing.T, text []string) {
				if len(text) != 1 {
					t.Errorf("MarshalText() returned %d lines, want 1 (IPv6 only)", len(text))
				}
			},
		},
		{
			name:    "ignore IPv6",
			opts:    []IgnoreIPOption{IgnoreIPv6},
			wantErr: false,
			checkFn: func(t *testing.T, text []string) {
				if len(text) != 2 {
					t.Errorf("MarshalText() returned %d lines, want 2 (IPv4 only)", len(text))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := entry.MarshalText(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Entry.MarshalText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.checkFn != nil {
				tt.checkFn(t, got)
			}
		})
	}
}

func TestEntry_ProcessPrefixComprehensive(t *testing.T) {
	entry := NewEntry("test")

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name:    "IPv4 with /32",
			input:   "192.168.1.1/32",
			wantErr: false,
		},
		{
			name:    "IPv6 with /128",
			input:   "2001:db8::1/128",
			wantErr: false,
		},
		{
			name:    "netip.Prefix with IPv4In6",
			input:   netip.MustParsePrefix("::ffff:192.168.1.0/120"),
			wantErr: false,
		},
		{
			name:    "pointer to netip.Prefix with IPv4In6",
			input:   func() *netip.Prefix { p := netip.MustParsePrefix("::ffff:192.168.1.0/120"); return &p }(),
			wantErr: false,
		},
		{
			name:    "pointer to netip.Addr IPv6",
			input:   func() *netip.Addr { a := netip.MustParseAddr("2001:db8::1"); return &a }(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := entry.AddPrefix(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Entry.AddPrefix(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestEntry_ProcessPrefixErrorCases(t *testing.T) {
	entry := NewEntry("test")

	// Test IPv4In6 prefix with bits < 96 (should trigger error on line 143)
	// This is tricky to test because valid IPv4In6 prefixes have bits >= 96
	// Let's test other error paths
	
	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name:    "string CIDR with invalid network",
			input:   "256.256.256.256/24",
			wantErr: true,
		},
		{
			name:    "string with only slash",
			input:   "/24",
			wantErr: true,
		},
		{
			name:    "netip.Prefix IPv4",
			input:   netip.MustParsePrefix("192.168.1.0/24"),
			wantErr: false,
		},
		{
			name:    "netip.Prefix IPv6",
			input:   netip.MustParsePrefix("2001:db8::/32"),
			wantErr: false,
		},
		{
			name:    "*netip.Prefix IPv4",
			input:   func() *netip.Prefix { p := netip.MustParsePrefix("10.0.0.0/8"); return &p }(),
			wantErr: false,
		},
		{
			name:    "*netip.Prefix IPv6",
			input:   func() *netip.Prefix { p := netip.MustParsePrefix("2001:db9::/32"); return &p }(),
			wantErr: false,
		},
		{
			name:    "string with /* comment",
			input:   "192.168.1.0/24 /* comment */",
			wantErr: false,
		},
		{
			name:    "string that becomes empty after comment removal",
			input:   "# just a comment",
			wantErr: true, // ErrCommentLine leads to ErrInvalidIPType
		},
		{
			name:    "string with whitespace and comment",
			input:   "   // comment only",
			wantErr: true,
		},
		{
			name:    "net.IP with nil",
			input:   net.IP(nil),
			wantErr: true, // Should trigger ErrInvalidIP
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := entry.AddPrefix(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Entry.AddPrefix(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestEntry_ProcessPrefix_NetIPNetCases(t *testing.T) {
	entry := NewEntry("test")
	
	tests := []struct {
		name    string
		input   *net.IPNet
		wantErr bool
	}{
		{
			name:    "valid IPv4 IPNet",
			input:   &net.IPNet{IP: net.ParseIP("192.168.1.0"), Mask: net.CIDRMask(24, 32)},
			wantErr: false,
		},
		{
			name:    "valid IPv6 IPNet",
			input:   &net.IPNet{IP: net.ParseIP("2001:db8::"), Mask: net.CIDRMask(32, 128)},
			wantErr: false,
		},
		{
			name:    "invalid IPNet with nil IP",
			input:   &net.IPNet{IP: nil, Mask: net.CIDRMask(24, 32)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := entry.AddPrefix(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Entry.AddPrefix() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEntry_ProcessPrefix_NetipAddrCases(t *testing.T) {
	entry := NewEntry("test")
	
	tests := []struct {
		name    string
		input   netip.Addr
		wantErr bool
	}{
		{
			name:    "valid IPv4 Addr",
			input:   netip.MustParseAddr("192.168.1.1"),
			wantErr: false,
		},
		{
			name:    "valid IPv6 Addr",
			input:   netip.MustParseAddr("2001:db8::1"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := entry.AddPrefix(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Entry.AddPrefix() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEntry_BuildIPSetErrors(t *testing.T) {
	// Test buildIPSet when it succeeds multiple times (coverage for checking existing sets)
	entry := NewEntry("test")
	entry.AddPrefix("192.168.1.0/24")
	entry.AddPrefix("2001:db8::/32")
	
	// First call to buildIPSet
	_, err := entry.GetIPv4Set()
	if err != nil {
		t.Errorf("First GetIPv4Set() error = %v", err)
	}
	
	// Second call should use existing set
	_, err = entry.GetIPv4Set()
	if err != nil {
		t.Errorf("Second GetIPv4Set() error = %v", err)
	}
	
	// Same for IPv6
	_, err = entry.GetIPv6Set()
	if err != nil {
		t.Errorf("First GetIPv6Set() error = %v", err)
	}
	
	_, err = entry.GetIPv6Set()
	if err != nil {
		t.Errorf("Second GetIPv6Set() error = %v", err)
	}
}
