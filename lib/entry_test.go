package lib

import (
	"net"
	"net/netip"
	"strings"
	"testing"
)

func TestNewEntry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "lowercase", input: "cn", want: "CN"},
		{name: "mixed case", input: "Private", want: "PRIVATE"},
		{name: "surrounding spaces", input: "  geolocation-cn \t", want: "GEOLOCATION-CN"},
		{name: "already uppercase", input: "US", want: "US"},
		{name: "empty", input: "", want: ""},
		{name: "only spaces", input: "   ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry := NewEntry(tt.input)
			if got := entry.GetName(); got != tt.want {
				t.Fatalf("GetName() = %q, want %q", got, tt.want)
			}
			if entry.hasIPv4Builder() || entry.hasIPv6Builder() || entry.hasIPv4Set() || entry.hasIPv6Set() {
				t.Fatal("new entry should not have any builder or set")
			}
		})
	}
}

func TestEntryProcessPrefix(t *testing.T) {
	t.Parallel()

	ipv4 := net.ParseIP("1.2.3.4")
	ipv4Net := net.IPNet{IP: net.IP{1, 2, 3, 0}, Mask: net.CIDRMask(24, 32)}
	ipv4Addr := netip.MustParseAddr("1.2.3.4")
	ipv4Prefix := netip.MustParsePrefix("1.2.3.0/24")
	ipv4String := "1.2.3.0/24"

	tests := []struct {
		name     string
		input    any
		want     string
		wantType IPType
		wantErr  error
	}{
		// net.IP
		{name: "net.IP IPv4 16-byte form", input: net.ParseIP("1.2.3.4"), want: "1.2.3.4/32", wantType: IPv4},
		{name: "net.IP IPv4 4-byte form", input: net.IP{1, 2, 3, 4}, want: "1.2.3.4/32", wantType: IPv4},
		{name: "net.IP IPv4-mapped IPv6", input: net.ParseIP("::ffff:1.2.3.4"), want: "1.2.3.4/32", wantType: IPv4},
		{name: "net.IP IPv6", input: net.ParseIP("2001:db8::1"), want: "2001:db8::1/128", wantType: IPv6},
		{name: "net.IP invalid length", input: net.IP{1, 2, 3}, wantErr: ErrInvalidIP},
		{name: "net.IP nil", input: net.IP(nil), wantErr: ErrInvalidIP},
		{name: "*net.IP", input: &ipv4, want: "1.2.3.4/32", wantType: IPv4},
		{name: "*net.IP nil", input: (*net.IP)(nil), wantErr: ErrInvalidPrefixType},

		// net.IPNet
		{name: "net.IPNet IPv4", input: net.IPNet{IP: net.IP{1, 2, 3, 4}, Mask: net.CIDRMask(24, 32)}, want: "1.2.3.0/24", wantType: IPv4},
		{name: "net.IPNet IPv4 /0", input: net.IPNet{IP: net.IP{1, 2, 3, 4}, Mask: net.CIDRMask(0, 32)}, want: "0.0.0.0/0", wantType: IPv4},
		{name: "net.IPNet IPv4 /32", input: net.IPNet{IP: net.IP{1, 2, 3, 4}, Mask: net.CIDRMask(32, 32)}, want: "1.2.3.4/32", wantType: IPv4},
		{name: "net.IPNet IPv4 16-byte IP with 4-byte mask", input: net.IPNet{IP: net.ParseIP("1.2.3.4"), Mask: net.CIDRMask(24, 32)}, want: "1.2.3.0/24", wantType: IPv4},
		{name: "net.IPNet IPv4-mapped IPv6 /120", input: net.IPNet{IP: net.ParseIP("::ffff:1.2.3.4"), Mask: net.CIDRMask(120, 128)}, want: "1.2.3.0/24", wantType: IPv4},
		{name: "net.IPNet IPv4-mapped IPv6 /128", input: net.IPNet{IP: net.ParseIP("::ffff:1.2.3.4"), Mask: net.CIDRMask(128, 128)}, want: "1.2.3.4/32", wantType: IPv4},
		{name: "net.IPNet IPv4-mapped IPv6 /96", input: net.IPNet{IP: net.ParseIP("::ffff:1.2.3.4"), Mask: net.CIDRMask(96, 128)}, want: "0.0.0.0/0", wantType: IPv4},
		{name: "net.IPNet IPv4-mapped IPv6 /95", input: net.IPNet{IP: net.ParseIP("::ffff:1.2.3.4"), Mask: net.CIDRMask(95, 128)}, wantErr: ErrInvalidPrefix},
		{name: "net.IPNet IPv4-mapped IPv6 /0", input: net.IPNet{IP: net.ParseIP("::ffff:1.2.3.4"), Mask: net.CIDRMask(0, 128)}, wantErr: ErrInvalidPrefix},
		{name: "net.IPNet IPv6", input: net.IPNet{IP: net.ParseIP("2001:db8::1"), Mask: net.CIDRMask(32, 128)}, want: "2001:db8::/32", wantType: IPv6},
		{name: "net.IPNet IPv6 /128", input: net.IPNet{IP: net.ParseIP("2001:db8::1"), Mask: net.CIDRMask(128, 128)}, want: "2001:db8::1/128", wantType: IPv6},
		{name: "net.IPNet IPv4 IP with IPv6 mask", input: net.IPNet{IP: net.IP{1, 2, 3, 0}, Mask: net.CIDRMask(120, 128)}, wantErr: ErrInvalidIPNet},
		{name: "net.IPNet IPv6 IP with IPv4 mask", input: net.IPNet{IP: net.ParseIP("2001:db8::"), Mask: net.CIDRMask(24, 32)}, wantErr: ErrInvalidIPNet},
		{name: "net.IPNet non-contiguous mask", input: net.IPNet{IP: net.IP{1, 2, 3, 0}, Mask: net.IPMask{255, 0, 255, 0}}, wantErr: ErrInvalidIPNet},
		{name: "net.IPNet invalid IP length", input: net.IPNet{IP: net.IP{1, 2, 3}, Mask: net.CIDRMask(24, 32)}, wantErr: ErrInvalidIPNet},
		{name: "net.IPNet empty", input: net.IPNet{}, wantErr: ErrInvalidIPNet},
		{name: "*net.IPNet", input: &ipv4Net, want: "1.2.3.0/24", wantType: IPv4},
		{name: "*net.IPNet nil", input: (*net.IPNet)(nil), wantErr: ErrInvalidPrefixType},

		// netip.Addr
		{name: "netip.Addr IPv4", input: netip.MustParseAddr("1.2.3.4"), want: "1.2.3.4/32", wantType: IPv4},
		{name: "netip.Addr IPv4-mapped IPv6", input: netip.MustParseAddr("::ffff:1.2.3.4"), want: "1.2.3.4/32", wantType: IPv4},
		{name: "netip.Addr IPv6", input: netip.MustParseAddr("2001:db8::1"), want: "2001:db8::1/128", wantType: IPv6},
		{name: "netip.Addr IPv6 with zone", input: netip.MustParseAddr("fe80::1%eth0"), want: "fe80::1/128", wantType: IPv6},
		{name: "netip.Addr zero value", input: netip.Addr{}, wantErr: ErrInvalidIPType},
		{name: "*netip.Addr", input: &ipv4Addr, want: "1.2.3.4/32", wantType: IPv4},
		{name: "*netip.Addr nil", input: (*netip.Addr)(nil), wantErr: ErrInvalidPrefixType},

		// netip.Prefix
		{name: "netip.Prefix IPv4", input: netip.MustParsePrefix("1.2.3.0/24"), want: "1.2.3.0/24", wantType: IPv4},
		{name: "netip.Prefix IPv4 with host bits", input: netip.MustParsePrefix("1.2.3.4/24"), want: "1.2.3.0/24", wantType: IPv4},
		{name: "netip.Prefix IPv4 invalid bits", input: netip.PrefixFrom(netip.MustParseAddr("1.2.3.4"), 33), wantErr: ErrInvalidPrefix},
		{name: "netip.Prefix IPv4-mapped IPv6 /120", input: netip.MustParsePrefix("::ffff:1.2.3.4/120"), want: "1.2.3.0/24", wantType: IPv4},
		{name: "netip.Prefix IPv4-mapped IPv6 /128", input: netip.MustParsePrefix("::ffff:1.2.3.4/128"), want: "1.2.3.4/32", wantType: IPv4},
		{name: "netip.Prefix IPv4-mapped IPv6 /96", input: netip.MustParsePrefix("::ffff:1.2.3.4/96"), want: "0.0.0.0/0", wantType: IPv4},
		{name: "netip.Prefix IPv4-mapped IPv6 /95", input: netip.MustParsePrefix("::ffff:1.2.3.4/95"), wantErr: ErrInvalidPrefix},
		{name: "netip.Prefix IPv4-mapped IPv6 /0", input: netip.MustParsePrefix("::ffff:1.2.3.4/0"), wantErr: ErrInvalidPrefix},
		{name: "netip.Prefix IPv4-mapped IPv6 invalid bits", input: netip.PrefixFrom(netip.MustParseAddr("::ffff:1.2.3.4"), 129), wantErr: ErrInvalidPrefix},
		{name: "netip.Prefix IPv6", input: netip.MustParsePrefix("2001:db8::1/32"), want: "2001:db8::/32", wantType: IPv6},
		{name: "netip.Prefix IPv6 invalid bits", input: netip.PrefixFrom(netip.MustParseAddr("2001:db8::1"), 129), wantErr: ErrInvalidPrefix},
		{name: "netip.Prefix zero value", input: netip.Prefix{}, wantErr: ErrInvalidIPType},
		{name: "*netip.Prefix", input: &ipv4Prefix, want: "1.2.3.0/24", wantType: IPv4},
		{name: "*netip.Prefix nil", input: (*netip.Prefix)(nil), wantErr: ErrInvalidPrefixType},

		// string: IP address
		{name: "string IPv4", input: "1.2.3.4", want: "1.2.3.4/32", wantType: IPv4},
		{name: "string IPv4 with spaces", input: " \t1.2.3.4  ", want: "1.2.3.4/32", wantType: IPv4},
		{name: "string IPv6", input: "2001:db8::1", want: "2001:db8::1/128", wantType: IPv6},
		{name: "string IPv6 with zone", input: "fe80::1%eth0", want: "fe80::1/128", wantType: IPv6},
		{name: "string IPv4-mapped IPv6", input: "::ffff:1.2.3.4", want: "1.2.3.4/32", wantType: IPv4},
		{name: "string IPv4-mapped IPv6 hex form", input: "::ffff:102:304", want: "1.2.3.4/32", wantType: IPv4},
		{name: "string invalid IP", input: "abc", wantErr: ErrInvalidIP},
		{name: "string invalid IPv4 octet", input: "1.2.3.256", wantErr: ErrInvalidIP},
		{name: "string invalid IPv4-mapped IPv6", input: "::ffff:1.2.3.256", wantErr: ErrInvalidIP},
		{name: "string invalid IPv6", input: "2001:db8:::1", wantErr: ErrInvalidIP},

		// string: CIDR
		{name: "string IPv4 CIDR", input: "1.2.3.0/24", want: "1.2.3.0/24", wantType: IPv4},
		{name: "string IPv4 CIDR with host bits", input: "192.168.0.3/24", want: "192.168.0.0/24", wantType: IPv4},
		{name: "string IPv4 CIDR /0", input: "0.0.0.0/0", want: "0.0.0.0/0", wantType: IPv4},
		{name: "string IPv4 CIDR /32", input: "1.2.3.4/32", want: "1.2.3.4/32", wantType: IPv4},
		{name: "string IPv6 CIDR", input: "2001:db8::1/64", want: "2001:db8::/64", wantType: IPv6},
		{name: "string IPv6 CIDR /0", input: "::/0", want: "::/0", wantType: IPv6},
		{name: "string IPv6 CIDR /128", input: "2001:db8::1/128", want: "2001:db8::1/128", wantType: IPv6},
		{name: "string IPv4-mapped IPv6 CIDR /120", input: "::ffff:192.168.0.0/120", want: "192.168.0.0/24", wantType: IPv4},
		{name: "string IPv4-mapped IPv6 CIDR /120 with host bits", input: "::ffff:192.168.0.5/120", want: "192.168.0.0/24", wantType: IPv4},
		{name: "string IPv4-mapped IPv6 CIDR /104", input: "::ffff:192.168.0.0/104", want: "192.0.0.0/8", wantType: IPv4},
		{name: "string IPv4-mapped IPv6 CIDR /128", input: "::ffff:192.168.0.1/128", want: "192.168.0.1/32", wantType: IPv4},
		{name: "string IPv4-mapped IPv6 CIDR /96", input: "::ffff:192.168.0.0/96", want: "0.0.0.0/0", wantType: IPv4},
		{name: "string IPv4-mapped IPv6 CIDR hex form", input: "::ffff:c0a8:0/120", want: "192.168.0.0/24", wantType: IPv4},
		{name: "string IPv4-mapped IPv6 CIDR /95", input: "::ffff:192.168.0.0/95", wantErr: ErrInvalidCIDR},
		{name: "string IPv4-mapped IPv6 CIDR /80", input: "::ffff:192.168.0.0/80", wantErr: ErrInvalidCIDR},
		{name: "string IPv4-mapped IPv6 CIDR /24", input: "::ffff:1.2.3.4/24", wantErr: ErrInvalidCIDR},
		{name: "string IPv4-mapped IPv6 CIDR /0", input: "::ffff:1.2.3.4/0", wantErr: ErrInvalidCIDR},
		{name: "string IPv4-mapped IPv6 CIDR /129", input: "::ffff:192.168.0.0/129", wantErr: ErrInvalidCIDR},
		{name: "string IPv4-mapped IPv6 CIDR negative bits", input: "::ffff:192.168.0.0/-1", wantErr: ErrInvalidCIDR},
		{name: "string IPv4-mapped IPv6 CIDR invalid IP", input: "::ffff:1.2.3.256/120", wantErr: ErrInvalidCIDR},
		{name: "string IPv4 CIDR invalid bits", input: "1.2.3.4/33", wantErr: ErrInvalidCIDR},
		{name: "string IPv4 CIDR missing bits", input: "1.2.3.4/", wantErr: ErrInvalidCIDR},
		{name: "string IPv4 CIDR non-numeric bits", input: "1.2.3.4/abc", wantErr: ErrInvalidCIDR},
		{name: "string IPv4 CIDR incomplete IP", input: "1.2.3/24", wantErr: ErrInvalidCIDR},
		{name: "string IPv6 CIDR invalid bits", input: "2001:db8::/129", wantErr: ErrInvalidCIDR},
		{name: "string IPv6 CIDR with zone", input: "fe80::1%eth0/64", wantErr: ErrInvalidCIDR},
		{name: "string invalid CIDR", input: "abc/24", wantErr: ErrInvalidCIDR},

		// string: comments
		{name: "string empty", input: "", wantErr: ErrCommentLine},
		{name: "string only spaces", input: "  \t ", wantErr: ErrCommentLine},
		{name: "string hash comment", input: "# 1.2.3.4", wantErr: ErrCommentLine},
		{name: "string double slash comment", input: "// 1.2.3.4", wantErr: ErrCommentLine},
		{name: "string block comment", input: "/* 1.2.3.4 */", wantErr: ErrCommentLine},
		{name: "string IP with hash comment", input: "1.2.3.4 # comment", want: "1.2.3.4/32", wantType: IPv4},
		{name: "string CIDR with double slash comment", input: "1.2.3.0/24 // comment", want: "1.2.3.0/24", wantType: IPv4},
		{name: "string CIDR with block comment", input: "2001:db8::/32 /* comment */", want: "2001:db8::/32", wantType: IPv6},
		{name: "*string", input: &ipv4String, want: "1.2.3.0/24", wantType: IPv4},
		{name: "*string nil", input: (*string)(nil), wantErr: ErrInvalidPrefixType},

		// unsupported types
		{name: "nil", input: nil, wantErr: ErrInvalidPrefixType},
		{name: "int", input: 123, wantErr: ErrInvalidPrefixType},
		{name: "string slice", input: []string{"1.2.3.4"}, wantErr: ErrInvalidPrefixType},
		{name: "byte slice", input: []byte{1, 2, 3, 4}, wantErr: ErrInvalidPrefixType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry := NewEntry("test")
			prefix, ipType, err := entry.processPrefix(tt.input)
			if tt.wantErr != nil {
				assertErrorIs(t, err, tt.wantErr)
				if prefix != nil {
					t.Fatalf("prefix = %v, want nil", prefix)
				}
				if ipType != "" {
					t.Fatalf("ipType = %q, want empty", ipType)
				}
				return
			}
			assertErrorIs(t, err, nil)
			if prefix == nil {
				t.Fatal("prefix is nil")
			}
			if want := netip.MustParsePrefix(tt.want); *prefix != want {
				t.Fatalf("prefix = %s, want %s", prefix, want)
			}
			if ipType != tt.wantType {
				t.Fatalf("ipType = %q, want %q", ipType, tt.wantType)
			}
		})
	}
}

func TestEntryAddPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		inputs []any
		want   []string
	}{
		{
			name:   "merge adjacent IPv4 CIDRs with host bits",
			inputs: []any{"192.168.0.3/24", "192.168.1.5/24"},
			want:   []string{"192.168.0.0/23"},
		},
		{
			name:   "merge adjacent IPv4 CIDRs in reverse order",
			inputs: []any{"192.168.1.5/24", "192.168.0.3/24"},
			want:   []string{"192.168.0.0/23"},
		},
		{
			name:   "merge four adjacent IPv4 CIDRs",
			inputs: []any{"10.0.0.0/24", "10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"},
			want:   []string{"10.0.0.0/22"},
		},
		{
			name:   "merge consecutive IPv4 addresses",
			inputs: []any{"1.1.1.0", "1.1.1.1", "1.1.1.2", "1.1.1.3"},
			want:   []string{"1.1.1.0/30"},
		},
		{
			name:   "unaligned adjacent IPv4 CIDRs are not merged",
			inputs: []any{"192.168.1.0/24", "192.168.2.0/24"},
			want:   []string{"192.168.1.0/24", "192.168.2.0/24"},
		},
		{
			name:   "unaligned consecutive IPv4 addresses are not merged",
			inputs: []any{"1.1.1.1", "1.1.1.2"},
			want:   []string{"1.1.1.1/32", "1.1.1.2/32"},
		},
		{
			name:   "contained IPv4 CIDR is absorbed",
			inputs: []any{"10.1.0.0/16", "10.0.0.0/8", "10.2.3.4"},
			want:   []string{"10.0.0.0/8"},
		},
		{
			name:   "duplicated IPv4 CIDR",
			inputs: []any{"1.2.3.0/24", "1.2.3.0/24"},
			want:   []string{"1.2.3.0/24"},
		},
		{
			name:   "IPv4 CIDRs are sorted",
			inputs: []any{"10.0.0.0/8", "1.0.0.0/8", "8.8.8.8"},
			want:   []string{"1.0.0.0/8", "8.8.8.8/32", "10.0.0.0/8"},
		},
		{
			name:   "IPv4-mapped IPv6 CIDR merges with IPv4 CIDR",
			inputs: []any{"::ffff:192.168.0.0/120", "192.168.1.0/24"},
			want:   []string{"192.168.0.0/23"},
		},
		{
			name:   "IPv4-mapped IPv6 CIDRs merge with each other",
			inputs: []any{"::ffff:192.168.0.3/120", "::ffff:192.168.1.5/120"},
			want:   []string{"192.168.0.0/23"},
		},
		{
			name:   "IPv4-mapped IPv6 addresses merge with IPv4 addresses",
			inputs: []any{"::ffff:1.1.1.0", "1.1.1.1"},
			want:   []string{"1.1.1.0/31"},
		},
		{
			name: "IPv4-mapped IPv6 of mixed types merge",
			inputs: []any{
				netip.MustParsePrefix("::ffff:10.0.0.0/120"),
				net.IPNet{IP: net.ParseIP("::ffff:10.0.1.0"), Mask: net.CIDRMask(120, 128)},
				netip.MustParseAddr("::ffff:10.0.2.0"),
				net.ParseIP("::ffff:10.0.2.1"),
			},
			want: []string{"10.0.0.0/23", "10.0.2.0/31"},
		},
		{
			name:   "merge adjacent IPv6 CIDRs",
			inputs: []any{"2001:db8::/33", "2001:db8:8000::/33"},
			want:   []string{"2001:db8::/32"},
		},
		{
			name:   "merge adjacent IPv6 CIDRs with host bits",
			inputs: []any{"2001:db8::1/33", "2001:db8:8000::1/33"},
			want:   []string{"2001:db8::/32"},
		},
		{
			name:   "merge consecutive IPv6 addresses",
			inputs: []any{"2001:db8::", "2001:db8::1"},
			want:   []string{"2001:db8::/127"},
		},
		{
			name:   "unaligned adjacent IPv6 CIDRs are not merged",
			inputs: []any{"2001:db8:1::/48", "2001:db8:2::/48"},
			want:   []string{"2001:db8:1::/48", "2001:db8:2::/48"},
		},
		{
			name:   "contained IPv6 CIDR is absorbed",
			inputs: []any{"2001:db8:1::/48", "2001:db8::/32"},
			want:   []string{"2001:db8::/32"},
		},
		{
			name:   "IPv4 and IPv6 are merged separately and IPv4 comes first",
			inputs: []any{"2001:db8::/33", "192.168.0.3/24", "2001:db8:8000::/33", "192.168.1.5/24"},
			want:   []string{"192.168.0.0/23", "2001:db8::/32"},
		},
		{
			name:   "comments and empty lines are ignored",
			inputs: []any{"# comment", "", "// comment", "/* comment */", "1.1.1.1 # one"},
			want:   []string{"1.1.1.1/32"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry := NewEntry("test")
			for _, input := range tt.inputs {
				if err := entry.AddPrefix(input); err != nil {
					t.Fatalf("AddPrefix(%v) failed: %v", input, err)
				}
			}
			assertStringSlice(t, entryText(t, entry), tt.want)
		})
	}
}

func TestEntryAddPrefixInvalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   any
		wantErr error
	}{
		{name: "invalid IP string", input: "1.2.3.256", wantErr: ErrInvalidIP},
		{name: "invalid CIDR string", input: "1.2.3.4/33", wantErr: ErrInvalidCIDR},
		{name: "IPv4-mapped IPv6 CIDR with too few bits", input: "::ffff:1.2.3.4/95", wantErr: ErrInvalidCIDR},
		{name: "IPv4-mapped IPv6 CIDR with too many bits", input: "::ffff:1.2.3.4/129", wantErr: ErrInvalidCIDR},
		{name: "IPv4-mapped IPv6 prefix with too few bits", input: netip.MustParsePrefix("::ffff:1.2.3.4/95"), wantErr: ErrInvalidPrefix},
		{name: "IPv4-mapped IPv6 IPNet with too few bits", input: net.IPNet{IP: net.ParseIP("::ffff:1.2.3.4"), Mask: net.CIDRMask(95, 128)}, wantErr: ErrInvalidPrefix},
		{name: "unsupported type", input: 42, wantErr: ErrInvalidPrefixType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry := newTestEntry(t, "test", "1.1.1.1", "2001:db8::1")
			assertErrorIs(t, entry.AddPrefix(tt.input), tt.wantErr)
			// The entry must be left untouched.
			assertStringSlice(t, entryText(t, entry), []string{"1.1.1.1/32", "2001:db8::1/128"})
		})
	}
}

func TestEntryRemovePrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		initial []string
		remove  []string
		want    []string
	}{
		{
			name:    "split IPv4 CIDR by removing upper half",
			initial: []string{"192.168.0.0/23"},
			remove:  []string{"192.168.1.5/24"},
			want:    []string{"192.168.0.0/24"},
		},
		{
			name:    "split IPv4 CIDR by removing lower half",
			initial: []string{"192.168.0.0/23"},
			remove:  []string{"192.168.0.3/24"},
			want:    []string{"192.168.1.0/24"},
		},
		{
			name:    "split IPv4 CIDR by removing a quarter",
			initial: []string{"192.168.0.0/23"},
			remove:  []string{"192.168.0.128/25"},
			want:    []string{"192.168.0.0/25", "192.168.1.0/24"},
		},
		{
			name:    "split IPv4 CIDR by removing a single IP",
			initial: []string{"192.168.0.0/24"},
			remove:  []string{"192.168.0.0"},
			want: []string{
				"192.168.0.1/32", "192.168.0.2/31", "192.168.0.4/30", "192.168.0.8/29",
				"192.168.0.16/28", "192.168.0.32/27", "192.168.0.64/26", "192.168.0.128/25",
			},
		},
		{
			name:    "split IPv4 CIDR by removing IPv4-mapped IPv6 CIDR",
			initial: []string{"192.168.0.0/23"},
			remove:  []string{"::ffff:192.168.1.0/120"},
			want:    []string{"192.168.0.0/24"},
		},
		{
			name:    "split IPv4 CIDR by removing IPv4-mapped IPv6 address",
			initial: []string{"192.168.0.0/31"},
			remove:  []string{"::ffff:192.168.0.1"},
			want:    []string{"192.168.0.0/32"},
		},
		{
			name:    "split IPv6 CIDR by removing upper half",
			initial: []string{"2001:db8::/32"},
			remove:  []string{"2001:db8:8000::/33"},
			want:    []string{"2001:db8::/33"},
		},
		{
			name:    "split IPv6 CIDR by removing lower half",
			initial: []string{"2001:db8::/32"},
			remove:  []string{"2001:db8::1/33"},
			want:    []string{"2001:db8:8000::/33"},
		},
		{
			name:    "split IPv6 CIDR by removing a single IP",
			initial: []string{"2001:db8::/126"},
			remove:  []string{"2001:db8::3"},
			want:    []string{"2001:db8::/127", "2001:db8::2/128"},
		},
		{
			name:    "split both IPv4 and IPv6 CIDRs",
			initial: []string{"192.168.0.0/23", "2001:db8::/32"},
			remove:  []string{"192.168.1.0/24", "2001:db8:8000::/33"},
			want:    []string{"192.168.0.0/24", "2001:db8::/33"},
		},
		{
			name:    "remove everything",
			initial: []string{"1.1.1.0/24", "2001:db8::/32"},
			remove:  []string{"0.0.0.0/0", "::/0"},
			want:    nil,
		},
		{
			name:    "remove non-overlapping CIDRs",
			initial: []string{"1.1.1.0/24", "2001:db8::/32"},
			remove:  []string{"2.2.2.0/24", "2001:db9::/32"},
			want:    []string{"1.1.1.0/24", "2001:db8::/32"},
		},
		{
			name:    "remove IPv6 from entry without IPv6",
			initial: []string{"1.1.1.0/24"},
			remove:  []string{"2001:db8::/32"},
			want:    []string{"1.1.1.0/24"},
		},
		{
			name:    "remove IPv4 from entry without IPv4",
			initial: []string{"2001:db8::/32"},
			remove:  []string{"1.1.1.1"},
			want:    []string{"2001:db8::/32"},
		},
		{
			name:    "remove from empty entry",
			initial: nil,
			remove:  []string{"1.1.1.1", "2001:db8::1"},
			want:    nil,
		},
		{
			name:    "remove comments is a no-op",
			initial: []string{"1.1.1.0/24"},
			remove:  []string{"", "# 1.1.1.0/24", "// 1.1.1.0/24"},
			want:    []string{"1.1.1.0/24"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry := newTestEntry(t, "test", tt.initial...)
			// Build sets first to ensure removals invalidate cached sets.
			_ = entryText(t, entry)
			for _, cidr := range tt.remove {
				if err := entry.RemovePrefix(cidr); err != nil {
					t.Fatalf("RemovePrefix(%q) failed: %v", cidr, err)
				}
			}
			assertStringSlice(t, entryText(t, entry), tt.want)
		})
	}
}

func TestEntryRemovePrefixInvalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "invalid IP", input: "abc", wantErr: ErrInvalidIP},
		{name: "invalid CIDR", input: "1.1.1.0/33", wantErr: ErrInvalidCIDR},
		{name: "IPv4-mapped IPv6 CIDR with too few bits", input: "::ffff:1.1.1.0/95", wantErr: ErrInvalidCIDR},
		{name: "IPv4-mapped IPv6 CIDR with too many bits", input: "::ffff:1.1.1.0/129", wantErr: ErrInvalidCIDR},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry := newTestEntry(t, "test", "1.1.1.0/24")
			assertErrorIs(t, entry.RemovePrefix(tt.input), tt.wantErr)
			assertStringSlice(t, entryText(t, entry), []string{"1.1.1.0/24"})
		})
	}
}

func TestEntryAddRemoveOrdering(t *testing.T) {
	t.Parallel()

	entry := NewEntry("test")
	steps := []struct {
		add  bool
		cidr string
		want []string
	}{
		{add: true, cidr: "1.1.1.0/24", want: []string{"1.1.1.0/24"}},
		{add: false, cidr: "1.1.1.0/25", want: []string{"1.1.1.128/25"}},
		{add: true, cidr: "1.1.1.0/26", want: []string{"1.1.1.0/26", "1.1.1.128/25"}},
		{add: true, cidr: "1.1.1.64/26", want: []string{"1.1.1.0/24"}},
		{add: false, cidr: "0.0.0.0/0", want: nil},
		{add: true, cidr: "2001:db8::/32", want: []string{"2001:db8::/32"}},
	}

	for i, step := range steps {
		var err error
		if step.add {
			err = entry.AddPrefix(step.cidr)
		} else {
			err = entry.RemovePrefix(step.cidr)
		}
		if err != nil {
			t.Fatalf("step %d: unexpected error: %v", i, err)
		}
		assertStringSlice(t, entryText(t, entry), step.want)
	}
}

