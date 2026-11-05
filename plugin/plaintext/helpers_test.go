package plaintext

import (
	"errors"
	"slices"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read error") }

// assertCIDRs compares both the length and exact content of two CIDR lists.
func assertCIDRs(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("CIDR length mismatch: got %d %v, want %d %v", len(got), got, len(want), want)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("CIDR content mismatch: got %v, want %v", got, want)
	}
}

func entryCIDRs(t *testing.T, entry *lib.Entry) []string {
	t.Helper()
	cidrs, err := entry.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error = %v", err)
	}
	return cidrs
}

func assertErr(t *testing.T, err error, wantErr bool) {
	t.Helper()
	if (err != nil) != wantErr {
		t.Fatalf("error = %v, wantErr %v", err, wantErr)
	}
}
