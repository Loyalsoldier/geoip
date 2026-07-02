package singbox

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestSRSIn_NewSRSIn(t *testing.T) {
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
			expectType:   TypeSRSIn,
			expectIPType: "",
			expectErr:    false,
		},
		{
			name:         "Valid action with name and URI",
			action:       lib.ActionAdd,
			data:         json.RawMessage(`{"name": "testentry", "uri": "/tmp/test.srs"}`),
			expectType:   TypeSRSIn,
			expectIPType: "",
			expectErr:    false,
		},
		{
			name:         "Valid action with IPv4 only",
			action:       lib.ActionAdd,
			data:         json.RawMessage(`{"inputDir": "/tmp/test", "onlyIPType": "ipv4"}`),
			expectType:   TypeSRSIn,
			expectIPType: lib.IPv4,
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
			converter, err := newSRSIn(tt.action, tt.data)
			if (err != nil) != tt.expectErr {
				t.Errorf("newSRSIn() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr {
				srsIn := converter.(*SRSIn)
				if srsIn.GetType() != tt.expectType {
					t.Errorf("GetType() = %v, expect %v", srsIn.GetType(), tt.expectType)
				}
				if srsIn.GetAction() != tt.action {
					t.Errorf("GetAction() = %v, expect %v", srsIn.GetAction(), tt.action)
				}
				if srsIn.OnlyIPType != tt.expectIPType {
					t.Errorf("OnlyIPType = %v, expect %v", srsIn.OnlyIPType, tt.expectIPType)
				}
			}
		})
	}
}

func TestSRSIn_GetType(t *testing.T) {
	srsIn := &SRSIn{Type: TypeSRSIn}
	result := srsIn.GetType()
	if result != TypeSRSIn {
		t.Errorf("GetType() = %v, expect %v", result, TypeSRSIn)
	}
}

func TestSRSIn_GetAction(t *testing.T) {
	action := lib.ActionAdd
	srsIn := &SRSIn{Action: action}
	result := srsIn.GetAction()
	if result != action {
		t.Errorf("GetAction() = %v, expect %v", result, action)
	}
}

func TestSRSIn_GetDescription(t *testing.T) {
	srsIn := &SRSIn{Description: DescSRSIn}
	result := srsIn.GetDescription()
	if result != DescSRSIn {
		t.Errorf("GetDescription() = %v, expect %v", result, DescSRSIn)
	}
}

