package lib

import (
	"net"
	"net/netip"
	"testing"

	"go4.org/netipx"
)

func ptr[T any](v T) *T { return &v }

func TestNewEntry(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"cn", "CN"},
		{"  Private \t", "PRIVATE"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			e := NewEntry(tt.input)
			if got := e.GetName(); got != tt.want {
				t.Errorf("GetName() = %q, want %q", got, tt.want)
			}
			if e.hasIPv4Builder() || e.hasIPv6Builder() || e.hasIPv4Set() || e.hasIPv6Set() {
				t.Error("new entry should be empty")
			}
		})
	}
}

func TestEntryProcessPrefix(t *testing.T) {
	v4 := netip.MustParseAddr("1.2.3.4")
	v6 := netip.MustParseAddr("2001:db8::1")
	mapped := netip.MustParseAddr("::ffff:1.2.3.4")

	tests := []struct {
		name     string
		input    any
		want     string
		wantType IPType
		wantErr  error
	}{
		// string: IPv4
		{"string ipv4", "1.2.3.4", "1.2.3.4/32", IPv4, nil},
		{"string ipv4 cidr masked", "1.2.3.4/24", "1.2.3.0/24", IPv4, nil},
		{"string ipv4 /0", "1.2.3.4/0", "0.0.0.0/0", IPv4, nil},
		{"string ipv4 /32", "1.2.3.4/32", "1.2.3.4/32", IPv4, nil},
		{"string ipv4 with spaces and comment", "  1.2.3.4  # comment", "1.2.3.4/32", IPv4, nil},
		{"string ipv4 cidr with // comment", "10.0.0.0/8 // private", "10.0.0.0/8", IPv4, nil},
		{"string ipv4 with /* comment", "10.0.0.1 /* c */", "10.0.0.1/32", IPv4, nil},
		{"string ipv4 bits too large", "1.2.3.4/33", "", "", ErrInvalidCIDR},
		{"string ipv4 negative bits", "1.2.3.4/-1", "", "", ErrInvalidCIDR},
		{"string ipv4 non-numeric bits", "1.2.3.4/abc", "", "", ErrInvalidCIDR},
		{"string malformed ipv4", "1.2.3", "", "", ErrInvalidIP},
		{"string malformed ipv4 octet", "256.1.1.1", "", "", ErrInvalidIP},
		{"string malformed ipv4 cidr", "256.1.1.1/24", "", "", ErrInvalidCIDR},
		// string: IPv6
		{"string ipv6", "2001:db8::1", "2001:db8::1/128", IPv6, nil},
		{"string ipv6 cidr masked", "2001:db8::1/32", "2001:db8::/32", IPv6, nil},
		{"string ipv6 /0", "2001:db8::1/0", "::/0", IPv6, nil},
		{"string ipv6 bits too large", "2001:db8::/129", "", "", ErrInvalidCIDR},
		{"string malformed ipv6", "2001:db8:::1", "", "", ErrInvalidIP},
		{"string malformed ipv6 cidr", "2001:zz8::/32", "", "", ErrInvalidCIDR},
		// string: IPv4-mapped IPv6
		{"string mapped ip", "::ffff:1.2.3.4", "1.2.3.4/32", IPv4, nil},
		{"string mapped ip hex form", "::ffff:102:304", "1.2.3.4/32", IPv4, nil},
		{"string mapped cidr /128", "::ffff:1.2.3.4/128", "1.2.3.4/32", IPv4, nil},
		{"string mapped cidr /120", "::ffff:1.2.3.4/120", "1.2.3.0/24", IPv4, nil},
		{"string mapped cidr /104", "::ffff:1.2.3.4/104", "1.0.0.0/8", IPv4, nil},
		{"string mapped cidr /96", "::ffff:1.2.3.4/96", "0.0.0.0/0", IPv4, nil},
		{"string mapped cidr /95 too small", "::ffff:1.2.3.4/95", "", "", ErrInvalidPrefix},
		{"string mapped cidr /0 too small", "::ffff:1.2.3.4/0", "", "", ErrInvalidPrefix},
		{"string mapped cidr /129 too large", "::ffff:1.2.3.4/129", "", "", ErrInvalidCIDR},
		{"string malformed mapped ip", "::ffff:1.2.3", "", "", ErrInvalidIP},
		{"string malformed mapped ip octet", "::ffff:1.2.3.256", "", "", ErrInvalidIP},
		{"string malformed mapped cidr", "::ffff:1.2.3/120", "", "", ErrInvalidCIDR},
		// string: comment lines
		{"string empty", "", "", "", ErrCommentLine},
		{"string spaces", "   ", "", "", ErrCommentLine},
		{"string hash comment", "# 1.1.1.1", "", "", ErrCommentLine},
		{"string slash comment", "// 1.1.1.1", "", "", ErrCommentLine},
		{"string block comment", "/* 1.1.1.1 */", "", "", ErrCommentLine},
		// *string
		{"*string valid", ptr("::ffff:10.0.0.1/104"), "10.0.0.0/8", IPv4, nil},
		{"*string nil", (*string)(nil), "", "", ErrInvalidPrefixType},
		// net.IP
		{"net.IP ipv4 4-byte", net.IPv4(1, 2, 3, 4).To4(), "1.2.3.4/32", IPv4, nil},
		{"net.IP ipv4 16-byte mapped", net.ParseIP("1.2.3.4"), "1.2.3.4/32", IPv4, nil},
		{"net.IP explicit mapped", net.ParseIP("::ffff:5.6.7.8"), "5.6.7.8/32", IPv4, nil},
		{"net.IP ipv6", net.ParseIP("2001:db8::1"), "2001:db8::1/128", IPv6, nil},
		{"net.IP malformed length", net.IP{1, 2, 3}, "", "", ErrInvalidIP},
		{"net.IP nil", net.IP(nil), "", "", ErrInvalidIP},
		{"*net.IP valid", ptr(net.ParseIP("2001:db8::2")), "2001:db8::2/128", IPv6, nil},
		{"*net.IP nil", (*net.IP)(nil), "", "", ErrInvalidPrefixType},
		// net.IPNet
		{"net.IPNet ipv4", net.IPNet{IP: net.IP{1, 2, 3, 4}, Mask: net.CIDRMask(24, 32)}, "1.2.3.0/24", IPv4, nil},
		{"net.IPNet ipv4 16-byte ip 32-bit mask", net.IPNet{IP: net.ParseIP("1.2.3.4"), Mask: net.CIDRMask(16, 32)}, "1.2.0.0/16", IPv4, nil},
		{"net.IPNet mapped /120", net.IPNet{IP: net.ParseIP("::ffff:1.2.3.4"), Mask: net.CIDRMask(120, 128)}, "1.2.3.0/24", IPv4, nil},
		{"net.IPNet mapped /96", net.IPNet{IP: net.ParseIP("::ffff:1.2.3.4"), Mask: net.CIDRMask(96, 128)}, "0.0.0.0/0", IPv4, nil},
		{"net.IPNet mapped /95 too small", net.IPNet{IP: net.ParseIP("::ffff:1.2.3.4"), Mask: net.CIDRMask(95, 128)}, "", "", ErrInvalidPrefix},
		{"net.IPNet mapped /0 too small", net.IPNet{IP: net.ParseIP("::ffff:1.2.3.4"), Mask: net.CIDRMask(0, 128)}, "", "", ErrInvalidPrefix},
		{"net.IPNet ipv6", net.IPNet{IP: net.ParseIP("2001:db8::1"), Mask: net.CIDRMask(48, 128)}, "2001:db8::/48", IPv6, nil},
		{"net.IPNet ipv4 ip with 128-bit mask", net.IPNet{IP: net.IP{1, 2, 3, 4}, Mask: net.CIDRMask(64, 128)}, "", "", ErrInvalidIPNet},
		{"net.IPNet ipv6 ip with 32-bit mask", net.IPNet{IP: net.ParseIP("2001:db8::1"), Mask: net.CIDRMask(24, 32)}, "", "", ErrInvalidIPNet},
		{"net.IPNet non-canonical mask", net.IPNet{IP: net.IP{1, 2, 3, 4}, Mask: net.IPMask{255, 0, 255, 0}}, "", "", ErrInvalidIPNet},
		{"net.IPNet malformed ip", net.IPNet{IP: net.IP{1, 2}, Mask: net.CIDRMask(8, 32)}, "", "", ErrInvalidIPNet},
		{"*net.IPNet valid", &net.IPNet{IP: net.IP{10, 1, 2, 3}, Mask: net.CIDRMask(8, 32)}, "10.0.0.0/8", IPv4, nil},
		{"*net.IPNet nil", (*net.IPNet)(nil), "", "", ErrInvalidPrefixType},
		// netip.Addr
		{"netip.Addr ipv4", v4, "1.2.3.4/32", IPv4, nil},
		{"netip.Addr ipv6", v6, "2001:db8::1/128", IPv6, nil},
		{"netip.Addr mapped", mapped, "1.2.3.4/32", IPv4, nil},
		{"netip.Addr zero", netip.Addr{}, "", "", ErrInvalidIPType},
		{"*netip.Addr valid", &v6, "2001:db8::1/128", IPv6, nil},
		{"*netip.Addr nil", (*netip.Addr)(nil), "", "", ErrInvalidPrefixType},
		// netip.Prefix
		{"netip.Prefix ipv4", netip.PrefixFrom(v4, 16), "1.2.0.0/16", IPv4, nil},
		{"netip.Prefix ipv4 invalid bits", netip.PrefixFrom(v4, 33), "", "", ErrInvalidPrefix},
		{"netip.Prefix ipv6", netip.PrefixFrom(v6, 64), "2001:db8::/64", IPv6, nil},
		{"netip.Prefix ipv6 invalid bits", netip.PrefixFrom(v6, 129), "", "", ErrInvalidPrefix},
		{"netip.Prefix mapped /128", netip.PrefixFrom(mapped, 128), "1.2.3.4/32", IPv4, nil},
		{"netip.Prefix mapped /112", netip.PrefixFrom(mapped, 112), "1.2.0.0/16", IPv4, nil},
		{"netip.Prefix mapped /96", netip.PrefixFrom(mapped, 96), "0.0.0.0/0", IPv4, nil},
		{"netip.Prefix mapped /95 too small", netip.PrefixFrom(mapped, 95), "", "", ErrInvalidPrefix},
		{"netip.Prefix mapped invalid bits", netip.PrefixFrom(mapped, 129), "", "", ErrInvalidPrefix},
		{"netip.Prefix zero", netip.Prefix{}, "", "", ErrInvalidIPType},
		{"*netip.Prefix valid", ptr(netip.MustParsePrefix("::ffff:192.168.1.1/120")), "192.168.1.0/24", IPv4, nil},
		{"*netip.Prefix nil", (*netip.Prefix)(nil), "", "", ErrInvalidPrefixType},
		// unsupported
		{"unsupported int", 123, "", "", ErrInvalidPrefixType},
		{"unsupported nil", nil, "", "", ErrInvalidPrefixType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEntry("test")
			got, gotType, err := e.processPrefix(tt.input)
			assertErr(t, err, tt.wantErr, tt.wantErr != nil)
			if tt.wantErr != nil {
				if got != nil || gotType != "" {
					t.Fatalf("expected nil prefix and empty type on error, got %v %q", got, gotType)
				}
				return
			}
			if got.String() != tt.want {
				t.Errorf("prefix = %s, want %s", got, tt.want)
			}
			if gotType != tt.wantType {
				t.Errorf("type = %q, want %q", gotType, tt.wantType)
			}
		})
	}
}

