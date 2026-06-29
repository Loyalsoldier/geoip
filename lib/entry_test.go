package lib

import (
	"net"
	"net/netip"
	"testing"

	"go4.org/netipx"
)

func TestProcessPrefixVariants(t *testing.T) {
	e := NewEntry("proc")

	ipv4 := net.ParseIP("1.1.1.1")
	p, ipType, err := e.processPrefix(ipv4)
	if err != nil || ipType != IPv4 || p.String() != "1.1.1.1/32" {
		t.Fatalf("processPrefix(net.IPv4) = %v %v %v", p, ipType, err)
	}

	if _, _, err := e.processPrefix(net.IP{}); err != ErrInvalidIP {
		t.Fatalf("expected ErrInvalidIP for empty net.IP, got %v", err)
	}

	ipv6 := net.ParseIP("2001:db8::1")
	p, ipType, err = e.processPrefix(ipv6)
	if err != nil || ipType != IPv6 || p.String() != "2001:db8::1/128" {
		t.Fatalf("processPrefix(net.IPv6) = %v %v %v", p, ipType, err)
	}

	_, n, _ := net.ParseCIDR("10.0.0.0/24")
	p, ipType, err = e.processPrefix(n)
	if err != nil || ipType != IPv4 || p.String() != "10.0.0.0/24" {
		t.Fatalf("processPrefix(*net.IPNet) = %v %v %v", p, ipType, err)
	}

	_, n6, _ := net.ParseCIDR("2001:db8:ffff::/48")
	p, ipType, err = e.processPrefix(n6)
	if err != nil || ipType != IPv6 || p.String() != "2001:db8:ffff::/48" {
		t.Fatalf("processPrefix(*net.IPNet ipv6) = %v %v %v", p, ipType, err)
	}

	badNet := &net.IPNet{IP: net.IPv4(1, 2, 3, 4), Mask: net.IPMask{1}}
	if _, _, err := e.processPrefix(badNet); err != ErrInvalidIPNet {
		t.Fatalf("expected ErrInvalidIPNet, got %v", err)
	}

	addr := netip.MustParseAddr("192.0.2.1")
	p, ipType, err = e.processPrefix(addr)
	if err != nil || ipType != IPv4 || p.String() != "192.0.2.1/32" {
		t.Fatalf("processPrefix(netip.Addr) = %v %v %v", p, ipType, err)
	}

	ipv6Addr := netip.MustParseAddr("2001:db8::3")
	p, ipType, err = e.processPrefix(ipv6Addr)
	if err != nil || ipType != IPv6 || p.String() != "2001:db8::3/128" {
		t.Fatalf("processPrefix(netip.Addr ipv6) = %v %v %v", p, ipType, err)
	}

	addrPtr := netip.MustParseAddr("2001:db8::2")
	p, ipType, err = e.processPrefix(&addrPtr)
	if err != nil || ipType != IPv6 || p.String() != "2001:db8::2/128" {
		t.Fatalf("processPrefix(*netip.Addr) = %v %v %v", p, ipType, err)
	}

	addrPtr4 := netip.MustParseAddr("198.18.0.1")
	p, ipType, err = e.processPrefix(&addrPtr4)
	if err != nil || ipType != IPv4 || p.String() != "198.18.0.1/32" {
		t.Fatalf("processPrefix(*netip.Addr ipv4) = %v %v %v", p, ipType, err)
	}

	prefix := netip.MustParsePrefix("198.51.100.0/24")
	p, ipType, err = e.processPrefix(prefix)
	if err != nil || ipType != IPv4 || p.String() != "198.51.100.0/24" {
		t.Fatalf("processPrefix(netip.Prefix) = %v %v %v", p, ipType, err)
	}

	ipv6PrefixVal := netip.MustParsePrefix("2001:db8:abcd::/48")
	if p, ipType, err := e.processPrefix(ipv6PrefixVal); err != nil || ipType != IPv6 || p.String() != "2001:db8:abcd::/48" {
		t.Fatalf("processPrefix(netip.Prefix ipv6) = %v %v %v", p, ipType, err)
	}

	prefixPtr := netip.MustParsePrefix("2001:db8:ffff::/48")
	p, ipType, err = e.processPrefix(&prefixPtr)
	if err != nil || ipType != IPv6 || p.String() != "2001:db8:ffff::/48" {
		t.Fatalf("processPrefix(*netip.Prefix) = %v %v %v", p, ipType, err)
	}

	prefixPtr4 := netip.MustParsePrefix("198.51.100.0/24")
	p, ipType, err = e.processPrefix(&prefixPtr4)
	if err != nil || ipType != IPv4 || p.String() != "198.51.100.0/24" {
		t.Fatalf("processPrefix(*netip.Prefix ipv4) = %v %v %v", p, ipType, err)
	}

	// IPv4-mapped IPv6 with insufficient bits should be rejected
	badPrefix := netip.MustParsePrefix("::ffff:192.0.2.1/95")
	if _, _, err := e.processPrefix(badPrefix); err != ErrInvalidPrefix {
		t.Fatalf("expected ErrInvalidPrefix, got %v", err)
	}

	mappedPrefix := netip.MustParsePrefix("::ffff:192.0.2.0/120")
	if p, ipType, err := e.processPrefix(mappedPrefix); err != nil || ipType != IPv4 || p.String() != "192.0.2.0/24" {
		t.Fatalf("processPrefix(mappedPrefix) = %v %v %v", p, ipType, err)
	}

	invalidPrefix4 := netip.PrefixFrom(netip.MustParseAddr("1.1.1.1"), 40)
	if _, _, err := e.processPrefix(invalidPrefix4); err != ErrInvalidPrefix {
		t.Fatalf("expected ErrInvalidPrefix for invalid ipv4 prefix, got %v", err)
	}

	invalidPrefix6 := netip.PrefixFrom(netip.MustParseAddr("2001:db8::1"), 200)
	if _, _, err := e.processPrefix(invalidPrefix6); err != ErrInvalidPrefix {
		t.Fatalf("expected ErrInvalidPrefix for invalid ipv6 prefix, got %v", err)
	}

	invalidPrefix4Ptr := invalidPrefix4
	if _, _, err := e.processPrefix(&invalidPrefix4Ptr); err != ErrInvalidPrefix {
		t.Fatalf("expected ErrInvalidPrefix for invalid ipv4 prefix pointer, got %v", err)
	}

	invalidPrefix6Ptr := invalidPrefix6
	if _, _, err := e.processPrefix(&invalidPrefix6Ptr); err != ErrInvalidPrefix {
		t.Fatalf("expected ErrInvalidPrefix for invalid ipv6 prefix pointer, got %v", err)
	}

	zeroPrefix := netip.Prefix{}
	if _, _, err := e.processPrefix(zeroPrefix); err != ErrInvalidIPLength {
		t.Fatalf("expected ErrInvalidIPLength for zero prefix, got %v", err)
	}

	badPrefixPtr := badPrefix
	if _, _, err := e.processPrefix(&badPrefixPtr); err != ErrInvalidPrefix {
		t.Fatalf("expected ErrInvalidPrefix for pointer bad prefix, got %v", err)
	}

	zeroPrefixPtr := zeroPrefix
	if _, _, err := e.processPrefix(&zeroPrefixPtr); err != ErrInvalidIPLength {
		t.Fatalf("expected ErrInvalidIPLength for zero prefix pointer, got %v", err)
	}

	addrZero := netip.Addr{}
	if _, _, err := e.processPrefix(&addrZero); err != ErrInvalidIPLength {
		t.Fatalf("expected ErrInvalidIPLength for zero addr pointer, got %v", err)
	}

	mappedPrefixPtr := mappedPrefix
	if p, ipType, err := e.processPrefix(&mappedPrefixPtr); err != nil || ipType != IPv4 || p.String() != "192.0.2.0/24" {
		t.Fatalf("processPrefix(mappedPrefixPtr) = %v %v %v", p, ipType, err)
	}

	if _, _, err := e.processPrefix(netip.Addr{}); err != ErrInvalidIPLength {
		t.Fatalf("expected ErrInvalidIPLength, got %v", err)
	}

	if _, _, err := e.processPrefix("1.2.3.4"); err != nil {
		t.Fatalf("processPrefix(string ip) error = %v", err)
	}

	if _, _, err := e.processPrefix("2001:db8::1"); err != nil {
		t.Fatalf("processPrefix(string ipv6) error = %v", err)
	}

	if _, _, err := e.processPrefix("10.0.0.0/8"); err != nil {
		t.Fatalf("processPrefix(string cidr) error = %v", err)
	}

	if _, _, err := e.processPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("processPrefix(string cidr ipv6) error = %v", err)
	}

	if _, _, err := e.processPrefix("invalid/24"); err != ErrInvalidCIDR {
		t.Fatalf("expected ErrInvalidCIDR, got %v", err)
	}

	if _, _, err := e.processPrefix(" //comment"); err != ErrCommentLine {
		t.Fatalf("expected ErrCommentLine, got %v", err)
	}

	if _, _, err := e.processPrefix(123); err != ErrInvalidPrefixType {
		t.Fatalf("expected ErrInvalidPrefixType, got %v", err)
	}
}

