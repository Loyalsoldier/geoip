package maxmind

import (
	"bytes"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

// mmdbRecord is a CIDR and the data inserted for it into a test MMDB file.
type mmdbRecord struct {
	cidr string
	data mmdbtype.DataType
}

// buildMMDB builds MMDB content in memory from the given records.
func buildMMDB(t *testing.T, dbType string, recordSize int, records []mmdbRecord) []byte {
	t.Helper()

	writer, err := mmdbwriter.New(mmdbwriter.Options{
		DatabaseType:            dbType,
		Description:             map[string]string{"en": "test database"},
		RecordSize:              recordSize,
		IncludeReservedNetworks: true,
	})
	if err != nil {
		t.Fatalf("mmdbwriter.New() error = %v", err)
	}

	for _, r := range records {
		_, network, err := net.ParseCIDR(r.cidr)
		if err != nil {
			t.Fatalf("net.ParseCIDR(%q) error = %v", r.cidr, err)
		}
		if err := writer.Insert(network, r.data); err != nil {
			t.Fatalf("writer.Insert(%q) error = %v", r.cidr, err)
		}
	}

	var buf bytes.Buffer
	if _, err := writer.WriteTo(&buf); err != nil {
		t.Fatalf("writer.WriteTo() error = %v", err)
	}
	return buf.Bytes()
}

// writeTempFile writes content to a file in a fresh temp dir and returns its path.
func writeTempFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	return path
}

// newFileServer serves files from the given map; unknown paths return 404.
func newFileServer(t *testing.T, files map[string][]byte) *httptest.Server {
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

var errAdd = errors.New("test container rejected entry")

type rejectingContainer struct{ lib.Container }

func (rejectingContainer) Add(*lib.Entry, ...lib.IgnoreIPOption) error { return errAdd }

func assertErrorIs(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("error = %v, want %v", got, want)
	}
}

// containerCIDRs returns the prefixes in each entry's MarshalText order.
func containerCIDRs(t *testing.T, c lib.Container) map[string][]string {
	t.Helper()
	result := make(map[string][]string)
	for entry := range c.Loop() {
		cidrs, err := entry.MarshalText()
		if err != nil {
			t.Fatalf("MarshalText(%s): %v", entry.GetName(), err)
		}
		result[entry.GetName()] = cidrs
	}
	return result
}

// assertCIDRs compares both the number of entries/CIDRs and the exact content.
func assertCIDRs(t *testing.T, got, want map[string][]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("entry count = %d, want %d; got %v", len(got), len(want), got)
	}
	for name, wantCIDRs := range want {
		gotCIDRs, ok := got[name]
		if !ok {
			t.Fatalf("entry %s not found; got %v", name, got)
		}
		if len(gotCIDRs) != len(wantCIDRs) {
			t.Fatalf("entry %s CIDR count = %d, want %d; got %v, want %v", name, len(gotCIDRs), len(wantCIDRs), gotCIDRs, wantCIDRs)
		}
		if !slices.Equal(gotCIDRs, wantCIDRs) {
			t.Fatalf("entry %s CIDRs = %v, want %v", name, gotCIDRs, wantCIDRs)
		}
	}
}

// newContainerWith creates a container with the given entries.
func newContainerWith(t *testing.T, data map[string][]string) lib.Container {
	t.Helper()
	c := lib.NewContainer()
	for name, cidrs := range data {
		entry := lib.NewEntry(name)
		for _, cidr := range cidrs {
			if err := entry.AddPrefix(cidr); err != nil {
				t.Fatalf("AddPrefix(%q) error = %v", cidr, err)
			}
		}
		if err := c.Add(entry); err != nil {
			t.Fatalf("container.Add() error = %v", err)
		}
	}
	return c
}
