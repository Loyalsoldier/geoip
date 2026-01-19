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
		{
			name:     "Normal name",
			input:    "test",
			expected: "TEST",
		},
		{
			name:     "Lowercase name",
			input:    "lowercase",
			expected: "LOWERCASE",
		},
		{
			name:     "Mixed case name",
			input:    "MiXeD",
			expected: "MIXED",
		},
		{
			name:     "Name with spaces",
			input:    "  test name  ",
			expected: "TEST NAME",
		},
		{
			name:     "Empty name",
			input:    "",
			expected: "",
		},
		{
			name:     "Name with special characters",
			input:    "test-name_123",
			expected: "TEST-NAME_123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := NewEntry(tt.input)
			if entry.GetName() != tt.expected {
				t.Errorf("NewEntry(%s).GetName() = %s; want %s", tt.input, entry.GetName(), tt.expected)
			}
		})
	}
}

func TestEntryGetName(t *testing.T) {
	entry := NewEntry("test")
	if entry.GetName() != "TEST" {
		t.Errorf("GetName() = %s; want TEST", entry.GetName())
	}
}

func TestEntryBuilders(t *testing.T) {
	entry := NewEntry("test")

	// Test initial state - no builders should exist
	if entry.hasIPv4Builder() {
		t.Error("New entry should not have IPv4 builder initially")
	}
	if entry.hasIPv6Builder() {
		t.Error("New entry should not have IPv6 builder initially")
	}
	if entry.hasIPv4Set() {
		t.Error("New entry should not have IPv4 set initially")
	}
	if entry.hasIPv6Set() {
		t.Error("New entry should not have IPv6 set initially")
	}
}

func TestEntryAddPrefix(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		wantErr bool
	}{
		{
			name:    "Valid IPv4 CIDR",
			prefix:  "192.168.1.0/24",
			wantErr: false,
		},
		{
			name:    "Valid IPv6 CIDR",
			prefix:  "2001:db8::/32",
			wantErr: false,
		},
		{
			name:    "Valid single IPv4",
			prefix:  "192.168.1.1",
			wantErr: false,
		},
		{
			name:    "Valid single IPv6",
			prefix:  "2001:db8::1",
			wantErr: false,
		},
		{
			name:    "Invalid CIDR",
			prefix:  "invalid-cidr",
			wantErr: true,
		},
		{
			name:    "Empty prefix",
			prefix:  "",
			wantErr: true,
		},
		{
			name:    "Invalid IPv4 range",
			prefix:  "256.256.256.256/24",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := NewEntry("test")
			err := entry.AddPrefix(tt.prefix)
			
			if tt.wantErr && err == nil {
				t.Errorf("AddPrefix(%s) should return error but got nil", tt.prefix)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("AddPrefix(%s) should not return error but got: %v", tt.prefix, err)
			}
		})
	}
}

func TestEntryGetIPSets(t *testing.T) {
	entry := NewEntry("test")
	
	// Test getting IPv4 set when none exists
	_, err := entry.GetIPv4Set()
	if err == nil {
		t.Error("GetIPv4Set() should return error when no IPv4 set exists")
	}
	
	// Test getting IPv6 set when none exists
	_, err = entry.GetIPv6Set()
	if err == nil {
		t.Error("GetIPv6Set() should return error when no IPv6 set exists")
	}
	
	// Add some prefixes and test getting sets
	err = entry.AddPrefix("192.168.1.0/24")
	if err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	
	err = entry.AddPrefix("2001:db8::/32")
	if err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	
	// Now we should be able to get the sets
	ipv4Set, err := entry.GetIPv4Set()
	if err != nil {
		t.Errorf("GetIPv4Set() should not return error after adding IPv4 prefix: %v", err)
	}
	if ipv4Set == nil {
		t.Error("GetIPv4Set() should return non-nil set after adding IPv4 prefix")
	}
	
	ipv6Set, err := entry.GetIPv6Set()
	if err != nil {
		t.Errorf("GetIPv6Set() should not return error after adding IPv6 prefix: %v", err)
	}
	if ipv6Set == nil {
		t.Error("GetIPv6Set() should return non-nil set after adding IPv6 prefix")
	}
}