func TestEntryAddAndRemoveInvalidIPType(t *testing.T) {
	t.Parallel()

	prefix := netip.MustParsePrefix("1.1.1.0/24")
	tests := []struct {
		name string
		fn   func(e *Entry) error
	}{
		{name: "add", fn: func(e *Entry) error { return e.add(&prefix, IPType("ipv5")) }},
		{name: "remove", fn: func(e *Entry) error { return e.remove(&prefix, IPType("")) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry := NewEntry("test")
			assertErrorIs(t, tt.fn(entry), ErrInvalidIPType)
			if entry.hasIPv4Builder() || entry.hasIPv6Builder() {
				t.Fatal("entry should not have any builder")
			}
		})
	}
}

func TestEntryGetIPSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cidrs    []string
		poison4  bool
		poison6  bool
		want4    []string
		want6    []string
		wantErr4 string
		wantErr6 string
	}{
		{
			name:  "IPv4 and IPv6",
			cidrs: []string{"192.168.0.3/24", "192.168.1.5/24", "2001:db8::/33", "2001:db8:8000::/33"},
			want4: []string{"192.168.0.0/23"},
			want6: []string{"2001:db8::/32"},
		},
		{
			name:     "only IPv4",
			cidrs:    []string{"1.1.1.0/24"},
			want4:    []string{"1.1.1.0/24"},
			wantErr6: "entry TEST has no ipv6 set",
		},
		{
			name:     "only IPv6",
			cidrs:    []string{"2001:db8::/32"},
			wantErr4: "entry TEST has no ipv4 set",
			want6:    []string{"2001:db8::/32"},
		},
		{
			name:     "empty entry",
			wantErr4: "entry TEST has no ipv4 set",
			wantErr6: "entry TEST has no ipv6 set",
		},
		{
			// Both GetIPv4Set and GetIPv6Set build all sets first, so an
			// error of either builder is returned by both of them.
			name:     "IPv4 builder error",
			cidrs:    []string{"1.1.1.0/24", "2001:db8::/32"},
			poison4:  true,
			wantErr4: "AddPrefix(invalid IP/-1)",
			wantErr6: "AddPrefix(invalid IP/-1)",
		},
		{
			name:     "IPv6 builder error",
			cidrs:    []string{"1.1.1.0/24", "2001:db8::/32"},
			poison6:  true,
			wantErr4: "AddPrefix(invalid IP/-1)",
			wantErr6: "AddPrefix(invalid IP/-1)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry := newTestEntry(t, "test", tt.cidrs...)
			if tt.poison4 {
				poisonIPv4Builder(entry)
			}
			if tt.poison6 {
				poisonIPv6Builder(entry)
			}

			set4, err := entry.GetIPv4Set()
			if tt.wantErr4 != "" {
				assertErrorContains(t, err, tt.wantErr4)
				if set4 != nil {
					t.Fatal("GetIPv4Set() should return nil set on error")
				}
			} else {
				if err != nil {
					t.Fatalf("GetIPv4Set() unexpected error: %v", err)
				}
				assertPrefixSlice(t, set4.Prefixes(), tt.want4)
			}

			// netipx.IPSetBuilder resets its errors after IPSet() is called,
			// so poison the builders again.
			if tt.poison4 {
				poisonIPv4Builder(entry)
			}
			if tt.poison6 {
				poisonIPv6Builder(entry)
			}

			set6, err := entry.GetIPv6Set()
			if tt.wantErr6 != "" {
				assertErrorContains(t, err, tt.wantErr6)
				if set6 != nil {
					t.Fatal("GetIPv6Set() should return nil set on error")
				}
			} else {
				if err != nil {
					t.Fatalf("GetIPv6Set() unexpected error: %v", err)
				}
				assertPrefixSlice(t, set6.Prefixes(), tt.want6)
			}
		})
	}
}

