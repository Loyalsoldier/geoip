package lib

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewInstance(t *testing.T) {
	instance, err := NewInstance()
	if err != nil {
		t.Errorf("NewInstance() should not return error: %v", err)
	}
	if instance == nil {
		t.Error("NewInstance() should return non-nil instance")
	}
}

func TestInstance_AddInput(t *testing.T) {
	instance, err := NewInstance()
	if err != nil {
		t.Fatalf("NewInstance() failed: %v", err)
	}

	mockConverter := &mockInputConverter{
		mockTyper:        &mockTyper{typeValue: "test"},
		mockActioner:     &mockActioner{actionValue: ActionAdd},
		mockDescriptioner: &mockDescriptioner{descValue: "test"},
	}

	instance.AddInput(mockConverter)
	// No direct way to verify, but should not panic
}

func TestInstance_AddOutput(t *testing.T) {
	instance, err := NewInstance()
	if err != nil {
		t.Fatalf("NewInstance() failed: %v", err)
	}

	mockConverter := &mockOutputConverter{
		mockTyper:        &mockTyper{typeValue: "test"},
		mockActioner:     &mockActioner{actionValue: ActionOutput},
		mockDescriptioner: &mockDescriptioner{descValue: "test"},
	}

	instance.AddOutput(mockConverter)
	// No direct way to verify, but should not panic
}

func TestInstance_ResetInput(t *testing.T) {
	instance, err := NewInstance()
	if err != nil {
		t.Fatalf("NewInstance() failed: %v", err)
	}

	// Add some input converters
	mockConverter := &mockInputConverter{
		mockTyper:        &mockTyper{typeValue: "test"},
		mockActioner:     &mockActioner{actionValue: ActionAdd},
		mockDescriptioner: &mockDescriptioner{descValue: "test"},
	}
	instance.AddInput(mockConverter)
	instance.AddInput(mockConverter)

	// Reset should clear all inputs
	instance.ResetInput()
	// No direct way to verify, but should not panic
}

func TestInstance_ResetOutput(t *testing.T) {
	instance, err := NewInstance()
	if err != nil {
		t.Fatalf("NewInstance() failed: %v", err)
	}

	// Add some output converters
	mockConverter := &mockOutputConverter{
		mockTyper:        &mockTyper{typeValue: "test"},
		mockActioner:     &mockActioner{actionValue: ActionOutput},
		mockDescriptioner: &mockDescriptioner{descValue: "test"},
	}
	instance.AddOutput(mockConverter)
	instance.AddOutput(mockConverter)

	// Reset should clear all outputs
	instance.ResetOutput()
	// No direct way to verify, but should not panic
}

func TestInstance_RunInput(t *testing.T) {
	instance, err := NewInstance()
	if err != nil {
		t.Fatalf("NewInstance() failed: %v", err)
	}

	container := NewContainer()

	// Test with no input converters
	err = instance.RunInput(container)
	if err != nil {
		t.Errorf("RunInput() with no converters should not return error: %v", err)
	}

	// Add a mock input converter
	mockConverter := &mockInputConverter{
		mockTyper:        &mockTyper{typeValue: "test"},
		mockActioner:     &mockActioner{actionValue: ActionAdd},
		mockDescriptioner: &mockDescriptioner{descValue: "test"},
	}
	instance.AddInput(mockConverter)

	// Test with input converter
	err = instance.RunInput(container)
	if err != nil {
		t.Errorf("RunInput() should not return error: %v", err)
	}
}

func TestInstance_RunOutput(t *testing.T) {
	instance, err := NewInstance()
	if err != nil {
		t.Fatalf("NewInstance() failed: %v", err)
	}

	container := NewContainer()

	// Test with no output converters
	err = instance.RunOutput(container)
	if err != nil {
		t.Errorf("RunOutput() with no converters should not return error: %v", err)
	}

	// Add a mock output converter
	mockConverter := &mockOutputConverter{
		mockTyper:        &mockTyper{typeValue: "test"},
		mockActioner:     &mockActioner{actionValue: ActionOutput},
		mockDescriptioner: &mockDescriptioner{descValue: "test"},
	}
	instance.AddOutput(mockConverter)

	// Test with output converter
	err = instance.RunOutput(container)
	if err != nil {
		t.Errorf("RunOutput() should not return error: %v", err)
	}
}

func TestInstance_Run(t *testing.T) {
	instance, err := NewInstance()
	if err != nil {
		t.Fatalf("NewInstance() failed: %v", err)
	}

	// Test with no converters - should return error
	err = instance.Run()
	if err == nil {
		t.Error("Run() should return error when no input/output converters are specified")
	}
	if !strings.Contains(err.Error(), "input type and output type must be specified") {
		t.Errorf("Error should mention input/output types, got: %v", err)
	}

	// Add input but no output - should return error
	mockInputConverter := &mockInputConverter{
		mockTyper:        &mockTyper{typeValue: "test"},
		mockActioner:     &mockActioner{actionValue: ActionAdd},
		mockDescriptioner: &mockDescriptioner{descValue: "test"},
	}
	instance.AddInput(mockInputConverter)

	err = instance.Run()
	if err == nil {
		t.Error("Run() should return error when no output converters are specified")
	}

	// Add output converter
	mockOutputConverter := &mockOutputConverter{
		mockTyper:        &mockTyper{typeValue: "test"},
		mockActioner:     &mockActioner{actionValue: ActionOutput},
		mockDescriptioner: &mockDescriptioner{descValue: "test"},
	}
	instance.AddOutput(mockOutputConverter)

	// Now should work
	err = instance.Run()
	if err != nil {
		t.Errorf("Run() should not return error when both input and output are specified: %v", err)
	}
}

