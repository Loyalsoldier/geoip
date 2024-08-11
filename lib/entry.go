package lib

import (
	"fmt"
	"net"
	"net/netip"
	"strings"

	"go4.org/netipx"
)

type Entry struct {
	name        string
	ipv4Builder *netipx.IPSetBuilder
	ipv6Builder *netipx.IPSetBuilder
	ipv4Set     *netipx.IPSet
	ipv6Set     *netipx.IPSet
}

func NewEntry(name string) *Entry {
	return &Entry{
		name: strings.ToUpper(strings.TrimSpace(name)),
	}
}

func (e *Entry) GetName() string {
	return e.name
}

func (e *Entry) hasIPv4Builder() bool {
	return e.ipv4Builder != nil
}

func (e *Entry) hasIPv6Builder() bool {
	return e.ipv6Builder != nil
}

func (e *Entry) hasIPv4Set() bool {
	return e.ipv4Set != nil
}

func (e *Entry) hasIPv6Set() bool {
	return e.ipv6Set != nil
}

func (e *Entry) GetIPv4Set() (*netipx.IPSet, error) {
	if err := e.buildIPSet(); err != nil {
		return nil, err
	}

	if e.hasIPv4Set() {
		return e.ipv4Set, nil
	}

	return nil, fmt.Errorf("entry %s has no ipv4 set", e.GetName())
}

func (e *Entry) GetIPv6Set() (*netipx.IPSet, error) {
	if err := e.buildIPSet(); err != nil {
		return nil, err
	}

	if e.hasIPv6Set() {
		return e.ipv6Set, nil
	}

	return nil, fmt.Errorf("entry %s has no ipv6 set", e.GetName())
}