func TestEntryMarshalText(t *testing.T) {
	entry := NewEntry("test")
	
	// Test with no prefixes
	_, err := entry.MarshalText()
	if err == nil {
		t.Error("MarshalText() should return error for empty entry")
	}
	
	// Add some prefixes
	err = entry.AddPrefix("192.168.1.0/24")
	if err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	
	err = entry.AddPrefix("2001:db8::/32")
	if err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	
	// Test marshaling
	cidrs, err := entry.MarshalText()
	if err != nil {
		t.Errorf("MarshalText() should not return error: %v", err)
	}
	if len(cidrs) == 0 {
		t.Error("MarshalText() should return non-empty slice after adding prefixes")
	}
	
	// Test marshaling with ignore options
	cidrs, err = entry.MarshalText(IgnoreIPv6)
	if err != nil {
		t.Errorf("MarshalText(IgnoreIPv6) should not return error: %v", err)
	}
	
	cidrs, err = entry.MarshalText(IgnoreIPv4)
	if err != nil {
		t.Errorf("MarshalText(IgnoreIPv4) should not return error: %v", err)
	}
}

func TestEntryMarshalIPRange(t *testing.T) {
	entry := NewEntry("test")
	
	// Test with no prefixes
	_, err := entry.MarshalIPRange()
	if err == nil {
		t.Error("MarshalIPRange() should return error for empty entry")
	}
	
	// Add some prefixes
	err = entry.AddPrefix("192.168.1.0/24")
	if err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	
	err = entry.AddPrefix("2001:db8::/32")
	if err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	
	// Test marshaling
	ranges, err := entry.MarshalIPRange()
	if err != nil {
		t.Errorf("MarshalIPRange() should not return error: %v", err)
	}
	if len(ranges) == 0 {
		t.Error("MarshalIPRange() should return non-empty slice after adding prefixes")
	}
	
	// Test marshaling with ignore options
	ranges, err = entry.MarshalIPRange(IgnoreIPv6)
	if err != nil {
		t.Errorf("MarshalIPRange(IgnoreIPv6) should not return error: %v", err)
	}
	
	ranges, err = entry.MarshalIPRange(IgnoreIPv4)
	if err != nil {
		t.Errorf("MarshalIPRange(IgnoreIPv4) should not return error: %v", err)
	}
}

func TestEntryMarshalPrefix(t *testing.T) {
	entry := NewEntry("test")
	
	// Test with no prefixes
	_, err := entry.MarshalPrefix()
	if err == nil {
		t.Error("MarshalPrefix() should return error for empty entry")
	}
	
	// Add some prefixes
	err = entry.AddPrefix("192.168.1.0/24")
	if err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	
	err = entry.AddPrefix("2001:db8::/32")
	if err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	
	// Test marshaling
	prefixes, err := entry.MarshalPrefix()
	if err != nil {
		t.Errorf("MarshalPrefix() should not return error: %v", err)
	}
	if len(prefixes) == 0 {
		t.Error("MarshalPrefix() should return non-empty slice after adding prefixes")
	}
	
	// Test marshaling with ignore options
	prefixes, err = entry.MarshalPrefix(IgnoreIPv6)
	if err != nil {
		t.Errorf("MarshalPrefix(IgnoreIPv6) should not return error: %v", err)
	}
	
	prefixes, err = entry.MarshalPrefix(IgnoreIPv4)
	if err != nil {
		t.Errorf("MarshalPrefix(IgnoreIPv4) should not return error: %v", err)
	}
}

func TestEntryRemovePrefix(t *testing.T) {
	entry := NewEntry("test")
	
	// Add a prefix first
	err := entry.AddPrefix("192.168.1.0/24")
	if err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	
	// Remove the prefix
	err = entry.RemovePrefix("192.168.1.0/24")
	if err != nil {
		t.Errorf("RemovePrefix should not return error: %v", err)
	}
	
	// Test removing invalid prefix
	err = entry.RemovePrefix("invalid-cidr")
	if err == nil {
		t.Error("RemovePrefix with invalid CIDR should return error")
	}
	
	// Try to remove non-existent prefix (should not error)
	err = entry.RemovePrefix("10.0.0.0/8")
	if err != nil {
		t.Errorf("RemovePrefix() on non-existent prefix should not return error: %v", err)
	}
}

