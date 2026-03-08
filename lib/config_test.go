package lib

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// Mock implementations for testing
type mockInputConfigCreator struct {
	shouldError bool
	converter   InputConverter
}

func (m *mockInputConfigCreator) create(action Action, data json.RawMessage) (InputConverter, error) {
	if m.shouldError {
		return nil, errors.New("mock error")
	}
	return m.converter, nil
}

type mockOutputConfigCreator struct {
	shouldError bool
	converter   OutputConverter
}

func (m *mockOutputConfigCreator) create(action Action, data json.RawMessage) (OutputConverter, error) {
	if m.shouldError {
		return nil, errors.New("mock error")
	}
	return m.converter, nil
}

func TestRegisterInputConfigCreator(t *testing.T) {
	// Test successful registration
	err := RegisterInputConfigCreator("test-input-creator", func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{
			mockTyper:        &mockTyper{typeValue: "test"},
			mockActioner:     &mockActioner{actionValue: action},
			mockDescriptioner: &mockDescriptioner{descValue: "test"},
		}, nil
	})
	if err != nil {
		t.Errorf("RegisterInputConfigCreator() should not return error: %v", err)
	}

	// Test duplicate registration
	err = RegisterInputConfigCreator("test-input-creator", func(action Action, data json.RawMessage) (InputConverter, error) {
		return nil, nil
	})
	if err == nil {
		t.Error("RegisterInputConfigCreator() should return error for duplicate registration")
	}
	if !strings.Contains(err.Error(), "already been registered") {
		t.Errorf("Error should mention already registered, got: %v", err)
	}

	// Test case insensitive registration
	err = RegisterInputConfigCreator("TEST-INPUT-CREATOR", func(action Action, data json.RawMessage) (InputConverter, error) {
		return nil, nil
	})
	if err == nil {
		t.Error("RegisterInputConfigCreator() should return error for case-insensitive duplicate")
	}
}

func TestRegisterOutputConfigCreator(t *testing.T) {
	// Test successful registration
	err := RegisterOutputConfigCreator("test-output-creator", func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{
			mockTyper:        &mockTyper{typeValue: "test"},
			mockActioner:     &mockActioner{actionValue: action},
			mockDescriptioner: &mockDescriptioner{descValue: "test"},
		}, nil
	})
	if err != nil {
		t.Errorf("RegisterOutputConfigCreator() should not return error: %v", err)
	}

	// Test duplicate registration
	err = RegisterOutputConfigCreator("test-output-creator", func(action Action, data json.RawMessage) (OutputConverter, error) {
		return nil, nil
	})
	if err == nil {
		t.Error("RegisterOutputConfigCreator() should return error for duplicate registration")
	}
}

func TestCreateInputConfig(t *testing.T) {
	// Register a test creator first
	RegisterInputConfigCreator("test-create-input", func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{
			mockTyper:        &mockTyper{typeValue: "test-create-input"},
			mockActioner:     &mockActioner{actionValue: action},
			mockDescriptioner: &mockDescriptioner{descValue: "test"},
		}, nil
	})

	// Test successful creation
	converter, err := createInputConfig("test-create-input", ActionAdd, json.RawMessage(`{}`))
	if err != nil {
		t.Errorf("createInputConfig() should not return error: %v", err)
	}
	if converter == nil {
		t.Error("createInputConfig() should return non-nil converter")
	}
	if converter.GetType() != "test-create-input" {
		t.Errorf("Converter type = %s; want test-create-input", converter.GetType())
	}
	if converter.GetAction() != ActionAdd {
		t.Errorf("Converter action = %s; want %s", converter.GetAction(), ActionAdd)
	}

	// Test unknown config type
	_, err = createInputConfig("unknown-type", ActionAdd, json.RawMessage(`{}`))
	if err == nil {
		t.Error("createInputConfig() should return error for unknown type")
	}
	if !strings.Contains(err.Error(), "unknown config type") {
		t.Errorf("Error should mention unknown config type, got: %v", err)
	}

	// Test case insensitive lookup
	converter, err = createInputConfig("TEST-CREATE-INPUT", ActionRemove, json.RawMessage(`{}`))
	if err != nil {
		t.Errorf("createInputConfig() should work case insensitively: %v", err)
	}
	if converter.GetAction() != ActionRemove {
		t.Errorf("Converter action = %s; want %s", converter.GetAction(), ActionRemove)
	}
}

func TestCreateOutputConfig(t *testing.T) {
	// Register a test creator first
	RegisterOutputConfigCreator("test-create-output", func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{
			mockTyper:        &mockTyper{typeValue: "test-create-output"},
			mockActioner:     &mockActioner{actionValue: action},
			mockDescriptioner: &mockDescriptioner{descValue: "test"},
		}, nil
	})

	// Test successful creation
	converter, err := createOutputConfig("test-create-output", ActionOutput, json.RawMessage(`{}`))
	if err != nil {
		t.Errorf("createOutputConfig() should not return error: %v", err)
	}
	if converter == nil {
		t.Error("createOutputConfig() should return non-nil converter")
	}
	if converter.GetType() != "test-create-output" {
		t.Errorf("Converter type = %s; want test-create-output", converter.GetType())
	}

	// Test unknown config type
	_, err = createOutputConfig("unknown-type", ActionOutput, json.RawMessage(`{}`))
	if err == nil {
		t.Error("createOutputConfig() should return error for unknown type")
	}
}