func TestEntryGetIPSetCache(t *testing.T) {
	t.Parallel()

	entry := newTestEntry(t, "test", "1.1.1.0/24", "2001:db8::/33")

	set4, err := entry.GetIPv4Set()
	if err != nil {
		t.Fatal(err)
	}
	set6, err := entry.GetIPv6Set()
	if err != nil {
		t.Fatal(err)
	}

	// Without changes the cached sets are returned.
	again4, _ := entry.GetIPv4Set()
	again6, _ := entry.GetIPv6Set()
	if again4 != set4 || again6 != set6 {
		t.Fatal("cached sets should be returned when the entry is unchanged")
	}

	// Adding prefixes invalidates the caches and the old sets are untouched.
	if err := entry.AddPrefix("1.1.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := entry.AddPrefix("2001:db8:8000::/33"); err != nil {
		t.Fatal(err)
	}
	new4, _ := entry.GetIPv4Set()
	new6, _ := entry.GetIPv6Set()
	assertPrefixSlice(t, set4.Prefixes(), []string{"1.1.1.0/24"})
	assertPrefixSlice(t, set6.Prefixes(), []string{"2001:db8::/33"})
	assertPrefixSlice(t, new4.Prefixes(), []string{"1.1.0.0/23"})
	assertPrefixSlice(t, new6.Prefixes(), []string{"2001:db8::/32"})

	// Removing prefixes invalidates the caches too.
	if err := entry.RemovePrefix("1.1.1.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := entry.RemovePrefix("2001:db8::/33"); err != nil {
		t.Fatal(err)
	}
	removed4, _ := entry.GetIPv4Set()
	removed6, _ := entry.GetIPv6Set()
	assertPrefixSlice(t, new4.Prefixes(), []string{"1.1.0.0/23"})
	assertPrefixSlice(t, new6.Prefixes(), []string{"2001:db8::/32"})
	assertPrefixSlice(t, removed4.Prefixes(), []string{"1.1.0.0/24"})
	assertPrefixSlice(t, removed6.Prefixes(), []string{"2001:db8:8000::/33"})
}

