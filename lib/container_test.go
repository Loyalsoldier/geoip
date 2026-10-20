package lib

import (
	"slices"
	"strings"
	"testing"
)

// entrySpec describes an entry to be created in tests.
type entrySpec struct {
	name  string
	cidrs []string
}

// newTestContainer creates a container with the given entries.
func newTestContainer(t *testing.T, specs ...entrySpec) Container {
	t.Helper()
	c := NewContainer()
	for _, spec := range specs {
		if err := c.Add(newTestEntry(t, spec.name, spec.cidrs...)); err != nil {
			t.Fatalf("Add(%s) failed: %v", spec.name, err)
		}
	}
	return c
}

func TestNewContainer(t *testing.T) {
	t.Parallel()

	c := NewContainer()
	if c == nil {
		t.Fatal("NewContainer() returned nil")
	}
	if got := c.Len(); got != 0 {
		t.Fatalf("Len() = %d, want 0", got)
	}
	if entry, found := c.GetEntry("cn"); found || entry != nil {
		t.Fatalf("GetEntry() = %v, %v, want nil, false", entry, found)
	}
	count := 0
	for range c.Loop() {
		count++
	}
	if count != 0 {
		t.Fatalf("Loop() yielded %d entries, want 0", count)
	}
}

func TestContainerZeroValue(t *testing.T) {
	t.Parallel()

	c := &container{}
	if c.isValid() {
		t.Fatal("zero-value container should be invalid")
	}
	if got := c.Len(); got != 0 {
		t.Fatalf("Len() = %d, want 0", got)
	}
	if entry, found := c.GetEntry("cn"); found || entry != nil {
		t.Fatalf("GetEntry() = %v, %v, want nil, false", entry, found)
	}
	count := 0
	for range c.Loop() {
		count++
	}
	if count != 0 {
		t.Fatalf("Loop() yielded %d entries, want 0", count)
	}
}

func TestContainerGetEntry(t *testing.T) {
	t.Parallel()

	c := newTestContainer(t,
		entrySpec{name: "cn", cidrs: []string{"1.0.1.0/24"}},
		entrySpec{name: "US", cidrs: []string{"8.8.8.0/24"}},
	)

	tests := []struct {
		name      string
		search    string
		wantFound bool
		wantName  string
		wantCIDRs []string
	}{
		{name: "exact uppercase", search: "CN", wantFound: true, wantName: "CN", wantCIDRs: []string{"1.0.1.0/24"}},
		{name: "lowercase", search: "cn", wantFound: true, wantName: "CN", wantCIDRs: []string{"1.0.1.0/24"}},
		{name: "mixed case with spaces", search: "  uS \t", wantFound: true, wantName: "US", wantCIDRs: []string{"8.8.8.0/24"}},
		{name: "missing", search: "jp", wantFound: false},
		{name: "empty", search: "", wantFound: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, found := c.GetEntry(tt.search)
			if found != tt.wantFound {
				t.Fatalf("found = %v, want %v", found, tt.wantFound)
			}
			if !tt.wantFound {
				if entry != nil {
					t.Fatalf("entry = %v, want nil", entry)
				}
				return
			}
			if entry.GetName() != tt.wantName {
				t.Fatalf("name = %q, want %q", entry.GetName(), tt.wantName)
			}
			assertStringSlice(t, entryText(t, entry), tt.wantCIDRs)
		})
	}
}

func TestContainerLenAndLoop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		specs []entrySpec
		want  []string
	}{
		{name: "empty", specs: nil, want: []string{}},
		{name: "single", specs: []entrySpec{{name: "a", cidrs: []string{"1.1.1.1"}}}, want: []string{"A"}},
		{
			name: "multiple",
			specs: []entrySpec{
				{name: "c", cidrs: []string{"3.3.3.3"}},
				{name: "a", cidrs: []string{"1.1.1.1"}},
				{name: "b", cidrs: []string{"2001:db8::1"}},
			},
			want: []string{"A", "B", "C"},
		},
		{
			name: "same name is merged",
			specs: []entrySpec{
				{name: "a", cidrs: []string{"1.1.1.1"}},
				{name: "A", cidrs: []string{"2.2.2.2"}},
			},
			want: []string{"A"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newTestContainer(t, tt.specs...)
			if got := c.Len(); got != len(tt.want) {
				t.Fatalf("Len() = %d, want %d", got, len(tt.want))
			}
			got := make([]string, 0, c.Len())
			for entry := range c.Loop() {
				got = append(got, entry.GetName())
			}
			slices.Sort(got)
			assertStringSlice(t, got, tt.want)
		})
	}
}

