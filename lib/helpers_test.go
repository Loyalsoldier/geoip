package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"testing"

	"go4.org/netipx"
)

// assertStringSlice checks that got and want have both the same length and
// the exact same content in the same order.
func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("element %d mismatch: got %q, want %q (got %v, want %v)", i, got[i], want[i], got, want)
		}
	}
}

// assertPrefixSlice checks that got and want have both the same length and
// the exact same content in the same order.
func assertPrefixSlice(t *testing.T, got []netip.Prefix, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != netip.MustParsePrefix(want[i]) {
			t.Fatalf("element %d mismatch: got %s, want %s (got %v, want %v)", i, got[i], want[i], got, want)
		}
	}
}

// assertIPRangeSlice checks that got and want have both the same length and
// the exact same content in the same order. Each want element is "from-to".
func assertIPRangeSlice(t *testing.T, got []netipx.IPRange, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i].String() != want[i] {
			t.Fatalf("element %d mismatch: got %s, want %s (got %v, want %v)", i, got[i], want[i], got, want)
		}
	}
}

// assertErrorIs checks err against the wanted error. A nil want means no error
// is expected.
func assertErrorIs(t *testing.T, err, want error) {
	t.Helper()
	switch {
	case want == nil && err != nil:
		t.Fatalf("unexpected error: %v", err)
	case want != nil && !errors.Is(err, want):
		t.Fatalf("error mismatch: got %v, want %v", err, want)
	}
}

// newTestEntry creates an entry and adds all cidrs to it.
func newTestEntry(t *testing.T, name string, cidrs ...string) *Entry {
	t.Helper()
	entry := NewEntry(name)
	for _, cidr := range cidrs {
		if err := entry.AddPrefix(cidr); err != nil {
			t.Fatalf("AddPrefix(%q) failed: %v", cidr, err)
		}
	}
	return entry
}

// entryText returns the CIDR list of entry, or nil if entry has no prefix.
func entryText(t *testing.T, entry *Entry, opts ...IgnoreIPOption) []string {
	t.Helper()
	text, err := entry.MarshalText(opts...)
	if err != nil {
		if err.Error() != fmt.Sprintf("entry %s has no prefix", entry.GetName()) {
			t.Fatalf("MarshalText() error = %v", err)
		}
		return nil
	}
	return text
}

// containerText returns the CIDR list of the named entry in container.
func containerText(t *testing.T, c Container, name string, opts ...IgnoreIPOption) []string {
	t.Helper()
	entry, found := c.GetEntry(name)
	if !found {
		t.Fatalf("entry %s not found", name)
	}
	return entryText(t, entry, opts...)
}

// invalidPrefix is a prefix that makes netipx.IPSetBuilder record an error.
var invalidPrefix = netip.Prefix{}

// poisonIPv4Builder makes the next IPSet() call of entry's IPv4 builder fail.
func poisonIPv4Builder(entry *Entry) {
	if entry.ipv4Builder == nil {
		entry.ipv4Builder = new(netipx.IPSetBuilder)
	}
	entry.ipv4Builder.AddPrefix(invalidPrefix)
	entry.ipv4Set = nil
}

// poisonIPv6Builder makes the next IPSet() call of entry's IPv6 builder fail.
func poisonIPv6Builder(entry *Entry) {
	if entry.ipv6Builder == nil {
		entry.ipv6Builder = new(netipx.IPSetBuilder)
	}
	entry.ipv6Builder.AddPrefix(invalidPrefix)
	entry.ipv6Set = nil
}

// captureStdout runs fn and returns everything it writes to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}

	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	fn()

	w.Close()
	out := <-done
	r.Close()
	return out
}

// mockInputConverter is a configurable InputConverter for tests.
type mockInputConverter struct {
	typ    string
	action Action
	desc   string
	cidrs  []string
	err    error
	calls  int
}

func (m *mockInputConverter) GetType() string        { return m.typ }
func (m *mockInputConverter) GetAction() Action      { return m.action }
func (m *mockInputConverter) GetDescription() string { return m.desc }
func (m *mockInputConverter) Input(c Container) (Container, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	entry := NewEntry(m.typ)
	for _, cidr := range m.cidrs {
		if err := entry.AddPrefix(cidr); err != nil {
			return nil, err
		}
	}
	if err := c.Add(entry); err != nil {
		return nil, err
	}
	return c, nil
}

// mockOutputConverter is a configurable OutputConverter for tests.
type mockOutputConverter struct {
	typ    string
	action Action
	desc   string
	err    error
	calls  int
	got    Container
}

func (m *mockOutputConverter) GetType() string        { return m.typ }
func (m *mockOutputConverter) GetAction() Action      { return m.action }
func (m *mockOutputConverter) GetDescription() string { return m.desc }
func (m *mockOutputConverter) Output(c Container) error {
	m.calls++
	m.got = c
	return m.err
}

// mockArgs is the args of mock config creators.
type mockArgs struct {
	CIDRs []string `json:"cidrs"`
	Fail  bool     `json:"fail"`
}

func parseMockArgs(data json.RawMessage) (mockArgs, error) {
	var args mockArgs
	if len(data) > 0 {
		if err := json.Unmarshal(data, &args); err != nil {
			return args, err
		}
	}
	if args.Fail {
		return args, errors.New("mock creator failure")
	}
	return args, nil
}