func TestEntryAddAndRemovePrefix(t *testing.T) {
	e := NewEntry("demo")

	if err := e.AddPrefix("10.0.0.0/24"); err != nil {
		t.Fatalf("AddPrefix() error = %v", err)
	}
	if err := e.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("AddPrefix() error = %v", err)
	}

	ipv4set, err := e.GetIPv4Set()
	if err != nil || !ipv4set.Contains(netip.MustParseAddr("10.0.0.1")) {
		t.Fatalf("IPv4 set missing data: %v %v", ipv4set, err)
	}

	ipv6set, err := e.GetIPv6Set()
	if err != nil || !ipv6set.Contains(netip.MustParseAddr("2001:db8::1")) {
		t.Fatalf("IPv6 set missing data: %v %v", ipv6set, err)
	}

	if err := e.RemovePrefix("10.0.0.0/24"); err != nil {
		t.Fatalf("RemovePrefix() error = %v", err)
	}
	e.ipv4Set = nil
	ipv4set, _ = e.GetIPv4Set()
	if ipv4set.Contains(netip.MustParseAddr("10.0.0.1")) {
		t.Fatalf("prefix should be removed")
	}

	if err := e.RemovePrefix("2001:db8::/32"); err != nil {
		t.Fatalf("RemovePrefix() error = %v", err)
	}
	e.ipv6Set = nil
	ipv6set, _ = e.GetIPv6Set()
	if ipv6set.Contains(netip.MustParseAddr("2001:db8::1")) {
		t.Fatalf("ipv6 prefix should be removed")
	}

	if err := e.RemovePrefix("invalid"); err == nil {
		t.Fatalf("expected error for invalid prefix")
	}
}

