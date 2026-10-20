package v2ray

import (
	"bytes"
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
	"google.golang.org/protobuf/proto"
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

// entryCIDRs returns the CIDRs of a list (IPv4 first, then IPv6, each sorted
// and merged). It returns nil when the list exists but holds no prefix.
func entryCIDRs(t *testing.T, container lib.Container, name string) []string {
	t.Helper()
	entry, found := container.GetEntry(name)
	if !found {
		t.Fatalf("list %s not found in container", name)
	}
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
		assertStrings(t, "CIDRs of list "+name, entryCIDRs(t, container, name), want[name])
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

// rawCIDR builds a v2ray CIDR keeping the byte length of the parsed address:
// 4 bytes for IPv4, 16 bytes for IPv6 and IPv4-mapped IPv6.
func rawCIDR(addr string, bits uint32) *CIDR {
	return &CIDR{Ip: netip.MustParseAddr(addr).AsSlice(), Prefix: bits}
}

func geoIP(code string, cidrs ...*CIDR) *GeoIP {
	return &GeoIP{CountryCode: code, Cidr: cidrs}
}

func marshalGeoIPList(t *testing.T, entries ...*GeoIP) []byte {
	t.Helper()
	data, err := proto.Marshal(&GeoIPList{Entry: entries})
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}
	return data
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

func readGeoIPList(t *testing.T, path string) *GeoIPList {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	list := new(GeoIPList)
	if err := proto.Unmarshal(data, list); err != nil {
		t.Fatalf("proto.Unmarshal(%s): %v", path, err)
	}
	return list
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