// marshalTestCases are shared by MarshalPrefix, MarshalIPRange and MarshalText.
var marshalTestCases = []struct {
	name       string
	cidrs      []string
	opts       []IgnoreIPOption
	poison4    bool
	poison6    bool
	wantText   []string
	wantRanges []string
	wantErr    string
}{
	{
		name:       "IPv4 and IPv6 without options",
		cidrs:      []string{"2001:db8::/33", "192.168.0.3/24", "2001:db8:8000::/33", "192.168.1.5/24", "10.0.0.1"},
		wantText:   []string{"10.0.0.1/32", "192.168.0.0/23", "2001:db8::/32"},
		wantRanges: []string{"10.0.0.1-10.0.0.1", "192.168.0.0-192.168.1.255", "2001:db8::-2001:db8:ffff:ffff:ffff:ffff:ffff:ffff"},
	},
	{
		name:       "unaligned range",
		cidrs:      []string{"1.1.1.1", "1.1.1.2", "1.1.1.3"},
		wantText:   []string{"1.1.1.1/32", "1.1.1.2/31"},
		wantRanges: []string{"1.1.1.1-1.1.1.3"},
	},
	{
		name:       "nil option",
		cidrs:      []string{"1.1.1.0/24", "2001:db8::/32"},
		opts:       []IgnoreIPOption{nil},
		wantText:   []string{"1.1.1.0/24", "2001:db8::/32"},
		wantRanges: []string{"1.1.1.0-1.1.1.255", "2001:db8::-2001:db8:ffff:ffff:ffff:ffff:ffff:ffff"},
	},
	{
		name:       "ignore IPv4",
		cidrs:      []string{"1.1.1.0/24", "2001:db8::/32"},
		opts:       []IgnoreIPOption{IgnoreIPv4},
		wantText:   []string{"2001:db8::/32"},
		wantRanges: []string{"2001:db8::-2001:db8:ffff:ffff:ffff:ffff:ffff:ffff"},
	},
	{
		name:       "ignore IPv6",
		cidrs:      []string{"1.1.1.0/24", "2001:db8::/32"},
		opts:       []IgnoreIPOption{IgnoreIPv6},
		wantText:   []string{"1.1.1.0/24"},
		wantRanges: []string{"1.1.1.0-1.1.1.255"},
	},
	{
		name:       "last non-nil option wins",
		cidrs:      []string{"1.1.1.0/24", "2001:db8::/32"},
		opts:       []IgnoreIPOption{IgnoreIPv4, IgnoreIPv6, nil},
		wantText:   []string{"1.1.1.0/24"},
		wantRanges: []string{"1.1.1.0-1.1.1.255"},
	},
	{
		name:       "option from GetIgnoreIPType",
		cidrs:      []string{"1.1.1.0/24", "2001:db8::/32"},
		opts:       []IgnoreIPOption{GetIgnoreIPType(IPv6)},
		wantText:   []string{"2001:db8::/32"},
		wantRanges: []string{"2001:db8::-2001:db8:ffff:ffff:ffff:ffff:ffff:ffff"},
	},
	{
		name:    "ignore IPv4 on IPv4-only entry",
		cidrs:   []string{"1.1.1.0/24"},
		opts:    []IgnoreIPOption{IgnoreIPv4},
		wantErr: "entry TEST has no prefix",
	},
	{
		name:    "ignore IPv6 on IPv6-only entry",
		cidrs:   []string{"2001:db8::/32"},
		opts:    []IgnoreIPOption{IgnoreIPv6},
		wantErr: "entry TEST has no prefix",
	},
	{
		name:    "empty entry",
		wantErr: "entry TEST has no prefix",
	},
	{
		name:    "IPv4 builder error",
		cidrs:   []string{"1.1.1.0/24"},
		poison4: true,
		wantErr: "AddPrefix(invalid IP/-1)",
	},
	{
		name:    "IPv6 builder error",
		cidrs:   []string{"2001:db8::/32"},
		poison6: true,
		wantErr: "AddPrefix(invalid IP/-1)",
	},
}

func newMarshalTestEntry(t *testing.T, cidrs []string, poison4, poison6 bool) *Entry {
	t.Helper()
	entry := newTestEntry(t, "test", cidrs...)
	if poison4 {
		poisonIPv4Builder(entry)
	}
	if poison6 {
		poisonIPv6Builder(entry)
	}
	return entry
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want containing %q", err, want)
	}
}

func TestEntryMarshalPrefix(t *testing.T) {
	t.Parallel()

	for _, tt := range marshalTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry := newMarshalTestEntry(t, tt.cidrs, tt.poison4, tt.poison6)
			got, err := entry.MarshalPrefix(tt.opts...)
			if tt.wantErr != "" {
				assertErrorContains(t, err, tt.wantErr)
				if got != nil {
					t.Fatalf("got %v, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertPrefixSlice(t, got, tt.wantText)
		})
	}
}

func TestEntryMarshalIPRange(t *testing.T) {
	t.Parallel()

	for _, tt := range marshalTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry := newMarshalTestEntry(t, tt.cidrs, tt.poison4, tt.poison6)
			got, err := entry.MarshalIPRange(tt.opts...)
			if tt.wantErr != "" {
				assertErrorContains(t, err, tt.wantErr)
				if got != nil {
					t.Fatalf("got %v, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertIPRangeSlice(t, got, tt.wantRanges)
		})
	}
}

func TestEntryMarshalText(t *testing.T) {
	t.Parallel()

	for _, tt := range marshalTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry := newMarshalTestEntry(t, tt.cidrs, tt.poison4, tt.poison6)
			got, err := entry.MarshalText(tt.opts...)
			if tt.wantErr != "" {
				assertErrorContains(t, err, tt.wantErr)
				if got != nil {
					t.Fatalf("got %v, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertStringSlice(t, got, tt.wantText)
		})
	}
}
