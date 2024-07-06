package lib

import (
	"log"
	"strings"
	"sync"

	"go4.org/netipx"
)

type Container interface {
	GetEntry(name string) (*Entry, bool)
	Add(entry *Entry, opts ...IgnoreIPOption) error
	Remove(name string, opts ...IgnoreIPOption)
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
		c.entries.Range(func(key, value any) bool {
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
		var ipv4set, ipv6set *netipx.IPSet
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
				val.ipv6Builder = new(netipx.IPSetBuilder)
			}
			val.ipv6Builder.AddSet(ipv6set)
		case IPv6:
			if !val.hasIPv4Builder() {
				val.ipv4Builder = new(netipx.IPSetBuilder)
			}
			val.ipv4Builder.AddSet(ipv4set)
		default:
			if !val.hasIPv4Builder() {
				val.ipv4Builder = new(netipx.IPSetBuilder)
			}
			if !val.hasIPv6Builder() {
				val.ipv6Builder = new(netipx.IPSetBuilder)
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
