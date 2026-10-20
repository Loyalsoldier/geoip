package mihomo

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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

var (
	// cnRanges are adjacent ranges that merge into one prefix per family.
	cnRanges = []ipRange{
		{"192.168.0.0", "192.168.0.255"},
		{"192.168.1.0", "192.168.1.255"},
		{"2001:db8::", "2001:db8:7fff:ffff:ffff:ffff:ffff:ffff"},
		{"2001:db8:8000::", "2001:db8:ffff:ffff:ffff:ffff:ffff:ffff"},
	}
	cnCIDRs = []string{"192.168.0.0/23", "2001:db8::/32"}

	usRanges = []ipRange{
		{"10.0.0.0", "11.255.255.255"},
		{"2400:cb00::", "2400:cb00:ffff:ffff:ffff:ffff:ffff:ffff"},
	}
	usCIDRs = []string{"10.0.0.0/7", "2400:cb00::/32"}

	v4Ranges = []ipRange{{"1.1.1.0", "1.1.1.255"}}
	v4CIDRs  = []string{"1.1.1.0/24"}
)

func TestNewMRSIn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		opts         []lib.InputOption
		wantErr      string
		wantName     string
		wantURI      string
		wantInputDir string
	}{
		{name: "no option", wantErr: "missing name or uri or inputDir"},
		{name: "nil option is ignored", opts: []lib.InputOption{nil}, wantErr: "missing name or uri or inputDir"},
		{name: "blank values", opts: []lib.InputOption{WithNameAndURI(" ", "\t"), WithInputDir("  ")}, wantErr: "missing name or uri or inputDir"},
		{name: "name without uri", opts: []lib.InputOption{WithNameAndURI("cn", "")}, wantErr: "name and uri must be specified together"},
		{name: "uri without name", opts: []lib.InputOption{WithNameAndURI("", "cn.mrs")}, wantErr: "name and uri must be specified together"},
		{name: "input dir with name only", opts: []lib.InputOption{WithInputDir("dir"), WithNameAndURI("cn", " ")}, wantErr: "name and uri must be specified together"},
		{
			name:     "name and uri are trimmed",
			opts:     []lib.InputOption{nil, WithNameAndURI(" cn ", " ./cn.mrs\n")},
			wantName: "cn",
			wantURI:  "./cn.mrs",
		},
		{
			name:         "input dir is trimmed",
			opts:         []lib.InputOption{WithInputDir("  ./mrs  ")},
			wantInputDir: "./mrs",
		},
		{
			name:         "input dir with name and uri",
			opts:         []lib.InputOption{WithInputDir("dir"), WithNameAndURI("cn", "cn.mrs")},
			wantName:     "cn",
			wantURI:      "cn.mrs",
			wantInputDir: "dir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewMRSIn(lib.ActionAdd, tt.opts...)
			if tt.wantErr != "" {
				assertErrorContains(t, err, "[type mihomoMRS | action add]", tt.wantErr)
				if conv != nil {
					t.Errorf("got converter %v, want nil", conv)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			in := conv.(*mrs_in)
			if in.Name != tt.wantName || in.URI != tt.wantURI || in.InputDir != tt.wantInputDir {
				t.Errorf("got name %q uri %q inputDir %q, want %q %q %q", in.Name, in.URI, in.InputDir, tt.wantName, tt.wantURI, tt.wantInputDir)
			}
			if got := conv.GetType(); got != TypeMRSIn {
				t.Errorf("GetType: got %q, want %q", got, TypeMRSIn)
			}
			if got := conv.GetAction(); got != lib.ActionAdd {
				t.Errorf("GetAction: got %q, want %q", got, lib.ActionAdd)
			}
			if got := conv.GetDescription(); got != DescMRSIn {
				t.Errorf("GetDescription: got %q, want %q", got, DescMRSIn)
			}
		})
	}
}