func TestEntryAddRemovePrefixMergeSplit(t *testing.T) {
	tests := []struct {
		name   string
		add    []any
		remove []string
		want   []string
	}{
		{
			name: "ipv4 merge two /24 into /23",
			add:  []any{"192.168.0.3/24", "192.168.1.5/24"},
			want: []string{"192.168.0.0/23"},
		},
		{
			name:   "ipv4 split /23 into /24",
			add:    []any{"192.168.0.0/23"},
			remove: []string{"192.168.1.5/24"},
			want:   []string{"192.168.0.0/24"},
		},
		{
			name:   "ipv4 split /23 into multiple prefixes",
			add:    []any{"192.168.0.0/23"},
			remove: []string{"192.168.0.128/25"},
			want:   []string{"192.168.0.0/25", "192.168.1.0/24"},
		},
		{
			name:   "ipv4 remove single ip",
			add:    []any{"10.0.0.0/30"},
			remove: []string{"10.0.0.0"},
			want:   []string{"10.0.0.1/32", "10.0.0.2/31"},
		},
		{
			name: "ipv4 merge four /26 and duplicates",
			add:  []any{"10.0.0.0/26", "10.0.0.64/26", "10.0.0.128/26", "10.0.0.192/26", "10.0.0.5"},
			want: []string{"10.0.0.0/24"},
		},
		{
			name: "ipv4 non adjacent stay separate",
			add:  []any{"192.168.1.0/24", "192.168.2.0/24"},
			want: []string{"192.168.1.0/24", "192.168.2.0/24"},
		},
		{
			name: "ipv4 mapped ipv6 merges with ipv4",
			add:  []any{"::ffff:192.168.0.3/120", netip.MustParsePrefix("192.168.1.5/24")},
			want: []string{"192.168.0.0/23"},
		},
		{
			name:   "ipv4 mapped ipv6 removal splits ipv4",
			add:    []any{"192.168.0.0/23"},
			remove: []string{"::ffff:192.168.1.0/120"},
			want:   []string{"192.168.0.0/24"},
		},
		{
			name: "ipv6 merge two /33 into /32",
			add:  []any{"2001:db8::1/33", "2001:db8:8000::1/33"},
			want: []string{"2001:db8::/32"},
		},
		{
			name:   "ipv6 split /32 into /33",
			add:    []any{"2001:db8::/32"},
			remove: []string{"2001:db8:8000::/33"},
			want:   []string{"2001:db8::/33"},
		},
		{
			name:   "ipv6 split /32 into multiple prefixes",
			add:    []any{"2001:db8::/32"},
			remove: []string{"2001:db8::/34"},
			want:   []string{"2001:db8:4000::/34", "2001:db8:8000::/33"},
		},
		{
			name: "mixed ipv4 and ipv6, ipv4 first",
			add:  []any{"2001:db8::/32", "1.0.0.0/24", "1.0.1.0/24"},
			want: []string{"1.0.0.0/23", "2001:db8::/32"},
		},
		{
			name:   "remove all",
			add:    []any{"1.1.1.1", "2001:db8::1"},
			remove: []string{"1.1.1.1/32", "2001:db8::1/128"},
			want:   []string{},
		},
		{
			name:   "remove from entry without builders is no-op",
			remove: []string{"1.1.1.1", "2001:db8::1"},
			want:   []string{},
		},
		{
			name:   "comment lines ignored",
			add:    []any{"# comment", "1.1.1.1", ""},
			remove: []string{"// comment"},
			want:   []string{"1.1.1.1/32"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEntry("test")
			for _, p := range tt.add {
				if err := e.AddPrefix(p); err != nil {
					t.Fatalf("AddPrefix(%v) error: %v", p, err)
				}
			}
			for _, p := range tt.remove {
				if err := e.RemovePrefix(p); err != nil {
					t.Fatalf("RemovePrefix(%v) error: %v", p, err)
				}
			}
			got, err := e.MarshalText()
			if len(tt.want) == 0 {
				if err == nil {
					t.Fatalf("expected error for empty entry, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			assertStrings(t, got, tt.want)
		})
	}
}

func TestEntryAddRemovePrefixErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"malformed ip", "1.1.1", ErrInvalidIP},
		{"malformed cidr", "1.1.1.1/40", ErrInvalidCIDR},
		{"mapped prefix bits too small", "::ffff:1.1.1.1/64", ErrInvalidPrefix},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEntry("test")
			assertErr(t, e.AddPrefix(tt.input), tt.wantErr, true)
			assertErr(t, e.RemovePrefix(tt.input), tt.wantErr, true)
			if e.hasIPv4Builder() || e.hasIPv6Builder() {
				t.Error("builders should not be created on error")
			}
		})
	}
	t.Run("unsupported type", func(t *testing.T) {
		assertErr(t, NewEntry("t").AddPrefix(1), ErrInvalidPrefixType, true)
	})
}

