package lib

import (
	"errors"
	"net"
	"net/netip"
	"testing"

	"go4.org/netipx"
)

func mustParseCIDR(t *testing.T, s string) *net.IPNet {
	t.Helper()
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		t.Fatalf("ParseCIDR(%q): %v", s, err)
	}
	return n
}

func compareStrings(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNewEntryAndGetName(t *testing.T) {
	tests := []struct{ in, want string }{
		{"cn", "CN"},
		{"  private\t", "PRIVATE"},
		{"", ""},
		{"Mixed-Case_1", "MIXED-CASE_1"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := NewEntry(tt.in).GetName(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEntryGetPrefixFromString(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		want     string
		wantType IPType
		wantErr  error
	}{
		{"ipv4", "1.1.1.1", "1.1.1.1/32", IPv4, nil},
		{"ipv4 with spaces", "  1.1.1.1  ", "1.1.1.1/32", IPv4, nil},
		{"ipv4 cidr masked", "192.168.0.3/24", "192.168.0.0/24", IPv4, nil},
		{"ipv4 cidr /0", "1.2.3.4/0", "0.0.0.0/0", IPv4, nil},
		{"ipv4 cidr /32", "1.2.3.4/32", "1.2.3.4/32", IPv4, nil},
		{"ipv4 invalid bits", "1.2.3.4/33", "", "", ErrInvalidCIDR},
		{"ipv4 negative bits", "1.2.3.4/-1", "", "", ErrInvalidCIDR},
		{"ipv4 invalid", "256.1.1.1", "", "", ErrInvalidIP},
		{"ipv4 invalid cidr", "256.1.1.1/24", "", "", ErrInvalidCIDR},
		{"ipv6", "2001:db8::1", "2001:db8::1/128", IPv6, nil},
		{"ipv6 cidr masked", "2001:db8::1/32", "2001:db8::/32", IPv6, nil},
		{"ipv6 cidr /0", "2001:db8::1/0", "::/0", IPv6, nil},
		{"ipv6 cidr /128", "2001:db8::1/128", "2001:db8::1/128", IPv6, nil},
		{"ipv6 invalid bits", "2001:db8::1/129", "", "", ErrInvalidCIDR},
		{"ipv6 invalid", "2001:db8:::1", "", "", ErrInvalidIP},
		{"ipv4-mapped ipv6 addr", "::ffff:1.2.3.4", "1.2.3.4/32", IPv4, nil},
		{"ipv4-mapped ipv6 addr hex", "::ffff:102:304", "1.2.3.4/32", IPv4, nil},
		{"ipv4-mapped ipv6 cidr /128", "::ffff:1.2.3.4/128", "1.2.3.4/32", IPv4, nil},
		{"ipv4-mapped ipv6 cidr /120", "::ffff:192.168.0.3/120", "192.168.0.0/24", IPv4, nil},
		{"ipv4-mapped ipv6 cidr /104", "::ffff:10.1.2.3/104", "10.0.0.0/8", IPv4, nil},
		{"ipv4-mapped ipv6 cidr /96", "::ffff:1.2.3.4/96", "0.0.0.0/0", IPv4, nil},
		{"ipv4-mapped ipv6 cidr /95", "::ffff:1.2.3.4/95", "", "", ErrInvalidCIDR},
		{"ipv4-mapped ipv6 cidr /64", "::ffff:1.2.3.4/64", "", "", ErrInvalidCIDR},
		{"ipv4-mapped ipv6 cidr /129", "::ffff:1.2.3.4/129", "", "", ErrInvalidCIDR},
		{"comment hash", "# comment", "", "", ErrCommentLine},
		{"comment slashes", "// comment", "", "", ErrCommentLine},
		{"comment block", "/* comment", "", "", ErrCommentLine},
		{"trailing comment hash", "1.1.1.1 # one", "1.1.1.1/32", IPv4, nil},
		{"trailing comment slashes", "1.1.1.0/24 // one", "1.1.1.0/24", IPv4, nil},
		{"trailing comment block", "2001:db8::/32 /* x */", "2001:db8::/32", IPv6, nil},
		{"empty", "", "", "", ErrCommentLine},
		{"blank", "   ", "", "", ErrCommentLine},
		{"garbage", "abc", "", "", ErrInvalidIP},
		{"garbage cidr", "abc/24", "", "", ErrInvalidCIDR},
	}
	e := NewEntry("test")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, typ, err := e.getPrefixFromString(tt.in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if p != nil || typ != "" {
					t.Errorf("expected nil prefix and empty type, got %v %q", p, typ)
				}
				return
			}
			if p.String() != tt.want || typ != tt.wantType {
				t.Errorf("got %s %s, want %s %s", p, typ, tt.want, tt.wantType)
			}
		})
	}
}

func TestEntryProcessPrefix(t *testing.T) {
	ip4 := net.ParseIP("1.2.3.4")
	ip6 := net.ParseIP("2001:db8::1")
	ipnet4 := mustParseCIDR(t, "192.168.0.3/24")
	ipnet6 := mustParseCIDR(t, "2001:db8::1/48")
	addr4 := netip.MustParseAddr("1.2.3.4")
	addr6 := netip.MustParseAddr("2001:db8::1")
	addrMapped := netip.MustParseAddr("::ffff:1.2.3.4")
	pfx4 := netip.MustParsePrefix("10.1.2.3/8")
	pfx6 := netip.MustParsePrefix("2001:db8::1/32")
	str := "172.16.1.1/12"

	var (
		nilIP     *net.IP
		nilIPNet  *net.IPNet
		nilAddr   *netip.Addr
		nilPrefix *netip.Prefix
		nilString *string
	)

	tests := []struct {
		name     string
		in       any
		want     string
		wantType IPType
		wantErr  error
	}{
		{"net.IP v4", ip4, "1.2.3.4/32", IPv4, nil},
		{"net.IP v4 4-byte", ip4.To4(), "1.2.3.4/32", IPv4, nil},
		{"net.IP v6", ip6, "2001:db8::1/128", IPv6, nil},
		{"*net.IP v4", &ip4, "1.2.3.4/32", IPv4, nil},
		{"*net.IP nil", nilIP, "", "", ErrInvalidPrefixType},
		{"net.IP invalid length", net.IP{1, 2, 3}, "", "", ErrInvalidIP},
		{"net.IP empty", net.IP{}, "", "", ErrInvalidIP},

		{"net.IPNet v4", *ipnet4, "192.168.0.0/24", IPv4, nil},
		{"*net.IPNet v4", ipnet4, "192.168.0.0/24", IPv4, nil},
		{"*net.IPNet v6", ipnet6, "2001:db8::/48", IPv6, nil},
		{"*net.IPNet nil", nilIPNet, "", "", ErrInvalidPrefixType},
		{"net.IPNet v4 16-byte ip with 4-byte mask", net.IPNet{IP: net.ParseIP("1.2.3.4"), Mask: net.CIDRMask(24, 32)}, "1.2.3.0/24", IPv4, nil},
		{"net.IPNet mapped valid /120", net.IPNet{IP: net.ParseIP("::ffff:1.2.3.4"), Mask: net.CIDRMask(120, 128)}, "1.2.3.0/24", IPv4, nil},
		{"net.IPNet mapped valid /128", net.IPNet{IP: net.ParseIP("::ffff:1.2.3.4"), Mask: net.CIDRMask(128, 128)}, "1.2.3.4/32", IPv4, nil},
		{"net.IPNet mapped valid /96", net.IPNet{IP: net.ParseIP("::ffff:1.2.3.4"), Mask: net.CIDRMask(96, 128)}, "0.0.0.0/0", IPv4, nil},
		{"net.IPNet mapped invalid /64", net.IPNet{IP: net.ParseIP("::ffff:1.2.3.4"), Mask: net.CIDRMask(64, 128)}, "", "", ErrInvalidPrefix},
		{"net.IPNet invalid ip", net.IPNet{IP: net.IP{1, 2}, Mask: net.CIDRMask(8, 32)}, "", "", ErrInvalidIPNet},
		{"net.IPNet v4 ip with v6 mask", net.IPNet{IP: net.IP{1, 2, 3, 4}, Mask: net.CIDRMask(64, 128)}, "", "", ErrInvalidIPNet},
		{"net.IPNet non-canonical mask", net.IPNet{IP: net.IP{1, 2, 3, 4}, Mask: net.IPMask{255, 0, 255, 0}}, "", "", ErrInvalidIPNet},
		{"net.IPNet v4 invalid prefix", net.IPNet{IP: net.IP{1, 2, 3, 4}, Mask: net.IPMask{255, 0, 255, 0}}, "", "", ErrInvalidIPNet},

		{"netip.Addr v4", addr4, "1.2.3.4/32", IPv4, nil},
		{"netip.Addr v6", addr6, "2001:db8::1/128", IPv6, nil},
		{"netip.Addr mapped", addrMapped, "1.2.3.4/32", IPv4, nil},
		{"*netip.Addr v4", &addr4, "1.2.3.4/32", IPv4, nil},
		{"*netip.Addr nil", nilAddr, "", "", ErrInvalidPrefixType},
		{"netip.Addr zero", netip.Addr{}, "", "", ErrInvalidIPType},

		{"netip.Prefix v4", pfx4, "10.0.0.0/8", IPv4, nil},
		{"netip.Prefix v6", pfx6, "2001:db8::/32", IPv6, nil},
		{"*netip.Prefix v4", &pfx4, "10.0.0.0/8", IPv4, nil},
		{"*netip.Prefix nil", nilPrefix, "", "", ErrInvalidPrefixType},
		{"netip.Prefix zero", netip.Prefix{}, "", "", ErrInvalidIPType},
		{"netip.Prefix v4 invalid bits", netip.PrefixFrom(addr4, 33), "", "", ErrInvalidPrefix},
		{"netip.Prefix v6 invalid bits", netip.PrefixFrom(addr6, 129), "", "", ErrInvalidPrefix},
		{"netip.Prefix mapped /120", netip.PrefixFrom(addrMapped, 120), "1.2.3.0/24", IPv4, nil},
		{"netip.Prefix mapped /128", netip.PrefixFrom(addrMapped, 128), "1.2.3.4/32", IPv4, nil},
		{"netip.Prefix mapped /96", netip.PrefixFrom(addrMapped, 96), "0.0.0.0/0", IPv4, nil},
		{"netip.Prefix mapped /95", netip.PrefixFrom(addrMapped, 95), "", "", ErrInvalidPrefix},
		{"netip.Prefix mapped /129", netip.PrefixFrom(addrMapped, 129), "", "", ErrInvalidPrefix},

		{"string", str, "172.16.0.0/12", IPv4, nil},
		{"*string", &str, "172.16.0.0/12", IPv4, nil},
		{"*string nil", nilString, "", "", ErrInvalidPrefixType},

		{"unsupported int", 123, "", "", ErrInvalidPrefixType},
		{"unsupported nil", nil, "", "", ErrInvalidPrefixType},
	}
	e := NewEntry("test")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, typ, err := e.processPrefix(tt.in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if p.String() != tt.want || typ != tt.wantType {
				t.Errorf("got %s %s, want %s %s", p, typ, tt.want, tt.wantType)
			}
		})
	}
}

