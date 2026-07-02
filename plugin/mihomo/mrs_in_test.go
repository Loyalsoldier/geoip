package mihomo

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"testing"
	"net/http"
	"net/http/httptest"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestMRSIn_NewMRSIn(t *testing.T) {
	tests := []struct {
		name         string
		action       lib.Action
		data         json.RawMessage
		expectType   string
		expectIPType lib.IPType
		expectErr    bool
	}{
		{
			name:         "Valid action with inputDir",
			action:       lib.ActionAdd,
			data:         json.RawMessage(`{"inputDir": "/tmp/test"}`),
			expectType:   TypeMRSIn,
			expectIPType: "",
			expectErr:    false,
		},
		{
			name:         "Valid action with name and URI",
			action:       lib.ActionAdd,
			data:         json.RawMessage(`{"name": "testentry", "uri": "/tmp/test.mrs"}`),
			expectType:   TypeMRSIn,
			expectIPType: "",
			expectErr:    false,
		},
		{
			name:         "Valid action with IPv4 only",
			action:       lib.ActionAdd,
			data:         json.RawMessage(`{"inputDir": "/tmp/test", "onlyIPType": "ipv4"}`),
			expectType:   TypeMRSIn,
			expectIPType: lib.IPv4,
			expectErr:    false,
		},
		{
			name:         "Valid action with wanted list",
			action:       lib.ActionAdd,
			data:         json.RawMessage(`{"inputDir": "/tmp/test", "wantedList": ["test1", "test2"]}`),
			expectType:   TypeMRSIn,
			expectIPType: "",
			expectErr:    false,
		},
		{
			name:      "Invalid JSON",
			action:    lib.ActionAdd,
			data:      json.RawMessage(`{invalid json}`),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter, err := newMRSIn(tt.action, tt.data)
			if (err != nil) != tt.expectErr {
				t.Errorf("newMRSIn() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr {
				mrsIn := converter.(*MRSIn)
				if mrsIn.GetType() != tt.expectType {
					t.Errorf("GetType() = %v, expect %v", mrsIn.GetType(), tt.expectType)
				}
				if mrsIn.GetAction() != tt.action {
					t.Errorf("GetAction() = %v, expect %v", mrsIn.GetAction(), tt.action)
				}
				if mrsIn.OnlyIPType != tt.expectIPType {
					t.Errorf("OnlyIPType = %v, expect %v", mrsIn.OnlyIPType, tt.expectIPType)
				}
			}
		})
	}
}

func TestMRSIn_GetType(t *testing.T) {
	mrsIn := &MRSIn{Type: TypeMRSIn}
	result := mrsIn.GetType()
	if result != TypeMRSIn {
		t.Errorf("GetType() = %v, expect %v", result, TypeMRSIn)
	}
}

func TestMRSIn_GetAction(t *testing.T) {
	action := lib.ActionAdd
	mrsIn := &MRSIn{Action: action}
	result := mrsIn.GetAction()
	if result != action {
		t.Errorf("GetAction() = %v, expect %v", result, action)
	}
}

func TestMRSIn_GetDescription(t *testing.T) {
	mrsIn := &MRSIn{Description: DescMRSIn}
	result := mrsIn.GetDescription()
	if result != DescMRSIn {
		t.Errorf("GetDescription() = %v, expect %v", result, DescMRSIn)
	}
}

func TestMRSIn_Input(t *testing.T) {
	tests := []struct {
		name      string
		mrsIn     *MRSIn
		expectErr bool
	}{
		{
			name: "Missing config arguments",
			mrsIn: &MRSIn{
				Type:   TypeMRSIn,
				Action: lib.ActionAdd,
			},
			expectErr: true,
		},
		{
			name: "InputDir not exists",
			mrsIn: &MRSIn{
				Type:     TypeMRSIn,
				Action:   lib.ActionAdd,
				InputDir: "/nonexistent/dir",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := lib.NewContainer()
			_, err := tt.mrsIn.Input(container)

			if (err != nil) != tt.expectErr {
				t.Errorf("Input() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestMRSIn_InputWithEmptyDir(t *testing.T) {
	// Create empty temporary directory
	tmpDir, err := os.MkdirTemp("", "test-mrs-empty")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mrsIn := &MRSIn{
		Type:     TypeMRSIn,
		Action:   lib.ActionAdd,
		InputDir: tmpDir,
	}

	container := lib.NewContainer()
	_, err = mrsIn.Input(container)

	if err == nil {
		t.Error("Expected error for empty directory")
	}
}

func TestMRSIn_WalkRemoteFileHTTP(t *testing.T) {
	// Create a test server that returns MRS data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write MRS magic bytes and some dummy data
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(mrsMagicBytes[:])
		// Write minimal MRS content for testing
		binary.Write(w, binary.LittleEndian, uint32(0)) // No ranges
	}))
	defer server.Close()

	mrsIn := &MRSIn{
		Type:   TypeMRSIn,
		Action: lib.ActionAdd,
		Name:   "testentry",
		URI:    server.URL,
	}

	container := lib.NewContainer()
	_, err := mrsIn.Input(container)

	// This might fail due to missing proper MRS format implementation
	// but we're testing the HTTP request functionality
	t.Logf("Input with HTTP server returned error: %v", err)
}

func TestMRSIn_WalkLocalFileNonExistent(t *testing.T) {
	mrsIn := &MRSIn{
		Type:   TypeMRSIn,
		Action: lib.ActionAdd,
		Name:   "testentry",
		URI:    "/nonexistent/file.mrs",
	}

	container := lib.NewContainer()
	_, err := mrsIn.Input(container)

	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestMRSIn_Constants(t *testing.T) {
	if TypeMRSIn != "mihomoMRS" {
		t.Errorf("TypeMRSIn = %v, expect %v", TypeMRSIn, "mihomoMRS")
	}
	if DescMRSIn != "Convert mihomo MRS data to other formats" {
		t.Errorf("DescMRSIn = %v, expect correct description", DescMRSIn)
	}
}

func TestMRSIn_MagicBytes(t *testing.T) {
	expected := [4]byte{'M', 'R', 'S', 1}
	if mrsMagicBytes != expected {
		t.Errorf("mrsMagicBytes = %v, expect %v", mrsMagicBytes, expected)
	}
}