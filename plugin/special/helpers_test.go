package special

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

// captureStdout runs fn while os.Stdout is redirected to a pipe and returns
// everything written to it. os.Stdout is always restored, even if fn calls
// t.FailNow.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}

	done := make(chan string, 1)
	go func() {
		var b strings.Builder
		_, _ = io.Copy(&b, r)
		_ = r.Close()
		done <- b.String()
	}()

	orig := os.Stdout
	os.Stdout = w
	func() {
		defer func() {
			os.Stdout = orig
			_ = w.Close()
		}()
		fn()
	}()

	return <-done
}

// setStdin replaces os.Stdin with a file holding content for the rest of the
// test. The original os.Stdin is restored on cleanup.
func setStdin(t *testing.T, content string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "stdin.txt")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("os.Open() error = %v", err)
	}

	orig := os.Stdin
	os.Stdin = f
	t.Cleanup(func() {
		os.Stdin = orig
		_ = f.Close()
	})
}

// testEntry describes an entry by name and its CIDRs.
type testEntry struct {
	name  string
	cidrs []string
}

// newTestContainer builds a container holding the given entries. An entry
// with no CIDRs is added as an empty entry.
func newTestContainer(t *testing.T, entries ...testEntry) lib.Container {
	t.Helper()

	container := lib.NewContainer()
	for _, e := range entries {
		entry := lib.NewEntry(e.name)
		for _, cidr := range e.cidrs {
			if err := entry.AddPrefix(cidr); err != nil {
				t.Fatalf("AddPrefix(%q) for entry %q error = %v", cidr, e.name, err)
			}
		}
		if err := container.Add(entry); err != nil {
			t.Fatalf("container.Add(%q) error = %v", e.name, err)
		}
	}

	return container
}

// assertStringSlice compares the length first and then every element.
func assertStringSlice(t *testing.T, label string, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s: len = %d, want %d\ngot:  %q\nwant: %q", label, len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s: [%d] = %q, want %q\ngot:  %q\nwant: %q", label, i, got[i], want[i], got, want)
		}
	}
}

// assertEntryCIDRs asserts that the named entry exists in container and that
// its merged CIDR list (IPv4 first, then IPv6) equals want. An empty want
// means the entry must exist but hold no prefix at all.
func assertEntryCIDRs(t *testing.T, container lib.Container, name string, want []string) {
	t.Helper()

	entry, found := container.GetEntry(name)
	if !found {
		t.Fatalf("entry %q not found in container", name)
	}

	got, err := entry.MarshalText()
	if len(want) == 0 {
		if err == nil {
			t.Fatalf("entry %q: MarshalText() = %q, want no prefix (error)", name, got)
		}
		return
	}
	if err != nil {
		t.Fatalf("entry %q: MarshalText() error = %v", name, err)
	}

	assertStringSlice(t, "entry "+name, got, want)
}

// assertEntryAbsent asserts that the named entry is not in container.
func assertEntryAbsent(t *testing.T, container lib.Container, name string) {
	t.Helper()

	if _, found := container.GetEntry(name); found {
		t.Fatalf("entry %q should not exist in container", name)
	}
}

// errFakeContainer is returned by failingContainer.
var errFakeContainer = errors.New("fake container failure")

// failingContainer wraps a real container but makes Add and/or Remove fail,
// so error propagation of the converters can be tested.
type failingContainer struct {
	lib.Container
	failAdd    bool
	failRemove bool
}

func (f *failingContainer) Add(entry *lib.Entry, opts ...lib.IgnoreIPOption) error {
	if f.failAdd {
		return errFakeContainer
	}
	return f.Container.Add(entry, opts...)
}

func (f *failingContainer) Remove(entry *lib.Entry, rCase lib.CaseRemove, opts ...lib.IgnoreIPOption) error {
	if f.failRemove {
		return errFakeContainer
	}
	return f.Container.Remove(entry, rCase, opts...)
}

// assertErrorString asserts err is non-nil and its message equals want.
func assertErrorString(t *testing.T, err error, want string) {
	t.Helper()

	if err == nil {
		t.Fatalf("error = nil, want %q", want)
	}
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}
