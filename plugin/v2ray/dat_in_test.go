package v2ray

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

// datInFixture covers merging, duplicated and untrimmed country codes, and
// single-family lists.
func datInFixture() []*GeoIP {
	return []*GeoIP{
		geoIP(" cn ",
			rawCIDR("192.168.0.3", 24),
			rawCIDR("192.168.1.5", 24),
			rawCIDR("2001:db8::", 33),
			rawCIDR("2001:db8:8000::", 33),
		),
		geoIP("US", rawCIDR("10.0.0.0", 8), rawCIDR("2400:cb00::", 32)),
		geoIP("us", rawCIDR("11.0.0.0", 8)),
		geoIP("V4ONLY", rawCIDR("1.1.1.1", 24)),
		geoIP("V6ONLY", rawCIDR("2606:4700::", 32)),
	}
}

var datInFixtureLists = map[string][]string{
	"CN":     {"192.168.0.0/23", "2001:db8::/32"},
	"US":     {"10.0.0.0/7", "2400:cb00::/32"},
	"V4ONLY": {"1.1.1.0/24"},
	"V6ONLY": {"2606:4700::/32"},
}

func TestNewGeoIPDatIn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		action  lib.Action
		opts    []lib.InputOption
		wantURI string
		wantErr string
	}{
		{name: "missing uri", action: lib.ActionAdd, wantErr: "uri must be specified"},
		{name: "blank uri", action: lib.ActionAdd, opts: []lib.InputOption{WithURI(" \t\n ")}, wantErr: "uri must be specified"},
		{name: "nil option is ignored", action: lib.ActionRemove, opts: []lib.InputOption{nil}, wantErr: "uri must be specified"},
		{name: "uri is trimmed", action: lib.ActionAdd, opts: []lib.InputOption{nil, WithURI("  ./geoip.dat \n")}, wantURI: "./geoip.dat"},
		{name: "remote uri", action: lib.ActionRemove, opts: []lib.InputOption{WithURI("https://example.com/geoip.dat")}, wantURI: "https://example.com/geoip.dat"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewGeoIPDatIn(tt.action, tt.opts...)
			if tt.wantErr != "" {
				assertErrorContains(t, err, "[type "+TypeGeoIPDatIn+" | action "+string(tt.action)+"]", tt.wantErr)
				if conv != nil {
					t.Errorf("got converter %v, want nil", conv)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			in, ok := conv.(*geoip_dat_in)
			if !ok {
				t.Fatalf("got converter type %T, want *geoip_dat_in", conv)
			}
			if in.URI != tt.wantURI {
				t.Errorf("URI: got %q, want %q", in.URI, tt.wantURI)
			}
			if got := conv.GetType(); got != TypeGeoIPDatIn {
				t.Errorf("GetType: got %q, want %q", got, TypeGeoIPDatIn)
			}
			if got := conv.GetAction(); got != tt.action {
				t.Errorf("GetAction: got %q, want %q", got, tt.action)
			}
			if got := conv.GetDescription(); got != DescGeoIPDatIn {
				t.Errorf("GetDescription: got %q, want %q", got, DescGeoIPDatIn)
			}
		})
	}
}

func TestGeoIPDatIn_WithInputWantedList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		lists []string
		want  map[string]bool
	}{
		{name: "nil list", lists: nil, want: map[string]bool{}},
		{name: "blank items are dropped", lists: []string{"", "  ", "\t"}, want: map[string]bool{}},
		{name: "items are trimmed and upper-cased", lists: []string{" cn ", "Us", "CN", ""}, want: map[string]bool{"CN": true, "US": true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			in := &geoip_dat_in{}
			WithInputWantedList(tt.lists)(in)
			if len(in.Want) != len(tt.want) {
				t.Fatalf("got %d wanted lists %v, want %d %v", len(in.Want), in.Want, len(tt.want), tt.want)
			}
			if !maps.Equal(in.Want, tt.want) {
				t.Errorf("got wanted lists %v, want %v", in.Want, tt.want)
			}
		})
	}
}

