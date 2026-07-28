package lib

import (
	"slices"
	"testing"
)

func TestEntryMutationInvalidatesCachedIPSet(t *testing.T) {
	entry := NewEntry("test")
	if err := entry.AddPrefix("192.0.2.0/25"); err != nil {
		t.Fatal(err)
	}
	if _, err := entry.MarshalText(); err != nil {
		t.Fatal(err)
	}

	if err := entry.AddPrefix("192.0.2.128/25"); err != nil {
		t.Fatal(err)
	}
	got, err := entry.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{"192.0.2.0/24"}) {
		t.Fatalf("after add, got %v, want [192.0.2.0/24]", got)
	}

	if err := entry.RemovePrefix("192.0.2.128/25"); err != nil {
		t.Fatal(err)
	}
	got, err = entry.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{"192.0.2.0/25"}) {
		t.Fatalf("after remove, got %v, want [192.0.2.0/25]", got)
	}
}
