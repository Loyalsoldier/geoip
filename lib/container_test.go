package lib

import (
	"sort"
	"testing"
)

func newEntryWith(t *testing.T, name string, prefixes ...string) *Entry {
	t.Helper()
	e := NewEntry(name)
	for _, p := range prefixes {
		if err := e.AddPrefix(p); err != nil {
			t.Fatalf("AddPrefix(%q): %v", p, err)
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
	got, err := e.MarshalText()
	if err != nil {
		return []string{}
	}
	return got
}

func TestContainerInvalid(t *testing.T) {
	c := &container{}
	if c.Len() != 0 {
		t.Error("invalid container Len should be 0")
	}
	if e, ok := c.GetEntry("x"); ok || e != nil {
		t.Error("invalid container GetEntry should return nil, false")
	}
}

func TestContainerGetEntryLenLoop(t *testing.T) {
	c := NewContainer()
	if c.Len() != 0 {
		t.Fatal("new container should be empty")
	}
	for _, n := range []string{"a", "b", "c"} {
		if err := c.Add(newEntryWith(t, n, "1.1.1.1")); err != nil {
			t.Fatal(err)
		}
	}
	if c.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", c.Len())
	}
	tests := []struct {
		name  string
		found bool
	}{
		{"a", true}, {" B ", true}, {"c", true}, {"d", false}, {"", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := c.GetEntry(tt.name)
			if ok != tt.found {
				t.Errorf("GetEntry(%q) found = %v, want %v", tt.name, ok, tt.found)
			}
		})
	}
	names := []string{}
	for e := range c.Loop() {
		names = append(names, e.GetName())
	}
	sort.Strings(names)
	assertStrings(t, names, []string{"A", "B", "C"})
}

func TestContainerAdd(t *testing.T) {
	tests := []struct {
		name     string
		existing []string // prefixes of an existing entry; nil means no existing entry
		adding   []string
		opts     []IgnoreIPOption
		want     []string
	}{
		{"new entry", nil, []string{"1.1.1.1", "2001:db8::1"}, nil, []string{"1.1.1.1/32", "2001:db8::1/128"}},
		{"new entry nil option", nil, []string{"1.1.1.1", "2001:db8::1"}, []IgnoreIPOption{nil}, []string{"1.1.1.1/32", "2001:db8::1/128"}},
		{"new entry ignore ipv4", nil, []string{"1.1.1.1", "2001:db8::1"}, []IgnoreIPOption{IgnoreIPv4}, []string{"2001:db8::1/128"}},
		{"new entry ignore ipv6", nil, []string{"1.1.1.1", "2001:db8::1"}, []IgnoreIPOption{IgnoreIPv6}, []string{"1.1.1.1/32"}},
		{"merge ipv4 /24 into /23", []string{"192.168.0.3/24"}, []string{"192.168.1.5/24"}, nil, []string{"192.168.0.0/23"}},
		{"merge ipv6 /33 into /32", []string{"2001:db8::/33"}, []string{"2001:db8:8000::/33"}, nil, []string{"2001:db8::/32"}},
		{"merge both into empty-builder entry", []string{}, []string{"10.0.0.0/8", "2001:db8::/32"}, nil, []string{"10.0.0.0/8", "2001:db8::/32"}},
		{"merge ignore ipv4", []string{"10.0.0.0/8"}, []string{"11.0.0.0/8", "2001:db8::/32"}, []IgnoreIPOption{IgnoreIPv4}, []string{"10.0.0.0/8", "2001:db8::/32"}},
		{"merge ignore ipv6", []string{"2001:db8::/32"}, []string{"10.0.0.0/8", "2001:db9::/32"}, []IgnoreIPOption{IgnoreIPv6}, []string{"10.0.0.0/8", "2001:db8::/32"}},
		{"merge ignore ipv4 existing ipv6", []string{"2001:db8::/33"}, []string{"2001:db8:8000::/33"}, []IgnoreIPOption{IgnoreIPv4}, []string{"2001:db8::/32"}},
		{"merge ignore ipv6 existing ipv4", []string{"192.168.0.0/24"}, []string{"::ffff:192.168.1.0/120"}, []IgnoreIPOption{IgnoreIPv6}, []string{"192.168.0.0/23"}},
		{"merge empty adding entry", []string{"1.1.1.1"}, []string{}, nil, []string{"1.1.1.1/32"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewContainer()
			if tt.existing != nil {
				existing := newEntryWith(t, "test", tt.existing...)
				// Cache sets to verify they get invalidated on merge.
				_, _ = existing.MarshalText()
				if err := c.Add(existing); err != nil {
					t.Fatal(err)
				}
			}
			if err := c.Add(newEntryWith(t, "TEST", tt.adding...), tt.opts...); err != nil {
				t.Fatal(err)
			}
			if c.Len() != 1 {
				t.Fatalf("Len() = %d, want 1", c.Len())
			}
			assertStrings(t, entryText(t, c, "test"), tt.want)
		})
	}
}

