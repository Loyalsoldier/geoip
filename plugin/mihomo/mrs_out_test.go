package mihomo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestMRSOut_NewMRSOut(t *testing.T) {
	tests := []struct {
		name           string
		action         lib.Action
		data           json.RawMessage
		expectType     string
		expectIPType   lib.IPType
		expectOutputDir string
		expectErr      bool
	}{
		{
			name:           "Valid action with default output dir",
			action:         lib.ActionOutput,
			data:           json.RawMessage(`{}`),
			expectType:     TypeMRSOut,
			expectIPType:   "",
			expectOutputDir: defaultOutputDir,
			expectErr:      false,
		},
		{
			name:           "Valid action with custom output dir",
			action:         lib.ActionOutput,
			data:           json.RawMessage(`{"outputDir": "/tmp/custom"}`),
			expectType:     TypeMRSOut,
			expectIPType:   "",
			expectOutputDir: "/tmp/custom",
			expectErr:      false,
		},
		{
			name:           "Valid action with IPv4 only",
			action:         lib.ActionOutput,
			data:           json.RawMessage(`{"onlyIPType": "ipv4"}`),
			expectType:     TypeMRSOut,
			expectIPType:   lib.IPv4,
			expectOutputDir: defaultOutputDir,
			expectErr:      false,
		},
		{
			name:           "Valid action with wanted list",
			action:         lib.ActionOutput,
			data:           json.RawMessage(`{"wantedList": ["test1", "test2"]}`),
			expectType:     TypeMRSOut,
			expectIPType:   "",
			expectOutputDir: defaultOutputDir,
			expectErr:      false,
		},
		{
			name:           "Valid action with excluded list",
			action:         lib.ActionOutput,
			data:           json.RawMessage(`{"excludedList": ["exclude1"]}`),
			expectType:     TypeMRSOut,
			expectIPType:   "",
			expectOutputDir: defaultOutputDir,
			expectErr:      false,
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
			converter, err := newMRSOut(tt.action, tt.data)
			if (err != nil) != tt.expectErr {
				t.Errorf("newMRSOut() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr {
				mrsOut := converter.(*MRSOut)
				if mrsOut.GetType() != tt.expectType {
					t.Errorf("GetType() = %v, expect %v", mrsOut.GetType(), tt.expectType)
				}
				if mrsOut.GetAction() != tt.action {
					t.Errorf("GetAction() = %v, expect %v", mrsOut.GetAction(), tt.action)
				}
				if mrsOut.OnlyIPType != tt.expectIPType {
					t.Errorf("OnlyIPType = %v, expect %v", mrsOut.OnlyIPType, tt.expectIPType)
				}
				if mrsOut.OutputDir != tt.expectOutputDir {
					t.Errorf("OutputDir = %v, expect %v", mrsOut.OutputDir, tt.expectOutputDir)
				}
			}
		})
	}
}

func TestMRSOut_GetType(t *testing.T) {
	mrsOut := &MRSOut{Type: TypeMRSOut}
	result := mrsOut.GetType()
	if result != TypeMRSOut {
		t.Errorf("GetType() = %v, expect %v", result, TypeMRSOut)
	}
}

func TestMRSOut_GetAction(t *testing.T) {
	action := lib.ActionOutput
	mrsOut := &MRSOut{Action: action}
	result := mrsOut.GetAction()
	if result != action {
		t.Errorf("GetAction() = %v, expect %v", result, action)
	}
}

func TestMRSOut_GetDescription(t *testing.T) {
	mrsOut := &MRSOut{Description: DescMRSOut}
	result := mrsOut.GetDescription()
	if result != DescMRSOut {
		t.Errorf("GetDescription() = %v, expect %v", result, DescMRSOut)
	}
}

func TestMRSOut_Output(t *testing.T) {
	// Create temporary output directory
	tmpDir, err := os.MkdirTemp("", "test-mrs-output")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name      string
		mrsOut    *MRSOut
		expectErr bool
	}{
		{
			name: "Output all entries",
			mrsOut: &MRSOut{
				Type:      TypeMRSOut,
				Action:    lib.ActionOutput,
				OutputDir: tmpDir,
			},
			expectErr: false,
		},
		{
			name: "Output with wanted list",
			mrsOut: &MRSOut{
				Type:      TypeMRSOut,
				Action:    lib.ActionOutput,
				OutputDir: tmpDir,
				Want:      []string{"TEST1"},
			},
			expectErr: false,
		},
		{
			name: "Output with excluded list",
			mrsOut: &MRSOut{
				Type:      TypeMRSOut,
				Action:    lib.ActionOutput,
				OutputDir: tmpDir,
				Exclude:   []string{"TEST2"},
			},
			expectErr: false,
		},
		{
			name: "Output with IPv4 only",
			mrsOut: &MRSOut{
				Type:       TypeMRSOut,
				Action:     lib.ActionOutput,
				OutputDir:  tmpDir,
				OnlyIPType: lib.IPv4,
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

			entry2 := lib.NewEntry("TEST2")
			if err := entry2.AddPrefix("192.168.2.0/24"); err != nil {
				t.Fatalf("Failed to add prefix to entry2: %v", err)
			}
			if err := container.Add(entry2); err != nil {
				t.Fatalf("Failed to add entry2: %v", err)
			}

			err := tt.mrsOut.Output(container)

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

				// Check for .mrs files
				foundMRS := false
				for _, file := range files {
					if filepath.Ext(file.Name()) == ".mrs" {
						foundMRS = true
						break
					}
				}
				if !foundMRS {
					t.Error("No .mrs files were created")
				}
			}
		})
	}
}

