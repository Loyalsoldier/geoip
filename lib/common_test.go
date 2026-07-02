package lib

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetRemoteURLContent(t *testing.T) {
	// Success
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello world"))
	}))
	defer server.Close()

	content, err := GetRemoteURLContent(server.URL)
	if err != nil {
		t.Errorf("GetRemoteURLContent error = %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(content))
	}

	// Non-200 status
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server2.Close()

	_, err = GetRemoteURLContent(server2.URL)
	if err == nil {
		t.Error("expected error for non-200 status")
	}

	// Invalid URL
	_, err = GetRemoteURLContent("http://invalid-host-that-does-not-exist.example.com")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestGetRemoteURLReader(t *testing.T) {
	// Success
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	}))
	defer server.Close()

	reader, err := GetRemoteURLReader(server.URL)
	if err != nil {
		t.Errorf("GetRemoteURLReader error = %v", err)
	}
	if reader != nil {
		reader.Close()
	}

	// Non-200 status
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server2.Close()

	_, err = GetRemoteURLReader(server2.URL)
	if err == nil {
		t.Error("expected error for non-200 status")
	}

	// Invalid URL
	_, err = GetRemoteURLReader("http://invalid-host-that-does-not-exist.example.com")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestGetIgnoreIPType(t *testing.T) {
	// IPv4 -> IgnoreIPv6
	opt := GetIgnoreIPType(IPv4)
	if opt == nil {
		t.Fatal("expected non-nil option for IPv4")
	}
	if opt() != IPv6 {
		t.Error("expected IgnoreIPv6 for IPv4 input")
	}

	// IPv6 -> IgnoreIPv4
	opt = GetIgnoreIPType(IPv6)
	if opt == nil {
		t.Fatal("expected non-nil option for IPv6")
	}
	if opt() != IPv4 {
		t.Error("expected IgnoreIPv4 for IPv6 input")
	}

	// Other -> nil
	opt = GetIgnoreIPType(IPType("other"))
	if opt != nil {
		t.Error("expected nil option for unknown IP type")
	}

	// Empty -> nil
	opt = GetIgnoreIPType(IPType(""))
	if opt != nil {
		t.Error("expected nil option for empty IP type")
	}
}

func TestWantedListExtendedUnmarshalJSON(t *testing.T) {
	// Slice format
	w := &WantedListExtended{}
	data := []byte(`["type1", "type2"]`)
	if err := json.Unmarshal(data, w); err != nil {
		t.Errorf("UnmarshalJSON slice error = %v", err)
	}
	if len(w.TypeSlice) != 2 {
		t.Errorf("expected 2 types, got %d", len(w.TypeSlice))
	}
	if w.TypeSlice[0] != "type1" || w.TypeSlice[1] != "type2" {
		t.Errorf("unexpected TypeSlice: %v", w.TypeSlice)
	}

	// Map format
	w2 := &WantedListExtended{}
	data2 := []byte(`{"key1": ["val1", "val2"], "key2": ["val3"]}`)
	if err := json.Unmarshal(data2, w2); err != nil {
		t.Errorf("UnmarshalJSON map error = %v", err)
	}
	if len(w2.TypeMap) != 2 {
		t.Errorf("expected 2 keys in map, got %d", len(w2.TypeMap))
	}

	// Empty data
	w3 := &WantedListExtended{}
	if err := w3.UnmarshalJSON(nil); err != nil {
		t.Errorf("UnmarshalJSON empty error = %v", err)
	}
	if err := w3.UnmarshalJSON([]byte{}); err != nil {
		t.Errorf("UnmarshalJSON empty bytes error = %v", err)
	}

	// Invalid JSON
	w4 := &WantedListExtended{}
	if err := w4.UnmarshalJSON([]byte(`{invalid}`)); err == nil {
		t.Error("expected error for invalid JSON")
	}
}
