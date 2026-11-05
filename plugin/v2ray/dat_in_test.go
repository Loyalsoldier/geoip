package v2ray

import (
	"bytes"
	"errors"
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

// mockContainer wraps a real container and allows forcing errors on Add/Remove.
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

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read error") }

func cidr(t *testing.T, s string) *CIDR {
	t.Helper()
	p := netip.MustParsePrefix(s)
	return &CIDR{Ip: p.Addr().AsSlice(), Prefix: uint32(p.Bits())}
}

// buildDat builds V2Ray GeoIP dat bytes from country code -> CIDR list.
func buildDat(t *testing.T, lists ...*GeoIP) []byte {
	t.Helper()
	data, err := proto.Marshal(&GeoIPList{Entry: lists})
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}
	return data
}

func geoip(t *testing.T, code string, cidrs ...string) *GeoIP {
	t.Helper()
	g := &GeoIP{CountryCode: code}
	for _, c := range cidrs {
		g.Cidr = append(g.Cidr, cidr(t, c))
	}
	return g
}

func entryCIDRs(t *testing.T, c lib.Container, name string) []string {
	t.Helper()
	entry, found := c.GetEntry(name)
	if !found {
		t.Fatalf("entry %s not found", name)
	}
	cidrs, err := entry.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	return cidrs
}

func assertCIDRs(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("CIDR count mismatch: got %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("CIDR content mismatch: got %v, want %v", got, want)
	}
}

