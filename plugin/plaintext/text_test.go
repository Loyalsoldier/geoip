package plaintext

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestTextOut_NewTextOut(t *testing.T) {
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
			name:            "Valid action with default settings",
			action:          lib.ActionOutput,
			data:            json.RawMessage(`{}`),
			expectType:      TypeTextOut,
			expectIPType:    "",
			expectOutputDir: defaultOutputDirForTextOut,
			expectErr:       false,
		},
		{
			name:            "Valid action with custom output dir",
			action:          lib.ActionOutput,
			data:            json.RawMessage(`{"outputDir": "/tmp/custom"}`),
			expectType:      TypeTextOut,
			expectIPType:    "",
			expectOutputDir: "/tmp/custom",
			expectErr:       false,
		},
		{
			name:            "Valid action with IPv4 only",
			action:          lib.ActionOutput,
			data:            json.RawMessage(`{"onlyIPType": "ipv4"}`),
			expectType:      TypeTextOut,
			expectIPType:    lib.IPv4,
			expectOutputDir: defaultOutputDirForTextOut,
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
			converter, err := newTextOut(TypeTextOut, DescTextOut, tt.action, tt.data)
			if (err != nil) != tt.expectErr {
				t.Errorf("newTextOut() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr {
				textOut := converter.(*TextOut)
				if textOut.GetType() != tt.expectType {
					t.Errorf("GetType() = %v, expect %v", textOut.GetType(), tt.expectType)
				}
				if textOut.GetAction() != tt.action {
					t.Errorf("GetAction() = %v, expect %v", textOut.GetAction(), tt.action)
				}
				if textOut.OnlyIPType != tt.expectIPType {
					t.Errorf("OnlyIPType = %v, expect %v", textOut.OnlyIPType, tt.expectIPType)
				}
				if textOut.OutputDir != tt.expectOutputDir {
					t.Errorf("OutputDir = %v, expect %v", textOut.OutputDir, tt.expectOutputDir)
				}
			}
		})
	}
}

func TestTextOut_GetType(t *testing.T) {
	textOut := &TextOut{Type: TypeTextOut}
	result := textOut.GetType()
	if result != TypeTextOut {
		t.Errorf("GetType() = %v, expect %v", result, TypeTextOut)
	}
}

func TestTextOut_GetAction(t *testing.T) {
	action := lib.ActionOutput
	textOut := &TextOut{Action: action}
	result := textOut.GetAction()
	if result != action {
		t.Errorf("GetAction() = %v, expect %v", result, action)
	}
}

func TestTextOut_GetDescription(t *testing.T) {
	textOut := &TextOut{Description: DescTextOut}
	result := textOut.GetDescription()
	if result != DescTextOut {
		t.Errorf("GetDescription() = %v, expect %v", result, DescTextOut)
	}
}

func TestTextOut_Output(t *testing.T) {
	// Create temporary output directory
	tmpDir, err := os.MkdirTemp("", "test-text-output")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	textOut := &TextOut{
		Type:      TypeTextOut,
		Action:    lib.ActionOutput,
		OutputDir: tmpDir,
		OutputExt: ".txt",
	}

	// Create a container with test entries
	container := lib.NewContainer()

	entry1 := lib.NewEntry("TEST1")
	if err := entry1.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("Failed to add prefix to entry1: %v", err)
	}
	if err := container.Add(entry1); err != nil {
		t.Fatalf("Failed to add entry1: %v", err)
	}

	err = textOut.Output(container)
	if err != nil {
		t.Errorf("Output() error = %v", err)
		return
	}

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
}

func TestTextOut_Constants(t *testing.T) {
	if TypeTextOut != "text" {
		t.Errorf("TypeTextOut = %v, expect %v", TypeTextOut, "text")
	}
	if DescTextOut != "Convert data to plaintext CIDR format" {
		t.Errorf("DescTextOut = %v, expect correct description", DescTextOut)
	}
}