func TestContainerAdd(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		existing []string // nil means the entry does not exist yet
		add      []string
		opts     []IgnoreIPOption
		want     []string
	}{
		{
			name: "new entry",
			add:  []string{"192.168.0.0/24", "2001:db8::/32"},
			want: []string{"192.168.0.0/24", "2001:db8::/32"},
		},
		{
			name: "new entry with nil option",
			add:  []string{"192.168.0.0/24", "2001:db8::/32"},
			opts: []IgnoreIPOption{nil},
			want: []string{"192.168.0.0/24", "2001:db8::/32"},
		},
		{
			name: "new entry ignoring IPv4",
			add:  []string{"192.168.0.0/24", "2001:db8::/32"},
			opts: []IgnoreIPOption{IgnoreIPv4},
			want: []string{"2001:db8::/32"},
		},
		{
			name: "new entry ignoring IPv6",
			add:  []string{"192.168.0.0/24", "2001:db8::/32"},
			opts: []IgnoreIPOption{IgnoreIPv6},
			want: []string{"192.168.0.0/24"},
		},
		{
			name:     "merge IPv4 into existing entry",
			existing: []string{"192.168.0.3/24"},
			add:      []string{"192.168.1.5/24"},
			want:     []string{"192.168.0.0/23"},
		},
		{
			name:     "merge IPv4-mapped IPv6 into existing entry",
			existing: []string{"192.168.0.0/24"},
			add:      []string{"::ffff:192.168.1.0/120"},
			want:     []string{"192.168.0.0/23"},
		},
		{
			name:     "merge IPv6 into existing entry",
			existing: []string{"2001:db8::/33"},
			add:      []string{"2001:db8:8000::/33"},
			want:     []string{"2001:db8::/32"},
		},
		{
			name:     "merge IPv4 and IPv6 into existing entry",
			existing: []string{"192.168.0.0/24", "2001:db8::/33"},
			add:      []string{"192.168.1.0/24", "2001:db8:8000::/33"},
			want:     []string{"192.168.0.0/23", "2001:db8::/32"},
		},
		{
			name:     "merge into existing entry ignoring IPv4",
			existing: []string{"192.168.0.0/24", "2001:db8::/33"},
			add:      []string{"192.168.1.0/24", "2001:db8:8000::/33"},
			opts:     []IgnoreIPOption{IgnoreIPv4},
			want:     []string{"192.168.0.0/24", "2001:db8::/32"},
		},
		{
			name:     "merge into existing entry ignoring IPv6",
			existing: []string{"192.168.0.0/24", "2001:db8::/33"},
			add:      []string{"192.168.1.0/24", "2001:db8:8000::/33"},
			opts:     []IgnoreIPOption{IgnoreIPv6},
			want:     []string{"192.168.0.0/23", "2001:db8::/33"},
		},
		{
			name:     "add IPv6 to existing IPv4-only entry",
			existing: []string{"1.1.1.0/24"},
			add:      []string{"2001:db8::/32"},
			want:     []string{"1.1.1.0/24", "2001:db8::/32"},
		},
		{
			name:     "add IPv4 to existing IPv6-only entry",
			existing: []string{"2001:db8::/32"},
			add:      []string{"1.1.1.0/24"},
			want:     []string{"1.1.1.0/24", "2001:db8::/32"},
		},
		{
			name:     "add IPv6 to existing IPv4-only entry ignoring IPv4",
			existing: []string{"1.1.1.0/24"},
			add:      []string{"1.1.2.0/24", "2001:db8::/32"},
			opts:     []IgnoreIPOption{IgnoreIPv4},
			want:     []string{"1.1.1.0/24", "2001:db8::/32"},
		},
		{
			name:     "add IPv4 to existing IPv6-only entry ignoring IPv6",
			existing: []string{"2001:db8::/32"},
			add:      []string{"1.1.1.0/24", "2001:db9::/32"},
			opts:     []IgnoreIPOption{IgnoreIPv6},
			want:     []string{"1.1.1.0/24", "2001:db8::/32"},
		},
		{
			name:     "add IPv4-only entry ignoring IPv4 is a no-op",
			existing: []string{"1.1.1.0/24"},
			add:      []string{"1.1.2.0/24"},
			opts:     []IgnoreIPOption{IgnoreIPv4},
			want:     []string{"1.1.1.0/24"},
		},
		{
			name:     "add IPv6-only entry ignoring IPv6 is a no-op",
			existing: []string{"2001:db8::/32"},
			add:      []string{"2001:db9::/32"},
			opts:     []IgnoreIPOption{IgnoreIPv6},
			want:     []string{"2001:db8::/32"},
		},
		{
			name:     "add empty entry to existing entry",
			existing: []string{"1.1.1.0/24", "2001:db8::/32"},
			add:      nil,
			want:     []string{"1.1.1.0/24", "2001:db8::/32"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewContainer()
			if tt.existing != nil {
				existing := newTestEntry(t, "test", tt.existing...)
				if err := c.Add(existing); err != nil {
					t.Fatalf("Add(existing) failed: %v", err)
				}
				// Build the cached sets so that Add must invalidate them.
				_ = entryText(t, existing)
			}

			// Use a differently-cased name to verify the lookup is normalized.
			toAdd := newTestEntry(t, " Test ", tt.add...)
			if err := c.Add(toAdd, tt.opts...); err != nil {
				t.Fatalf("Add() failed: %v", err)
			}

			if got := c.Len(); got != 1 {
				t.Fatalf("Len() = %d, want 1", got)
			}
			assertStringSlice(t, containerText(t, c, "test"), tt.want)

			// The added entry itself must not be changed when merged into an existing entry.
			if tt.existing != nil {
				want := entryText(t, newTestEntry(t, "test", tt.add...))
				assertStringSlice(t, entryText(t, toAdd), want)
			}
		})
	}
}

