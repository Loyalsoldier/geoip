package lib

import (
	"sort"
	"testing"
)

func newEntryWith(t *testing.T, name string, cidrs ...string) *Entry {
	t.Helper()
	e := NewEntry(name)
	for _, c := range cidrs {
		if err := e.AddPrefix(c); err != nil {
			t.Fatalf("AddPrefix(%q): %v", c, err)
		}
	}
	return e
}

func entryText(t *testing.T, c Container, name string) []string {
	t.Helper()
	e, ok := c.GetEntry(name)
	if !ok {
		t.Fatalf("entry %s not found", name)
	}
	text, err := e.MarshalText()
	if err != nil {
		return nil
	}
	return text
}

func TestContainerBasics(t *testing.T) {
	c := NewContainer()
	if c.Len() != 0 {
		t.Fatalf("Len = %d", c.Len())
	}
	if _, ok := c.GetEntry("cn"); ok {
		t.Error("unexpected entry")
	}
	if err := c.Add(newEntryWith(t, "cn", "1.0.0.0/24")); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(newEntryWith(t, "us", "2.0.0.0/24")); err != nil {
		t.Fatal(err)
	}
	if c.Len() != 2 {
		t.Fatalf("Len = %d", c.Len())
	}
	if e, ok := c.GetEntry("  Cn "); !ok || e.GetName() != "CN" {
		t.Error("GetEntry should be case-insensitive and trim spaces")
	}
	names := make([]string, 0)
	for e := range c.Loop() {
		names = append(names, e.GetName())
	}
	sort.Strings(names)
	compareStrings(t, names, []string{"CN", "US"})

	invalid := &container{}
	if invalid.Len() != 0 {
		t.Error("invalid container Len should be 0")
	}
	if _, ok := invalid.GetEntry("cn"); ok {
		t.Error("invalid container should not find entries")
	}
}

func TestContainerAdd(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		add      []string
		opts     []IgnoreIPOption
		want     []string
	}{
		{"new entry", nil, []string{"1.0.0.0/24", "2001:db8::/32"}, nil, []string{"1.0.0.0/24", "2001:db8::/32"}},
		{"new entry ignore ipv4", nil, []string{"1.0.0.0/24", "2001:db8::/32"}, []IgnoreIPOption{IgnoreIPv4}, []string{"2001:db8::/32"}},
		{"new entry ignore ipv6", nil, []string{"1.0.0.0/24", "2001:db8::/32"}, []IgnoreIPOption{nil, IgnoreIPv6}, []string{"1.0.0.0/24"}},
		{"merge ipv4 into /23", []string{"192.168.0.3/24"}, []string{"192.168.1.5/24"}, nil, []string{"192.168.0.0/23"}},
		{"merge ipv6", []string{"2001:db8::/33"}, []string{"2001:db8:8000::/33"}, nil, []string{"2001:db8::/32"}},
		{"merge both into empty existing", nil, nil, nil, nil},
		{"merge into existing without builders", []string{}, []string{"1.0.0.0/24", "2001:db8::/32"}, nil, []string{"1.0.0.0/24", "2001:db8::/32"}},
		{"merge ignore ipv4", []string{"1.0.0.0/24"}, []string{"1.0.1.0/24", "2001:db8::/32"}, []IgnoreIPOption{IgnoreIPv4}, []string{"1.0.0.0/24", "2001:db8::/32"}},
		{"merge ignore ipv4 existing has ipv6", []string{"2001:db8::/33"}, []string{"2001:db8:8000::/33"}, []IgnoreIPOption{IgnoreIPv4}, []string{"2001:db8::/32"}},
		{"merge ignore ipv6", []string{"2001:db8::/32"}, []string{"1.0.0.0/24", "2001:db9::/32"}, []IgnoreIPOption{IgnoreIPv6}, []string{"1.0.0.0/24", "2001:db8::/32"}},
		{"merge ignore ipv6 existing has ipv4", []string{"1.0.0.0/24"}, []string{"1.0.1.0/24"}, []IgnoreIPOption{IgnoreIPv6}, []string{"1.0.0.0/23"}},
		{"merge ipv4-mapped", []string{"192.168.0.0/24"}, []string{"::ffff:192.168.1.0/120"}, nil, []string{"192.168.0.0/23"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewContainer()
			if tt.existing != nil {
				if err := c.Add(newEntryWith(t, "x", tt.existing...)); err != nil {
					t.Fatal(err)
				}
			}
			if err := c.Add(newEntryWith(t, "x", tt.add...), tt.opts...); err != nil {
				t.Fatal(err)
			}
			compareStrings(t, entryText(t, c, "x"), tt.want)
		})
	}
}

func TestContainerAddErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(e *Entry)
	}{
		{"broken ipv4", func(e *Entry) { e.ipv4Builder = brokenBuilder() }},
		{"broken ipv6", func(e *Entry) { e.ipv6Builder = brokenBuilder() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewContainer()
			_ = c.Add(NewEntry("x"))
			e := NewEntry("x")
			tt.setup(e)
			if err := c.Add(e); err == nil {
				t.Error("expected error")
			}
			e = NewEntry("x")
			tt.setup(e)
			if err := c.Remove(e, CaseRemovePrefix); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestContainerRemove(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		remove   []string
		rCase    CaseRemove
		opts     []IgnoreIPOption
		want     []string
		gone     bool
		wantErr  bool
	}{
		{"split ipv4 /23", []string{"192.168.0.0/23"}, []string{"192.168.1.5/24"}, CaseRemovePrefix, nil, []string{"192.168.0.0/24"}, false, false},
		{"split ipv6 /32", []string{"2001:db8::/32"}, []string{"2001:db8:8000::/33"}, CaseRemovePrefix, nil, []string{"2001:db8::/33"}, false, false},
		{"split ipv4-mapped", []string{"192.168.0.0/23"}, []string{"::ffff:192.168.0.0/120"}, CaseRemovePrefix, nil, []string{"192.168.1.0/24"}, false, false},
		{"remove both", []string{"1.0.0.0/23", "2001:db8::/32"}, []string{"1.0.0.0/24", "2001:db8::/33"}, CaseRemovePrefix, nil, []string{"1.0.1.0/24", "2001:db8:8000::/33"}, false, false},
		{"remove from existing without builders", []string{}, []string{"1.0.0.0/24", "2001:db8::/32"}, CaseRemovePrefix, nil, nil, false, false},
		{"remove prefix ignore ipv4", []string{"1.0.0.0/23", "2001:db8::/32"}, []string{"1.0.0.0/24", "2001:db8::/33"}, CaseRemovePrefix, []IgnoreIPOption{IgnoreIPv4}, []string{"1.0.0.0/23", "2001:db8:8000::/33"}, false, false},
		{"remove prefix ignore ipv4 no ipv6 builder", []string{"1.0.0.0/23"}, []string{"2001:db8::/33"}, CaseRemovePrefix, []IgnoreIPOption{IgnoreIPv4}, []string{"1.0.0.0/23"}, false, false},
		{"remove prefix ignore ipv6", []string{"1.0.0.0/23", "2001:db8::/32"}, []string{"1.0.0.0/24", "2001:db8::/33"}, CaseRemovePrefix, []IgnoreIPOption{IgnoreIPv6}, []string{"1.0.1.0/24", "2001:db8::/32"}, false, false},
		{"remove prefix ignore ipv6 no ipv4 builder", []string{"2001:db8::/32"}, []string{"1.0.0.0/24"}, CaseRemovePrefix, []IgnoreIPOption{IgnoreIPv6}, []string{"2001:db8::/32"}, false, false},
		{"remove entry", []string{"1.0.0.0/24"}, nil, CaseRemoveEntry, nil, nil, true, false},
		{"remove entry ignore ipv4", []string{"1.0.0.0/24", "2001:db8::/32"}, nil, CaseRemoveEntry, []IgnoreIPOption{IgnoreIPv4}, []string{"1.0.0.0/24"}, false, false},
		{"remove entry ignore ipv6", []string{"1.0.0.0/24", "2001:db8::/32"}, nil, CaseRemoveEntry, []IgnoreIPOption{nil, IgnoreIPv6}, []string{"2001:db8::/32"}, false, false},
		{"unknown case", []string{"1.0.0.0/24"}, nil, CaseRemove(99), nil, nil, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewContainer()
			if err := c.Add(newEntryWith(t, "x", tt.existing...)); err != nil {
				t.Fatal(err)
			}
			err := c.Remove(newEntryWith(t, "x", tt.remove...), tt.rCase, tt.opts...)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.gone {
				if _, ok := c.GetEntry("x"); ok {
					t.Fatal("entry should be removed")
				}
				return
			}
			compareStrings(t, entryText(t, c, "x"), tt.want)
		})
	}

	t.Run("not found", func(t *testing.T) {
		if err := NewContainer().Remove(NewEntry("x"), CaseRemoveEntry); err == nil {
			t.Error("expected error")
		}
	})
}

func TestContainerLookup(t *testing.T) {
	c := NewContainer()
	_ = c.Add(newEntryWith(t, "a", "192.168.0.0/23", "2001:db8::/32"))
	_ = c.Add(newEntryWith(t, "b", "192.168.1.0/24"))
	_ = c.Add(newEntryWith(t, "c", "2001:db8:1::/48"))

	tests := []struct {
		name    string
		in      string
		search  []string
		want    []string
		found   bool
		wantErr bool
	}{
		{"ipv4 multiple", "192.168.1.1", nil, []string{"A", "B"}, true, false},
		{"ipv4 single", "192.168.0.1", nil, []string{"A"}, true, false},
		{"ipv4 none", "10.0.0.1", nil, []string{}, false, false},
		{"ipv4 cidr", "192.168.1.0/25", nil, []string{"A", "B"}, true, false},
		{"ipv4 cidr wider", "192.168.0.0/22", nil, []string{}, false, false},
		{"ipv4 search list", "192.168.1.1", []string{" b ", ""}, []string{"B"}, true, false},
		{"ipv4-mapped ip", "::ffff:192.168.1.1", nil, []string{"A", "B"}, true, false},
		{"ipv4-mapped cidr /120", "::ffff:192.168.1.0/120", nil, []string{"A", "B"}, true, false},
		{"ipv4-mapped cidr /119", "::ffff:192.168.0.0/119", nil, []string{"A"}, true, false},
		{"ipv4-mapped cidr /95", "::ffff:192.168.0.0/95", nil, nil, false, true},
		{"ipv6", "2001:db8:1::1", nil, []string{"A", "C"}, true, false},
		{"ipv6 cidr", "2001:db8::/33", nil, []string{"A"}, true, false},
		{"ipv6 none", "2001:db9::1", nil, []string{}, false, false},
		{"invalid ip", "abc", nil, nil, false, true},
		{"invalid cidr", "1.2.3.4/33", nil, nil, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found, err := c.Lookup(tt.in, tt.search...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v", err)
			}
			if found != tt.found {
				t.Errorf("found = %v, want %v", found, tt.found)
			}
			sort.Strings(got)
			if tt.want != nil {
				compareStrings(t, got, tt.want)
			}
		})
	}

	t.Run("broken builder", func(t *testing.T) {
		c := NewContainer()
		e := NewEntry("x")
		e.ipv4Builder = brokenBuilder()
		_ = c.Add(e)
		if _, _, err := c.Lookup("1.1.1.1"); err == nil {
			t.Error("expected error")
		}
	})
}