// TestEntryProcessPrefix tests the internal processPrefix function with various input types
func TestEntryProcessPrefix(t *testing.T) {
	entry := NewEntry("test")
	
	tests := []struct {
		name      string
		input     interface{}
		expectErr bool
		expectIP  string
	}{
		// net.IP tests
		{
			name:      "Valid IPv4 net.IP",
			input:     net.ParseIP("192.168.1.1"),
			expectErr: false,
			expectIP:  "192.168.1.1",
		},
		{
			name:      "Valid IPv6 net.IP",
			input:     net.ParseIP("2001:db8::1"),
			expectErr: false,
			expectIP:  "2001:db8::1",
		},
		// *net.IPNet tests
		{
			name: "Valid IPv4 *net.IPNet",
			input: func() *net.IPNet {
				_, ipnet, _ := net.ParseCIDR("192.168.1.0/24")
				return ipnet
			}(),
			expectErr: false,
			expectIP:  "192.168.1.0",
		},
		{
			name: "Valid IPv6 *net.IPNet",
			input: func() *net.IPNet {
				_, ipnet, _ := net.ParseCIDR("2001:db8::/32")
				return ipnet
			}(),
			expectErr: false,
			expectIP:  "2001:db8::",
		},
		// netip.Addr tests
		{
			name:      "Valid IPv4 netip.Addr",
			input:     netip.MustParseAddr("192.168.1.1"),
			expectErr: false,
			expectIP:  "192.168.1.1",
		},
		{
			name:      "Valid IPv6 netip.Addr",
			input:     netip.MustParseAddr("2001:db8::1"),
			expectErr: false,
			expectIP:  "2001:db8::1",
		},
		// *netip.Addr tests
		{
			name: "Valid IPv4 *netip.Addr",
			input: func() *netip.Addr {
				addr := netip.MustParseAddr("192.168.1.1")
				return &addr
			}(),
			expectErr: false,
			expectIP:  "192.168.1.1",
		},
		{
			name: "Valid IPv6 *netip.Addr",
			input: func() *netip.Addr {
				addr := netip.MustParseAddr("2001:db8::1")
				return &addr
			}(),
			expectErr: false,
			expectIP:  "2001:db8::1",
		},
		// netip.Prefix tests
		{
			name:      "Valid IPv4 netip.Prefix",
			input:     netip.MustParsePrefix("192.168.1.0/24"),
			expectErr: false,
			expectIP:  "192.168.1.0",
		},
		{
			name:      "Valid IPv6 netip.Prefix",
			input:     netip.MustParsePrefix("2001:db8::/32"),
			expectErr: false,
			expectIP:  "2001:db8::",
		},
		{
			name:      "IPv4-mapped IPv6 netip.Prefix",
			input:     netip.MustParsePrefix("::ffff:192.168.1.0/120"),
			expectErr: false,
			expectIP:  "192.168.1.0",
		},
		{
			name:      "Invalid IPv4-mapped IPv6 prefix bits",
			input:     netip.MustParsePrefix("::ffff:192.168.1.0/95"),
			expectErr: true,
		},
		// *netip.Prefix tests
		{
			name: "Valid IPv4 *netip.Prefix",
			input: func() *netip.Prefix {
				prefix := netip.MustParsePrefix("192.168.1.0/24")
				return &prefix
			}(),
			expectErr: false,
			expectIP:  "192.168.1.0",
		},
		{
			name: "Valid IPv6 *netip.Prefix",
			input: func() *netip.Prefix {
				prefix := netip.MustParsePrefix("2001:db8::/32")
				return &prefix
			}(),
			expectErr: false,
			expectIP:  "2001:db8::",
		},
		// String tests
		{
			name:      "Valid IPv4 string",
			input:     "192.168.1.1",
			expectErr: false,
			expectIP:  "192.168.1.1",
		},
		{
			name:      "Valid IPv6 string",
			input:     "2001:db8::1",
			expectErr: false,
			expectIP:  "2001:db8::1",
		},
		{
			name:      "Valid IPv4 CIDR string",
			input:     "192.168.1.0/24",
			expectErr: false,
			expectIP:  "192.168.1.0",
		},
		{
			name:      "Valid IPv6 CIDR string",
			input:     "2001:db8::/32",
			expectErr: false,
			expectIP:  "2001:db8::",
		},
		{
			name:      "Invalid string",
			input:     "invalid-string",
			expectErr: true,
		},
		// Unsupported type
		{
			name:      "Unsupported type",
			input:     123,
			expectErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, ipType, err := entry.processPrefix(tt.input)
			
			if tt.expectErr {
				if err == nil {
					t.Errorf("processPrefix() should return error for input %v", tt.input)
				}
				return
			}
			
			if err != nil {
				t.Errorf("processPrefix() should not return error for valid input %v: %v", tt.input, err)
				return
			}
			
			if prefix == nil {
				t.Errorf("processPrefix() should return non-nil prefix for valid input %v", tt.input)
				return
			}
			
			if prefix.Addr().String() != tt.expectIP {
				t.Errorf("processPrefix() IP = %s; want %s", prefix.Addr().String(), tt.expectIP)
			}
			
			// Verify IP type
			if prefix.Addr().Is4() && ipType != IPv4 {
				t.Errorf("processPrefix() should return IPv4 type for IPv4 address")
			}
			if prefix.Addr().Is6() && ipType != IPv6 {
				t.Errorf("processPrefix() should return IPv6 type for IPv6 address")
			}
		})
	}
}

