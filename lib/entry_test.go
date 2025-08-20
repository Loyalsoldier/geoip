package lib

import (
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
}