func TestContainerAddError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		poison4 bool
		poison6 bool
	}{
		{name: "IPv4 builder error", poison4: true},
		{name: "IPv6 builder error", poison6: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newTestContainer(t, entrySpec{name: "test", cidrs: []string{"1.1.1.0/24", "2001:db8::/32"}})

			entry := newTestEntry(t, "test", "1.1.2.0/24", "2001:db9::/32")
			if tt.poison4 {
				poisonIPv4Builder(entry)
			}
			if tt.poison6 {
				poisonIPv6Builder(entry)
			}

			assertErrorContains(t, c.Add(entry), "AddPrefix(invalid IP/-1)")
			assertStringSlice(t, containerText(t, c, "test"), []string{"1.1.1.0/24", "2001:db8::/32"})
		})
	}
}

func TestContainerRemove(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		existing  []string
		remove    []string
		rCase     CaseRemove
		opts      []IgnoreIPOption
		want      []string
		wantFound bool
	}{
		{
			name:      "split IPv4 CIDR",
			existing:  []string{"192.168.0.0/23"},
			remove:    []string{"192.168.1.5/24"},
			rCase:     CaseRemovePrefix,
			want:      []string{"192.168.0.0/24"},
			wantFound: true,
		},
		{
			name:      "split IPv4 CIDR by IPv4-mapped IPv6 CIDR",
			existing:  []string{"192.168.0.0/23"},
			remove:    []string{"::ffff:192.168.0.0/120"},
			rCase:     CaseRemovePrefix,
			want:      []string{"192.168.1.0/24"},
			wantFound: true,
		},
		{
			name:      "split IPv6 CIDR",
			existing:  []string{"2001:db8::/32"},
			remove:    []string{"2001:db8:8000::/33"},
			rCase:     CaseRemovePrefix,
			want:      []string{"2001:db8::/33"},
			wantFound: true,
		},
		{
			name:      "split IPv4 and IPv6 CIDRs",
			existing:  []string{"192.168.0.0/23", "2001:db8::/32"},
			remove:    []string{"192.168.0.0/24", "2001:db8::/33"},
			rCase:     CaseRemovePrefix,
			want:      []string{"192.168.1.0/24", "2001:db8:8000::/33"},
			wantFound: true,
		},
		{
			name:      "split with nil option",
			existing:  []string{"192.168.0.0/23", "2001:db8::/32"},
			remove:    []string{"192.168.0.0/24", "2001:db8::/33"},
			rCase:     CaseRemovePrefix,
			opts:      []IgnoreIPOption{nil},
			want:      []string{"192.168.1.0/24", "2001:db8:8000::/33"},
			wantFound: true,
		},
		{
			name:      "split ignoring IPv4",
			existing:  []string{"192.168.0.0/23", "2001:db8::/32"},
			remove:    []string{"192.168.0.0/24", "2001:db8::/33"},
			rCase:     CaseRemovePrefix,
			opts:      []IgnoreIPOption{IgnoreIPv4},
			want:      []string{"192.168.0.0/23", "2001:db8:8000::/33"},
			wantFound: true,
		},
		{
			name:      "split ignoring IPv6",
			existing:  []string{"192.168.0.0/23", "2001:db8::/32"},
			remove:    []string{"192.168.0.0/24", "2001:db8::/33"},
			rCase:     CaseRemovePrefix,
			opts:      []IgnoreIPOption{IgnoreIPv6},
			want:      []string{"192.168.1.0/24", "2001:db8::/32"},
			wantFound: true,
		},
		{
			name:      "remove IPv6 from IPv4-only entry",
			existing:  []string{"1.1.1.0/24"},
			remove:    []string{"2001:db8::/32"},
			rCase:     CaseRemovePrefix,
			want:      []string{"1.1.1.0/24"},
			wantFound: true,
		},
		{
			name:      "remove IPv4 from IPv6-only entry",
			existing:  []string{"2001:db8::/32"},
			remove:    []string{"1.1.1.0/24"},
			rCase:     CaseRemovePrefix,
			want:      []string{"2001:db8::/32"},
			wantFound: true,
		},
		{
			name:      "remove IPv6 from IPv4-only entry ignoring IPv4",
			existing:  []string{"1.1.1.0/24"},
			remove:    []string{"1.1.1.0/24", "2001:db8::/32"},
			rCase:     CaseRemovePrefix,
			opts:      []IgnoreIPOption{IgnoreIPv4},
			want:      []string{"1.1.1.0/24"},
			wantFound: true,
		},
		{
			name:      "remove IPv4 from IPv6-only entry ignoring IPv6",
			existing:  []string{"2001:db8::/32"},
			remove:    []string{"1.1.1.0/24", "2001:db8::/32"},
			rCase:     CaseRemovePrefix,
			opts:      []IgnoreIPOption{IgnoreIPv6},
			want:      []string{"2001:db8::/32"},
			wantFound: true,
		},
		{
			name:      "remove all prefixes keeps an empty entry",
			existing:  []string{"1.1.1.0/24", "2001:db8::/32"},
			remove:    []string{"0.0.0.0/0", "::/0"},
			rCase:     CaseRemovePrefix,
			want:      nil,
			wantFound: true,
		},
		{
			name:      "remove empty entry",
			existing:  []string{"1.1.1.0/24", "2001:db8::/32"},
			remove:    nil,
			rCase:     CaseRemovePrefix,
			want:      []string{"1.1.1.0/24", "2001:db8::/32"},
			wantFound: true,
		},
		{
			name:      "remove whole entry",
			existing:  []string{"1.1.1.0/24", "2001:db8::/32"},
			remove:    nil,
			rCase:     CaseRemoveEntry,
			wantFound: false,
		},
		{
			name:      "remove whole entry with nil option",
			existing:  []string{"1.1.1.0/24", "2001:db8::/32"},
			rCase:     CaseRemoveEntry,
			opts:      []IgnoreIPOption{nil},
			wantFound: false,
		},
		{
			name:      "remove whole entry ignoring IPv4",
			existing:  []string{"1.1.1.0/24", "2001:db8::/32"},
			remove:    []string{"9.9.9.9"},
			rCase:     CaseRemoveEntry,
			opts:      []IgnoreIPOption{IgnoreIPv4},
			want:      []string{"1.1.1.0/24"},
			wantFound: true,
		},
		{
			name:      "remove whole entry ignoring IPv6",
			existing:  []string{"1.1.1.0/24", "2001:db8::/32"},
			remove:    []string{"2001:db9::1"},
			rCase:     CaseRemoveEntry,
			opts:      []IgnoreIPOption{IgnoreIPv6},
			want:      []string{"2001:db8::/32"},
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newTestContainer(t, entrySpec{name: "test", cidrs: tt.existing})
			// Build the cached sets so that Remove must invalidate them.
			_ = containerText(t, c, "test")

			toRemove := newTestEntry(t, "TEST", tt.remove...)
			if err := c.Remove(toRemove, tt.rCase, tt.opts...); err != nil {
				t.Fatalf("Remove() failed: %v", err)
			}

			entry, found := c.GetEntry("test")
			if found != tt.wantFound {
				t.Fatalf("found = %v, want %v", found, tt.wantFound)
			}
			if !found {
				if c.Len() != 0 {
					t.Fatalf("Len() = %d, want 0", c.Len())
				}
				return
			}
			assertStringSlice(t, entryText(t, entry), tt.want)
		})
	}
}

