package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetRemoteURLContent(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    bool
		errMessage string
		want       string
	}{
		{
			name: "successful request",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("test content"))
			},
			wantErr: false,
			want:    "test content",
		},
		{
			name: "404 not found",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr:    true,
			errMessage: "404 Not Found",
		},
		{
			name: "500 internal server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr:    true,
			errMessage: "500 Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			got, err := GetRemoteURLContent(server.URL)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRemoteURLContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err != nil && tt.errMessage != "" && err.Error() != fmt.Sprintf("failed to get remote content -> %s: %s", server.URL, tt.errMessage) {
					t.Errorf("GetRemoteURLContent() error message = %v, want substring %v", err.Error(), tt.errMessage)
				}
				return
			}
			if string(got) != tt.want {
				t.Errorf("GetRemoteURLContent() = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestGetRemoteURLContentInvalidURL(t *testing.T) {
	_, err := GetRemoteURLContent("http://invalid-url-that-does-not-exist-12345.com")
	if err == nil {
		t.Error("GetRemoteURLContent() expected error for invalid URL, got nil")
	}
}

func TestGetRemoteURLReader(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    bool
		errMessage string
		want       string
	}{
		{
			name: "successful request",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("test reader content"))
			},
			wantErr: false,
			want:    "test reader content",
		},
		{
			name: "404 not found",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr:    true,
			errMessage: "404 Not Found",
		},
		{
			name: "403 forbidden",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			},
			wantErr:    true,
			errMessage: "403 Forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			reader, err := GetRemoteURLReader(server.URL)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRemoteURLReader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err != nil && tt.errMessage != "" && err.Error() != fmt.Sprintf("failed to get remote content -> %s: %s", server.URL, tt.errMessage) {
					t.Errorf("GetRemoteURLReader() error message = %v, want substring %v", err.Error(), tt.errMessage)
				}
				return
			}
			defer reader.Close()
			got, err := io.ReadAll(reader)
			if err != nil {
				t.Errorf("Failed to read from reader: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("GetRemoteURLReader() content = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestGetRemoteURLReaderInvalidURL(t *testing.T) {
	_, err := GetRemoteURLReader("http://invalid-url-that-does-not-exist-12345.com")
	if err == nil {
		t.Error("GetRemoteURLReader() expected error for invalid URL, got nil")
	}
}

func TestGetIgnoreIPType(t *testing.T) {
	tests := []struct {
		name       string
		onlyIPType IPType
		want       IPType
	}{
		{
			name:       "IPv4 returns IgnoreIPv6",
			onlyIPType: IPv4,
			want:       IPv6,
		},
		{
			name:       "IPv6 returns IgnoreIPv4",
			onlyIPType: IPv6,
			want:       IPv4,
		},
		{
			name:       "empty string returns nil",
			onlyIPType: IPType(""),
			want:       IPType(""),
		},
		{
			name:       "invalid type returns nil",
			onlyIPType: IPType("invalid"),
			want:       IPType(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetIgnoreIPType(tt.onlyIPType)
			if got == nil && tt.want == "" {
				// nil is expected
				return
			}
			if got == nil {
				t.Errorf("GetIgnoreIPType(%q) = nil, want %q", tt.onlyIPType, tt.want)
				return
			}
			result := got()
			if result != tt.want {
				t.Errorf("GetIgnoreIPType(%q)() = %q, want %q", tt.onlyIPType, result, tt.want)
			}
		})
	}
}

func TestWantedListExtended_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantSlice []string
		wantMap   map[string][]string
		wantErr   bool
	}{
		{
			name:      "slice format",
			input:     `["item1", "item2", "item3"]`,
			wantSlice: []string{"item1", "item2", "item3"},
			wantMap:   map[string][]string{},
			wantErr:   false,
		},
		{
			name:      "map format",
			input:     `{"key1": ["val1", "val2"], "key2": ["val3"]}`,
			wantSlice: []string{},
			wantMap:   map[string][]string{"key1": {"val1", "val2"}, "key2": {"val3"}},
			wantErr:   false,
		},
		{
			name:      "empty slice",
			input:     `[]`,
			wantSlice: []string{},
			wantMap:   map[string][]string{},
			wantErr:   false,
		},
		{
			name:      "empty map",
			input:     `{}`,
			wantSlice: []string{},
			wantMap:   map[string][]string{},
			wantErr:   false,
		},
		{
			name:      "invalid json",
			input:     `{invalid}`,
			wantSlice: nil,
			wantMap:   nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w WantedListExtended
			err := json.Unmarshal([]byte(tt.input), &w)
			if (err != nil) != tt.wantErr {
				t.Errorf("WantedListExtended.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Check slice
			if tt.wantSlice == nil && w.TypeSlice != nil {
				t.Errorf("WantedListExtended.TypeSlice = %v, want nil", w.TypeSlice)
			} else if tt.wantSlice != nil {
				if len(w.TypeSlice) != len(tt.wantSlice) {
					t.Errorf("WantedListExtended.TypeSlice length = %d, want %d", len(w.TypeSlice), len(tt.wantSlice))
				} else {
					for i, v := range tt.wantSlice {
						if w.TypeSlice[i] != v {
							t.Errorf("WantedListExtended.TypeSlice[%d] = %q, want %q", i, w.TypeSlice[i], v)
						}
					}
				}
			}

			// Check map
			if tt.wantMap == nil && w.TypeMap != nil {
				t.Errorf("WantedListExtended.TypeMap = %v, want nil", w.TypeMap)
			} else if tt.wantMap != nil {
				if len(w.TypeMap) != len(tt.wantMap) {
					t.Errorf("WantedListExtended.TypeMap length = %d, want %d", len(w.TypeMap), len(tt.wantMap))
				} else {
					for k, v := range tt.wantMap {
						gotV, ok := w.TypeMap[k]
						if !ok {
							t.Errorf("WantedListExtended.TypeMap missing key %q", k)
							continue
						}
						if len(gotV) != len(v) {
							t.Errorf("WantedListExtended.TypeMap[%q] length = %d, want %d", k, len(gotV), len(v))
							continue
						}
						for i, val := range v {
							if gotV[i] != val {
								t.Errorf("WantedListExtended.TypeMap[%q][%d] = %q, want %q", k, i, gotV[i], val)
							}
						}
					}
				}
			}
		})
	}
}
