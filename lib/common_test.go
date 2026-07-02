package lib

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetRemoteURLContent(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{
			name:       "successful request",
			statusCode: http.StatusOK,
			body:       "test content",
			wantErr:    false,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			body:       "",
			wantErr:    true,
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			body:       "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			got, err := GetRemoteURLContent(server.URL)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRemoteURLContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.body {
				t.Errorf("GetRemoteURLContent() = %v, want %v", string(got), tt.body)
			}
		})
	}
}

func TestGetRemoteURLContentInvalidURL(t *testing.T) {
	_, err := GetRemoteURLContent("invalid://url")
	if err == nil {
		t.Error("GetRemoteURLContent() should return error for invalid URL")
	}
}

func TestGetRemoteURLReader(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{
			name:       "successful request",
			statusCode: http.StatusOK,
			body:       "test content",
			wantErr:    false,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			body:       "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			got, err := GetRemoteURLReader(server.URL)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRemoteURLReader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				defer got.Close()
				if got == nil {
					t.Error("GetRemoteURLReader() returned nil reader")
				}
			}
		})
	}
}

func TestGetRemoteURLReaderInvalidURL(t *testing.T) {
	_, err := GetRemoteURLReader("invalid://url")
	if err == nil {
		t.Error("GetRemoteURLReader() should return error for invalid URL")
	}
}

func TestWantedListExtended_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantErr  bool
		checkFn  func(*testing.T, *WantedListExtended)
	}{
		{
			name:    "empty object",
			json:    `{}`,
			wantErr: false,
			checkFn: func(t *testing.T, w *WantedListExtended) {
				if len(w.TypeSlice) != 0 || len(w.TypeMap) != 0 {
					t.Error("WantedListExtended should have empty slices/maps for empty object")
				}
			},
		},
		{
			name:    "empty array",
			json:    `[]`,
			wantErr: false,
			checkFn: func(t *testing.T, w *WantedListExtended) {
				if len(w.TypeSlice) != 0 {
					t.Error("TypeSlice should be empty for empty array")
				}
			},
		},
		{
			name:    "slice format",
			json:    `["type1", "type2", "type3"]`,
			wantErr: false,
			checkFn: func(t *testing.T, w *WantedListExtended) {
				if len(w.TypeSlice) != 3 {
					t.Errorf("TypeSlice length = %d, want 3", len(w.TypeSlice))
				}
				if w.TypeSlice[0] != "type1" {
					t.Errorf("TypeSlice[0] = %s, want 'type1'", w.TypeSlice[0])
				}
			},
		},
		{
			name:    "map format",
			json:    `{"key1": ["val1", "val2"], "key2": ["val3"]}`,
			wantErr: false,
			checkFn: func(t *testing.T, w *WantedListExtended) {
				if len(w.TypeMap) != 2 {
					t.Errorf("TypeMap length = %d, want 2", len(w.TypeMap))
				}
				if len(w.TypeMap["key1"]) != 2 {
					t.Errorf("TypeMap[key1] length = %d, want 2", len(w.TypeMap["key1"]))
				}
			},
		},
		{
			name:    "invalid json",
			json:    `{invalid}`,
			wantErr: true,
			checkFn: nil,
		},
		{
			name:    "number type (not array or map)",
			json:    `123`,
			wantErr: true,
			checkFn: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w WantedListExtended
			err := json.Unmarshal([]byte(tt.json), &w)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.checkFn != nil {
				tt.checkFn(t, &w)
			}
		})
	}
}