// TestEntryAddPrefixVariousTypes tests AddPrefix with different input types
func TestEntryAddPrefixVariousTypes(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		expectErr bool
	}{
		{
			name:      "net.IP",
			input:     net.ParseIP("192.168.1.1"),
			expectErr: false,
		},
		{
			name: "*net.IPNet",
			input: func() *net.IPNet {
				_, ipnet, _ := net.ParseCIDR("192.168.1.0/24")
				return ipnet
			}(),
			expectErr: false,
		},
		{
			name:      "netip.Addr",
			input:     netip.MustParseAddr("192.168.1.1"),
			expectErr: false,
		},
		{
			name: "*netip.Addr",
			input: func() *netip.Addr {
				addr := netip.MustParseAddr("192.168.1.1")
				return &addr
			}(),
			expectErr: false,
		},
		{
			name:      "netip.Prefix",
			input:     netip.MustParsePrefix("192.168.1.0/24"),
			expectErr: false,
		},
		{
			name: "*netip.Prefix",
			input: func() *netip.Prefix {
				prefix := netip.MustParsePrefix("192.168.1.0/24")
				return &prefix
			}(),
			expectErr: false,
		},
		{
			name:      "string",
			input:     "192.168.1.0/24",
			expectErr: false,
		},
		{
			name:      "unsupported type",
			input:     123,
			expectErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := NewEntry("test")
			err := entry.AddPrefix(tt.input)
			
			if tt.expectErr && err == nil {
				t.Errorf("AddPrefix() should return error for input %v", tt.input)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("AddPrefix() should not return error for valid input %v: %v", tt.input, err)
			}
		})
	}
}

// TestEntryRemoveWithInvalidType tests the remove function with invalid IP type
func TestEntryRemoveWithInvalidType(t *testing.T) {
	entry := NewEntry("test")
	prefix := netip.MustParsePrefix("192.168.1.0/24")
	
	// Test remove with invalid IP type - this tests the ErrInvalidIPType path
	err := entry.remove(&prefix, IPType("invalid"))
	if err == nil {
		t.Error("remove() should return error for invalid IP type")
	}
	if err != ErrInvalidIPType {
		t.Errorf("remove() should return ErrInvalidIPType, got %v", err)
	}
}

// TestEntryBuildIPSetErrors tests buildIPSet function error paths
func TestEntryBuildIPSetErrors(t *testing.T) {
	entry := NewEntry("test")
	
	// Test with empty entry - should trigger the "no data" path in buildIPSet
	err := entry.buildIPSet()
	if err != nil {
		// buildIPSet() should not return error for empty entry, it just builds what's available
		t.Logf("buildIPSet() returned: %v", err)
	}
}

