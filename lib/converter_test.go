package lib

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestListInputConverter(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ListInputConverter()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "All available input formats:") {
		t.Error("ListInputConverter() should print header")
	}
}

func TestListOutputConverter(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ListOutputConverter()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "All available output formats:") {
		t.Error("ListOutputConverter() should print header")
	}
}

func TestRegisterInputConverter(t *testing.T) {
	// Create a mock input converter
	mockConverter := &mockInputConverter{
		mockTyper:        &mockTyper{typeValue: "test-input"},
		mockActioner:     &mockActioner{actionValue: ActionAdd},
		mockDescriptioner: &mockDescriptioner{descValue: "test input converter"},
	}

	// Test successful registration
	err := RegisterInputConverter("test-input", mockConverter)
	if err != nil {
		t.Errorf("RegisterInputConverter() should not return error: %v", err)
	}

	// Test duplicate registration
	err = RegisterInputConverter("test-input", mockConverter)
	if err != ErrDuplicatedConverter {
		t.Errorf("RegisterInputConverter() should return ErrDuplicatedConverter for duplicate, got: %v", err)
	}

	// Test registration with leading/trailing spaces
	err = RegisterInputConverter("  test-input2  ", mockConverter)
	if err != nil {
		t.Errorf("RegisterInputConverter() should handle names with spaces: %v", err)
	}
}

func TestRegisterOutputConverter(t *testing.T) {
	// Create a mock output converter
	mockConverter := &mockOutputConverter{
		mockTyper:        &mockTyper{typeValue: "test-output"},
		mockActioner:     &mockActioner{actionValue: ActionOutput},
		mockDescriptioner: &mockDescriptioner{descValue: "test output converter"},
	}

	// Test successful registration
	err := RegisterOutputConverter("test-output", mockConverter)
	if err != nil {
		t.Errorf("RegisterOutputConverter() should not return error: %v", err)
	}

	// Test duplicate registration
	err = RegisterOutputConverter("test-output", mockConverter)
	if err != ErrDuplicatedConverter {
		t.Errorf("RegisterOutputConverter() should return ErrDuplicatedConverter for duplicate, got: %v", err)
	}

	// Test registration with leading/trailing spaces
	err = RegisterOutputConverter("  test-output2  ", mockConverter)
	if err != nil {
		t.Errorf("RegisterOutputConverter() should handle names with spaces: %v", err)
	}
}

func TestConverterRegistryIntegration(t *testing.T) {
	// Create mock converters
	inputConverter := &mockInputConverter{
		mockTyper:        &mockTyper{typeValue: "integration-input"},
		mockActioner:     &mockActioner{actionValue: ActionAdd},
		mockDescriptioner: &mockDescriptioner{descValue: "integration test input"},
	}

	outputConverter := &mockOutputConverter{
		mockTyper:        &mockTyper{typeValue: "integration-output"},
		mockActioner:     &mockActioner{actionValue: ActionOutput},
		mockDescriptioner: &mockDescriptioner{descValue: "integration test output"},
	}

	// Register converters
	err := RegisterInputConverter("integration-input", inputConverter)
	if err != nil {
		t.Fatalf("Failed to register input converter: %v", err)
	}

	err = RegisterOutputConverter("integration-output", outputConverter)
	if err != nil {
		t.Fatalf("Failed to register output converter: %v", err)
	}

	// Test that they appear in the list
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ListInputConverter()
	ListOutputConverter()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "integration-input") {
		t.Error("Registered input converter should appear in list")
	}
	if !strings.Contains(output, "integration-output") {
		t.Error("Registered output converter should appear in list")
	}
	if !strings.Contains(output, "integration test input") {
		t.Error("Input converter description should appear in list")
	}
	if !strings.Contains(output, "integration test output") {
		t.Error("Output converter description should appear in list")
	}
}

func TestConverterNaming(t *testing.T) {
	tests := []struct {
		name         string
		inputName    string
		expectedName string
		expectError  bool
	}{
		{
			name:         "Normal name",
			inputName:    "test-normal",
			expectedName: "test-normal",
			expectError:  false,
		},
		{
			name:         "Name with spaces",
			inputName:    "  test-spaces  ",
			expectedName: "test-spaces",
			expectError:  false,
		},
		{
			name:         "Empty name",
			inputName:    "",
			expectedName: "",
			expectError:  false,
		},
		{
			name:         "Name with tabs",
			inputName:    "\ttest-tabs\t",
			expectedName: "test-tabs",
			expectError:  false,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use unique names to avoid conflicts
			uniqueName := fmt.Sprintf("test-naming-%d", i)
			mockConverter := &mockInputConverter{
				mockTyper:        &mockTyper{typeValue: uniqueName},
				mockActioner:     &mockActioner{actionValue: ActionAdd},
				mockDescriptioner: &mockDescriptioner{descValue: "test"},
			}

			// The converter name should be trimmed when registered
			err := RegisterInputConverter(tt.inputName+fmt.Sprintf("-%d", i), mockConverter)
			if tt.expectError && err == nil {
				t.Errorf("RegisterInputConverter() should return error but got nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("RegisterInputConverter() should not return error for name '%s': %v", tt.inputName, err)
			}
		})
	}
}