func TestContainerRemoveError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entry   string
		rCase   CaseRemove
		poison4 bool
		poison6 bool
		wantErr string
	}{
		{name: "entry not found", entry: "missing", rCase: CaseRemovePrefix, wantErr: "entry MISSING not found"},
		{name: "entry not found for whole entry removal", entry: "missing", rCase: CaseRemoveEntry, wantErr: "entry MISSING not found"},
		{name: "unknown remove case", entry: "test", rCase: CaseRemove(99), wantErr: "unknown remove case 99"},
		{name: "IPv4 builder error", entry: "test", rCase: CaseRemovePrefix, poison4: true, wantErr: "AddPrefix(invalid IP/-1)"},
		{name: "IPv6 builder error", entry: "test", rCase: CaseRemovePrefix, poison6: true, wantErr: "AddPrefix(invalid IP/-1)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newTestContainer(t, entrySpec{name: "test", cidrs: []string{"1.1.1.0/24", "2001:db8::/32"}})

			entry := newTestEntry(t, tt.entry, "1.1.1.0/25", "2001:db8::/33")
			if tt.poison4 {
				poisonIPv4Builder(entry)
			}
			if tt.poison6 {
				poisonIPv6Builder(entry)
			}

			err := c.Remove(entry, tt.rCase)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Remove() error = %v, want containing %q", err, tt.wantErr)
			}
			assertStringSlice(t, containerText(t, c, "test"), []string{"1.1.1.0/24", "2001:db8::/32"})
		})
	}
}