func TestGeoIPDatIn_WithInputOnlyIPType(t *testing.T) {
	t.Parallel()

	for _, ipType := range []lib.IPType{"", lib.IPv4, lib.IPv6} {
		in := &geoip_dat_in{}
		WithInputOnlyIPType(ipType)(in)
		if in.OnlyIPType != ipType {
			t.Errorf("got OnlyIPType %q, want %q", in.OnlyIPType, ipType)
		}
	}
}

func TestNewGeoIPDatInFromBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		data           string
		wantErr        string
		wantErrAs      any
		wantURI        string
		wantWant       map[string]bool
		wantOnlyIPType lib.IPType
	}{
		{name: "invalid json", data: "{", wantErr: "[type v2rayGeoIPDat | action add]", wantErrAs: new(*json.SyntaxError)},
		{name: "wrong field type", data: `{"uri": 1}`, wantErr: "[type v2rayGeoIPDat | action add]", wantErrAs: new(*json.UnmarshalTypeError)},
		{name: "empty data", data: "", wantErr: "uri must be specified"},
		{name: "empty object", data: "{}", wantErr: "uri must be specified"},
		{
			name:           "all fields",
			data:           `{"uri": "  https://example.com/geoip.dat ", "wantedList": ["cn", " us ", ""], "onlyIPType": "ipv6"}`,
			wantURI:        "https://example.com/geoip.dat",
			wantWant:       map[string]bool{"CN": true, "US": true},
			wantOnlyIPType: lib.IPv6,
		},
		{
			name:     "uri only",
			data:     `{"uri": "geoip.dat"}`,
			wantURI:  "geoip.dat",
			wantWant: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewGeoIPDatInFromBytes(lib.ActionAdd, []byte(tt.data))
			if tt.wantErr != "" {
				assertErrorContains(t, err, tt.wantErr)
				if tt.wantErrAs != nil && !errors.As(err, tt.wantErrAs) {
					t.Errorf("error %v is not %T", err, tt.wantErrAs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			in := conv.(*geoip_dat_in)
			if in.URI != tt.wantURI {
				t.Errorf("URI: got %q, want %q", in.URI, tt.wantURI)
			}
			if len(in.Want) != len(tt.wantWant) || !maps.Equal(in.Want, tt.wantWant) {
				t.Errorf("Want: got %v, want %v", in.Want, tt.wantWant)
			}
			if in.OnlyIPType != tt.wantOnlyIPType {
				t.Errorf("OnlyIPType: got %q, want %q", in.OnlyIPType, tt.wantOnlyIPType)
			}
		})
	}
}

func TestGeoIPDatIn_Input(t *testing.T) {
	t.Parallel()

	path := writeFile(t, filepath.Join(t.TempDir(), "geoip.dat"), marshalGeoIPList(t, datInFixture()...))

	tests := []struct {
		name      string
		action    lib.Action
		opts      []lib.InputOption
		want      map[string][]string
		wantErr   string
		wantErrIs error
	}{
		{
			name:   "add all lists",
			action: lib.ActionAdd,
			want:   datInFixtureLists,
		},
		{
			name:   "wanted list",
			action: lib.ActionAdd,
			opts:   []lib.InputOption{WithInputWantedList([]string{" us", "cn ", "NOTEXIST"})},
			want: map[string][]string{
				"CN": datInFixtureLists["CN"],
				"US": datInFixtureLists["US"],
			},
		},
		{
			name:   "only ipv4",
			action: lib.ActionAdd,
			opts:   []lib.InputOption{WithInputOnlyIPType(lib.IPv4)},
			want: map[string][]string{
				"CN":     {"192.168.0.0/23"},
				"US":     {"10.0.0.0/7"},
				"V4ONLY": {"1.1.1.0/24"},
				"V6ONLY": nil,
			},
		},
		{
			name:   "only ipv6",
			action: lib.ActionAdd,
			opts:   []lib.InputOption{WithInputOnlyIPType(lib.IPv6)},
			want: map[string][]string{
				"CN":     {"2001:db8::/32"},
				"US":     {"2400:cb00::/32"},
				"V4ONLY": nil,
				"V6ONLY": {"2606:4700::/32"},
			},
		},
		{
			name:   "unknown ip type keeps both families",
			action: lib.ActionAdd,
			opts:   []lib.InputOption{WithInputOnlyIPType(lib.IPType("ipv5"))},
			want:   datInFixtureLists,
		},
		{
			name:    "wanted list matches nothing",
			action:  lib.ActionAdd,
			opts:    []lib.InputOption{WithInputWantedList([]string{"NOTEXIST"})},
			wantErr: "no entry is generated",
		},
		{
			name:      "unknown action",
			action:    lib.Action("unknown"),
			wantErr:   "[type v2rayGeoIPDat | action unknown]",
			wantErrIs: lib.ErrUnknownAction,
		},
		{
			name:      "output action is not an input action",
			action:    lib.ActionOutput,
			wantErrIs: lib.ErrUnknownAction,
		},
		{
			name:    "remove from empty container",
			action:  lib.ActionRemove,
			wantErr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := append([]lib.InputOption{WithURI(path)}, tt.opts...)
			conv, err := NewGeoIPDatIn(tt.action, opts...)
			if err != nil {
				t.Fatalf("NewGeoIPDatIn: %v", err)
			}

			got, err := conv.Input(lib.NewContainer())
			if tt.wantErr != "" || tt.wantErrIs != nil {
				if err == nil {
					t.Fatal("got nil error, want error")
				}
				if tt.wantErr != "" {
					assertErrorContains(t, err, tt.wantErr)
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("error %v is not %v", err, tt.wantErrIs)
				}
				if got != nil {
					t.Errorf("got container %v, want nil on error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Input: %v", err)
			}
			assertContainer(t, got, tt.want)
		})
	}
}

func TestGeoIPDatIn_Input_Remove(t *testing.T) {
	t.Parallel()

	removal := writeFile(t, filepath.Join(t.TempDir(), "remove.dat"), marshalGeoIPList(t,
		geoIP("CN", rawCIDR("10.0.1.0", 24), rawCIDR("2001:db8::", 33)),
		geoIP("US", rawCIDR("8.8.8.0", 24)),
	))
	initial := map[string][]string{
		"CN": {"10.0.0.0/23", "2001:db8::/32"},
		"US": {"8.8.0.0/16", "2400:cb00::/32"},
		"JP": {"1.0.0.0/8"},
	}

	tests := []struct {
		name string
		opts []lib.InputOption
		want map[string][]string
	}{
		{
			name: "remove both families",
			want: map[string][]string{
				"CN": {"10.0.0.0/24", "2001:db8:8000::/33"},
				"US": {"8.8.0.0/21", "8.8.9.0/24", "8.8.10.0/23", "8.8.12.0/22", "8.8.16.0/20", "8.8.32.0/19", "8.8.64.0/18", "8.8.128.0/17", "2400:cb00::/32"},
				"JP": {"1.0.0.0/8"},
			},
		},
		{
			name: "remove only ipv4",
			opts: []lib.InputOption{WithInputOnlyIPType(lib.IPv4), WithInputWantedList([]string{"cn"})},
			want: map[string][]string{
				"CN": {"10.0.0.0/24", "2001:db8::/32"},
				"US": initial["US"],
				"JP": {"1.0.0.0/8"},
			},
		},
		{
			name: "remove only ipv6",
			opts: []lib.InputOption{WithInputOnlyIPType(lib.IPv6)},
			want: map[string][]string{
				"CN": {"10.0.0.0/23", "2001:db8:8000::/33"},
				"US": initial["US"],
				"JP": {"1.0.0.0/8"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewGeoIPDatIn(lib.ActionRemove, append([]lib.InputOption{WithURI(removal)}, tt.opts...)...)
			if err != nil {
				t.Fatalf("NewGeoIPDatIn: %v", err)
			}
			got, err := conv.Input(newTestContainer(t, initial))
			if err != nil {
				t.Fatalf("Input: %v", err)
			}
			assertContainer(t, got, tt.want)
		})
	}
}

func TestGeoIPDatIn_Input_AddMergesIntoExistingContainer(t *testing.T) {
	t.Parallel()

	path := writeFile(t, filepath.Join(t.TempDir(), "geoip.dat"), marshalGeoIPList(t,
		geoIP("CN", rawCIDR("192.168.1.0", 24), rawCIDR("2001:db8:8000::", 33)),
	))
	conv, err := NewGeoIPDatIn(lib.ActionAdd, WithURI(path))
	if err != nil {
		t.Fatalf("NewGeoIPDatIn: %v", err)
	}

	got, err := conv.Input(newTestContainer(t, map[string][]string{
		"CN": {"192.168.0.0/24", "2001:db8::/33"},
	}))
	if err != nil {
		t.Fatalf("Input: %v", err)
	}
	assertContainer(t, got, map[string][]string{
		"CN": {"192.168.0.0/23", "2001:db8::/32"},
	})
}

func TestGeoIPDatIn_Input_LocalFileErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tests := []struct {
		name      string
		uri       string
		wantErr   string
		wantErrIs error
	}{
		{
			name:      "file does not exist",
			uri:       filepath.Join(dir, "missing.dat"),
			wantErrIs: fs.ErrNotExist,
		},
		{
			name:    "path is a directory",
			uri:     dir,
			wantErr: "[type v2rayGeoIPDat | action add]",
		},
		{
			name:    "not a protobuf file",
			uri:     writeFile(t, filepath.Join(dir, "garbage.dat"), []byte{0xff, 0xff, 0xff, 0xff}),
			wantErr: "[type v2rayGeoIPDat | action add]",
		},
		{
			name:    "empty file has no entry",
			uri:     writeFile(t, filepath.Join(dir, "empty.dat"), nil),
			wantErr: "no entry is generated",
		},
		{
			name: "invalid ip length",
			uri: writeFile(t, filepath.Join(dir, "badip.dat"), marshalGeoIPList(t,
				geoIP("cn", &CIDR{Ip: []byte{1, 2, 3, 4, 5}, Prefix: 24}),
			)),
			wantErr: "invalid IP in list CN",
		},
		{
			name: "invalid prefix bits",
			uri: writeFile(t, filepath.Join(dir, "badbits.dat"), marshalGeoIPList(t,
				geoIP("CN", rawCIDR("1.2.3.0", 33)),
			)),
			wantErrIs: lib.ErrInvalidPrefix,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewGeoIPDatIn(lib.ActionAdd, WithURI(tt.uri))
			if err != nil {
				t.Fatalf("NewGeoIPDatIn: %v", err)
			}
			got, err := conv.Input(lib.NewContainer())
			if err == nil {
				t.Fatal("got nil error, want error")
			}
			if tt.wantErr != "" {
				assertErrorContains(t, err, tt.wantErr)
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("error %v is not %v", err, tt.wantErrIs)
			}
			if got != nil {
				t.Errorf("got container %v, want nil", got)
			}
		})
	}
}

