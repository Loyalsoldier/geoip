package v2ray

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestGeoIPDatOut_NewGeoIPDatOut(t *testing.T) {
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
			expectType:      TypeGeoIPDatOut,
			expectIPType:    "",
			expectOutputDir: defaultOutputDir,
			expectErr:       false,
		},
		{
			name:            "Valid action with custom output dir",
			action:          lib.ActionOutput,
			data:            json.RawMessage(`{"outputDir": "/tmp/custom"}`),
			expectType:      TypeGeoIPDatOut,
			expectIPType:    "",
			expectOutputDir: "/tmp/custom",
			expectErr:       false,
		},
		{
			name:            "Valid action with IPv4 only",
			action:          lib.ActionOutput,
			data:            json.RawMessage(`{"onlyIPType": "ipv4"}`),
			expectType:      TypeGeoIPDatOut,
			expectIPType:    lib.IPv4,
			expectOutputDir: defaultOutputDir,
			expectErr:       false,
		},
		{
			name:            "Valid action with one file per list",
			action:          lib.ActionOutput,
			data:            json.RawMessage(`{"oneFilePerList": true}`),
			expectType:      TypeGeoIPDatOut,
			expectIPType:    "",
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
			converter, err := newGeoIPDatOut(tt.action, tt.data)
			if (err != nil) != tt.expectErr {
				t.Errorf("newGeoIPDatOut() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr {
				datOut := converter.(*GeoIPDatOut)
				if datOut.GetType() != tt.expectType {
					t.Errorf("GetType() = %v, expect %v", datOut.GetType(), tt.expectType)
				}
				if datOut.GetAction() != tt.action {
					t.Errorf("GetAction() = %v, expect %v", datOut.GetAction(), tt.action)
				}
				if datOut.OnlyIPType != tt.expectIPType {
					t.Errorf("OnlyIPType = %v, expect %v", datOut.OnlyIPType, tt.expectIPType)
				}
				if datOut.OutputDir != tt.expectOutputDir {
					t.Errorf("OutputDir = %v, expect %v", datOut.OutputDir, tt.expectOutputDir)
				}
			}
		})
	}
}

func TestGeoIPDatOut_GetType(t *testing.T) {
	datOut := &GeoIPDatOut{Type: TypeGeoIPDatOut}
	result := datOut.GetType()
	if result != TypeGeoIPDatOut {
		t.Errorf("GetType() = %v, expect %v", result, TypeGeoIPDatOut)
	}
}

func TestGeoIPDatOut_GetAction(t *testing.T) {
	action := lib.ActionOutput
	datOut := &GeoIPDatOut{Action: action}
	result := datOut.GetAction()
	if result != action {
		t.Errorf("GetAction() = %v, expect %v", result, action)
	}
}

func TestGeoIPDatOut_GetDescription(t *testing.T) {
	datOut := &GeoIPDatOut{Description: DescGeoIPDatOut}
	result := datOut.GetDescription()
	if result != DescGeoIPDatOut {
		t.Errorf("GetDescription() = %v, expect %v", result, DescGeoIPDatOut)
	}
}

func TestGeoIPDatOut_Output(t *testing.T) {
	// Create temporary output directory
	tmpDir, err := os.MkdirTemp("", "test-dat-output")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name      string
		datOut    *GeoIPDatOut
		expectErr bool
	}{
		{
			name: "Output all entries",
			datOut: &GeoIPDatOut{
				Type:       TypeGeoIPDatOut,
				Action:     lib.ActionOutput,
				OutputDir:  tmpDir,
				OutputName: "geoip.dat",
			},
			expectErr: false,
		},
		{
			name: "Output with wanted list",
			datOut: &GeoIPDatOut{
				Type:       TypeGeoIPDatOut,
				Action:     lib.ActionOutput,
				OutputDir:  tmpDir,
				OutputName: "geoip.dat",
				Want:       []string{"TEST1"},
			},
			expectErr: false,
		},
		{
			name: "Output with one file per list",
			datOut: &GeoIPDatOut{
				Type:           TypeGeoIPDatOut,
				Action:         lib.ActionOutput,
				OutputDir:      tmpDir,
				OutputName:     "geoip.dat",
				OneFilePerList: true,
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

			err := tt.datOut.Output(container)

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

				// Check for .dat files
				foundDat := false
				for _, file := range files {
					if filepath.Ext(file.Name()) == ".dat" {
						foundDat = true
						break
					}
				}
				if !foundDat {
					t.Error("No .dat files were created")
				}
			}
		})
	}
}

func TestGeoIPDatOut_FilterAndSortList(t *testing.T) {
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
		datOut   *GeoIPDatOut
		expected []string
	}{
		{
			name: "No filters",
			datOut: &GeoIPDatOut{
				Want:    nil,
				Exclude: nil,
			},
			expected: []string{"EXCLUDE", "TEST1", "TEST2"}, // Sorted
		},
		{
			name: "With wanted list",
			datOut: &GeoIPDatOut{
				Want:    []string{"TEST1", "TEST2"},
				Exclude: nil,
			},
			expected: []string{"TEST1", "TEST2"},
		},
		{
			name: "With excluded list",
			datOut: &GeoIPDatOut{
				Want:    nil,
				Exclude: []string{"EXCLUDE"},
			},
			expected: []string{"TEST1", "TEST2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.datOut.filterAndSortList(container)
			if len(result) != len(tt.expected) {
				t.Errorf("filterAndSortList() length = %v, expect %v", len(result), len(tt.expected))
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

func TestGeoIPDatOut_GenerateGeoIP(t *testing.T) {
	datOut := &GeoIPDatOut{
		Type:   TypeGeoIPDatOut,
		Action: lib.ActionOutput,
	}

	tests := []struct {
		name      string
		entry     *lib.Entry
		expectErr bool
	}{
		{
			name: "Valid entry with prefixes",
			entry: func() *lib.Entry {
				entry := lib.NewEntry("TEST")
				entry.AddPrefix("192.168.1.0/24")
				return entry
			}(),
			expectErr: false,
		},
		{
			name:      "Empty entry",
			entry:     lib.NewEntry("EMPTY"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := datOut.generateGeoIP(tt.entry)
			if (err != nil) != tt.expectErr {
				t.Errorf("generateGeoIP() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr && result == nil {
				t.Error("generateGeoIP() returned nil result")
			}
		})
	}
}

func TestGeoIPDatOut_Constants(t *testing.T) {
	if TypeGeoIPDatOut != "v2rayGeoIPDat" {
		t.Errorf("TypeGeoIPDatOut = %v, expect %v", TypeGeoIPDatOut, "v2rayGeoIPDat")
	}
	if DescGeoIPDatOut != "Convert data to V2Ray GeoIP dat format" {
		t.Errorf("DescGeoIPDatOut = %v, expect correct description", DescGeoIPDatOut)
	}
	expectedDefaultDir := filepath.Join("./", "output", "dat")
	if defaultOutputDir != expectedDefaultDir {
		t.Errorf("defaultOutputDir = %v, expect %v", defaultOutputDir, expectedDefaultDir)
	}
}