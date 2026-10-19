package maxmind

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

// mmdbRecord is a single network/data pair used to build MMDB fixtures.
type mmdbRecord struct {
	cidr string
	data mmdbtype.DataType
}

// buildMMDB builds an MMDB database in memory and returns its bytes.
func buildMMDB(t *testing.T, dbType string, records []mmdbRecord) []byte {
	t.Helper()
	w, err := mmdbwriter.New(mmdbwriter.Options{
		DatabaseType:            dbType,
		Description:             map[string]string{"en": "test fixture"},
		RecordSize:              24,
		IncludeReservedNetworks: true,
	})
	if err != nil {
		t.Fatalf("mmdbwriter.New: %v", err)
	}
	for _, r := range records {
		_, network, err := net.ParseCIDR(r.cidr)
		if err != nil {
			t.Fatalf("parse cidr %q: %v", r.cidr, err)
		}
		if err := w.Insert(network, r.data); err != nil {
			t.Fatalf("insert %q: %v", r.cidr, err)
		}
	}
	var sb strings.Builder
	if _, err := w.WriteTo(&sb); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	return []byte(sb.String())
}

// writeMMDB writes an MMDB fixture into a temp dir and returns its path.
func writeMMDB(t *testing.T, dbType string, records []mmdbRecord) string {
	t.Helper()
	return writeFile(t, "fixture.mmdb", buildMMDB(t, dbType, records))
}

func writeFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

// serveFiles starts an httptest server serving the given path->content map.
// Unknown paths get 404.
func serveFiles(t *testing.T, files map[string][]byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(content)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// dumpContainer returns name -> CIDR strings for every entry in the container.
func dumpContainer(t *testing.T, c lib.Container) map[string][]string {
	t.Helper()
	out := make(map[string][]string)
	for e := range c.Loop() {
		cidrs, err := e.MarshalText()
		if err != nil {
			out[e.GetName()] = nil
			continue
		}
		out[e.GetName()] = cidrs
	}
	return out
}

// assertEntries compares container content with expected, checking both
// the number of CIDRs and their exact string values.
func assertEntries(t *testing.T, c lib.Container, want map[string][]string) {
	t.Helper()
	got := dumpContainer(t, c)
	if len(got) != len(want) {
		t.Fatalf("entry count: got %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for name, wantCIDRs := range want {
		gotCIDRs, ok := got[name]
		if !ok {
			t.Fatalf("missing entry %s; got %v", name, got)
		}
		if len(gotCIDRs) != len(wantCIDRs) {
			t.Fatalf("entry %s cidr count: got %d %v, want %d %v", name, len(gotCIDRs), gotCIDRs, len(wantCIDRs), wantCIDRs)
		}
		if !slices.Equal(gotCIDRs, wantCIDRs) {
			t.Fatalf("entry %s cidrs: got %v, want %v", name, gotCIDRs, wantCIDRs)
		}
	}
}

func assertErrContains(t *testing.T, err error, sub string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", sub)
	}
	if !strings.Contains(err.Error(), sub) {
		t.Fatalf("error %q does not contain %q", err.Error(), sub)
	}
}

// containerWith builds a container holding the given entries.
func containerWith(t *testing.T, data map[string][]string) lib.Container {
	t.Helper()
	c := lib.NewContainer()
	for name, cidrs := range data {
		e := lib.NewEntry(name)
		for _, cidr := range cidrs {
			if err := e.AddPrefix(cidr); err != nil {
				t.Fatalf("AddPrefix(%s): %v", cidr, err)
			}
		}
		if err := c.Add(e); err != nil {
			t.Fatalf("container.Add: %v", err)
		}
	}
	return c
}