func TestMRSIn_WithInputWantedList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		lists []string
		want  map[string]bool
	}{
		{name: "nil list", lists: nil, want: map[string]bool{}},
		{name: "blank items are dropped", lists: []string{"", " ", "\n"}, want: map[string]bool{}},
		{name: "items are trimmed and upper-cased", lists: []string{" cn ", "Us", "CN"}, want: map[string]bool{"CN": true, "US": true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			in := &mrs_in{}
			WithInputWantedList(tt.lists)(in)
			if len(in.Want) != len(tt.want) || !maps.Equal(in.Want, tt.want) {
				t.Errorf("got %v, want %v", in.Want, tt.want)
			}
		})
	}
}

func TestMRSIn_WithInputOnlyIPType(t *testing.T) {
	t.Parallel()

	for _, ipType := range []lib.IPType{"", lib.IPv4, lib.IPv6} {
		in := &mrs_in{}
		WithInputOnlyIPType(ipType)(in)
		if in.OnlyIPType != ipType {
			t.Errorf("got %q, want %q", in.OnlyIPType, ipType)
		}
	}
}

func TestNewMRSInFromBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		data      string
		wantErr   string
		wantErrAs any
		want      mrs_in
	}{
		{name: "invalid json", data: "{", wantErr: "[type mihomoMRS | action add]", wantErrAs: new(*json.SyntaxError)},
		{name: "wrong field type", data: `{"wantedList": "cn"}`, wantErr: "[type mihomoMRS | action add]", wantErrAs: new(*json.UnmarshalTypeError)},
		{name: "empty data", data: "", wantErr: "missing name or uri or inputDir"},
		{name: "name only", data: `{"name": "cn"}`, wantErr: "name and uri must be specified together"},
		{
			name: "name and uri",
			data: `{"name": " cn ", "uri": " https://example.com/cn.mrs ", "wantedList": ["cn", ""], "onlyIPType": "ipv4"}`,
			want: mrs_in{Name: "cn", URI: "https://example.com/cn.mrs", Want: map[string]bool{"CN": true}, OnlyIPType: lib.IPv4},
		},
		{
			name: "input dir",
			data: `{"inputDir": " ./mrs ", "onlyIPType": "ipv6"}`,
			want: mrs_in{InputDir: "./mrs", Want: map[string]bool{}, OnlyIPType: lib.IPv6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewMRSInFromBytes(lib.ActionAdd, []byte(tt.data))
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

			in := conv.(*mrs_in)
			if in.Name != tt.want.Name || in.URI != tt.want.URI || in.InputDir != tt.want.InputDir {
				t.Errorf("got name %q uri %q inputDir %q, want %q %q %q", in.Name, in.URI, in.InputDir, tt.want.Name, tt.want.URI, tt.want.InputDir)
			}
			if len(in.Want) != len(tt.want.Want) || !maps.Equal(in.Want, tt.want.Want) {
				t.Errorf("Want: got %v, want %v", in.Want, tt.want.Want)
			}
			if in.OnlyIPType != tt.want.OnlyIPType {
				t.Errorf("OnlyIPType: got %q, want %q", in.OnlyIPType, tt.want.OnlyIPType)
			}
		})
	}
}

