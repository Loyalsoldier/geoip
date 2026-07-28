package lib

import (
	"slices"
	"testing"
)

func TestContainerAddHandlesMissingAddressFamily(t *testing.T) {
	container := NewContainer()
	existing := NewEntry("test")
	if err := existing.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatal(err)
	}
	if err := container.Add(existing); err != nil {
		t.Fatal(err)
	}

	added := NewEntry("test")
	if err := added.AddPrefix("192.0.2.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := container.Add(added, IgnoreIPv6); err != nil {
		t.Fatal(err)
	}

	got, err := existing.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"192.0.2.0/24", "2001:db8::/32"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestContainerMutationInvalidatesCachedIPSet(t *testing.T) {
	container := NewContainer()
	existing := NewEntry("test")
	if err := existing.AddPrefix("192.0.2.0/25"); err != nil {
		t.Fatal(err)
	}
	if err := container.Add(existing); err != nil {
		t.Fatal(err)
	}
	if _, err := existing.MarshalText(); err != nil {
		t.Fatal(err)
	}

	added := NewEntry("test")
	if err := added.AddPrefix("192.0.2.128/25"); err != nil {
		t.Fatal(err)
	}
	if err := container.Add(added); err != nil {
		t.Fatal(err)
	}

	got, err := existing.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{"192.0.2.0/24"}) {
		t.Fatalf("got %v, want [192.0.2.0/24]", got)
	}
}