func TestGeoIPDatIn_Input_Remote(t *testing.T) {
	t.Parallel()

	body := marshalGeoIPList(t, datInFixture()...)
	okServer := newServer(t, http.StatusOK, body)
	notFoundServer := newServer(t, http.StatusNotFound, body)
	garbageServer := newServer(t, http.StatusOK, []byte{0xff, 0xff, 0xff, 0xff})
	tlsServer := httptest.NewTLSServer(http.NotFoundHandler())
	t.Cleanup(tlsServer.Close)

	tests := []struct {
		name         string
		uri          string
		want         map[string][]string
		wantErr      string
		wantURLError bool
	}{
		{name: "status 200", uri: okServer.URL + "/geoip.dat", want: datInFixtureLists},
		{name: "upper-case scheme", uri: strings.Replace(okServer.URL, "http://", "HTTP://", 1) + "/geoip.dat", want: datInFixtureLists},
		{name: "status 404", uri: notFoundServer.URL + "/geoip.dat", wantErr: "http status code 404"},
		{name: "status 200 with invalid content", uri: garbageServer.URL + "/geoip.dat", wantErr: "[type v2rayGeoIPDat | action add]"},
		{name: "connection refused", uri: closedServerURL(t) + "/geoip.dat", wantURLError: true},
		// The default client does not trust the test certificate: the error
		// proves https URIs are fetched remotely instead of opened locally.
		{name: "https is fetched remotely", uri: tlsServer.URL + "/geoip.dat", wantURLError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewGeoIPDatIn(lib.ActionAdd, WithURI(tt.uri))
			if err != nil {
				t.Fatalf("NewGeoIPDatIn: %v", err)
			}
			got, err := conv.Input(lib.NewContainer())
			switch {
			case tt.wantErr != "":
				assertErrorContains(t, err, "[type v2rayGeoIPDat | action add]", tt.wantErr)
			case tt.wantURLError:
				var urlErr *url.Error
				if !errors.As(err, &urlErr) {
					t.Fatalf("got error %v, want *url.Error", err)
				}
			default:
				if err != nil {
					t.Fatalf("Input: %v", err)
				}
				assertContainer(t, got, tt.want)
			}
		})
	}
}