func TestMRSIn_parseMRS(t *testing.T) {
	t.Parallel()

	z := func(parts ...[]byte) []byte { return zstdCompress(t, join(parts...)) }
	oneRange := mrsRanges(ipRange{"1.0.0.0", "1.0.0.255"})

	tests := []struct {
		name       string
		data       []byte
		want       []string
		wantErr    string
		wantErrIs  error
		wantAnyErr bool
	}{
		// Well-formed data.
		{name: "ipv4 and ipv6 ranges", data: z(mrsPayload(cnRanges...)), want: cnCIDRs},
		{name: "ipv4-mapped ipv6 ranges are unmapped", data: z(mrsPayload(ipRange{"::ffff:10.0.0.0", "::ffff:10.0.0.255"}, ipRange{"::ffff:10.0.1.0", "::ffff:10.0.1.255"})), want: []string{"10.0.0.0/23"}},
		{name: "whole ipv4 space", data: z(mrsPayload(ipRange{"0.0.0.0", "255.255.255.255"})), want: []string{"0.0.0.0/0"}},
		{name: "low ipv6 range is not unmapped", data: z(mrsPayload(ipRange{"::", "::ffff"})), want: []string{"::/112"}},
		{name: "unaligned range is split into prefixes", data: z(mrsPayload(ipRange{"1.1.1.1", "1.1.1.10"})), want: []string{"1.1.1.1/32", "1.1.1.2/31", "1.1.1.4/30", "1.1.1.8/31", "1.1.1.10/32"}},
		{name: "single address", data: z(mrsPayload(ipRange{"2001:db8::1", "2001:db8::1"})), want: []string{"2001:db8::1/128"}},
		{name: "extra data is skipped", data: z(mrsHeader(1, 5, []byte("extra"), 1), oneRange), want: []string{"1.0.0.0/24"}},
		{name: "count field is not checked", data: z(mrsHeader(99, 0, nil, 1), oneRange), want: []string{"1.0.0.0/24"}},
		{name: "trailing data is ignored", data: z(mrsPayload(ipRange{"1.0.0.0", "1.0.0.255"}), []byte("trailing")), want: []string{"1.0.0.0/24"}},
		// Ranges that are dropped without error.
		{name: "inverted range is dropped", data: z(mrsPayload(ipRange{"10.0.0.255", "10.0.0.0"})), want: nil},
		{name: "range crossing the ipv4-mapped boundary is dropped", data: z(mrsPayload(ipRange{"::ffff:255.255.255.0", "::1:0:0:0"})), want: nil},
		// Malformed data.
		{name: "not zstd data", data: []byte("definitely not zstd"), wantAnyErr: true},
		{name: "empty stream", data: z(nil), wantErrIs: io.EOF},
		{name: "truncated magic", data: z([]byte("MR")), wantErrIs: io.ErrUnexpectedEOF},
		{name: "bad magic", data: z([]byte("MRS\x02"), mrsPayload(v4Ranges...)[4:]), wantErr: "invalid MRS format"},
		{name: "missing behavior", data: z(mrsMagicBytes[:]), wantErrIs: io.EOF},
		{name: "domain behavior", data: z(mrsMagicBytes[:], []byte{0}, mrsPayload(v4Ranges...)[5:]), wantErr: "invalid MRS IPCIDR data"},
		{name: "classical behavior", data: z(mrsMagicBytes[:], []byte{2}), wantErr: "invalid MRS IPCIDR data"},
		{name: "missing count", data: z(mrsMagicBytes[:], []byte{1}), wantErrIs: io.EOF},
		{name: "truncated count", data: z(mrsMagicBytes[:], []byte{1}, []byte{0, 0, 1}), wantErrIs: io.ErrUnexpectedEOF},
		{name: "missing extra length", data: z(mrsMagicBytes[:], []byte{1}, be64(1)), wantErrIs: io.EOF},
		{name: "negative extra length", data: z(mrsMagicBytes[:], []byte{1}, be64(1), be64(-1)), wantErr: "invalid MRS extra length"},
		{name: "extra length beyond data", data: z(mrsMagicBytes[:], []byte{1}, be64(1), be64(10), []byte("abc")), wantErrIs: io.EOF},
		{name: "missing rule version", data: z(mrsMagicBytes[:], []byte{1}, be64(1), be64(0)), wantErrIs: io.EOF},
		{name: "rule version 0", data: z(mrsMagicBytes[:], []byte{1}, be64(1), be64(0), []byte{0}), wantErr: "invalid MRS rule version"},
		{name: "rule version 2", data: z(mrsMagicBytes[:], []byte{1}, be64(1), be64(0), []byte{2}, be64(1), oneRange), wantErr: "invalid MRS rule version"},
		{name: "missing rule length", data: z(mrsMagicBytes[:], []byte{1}, be64(1), be64(0), []byte{1}), wantErrIs: io.EOF},
		{name: "truncated rule length", data: z(mrsMagicBytes[:], []byte{1}, be64(1), be64(0), []byte{1}, []byte{0, 1}), wantErrIs: io.ErrUnexpectedEOF},
		{name: "zero rule length", data: z(mrsHeader(0, 0, nil, 0)), wantErr: "invalid MRS rule length"},
		{name: "negative rule length", data: z(mrsHeader(1, 0, nil, -5), oneRange), wantErr: "invalid MRS rule length"},
		{name: "missing ranges", data: z(mrsHeader(1, 0, nil, 1)), wantErrIs: io.EOF},
		{name: "truncated range start", data: z(mrsHeader(1, 0, nil, 1), oneRange[:8]), wantErrIs: io.ErrUnexpectedEOF},
		{name: "missing range end", data: z(mrsHeader(1, 0, nil, 1), oneRange[:16]), wantErrIs: io.EOF},
		{name: "truncated range end", data: z(mrsHeader(1, 0, nil, 1), oneRange[:24]), wantErrIs: io.ErrUnexpectedEOF},
		{name: "fewer ranges than rule length", data: z(mrsHeader(2, 0, nil, 2), oneRange), wantErrIs: io.EOF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entry := lib.NewEntry("test")
			err := (&mrs_in{}).parseMRS(tt.data, entry)
			if tt.wantErr != "" || tt.wantErrIs != nil || tt.wantAnyErr {
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
				t.Fatalf("parseMRS: %v", err)
			}
			assertStrings(t, "CIDRs", entryText(entry), tt.want)
		})
	}
}

