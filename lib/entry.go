package lib

import (
	"fmt"
	"net"
	"net/netip"
	"strings"
	"sync"

	"go4.org/netipx"
)

type Entry struct {
	name        string
	mu          *sync.Mutex
	ipv4Builder *netipx.IPSetBuilder
	ipv6Builder *netipx.IPSetBuilder
}

func NewEntry(name string) *Entry {
	return &Entry{
		name:        strings.ToUpper(strings.TrimSpace(name)),
		mu:          new(sync.Mutex),
		ipv4Builder: new(netipx.IPSetBuilder),
		ipv6Builder: new(netipx.IPSetBuilder),
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
	e.mu.Lock()
	defer e.mu.Unlock()

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
	e.mu.Lock()
	defer e.mu.Unlock()

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

	prefixSet := make([]string, 0, 1024)

	if !disableIPv4 && e.hasIPv4Builder() {
		ipv4set, err := e.ipv4Builder.IPSet()
		if err != nil {
			return nil, err
		}
		prefixes := ipv4set.Prefixes()
		for _, prefix := range prefixes {
			prefixSet = append(prefixSet, prefix.String())
		}
	}

	if !disableIPv6 && e.hasIPv6Builder() {
		ipv6set, err := e.ipv6Builder.IPSet()
		if err != nil {
			return nil, err
		}
		prefixes := ipv6set.Prefixes()
		for _, prefix := range prefixes {
			prefixSet = append(prefixSet, prefix.String())
		}
	}

	if len(prefixSet) > 0 {
		return prefixSet, nil
	}

	return nil, fmt.Errorf("entry %s has no prefix", e.GetName())
}