func TestContainerLookup(t *testing.T) {
	t.Parallel()

	c := newTestContainer(t,
		entrySpec{name: "cn", cidrs: []string{"1.0.1.0/24", "1.0.2.0/23", "2001:250::/35"}},
		entrySpec{name: "us", cidrs: []string{"8.8.8.0/24", "2001:4860::/32"}},
		entrySpec{name: "private", cidrs: []string{"10.0.0.0/8", "192.168.0.0/16", "fc00::/7"}},
		entrySpec{name: "onlyipv4", cidrs: []string{"1.0.1.0/24"}},
		entrySpec{name: "onlyipv6", cidrs: []string{"2001:4860::/32"}},
		entrySpec{name: "empty"},
	)

	tests := []struct {
		name       string
		search     string
		searchList []string
		want       []string
		wantFound  bool
		wantErr    string
	}{
		{name: "IPv4 in multiple lists", search: "1.0.1.1", want: []string{"CN", "ONLYIPV4"}, wantFound: true},
		{name: "IPv4 in single list", search: "1.0.3.255", want: []string{"CN"}, wantFound: true},
		{name: "IPv4 not found", search: "9.9.9.9", want: []string{}, wantFound: false},
		{name: "IPv4 CIDR equal", search: "1.0.1.0/24", want: []string{"CN", "ONLYIPV4"}, wantFound: true},
		{name: "IPv4 CIDR contained", search: "1.0.1.128/25", want: []string{"CN", "ONLYIPV4"}, wantFound: true},
		{name: "IPv4 CIDR with host bits", search: "1.0.2.5/23", want: []string{"CN"}, wantFound: true},
		{name: "IPv4 CIDR partially contained", search: "1.0.0.0/22", want: []string{}, wantFound: false},
		{name: "IPv4 CIDR not found", search: "9.9.9.0/24", want: []string{}, wantFound: false},
		{name: "IPv4-mapped IPv6 address", search: "::ffff:1.0.1.1", want: []string{"CN", "ONLYIPV4"}, wantFound: true},
		{name: "IPv4-mapped IPv6 address hex form", search: "::ffff:808:808", want: []string{"US"}, wantFound: true},
		{name: "IPv4-mapped IPv6 CIDR /120", search: "::ffff:1.0.1.0/120", want: []string{"CN", "ONLYIPV4"}, wantFound: true},
		{name: "IPv4-mapped IPv6 CIDR /128", search: "::ffff:10.1.2.3/128", want: []string{"PRIVATE"}, wantFound: true},
		{name: "IPv4-mapped IPv6 CIDR /104", search: "::ffff:10.0.0.0/104", want: []string{"PRIVATE"}, wantFound: true},
		{name: "IPv4-mapped IPv6 CIDR /96", search: "::ffff:0.0.0.0/96", want: []string{}, wantFound: false},
		{name: "IPv4-mapped IPv6 CIDR not found", search: "::ffff:9.9.9.0/120", want: []string{}, wantFound: false},
		{name: "IPv4-mapped IPv6 CIDR /95", search: "::ffff:1.0.1.0/95", wantErr: ErrInvalidCIDR.Error()},
		{name: "IPv4-mapped IPv6 CIDR /0", search: "::ffff:1.0.1.0/0", wantErr: ErrInvalidCIDR.Error()},
		{name: "IPv4-mapped IPv6 CIDR /129", search: "::ffff:1.0.1.0/129", wantErr: "prefix length out of range"},
		{name: "IPv6 in multiple lists", search: "2001:4860::8888", want: []string{"ONLYIPV6", "US"}, wantFound: true},
		{name: "IPv6 in single list", search: "fc00::1", want: []string{"PRIVATE"}, wantFound: true},
		{name: "IPv6 not found", search: "2001:db8::1", want: []string{}, wantFound: false},
		{name: "IPv6 with zone not found", search: "fe80::1%eth0", want: []string{}, wantFound: false},
		{name: "IPv6 CIDR equal", search: "2001:4860::/32", want: []string{"ONLYIPV6", "US"}, wantFound: true},
		{name: "IPv6 CIDR contained", search: "2001:250::/36", want: []string{"CN"}, wantFound: true},
		{name: "IPv6 CIDR partially contained", search: "2001:4860::/31", want: []string{}, wantFound: false},
		{name: "search list single", search: "1.0.1.1", searchList: []string{"cn"}, want: []string{"CN"}, wantFound: true},
		{name: "search list normalized", search: "1.0.1.1", searchList: []string{" Cn ", "", "  ", "onlyIPv4"}, want: []string{"CN", "ONLYIPV4"}, wantFound: true},
		{name: "search list without match", search: "10.1.1.1", searchList: []string{"cn", "us"}, want: []string{}, wantFound: false},
		{name: "search list with missing list", search: "10.1.1.1", searchList: []string{"missing"}, want: []string{}, wantFound: false},
		{name: "search list only empty names", search: "10.1.1.1", searchList: []string{"", " "}, want: []string{"PRIVATE"}, wantFound: true},
		{name: "IPv4 search skips IPv6-only list", search: "1.0.1.1", searchList: []string{"onlyipv6"}, want: []string{}, wantFound: false},
		{name: "IPv6 search skips IPv4-only list", search: "2001:4860::1", searchList: []string{"onlyipv4"}, want: []string{}, wantFound: false},
		{name: "invalid IP", search: "1.2.3.256", wantErr: "ParseAddr"},
		{name: "invalid CIDR", search: "1.2.3.0/33", wantErr: "prefix length out of range"},
		{name: "garbage", search: "abc", wantErr: "ParseAddr"},
		{name: "empty", search: "", wantErr: "ParseAddr"},
	}

	// Subtests share the container and Lookup lazily builds cached IP sets,
	// so they must not run in parallel.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found, err := c.Lookup(tt.search, tt.searchList...)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Lookup() error = %v, want containing %q", err, tt.wantErr)
				}
				if got != nil || found {
					t.Fatalf("Lookup() = %v, %v, want nil, false", got, found)
				}
				return
			}
			if err != nil {
				t.Fatalf("Lookup() unexpected error: %v", err)
			}
			if found != tt.wantFound {
				t.Fatalf("found = %v, want %v", found, tt.wantFound)
			}
			slices.Sort(got)
			assertStringSlice(t, got, tt.want)
		})
	}
}

func TestContainerLookupError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		search  string
		poison4 bool
		poison6 bool
	}{
		{name: "IPv4 lookup with IPv4 builder error", search: "1.1.1.1", poison4: true},
		{name: "IPv4 CIDR lookup with IPv4 builder error", search: "1.1.1.0/24", poison4: true},
		{name: "IPv6 lookup with IPv6 builder error", search: "2001:db8::1", poison6: true},
		{name: "IPv6 CIDR lookup with IPv6 builder error", search: "2001:db8::/48", poison6: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry := newTestEntry(t, "test", "1.1.1.0/24", "2001:db8::/32")
			c := NewContainer()
			if err := c.Add(entry); err != nil {
				t.Fatal(err)
			}
			if tt.poison4 {
				poisonIPv4Builder(entry)
			}
			if tt.poison6 {
				poisonIPv6Builder(entry)
			}

			got, found, err := c.Lookup(tt.search)
			assertErrorContains(t, err, "AddPrefix(invalid IP/-1)")
			if got != nil || found {
				t.Fatalf("Lookup() = %v, %v, want nil, false", got, found)
			}
		})
	}
}