func TestInstance_InitConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := os.TempDir()
	configFile := filepath.Join(tempDir, "test_config.json")

	// Register test creators
	RegisterInputConfigCreator("test-file-input", func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{
			mockTyper:        &mockTyper{typeValue: "test-file-input"},
			mockActioner:     &mockActioner{actionValue: action},
			mockDescriptioner: &mockDescriptioner{descValue: "test"},
		}, nil
	})
	RegisterOutputConfigCreator("test-file-output", func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{
			mockTyper:        &mockTyper{typeValue: "test-file-output"},
			mockActioner:     &mockActioner{actionValue: action},
			mockDescriptioner: &mockDescriptioner{descValue: "test"},
		}, nil
	})

	configContent := `{
		"input": [
			{"type": "test-file-input", "action": "add", "args": {}}
		],
		"output": [
			{"type": "test-file-output", "action": "output", "args": {}}
		]
	}`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	defer os.Remove(configFile)

	instance, err := NewInstance()
	if err != nil {
		t.Fatalf("NewInstance() failed: %v", err)
	}

	// Test successful config loading
	err = instance.InitConfig(configFile)
	if err != nil {
		t.Errorf("InitConfig() should not return error: %v", err)
	}

	// Test with non-existent file
	err = instance.InitConfig("non-existent-file.json")
	if err == nil {
		t.Error("InitConfig() should return error for non-existent file")
	}

	// Test with invalid JSON file
	invalidConfigFile := filepath.Join(tempDir, "invalid_config.json")
	err = os.WriteFile(invalidConfigFile, []byte(`{invalid json}`), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid config file: %v", err)
	}
	defer os.Remove(invalidConfigFile)

	err = instance.InitConfig(invalidConfigFile)
	if err == nil {
		t.Error("InitConfig() should return error for invalid JSON")
	}
}

func TestInstance_InitConfigFromBytes(t *testing.T) {
	// Register test creators
	RegisterInputConfigCreator("test-bytes-input", func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{
			mockTyper:        &mockTyper{typeValue: "test-bytes-input"},
			mockActioner:     &mockActioner{actionValue: action},
			mockDescriptioner: &mockDescriptioner{descValue: "test"},
		}, nil
	})
	RegisterOutputConfigCreator("test-bytes-output", func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{
			mockTyper:        &mockTyper{typeValue: "test-bytes-output"},
			mockActioner:     &mockActioner{actionValue: action},
			mockDescriptioner: &mockDescriptioner{descValue: "test"},
		}, nil
	})

	instance, err := NewInstance()
	if err != nil {
		t.Fatalf("NewInstance() failed: %v", err)
	}

	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name: "Valid config",
			content: `{
				"input": [
					{"type": "test-bytes-input", "action": "add", "args": {}}
				],
				"output": [
					{"type": "test-bytes-output", "action": "output", "args": {}}
				]
			}`,
			expectError: false,
		},
		{
			name: "Valid config with comments (hujson)",
			content: `{
				// This is a comment
				"input": [
					{"type": "test-bytes-input", "action": "add", "args": {}}, // trailing comma
				],
				"output": [
					{"type": "test-bytes-output", "action": "output", "args": {}}
				]
			}`,
			expectError: false,
		},
		{
			name:        "Invalid JSON",
			content:     `{invalid json}`,
			expectError: true,
		},
		{
			name:        "Empty content",
			content:     ``,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := instance.InitConfigFromBytes([]byte(tt.content))
			if tt.expectError && err == nil {
				t.Errorf("InitConfigFromBytes() should return error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("InitConfigFromBytes() should not return error but got: %v", err)
			}
		})
	}
}

// Error-returning mock converters for testing error scenarios
type errorInputConverter struct {
	*mockInputConverter
}

func (e *errorInputConverter) Input(container Container) (Container, error) {
	return nil, errors.New("input error")
}

type errorOutputConverter struct {
	*mockOutputConverter
}

func (e *errorOutputConverter) Output(container Container) error {
	return errors.New("output error")
}

func TestInstance_RunInput_Error(t *testing.T) {
	instance, err := NewInstance()
	if err != nil {
		t.Fatalf("NewInstance() failed: %v", err)
	}

	container := NewContainer()

	// Add an error-returning input converter
	errorConverter := &errorInputConverter{
		mockInputConverter: &mockInputConverter{
			mockTyper:        &mockTyper{typeValue: "error"},
			mockActioner:     &mockActioner{actionValue: ActionAdd},
			mockDescriptioner: &mockDescriptioner{descValue: "error"},
		},
	}
	instance.AddInput(errorConverter)

	err = instance.RunInput(container)
	if err == nil {
		t.Error("RunInput() should return error when input converter fails")
	}
	if !strings.Contains(err.Error(), "input error") {
		t.Errorf("Error should mention input error, got: %v", err)
	}
}

func TestInstance_RunOutput_Error(t *testing.T) {
	instance, err := NewInstance()
	if err != nil {
		t.Fatalf("NewInstance() failed: %v", err)
	}

	container := NewContainer()

	// Add an error-returning output converter
	errorConverter := &errorOutputConverter{
		mockOutputConverter: &mockOutputConverter{
			mockTyper:        &mockTyper{typeValue: "error"},
			mockActioner:     &mockActioner{actionValue: ActionOutput},
			mockDescriptioner: &mockDescriptioner{descValue: "error"},
		},
	}
	instance.AddOutput(errorConverter)

	err = instance.RunOutput(container)
	if err == nil {
		t.Error("RunOutput() should return error when output converter fails")
	}
	if !strings.Contains(err.Error(), "output error") {
		t.Errorf("Error should mention output error, got: %v", err)
	}
}