package lib

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	})
	mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(validConfig))
	})
	mux.HandleFunc("/notfound", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestGetRemoteURLContent(t *testing.T) {
	srv := newTestServer(t)
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{"ok", srv.URL + "/ok", "hello", false},
		{"non-200 status", srv.URL + "/notfound", "", true},
		{"invalid url", "://bad-url", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetRemoteURLContent(tt.url)
			assertErr(t, err, nil, tt.wantErr)
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetRemoteURLReader(t *testing.T) {
	srv := newTestServer(t)
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{"ok", srv.URL + "/ok", "hello", false},
		{"non-200 status", srv.URL + "/notfound", "", true},
		{"invalid url", "://bad-url", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc, err := GetRemoteURLReader(tt.url)
			assertErr(t, err, nil, tt.wantErr)
			if err != nil {
				if rc != nil {
					t.Fatal("expected nil reader on error")
				}
				return
			}
			defer rc.Close()
			got, err := io.ReadAll(rc)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetIgnoreIPType(t *testing.T) {
	tests := []struct {
		name  string
		input IPType
		want  IPType
		isNil bool
	}{
		{"only ipv4 ignores ipv6", IPv4, IPv6, false},
		{"only ipv6 ignores ipv4", IPv6, IPv4, false},
		{"empty", "", "", true},
		{"unknown", IPType("ipv5"), "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetIgnoreIPType(tt.input)
			if tt.isNil {
				if got != nil {
					t.Fatalf("expected nil option, got %v", got())
				}
				return
			}
			if got == nil || got() != tt.want {
				t.Fatalf("unexpected option result")
			}
		})
	}
}
