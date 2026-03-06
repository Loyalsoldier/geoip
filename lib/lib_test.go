package lib

import "testing"

func TestIgnoreIPHelpers(t *testing.T) {
	if got := IgnoreIPv4(); got != IPv4 {
		t.Fatalf("IgnoreIPv4() = %s, want %s", got, IPv4)
	}
	if got := IgnoreIPv6(); got != IPv6 {
		t.Fatalf("IgnoreIPv6() = %s, want %s", got, IPv6)
	}
}