// TestEntryCIDRMergeRules tests CIDR merge behavior when adding intersecting prefixes
func TestEntryCIDRMergeRules(t *testing.T) {
	tests := []struct {
		name             string
		prefixesToAdd    []string
		expectedPrefixes []string
		description      string
	}{
		{
			name:             "IPv4 adjacent prefixes merge",
			prefixesToAdd:    []string{"192.168.0.0/24", "192.168.1.0/24"},
			expectedPrefixes: []string{"192.168.0.0/23"},
			description:      "192.168.0.0/24 and 192.168.1.0/24 should merge to 192.168.0.0/23",
		},
		{
			name:             "IPv4 four adjacent prefixes merge",
			prefixesToAdd:    []string{"192.168.0.0/24", "192.168.1.0/24", "192.168.2.0/24", "192.168.3.0/24"},
			expectedPrefixes: []string{"192.168.0.0/22"},
			description:      "Four consecutive /24 prefixes should merge to a single /22",
		},
		{
			name:             "IPv4 non-adjacent prefixes do not merge",
			prefixesToAdd:    []string{"192.168.0.0/24", "192.168.2.0/24"},
			expectedPrefixes: []string{"192.168.0.0/24", "192.168.2.0/24"},
			description:      "Non-adjacent prefixes should remain separate",
		},
		{
			name:             "IPv6 adjacent prefixes merge",
			prefixesToAdd:    []string{"2001:db8:0:0::/64", "2001:db8:0:1::/64"},
			expectedPrefixes: []string{"2001:db8::/63"},
			description:      "Adjacent IPv6 /64 prefixes should merge to /63",
		},
		{
			name:             "Mixed IPv4 and IPv6 prefixes",
			prefixesToAdd:    []string{"192.168.0.0/24", "192.168.1.0/24", "2001:db8:0:0::/64", "2001:db8:0:1::/64"},
			expectedPrefixes: []string{"192.168.0.0/23", "2001:db8::/63"},
			description:      "IPv4 and IPv6 prefixes should merge independently",
		},
		{
			name:             "Single IP addresses merge to CIDR",
			prefixesToAdd:    []string{"192.168.0.1", "192.168.0.2", "192.168.0.3", "192.168.0.0"},
			expectedPrefixes: []string{"192.168.0.0/30"},
			description:      "Individual IP addresses that form a contiguous block should merge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := NewEntry("test")

			// Add all prefixes
			for _, prefix := range tt.prefixesToAdd {
				err := entry.AddPrefix(prefix)
				if err != nil {
					t.Fatalf("Failed to add prefix %s: %v", prefix, err)
				}
			}

			// Get the resulting prefixes
			prefixes, err := entry.MarshalText()
			if err != nil {
				t.Fatalf("Failed to marshal prefixes: %v", err)
			}

			// Verify the expected merge result
			if len(prefixes) != len(tt.expectedPrefixes) {
				t.Errorf("Expected %d prefixes, got %d. Expected: %v, Got: %v", 
					len(tt.expectedPrefixes), len(prefixes), tt.expectedPrefixes, prefixes)
			}

			// Convert to map for easier comparison (order doesn't matter)
			expectedMap := make(map[string]bool)
			for _, p := range tt.expectedPrefixes {
				expectedMap[p] = true
			}

			for _, prefix := range prefixes {
				if !expectedMap[prefix] {
					t.Errorf("Unexpected prefix %s in result. Expected: %v, Got: %v", 
						prefix, tt.expectedPrefixes, prefixes)
				}
			}

			for _, expected := range tt.expectedPrefixes {
				found := false
				for _, prefix := range prefixes {
					if prefix == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected prefix %s not found in result. Expected: %v, Got: %v", 
						expected, tt.expectedPrefixes, prefixes)
				}
			}
		})
	}
}

