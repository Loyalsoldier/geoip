package mihomo

import (
	"bytes"
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
	"github.com/klauspost/compress/zstd"
)

func TestMain(m *testing.M) {
	// The converters log every written file; keep the test output readable.
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

// errReader always fails, to exercise read error paths.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("forced read failure") }

// errWriter always fails, to exercise write error paths.
type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("forced write failure") }

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

// closedServerURL returns the URL of a server that no longer accepts connections.
func closedServerURL(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	return url
}

// --- MRS fixtures ---

// ipRange is an inclusive address range; IPv4 addresses are encoded as
// IPv4-mapped IPv6, as mihomo does.
type ipRange struct{ from, to string }

func be64(v int64) []byte {
	return binary.BigEndian.AppendUint64(nil, uint64(v))
}

func addr16(s string) []byte {
	a := netip.MustParseAddr(s).As16()
	return a[:]
}

func join(parts ...[]byte) []byte {
	return bytes.Join(parts, nil)
}

// mrsHeader returns the uncompressed MRS header up to and including the rule
// length field.
func mrsHeader(count, extraLen int64, extra []byte, ruleLength int64) []byte {
	return join(mrsMagicBytes[:], []byte{1}, be64(count), be64(extraLen), extra, []byte{1}, be64(ruleLength))
}

func mrsRanges(ranges ...ipRange) []byte {
	var buf []byte
	for _, r := range ranges {
		buf = append(buf, addr16(r.from)...)
		buf = append(buf, addr16(r.to)...)
	}
	return buf
}

// mrsPayload returns a well-formed uncompressed MRS payload.
func mrsPayload(ranges ...ipRange) []byte {
	n := int64(len(ranges))
	return join(mrsHeader(n, 0, nil, n), mrsRanges(ranges...))
}

func zstdCompress(t *testing.T, raw []byte) []byte {
	t.Helper()
	enc, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatalf("zstd.NewWriter: %v", err)
	}
	defer enc.Close()
	return enc.EncodeAll(raw, nil)
}

func zstdDecompress(t *testing.T, data []byte) []byte {
	t.Helper()
	dec, err := zstd.NewReader(nil)
	if err != nil {
		t.Fatalf("zstd.NewReader: %v", err)
	}
	defer dec.Close()
	raw, err := dec.DecodeAll(data, nil)
	if err != nil {
		t.Fatalf("zstd DecodeAll: %v", err)
	}
	return raw
}

// writeMRS writes a well-formed MRS file holding the given ranges.
func writeMRS(t *testing.T, path string, ranges ...ipRange) string {
	t.Helper()
	return writeFile(t, path, zstdCompress(t, mrsPayload(ranges...)))
}

// decodedMRS is an independent strict decoding of an MRS file.
type decodedMRS struct {
	count  int64
	ranges [][2]netip.Addr
}

func decodeMRS(t *testing.T, data []byte) decodedMRS {
	t.Helper()
	raw := zstdDecompress(t, data)
	const headerLen = 4 + 1 + 8 + 8
	if len(raw) < headerLen+1+8 {
		t.Fatalf("MRS payload too short: %d bytes", len(raw))
	}
	if !bytes.Equal(raw[:4], mrsMagicBytes[:]) {
		t.Fatalf("magic: got %q", raw[:4])
	}
	if raw[4] != 1 {
		t.Fatalf("behavior: got %d, want 1 (IPCIDR)", raw[4])
	}
	count := int64(binary.BigEndian.Uint64(raw[5:13]))
	if extra := int64(binary.BigEndian.Uint64(raw[13:21])); extra != 0 {
		t.Fatalf("extra length: got %d, want 0", extra)
	}
	if raw[21] != 1 {
		t.Fatalf("rule version: got %d, want 1", raw[21])
	}
	ruleLength := int64(binary.BigEndian.Uint64(raw[22:30]))
	body := raw[30:]
	if int64(len(body)) != ruleLength*32 {
		t.Fatalf("got %d range bytes, want %d", len(body), ruleLength*32)
	}
	decoded := decodedMRS{count: count}
	for i := int64(0); i < ruleLength; i++ {
		from := netip.AddrFrom16([16]byte(body[i*32 : i*32+16]))
		to := netip.AddrFrom16([16]byte(body[i*32+16 : i*32+32]))
		decoded.ranges = append(decoded.ranges, [2]netip.Addr{from, to})
	}
	return decoded
}
