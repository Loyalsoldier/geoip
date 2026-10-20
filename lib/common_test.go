package lib

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestServer returns a server responding "hello" on /ok and errors otherwise.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.Write([]byte("hello"))
		case "/empty":
		case "/error":
			http.Error(w, "boom", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// failingServerURL returns the URL of a server that aborts every connection.
func failingServerURL(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func remoteURLTestCases(t *testing.T) []struct {
	name    string
	url     string
	want    string
	wantErr string
} {
	t.Helper()
	server := newTestServer(t)
	return []struct {
		name    string
		url     string
		want    string
		wantErr string
	}{
		{name: "200 OK", url: server.URL + "/ok", want: "hello"},
		{name: "200 OK with empty body", url: server.URL + "/empty", want: ""},
		{name: "404 Not Found", url: server.URL + "/missing", wantErr: "failed to get remote content -> " + server.URL + "/missing: 404 Not Found"},
		{name: "500 Internal Server Error", url: server.URL + "/error", wantErr: "failed to get remote content -> " + server.URL + "/error: 500 Internal Server Error"},
		{name: "connection aborted", url: failingServerURL(t) + "/ok", wantErr: "EOF"},
		{name: "invalid URL", url: "http://[::1", wantErr: "missing ']' in host"},
		{name: "unsupported scheme", url: "ftp://example.com/file", wantErr: "unsupported protocol scheme"},
	}
}

func TestGetRemoteURLContent(t *testing.T) {
	t.Parallel()

	for _, tt := range remoteURLTestCases(t) {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := GetRemoteURLContent(tt.url)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				if got != nil {
					t.Fatalf("content = %q, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("content = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetRemoteURLReader(t *testing.T) {
	t.Parallel()

	for _, tt := range remoteURLTestCases(t) {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rc, err := GetRemoteURLReader(tt.url)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				if rc != nil {
					t.Fatal("reader should be nil on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer rc.Close()
			got, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("ReadAll error: %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("content = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetIgnoreIPType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		onlyIPType IPType
		wantNil    bool
		want       IPType
	}{
		{name: "only IPv4 ignores IPv6", onlyIPType: IPv4, want: IPv6},
		{name: "only IPv6 ignores IPv4", onlyIPType: IPv6, want: IPv4},
		{name: "empty", onlyIPType: "", wantNil: true},
		{name: "uppercase is not supported", onlyIPType: "IPV4", wantNil: true},
		{name: "unknown", onlyIPType: "ipv5", wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opt := GetIgnoreIPType(tt.onlyIPType)
			if tt.wantNil {
				if opt != nil {
					t.Fatalf("GetIgnoreIPType(%q) = %v, want nil", tt.onlyIPType, opt())
				}
				return
			}
			if opt == nil {
				t.Fatalf("GetIgnoreIPType(%q) = nil", tt.onlyIPType)
			}
			if got := opt(); got != tt.want {
				t.Fatalf("GetIgnoreIPType(%q)() = %q, want %q", tt.onlyIPType, got, tt.want)
			}
		})
	}
}

func TestWantedListExtendedUnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		data      string
		wantSlice []string
		wantMap   map[string][]string
		wantErr   string
	}{
		{name: "empty data", data: "", wantSlice: nil, wantMap: nil},
		{name: "string slice", data: `["cn", "us"]`, wantSlice: []string{"cn", "us"}, wantMap: map[string][]string{}},
		{name: "empty slice", data: `[]`, wantSlice: []string{}, wantMap: map[string][]string{}},
		{
			name:      "map of slices",
			data:      `{"cn.txt": ["cn"], "other.txt": ["us", "jp"]}`,
			wantSlice: []string{},
			wantMap:   map[string][]string{"cn.txt": {"cn"}, "other.txt": {"us", "jp"}},
		},
		{name: "empty map", data: `{}`, wantSlice: []string{}, wantMap: map[string][]string{}},
		{name: "null", data: `null`, wantSlice: nil, wantMap: map[string][]string{}},
		{name: "number", data: `123`, wantErr: "cannot unmarshal number"},
		{name: "string", data: `"cn"`, wantErr: "cannot unmarshal string"},
		{name: "map of strings", data: `{"cn.txt": "cn"}`, wantErr: "cannot unmarshal string"},
		{name: "invalid JSON", data: `[`, wantErr: "unexpected end of JSON input"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var w WantedListExtended
			err := w.UnmarshalJSON([]byte(tt.data))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if (w.TypeSlice == nil) != (tt.wantSlice == nil) {
				t.Fatalf("TypeSlice = %#v, want %#v", w.TypeSlice, tt.wantSlice)
			}
			assertStringSlice(t, w.TypeSlice, tt.wantSlice)

			if (w.TypeMap == nil) != (tt.wantMap == nil) {
				t.Fatalf("TypeMap = %#v, want %#v", w.TypeMap, tt.wantMap)
			}
			if len(w.TypeMap) != len(tt.wantMap) {
				t.Fatalf("TypeMap = %v, want %v", w.TypeMap, tt.wantMap)
			}
			for k, want := range tt.wantMap {
				got, ok := w.TypeMap[k]
				if !ok {
					t.Fatalf("TypeMap missing key %q", k)
				}
				assertStringSlice(t, got, want)
			}
		})
	}
}

func TestWantedListExtendedInStruct(t *testing.T) {
	t.Parallel()

	var cfg struct {
		Want WantedListExtended `json:"wantedList"`
	}
	if err := json.Unmarshal([]byte(`{"wantedList": {"a.txt": ["cn"]}}`), &cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Want.TypeSlice) != 0 {
		t.Fatalf("TypeSlice = %v, want empty", cfg.Want.TypeSlice)
	}
	if len(cfg.Want.TypeMap) != 1 {
		t.Fatalf("TypeMap = %v, want 1 key", cfg.Want.TypeMap)
	}
	assertStringSlice(t, cfg.Want.TypeMap["a.txt"], []string{"cn"})
}