func TestContainerAddBrokenBuilder(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Entry)
	}{
		{"ipv4", func(e *Entry) { e.ipv4Builder = brokenBuilder() }},
		{"ipv6", func(e *Entry) { e.ipv6Builder = brokenBuilder() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewContainer()
			_ = c.Add(newEntryWith(t, "x", "1.1.1.1"))
			// IPSetBuilder errors are consumed by IPSet(), so use fresh entries.
			bad := func() *Entry {
				e := NewEntry("x")
				tt.setup(e)
				return e
			}
			if err := c.Add(bad()); err == nil {
				t.Fatal("expected error")
			}
			if err := c.Remove(bad(), CaseRemovePrefix); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestContainerRemove(t *testing.T) {
	base := []string{"192.168.0.0/23", "2001:db8::/32"}
	tests := []struct {
		name       string
		existing   []string
		removing   []string
		rCase      CaseRemove
		opts       []IgnoreIPOption
		want       []string
		wantGone   bool
		wantErr    bool
		removeName string
	}{
		{name: "not found", existing: base, removeName: "other", rCase: CaseRemovePrefix, wantErr: true},
		{name: "unknown case", existing: base, rCase: CaseRemove(99), wantErr: true},
		{name: "split ipv4 /23 and ipv6 /32", existing: base, removing: []string{"192.168.1.0/24", "2001:db8:8000::/33"}, rCase: CaseRemovePrefix, want: []string{"192.168.0.0/24", "2001:db8::/33"}},
		{name: "prefix with nil option", existing: base, removing: []string{"192.168.1.0/24"}, rCase: CaseRemovePrefix, opts: []IgnoreIPOption{nil}, want: []string{"192.168.0.0/24", "2001:db8::/32"}},
		{name: "prefix ignore ipv4", existing: base, removing: []string{"192.168.1.0/24", "2001:db8:8000::/33"}, rCase: CaseRemovePrefix, opts: []IgnoreIPOption{IgnoreIPv4}, want: []string{"192.168.0.0/23", "2001:db8::/33"}},
		{name: "prefix ignore ipv6", existing: base, removing: []string{"::ffff:192.168.1.0/120", "2001:db8:8000::/33"}, rCase: CaseRemovePrefix, opts: []IgnoreIPOption{IgnoreIPv6}, want: []string{"192.168.0.0/24", "2001:db8::/32"}},
		{name: "prefix on entry missing builders", existing: []string{}, removing: []string{"1.1.1.1", "2001:db8::1"}, rCase: CaseRemovePrefix, want: []string{}},
		{name: "prefix ignore ipv4 missing builder", existing: []string{}, removing: []string{"2001:db8::1"}, rCase: CaseRemovePrefix, opts: []IgnoreIPOption{IgnoreIPv4}, want: []string{}},
		{name: "prefix ignore ipv6 missing builder", existing: []string{}, removing: []string{"1.1.1.1"}, rCase: CaseRemovePrefix, opts: []IgnoreIPOption{IgnoreIPv6}, want: []string{}},
		{name: "prefix empty removing entry", existing: base, rCase: CaseRemovePrefix, want: []string{"192.168.0.0/23", "2001:db8::/32"}},
		{name: "entry", existing: base, rCase: CaseRemoveEntry, wantGone: true},
		{name: "entry ignore ipv4", existing: base, rCase: CaseRemoveEntry, opts: []IgnoreIPOption{IgnoreIPv4}, want: []string{"192.168.0.0/23"}},
		{name: "entry ignore ipv6", existing: base, rCase: CaseRemoveEntry, opts: []IgnoreIPOption{IgnoreIPv6}, want: []string{"2001:db8::/32"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewContainer()
			existing := newEntryWith(t, "test", tt.existing...)
			_, _ = existing.MarshalText()
			_ = c.Add(existing)

			name := "test"
			if tt.removeName != "" {
				name = tt.removeName
			}
			err := c.Remove(newEntryWith(t, name, tt.removing...), tt.rCase, tt.opts...)
			assertErr(t, err, nil, tt.wantErr)
			if tt.wantErr {
				return
			}
			if tt.wantGone {
				if _, ok := c.GetEntry("test"); ok || c.Len() != 0 {
					t.Fatal("entry should be removed")
				}
				return
			}
			assertStrings(t, entryText(t, c, "test"), tt.want)
		})
	}
}

func TestContainerLookup(t *testing.T) {
	c := NewContainer()
	_ = c.Add(newEntryWith(t, "cn", "192.168.0.0/23", "2001:db8::/32"))
	_ = c.Add(newEntryWith(t, "us", "192.168.0.0/24", "10.0.0.0/8"))
	_ = c.Add(newEntryWith(t, "v6only", "2001:db8::/48"))
	_ = c.Add(NewEntry("empty"))

	tests := []struct {
		name       string
		query      string
		searchList []string
		want       []string
		wantErr    error
		anyErr     bool
	}{
		{name: "ipv4 ip in two", query: "192.168.0.1", want: []string{"CN", "US"}},
		{name: "ipv4 ip in one", query: "192.168.1.1", want: []string{"CN"}},
		{name: "ipv4 ip not found", query: "8.8.8.8", want: []string{}},
		{name: "ipv4 cidr contained", query: "192.168.0.0/24", want: []string{"CN", "US"}},
		{name: "ipv4 cidr /23 only cn", query: "192.168.0.5/23", want: []string{"CN"}},
		{name: "ipv4 cidr too wide", query: "192.168.0.0/22", want: []string{}},
		{name: "ipv6 ip", query: "2001:db8::1", want: []string{"CN", "V6ONLY"}},
		{name: "ipv6 cidr", query: "2001:db8:1::/48", want: []string{"CN"}},
		{name: "mapped ip", query: "::ffff:192.168.1.1", want: []string{"CN"}},
		{name: "mapped cidr /120", query: "::ffff:192.168.0.0/120", want: []string{"CN", "US"}},
		{name: "mapped cidr /119", query: "::ffff:192.168.0.0/119", want: []string{"CN"}},
		{name: "mapped cidr /96", query: "::ffff:0.0.0.0/96", want: []string{}},
		{name: "mapped cidr bits too small", query: "::ffff:192.168.0.0/95", wantErr: ErrInvalidCIDR, anyErr: true},
		{name: "mapped cidr bits too large", query: "::ffff:192.168.0.0/129", anyErr: true},
		{name: "malformed mapped ip", query: "::ffff:192.168.0", anyErr: true},
		{name: "malformed ip", query: "192.168.0", anyErr: true},
		{name: "malformed cidr", query: "192.168.0.0/33", anyErr: true},
		{name: "search list filters", query: "192.168.0.1", searchList: []string{" us ", "", "v6only"}, want: []string{"US"}},
		{name: "search list all blank", query: "192.168.0.1", searchList: []string{"  "}, want: []string{"CN", "US"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found, err := c.Lookup(tt.query, tt.searchList...)
			assertErr(t, err, tt.wantErr, tt.anyErr)
			if tt.anyErr {
				return
			}
			sort.Strings(got)
			assertStrings(t, got, tt.want)
			if found != (len(tt.want) > 0) {
				t.Errorf("found = %v, want %v", found, len(tt.want) > 0)
			}
		})
	}
}

func TestContainerLookupBrokenBuilder(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Entry)
		query string
	}{
		{"ipv4", func(e *Entry) { e.ipv4Builder = brokenBuilder() }, "1.1.1.1"},
		{"ipv6", func(e *Entry) { e.ipv6Builder = brokenBuilder() }, "2001:db8::/64"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewContainer()
			e := NewEntry("bad")
			tt.setup(e)
			_ = c.Add(e)
			got, found, err := c.Lookup(tt.query)
			if err == nil || found || got != nil {
				t.Fatalf("expected error, got %v %v %v", got, found, err)
			}
		})
	}
}