func TestMRSOut_FilterAndSortList(t *testing.T) {
	container := lib.NewContainer()

	// Add test entries
	entry1 := lib.NewEntry("TEST1")
	entry2 := lib.NewEntry("TEST2")
	entry3 := lib.NewEntry("EXCLUDE")

	if err := entry1.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("Failed to add prefix to entry1: %v", err)
	}
	if err := entry2.AddPrefix("192.168.2.0/24"); err != nil {
		t.Fatalf("Failed to add prefix to entry2: %v", err)
	}
	if err := entry3.AddPrefix("192.168.3.0/24"); err != nil {
		t.Fatalf("Failed to add prefix to entry3: %v", err)
	}

	if err := container.Add(entry1); err != nil {
		t.Fatalf("Failed to add entry1: %v", err)
	}
	if err := container.Add(entry2); err != nil {
		t.Fatalf("Failed to add entry2: %v", err)
	}
	if err := container.Add(entry3); err != nil {
		t.Fatalf("Failed to add entry3: %v", err)
	}

	tests := []struct {
		name     string
		mrsOut   *MRSOut
		expected []string
	}{
		{
			name: "No filters",
			mrsOut: &MRSOut{
				Want:    nil,
				Exclude: nil,
			},
			expected: []string{"EXCLUDE", "TEST1", "TEST2"}, // Sorted
		},
		{
			name: "With wanted list",
			mrsOut: &MRSOut{
				Want:    []string{"TEST1", "TEST2"},
				Exclude: nil,
			},
			expected: []string{"TEST1", "TEST2"},
		},
		{
			name: "With excluded list",
			mrsOut: &MRSOut{
				Want:    nil,
				Exclude: []string{"EXCLUDE"},
			},
			expected: []string{"TEST1", "TEST2"},
		},
		{
			name: "With both wanted and excluded",
			mrsOut: &MRSOut{
				Want:    []string{"TEST1", "TEST2", "EXCLUDE"},
				Exclude: []string{"EXCLUDE"},
			},
			expected: []string{"TEST1", "TEST2"},
		},
		{
			name: "Empty wanted list",
			mrsOut: &MRSOut{
				Want:    []string{},
				Exclude: []string{"TEST1"},
			},
			expected: []string{"EXCLUDE", "TEST2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.mrsOut.filterAndSortList(container)
			if len(result) != len(tt.expected) {
				t.Errorf("filterAndSortList() length = %v, expect %v", len(result), len(tt.expected))
				t.Errorf("Got: %v", result)
				t.Errorf("Expected: %v", tt.expected)
				return
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("filterAndSortList()[%d] = %v, expect %v", i, result[i], expected)
				}
			}
		})
	}
}

func TestMRSOut_Generate(t *testing.T) {
	// Create temporary output directory
	tmpDir, err := os.MkdirTemp("", "test-mrs-generate")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name       string
		onlyIPType lib.IPType
		prefix     string
		expectErr  bool
	}{
		{
			name:       "All IP types",
			onlyIPType: "",
			prefix:     "192.168.1.0/24",
			expectErr:  false,
		},
		{
			name:       "IPv4 only",
			onlyIPType: lib.IPv4,
			prefix:     "192.168.1.0/24",
			expectErr:  false,
		},
		{
			name:       "IPv6 only with IPv6 prefix",
			onlyIPType: lib.IPv6,
			prefix:     "2001:db8::/32",
			expectErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mrsOut := &MRSOut{
				Type:       TypeMRSOut,
				Action:     lib.ActionOutput,
				OutputDir:  tmpDir,
				OnlyIPType: tt.onlyIPType,
			}

			entry := lib.NewEntry("TEST")
			if err := entry.AddPrefix(tt.prefix); err != nil {
				t.Fatalf("Failed to add prefix: %v", err)
			}

			err := mrsOut.generate(entry)
			if (err != nil) != tt.expectErr {
				t.Errorf("generate() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			if !tt.expectErr {
				// Check if file was created
				expectedFile := filepath.Join(tmpDir, "test.mrs")
				if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
					t.Errorf("Expected file %s was not created", expectedFile)
				}
			}
		})
	}
}

func TestMRSOut_GenerateEmptyEntry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-mrs-generate-empty")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mrsOut := &MRSOut{
		Type:      TypeMRSOut,
		Action:    lib.ActionOutput,
		OutputDir: tmpDir,
	}

	entry := lib.NewEntry("EMPTY")

	err = mrsOut.generate(entry)
	if err == nil {
		t.Error("generate() should return error for empty entry")
	}
}

func TestMRSOut_WriteFileError(t *testing.T) {
	// Try to write to a non-existent/non-writable directory
	mrsOut := &MRSOut{
		Type:      TypeMRSOut,
		Action:    lib.ActionOutput,
		OutputDir: "/nonexistent/readonly/dir",
	}

	entry := lib.NewEntry("TEST")
	if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("Failed to add prefix: %v", err)
	}

	err := mrsOut.generate(entry)
	if err == nil {
		t.Error("generate() should return error for non-writable directory")
	}
}

func TestMRSOut_Constants(t *testing.T) {
	if TypeMRSOut != "mihomoMRS" {
		t.Errorf("TypeMRSOut = %v, expect %v", TypeMRSOut, "mihomoMRS")
	}
	if DescMRSOut != "Convert data to mihomo MRS format" {
		t.Errorf("DescMRSOut = %v, expect correct description", DescMRSOut)
	}
	expectedDefaultDir := filepath.Join("./", "output", "mrs")
	if defaultOutputDir != expectedDefaultDir {
		t.Errorf("defaultOutputDir = %v, expect %v", defaultOutputDir, expectedDefaultDir)
	}
}