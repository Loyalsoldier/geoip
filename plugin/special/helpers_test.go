package special

import (
	"errors"
	"io"
	"os"
	"slices"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

var errMock = errors.New("mock error")

// mockContainer wraps a real container and allows injecting errors.
type mockContainer struct {
	lib.Container
	addErr    error
	removeErr error
	lookupErr error
}

func (m *mockContainer) Add(entry *lib.Entry, opts ...lib.IgnoreIPOption) error {
	if m.addErr != nil {
		return m.addErr
	}
	return m.Container.Add(entry, opts...)
}

func (m *mockContainer) Remove(entry *lib.Entry, rCase lib.CaseRemove, opts ...lib.IgnoreIPOption) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	return m.Container.Remove(entry, rCase, opts...)
}

func (m *mockContainer) Lookup(ipOrCidr string, searchList ...string) ([]string, bool, error) {
	if m.lookupErr != nil {
		return nil, false, m.lookupErr
	}
	return m.Container.Lookup(ipOrCidr, searchList...)
}

// newContainerWith builds a container with entries named by keys and CIDRs as values.
func newContainerWith(t *testing.T, data map[string][]string) lib.Container {
	t.Helper()
	c := lib.NewContainer()
	for name, cidrs := range data {
		entry := lib.NewEntry(name)
		for _, cidr := range cidrs {
			if err := entry.AddPrefix(cidr); err != nil {
				t.Fatalf("AddPrefix(%q) failed: %v", cidr, err)
			}
		}
		if err := c.Add(entry); err != nil {
			t.Fatalf("Add(%q) failed: %v", name, err)
		}
	}
	return c
}

// entryCIDRs returns the CIDRs of the named entry, or nil if not found / empty.
func entryCIDRs(t *testing.T, c lib.Container, name string) []string {
	t.Helper()
	entry, found := c.GetEntry(name)
	if !found {
		return nil
	}
	cidrs, err := entry.MarshalText()
	if err != nil {
		return nil
	}
	return cidrs
}

func assertCIDRs(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("CIDR length mismatch: got %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("CIDR content mismatch: got %v, want %v", got, want)
	}
}

func entryNames(c lib.Container) []string {
	names := make([]string, 0, c.Len())
	for e := range c.Loop() {
		names = append(names, e.GetName())
	}
	slices.Sort(names)
	return names
}

// captureStdout runs fn and returns what it wrote to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	done := make(chan []byte)
	go func() {
		b, _ := io.ReadAll(r)
		done <- b
	}()
	defer func() { os.Stdout = orig }()
	fn()
	w.Close()
	os.Stdout = orig
	out := <-done
	r.Close()
	return string(out)
}

// withStdin replaces os.Stdin with a file containing content for the duration of the test.
func withStdin(t *testing.T, content string) {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	orig := os.Stdin
	os.Stdin = f
	t.Cleanup(func() {
		os.Stdin = orig
		f.Close()
	})
}
