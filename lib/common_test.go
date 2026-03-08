package lib

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetRemoteURLContent(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	}))
	defer server.Close()

	// Test successful request
	content, err := GetRemoteURLContent(server.URL)
	if err != nil {
		t.Errorf("GetRemoteURLContent() should not return error: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("GetRemoteURLContent() content = %s; want 'test content'", string(content))
	}
}

func TestGetRemoteURLContentNotFound(t *testing.T) {
	// Create a test server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	// Test 404 response
	_, err := GetRemoteURLContent(server.URL)
	if err == nil {
		t.Error("GetRemoteURLContent() should return error for 404 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("Error should mention 404 status, got: %v", err)
	}
}

func TestGetRemoteURLContentInvalidURL(t *testing.T) {
	// Test with invalid URL
	_, err := GetRemoteURLContent("invalid-url")
	if err == nil {
		t.Error("GetRemoteURLContent() should return error for invalid URL")
	}
}

func TestGetRemoteURLReader(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test reader content"))
	}))
	defer server.Close()

	// Test successful request
	reader, err := GetRemoteURLReader(server.URL)
	if err != nil {
		t.Errorf("GetRemoteURLReader() should not return error: %v", err)
	}
	defer reader.Close()

	if reader == nil {
		t.Error("GetRemoteURLReader() should return non-nil reader")
	}

	// Read content from reader
	buffer := make([]byte, 100)
	n, err := reader.Read(buffer)
	if err != nil && err.Error() != "EOF" {
		t.Errorf("Reading from reader should not return error: %v", err)
	}
	content := string(buffer[:n])
	if content != "test reader content" {
		t.Errorf("Reader content = %s; want 'test reader content'", content)
	}
}

func TestGetRemoteURLReaderNotFound(t *testing.T) {
	// Create a test server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Test 404 response
	_, err := GetRemoteURLReader(server.URL)
	if err == nil {
		t.Error("GetRemoteURLReader() should return error for 404 response")
	}
}

func TestGetRemoteURLReaderInvalidURL(t *testing.T) {
	// Test with invalid URL
	_, err := GetRemoteURLReader("invalid-url")
	if err == nil {
		t.Error("GetRemoteURLReader() should return error for invalid URL")
	}
}

func TestWantedListExtended_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectSlice []string
		expectMap   map[string][]string
		expectError bool
		sliceSet    bool
		mapSet      bool
	}{
		{
			name:        "Array input",
			input:       `["item1", "item2", "item3"]`,
			expectSlice: []string{"item1", "item2", "item3"},
			expectMap:   map[string][]string{},
			expectError: false,
			sliceSet:    true,
			mapSet:      false,
		},
		{
			name:        "Object input",
			input:       `{"key1": ["value1", "value2"], "key2": ["value3"]}`,
			expectSlice: []string{},
			expectMap:   map[string][]string{"key1": {"value1", "value2"}, "key2": {"value3"}},
			expectError: false,
			sliceSet:    false,
			mapSet:      true,
		},
		{
			name:        "Empty array",
			input:       `[]`,
			expectSlice: []string{},
			expectMap:   map[string][]string{},
			expectError: false,
			sliceSet:    true,
			mapSet:      false,
		},
		{
			name:        "Empty object",
			input:       `{}`,
			expectSlice: []string{},
			expectMap:   map[string][]string{},
			expectError: false,
			sliceSet:    false,
			mapSet:      true,
		},
		{
			name:        "Empty string input",
			input:       `""`,
			expectSlice: []string{},
			expectMap:   map[string][]string{},
			expectError: true,
			sliceSet:    false,
			mapSet:      false,
		},
		{
			name:        "Invalid JSON",
			input:       `{invalid json}`,
			expectSlice: nil,
			expectMap:   nil,
			expectError: true,
			sliceSet:    false,
			mapSet:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wl WantedListExtended
			err := json.Unmarshal([]byte(tt.input), &wl)

			if tt.expectError && err == nil {
				t.Errorf("UnmarshalJSON() should return error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("UnmarshalJSON() should not return error but got: %v", err)
			}

			if !tt.expectError {
				// For slice
				if tt.sliceSet {
					if len(wl.TypeSlice) != len(tt.expectSlice) {
						t.Errorf("TypeSlice length = %d; want %d", len(wl.TypeSlice), len(tt.expectSlice))
					}
					for i, expected := range tt.expectSlice {
						if i < len(wl.TypeSlice) && wl.TypeSlice[i] != expected {
							t.Errorf("TypeSlice[%d] = %s; want %s", i, wl.TypeSlice[i], expected)
						}
					}
				}

				// For map
				if tt.mapSet {
					if len(wl.TypeMap) != len(tt.expectMap) {
						t.Errorf("TypeMap length = %d; want %d", len(wl.TypeMap), len(tt.expectMap))
					}
					for key, expectedValues := range tt.expectMap {
						actualValues, exists := wl.TypeMap[key]
						if !exists {
							t.Errorf("TypeMap should contain key %s", key)
							continue
						}
						if len(actualValues) != len(expectedValues) {
							t.Errorf("TypeMap[%s] length = %d; want %d", key, len(actualValues), len(expectedValues))
							continue
						}
						for i, expectedValue := range expectedValues {
							if i < len(actualValues) && actualValues[i] != expectedValue {
								t.Errorf("TypeMap[%s][%d] = %s; want %s", key, i, actualValues[i], expectedValue)
							}
						}
					}
				}
			}
		})
	}
}

func TestWantedListExtended_UnmarshalJSON_EmptyData(t *testing.T) {
	var wl WantedListExtended
	err := wl.UnmarshalJSON([]byte{})
	if err != nil {
		t.Errorf("UnmarshalJSON() with empty data should not return error: %v", err)
	}
	// For empty data, the function returns early and doesn't set anything
	if wl.TypeSlice != nil {
		t.Error("TypeSlice should be nil for empty data")
	}
	if wl.TypeMap != nil {
		t.Error("TypeMap should be nil for empty data")
	}
}

func TestWantedListExtended_UnmarshalJSON_StringInput(t *testing.T) {
	// Test with string input (should fail to parse as both array and object)
	var wl WantedListExtended
	err := wl.UnmarshalJSON([]byte(`"string value"`))
	if err == nil {
		t.Error("UnmarshalJSON() with string input should return error")
	}
}

func TestRemoteContentHTTPMethods(t *testing.T) {
	// Test that both functions use GET method
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	// Test GetRemoteURLContent uses GET
	_, err := GetRemoteURLContent(server.URL)
	if err != nil {
		t.Errorf("GetRemoteURLContent() failed: %v", err)
	}

	// Test GetRemoteURLReader uses GET
	reader, err := GetRemoteURLReader(server.URL)
	if err != nil {
		t.Errorf("GetRemoteURLReader() failed: %v", err)
	}
	if reader != nil {
		reader.Close()
	}
}

func TestRemoteContentHeaders(t *testing.T) {
	// Test server that checks headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that User-Agent is set (Go's http client sets it by default)
		if r.UserAgent() == "" {
			t.Error("User-Agent header should be set")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	_, err := GetRemoteURLContent(server.URL)
	if err != nil {
		t.Errorf("GetRemoteURLContent() failed: %v", err)
	}
}