func TestEntryAddRemoveInvalidIPType(t *testing.T) {
	e := NewEntry("invalid")
	if err := e.add(nil, IPType("unknown")); err != ErrInvalidIPType {
		t.Fatalf("expected ErrInvalidIPType, got %v", err)
	}
	if err := e.remove(nil, IPType("unknown")); err != ErrInvalidIPType {
		t.Fatalf("expected ErrInvalidIPType, got %v", err)
	}
}

func TestEntryMarshalFunctions(t *testing.T) {
	e := NewEntry("marshal")
	_ = e.AddPrefix("203.0.113.0/24")
	_ = e.AddPrefix("2001:db8:abcd::/48")

	prefixes, err := e.MarshalPrefix()
	if err != nil || len(prefixes) != 2 {
		t.Fatalf("MarshalPrefix() = %v, %v", prefixes, err)
	}

	prefixes, err = e.MarshalPrefix(IgnoreIPv6)
	if err != nil || len(prefixes) != 1 {
		t.Fatalf("MarshalPrefix(IgnoreIPv6) = %v, %v", prefixes, err)
	}

	ranges, err := e.MarshalIPRange()
	if err != nil || len(ranges) != 2 {
		t.Fatalf("MarshalIPRange() = %v, %v", ranges, err)
	}

	ranges, err = e.MarshalIPRange(IgnoreIPv4)
	if err != nil || len(ranges) != 1 {
		t.Fatalf("MarshalIPRange(IgnoreIPv4) = %v, %v", ranges, err)
	}

	text, err := e.MarshalText()
	if err != nil || len(text) != 2 {
		t.Fatalf("MarshalText() = %v, %v", text, err)
	}

	// Ignore IPv4 results
	prefixes, err = e.MarshalPrefix(IgnoreIPv4)
	if err != nil || len(prefixes) != 1 {
		t.Fatalf("MarshalPrefix(IgnoreIPv4) = %v, %v", prefixes, err)
	}

	text, err = e.MarshalText(IgnoreIPv4)
	if err != nil || len(text) != 1 {
		t.Fatalf("MarshalText(IgnoreIPv4) = %v, %v", text, err)
	}

	text, err = e.MarshalText(IgnoreIPv6)
	if err != nil || len(text) != 1 {
		t.Fatalf("MarshalText(IgnoreIPv6) = %v, %v", text, err)
	}
}

func TestEntryMarshalErrors(t *testing.T) {
	e := NewEntry("empty")

	if _, err := e.MarshalPrefix(); err == nil {
		t.Fatalf("expected error for empty entry")
	}
	if _, err := e.MarshalIPRange(); err == nil {
		t.Fatalf("expected error for empty entry")
	}
	if _, err := e.MarshalText(); err == nil {
		t.Fatalf("expected error for empty entry")
	}
}

