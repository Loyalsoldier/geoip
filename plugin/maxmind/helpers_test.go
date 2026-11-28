package maxmind

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/oschwald/maxminddb-golang/v2"
	"go4.org/netipx"
)

// invalidHTTPSURL is rejected by the HTTP client before any connection is
// made, as its port is out of range.
const invalidHTTPSURL = "https://127.0.0.1:65536/file"

var (
	errAdd    = errors.New("add failed")
	errRemove = errors.New("remove failed")
)

// errCheck describes the error that a test case expects.
type errCheck struct {
	desc  string
	match func(error) bool
}

// errIs expects an error that matches target as errors.Is does.
func errIs(target error) *errCheck {
	return &errCheck{
		desc:  fmt.Sprintf("an error matching %q", target),
		match: func(err error) bool { return errors.Is(err, target) },
	}
}

// errAs expects an error of type T as errors.As does.
func errAs[T error]() *errCheck {
	return &errCheck{
		desc: fmt.Sprintf("an error of type %s", reflect.TypeFor[T]()),
		match: func(err error) bool {
			_, ok := errors.AsType[T](err)
			return ok
		},
	}
}

// errContains expects an error whose message contains substr.
func errContains(substr string) *errCheck {
	return &errCheck{
		desc:  fmt.Sprintf("an error containing %q", substr),
		match: func(err error) bool { return strings.Contains(err.Error(), substr) },
	}
}

// errPath expects an *os.PathError of the given operation on the given path.
func errPath(op, path string) *errCheck {
	return &errCheck{
		desc: fmt.Sprintf("a %s error on %s", op, path),
		match: func(err error) bool {
			pathErr, ok := errors.AsType[*os.PathError](err)
			return ok && pathErr.Op == op && pathErr.Path == path
		},
	}
}

// errAny expects an error, whatever it is.
func errAny() *errCheck {
	return &errCheck{desc: "an error", match: func(error) bool { return true }}
}

// assertErr verifies err against want, where a nil want means that no error
// is expected. It reports whether an error was expected.
func assertErr(t *testing.T, err error, want *errCheck) bool {
	t.Helper()
	if want == nil {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return false
	}
	if err == nil || !want.match(err) {
		t.Fatalf("got error %v, want %s", err, want.desc)
	}
	return true
}

// assertConverterErr verifies that an error returned by a converter tells
// which type and action it comes from.
func assertConverterErr(t *testing.T, err error, typ string, action lib.Action) {
	t.Helper()
	prefix := fmt.Sprintf("❌ [type %s | action %s] ", typ, action)
	if !strings.HasPrefix(err.Error(), prefix) {
		t.Fatalf("error %q is not prefixed with %q", err, prefix)
	}
}

// assertCIDRs compares both the length and the exact content of two CIDR lists.
func assertCIDRs(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("CIDR count mismatch: got %d %v, want %d %v", len(got), got, len(want), want)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("CIDR content mismatch: got %v, want %v", got, want)
	}
}

// entryCIDRs returns the CIDRs of an entry, IPv4 ones first.
func entryCIDRs(t *testing.T, entry *lib.Entry) []string {
	t.Helper()
	cidrs, err := entry.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() of entry %s: %v", entry.GetName(), err)
	}
	return cidrs
}

// assertEntries verifies that got holds exactly the wanted entries, keyed by
// their names, and compares both the length and the exact content of the
// CIDRs of each of them.
func assertEntries(t *testing.T, got map[string]*lib.Entry, want map[string][]string) {
	t.Helper()
	gotNames := slices.Sorted(maps.Keys(got))
	wantNames := slices.Sorted(maps.Keys(want))
	if len(gotNames) != len(wantNames) || !slices.Equal(gotNames, wantNames) {
		t.Fatalf("entries mismatch: got %d %v, want %d %v", len(gotNames), gotNames, len(wantNames), wantNames)
	}
	for _, name := range wantNames {
		if gotName := got[name].GetName(); gotName != name {
			t.Fatalf("entry stored as %q is named %q", name, gotName)
		}
		assertCIDRs(t, entryCIDRs(t, got[name]), want[name])
	}
}

// assertContainer verifies that a container holds exactly the wanted entries
// and compares both the length and the exact content of the CIDRs of each of
// them.
func assertContainer(t *testing.T, got lib.Container, want map[string][]string) {
	t.Helper()
	entries := make(map[string]*lib.Entry, got.Len())
	for entry := range got.Loop() {
		entries[entry.GetName()] = entry
	}
	assertEntries(t, entries, want)
}

// newTestContainer builds a container out of lists of CIDRs keyed by entry name.
func newTestContainer(t *testing.T, lists map[string][]string) lib.Container {
	t.Helper()
	container := lib.NewContainer()
	for name, cidrs := range lists {
		if err := container.Add(newTestEntry(t, name, cidrs...)); err != nil {
			t.Fatalf("Add(%s): %v", name, err)
		}
	}
	return container
}

func newTestEntry(t *testing.T, name string, cidrs ...string) *lib.Entry {
	t.Helper()
	entry := lib.NewEntry(name)
	for _, cidr := range cidrs {
		if err := entry.AddPrefix(cidr); err != nil {
			t.Fatalf("AddPrefix(%s): %v", cidr, err)
		}
	}
	return entry
}