func TestEntryAddPrefixMergeAndText(t *testing.T) {
	tests := []struct {
		name    string
		add     []any
		remove  []string
		opts    []IgnoreIPOption
		want    []string
		wantErr bool
	}{
		{
			name: "ipv4 merge into /23",
			add:  []any{"192.168.0.3/24", "192.168.1.5/24"},
			want: []string{"192.168.0.0/23"},
		},
		{
			name: "ipv4 merge adjacent hosts",
			add:  []any{"10.0.0.0", "10.0.0.1", "10.0.0.2", "10.0.0.3"},
			want: []string{"10.0.0.0/30"},
		},
		{
			name: "ipv4 non-adjacent not merged",
			add:  []any{"192.168.1.0/24", "192.168.2.0/24"},
			want: []string{"192.168.1.0/24", "192.168.2.0/24"},
		},
		{
			name: "ipv4 overlapping merged",
			add:  []any{"10.0.0.0/8", "10.1.0.0/16"},
			want: []string{"10.0.0.0/8"},
		},
		{
			name:   "ipv4 split /23 into /24",
			add:    []any{"192.168.0.0/23"},
			remove: []string{"192.168.1.5/24"},
			want:   []string{"192.168.0.0/24"},
		},
		{
			name:   "ipv4 split by removing single ip",
			add:    []any{"192.168.0.0/30"},
			remove: []string{"192.168.0.1"},
			want:   []string{"192.168.0.0/32", "192.168.0.2/31"},
		},
		{
			name: "ipv6 merge",
			add:  []any{"2001:db8::1/33", "2001:db8:8000::1/33"},
			want: []string{"2001:db8::/32"},
		},
		{
			name:   "ipv6 split",
			add:    []any{"2001:db8::/32"},
			remove: []string{"2001:db8:8000::/33"},
			want:   []string{"2001:db8::/33"},
		},
		{
			name: "ipv4-mapped ipv6 merged with ipv4",
			add:  []any{"::ffff:192.168.0.3/120", "192.168.1.5/24"},
			want: []string{"192.168.0.0/23"},
		},
		{
			name:   "ipv4-mapped ipv6 removal",
			add:    []any{"192.168.0.0/23"},
			remove: []string{"::ffff:192.168.1.0/120"},
			want:   []string{"192.168.0.0/24"},
		},
		{
			name: "mixed ordering ipv4 first",
			add:  []any{"2001:db8::/32", "1.0.0.0/24"},
			want: []string{"1.0.0.0/24", "2001:db8::/32"},
		},
		{
			name: "ignore ipv4",
			add:  []any{"2001:db8::/32", "1.0.0.0/24"},
			opts: []IgnoreIPOption{IgnoreIPv4},
			want: []string{"2001:db8::/32"},
		},
		{
			name: "ignore ipv6",
			add:  []any{"2001:db8::/32", "1.0.0.0/24"},
			opts: []IgnoreIPOption{nil, IgnoreIPv6},
			want: []string{"1.0.0.0/24"},
		},
		{
			name:    "ignore all present",
			add:     []any{"1.0.0.0/24"},
			opts:    []IgnoreIPOption{IgnoreIPv4},
			wantErr: true,
		},
		{
			name:    "removed everything",
			add:     []any{"1.0.0.0/24"},
			remove:  []string{"1.0.0.0/23"},
			wantErr: true,
		},
		{
			name:    "empty entry",
			wantErr: true,
		},
		{
			name:   "comments ignored",
			add:    []any{"# comment", "1.0.0.0/24"},
			remove: []string{"// comment", "2001:db8::/32", "1.1.1.1"},
			want:   []string{"1.0.0.0/24"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEntry("test")
			for _, a := range tt.add {
				if err := e.AddPrefix(a); err != nil {
					t.Fatalf("AddPrefix(%v): %v", a, err)
				}
			}
			for _, r := range tt.remove {
				if err := e.RemovePrefix(r); err != nil {
					t.Fatalf("RemovePrefix(%v): %v", r, err)
				}
			}

			text, err := e.MarshalText(tt.opts...)
			prefixes, perr := e.MarshalPrefix(tt.opts...)
			ranges, rerr := e.MarshalIPRange(tt.opts...)
			if tt.wantErr {
				if err == nil || perr == nil || rerr == nil {
					t.Fatalf("expected errors, got %v %v %v", err, perr, rerr)
				}
				return
			}
			if err != nil || perr != nil || rerr != nil {
				t.Fatalf("unexpected errors: %v %v %v", err, perr, rerr)
			}
			compareStrings(t, text, tt.want)

			pstr := make([]string, 0, len(prefixes))
			for _, p := range prefixes {
				pstr = append(pstr, p.String())
			}
			compareStrings(t, pstr, tt.want)

			var b netipx.IPSetBuilder
			for _, r := range ranges {
				b.AddRange(r)
			}
			set, _ := b.IPSet()
			rstr := make([]string, 0)
			for _, p := range set.Prefixes() {
				rstr = append(rstr, p.String())
			}
			compareStrings(t, rstr, tt.want)
		})
	}
}