func (e *Entry) processPrefix(src any) (*netip.Prefix, IPType, error) {
	switch src := src.(type) {
	case net.IP:
		ip, ok := netipx.FromStdIP(src)
		if !ok {
			return nil, "", ErrInvalidIP
		}
		ip = ip.Unmap()
		switch {
		case ip.Is4():
			prefix := netip.PrefixFrom(ip, 32)
			return &prefix, IPv4, nil
		case ip.Is6():
			prefix := netip.PrefixFrom(ip, 128)
			return &prefix, IPv6, nil
		default:
			return nil, "", ErrInvalidIPLength
		}

	case *net.IPNet:
		prefix, ok := netipx.FromStdIPNet(src)
		if !ok {
			return nil, "", ErrInvalidIPNet
		}
		ip := prefix.Addr().Unmap()
		switch {
		case ip.Is4():
			return &prefix, IPv4, nil
		case ip.Is6():
			return &prefix, IPv6, nil
		default:
			return nil, "", ErrInvalidIPLength
		}

	case netip.Addr:
		src = src.Unmap()
		switch {
		case src.Is4():
			prefix := netip.PrefixFrom(src, 32)
			return &prefix, IPv4, nil
		case src.Is6():
			prefix := netip.PrefixFrom(src, 128)
			return &prefix, IPv6, nil
		default:
			return nil, "", ErrInvalidIPLength
		}

	case *netip.Addr:
		*src = (*src).Unmap()
		switch {
		case src.Is4():
			prefix := netip.PrefixFrom(*src, 32)
			return &prefix, IPv4, nil
		case src.Is6():
			prefix := netip.PrefixFrom(*src, 128)
			return &prefix, IPv6, nil
		default:
			return nil, "", ErrInvalidIPLength
		}

	case netip.Prefix:
		ip := src.Addr()
		switch {
		case ip.Is4():
			prefix, err := ip.Prefix(src.Bits())
			if err != nil {
				return nil, "", ErrInvalidPrefix
			}
			return &prefix, IPv4, nil
		case ip.Is4In6():
			ip = ip.Unmap()
			bits := src.Bits()
			if bits < 96 {
				return nil, "", ErrInvalidPrefix
			}
			prefix, err := ip.Prefix(bits - 96)
			if err != nil {
				return nil, "", ErrInvalidPrefix
			}
			return &prefix, IPv4, nil
		case ip.Is6():
			prefix, err := ip.Prefix(src.Bits())
			if err != nil {
				return nil, "", ErrInvalidPrefix
			}
			return &prefix, IPv6, nil
		default:
			return nil, "", ErrInvalidIPLength
		}

	case *netip.Prefix:
		ip := src.Addr()
		switch {
		case ip.Is4():
			prefix, err := ip.Prefix(src.Bits())
			if err != nil {
				return nil, "", ErrInvalidPrefix
			}
			return &prefix, IPv4, nil
		case ip.Is4In6():
			ip = ip.Unmap()
			bits := src.Bits()
			if bits < 96 {
				return nil, "", ErrInvalidPrefix
			}
			prefix, err := ip.Prefix(bits - 96)
			if err != nil {
				return nil, "", ErrInvalidPrefix
			}
			return &prefix, IPv4, nil
		case ip.Is6():
			prefix, err := ip.Prefix(src.Bits())
			if err != nil {
				return nil, "", ErrInvalidPrefix
			}
			return &prefix, IPv6, nil
		default:
			return nil, "", ErrInvalidIPLength
		}

	case string:
		src, _, _ = strings.Cut(src, "#")
		src, _, _ = strings.Cut(src, "//")
		src, _, _ = strings.Cut(src, "/*")
		src = strings.TrimSpace(src)
		if src == "" {
			return nil, "", ErrCommentLine
		}

		switch strings.Contains(src, "/") {
		case true: // src is CIDR notation
			ip, network, err := net.ParseCIDR(src)
			if err != nil {
				return nil, "", ErrInvalidCIDR
			}
			addr, ok := netipx.FromStdIP(ip)
			if !ok {
				return nil, "", ErrInvalidIP
			}
			if addr.Unmap().Is4() && strings.Contains(network.String(), "::") { // src is invalid IPv4-mapped IPv6 address
				return nil, "", ErrInvalidCIDR
			}
			prefix, ok := netipx.FromStdIPNet(network)
			if !ok {
				return nil, "", ErrInvalidIPNet
			}

			addr = prefix.Addr().Unmap()
			switch {
			case addr.Is4():
				return &prefix, IPv4, nil
			case addr.Is6():
				return &prefix, IPv6, nil
			default:
				return nil, "", ErrInvalidIPLength
			}

		case false: // src is IP address
			ip, err := netip.ParseAddr(src)
			if err != nil {
				return nil, "", ErrInvalidIP
			}
			ip = ip.Unmap()
			switch {
			case ip.Is4():
				prefix := netip.PrefixFrom(ip, 32)
				return &prefix, IPv4, nil
			case ip.Is6():
				prefix := netip.PrefixFrom(ip, 128)
				return &prefix, IPv6, nil
			default:
				return nil, "", ErrInvalidIPLength
			}
		}
	}

	return nil, "", ErrInvalidPrefixType
}

func (e *Entry) add(prefix *netip.Prefix, ipType IPType) error {
	switch ipType {
	case IPv4:
		if !e.hasIPv4Builder() {
			e.ipv4Builder = new(netipx.IPSetBuilder)
		}
		e.ipv4Builder.AddPrefix(*prefix)
	case IPv6:
		if !e.hasIPv6Builder() {
			e.ipv6Builder = new(netipx.IPSetBuilder)
		}
		e.ipv6Builder.AddPrefix(*prefix)
	default:
		return ErrInvalidIPType
	}

	return nil
}

func (e *Entry) remove(prefix *netip.Prefix, ipType IPType) error {
	switch ipType {
	case IPv4:
		if e.hasIPv4Builder() {
			e.ipv4Builder.RemovePrefix(*prefix)
		}
	case IPv6:
		if e.hasIPv6Builder() {
			e.ipv6Builder.RemovePrefix(*prefix)
		}
	default:
		return ErrInvalidIPType
	}

	return nil
}

func (e *Entry) AddPrefix(cidr any) error {
	prefix, ipType, err := e.processPrefix(cidr)
	if err != nil && err != ErrCommentLine {
		return err
	}
	if err := e.add(prefix, ipType); err != nil {
		return err
	}
	return nil
}