func TestGeoIPDatIn_generateEntries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cidr      *CIDR
		want      []string
		wantErr   string
		wantErrIs error
	}{
		{name: "ipv4", cidr: rawCIDR("1.2.3.4", 24), want: []string{"1.2.3.0/24"}},
		{name: "ipv4 host", cidr: rawCIDR("1.2.3.4", 32), want: []string{"1.2.3.4/32"}},
		{name: "ipv6", cidr: rawCIDR("2001:db8::1", 32), want: []string{"2001:db8::/32"}},
		{name: "ipv6 host", cidr: rawCIDR("2001:db8::1", 128), want: []string{"2001:db8::1/128"}},
		{name: "ipv4-mapped with ipv4 prefix bits 24", cidr: rawCIDR("::ffff:1.2.3.4", 24), want: []string{"1.2.3.0/24"}},
		{name: "ipv4-mapped with ipv4 prefix bits 32", cidr: rawCIDR("::ffff:1.2.3.4", 32), want: []string{"1.2.3.4/32"}},
		{name: "ipv4-mapped with prefix bits 0", cidr: rawCIDR("::ffff:1.2.3.4", 0), want: []string{"0.0.0.0/0"}},
		{name: "ipv4-mapped with ipv6 prefix bits 120", cidr: rawCIDR("::ffff:1.2.3.4", 120), want: []string{"1.2.3.0/24"}},
		{name: "ipv4-mapped with ipv6 prefix bits 128", cidr: rawCIDR("::ffff:1.2.3.4", 128), want: []string{"1.2.3.4/32"}},
		{name: "ipv4-mapped with ipv6 prefix bits 96", cidr: rawCIDR("::ffff:1.2.3.4", 96), want: []string{"0.0.0.0/0"}},
		{name: "ipv4-mapped with prefix bits 64", cidr: rawCIDR("::ffff:1.2.3.4", 64), wantErrIs: lib.ErrInvalidPrefix},
		{name: "ipv4-mapped with prefix bits 129", cidr: rawCIDR("::ffff:1.2.3.4", 129), wantErrIs: lib.ErrInvalidPrefix},
		{name: "ipv4 with prefix bits 33", cidr: rawCIDR("1.2.3.4", 33), wantErrIs: lib.ErrInvalidPrefix},
		{name: "ipv6 with prefix bits 129", cidr: rawCIDR("2001:db8::", 129), wantErrIs: lib.ErrInvalidPrefix},
		{name: "5-byte ip", cidr: &CIDR{Ip: []byte{1, 2, 3, 4, 5}, Prefix: 24}, wantErr: "invalid IP in list TEST"},
		{name: "15-byte ip", cidr: &CIDR{Ip: make([]byte, 15), Prefix: 24}, wantErr: "invalid IP in list TEST"},
		{name: "empty ip", cidr: &CIDR{Prefix: 24}, wantErr: "invalid IP in list TEST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := marshalGeoIPList(t, geoIP("test", tt.cidr))
			entries := make(map[string]*lib.Entry)
			err := (&geoip_dat_in{}).generateEntries(bytes.NewReader(data), entries)
			if tt.wantErr != "" || tt.wantErrIs != nil {
				if err == nil {
					t.Fatal("got nil error, want error")
				}
				if tt.wantErr != "" {
					assertErrorContains(t, err, tt.wantErr)
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("error %v is not %v", err, tt.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("generateEntries: %v", err)
			}
			if len(entries) != 1 {
				t.Fatalf("got %d entries, want 1", len(entries))
			}
			entry, found := entries["TEST"]
			if !found {
				t.Fatalf("entry TEST not generated, got %v", entries)
			}
			got, err := entry.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText: %v", err)
			}
			assertStrings(t, "CIDRs", got, tt.want)
		})
	}
}