func TestEntryAddRemoveInvalidIPType(t *testing.T) {
	e := NewEntry("test")
	p := netip.MustParsePrefix("1.1.1.0/24")
	assertErr(t, e.add(&p, IPType("bad")), ErrInvalidIPType, true)
	assertErr(t, e.remove(&p, IPType("bad")), ErrInvalidIPType, true)
}

func TestEntrySetCacheInvalidation(t *testing.T) {
	e := NewEntry("test")
	_ = e.AddPrefix("1.1.1.0/24")
	_ = e.AddPrefix("2001:db8::/32")
	if _, err := e.GetIPv4Set(); err != nil {
		t.Fatal(err)
	}
	if _, err := e.GetIPv6Set(); err != nil {
		t.Fatal(err)
	}
	if !e.hasIPv4Set() || !e.hasIPv6Set() {
		t.Fatal("sets should be cached")
	}
	_ = e.AddPrefix("1.1.2.0/24")
	_ = e.RemovePrefix("2001:db8::/33")
	if e.hasIPv4Set() || e.hasIPv6Set() {
		t.Fatal("sets should be invalidated after modification")
	}
	got, err := e.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	assertStrings(t, got, []string{"1.1.1.0/24", "1.1.2.0/24", "2001:db8:8000::/33"})
}