func TestEntryMarshalIPRange(t *testing.T) {
	e := NewEntry("test")
	for _, s := range []string{"1.0.0.1", "1.0.0.2", "1.0.0.3", "2001:db8::/127"} {
		if err := e.AddPrefix(s); err != nil {
			t.Fatal(err)
		}
	}
	ranges, err := e.MarshalIPRange()
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(ranges))
	for _, r := range ranges {
		got = append(got, r.String())
	}
	compareStrings(t, got, []string{"1.0.0.1-1.0.0.3", "2001:db8::-2001:db8::1"})
}

func TestEntryAddRemoveErrors(t *testing.T) {
	tests := []struct {
		name string
		fn   func(e *Entry) error
	}{
		{"add invalid", func(e *Entry) error { return e.AddPrefix("invalid") }},
		{"add invalid mapped", func(e *Entry) error { return e.AddPrefix("::ffff:1.2.3.4/64") }},
		{"remove invalid", func(e *Entry) error { return e.RemovePrefix("1.2.3.4/33") }},
		{"add bad type", func(e *Entry) error { return e.add(&netip.Prefix{}, "bad") }},
		{"remove bad type", func(e *Entry) error { return e.remove(&netip.Prefix{}, "bad") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(NewEntry("x")); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestEntryGetIPSets(t *testing.T) {
	e := NewEntry("test")
	if _, err := e.GetIPv4Set(); err == nil {
		t.Error("expected error for empty ipv4 set")
	}
	if _, err := e.GetIPv6Set(); err == nil {
		t.Error("expected error for empty ipv6 set")
	}
	_ = e.AddPrefix("192.168.0.3/24")
	_ = e.AddPrefix("192.168.1.5/24")
	_ = e.AddPrefix("2001:db8::/32")
	s4, err := e.GetIPv4Set()
	if err != nil {
		t.Fatal(err)
	}
	if p := s4.Prefixes(); len(p) != 1 || p[0].String() != "192.168.0.0/23" {
		t.Errorf("got %v", p)
	}
	s6, err := e.GetIPv6Set()
	if err != nil {
		t.Fatal(err)
	}
	if p := s6.Prefixes(); len(p) != 1 || p[0].String() != "2001:db8::/32" {
		t.Errorf("got %v", p)
	}
	// cached set is reused
	s4b, _ := e.GetIPv4Set()
	if s4 != s4b {
		t.Error("expected cached ipv4 set")
	}
}

// brokenBuilder returns an IPSetBuilder that will fail on IPSet().
func brokenBuilder() *netipx.IPSetBuilder {
	b := new(netipx.IPSetBuilder)
	b.AddPrefix(netip.Prefix{})
	return b
}

func TestEntryBuildIPSetErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(e *Entry)
	}{
		{"ipv4", func(e *Entry) { e.ipv4Builder = brokenBuilder() }},
		{"ipv6", func(e *Entry) { e.ipv6Builder = brokenBuilder() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fns := map[string]func(e *Entry) error{
				"GetIPv4Set":     func(e *Entry) error { _, err := e.GetIPv4Set(); return err },
				"GetIPv6Set":     func(e *Entry) error { _, err := e.GetIPv6Set(); return err },
				"MarshalPrefix":  func(e *Entry) error { _, err := e.MarshalPrefix(); return err },
				"MarshalIPRange": func(e *Entry) error { _, err := e.MarshalIPRange(); return err },
				"MarshalText":    func(e *Entry) error { _, err := e.MarshalText(); return err },
			}
			for name, fn := range fns {
				e := NewEntry("x")
				tt.setup(e)
				if err := fn(e); err == nil {
					t.Errorf("%s: expected error", name)
				}
			}
		})
	}
}