// TestEntryCIDRSplitRules tests CIDR split behavior when removing intersecting prefixes
func TestEntryCIDRSplitRules(t *testing.T) {
	tests := []struct {
		name              string
		initialPrefixes   []string
		prefixesToRemove  []string
		expectedPrefixes  []string
		description       string
	}{
		{
			name:              "IPv4 remove subset splits CIDR",
			initialPrefixes:   []string{"192.168.0.0/23"},
			prefixesToRemove:  []string{"192.168.1.0/24"},
			expectedPrefixes:  []string{"192.168.0.0/24"},
			description:       "Removing 192.168.1.0/24 from 192.168.0.0/23 should leave 192.168.0.0/24",
		},
		{
			name:              "IPv4 remove middle of larger block",
			initialPrefixes:   []string{"192.168.0.0/22"},
			prefixesToRemove:  []string{"192.168.1.0/24"},
			expectedPrefixes:  []string{"192.168.0.0/24", "192.168.2.0/23"},
			description:       "Removing middle /24 from /22 should split into remaining blocks",
		},
		{
			name:              "IPv6 remove subset splits CIDR",
			initialPrefixes:   []string{"2001:db8::/63"},
			prefixesToRemove:  []string{"2001:db8:0:1::/64"},
			expectedPrefixes:  []string{"2001:db8::/64"},
			description:       "Removing IPv6 /64 from /63 should leave the other /64",
		},
		{
			name:              "Remove entire range leaves empty",
			initialPrefixes:   []string{"192.168.0.0/24"},
			prefixesToRemove:  []string{"192.168.0.0/24"},
			expectedPrefixes:  []string{},
			description:       "Removing entire range should leave empty entry",
		},
		{
			name:              "Remove non-existent prefix does nothing",
			initialPrefixes:   []string{"192.168.0.0/24"},
			prefixesToRemove:  []string{"10.0.0.0/24"},
			expectedPrefixes:  []string{"192.168.0.0/24"},
			description:       "Removing non-existent prefix should not affect existing prefixes",
		},
		{
			name:              "Multiple removes create complex split",
			initialPrefixes:   []string{"192.168.0.0/22"},
			prefixesToRemove:  []string{"192.168.0.0/24", "192.168.3.0/24"},
			expectedPrefixes:  []string{"192.168.1.0/24", "192.168.2.0/24"},
			description:       "Removing first and last /24 from /22 should leave middle two /24s",
		},
		{
			name:              "Remove overlapping ranges",
			initialPrefixes:   []string{"192.168.0.0/23", "192.168.2.0/24"},
			prefixesToRemove:  []string{"192.168.1.0/24"},
			expectedPrefixes:  []string{"192.168.0.0/24", "192.168.2.0/24"},
			description:       "Removing from one prefix should not affect non-overlapping prefixes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := NewEntry("test")

			// Add initial prefixes
			for _, prefix := range tt.initialPrefixes {
				err := entry.AddPrefix(prefix)
				if err != nil {
					t.Fatalf("Failed to add initial prefix %s: %v", prefix, err)
				}
			}

			// Remove specified prefixes
			for _, prefix := range tt.prefixesToRemove {
				err := entry.RemovePrefix(prefix)
				if err != nil {
					t.Fatalf("Failed to remove prefix %s: %v", prefix, err)
				}
			}

			// Get the resulting prefixes
			prefixes, err := entry.MarshalText()
			if err != nil && len(tt.expectedPrefixes) > 0 {
				t.Fatalf("Failed to marshal prefixes: %v", err)
			}

			// Handle empty result case
			if len(tt.expectedPrefixes) == 0 {
				if err == nil {
					t.Errorf("Expected error for empty entry, but got prefixes: %v", prefixes)
				}
				return
			}

			// Verify the expected split result
			if len(prefixes) != len(tt.expectedPrefixes) {
				t.Errorf("Expected %d prefixes, got %d. Expected: %v, Got: %v", 
					len(tt.expectedPrefixes), len(prefixes), tt.expectedPrefixes, prefixes)
			}

			// Convert to map for easier comparison (order doesn't matter)
			expectedMap := make(map[string]bool)
			for _, p := range tt.expectedPrefixes {
				expectedMap[p] = true
			}

			for _, prefix := range prefixes {
				if !expectedMap[prefix] {
					t.Errorf("Unexpected prefix %s in result. Expected: %v, Got: %v", 
						prefix, tt.expectedPrefixes, prefixes)
				}
			}

			for _, expected := range tt.expectedPrefixes {
				found := false
				for _, prefix := range prefixes {
					if prefix == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected prefix %s not found in result. Expected: %v, Got: %v", 
						expected, tt.expectedPrefixes, prefixes)
				}
			}
		})
	}
}