func TestSRSIn_Input(t *testing.T) {
	tests := []struct {
		name      string
		srsIn     *SRSIn
		expectErr bool
	}{
		{
			name: "Missing config arguments",
			srsIn: &SRSIn{
				Type:   TypeSRSIn,
				Action: lib.ActionAdd,
			},
			expectErr: true,
		},
		{
			name: "InputDir not exists",
			srsIn: &SRSIn{
				Type:     TypeSRSIn,
				Action:   lib.ActionAdd,
				InputDir: "/nonexistent/dir",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := lib.NewContainer()
			_, err := tt.srsIn.Input(container)

			if (err != nil) != tt.expectErr {
				t.Errorf("Input() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestSRSIn_Constants(t *testing.T) {
	if TypeSRSIn != "singboxSRS" {
		t.Errorf("TypeSRSIn = %v, expect %v", TypeSRSIn, "singboxSRS")
	}
	if DescSRSIn != "Convert sing-box SRS data to other formats" {
		t.Errorf("DescSRSIn = %v, expect correct description", DescSRSIn)
	}
}

// SRS Output Tests

func TestSRSOut_NewSRSOut(t *testing.T) {
	tests := []struct {
		name            string
		action          lib.Action
		data            json.RawMessage
		expectType      string
		expectIPType    lib.IPType
		expectOutputDir string
		expectErr       bool
	}{
		{
			name:            "Valid action with default output dir",
			action:          lib.ActionOutput,
			data:            json.RawMessage(`{}`),
			expectType:      TypeSRSOut,
			expectIPType:    "",
			expectOutputDir: defaultOutputDir,
			expectErr:       false,
		},
		{
			name:            "Valid action with custom output dir",
			action:          lib.ActionOutput,
			data:            json.RawMessage(`{"outputDir": "/tmp/custom"}`),
			expectType:      TypeSRSOut,
			expectIPType:    "",
			expectOutputDir: "/tmp/custom",
			expectErr:       false,
		},
		{
			name:            "Valid action with IPv4 only",
			action:          lib.ActionOutput,
			data:            json.RawMessage(`{"onlyIPType": "ipv4"}`),
			expectType:      TypeSRSOut,
			expectIPType:    lib.IPv4,
			expectOutputDir: defaultOutputDir,
			expectErr:       false,
		},
		{
			name:      "Invalid JSON",
			action:    lib.ActionOutput,
			data:      json.RawMessage(`{invalid json}`),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter, err := newSRSOut(tt.action, tt.data)
			if (err != nil) != tt.expectErr {
				t.Errorf("newSRSOut() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr {
				srsOut := converter.(*SRSOut)
				if srsOut.GetType() != tt.expectType {
					t.Errorf("GetType() = %v, expect %v", srsOut.GetType(), tt.expectType)
				}
				if srsOut.GetAction() != tt.action {
					t.Errorf("GetAction() = %v, expect %v", srsOut.GetAction(), tt.action)
				}
				if srsOut.OnlyIPType != tt.expectIPType {
					t.Errorf("OnlyIPType = %v, expect %v", srsOut.OnlyIPType, tt.expectIPType)
				}
				if srsOut.OutputDir != tt.expectOutputDir {
					t.Errorf("OutputDir = %v, expect %v", srsOut.OutputDir, tt.expectOutputDir)
				}
			}
		})
	}
}

func TestSRSOut_GetType(t *testing.T) {
	srsOut := &SRSOut{Type: TypeSRSOut}
	result := srsOut.GetType()
	if result != TypeSRSOut {
		t.Errorf("GetType() = %v, expect %v", result, TypeSRSOut)
	}
}

func TestSRSOut_GetAction(t *testing.T) {
	action := lib.ActionOutput
	srsOut := &SRSOut{Action: action}
	result := srsOut.GetAction()
	if result != action {
		t.Errorf("GetAction() = %v, expect %v", result, action)
	}
}

func TestSRSOut_GetDescription(t *testing.T) {
	srsOut := &SRSOut{Description: DescSRSOut}
	result := srsOut.GetDescription()
	if result != DescSRSOut {
		t.Errorf("GetDescription() = %v, expect %v", result, DescSRSOut)
	}
}

func TestSRSOut_Output(t *testing.T) {
	// Create temporary output directory
	tmpDir, err := os.MkdirTemp("", "test-srs-output")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name      string
		srsOut    *SRSOut
		expectErr bool
	}{
		{
			name: "Output all entries",
			srsOut: &SRSOut{
				Type:      TypeSRSOut,
				Action:    lib.ActionOutput,
				OutputDir: tmpDir,
			},
			expectErr: false,
		},
		{
			name: "Output with wanted list",
			srsOut: &SRSOut{
				Type:      TypeSRSOut,
				Action:    lib.ActionOutput,
				OutputDir: tmpDir,
				Want:      []string{"TEST1"},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a container with test entries
			container := lib.NewContainer()

			entry1 := lib.NewEntry("TEST1")
			if err := entry1.AddPrefix("192.168.1.0/24"); err != nil {
				t.Fatalf("Failed to add prefix to entry1: %v", err)
			}
			if err := container.Add(entry1); err != nil {
				t.Fatalf("Failed to add entry1: %v", err)
			}

			err := tt.srsOut.Output(container)

			if (err != nil) != tt.expectErr {
				t.Errorf("Output() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			if !tt.expectErr {
				// Check if output files were created
				files, err := os.ReadDir(tmpDir)
				if err != nil {
					t.Errorf("Failed to read output directory: %v", err)
					return
				}

				// Verify files were created
				if len(files) == 0 {
					t.Error("No output files were created")
				}

				// Check for .srs files
				foundSRS := false
				for _, file := range files {
					if filepath.Ext(file.Name()) == ".srs" {
						foundSRS = true
						break
					}
				}
				if !foundSRS {
					t.Error("No .srs files were created")
				}
			}
		})
	}
}

func TestSRSOut_Constants(t *testing.T) {
	if TypeSRSOut != "singboxSRS" {
		t.Errorf("TypeSRSOut = %v, expect %v", TypeSRSOut, "singboxSRS")
	}
	if DescSRSOut != "Convert data to sing-box SRS format" {
		t.Errorf("DescSRSOut = %v, expect correct description", DescSRSOut)
	}
	expectedDefaultDir := filepath.Join("./", "output", "srs")
	if defaultOutputDir != expectedDefaultDir {
		t.Errorf("defaultOutputDir = %v, expect %v", defaultOutputDir, expectedDefaultDir)
	}
}