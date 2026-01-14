package lib

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetRemoteURLContent_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	}))
	defer server.Close()

	content, err := GetRemoteURLContent(server.URL)
	if err != nil {
		t.Fatalf("GetRemoteURLContent failed: %v", err)
	}

	if string(content) != "test content" {
		t.Errorf("GetRemoteURLContent = %s, want 'test content'", string(content))
	}
}

func TestGetRemoteURLContent_NotFound(t *testing.T) {
	// Create test server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := GetRemoteURLContent(server.URL)
	if err == nil {
		t.Error("GetRemoteURLContent should fail for 404")
	}
}

func TestGetRemoteURLContent_InvalidURL(t *testing.T) {
	_, err := GetRemoteURLContent("http://invalid-url-that-does-not-exist.local")
	if err == nil {
		t.Error("GetRemoteURLContent should fail for invalid URL")
	}
}

func TestGetRemoteURLReader_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	}))
	defer server.Close()

	reader, err := GetRemoteURLReader(server.URL)
	if err != nil {
		t.Fatalf("GetRemoteURLReader failed: %v", err)
	}
	defer reader.Close()

	buf := make([]byte, 1024)
	n, _ := reader.Read(buf)
	if string(buf[:n]) != "test content" {
		t.Errorf("GetRemoteURLReader content = %s, want 'test content'", string(buf[:n]))
	}
}

func TestGetRemoteURLReader_NotFound(t *testing.T) {
	// Create test server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := GetRemoteURLReader(server.URL)
	if err == nil {
		t.Error("GetRemoteURLReader should fail for 404")
	}
}

func TestGetRemoteURLReader_InvalidURL(t *testing.T) {
	_, err := GetRemoteURLReader("http://invalid-url-that-does-not-exist.local")
	if err == nil {
		t.Error("GetRemoteURLReader should fail for invalid URL")
	}
}

func TestWantedListExtended_UnmarshalJSON_Slice(t *testing.T) {
	jsonData := []byte(`["item1", "item2", "item3"]`)

	var w WantedListExtended
	err := json.Unmarshal(jsonData, &w)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if len(w.TypeSlice) != 3 {
		t.Errorf("len(TypeSlice) = %d, want 3", len(w.TypeSlice))
	}
	if len(w.TypeMap) != 0 {
		t.Errorf("len(TypeMap) = %d, want 0", len(w.TypeMap))
	}

	expectedSlice := []string{"item1", "item2", "item3"}
	for i, v := range w.TypeSlice {
		if v != expectedSlice[i] {
			t.Errorf("TypeSlice[%d] = %s, want %s", i, v, expectedSlice[i])
		}
	}
}

func TestWantedListExtended_UnmarshalJSON_Map(t *testing.T) {
	jsonData := []byte(`{"key1": ["value1", "value2"], "key2": ["value3"]}`)

	var w WantedListExtended
	err := json.Unmarshal(jsonData, &w)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if len(w.TypeSlice) != 0 {
		t.Errorf("len(TypeSlice) = %d, want 0", len(w.TypeSlice))
	}
	if len(w.TypeMap) != 2 {
		t.Errorf("len(TypeMap) = %d, want 2", len(w.TypeMap))
	}

	if len(w.TypeMap["key1"]) != 2 {
		t.Errorf("len(TypeMap[key1]) = %d, want 2", len(w.TypeMap["key1"]))
	}
	if len(w.TypeMap["key2"]) != 1 {
		t.Errorf("len(TypeMap[key2]) = %d, want 1", len(w.TypeMap["key2"]))
	}
}

func TestWantedListExtended_UnmarshalJSON_EmptyData(t *testing.T) {
	// Test calling UnmarshalJSON directly with empty data
	var w WantedListExtended
	err := w.UnmarshalJSON([]byte{})
	if err != nil {
		t.Fatalf("UnmarshalJSON with empty data failed: %v", err)
	}

	if len(w.TypeSlice) != 0 {
		t.Errorf("len(TypeSlice) = %d, want 0", len(w.TypeSlice))
	}
	if len(w.TypeMap) != 0 {
		t.Errorf("len(TypeMap) = %d, want 0", len(w.TypeMap))
	}
}

func TestWantedListExtended_UnmarshalJSON_Invalid(t *testing.T) {
	// Invalid JSON that is neither slice nor map
	jsonData := []byte(`123`)

	var w WantedListExtended
	err := json.Unmarshal(jsonData, &w)
	if err == nil {
		t.Error("UnmarshalJSON should fail for invalid format")
	}
}

func TestWantedListExtended_UnmarshalJSON_EmptySlice(t *testing.T) {
	jsonData := []byte(`[]`)

	var w WantedListExtended
	err := json.Unmarshal(jsonData, &w)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if len(w.TypeSlice) != 0 {
		t.Errorf("len(TypeSlice) = %d, want 0", len(w.TypeSlice))
	}
}

func TestWantedListExtended_UnmarshalJSON_EmptyMap(t *testing.T) {
	jsonData := []byte(`{}`)

	var w WantedListExtended
	err := json.Unmarshal(jsonData, &w)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	// Empty object is a valid map
	if len(w.TypeMap) != 0 {
		t.Errorf("len(TypeMap) = %d, want 0", len(w.TypeMap))
	}
}