func TestInputConvConfig_UnmarshalJSON(t *testing.T) {
	// Register a test creator
	RegisterInputConfigCreator("test-unmarshal-input", func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{
			mockTyper:        &mockTyper{typeValue: "test-unmarshal-input"},
			mockActioner:     &mockActioner{actionValue: action},
			mockDescriptioner: &mockDescriptioner{descValue: "test"},
		}, nil
	})

	tests := []struct {
		name        string
		input       string
		expectError bool
		expectType  string
		expectAction Action
	}{
		{
			name:         "Valid config",
			input:        `{"type": "test-unmarshal-input", "action": "add", "args": {}}`,
			expectError:  false,
			expectType:   "test-unmarshal-input",
			expectAction: ActionAdd,
		},
		{
			name:        "Invalid action",
			input:       `{"type": "test-unmarshal-input", "action": "invalid", "args": {}}`,
			expectError: true,
		},
		{
			name:        "Missing type",
			input:       `{"action": "add", "args": {}}`,
			expectError: true,
		},
		{
			name:        "Unknown type",
			input:       `{"type": "unknown-type", "action": "add", "args": {}}`,
			expectError: true,
		},
		{
			name:        "Invalid JSON",
			input:       `{invalid json}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config inputConvConfig
			err := json.Unmarshal([]byte(tt.input), &config)

			if tt.expectError && err == nil {
				t.Errorf("UnmarshalJSON() should return error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("UnmarshalJSON() should not return error but got: %v", err)
			}

			if !tt.expectError {
				if config.iType != tt.expectType {
					t.Errorf("iType = %s; want %s", config.iType, tt.expectType)
				}
				if config.action != tt.expectAction {
					t.Errorf("action = %s; want %s", config.action, tt.expectAction)
				}
				if config.converter == nil {
					t.Error("converter should not be nil")
				}
			}
		})
	}
}

func TestOutputConvConfig_UnmarshalJSON(t *testing.T) {
	// Register a test creator
	RegisterOutputConfigCreator("test-unmarshal-output", func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{
			mockTyper:        &mockTyper{typeValue: "test-unmarshal-output"},
			mockActioner:     &mockActioner{actionValue: action},
			mockDescriptioner: &mockDescriptioner{descValue: "test"},
		}, nil
	})

	tests := []struct {
		name         string
		input        string
		expectError  bool
		expectType   string
		expectAction Action
	}{
		{
			name:         "Valid config",
			input:        `{"type": "test-unmarshal-output", "action": "output", "args": {}}`,
			expectError:  false,
			expectType:   "test-unmarshal-output",
			expectAction: ActionOutput,
		},
		{
			name:         "Missing action defaults to output",
			input:        `{"type": "test-unmarshal-output", "args": {}}`,
			expectError:  false,
			expectType:   "test-unmarshal-output",
			expectAction: ActionOutput,
		},
		{
			name:        "Invalid action",
			input:       `{"type": "test-unmarshal-output", "action": "invalid", "args": {}}`,
			expectError: true,
		},
		{
			name:        "Unknown type",
			input:       `{"type": "unknown-type", "action": "output", "args": {}}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config outputConvConfig
			err := json.Unmarshal([]byte(tt.input), &config)

			if tt.expectError && err == nil {
				t.Errorf("UnmarshalJSON() should return error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("UnmarshalJSON() should not return error but got: %v", err)
			}

			if !tt.expectError {
				if config.iType != tt.expectType {
					t.Errorf("iType = %s; want %s", config.iType, tt.expectType)
				}
				if config.action != tt.expectAction {
					t.Errorf("action = %s; want %s", config.action, tt.expectAction)
				}
				if config.converter == nil {
					t.Error("converter should not be nil")
				}
			}
		})
	}
}

func TestConfigStruct_UnmarshalJSON(t *testing.T) {
	// Register test creators
	RegisterInputConfigCreator("test-config-input", func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{
			mockTyper:        &mockTyper{typeValue: "test-config-input"},
			mockActioner:     &mockActioner{actionValue: action},
			mockDescriptioner: &mockDescriptioner{descValue: "test input"},
		}, nil
	})
	RegisterOutputConfigCreator("test-config-output", func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{
			mockTyper:        &mockTyper{typeValue: "test-config-output"},
			mockActioner:     &mockActioner{actionValue: action},
			mockDescriptioner: &mockDescriptioner{descValue: "test output"},
		}, nil
	})

	configJSON := `{
		"input": [
			{"type": "test-config-input", "action": "add", "args": {}}
		],
		"output": [
			{"type": "test-config-output", "action": "output", "args": {}}
		]
	}`

	var cfg config
	err := json.Unmarshal([]byte(configJSON), &cfg)
	if err != nil {
		t.Errorf("config UnmarshalJSON() should not return error: %v", err)
	}

	if len(cfg.Input) != 1 {
		t.Errorf("config should have 1 input, got %d", len(cfg.Input))
	}
	if len(cfg.Output) != 1 {
		t.Errorf("config should have 1 output, got %d", len(cfg.Output))
	}

	if cfg.Input[0].iType != "test-config-input" {
		t.Errorf("input type = %s; want test-config-input", cfg.Input[0].iType)
	}
	if cfg.Output[0].iType != "test-config-output" {
		t.Errorf("output type = %s; want test-config-output", cfg.Output[0].iType)
	}
}