func TestGeoIPDatIn_generateEntries_Lists(t *testing.T) {
	t.Parallel()

	t.Run("read error", func(t *testing.T) {
		t.Parallel()

		entries := make(map[string]*lib.Entry)
		err := (&geoip_dat_in{}).generateEntries(errReader{}, entries)
		assertErrorContains(t, err, "forced read failure")
		if len(entries) != 0 {
			t.Errorf("got %d entries, want 0", len(entries))
		}
	})

	t.Run("list without cidr is still generated", func(t *testing.T) {
		t.Parallel()

		data := marshalGeoIPList(t, geoIP("empty"))
		entries := make(map[string]*lib.Entry)
		if err := (&geoip_dat_in{}).generateEntries(bytes.NewReader(data), entries); err != nil {
			t.Fatalf("generateEntries: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("got %d entries, want 1", len(entries))
		}
		if _, err := entries["EMPTY"].MarshalText(); err == nil {
			t.Error("list EMPTY should not hold any prefix")
		}
	})

	t.Run("lists are merged into existing entries and filtered by wanted list", func(t *testing.T) {
		t.Parallel()

		in := &geoip_dat_in{}
		WithInputWantedList([]string{"cn"})(in)

		existing := lib.NewEntry("CN")
		if err := existing.AddPrefix("192.168.0.0/24"); err != nil {
			t.Fatal(err)
		}
		entries := map[string]*lib.Entry{"CN": existing}

		data := marshalGeoIPList(t,
			geoIP("cn", rawCIDR("192.168.1.0", 24)),
			geoIP("us", rawCIDR("8.8.8.0", 24)),
		)
		if err := in.generateEntries(bytes.NewReader(data), entries); err != nil {
			t.Fatalf("generateEntries: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("got %d entries, want 1: %v", len(entries), entries)
		}
		if entries["CN"] != existing {
			t.Error("existing entry CN was replaced instead of merged")
		}
		got, err := entries["CN"].MarshalText()
		if err != nil {
			t.Fatal(err)
		}
		assertStrings(t, "CIDRs of CN", got, []string{"192.168.0.0/23"})
	})
}

// TestGeoIPDatIn_Registration checks the converter is reachable through the
// lib config registry set up in init.
func TestGeoIPDatIn_Registration(t *testing.T) {
	t.Parallel()

	if err := lib.RegisterInputConverter(TypeGeoIPDatIn, &geoip_dat_in{}); !errors.Is(err, lib.ErrDuplicatedConverter) {
		t.Errorf("RegisterInputConverter: got %v, want %v", err, lib.ErrDuplicatedConverter)
	}
	if err := lib.RegisterInputConfigCreator(TypeGeoIPDatIn, nil); err == nil {
		t.Error("RegisterInputConfigCreator: got nil error for an already registered type")
	}

	path := writeFile(t, filepath.Join(t.TempDir(), "geoip.dat"), marshalGeoIPList(t, datInFixture()...))
	args, err := json.Marshal(map[string]any{"uri": path, "wantedList": []string{"cn"}, "onlyIPType": "ipv6"})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		config  string
		want    map[string][]string
		wantErr bool
	}{
		{
			name:   "valid args",
			config: `{"input": [{"type": "v2rayGeoIPDat", "action": "add", "args": ` + string(args) + `}]}`,
			want:   map[string][]string{"CN": {"2001:db8::/32"}},
		},
		{
			name:    "invalid args",
			config:  `{"input": [{"type": "v2rayGeoIPDat", "action": "add", "args": {"uri": 1}}]}`,
			wantErr: true,
		},
		{
			name:    "missing args",
			config:  `{"input": [{"type": "v2rayGeoIPDat", "action": "add"}]}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			instance, err := lib.NewInstance()
			if err != nil {
				t.Fatal(err)
			}
			err = instance.InitConfigFromBytes([]byte(tt.config))
			if tt.wantErr {
				if err == nil {
					t.Fatal("got nil error, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("InitConfigFromBytes: %v", err)
			}

			container := lib.NewContainer()
			if err := instance.RunInput(container); err != nil {
				t.Fatalf("RunInput: %v", err)
			}
			assertContainer(t, container, tt.want)
		})
	}
}
