package lib

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestGetRemoteURLContent(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test content"))
		}))
		defer server.Close()

		content, err := GetRemoteURLContent(server.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(content) != "test content" {
			t.Errorf("got %q, expected %q", string(content), "test content")
		}
	})

	t.Run("non-OK status code", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		_, err := GetRemoteURLContent(server.URL)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("network error", func(t *testing.T) {
		_, err := GetRemoteURLContent("http://invalid.invalid.invalid:1234")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetRemoteURLReader(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("reader content"))
		}))
		defer server.Close()

		reader, err := GetRemoteURLReader(server.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer reader.Close()

		content, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("failed to read: %v", err)
		}
		if string(content) != "reader content" {
			t.Errorf("got %q, expected %q", string(content), "reader content")
		}
	})

	t.Run("non-OK status code", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		_, err := GetRemoteURLReader(server.URL)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("network error", func(t *testing.T) {
		_, err := GetRemoteURLReader("http://invalid.invalid.invalid:1234")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestWantedListExtended_UnmarshalJSON(t *testing.T) {
	t.Run("empty data", func(t *testing.T) {
		var w WantedListExtended
		err := w.UnmarshalJSON([]byte{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("slice type", func(t *testing.T) {
		var w WantedListExtended
		data := []byte(`["a", "b", "c"]`)
		err := w.UnmarshalJSON(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := []string{"a", "b", "c"}
		if !reflect.DeepEqual(w.TypeSlice, expected) {
			t.Errorf("got %v, expected %v", w.TypeSlice, expected)
		}
		if w.TypeMap != nil && len(w.TypeMap) != 0 {
			t.Errorf("expected empty TypeMap, got %v", w.TypeMap)
		}
	})

	t.Run("map type", func(t *testing.T) {
		var w WantedListExtended
		data := []byte(`{"key1": ["val1", "val2"], "key2": ["val3"]}`)
		err := w.UnmarshalJSON(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := map[string][]string{
			"key1": {"val1", "val2"},
			"key2": {"val3"},
		}
		if !reflect.DeepEqual(w.TypeMap, expected) {
			t.Errorf("got %v, expected %v", w.TypeMap, expected)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		var w WantedListExtended
		data := []byte(`{invalid}`)
		err := w.UnmarshalJSON(data)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("standard json.Unmarshal", func(t *testing.T) {
		type container struct {
			Wanted WantedListExtended `json:"wanted"`
		}

		// Test with slice value
		var c1 container
		if err := json.Unmarshal([]byte(`{"wanted": ["item1", "item2"]}`), &c1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(c1.Wanted.TypeSlice) != 2 {
			t.Errorf("expected 2 items in TypeSlice, got %d", len(c1.Wanted.TypeSlice))
		}

		// Test with map value
		var c2 container
		if err := json.Unmarshal([]byte(`{"wanted": {"k": ["v1", "v2"]}}`), &c2); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(c2.Wanted.TypeMap) != 1 {
			t.Errorf("expected 1 key in TypeMap, got %d", len(c2.Wanted.TypeMap))
		}
	})
}
