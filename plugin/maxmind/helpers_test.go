package maxmind

import (
	"bytes"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

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

func assertErr(t *testing.T, err error, wantErr bool) {
	t.Helper()
	if (err != nil) != wantErr {
		t.Fatalf("error = %v, wantErr %v", err, wantErr)
	}
}

// containerCIDRs returns map[listName][]cidr of a container.
func containerCIDRs(t *testing.T, c lib.Container) map[string][]string {
	t.Helper()
	result := make(map[string][]string)
	for entry := range c.Loop() {
		cidrs, err := entry.MarshalText()
		if err != nil {
			t.Fatalf("MarshalText() error = %v", err)
		}
		result[entry.GetName()] = cidrs
	}
	return result
}

func assertContainer(t *testing.T, c lib.Container, want map[string][]string) {
	t.Helper()
	got := containerCIDRs(t, c)
	if len(got) != len(want) {
		t.Fatalf("entry count mismatch: got %d %v, want %d %v", len(got), got, len(want), want)
	}
	for name, wantCIDRs := range want {
		gotCIDRs, ok := got[name]
		if !ok {
			t.Fatalf("entry %s not found in %v", name, got)
		}
		assertCIDRs(t, gotCIDRs, wantCIDRs)
	}
}

func writeFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, content, 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

type mmdbRecord struct {
	cidr string
	data mmdbtype.DataType
}

func buildMMDB(t *testing.T, records ...mmdbRecord) []byte {
	t.Helper()
	return buildMMDBWithType(t, "test", records...)
}

func buildMMDBWithType(t *testing.T, databaseType string, records ...mmdbRecord) []byte {
	t.Helper()
	w, err := mmdbwriter.New(mmdbwriter.Options{
		DatabaseType:            databaseType,
		Description:             map[string]string{"en": "test"},
		RecordSize:              24,
		IncludeReservedNetworks: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range records {
		_, n, err := net.ParseCIDR(r.cidr)
		if err != nil {
			t.Fatal(err)
		}
		if err := w.Insert(n, r.data); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	if _, err := w.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func countryRec(field, code string) mmdbtype.Map {
	return mmdbtype.Map{mmdbtype.String(field): mmdbtype.Map{"iso_code": mmdbtype.String(code)}}
}

// Keep a valid record so that silently skipping the invalid record cannot
// pass a decode-error test by producing an empty result.
func badCountryMMDB(t *testing.T) []byte {
	t.Helper()
	return buildMMDB(t,
		mmdbRecord{"1.0.0.0/24", countryRec("country", "CN")},
		mmdbRecord{"2.0.0.0/24", mmdbtype.Map{"country": mmdbtype.String("x")}},
	)
}

func badIPInfoMMDB(t *testing.T) []byte {
	t.Helper()
	return buildMMDB(t,
		mmdbRecord{"1.0.0.0/24", mmdbtype.Map{"country_code": mmdbtype.String("CN")}},
		mmdbRecord{"2.0.0.0/24", mmdbtype.Map{"country_code": mmdbtype.Map{}}},
	)
}

func unverifiedMMDB(t *testing.T) []byte {
	t.Helper()
	// The records are readable, but Verify rejects an empty database_type.
	return buildMMDBWithType(t, "", mmdbRecord{"1.0.0.0/24", countryRec("country", "CN")})
}

func newServer(t *testing.T, files map[string][]byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(content)
	}))
	t.Cleanup(srv.Close)
	return srv
}

var errFake = errors.New("fake add error")

// failingContainer wraps a real container but always fails on Add.
type failingContainer struct {
	lib.Container
}

func (f *failingContainer) Add(*lib.Entry, ...lib.IgnoreIPOption) error {
	return errFake
}

func mustAddr(s string) netip.Addr {
	return netip.MustParseAddr(s)
}
