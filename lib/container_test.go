package lib

import (
	"fmt"
	"slices"
	"testing"
)

func TestLookupIPFamilies(t *testing.T) {
	container := NewContainer()
	entry := NewEntry("test")
	for _, cidr := range []string{"192.0.2.0/24", "2001:db8::/32"} {
		if err := entry.AddPrefix(cidr); err != nil {
			t.Fatal(err)
		}
	}
	if err := container.Add(entry); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		search  string
		found   bool
		wantErr bool
	}{
		{search: "192.0.2.1", found: true},
		{search: "192.0.2.0/24", found: true},
		{search: "192.0.2.1/24", found: true},
		{search: "::ffff:192.0.2.1", found: true},
		{search: "::ffff:192.0.2.0/120", found: true},
		{search: "::ffff:192.0.2.1/120", found: true},
		{search: "::ffff:192.0.2.1/128", found: true},
		{search: "::ffff:192.0.3.1/128"},
		{search: "::ffff:192.0.2.0/119"},
		{search: "::ffff:0.0.0.0/96"},
		{search: "::ffff:192.0.2.0/95", wantErr: true},
		{search: "::ffff:192.0.2.0/129", wantErr: true},
		{search: "2001:db8::1", found: true},
		{search: "2001:db8::/32", found: true},
		{search: "2001:db8::1/64", found: true},
		{search: "2001:db9::/32"},
		{search: "invalid/24", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.search, func(t *testing.T) {
			lists, found, err := container.Lookup(tt.search, " test ")
			if (err != nil) != tt.wantErr {
				t.Fatalf("Lookup error = %v, want error %v", err, tt.wantErr)
			}
			if found != tt.found {
				t.Errorf("found = %v, want %v", found, tt.found)
			}
			if found && !slices.Equal(lists, []string{"TEST"}) {
				t.Errorf("lists = %v, want [TEST]", lists)
			}
			if !found && len(lists) != 0 {
				t.Errorf("unexpected lists: %v", lists)
			}
		})
	}
}

func TestContainerLoopAllowsRemoval(t *testing.T) {
	for _, count := range []int{0, 1, 1000} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			container := NewContainer()
			for i := 0; i < count; i++ {
				if err := container.Add(NewEntry(fmt.Sprint(i))); err != nil {
					t.Fatal(err)
				}
			}
			visited := 0
			for entry := range container.Loop() {
				if err := container.Remove(entry, CaseRemoveEntry); err != nil {
					t.Fatal(err)
				}
				visited++
			}
			if visited != count || container.Len() != 0 {
				t.Errorf("visited %d of %d entries; %d remain", visited, count, container.Len())
			}
		})
	}
}
