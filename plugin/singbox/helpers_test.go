package singbox

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/sagernet/sing-box/common/srs"
	"github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

func TestMain(m *testing.M) {
	// The converters log every written file; keep the test output readable.
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

// errReader always fails, to exercise read error paths.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("forced read failure") }

// captureLog redirects the standard logger into a buffer for the rest of the
// test. It must only be used by tests that do not call t.Parallel.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := new(bytes.Buffer)
	prev := log.Writer()
	log.SetOutput(buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	return buf
}

// newTestContainer builds a container holding one entry per list name.
func newTestContainer(t *testing.T, lists map[string][]string) lib.Container {
	t.Helper()
	container := lib.NewContainer()
	for name, cidrs := range lists {
		entry := lib.NewEntry(name)
		for _, cidr := range cidrs {
			if err := entry.AddPrefix(cidr); err != nil {
				t.Fatalf("AddPrefix(%q) to list %s: %v", cidr, name, err)
			}
		}
		if err := container.Add(entry); err != nil {
			t.Fatalf("container.Add(%s): %v", name, err)
		}
	}
	return container
}

// entryText returns the CIDRs of an entry (IPv4 first, then IPv6, each sorted
// and merged), or nil when the entry holds no prefix.
func entryText(entry *lib.Entry) []string {
	cidrs, err := entry.MarshalText()
	if err != nil {
		return nil
	}
	return cidrs
}

func containerNames(container lib.Container) []string {
	names := make([]string, 0, container.Len())
	for entry := range container.Loop() {
		names = append(names, entry.GetName())
	}
	slices.Sort(names)
	return names
}

func assertStrings(t *testing.T, what string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %d items %q, want %d items %q", what, len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s[%d]: got %q, want %q", what, i, got[i], want[i])
		}
	}
}

// assertContainer checks that the container holds exactly the wanted lists,
// each with exactly the wanted CIDRs. An empty CIDR slice means the list
// exists without any prefix.
func assertContainer(t *testing.T, container lib.Container, want map[string][]string) {
	t.Helper()
	wantNames := make([]string, 0, len(want))
	for name := range want {
		wantNames = append(wantNames, name)
	}
	slices.Sort(wantNames)
	assertStrings(t, "list names", containerNames(container), wantNames)
	for _, name := range wantNames {
		entry, found := container.GetEntry(name)
		if !found {
			t.Fatalf("list %s not found", name)
		}
		assertStrings(t, "CIDRs of list "+name, entryText(entry), want[name])
	}
}

func assertErrorContains(t *testing.T, err error, substrs ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("got nil error, want error containing %q", substrs)
	}
	for _, s := range substrs {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("error %q does not contain %q", err, s)
		}
	}
}

func pick(lists map[string][]string, names ...string) map[string][]string {
	out := make(map[string][]string, len(names))
	for _, name := range names {
		out[name] = lists[name]
	}
	return out
}

func writeFile(t *testing.T, path string, data []byte) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func dirFileNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	slices.Sort(names)
	return names
}

// newServer starts a test HTTP server answering every request with the given
// status code and body.
func newServer(t *testing.T, status int, body []byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// failingServerURL returns the URL of a server that aborts every connection.
func failingServerURL(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// --- SRS fixtures ---

func ipcidrRule(cidrs ...string) option.HeadlessRule {
	return option.HeadlessRule{
		Type:           constant.RuleTypeDefault,
		DefaultOptions: option.DefaultHeadlessRule{IPCIDR: cidrs},
	}
}

func logicalRule(rules ...option.HeadlessRule) option.HeadlessRule {
	return option.HeadlessRule{
		Type:           constant.RuleTypeLogical,
		LogicalOptions: option.LogicalHeadlessRule{Mode: constant.LogicalTypeOr, Rules: rules},
	}
}

// srsBytesVersion encodes the rules with the sing-box SRS writer.
func srsBytesVersion(t *testing.T, version uint8, rules ...option.HeadlessRule) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := srs.Write(&buf, option.PlainRuleSet{Rules: rules}, version); err != nil {
		t.Fatalf("srs.Write: %v", err)
	}
	return buf.Bytes()
}

func srsBytes(t *testing.T, rules ...option.HeadlessRule) []byte {
	t.Helper()
	return srsBytesVersion(t, constant.RuleSetVersion1, rules...)
}

// writeSRS writes a rule-set holding a single ip_cidr rule.
func writeSRS(t *testing.T, path string, cidrs ...string) string {
	t.Helper()
	return writeFile(t, path, srsBytes(t, ipcidrRule(cidrs...)))
}

// rawSRS builds an SRS file around a handcrafted (uncompressed) payload.
func rawSRS(t *testing.T, version byte, payload []byte) []byte {
	t.Helper()
	return append([]byte{'S', 'R', 'S', version}, zlibCompress(t, payload)...)
}

func zlibCompress(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		t.Fatalf("zlib write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zlib close: %v", err)
	}
	return buf.Bytes()
}

func uvarint(v uint64) []byte {
	return binary.AppendUvarint(nil, v)
}

func join(parts ...[]byte) []byte {
	return bytes.Join(parts, nil)
}

// rawAddr encodes an address the way an SRS IP set stores it: a uvarint
// length followed by the raw bytes. IPv4-mapped IPv6 addresses keep their
// 16-byte form.
func rawAddr(s string) []byte {
	addr := netip.MustParseAddr(s)
	return append(uvarint(uint64(addr.BitLen()/8)), addr.AsSlice()...)
}

// rawIPSetItem encodes an ip_cidr item holding the given [from, to] pairs.
func rawIPSetItem(ranges ...[2]string) []byte {
	out := []byte{6, 1} // ip_cidr item, IP set version 1
	out = binary.BigEndian.AppendUint64(out, uint64(len(ranges)))
	for _, r := range ranges {
		out = append(out, rawAddr(r[0])...)
		out = append(out, rawAddr(r[1])...)
	}
	return out
}

// rawDefaultRule encodes a default rule made of the given items, followed by
// the final item and a false invert flag.
func rawDefaultRule(items ...[]byte) []byte {
	return join([]byte{0}, join(items...), []byte{0xFF, 0})
}

// rawRuleSet encodes a rule count followed by the given rules.
func rawRuleSet(rules ...[]byte) []byte {
	return join(uvarint(uint64(len(rules))), join(rules...))
}

// readSRS decodes SRS data with the sing-box reader, recovering ip_cidr
// strings.
func readSRS(t *testing.T, data []byte) option.PlainRuleSetCompat {
	t.Helper()
	ruleSet, err := srs.Read(bytes.NewReader(data), true)
	if err != nil {
		t.Fatalf("srs.Read: %v", err)
	}
	return ruleSet
}