func TestMRSIn_generateEntries(t *testing.T) {
	t.Parallel()

	t.Run("read error", func(t *testing.T) {
		t.Parallel()

		entries := make(map[string]*lib.Entry)
		err := (&mrs_in{}).generateEntries("cn", errReader{}, entries)
		assertErrorContains(t, err, "forced read failure")
		if len(entries) != 0 {
			t.Errorf("got %d entries, want 0", len(entries))
		}
	})

	t.Run("parse error", func(t *testing.T) {
		t.Parallel()

		entries := make(map[string]*lib.Entry)
		err := (&mrs_in{}).generateEntries("cn", strings.NewReader("garbage"), entries)
		if err == nil {
			t.Fatal("got nil error, want error")
		}
		if len(entries) != 0 {
			t.Errorf("got %d entries, want 0", len(entries))
		}
	})

	t.Run("unwanted list is skipped without reading", func(t *testing.T) {
		t.Parallel()

		in := &mrs_in{}
		WithInputWantedList([]string{"us"})(in)
		entries := make(map[string]*lib.Entry)
		if err := in.generateEntries("cn", errReader{}, entries); err != nil {
			t.Fatalf("generateEntries: %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("got %d entries, want 0", len(entries))
		}
	})

	t.Run("name is upper-cased and merged into existing entry", func(t *testing.T) {
		t.Parallel()

		existing := lib.NewEntry("CN")
		if err := existing.AddPrefix("192.168.0.0/24"); err != nil {
			t.Fatal(err)
		}
		entries := map[string]*lib.Entry{"CN": existing}
		data := zstdCompress(t, mrsPayload(ipRange{"192.168.1.0", "192.168.1.255"}))
		if err := (&mrs_in{}).generateEntries("cn", bytes.NewReader(data), entries); err != nil {
			t.Fatalf("generateEntries: %v", err)
		}
		if len(entries) != 1 || entries["CN"] != existing {
			t.Fatalf("got entries %v, want the existing CN entry only", entries)
		}
		assertStrings(t, "CIDRs of CN", entryText(existing), []string{"192.168.0.0/23"})
	})
}

func TestMRSIn_Input_NameAndURI(t *testing.T) {
	t.Parallel()

	path := writeMRS(t, filepath.Join(t.TempDir(), "file.mrs"), cnRanges...)

	tests := []struct {
		name      string
		action    lib.Action
		listName  string
		opts      []lib.InputOption
		want      map[string][]string
		wantErr   string
		wantErrIs error
	}{
		{name: "add", action: lib.ActionAdd, listName: " cn ", want: map[string][]string{"CN": cnCIDRs}},
		{name: "list name is not derived from file name", action: lib.ActionAdd, listName: "my-list", want: map[string][]string{"MY-LIST": cnCIDRs}},
		{name: "only ipv4", action: lib.ActionAdd, listName: "cn", opts: []lib.InputOption{WithInputOnlyIPType(lib.IPv4)}, want: map[string][]string{"CN": {"192.168.0.0/23"}}},
		{name: "only ipv6", action: lib.ActionAdd, listName: "cn", opts: []lib.InputOption{WithInputOnlyIPType(lib.IPv6)}, want: map[string][]string{"CN": {"2001:db8::/32"}}},
		{name: "wanted list contains name", action: lib.ActionAdd, listName: "cn", opts: []lib.InputOption{WithInputWantedList([]string{"CN"})}, want: map[string][]string{"CN": cnCIDRs}},
		{name: "wanted list misses name", action: lib.ActionAdd, listName: "cn", opts: []lib.InputOption{WithInputWantedList([]string{"us"})}, wantErr: "no entry is generated"},
		{name: "unknown action", action: lib.Action("unknown"), listName: "cn", wantErr: "[type mihomoMRS | action unknown]", wantErrIs: lib.ErrUnknownAction},
		{name: "remove from empty container", action: lib.ActionRemove, listName: "cn", wantErr: "not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewMRSIn(tt.action, append([]lib.InputOption{WithNameAndURI(tt.listName, path)}, tt.opts...)...)
			if err != nil {
				t.Fatalf("NewMRSIn: %v", err)
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
					t.Errorf("got container %v, want nil", got)
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

func TestMRSIn_Input_Remove(t *testing.T) {
	t.Parallel()

	path := writeMRS(t, filepath.Join(t.TempDir(), "cn.mrs"),
		ipRange{"10.0.1.0", "10.0.1.255"},
		ipRange{"2001:db8::", "2001:db8:7fff:ffff:ffff:ffff:ffff:ffff"},
	)
	initial := map[string][]string{
		"CN": {"10.0.0.0/23", "2001:db8::/32"},
		"US": {"8.8.8.0/24"},
	}

	tests := []struct {
		name       string
		onlyIPType lib.IPType
		want       map[string][]string
	}{
		{name: "both families", want: map[string][]string{"CN": {"10.0.0.0/24", "2001:db8:8000::/33"}, "US": {"8.8.8.0/24"}}},
		{name: "only ipv4", onlyIPType: lib.IPv4, want: map[string][]string{"CN": {"10.0.0.0/24", "2001:db8::/32"}, "US": {"8.8.8.0/24"}}},
		{name: "only ipv6", onlyIPType: lib.IPv6, want: map[string][]string{"CN": {"10.0.0.0/23", "2001:db8:8000::/33"}, "US": {"8.8.8.0/24"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewMRSIn(lib.ActionRemove, WithNameAndURI("cn", path), WithInputOnlyIPType(tt.onlyIPType))
			if err != nil {
				t.Fatalf("NewMRSIn: %v", err)
			}
			got, err := conv.Input(newTestContainer(t, initial))
			if err != nil {
				t.Fatalf("Input: %v", err)
			}
			assertContainer(t, got, tt.want)
		})
	}
}

func TestMRSIn_Input_LocalFileErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tests := []struct {
		name      string
		uri       string
		wantErr   string
		wantErrIs error
	}{
		{name: "file does not exist", uri: filepath.Join(dir, "missing.mrs"), wantErrIs: fs.ErrNotExist},
		{name: "path is a directory", uri: dir, wantErr: "[type mihomoMRS | action add]"},
		{name: "not an MRS file", uri: writeFile(t, filepath.Join(dir, "bad.mrs"), zstdCompress(t, []byte("not MRS"))), wantErr: "invalid MRS format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewMRSIn(lib.ActionAdd, WithNameAndURI("cn", tt.uri))
			if err != nil {
				t.Fatalf("NewMRSIn: %v", err)
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

func TestMRSIn_Input_Remote(t *testing.T) {
	t.Parallel()

	body := zstdCompress(t, mrsPayload(cnRanges...))
	okServer := newServer(t, http.StatusOK, body)
	notFoundServer := newServer(t, http.StatusNotFound, body)
	garbageServer := newServer(t, http.StatusOK, []byte("garbage"))
	tlsServer := httptest.NewTLSServer(http.NotFoundHandler())
	t.Cleanup(tlsServer.Close)

	tests := []struct {
		name         string
		uri          string
		want         map[string][]string
		wantErr      string
		wantURLError bool
	}{
		{name: "status 200", uri: okServer.URL + "/cn.mrs", want: map[string][]string{"CN": cnCIDRs}},
		{name: "upper-case scheme", uri: strings.Replace(okServer.URL, "http://", "HTTP://", 1) + "/cn.mrs", want: map[string][]string{"CN": cnCIDRs}},
		{name: "status 404", uri: notFoundServer.URL + "/cn.mrs", wantErr: "http status code 404"},
		{name: "status 200 with invalid content", uri: garbageServer.URL + "/cn.mrs", wantErr: "[type mihomoMRS | action add]"},
		{name: "connection aborted", uri: failingServerURL(t) + "/cn.mrs", wantURLError: true},
		// The default client does not trust the test certificate: the error
		// proves https URIs are fetched remotely instead of opened locally.
		{name: "https is fetched remotely", uri: tlsServer.URL + "/cn.mrs", wantURLError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewMRSIn(lib.ActionAdd, WithNameAndURI("cn", tt.uri))
			if err != nil {
				t.Fatalf("NewMRSIn: %v", err)
			}
			got, err := conv.Input(lib.NewContainer())
			switch {
			case tt.wantErr != "":
				assertErrorContains(t, err, tt.wantErr)
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

func TestMRSIn_Input_InputDir(t *testing.T) {
	t.Parallel()

	valid := func(ranges ...ipRange) []byte { return zstdCompress(t, mrsPayload(ranges...)) }
	malformed := zstdCompress(t, []byte("not MRS"))

	tests := []struct {
		name      string
		files     map[string][]byte
		opts      []lib.InputOption
		want      map[string][]string
		wantErr   string
		wantErrIs error
	}{
		{
			name: "walks sub directories",
			files: map[string][]byte{
				"cn.mrs":                valid(cnRanges...),
				"sub/us.mrs":            valid(usRanges...),
				"sub/deeper/v4only.mrs": valid(v4Ranges...),
			},
			want: map[string][]string{"CN": cnCIDRs, "US": usCIDRs, "V4ONLY": v4CIDRs},
		},
		{
			name: "entry names from file names",
			files: map[string][]byte{
				"geo.ip.mrs":       valid(v4Ranges...),
				".hidden":          valid(v4Ranges...),
				"noext":            valid(v4Ranges...),
				"Mixed-Case_1.MRS": valid(v4Ranges...),
			},
			want: map[string][]string{"GEO.IP": v4CIDRs, ".HIDDEN": v4CIDRs, "NOEXT": v4CIDRs, "MIXED-CASE_1": v4CIDRs},
		},
		{
			name:  "hidden file with extension",
			files: map[string][]byte{".hidden.mrs": valid(v4Ranges...)},
			want:  map[string][]string{".HIDDEN": v4CIDRs},
		},
		{
			name: "wanted list skips other files without parsing them",
			files: map[string][]byte{
				"cn.mrs":  valid(cnRanges...),
				"us.mrs":  valid(usRanges...),
				"bad.mrs": malformed,
			},
			opts: []lib.InputOption{WithInputWantedList([]string{" us ", "cn"})},
			want: map[string][]string{"CN": cnCIDRs, "US": usCIDRs},
		},
		{
			name: "only ipv6",
			files: map[string][]byte{
				"cn.mrs": valid(cnRanges...),
				"v4.mrs": valid(v4Ranges...),
			},
			opts: []lib.InputOption{WithInputOnlyIPType(lib.IPv6)},
			want: map[string][]string{"CN": {"2001:db8::/32"}, "V4": nil},
		},
		{
			name: "input dir takes precedence over name and uri",
			files: map[string][]byte{
				"cn.mrs": valid(cnRanges...),
			},
			opts: []lib.InputOption{WithNameAndURI("us", "/nonexistent/us.mrs")},
			want: map[string][]string{"CN": cnCIDRs},
		},
		{
			name:    "space in file name",
			files:   map[string][]byte{"bad name.mrs": valid(v4Ranges...)},
			wantErr: "filename bad name.mrs cannot be entry name",
		},
		{
			name:    "plus sign in file name",
			files:   map[string][]byte{"a+b.mrs": valid(v4Ranges...)},
			wantErr: "filename a+b.mrs cannot be entry name",
		},
		{
			name:    "non-ascii file name",
			files:   map[string][]byte{"中国.mrs": valid(v4Ranges...)},
			wantErr: "cannot be entry name",
		},
		{
			name:    "duplicated list in the same directory",
			files:   map[string][]byte{"cn.mrs": valid(cnRanges...), "cn.txt": valid(cnRanges...)},
			wantErr: "found duplicated list CN",
		},
		{
			name:    "duplicated list in sub directory",
			files:   map[string][]byte{"cn.mrs": valid(cnRanges...), "sub/CN.mrs": valid(cnRanges...)},
			wantErr: "found duplicated list CN",
		},
		{
			name:    "malformed file",
			files:   map[string][]byte{"cn.mrs": malformed},
			wantErr: "invalid MRS format",
		},
		{
			name:    "empty directory",
			files:   map[string][]byte{},
			wantErr: "no entry is generated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			for name, data := range tt.files {
				writeFile(t, filepath.Join(dir, filepath.FromSlash(name)), data)
			}
			conv, err := NewMRSIn(lib.ActionAdd, append([]lib.InputOption{WithInputDir(dir)}, tt.opts...)...)
			if err != nil {
				t.Fatalf("NewMRSIn: %v", err)
			}
			got, err := conv.Input(lib.NewContainer())
			if tt.wantErr != "" || tt.wantErrIs != nil {
				if err == nil {
					t.Fatal("got nil error, want error")
				}
				if tt.wantErr != "" {
					assertErrorContains(t, err, "[type mihomoMRS | action add]", tt.wantErr)
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("error %v is not %v", err, tt.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("Input: %v", err)
			}
			assertContainer(t, got, tt.want)
		})
	}

	t.Run("directory does not exist", func(t *testing.T) {
		t.Parallel()

		conv, err := NewMRSIn(lib.ActionAdd, WithInputDir(filepath.Join(t.TempDir(), "missing")))
		if err != nil {
			t.Fatalf("NewMRSIn: %v", err)
		}
		_, err = conv.Input(lib.NewContainer())
		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("got error %v, want %v", err, fs.ErrNotExist)
		}
	})
}

func TestMRSIn_Input_MissingArguments(t *testing.T) {
	t.Parallel()

	// The constructor rejects these configurations, so build them directly
	// to reach the defensive branch of Input.
	tests := []struct {
		name string
		in   *mrs_in
	}{
		{name: "nothing set", in: &mrs_in{Type: TypeMRSIn, Action: lib.ActionAdd}},
		{name: "name without uri", in: &mrs_in{Type: TypeMRSIn, Action: lib.ActionAdd, Name: "cn"}},
		{name: "uri without name", in: &mrs_in{Type: TypeMRSIn, Action: lib.ActionAdd, URI: "cn.mrs"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.in.Input(lib.NewContainer())
			assertErrorContains(t, err, "[type mihomoMRS | action add]", "config missing argument inputDir or name or uri")
			if got != nil {
				t.Errorf("got container %v, want nil", got)
			}
		})
	}
}

// TestMRSIn_Registration checks the converter is reachable through the lib
// config registry set up in init.
func TestMRSIn_Registration(t *testing.T) {
	t.Parallel()

	if err := lib.RegisterInputConverter(TypeMRSIn, &mrs_in{}); !errors.Is(err, lib.ErrDuplicatedConverter) {
		t.Errorf("RegisterInputConverter: got %v, want %v", err, lib.ErrDuplicatedConverter)
	}
	if err := lib.RegisterInputConfigCreator(TypeMRSIn, nil); err == nil {
		t.Error("RegisterInputConfigCreator: got nil error for an already registered type")
	}

	path := writeMRS(t, filepath.Join(t.TempDir(), "cn.mrs"), cnRanges...)
	args, err := json.Marshal(map[string]any{"name": "cn", "uri": path, "onlyIPType": "ipv4"})
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
			config: `{"input": [{"type": "mihomoMRS", "action": "add", "args": ` + string(args) + `}]}`,
			want:   map[string][]string{"CN": {"192.168.0.0/23"}},
		},
		{name: "invalid args", config: `{"input": [{"type": "mihomoMRS", "action": "add", "args": {"name": 1}}]}`, wantErr: true},
		{name: "missing args", config: `{"input": [{"type": "mihomoMRS", "action": "add"}]}`, wantErr: true},
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