func TestNewGeoIPDatIn(t *testing.T) {
	tests := []struct {
		name    string
		opts    []lib.InputOption
		wantErr string
		check   func(t *testing.T, g *geoip_dat_in)
	}{
		{name: "no options", wantErr: "uri must be specified"},
		{name: "nil option", opts: []lib.InputOption{nil}, wantErr: "uri must be specified"},
		{name: "blank uri", opts: []lib.InputOption{WithURI("  ")}, wantErr: "uri must be specified"},
		{
			name: "all options",
			opts: []lib.InputOption{
				WithURI(" geoip.dat "),
				WithInputWantedList([]string{" cn ", "", "us"}),
				WithInputOnlyIPType(lib.IPv6),
			},
			check: func(t *testing.T, g *geoip_dat_in) {
				if g.URI != "geoip.dat" {
					t.Errorf("URI = %q", g.URI)
				}
				if len(g.Want) != 2 || !g.Want["CN"] || !g.Want["US"] {
					t.Errorf("Want = %v", g.Want)
				}
				if g.OnlyIPType != lib.IPv6 {
					t.Errorf("OnlyIPType = %v", g.OnlyIPType)
				}
				if g.GetType() != TypeGeoIPDatIn || g.GetAction() != lib.ActionAdd || g.GetDescription() != DescGeoIPDatIn {
					t.Errorf("unexpected getters: %s %s %s", g.GetType(), g.GetAction(), g.GetDescription())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewGeoIPDatIn(lib.ActionAdd, tt.opts...)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.check(t, conv.(*geoip_dat_in))
		})
	}
}

func TestNewGeoIPDatInFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		check   func(t *testing.T, g *geoip_dat_in)
	}{
		{name: "empty", data: "", wantErr: true},
		{name: "invalid json", data: "{", wantErr: true},
		{
			name: "valid",
			data: `{"uri":"geoip.dat","wantedList":["cn"],"onlyIPType":"ipv4"}`,
			check: func(t *testing.T, g *geoip_dat_in) {
				if g.URI != "geoip.dat" || len(g.Want) != 1 || !g.Want["CN"] || g.OnlyIPType != lib.IPv4 {
					t.Errorf("unexpected: %+v", g)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewGeoIPDatInFromBytes(lib.ActionAdd, []byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.check(t, conv.(*geoip_dat_in))
		})
	}
}

func TestGeoIPDatIn_generateEntries(t *testing.T) {
	tests := []struct {
		name    string
		want    map[string]bool
		data    func(t *testing.T) []byte
		reader  func(t *testing.T) *bytes.Reader
		wantErr string
		expect  map[string][]string
	}{
		{
			name: "ipv4 and ipv6",
			data: func(t *testing.T) []byte {
				return buildDat(t,
					geoip(t, " cn ", "1.0.0.0/24", "1.0.1.0/24", "2001:db8::/32"),
					geoip(t, "us", "8.8.8.0/24"),
				)
			},
			expect: map[string][]string{
				"CN": {"1.0.0.0/23", "2001:db8::/32"},
				"US": {"8.8.8.0/24"},
			},
		},
		{
			name: "duplicated country code merges",
			data: func(t *testing.T) []byte {
				return buildDat(t, geoip(t, "cn", "1.0.0.0/24"), geoip(t, "CN", "2.0.0.0/24"))
			},
			expect: map[string][]string{"CN": {"1.0.0.0/24", "2.0.0.0/24"}},
		},
		{
			name: "wanted list filters",
			want: map[string]bool{"US": true},
			data: func(t *testing.T) []byte {
				return buildDat(t, geoip(t, "cn", "1.0.0.0/24"), geoip(t, "us", "8.8.8.0/24"))
			},
			expect: map[string][]string{"US": {"8.8.8.0/24"}},
		},
		{
			name: "ipv4-mapped ipv6 with small prefix is unmapped",
			data: func(t *testing.T) []byte {
				g := &GeoIP{CountryCode: "cn", Cidr: []*CIDR{{
					Ip:     netip.MustParseAddr("::ffff:1.0.0.0").AsSlice(),
					Prefix: 24,
				}}}
				return buildDat(t, g)
			},
			expect: map[string][]string{"CN": {"1.0.0.0/24"}},
		},
		{
			name: "ipv4-mapped ipv6 with large prefix",
			data: func(t *testing.T) []byte {
				g := &GeoIP{CountryCode: "cn", Cidr: []*CIDR{{
					Ip:     netip.MustParseAddr("::ffff:1.0.0.0").AsSlice(),
					Prefix: 120,
				}}}
				return buildDat(t, g)
			},
			expect: map[string][]string{"CN": {"1.0.0.0/24"}},
		},
		{
			name:    "invalid protobuf",
			data:    func(t *testing.T) []byte { return []byte{0x0a, 0x05} },
			wantErr: "cannot parse",
		},
		{
			name: "invalid ip length",
			data: func(t *testing.T) []byte {
				return buildDat(t, &GeoIP{CountryCode: "cn", Cidr: []*CIDR{{Ip: []byte{1, 2, 3}, Prefix: 24}}})
			},
			wantErr: "invalid IP in list CN",
		},
		{
			name: "invalid prefix bits",
			data: func(t *testing.T) []byte {
				return buildDat(t, &GeoIP{CountryCode: "cn", Cidr: []*CIDR{{Ip: []byte{1, 2, 3, 4}, Prefix: 33}}})
			},
			wantErr: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geoip_dat_in{Want: tt.want}
			entries := make(map[string]*lib.Entry)
			err := g.generateEntries(bytes.NewReader(tt.data(t)), entries)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(entries) != len(tt.expect) {
				t.Fatalf("entry count mismatch: got %d, want %d", len(entries), len(tt.expect))
			}
			for name, want := range tt.expect {
				entry, ok := entries[name]
				if !ok {
					t.Fatalf("entry %s not found", name)
				}
				got, err := entry.MarshalText()
				if err != nil {
					t.Fatal(err)
				}
				assertCIDRs(t, got, want)
			}
		})
	}

	t.Run("read error", func(t *testing.T) {
		if err := (&geoip_dat_in{}).generateEntries(errReader{}, map[string]*lib.Entry{}); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestGeoIPDatIn_Input(t *testing.T) {
	datData := buildDat(t,
		geoip(t, "cn", "1.0.0.0/24", "2001:db8::/32"),
		geoip(t, "us", "8.8.8.0/24"),
	)

	tmp := t.TempDir()
	datFile := filepath.Join(tmp, "geoip.dat")
	if err := os.WriteFile(datFile, datData, 0644); err != nil {
		t.Fatal(err)
	}
	invalidFile := filepath.Join(tmp, "invalid.dat")
	if err := os.WriteFile(invalidFile, []byte{0x0a, 0x05}, 0644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/geoip.dat":
			w.Write(datData)
		case "/invalid.dat":
			w.Write([]byte{0x0a, 0x05})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	closedServer := httptest.NewServer(http.NotFoundHandler())
	closedURL := closedServer.URL
	closedServer.Close()

	newCNContainer := func(t *testing.T) lib.Container {
		c := lib.NewContainer()
		e := lib.NewEntry("CN")
		for _, p := range []string{"1.0.0.0/23", "2001:db8::/31"} {
			if err := e.AddPrefix(p); err != nil {
				t.Fatal(err)
			}
		}
		if err := c.Add(e); err != nil {
			t.Fatal(err)
		}
		return c
	}

	tests := []struct {
		name      string
		input     *geoip_dat_in
		container func(t *testing.T) lib.Container
		wantErr   string
		want      map[string][]string
	}{
		{
			name:  "local file",
			input: &geoip_dat_in{Action: lib.ActionAdd, URI: datFile},
			want: map[string][]string{
				"CN": {"1.0.0.0/24", "2001:db8::/32"},
				"US": {"8.8.8.0/24"},
			},
		},
		{
			name:  "local file only ipv4 with wanted list",
			input: &geoip_dat_in{Action: lib.ActionAdd, URI: datFile, Want: map[string]bool{"CN": true}, OnlyIPType: lib.IPv4},
			want:  map[string][]string{"CN": {"1.0.0.0/24"}},
		},
		{
			name:  "remote file",
			input: &geoip_dat_in{Action: lib.ActionAdd, URI: server.URL + "/geoip.dat", Want: map[string]bool{"US": true}},
			want:  map[string][]string{"US": {"8.8.8.0/24"}},
		},
		{
			name:  "remote file uppercase scheme",
			input: &geoip_dat_in{Action: lib.ActionAdd, URI: strings.Replace(server.URL, "http://", "HTTP://", 1) + "/geoip.dat", Want: map[string]bool{"US": true}},
			want:  map[string][]string{"US": {"8.8.8.0/24"}},
		},
		{name: "local file not exist", input: &geoip_dat_in{Action: lib.ActionAdd, URI: filepath.Join(tmp, "nope.dat")}, wantErr: "no such file"},
		{name: "local invalid data", input: &geoip_dat_in{Action: lib.ActionAdd, URI: invalidFile}, wantErr: "cannot parse"},
		{name: "remote invalid data", input: &geoip_dat_in{Action: lib.ActionAdd, URI: server.URL + "/invalid.dat"}, wantErr: "cannot parse"},
		{name: "remote 404", input: &geoip_dat_in{Action: lib.ActionAdd, URI: server.URL + "/missing"}, wantErr: "http status code 404"},
		{name: "remote connection error", input: &geoip_dat_in{Action: lib.ActionAdd, URI: closedURL}, wantErr: "connect"},
		{name: "remote https connection error", input: &geoip_dat_in{Action: lib.ActionAdd, URI: strings.Replace(closedURL, "http://", "https://", 1)}, wantErr: "connect"},
		{name: "no entry generated", input: &geoip_dat_in{Action: lib.ActionAdd, URI: datFile, Want: map[string]bool{"XX": true}}, wantErr: "no entry is generated"},
		{
			name:      "remove prefix",
			input:     &geoip_dat_in{Action: lib.ActionRemove, URI: datFile, Want: map[string]bool{"CN": true}},
			container: newCNContainer,
			want:      map[string][]string{"CN": {"1.0.1.0/24", "2001:db9::/32"}},
		},
		{
			name:      "remove prefix only ipv6",
			input:     &geoip_dat_in{Action: lib.ActionRemove, URI: datFile, Want: map[string]bool{"CN": true}, OnlyIPType: lib.IPv6},
			container: newCNContainer,
			want:      map[string][]string{"CN": {"1.0.0.0/23", "2001:db9::/32"}},
		},
		{name: "remove missing entry", input: &geoip_dat_in{Action: lib.ActionRemove, URI: datFile}, wantErr: "not found"},
		{name: "unknown action", input: &geoip_dat_in{Action: lib.Action("unknown"), URI: datFile}, wantErr: lib.ErrUnknownAction.Error()},
		{
			name:  "container add error",
			input: &geoip_dat_in{Action: lib.ActionAdd, URI: datFile},
			container: func(t *testing.T) lib.Container {
				return &mockContainer{Container: lib.NewContainer(), addErr: errors.New("add failed")}
			},
			wantErr: "add failed",
		},
		{
			name:  "container remove error",
			input: &geoip_dat_in{Action: lib.ActionRemove, URI: datFile},
			container: func(t *testing.T) lib.Container {
				return &mockContainer{Container: lib.NewContainer(), removeErr: errors.New("remove failed")}
			},
			wantErr: "remove failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.input.Type = TypeGeoIPDatIn
			var c lib.Container
			if tt.container != nil {
				c = tt.container(t)
			} else {
				c = lib.NewContainer()
			}
			got, err := tt.input.Input(c)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Len() != len(tt.want) {
				t.Fatalf("entry count mismatch: got %d, want %d", got.Len(), len(tt.want))
			}
			for name, want := range tt.want {
				assertCIDRs(t, entryCIDRs(t, got, name), want)
			}
		})
	}
}