func TestEntryBuildIPSetError(t *testing.T) {
	e := NewEntry("errbuild")
	builder := &netipx.IPSetBuilder{}
	builder.AddPrefix(netip.Prefix{}) // invalid prefix triggers builder error
	e.ipv4Builder = builder

	if _, err := e.GetIPv4Set(); err == nil {
		t.Fatalf("expected buildIPSet error")
	}

	builder6 := &netipx.IPSetBuilder{}
	builder6.AddPrefix(netip.Prefix{})
	e.ipv6Builder = builder6
	if _, err := e.GetIPv6Set(); err == nil {
		t.Fatalf("expected buildIPSet error for ipv6")
	}

	if _, err := e.MarshalPrefix(); err == nil {
		t.Fatalf("expected MarshalPrefix error from builder")
	}

	if _, err := e.MarshalIPRange(); err == nil {
		t.Fatalf("expected MarshalIPRange error from builder")
	}

	if _, err := e.MarshalText(); err == nil {
		t.Fatalf("expected MarshalText error from builder")
	}

	// Use fresh entries to ensure builder errors are preserved
	e2 := NewEntry("errbuild2")
	b2 := &netipx.IPSetBuilder{}
	b2.AddPrefix(netip.Prefix{})
	e2.ipv4Builder = b2
	if _, err := e2.MarshalPrefix(); err == nil {
		t.Fatalf("expected MarshalPrefix error from builder")
	}

	e3 := NewEntry("errbuild3")
	b3 := &netipx.IPSetBuilder{}
	b3.AddPrefix(netip.Prefix{})
	e3.ipv4Builder = b3
	if _, err := e3.MarshalIPRange(); err == nil {
		t.Fatalf("expected MarshalIPRange error from builder")
	}

	e4 := NewEntry("errbuild4")
	b4 := &netipx.IPSetBuilder{}
	b4.AddPrefix(netip.Prefix{})
	e4.ipv4Builder = b4
	if _, err := e4.MarshalText(); err == nil {
		t.Fatalf("expected MarshalText error from builder")
	}
}

func TestEntryGetSetErrors(t *testing.T) {
	e := NewEntry("sets")
	_ = e.AddPrefix("2001:db8::/32")

	if _, err := e.GetIPv4Set(); err == nil {
		t.Fatalf("expected error for missing IPv4 set")
	}

	e2 := NewEntry("sets2")
	_ = e2.AddPrefix("192.0.2.0/24")
	if _, err := e2.GetIPv6Set(); err == nil {
		t.Fatalf("expected error for missing IPv6 set")
	}
}

func TestAddPrefixErrorPath(t *testing.T) {
	e := NewEntry("err")
	if err := e.AddPrefix("bad-prefix"); err == nil {
		t.Fatalf("expected error for bad prefix")
	}
	if err := e.RemovePrefix("bad-prefix"); err == nil {
		t.Fatalf("expected error for bad prefix removal")
	}
}

func TestEntryCommentLineHandling(t *testing.T) {
	e := NewEntry("comment")
	if err := e.AddPrefix("# this is comment"); err != ErrInvalidIPType {
		t.Fatalf("expected ErrInvalidIPType for comment line, got %v", err)
	}
	if err := e.RemovePrefix("// another comment"); err != ErrInvalidIPType {
		t.Fatalf("expected ErrInvalidIPType for comment line removal, got %v", err)
	}
}

func TestEntryMarshalIgnoreIPv6(t *testing.T) {
	e := NewEntry("ignore6")
	_ = e.AddPrefix("2001:db8::/32")
	_ = e.AddPrefix("198.51.100.0/24")

	ranges, err := e.MarshalIPRange(IgnoreIPv6)
	if err != nil || len(ranges) != 1 {
		t.Fatalf("MarshalIPRange(IgnoreIPv6) = %v, %v", ranges, err)
	}
}

func TestEntryBuildIPSetReuse(t *testing.T) {
	e := NewEntry("reuse")
	builder := netipx.IPSetBuilder{}
	builder.AddPrefix(netip.MustParsePrefix("10.10.0.0/16"))
	e.ipv4Builder = &builder

	if err := e.buildIPSet(); err != nil {
		t.Fatalf("buildIPSet() error = %v", err)
	}
	if !e.hasIPv4Set() {
		t.Fatalf("expected ipv4 set to be built")
	}
	// Second call should be a no-op but still succeed
	if err := e.buildIPSet(); err != nil {
		t.Fatalf("buildIPSet() second call error = %v", err)
	}
}
