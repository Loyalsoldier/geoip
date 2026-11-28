package maxmind

import (
	"bytes"
	"errors"
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
	"go4.org/netipx"
)

func writeFixture(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func serveFixture(t *testing.T, data []byte) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(data)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

// Generate small source databases independently of the plugin's output converter.
func mmdbFixture(t *testing.T, databaseType string, networks map[string]mmdbtype.DataType) []byte {
	t.Helper()
	tree, err := mmdbwriter.New(mmdbwriter.Options{
		DatabaseType:            databaseType,
		Description:             map[string]string{"en": "Test country data"},
		BuildEpoch:              1,
		IncludeReservedNetworks: true,
		DisableIPv4Aliasing:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	for cidr, record := range networks {
		if err := tree.Insert(netipx.PrefixIPNet(netip.MustParsePrefix(cidr)), record); err != nil {
			t.Fatalf("insert %s: %v", cidr, err)
		}
	}
	var buf bytes.Buffer
	if _, err := tree.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func containerWith(t *testing.T, lists map[string][]string) lib.Container {
	t.Helper()
	c := lib.NewContainer()
	for name, cidrs := range lists {
		entry := lib.NewEntry(name)
		for _, cidr := range cidrs {
			if err := entry.AddPrefix(cidr); err != nil {
				t.Fatalf("add %s to %s: %v", cidr, name, err)
			}
		}
		if err := c.Add(entry); err != nil {
			t.Fatal(err)
		}
	}
	return c
}

func assertCIDRs(t *testing.T, name string, got, want []string) {
	t.Helper()
	got, want = slices.Clone(got), slices.Clone(want)
	slices.Sort(got)
	slices.Sort(want)
	if len(got) != len(want) || !slices.Equal(got, want) {
		t.Errorf("%s: got %d CIDRs %v, want %d %v", name, len(got), got, len(want), want)
	}
}

func assertContainer(t *testing.T, c lib.Container, want map[string][]string) {
	t.Helper()
	if c.Len() != len(want) {
		t.Errorf("got %d entries, want %d", c.Len(), len(want))
	}
	for name, cidrs := range want {
		entry, ok := c.GetEntry(name)
		if !ok {
			t.Errorf("missing entry %s", name)
			continue
		}
		got, err := entry.MarshalText()
		if err != nil {
			t.Fatalf("marshal %s: %v", name, err)
		}
		assertCIDRs(t, name, got, cidrs)
	}
}

type failingContainer struct{ lib.Container }

var errContainer = errors.New("container mutation failed")

func (failingContainer) Add(*lib.Entry, ...lib.IgnoreIPOption) error { return errContainer }
func (failingContainer) Remove(*lib.Entry, lib.CaseRemove, ...lib.IgnoreIPOption) error {
	return errContainer
}

func testInputActionErrors(t *testing.T, newInput func(lib.Action) (lib.InputConverter, error)) {
	t.Helper()
	for _, tt := range []struct {
		action lib.Action
		want   error
	}{
		{lib.ActionAdd, errContainer},
		{lib.ActionRemove, errContainer},
		{"unknown", lib.ErrUnknownAction},
	} {
		t.Run(string(tt.action), func(t *testing.T) {
			in, err := newInput(tt.action)
			if err != nil {
				t.Fatal(err)
			}
			got, err := in.Input(failingContainer{lib.NewContainer()})
			if got != nil || !errors.Is(err, tt.want) {
				t.Errorf("Input = (%v, %v), want nil and %v", got, err, tt.want)
			}
		})
	}
}