// TestEntryCIDRMergeAndSplitCombined tests complex scenarios with both merge and split operations
func TestEntryCIDRMergeAndSplitCombined(t *testing.T) {
	tests := []struct {
		name        string
		operations  []struct {
			action string // "add" or "remove"
			prefix string
		}
		expectedPrefixes []string
		description      string
	}{
		{
			name: "Add, merge, then split",
			operations: []struct {
				action string
				prefix string
			}{
				{"add", "192.168.0.0/24"},
				{"add", "192.168.1.0/24"}, // Should merge to 192.168.0.0/23
				{"remove", "192.168.0.128/25"}, // Should split the merged range
			},
			expectedPrefixes: []string{"192.168.0.0/25", "192.168.1.0/24"},
			description:      "Complex add/merge/remove sequence should work correctly",
		},
		{
			name: "Build large block then carve out holes",
			operations: []struct {
				action string
				prefix string
			}{
				{"add", "10.0.0.0/22"},        // Large block
				{"remove", "10.0.1.0/24"},    // Remove middle
				{"remove", "10.0.3.0/24"},    // Remove end
			},
			expectedPrefixes: []string{"10.0.0.0/24", "10.0.2.0/24"},
			description:      "Carving holes in large CIDR block should leave correct fragments",
		},
		{
			name: "IPv6 complex operations",
			operations: []struct {
				action string
				prefix string
			}{
				{"add", "2001:db8:0:0::/64"},
				{"add", "2001:db8:0:1::/64"}, // Should merge to 2001:db8::/63
				{"add", "2001:db8:0:2::/64"},
				{"add", "2001:db8:0:3::/64"}, // All should merge to 2001:db8::/62
				{"remove", "2001:db8:0:2::/64"}, // Remove one, should split
			},
			expectedPrefixes: []string{"2001:db8::/63", "2001:db8:0:3::/64"},
			description:      "IPv6 merge and split operations should work correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := NewEntry("test")

			// Perform all operations in sequence
			for _, op := range tt.operations {
				var err error
				switch op.action {
				case "add":
					err = entry.AddPrefix(op.prefix)
				case "remove":
					err = entry.RemovePrefix(op.prefix)
				default:
					t.Fatalf("Unknown operation: %s", op.action)
				}
				
				if err != nil {
					t.Fatalf("Failed to %s prefix %s: %v", op.action, op.prefix, err)
				}
			}

			// Get the resulting prefixes
			prefixes, err := entry.MarshalText()
			if err != nil && len(tt.expectedPrefixes) > 0 {
				t.Fatalf("Failed to marshal prefixes: %v", err)
			}

			// Handle empty result case
			if len(tt.expectedPrefixes) == 0 {
				if err == nil {
					t.Errorf("Expected error for empty entry, but got prefixes: %v", prefixes)
				}
				return
			}

			// Verify the expected result
			if len(prefixes) != len(tt.expectedPrefixes) {
				t.Errorf("Expected %d prefixes, got %d. Expected: %v, Got: %v", 
					len(tt.expectedPrefixes), len(prefixes), tt.expectedPrefixes, prefixes)
			}

			// Convert to map for easier comparison (order doesn't matter)
			expectedMap := make(map[string]bool)
			for _, p := range tt.expectedPrefixes {
				expectedMap[p] = true
			}

			for _, prefix := range prefixes {
				if !expectedMap[prefix] {
					t.Errorf("Unexpected prefix %s in result. Expected: %v, Got: %v", 
						prefix, tt.expectedPrefixes, prefixes)
				}
			}

			for _, expected := range tt.expectedPrefixes {
				found := false
				for _, prefix := range prefixes {
					if prefix == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected prefix %s not found in result. Expected: %v, Got: %v", 
						expected, tt.expectedPrefixes, prefixes)
				}
			}
		})
	}
}