func TestEntryGetIPSet(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Entry)
		want4 []string
		want6 []string
		err4  bool
		err6  bool
	}{
		{"empty", func(*Entry) {}, nil, nil, true, true},
		{"ipv4 only", func(e *Entry) { _ = e.AddPrefix("1.1.1.1/24") }, []string{"1.1.1.0/24"}, nil, false, true},
		{"ipv6 only", func(e *Entry) { _ = e.AddPrefix("2001:db8::1/64") }, nil, []string{"2001:db8::/64"}, true, false},
		{"both", func(e *Entry) {
			_ = e.AddPrefix("::ffff:1.1.1.1")
			_ = e.AddPrefix("2001:db8::1")
		}, []string{"1.1.1.1/32"}, []string{"2001:db8::1/128"}, false, false},
		{"broken ipv4 builder", func(e *Entry) { e.ipv4Builder = brokenBuilder() }, nil, nil, true, true},
		{"broken ipv6 builder", func(e *Entry) { e.ipv6Builder = brokenBuilder() }, nil, nil, true, true},
	}
	toStrings := func(s *netipx.IPSet) []string {
		out := []string{}
		for _, p := range s.Prefixes() {
			out = append(out, p.String())
		}
		return out
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEntry("test")
			tt.setup(e)
			s4, err := e.GetIPv4Set()
			assertErr(t, err, nil, tt.err4)
			if !tt.err4 {
				assertStrings(t, toStrings(s4), tt.want4)
			}
			// IPSetBuilder errors are consumed by IPSet(), so rebuild the entry.
			e = NewEntry("test")
			tt.setup(e)
			s6, err := e.GetIPv6Set()
			assertErr(t, err, nil, tt.err6)
			if !tt.err6 {
				assertStrings(t, toStrings(s6), tt.want6)
			}
		})
	}
}

