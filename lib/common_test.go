package lib

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetRemoteURLContent(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("hello"))
		}))
		defer s.Close()

		data, err := GetRemoteURLContent(s.URL)
		if err != nil {
			t.Fatalf("GetRemoteURLContent() error = %v", err)
		}
		if string(data) != "hello" {
			t.Fatalf("GetRemoteURLContent() = %s, want %s", data, "hello")
		}
	})

	t.Run("status error", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer s.Close()

		if _, err := GetRemoteURLContent(s.URL); err == nil {
			t.Fatalf("expected error for non-200 response")
		}
	})

	t.Run("request error", func(t *testing.T) {
		if _, err := GetRemoteURLContent("http://[%"); err == nil {
			t.Fatalf("expected error for invalid URL")
		}
	})
}

func TestGetRemoteURLReader(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("world"))
		}))
		defer s.Close()

		rc, err := GetRemoteURLReader(s.URL)
		if err != nil {
			t.Fatalf("GetRemoteURLReader() error = %v", err)
		}
		defer rc.Close()

		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("unexpected read error: %v", err)
		}
		if string(data) != "world" {
			t.Fatalf("GetRemoteURLReader() = %s, want %s", data, "world")
		}
	})

	t.Run("status error", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer s.Close()

		if rc, err := GetRemoteURLReader(s.URL); err == nil {
			rc.Close()
			t.Fatalf("expected error for non-200 response")
		}
	})

	t.Run("request error", func(t *testing.T) {
		if _, err := GetRemoteURLReader("http://[%"); err == nil {
			t.Fatalf("expected error for invalid URL")
		}
	})
}

func TestGetIgnoreIPType(t *testing.T) {
	if fn := GetIgnoreIPType(IPv4); fn == nil || fn() != IPv6 {
		t.Fatalf("GetIgnoreIPType(IPv4) = %v", fn)
	}
	if fn := GetIgnoreIPType(IPv6); fn == nil || fn() != IPv4 {
		t.Fatalf("GetIgnoreIPType(IPv6) = %v", fn)
	}
	if fn := GetIgnoreIPType(IPType("other")); fn != nil {
		t.Fatalf("GetIgnoreIPType(other) = %v, want nil", fn)
	}
}

func TestWantedListExtendedUnmarshalJSON(t *testing.T) {
	t.Run("slice input", func(t *testing.T) {
		var w WantedListExtended
		if err := w.UnmarshalJSON([]byte(`["a","b"]`)); err != nil {
			t.Fatalf("UnmarshalJSON() error = %v", err)
		}
		if len(w.TypeSlice) != 2 || w.TypeSlice[0] != "a" || w.TypeSlice[1] != "b" {
			t.Fatalf("TypeSlice = %#v", w.TypeSlice)
		}
		if len(w.TypeMap) != 0 {
			t.Fatalf("TypeMap should be empty, got %#v", w.TypeMap)
		}
	})

	t.Run("map input", func(t *testing.T) {
		var w WantedListExtended
		if err := w.UnmarshalJSON([]byte(`{"x":["y"]}`)); err != nil {
			t.Fatalf("UnmarshalJSON() error = %v", err)
		}
		if len(w.TypeSlice) != 0 {
			t.Fatalf("TypeSlice should be empty, got %#v", w.TypeSlice)
		}
		if got := w.TypeMap["x"]; len(got) != 1 || got[0] != "y" {
			t.Fatalf("TypeMap = %#v", w.TypeMap)
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		var w WantedListExtended
		if err := w.UnmarshalJSON([]byte(`123`)); err == nil {
			t.Fatalf("expected error for invalid json")
		}
	})

	t.Run("empty input", func(t *testing.T) {
		var w WantedListExtended
		if err := w.UnmarshalJSON([]byte(``)); err != nil {
			t.Fatalf("UnmarshalJSON() error = %v", err)
		}
	})
}
