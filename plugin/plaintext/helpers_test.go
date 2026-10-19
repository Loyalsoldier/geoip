package plaintext

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

// errReader always fails on Read, used to exercise io.ReadAll error paths.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("forced read error") }

// newEntryWith builds an entry and adds all given CIDRs/IPs to it.
func newEntryWith(t *testing.T, name string, cidrs ...string) *lib.Entry {
	t.Helper()
	e := lib.NewEntry(name)
	for _, c := range cidrs {
		if err := e.AddPrefix(c); err != nil {
			t.Fatalf("AddPrefix(%q): %v", c, err)
		}
	}
	return e
}

// newContainerWith builds a container from a map of list name -> CIDRs.
func newContainerWith(t *testing.T, lists map[string][]string) lib.Container {
	t.Helper()
	c := lib.NewContainer()
	for name, cidrs := range lists {
		if err := c.Add(newEntryWith(t, name, cidrs...)); err != nil {
			t.Fatalf("container.Add(%s): %v", name, err)
		}
	}
	return c
}

// entryText returns the plaintext CIDRs of an entry, or nil if it has none.
func entryText(e *lib.Entry) []string {
	text, err := e.MarshalText()
	if err != nil {
		return nil
	}
	return text
}

// containerText returns all entries of a container as name -> plaintext CIDRs.
func containerText(c lib.Container) map[string][]string {
	got := make(map[string][]string)
	for e := range c.Loop() {
		got[e.GetName()] = entryText(e)
	}
	return got
}

// compareStrings compares both the count and the exact strings (in order).
func compareStrings(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: count = %d, want %d; got %q, want %q", label, len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s: [%d] = %q, want %q (got %q)", label, i, got[i], want[i], got)
		}
	}
}

// compareLists compares name -> CIDRs maps, checking list count and each list.
func compareLists(t *testing.T, got, want map[string][]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("list count = %d, want %d; got %v, want %v", len(got), len(want), got, want)
	}
	for name, w := range want {
		g, ok := got[name]
		if !ok {
			t.Errorf("list %s missing; got %v", name, got)
			continue
		}
		compareStrings(t, name, g, w)
	}
}

func checkErr(t *testing.T, err error, wantErr string) bool {
	t.Helper()
	if wantErr == "" {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return false
	}
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", wantErr)
	}
	if !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("error = %q, want it to contain %q", err.Error(), wantErr)
	}
	return true
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