func TestEntryMarshal(t *testing.T) {
	full := func(e *Entry) {
		for _, p := range []string{"192.168.0.3/24", "192.168.1.5/24", "10.0.0.1", "2001:db8::/33", "2001:db8:8000::/33"} {
			_ = e.AddPrefix(p)
		}
	}
	tests := []struct {
		name       string
		setup      func(*Entry)
		opts       []IgnoreIPOption
		wantText   []string
		wantRanges []string
		wantErr    bool
	}{
		{
			name:       "all",
			setup:      full,
			wantText:   []string{"10.0.0.1/32", "192.168.0.0/23", "2001:db8::/32"},
			wantRanges: []string{"10.0.0.1-10.0.0.1", "192.168.0.0-192.168.1.255", "2001:db8::-2001:db8:ffff:ffff:ffff:ffff:ffff:ffff"},
		},
		{
			name:       "nil option ignored",
			setup:      full,
			opts:       []IgnoreIPOption{nil},
			wantText:   []string{"10.0.0.1/32", "192.168.0.0/23", "2001:db8::/32"},
			wantRanges: []string{"10.0.0.1-10.0.0.1", "192.168.0.0-192.168.1.255", "2001:db8::-2001:db8:ffff:ffff:ffff:ffff:ffff:ffff"},
		},
		{
			name:       "ignore ipv4",
			setup:      full,
			opts:       []IgnoreIPOption{IgnoreIPv4},
			wantText:   []string{"2001:db8::/32"},
			wantRanges: []string{"2001:db8::-2001:db8:ffff:ffff:ffff:ffff:ffff:ffff"},
		},
		{
			name:       "ignore ipv6",
			setup:      full,
			opts:       []IgnoreIPOption{IgnoreIPv4, IgnoreIPv6},
			wantText:   []string{"10.0.0.1/32", "192.168.0.0/23"},
			wantRanges: []string{"10.0.0.1-10.0.0.1", "192.168.0.0-192.168.1.255"},
		},
		{name: "empty entry", setup: func(*Entry) {}, wantErr: true},
		{name: "ignore only present type", setup: func(e *Entry) { _ = e.AddPrefix("1.1.1.1") }, opts: []IgnoreIPOption{IgnoreIPv4}, wantErr: true},
		{name: "broken builder", setup: func(e *Entry) { e.ipv4Builder = brokenBuilder() }, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEntry("test")
			tt.setup(e)

			text, err := e.MarshalText(tt.opts...)
			assertErr(t, err, nil, tt.wantErr)
			if !tt.wantErr {
				assertStrings(t, text, tt.wantText)
			}

			// IPSetBuilder errors are consumed by IPSet(), so rebuild the entry.
			e = NewEntry("test")
			tt.setup(e)
			prefixes, err := e.MarshalPrefix(tt.opts...)
			assertErr(t, err, nil, tt.wantErr)
			if !tt.wantErr {
				got := make([]string, 0, len(prefixes))
				for _, p := range prefixes {
					got = append(got, p.String())
				}
				assertStrings(t, got, tt.wantText)
			}

			e = NewEntry("test")
			tt.setup(e)
			ranges, err := e.MarshalIPRange(tt.opts...)
			assertErr(t, err, nil, tt.wantErr)
			if !tt.wantErr {
				got := make([]string, 0, len(ranges))
				for _, r := range ranges {
					got = append(got, r.String())
				}
				assertStrings(t, got, tt.wantRanges)
			}
		})
	}
}