// mockContainer wraps a real container and allows forcing errors on Add and Remove.
type mockContainer struct {
	lib.Container
	addErr    error
	removeErr error
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

// inputCase is a test case for the Input method of an input converter.
type inputCase struct {
	name  string
	input lib.InputConverter
	// container is what the input is applied to, which defaults to an empty one.
	container lib.Container
	want      map[string][]string
	wantErr   *errCheck
}

// runInputCases runs the Input method of the converter of each test case and
// verifies either the entries of the resulting container or the error.
func runInputCases(t *testing.T, tests []inputCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := tt.container
			if container == nil {
				container = lib.NewContainer()
			}

			got, err := tt.input.Input(container)
			if assertErr(t, err, tt.wantErr) {
				assertConverterErr(t, err, tt.input.GetType(), tt.input.GetAction())
				if got != nil {
					t.Fatalf("got a container along with error %v", err)
				}
				return
			}
			assertContainer(t, got, tt.want)
		})
	}
}

// writeTestFile writes a file below dir, creating its parent directories as
// needed, and returns its path.
func writeTestFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// newTestServer starts an HTTP server that serves the given files by URL
// path, e.g. "/file.csv", and responds with 404 to any other request.
func newTestServer(t *testing.T, files map[string][]byte) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, found := files[r.URL.Path]
		if !found {
			http.NotFound(w, r)
			return
		}
		w.Write(content)
	}))
	t.Cleanup(server.Close)
	return server
}

// upperCaseScheme returns the URL of a test server with an upper case scheme.
func upperCaseScheme(server *httptest.Server) string {
	return strings.Replace(server.URL, "http://", "HTTP://", 1)
}

// captureStdout runs fn and returns what it wrote to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	orig := os.Stdout
	os.Stdout = f
	defer func() { os.Stdout = orig }()
	fn()

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// mmdbRecord is a network of an MMDB database along with the data stored for it.
type mmdbRecord struct {
	network string
	data    mmdbtype.DataType
}

// newMMDBWriter returns a writer of MMDB databases of the given IP version
// that accepts any network but the ones aliased to IPv4, just as the writer
// of the output converter.
func newMMDBWriter(t *testing.T, ipVersion int) *mmdbwriter.Tree {
	t.Helper()
	writer, err := mmdbwriter.New(mmdbwriter.Options{
		DatabaseType:            "Test",
		Description:             map[string]string{"en": "Test database"},
		IPVersion:               ipVersion,
		RecordSize:              24,
		IncludeReservedNetworks: true,
	})
	if err != nil {
		t.Fatalf("mmdbwriter.New(): %v", err)
	}
	return writer
}

// writeMMDB returns the MMDB database built by a writer.
func writeMMDB(t *testing.T, writer *mmdbwriter.Tree) []byte {
	t.Helper()
	var buf bytes.Buffer
	if _, err := writer.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo(): %v", err)
	}
	return buf.Bytes()
}

// buildMMDB builds an MMDB database that holds the given records.
func buildMMDB(t *testing.T, records ...mmdbRecord) []byte {
	t.Helper()
	return buildMMDBOfVersion(t, 6, records...)
}

// buildMMDBOfVersion builds an MMDB database of the given IP version, where
// a database of version 6 holds both IPv4 and IPv6 networks.
func buildMMDBOfVersion(t *testing.T, ipVersion int, records ...mmdbRecord) []byte {
	t.Helper()
	writer := newMMDBWriter(t, ipVersion)
	for _, record := range records {
		prefix, err := netip.ParsePrefix(record.network)
		if err != nil {
			t.Fatal(err)
		}
		if err := writer.Insert(netipx.PrefixIPNet(prefix), record.data); err != nil {
			t.Fatalf("Insert(%s): %v", record.network, err)
		}
	}
	return writeMMDB(t, writer)
}

// corruptMMDB returns a copy of an MMDB database that can still be opened but
// fails verification, as the separator between its search tree and its data
// section is no longer zeroed.
func corruptMMDB(t *testing.T, content []byte) []byte {
	t.Helper()
	db, err := maxminddb.OpenBytes(content)
	if err != nil {
		t.Fatalf("OpenBytes(): %v", err)
	}
	defer db.Close()

	corrupted := bytes.Clone(content)
	searchTreeSize := db.Metadata.NodeCount * db.Metadata.RecordSize / 4
	corrupted[searchTreeSize] = 0xff
	return corrupted
}

// readMMDB verifies an MMDB database and returns its metadata along with all
// of its records, ordered by network.
func readMMDB(t *testing.T, content []byte) (maxminddb.Metadata, []mmdbRecord) {
	t.Helper()
	db, err := maxminddb.OpenBytes(content)
	if err != nil {
		t.Fatalf("OpenBytes(): %v", err)
	}
	defer db.Close()

	if err := db.Verify(); err != nil {
		t.Fatalf("Verify(): %v", err)
	}

	var records []mmdbRecord
	unmarshaler := mmdbtype.NewUnmarshaler()
	for network := range db.Networks() {
		unmarshaler.Clear()
		if err := network.Decode(unmarshaler); err != nil {
			t.Fatalf("Decode(%s): %v", network.Prefix(), err)
		}
		records = append(records, mmdbRecord{network.Prefix().String(), unmarshaler.Result()})
	}
	return db.Metadata, records
}

// assertMMDBRecords compares both the length and the exact content of the
// networks of two lists of MMDB records, then the data of each record.
func assertMMDBRecords(t *testing.T, got, want []mmdbRecord) {
	t.Helper()
	networks := func(records []mmdbRecord) []string {
		cidrs := make([]string, 0, len(records))
		for _, record := range records {
			cidrs = append(cidrs, record.network)
		}
		return cidrs
	}
	assertCIDRs(t, networks(got), networks(want))

	for i := range want {
		if !got[i].data.Equal(want[i].data) {
			t.Fatalf("record of %s mismatch:\n got %v\nwant %v", want[i].network, got[i].data, want[i].data)
		}
	}
}