func (e *Entry) RemovePrefix(cidr string) error {
	prefix, ipType, err := e.processPrefix(cidr)
	if err != nil && err != ErrCommentLine {
		return err
	}
	if err := e.remove(prefix, ipType); err != nil {
		return err
	}
	return nil
}

func (e *Entry) buildIPSet() error {
	if e.hasIPv4Builder() && !e.hasIPv4Set() {
		ipv4set, err := e.ipv4Builder.IPSet()
		if err != nil {
			return err
		}
		e.ipv4Set = ipv4set
	}

	if e.hasIPv6Builder() && !e.hasIPv6Set() {
		ipv6set, err := e.ipv6Builder.IPSet()
		if err != nil {
			return err
		}
		e.ipv6Set = ipv6set
	}

	return nil
}

func (e *Entry) MarshalPrefix(opts ...IgnoreIPOption) ([]netip.Prefix, error) {
	var ignoreIPType IPType
	for _, opt := range opts {
		if opt != nil {
			ignoreIPType = opt()
		}
	}
	disableIPv4, disableIPv6 := false, false
	switch ignoreIPType {
	case IPv4:
		disableIPv4 = true
	case IPv6:
		disableIPv6 = true
	}

	if err := e.buildIPSet(); err != nil {
		return nil, err
	}

	prefixes := make([]netip.Prefix, 0, 1024)

	if !disableIPv4 && e.hasIPv4Set() {
		prefixes = append(prefixes, e.ipv4Set.Prefixes()...)
	}

	if !disableIPv6 && e.hasIPv6Set() {
		prefixes = append(prefixes, e.ipv6Set.Prefixes()...)
	}

	if len(prefixes) > 0 {
		return prefixes, nil
	}

	return nil, fmt.Errorf("entry %s has no prefix", e.GetName())
}

func (e *Entry) MarshalIPRange(opts ...IgnoreIPOption) ([]netipx.IPRange, error) {
	var ignoreIPType IPType
	for _, opt := range opts {
		if opt != nil {
			ignoreIPType = opt()
		}
	}
	disableIPv4, disableIPv6 := false, false
	switch ignoreIPType {
	case IPv4:
		disableIPv4 = true
	case IPv6:
		disableIPv6 = true
	}

	if err := e.buildIPSet(); err != nil {
		return nil, err
	}

	ipranges := make([]netipx.IPRange, 0, 1024)

	if !disableIPv4 && e.hasIPv4Set() {
		ipranges = append(ipranges, e.ipv4Set.Ranges()...)
	}

	if !disableIPv6 && e.hasIPv6Set() {
		ipranges = append(ipranges, e.ipv6Set.Ranges()...)
	}

	if len(ipranges) > 0 {
		return ipranges, nil
	}

	return nil, fmt.Errorf("entry %s has no prefix", e.GetName())
}

func (e *Entry) MarshalText(opts ...IgnoreIPOption) ([]string, error) {
	var ignoreIPType IPType
	for _, opt := range opts {
		if opt != nil {
			ignoreIPType = opt()
		}
	}
	disableIPv4, disableIPv6 := false, false
	switch ignoreIPType {
	case IPv4:
		disableIPv4 = true
	case IPv6:
		disableIPv6 = true
	}

	if err := e.buildIPSet(); err != nil {
		return nil, err
	}

	cidrList := make([]string, 0, 1024)

	if !disableIPv4 && e.hasIPv4Set() {
		for _, prefix := range e.ipv4Set.Prefixes() {
			cidrList = append(cidrList, prefix.String())
		}
	}

	if !disableIPv6 && e.hasIPv6Set() {
		for _, prefix := range e.ipv6Set.Prefixes() {
			cidrList = append(cidrList, prefix.String())
		}
	}

	if len(cidrList) > 0 {
		return cidrList, nil
	}

	return nil, fmt.Errorf("entry %s has no prefix", e.GetName())
}
