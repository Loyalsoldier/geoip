package lib

import (
	"reflect"
	"testing"
)

func TestIPv4MappedIPv6PrefixParsing(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "::ffff:192.168.1.0/120", want: "192.168.1.0/24"},
		{input: "::ffff:192.168.1.0/128", want: "192.168.1.0/32"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			entry := NewEntry("mapped")
			if err := entry.AddPrefix(tt.input); err != nil {
				t.Fatalf("AddPrefix(%q) returned error: %v", tt.input, err)
			}

			got, err := entry.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText() for %q returned error: %v", tt.input, err)
			}
			if !reflect.DeepEqual(got, []string{tt.want}) {
				t.Fatalf("MarshalText() for %q = %v, want %v", tt.input, got, []string{tt.want})
			}
		})
	}
}

func TestLookupIPv4MappedIPv6Entries(t *testing.T) {
	container := NewContainer()
	entry := NewEntry("mapped")
	if err := entry.AddPrefix("::ffff:192.168.1.0/120"); err != nil {
		t.Fatalf("AddPrefix() returned error: %v", err)
	}
	if err := container.Add(entry); err != nil {
		t.Fatalf("Add() returned error: %v", err)
	}

	matches, found, err := container.Lookup("192.168.1.42")
	if err != nil {
		t.Fatalf("Lookup() returned error: %v", err)
	}
	if !found {
		t.Fatal("Lookup() did not find the mapped IPv4 address")
	}
	if !reflect.DeepEqual(matches, []string{"MAPPED"}) {
		t.Fatalf("Lookup() = %v, want [MAPPED]", matches)
	}
}
