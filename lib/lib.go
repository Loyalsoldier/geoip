package lib

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"inet.af/netaddr"
)

const (
	ActionAdd     Action = "add"
	ActionRemove  Action = "remove"
	ActionReplace Action = "replace"
	ActionOutput  Action = "output"

	IPv4 IPType = "ipv4"
	IPv6 IPType = "ipv6"
)

var ActionsRegistry = map[Action]bool{
	ActionAdd:     true,
	ActionRemove:  true,
	ActionReplace: true,
	ActionOutput:  true,
}

type Action string

type IPType string

type Typer interface {
	GetType() string
}

type Actioner interface {
	GetAction() Action
}

type Descriptioner interface {
	GetDescription() string
}

type InputConverter interface {
	Typer
	Actioner
	Descriptioner
	Input(Container) (Container, error)
}

type OutputConverter interface {
	Typer
	Actioner
	Descriptioner
	Output(Container) error
}

type Entry struct {
	name        string
	mu          *sync.Mutex
	ipv4Builder *netaddr.IPSetBuilder
	ipv6Builder *netaddr.IPSetBuilder
}

func NewEntry(name string) *Entry {
	return &Entry{
		name:        strings.ToUpper(strings.TrimSpace(name)),
		mu:          new(sync.Mutex),
		ipv4Builder: new(netaddr.IPSetBuilder),
		ipv6Builder: new(netaddr.IPSetBuilder),
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

func (e *Entry) processPrefix(src interface{}) (*netaddr.IPPrefix, IPType, error) {
	switch src := src.(type) {
	case net.IP:
		ip, ok := netaddr.FromStdIP(src)
		if !ok {
			return nil, "", ErrInvalidIP
		}
		switch {
		case ip.Is4():
			prefix := netaddr.IPPrefixFrom(ip, 32)
			return &prefix, IPv4, nil
		case ip.Is6():
			prefix := netaddr.IPPrefixFrom(ip, 128)
			return &prefix, IPv6, nil
		default:
			return nil, "", ErrInvalidIPLength
		}

	case *net.IPNet:
		prefix, ok := netaddr.FromStdIPNet(src)
		if !ok {
			return nil, "", ErrInvalidIPNet
		}
		ip := prefix.IP()
		switch {
		case ip.Is4():
			return &prefix, IPv4, nil
		case ip.Is6():
			return &prefix, IPv6, nil
		default:
			return nil, "", ErrInvalidIPLength
		}

	case netaddr.IP:
		switch {
		case src.Is4():
			prefix := netaddr.IPPrefixFrom(src, 32)
			return &prefix, IPv4, nil
		case src.Is6():
			prefix := netaddr.IPPrefixFrom(src, 128)
			return &prefix, IPv6, nil
		default:
			return nil, "", ErrInvalidIPLength
		}

	case *netaddr.IP:
		switch {
		case src.Is4():
			prefix := netaddr.IPPrefixFrom(*src, 32)
			return &prefix, IPv4, nil
		case src.Is6():
			prefix := netaddr.IPPrefixFrom(*src, 128)
			return &prefix, IPv6, nil
		default:
			return nil, "", ErrInvalidIPLength
		}

	case netaddr.IPPrefix:
		ip := src.IP()
		switch {
		case ip.Is4():
			return &src, IPv4, nil
		case ip.Is6():
			return &src, IPv6, nil
		default:
			return nil, "", ErrInvalidIPLength
		}

	case *netaddr.IPPrefix:
		ip := src.IP()
		switch {
		case ip.Is4():
			return src, IPv4, nil
		case ip.Is6():
			return src, IPv6, nil
		default:
			return nil, "", ErrInvalidIPLength
		}

	case string:
		_, network, err := net.ParseCIDR(src)
		switch err {
		case nil:
			prefix, ok := netaddr.FromStdIPNet(network)
			if !ok {
				return nil, "", ErrInvalidIPNet
			}
			ip := prefix.IP()
			switch {
			case ip.Is4():
				return &prefix, IPv4, nil
			case ip.Is6():
				return &prefix, IPv6, nil
			default:
				return nil, "", ErrInvalidIPLength
			}
		default:
			ip, err := netaddr.ParseIP(src)
			if err != nil {
				return nil, "", err
			}
			switch {
			case ip.Is4():
				prefix := netaddr.IPPrefixFrom(ip, 32)
				return &prefix, IPv4, nil
			case ip.Is6():
				prefix := netaddr.IPPrefixFrom(ip, 128)
				return &prefix, IPv6, nil
			default:
				return nil, "", ErrInvalidIPLength
			}
		}
	}

	return nil, "", ErrInvalidPrefixType
}

func (e *Entry) add(prefix *netaddr.IPPrefix, ipType IPType) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	switch ipType {
	case IPv4:
		if !e.hasIPv4Builder() {
			e.ipv4Builder = new(netaddr.IPSetBuilder)
		}
		e.ipv4Builder.AddPrefix(*prefix)
	case IPv6:
		if !e.hasIPv6Builder() {
			e.ipv6Builder = new(netaddr.IPSetBuilder)
		}
		e.ipv6Builder.AddPrefix(*prefix)
	default:
		return ErrInvalidIPType
	}

	return nil
}

func (e *Entry) remove(prefix *netaddr.IPPrefix, ipType IPType) error {
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

func (e *Entry) AddPrefix(cidr interface{}) error {
	prefix, ipType, err := e.processPrefix(cidr)
	if err != nil {
		return err
	}
	if err := e.add(prefix, ipType); err != nil {
		return err
	}
	return nil
}

func (e *Entry) RemovePrefix(cidr string) error {
	prefix, ipType, err := e.processPrefix(cidr)
	if err != nil {
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

type IgnoreIPOption func() IPType

func IgnoreIPv4() IPType {
	return IPv4
}

func IgnoreIPv6() IPType {
	return IPv6
}

type Container interface {
	GetEntry(name string) (*Entry, bool)
	Add(entry *Entry, opts ...IgnoreIPOption) error
	Remove(name string, opts ...IgnoreIPOption)
	Replace(entry *Entry, opts ...IgnoreIPOption)
	Loop() <-chan *Entry
}

type container struct {
	entries *sync.Map // map[name]*Entry
}

func NewContainer() Container {
	return &container{
		entries: new(sync.Map),
	}
}

func (c *container) isValid() bool {
	if c == nil || c.entries == nil {
		return false
	}
	return true
}

func (c *container) GetEntry(name string) (*Entry, bool) {
	if !c.isValid() {
		return nil, false
	}
	val, ok := c.entries.Load(strings.ToUpper(strings.TrimSpace(name)))
	if !ok {
		return nil, false
	}
	return val.(*Entry), true
}

func (c *container) Loop() <-chan *Entry {
	ch := make(chan *Entry, 300)
	go func() {
		c.entries.Range(func(key, value interface{}) bool {
			ch <- value.(*Entry)
			return true
		})
		close(ch)
	}()
	return ch
}

func (c *container) Add(entry *Entry, opts ...IgnoreIPOption) error {
	var ignoreIPType IPType
	for _, opt := range opts {
		if opt != nil {
			ignoreIPType = opt()
		}
	}

	name := entry.GetName()
	val, found := c.GetEntry(name)
	switch found {
	case true:
		var ipv4set, ipv6set *netaddr.IPSet
		var err4, err6 error
		if entry.hasIPv4Builder() {
			ipv4set, err4 = entry.ipv4Builder.IPSet()
			if err4 != nil {
				return err4
			}
		}
		if entry.hasIPv6Builder() {
			ipv6set, err6 = entry.ipv6Builder.IPSet()
			if err6 != nil {
				return err6
			}
		}
		switch ignoreIPType {
		case IPv4:
			if !val.hasIPv6Builder() {
				val.ipv6Builder = new(netaddr.IPSetBuilder)
			}
			val.ipv6Builder.AddSet(ipv6set)
		case IPv6:
			if !val.hasIPv4Builder() {
				val.ipv4Builder = new(netaddr.IPSetBuilder)
			}
			val.ipv4Builder.AddSet(ipv4set)
		default:
			if !val.hasIPv4Builder() {
				val.ipv4Builder = new(netaddr.IPSetBuilder)
			}
			if !val.hasIPv6Builder() {
				val.ipv6Builder = new(netaddr.IPSetBuilder)
			}
			val.ipv4Builder.AddSet(ipv4set)
			val.ipv6Builder.AddSet(ipv6set)
		}
		c.entries.Store(name, val)

	case false:
		switch ignoreIPType {
		case IPv4:
			entry.ipv4Builder = nil
		case IPv6:
			entry.ipv6Builder = nil
		}
		c.entries.Store(name, entry)
	}

	return nil
}

func (c *container) Remove(name string, opts ...IgnoreIPOption) {
	val, found := c.GetEntry(name)
	if !found {
		log.Printf("failed to remove non-existent entry %s", name)
		return
	}

	var ignoreIPType IPType
	for _, opt := range opts {
		if opt != nil {
			ignoreIPType = opt()
		}
	}

	switch ignoreIPType {
	case IPv4:
		val.ipv6Builder = nil
		c.entries.Store(name, val)
	case IPv6:
		val.ipv4Builder = nil
		c.entries.Store(name, val)
	default:
		c.entries.Delete(name)
	}
}

func (c *container) Replace(entry *Entry, opts ...IgnoreIPOption) {
	var ignoreIPType IPType
	for _, opt := range opts {
		if opt != nil {
			ignoreIPType = opt()
		}
	}

	switch ignoreIPType {
	case IPv4:
		entry.ipv4Builder = nil
	case IPv6:
		entry.ipv6Builder = nil
	}

	name := entry.GetName()
	c.entries.Store(name, entry)
}
