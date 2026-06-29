package lib

import (
	"encoding/json"
	"testing"
)

func TestRegisterInputConfigCreator(t *testing.T) {
	// Clear cache
	inputConfigCreatorCache = make(map[string]inputConfigCreator)

	creator := func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typ: "test", action: action, description: "test"}, nil
	}

	err := RegisterInputConfigCreator("test", creator)
	if err != nil {
		t.Errorf("RegisterInputConfigCreator() error = %v, want nil", err)
	}

	// Verify creator was registered
	if _, ok := inputConfigCreatorCache["test"]; !ok {
		t.Error("Creator not found in inputConfigCreatorCache")
	}
}

func TestRegisterInputConfigCreator_Duplicate(t *testing.T) {
	// Clear cache
	inputConfigCreatorCache = make(map[string]inputConfigCreator)

	creator := func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typ: "test", action: action, description: "test"}, nil
	}

	// Register first time
	err := RegisterInputConfigCreator("test", creator)
	if err != nil {
		t.Errorf("RegisterInputConfigCreator() first call error = %v, want nil", err)
	}

	// Register duplicate
	err = RegisterInputConfigCreator("test", creator)
	if err == nil {
		t.Error("RegisterInputConfigCreator() duplicate expected error, got nil")
	}
	if err.Error() != "config creator has already been registered" {
		t.Errorf("RegisterInputConfigCreator() duplicate error = %v, want 'config creator has already been registered'", err)
	}
}

func TestRegisterInputConfigCreator_CaseInsensitive(t *testing.T) {
	// Clear cache
	inputConfigCreatorCache = make(map[string]inputConfigCreator)

	creator := func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typ: "test", action: action, description: "test"}, nil
	}

	// Register with uppercase
	err := RegisterInputConfigCreator("TEST", creator)
	if err != nil {
		t.Errorf("RegisterInputConfigCreator() error = %v, want nil", err)
	}

	// Verify lowercase key is used
	if _, ok := inputConfigCreatorCache["test"]; !ok {
		t.Error("Creator not found with lowercase key")
	}
}

func TestRegisterOutputConfigCreator(t *testing.T) {
	// Clear cache
	outputConfigCreatorCache = make(map[string]outputConfigCreator)

	creator := func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typ: "test", action: action, description: "test"}, nil
	}

	err := RegisterOutputConfigCreator("test", creator)
	if err != nil {
		t.Errorf("RegisterOutputConfigCreator() error = %v, want nil", err)
	}

	// Verify creator was registered
	if _, ok := outputConfigCreatorCache["test"]; !ok {
		t.Error("Creator not found in outputConfigCreatorCache")
	}
}

func TestRegisterOutputConfigCreator_Duplicate(t *testing.T) {
	// Clear cache
	outputConfigCreatorCache = make(map[string]outputConfigCreator)

	creator := func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typ: "test", action: action, description: "test"}, nil
	}

	// Register first time
	err := RegisterOutputConfigCreator("test", creator)
	if err != nil {
		t.Errorf("RegisterOutputConfigCreator() first call error = %v, want nil", err)
	}

	// Register duplicate
	err = RegisterOutputConfigCreator("test", creator)
	if err == nil {
		t.Error("RegisterOutputConfigCreator() duplicate expected error, got nil")
	}
	if err.Error() != "config creator has already been registered" {
		t.Errorf("RegisterOutputConfigCreator() duplicate error = %v, want 'config creator has already been registered'", err)
	}
}

func TestRegisterOutputConfigCreator_CaseInsensitive(t *testing.T) {
	// Clear cache
	outputConfigCreatorCache = make(map[string]outputConfigCreator)

	creator := func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typ: "test", action: action, description: "test"}, nil
	}

	// Register with uppercase
	err := RegisterOutputConfigCreator("TEST", creator)
	if err != nil {
		t.Errorf("RegisterOutputConfigCreator() error = %v, want nil", err)
	}

	// Verify lowercase key is used
	if _, ok := outputConfigCreatorCache["test"]; !ok {
		t.Error("Creator not found with lowercase key")
	}
}

func TestCreateInputConfig(t *testing.T) {
	// Clear cache and register creator
	inputConfigCreatorCache = make(map[string]inputConfigCreator)

	creator := func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typ: "test", action: action, description: "test"}, nil
	}

	RegisterInputConfigCreator("test", creator)

	// Create config
	conv, err := createInputConfig("test", ActionAdd, json.RawMessage(`{}`))
	if err != nil {
		t.Errorf("createInputConfig() error = %v, want nil", err)
	}
	if conv == nil {
		t.Error("createInputConfig() returned nil converter")
	}
	if conv.GetType() != "test" {
		t.Errorf("createInputConfig() type = %q, want %q", conv.GetType(), "test")
	}
}

func TestCreateInputConfig_CaseInsensitive(t *testing.T) {
	// Clear cache and register creator
	inputConfigCreatorCache = make(map[string]inputConfigCreator)

	creator := func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typ: "test", action: action, description: "test"}, nil
	}

	RegisterInputConfigCreator("test", creator)

	// Create config with uppercase
	conv, err := createInputConfig("TEST", ActionAdd, json.RawMessage(`{}`))
	if err != nil {
		t.Errorf("createInputConfig() error = %v, want nil", err)
	}
	if conv == nil {
		t.Error("createInputConfig() returned nil converter")
	}
}

func TestCreateInputConfig_NotFound(t *testing.T) {
	// Clear cache
	inputConfigCreatorCache = make(map[string]inputConfigCreator)

	// Try to create non-existent config
	_, err := createInputConfig("notfound", ActionAdd, json.RawMessage(`{}`))
	if err == nil {
		t.Error("createInputConfig() with unknown type expected error, got nil")
	}
	if err.Error() != "unknown config type" {
		t.Errorf("createInputConfig() error = %v, want 'unknown config type'", err)
	}
}

func TestCreateOutputConfig(t *testing.T) {
	// Clear cache and register creator
	outputConfigCreatorCache = make(map[string]outputConfigCreator)

	creator := func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typ: "test", action: action, description: "test"}, nil
	}

	RegisterOutputConfigCreator("test", creator)

	// Create config
	conv, err := createOutputConfig("test", ActionOutput, json.RawMessage(`{}`))
	if err != nil {
		t.Errorf("createOutputConfig() error = %v, want nil", err)
	}
	if conv == nil {
		t.Error("createOutputConfig() returned nil converter")
	}
	if conv.GetType() != "test" {
		t.Errorf("createOutputConfig() type = %q, want %q", conv.GetType(), "test")
	}
}

func TestCreateOutputConfig_CaseInsensitive(t *testing.T) {
	// Clear cache and register creator
	outputConfigCreatorCache = make(map[string]outputConfigCreator)

	creator := func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typ: "test", action: action, description: "test"}, nil
	}

	RegisterOutputConfigCreator("test", creator)

	// Create config with uppercase
	conv, err := createOutputConfig("TEST", ActionOutput, json.RawMessage(`{}`))
	if err != nil {
		t.Errorf("createOutputConfig() error = %v, want nil", err)
	}
	if conv == nil {
		t.Error("createOutputConfig() returned nil converter")
	}
}

func TestCreateOutputConfig_NotFound(t *testing.T) {
	// Clear cache
	outputConfigCreatorCache = make(map[string]outputConfigCreator)

	// Try to create non-existent config
	_, err := createOutputConfig("notfound", ActionOutput, json.RawMessage(`{}`))
	if err == nil {
		t.Error("createOutputConfig() with unknown type expected error, got nil")
	}
	if err.Error() != "unknown config type" {
		t.Errorf("createOutputConfig() error = %v, want 'unknown config type'", err)
	}
}

func TestInputConvConfig_UnmarshalJSON(t *testing.T) {
	// Clear cache and register creator
	inputConfigCreatorCache = make(map[string]inputConfigCreator)

	creator := func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typ: "test", action: action, description: "test"}, nil
	}

	RegisterInputConfigCreator("test", creator)

	// Test valid JSON
	jsonData := `{"type": "test", "action": "add", "args": {}}`
	var config inputConvConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		t.Errorf("inputConvConfig.UnmarshalJSON() error = %v, want nil", err)
	}
	if config.iType != "test" {
		t.Errorf("inputConvConfig.iType = %q, want %q", config.iType, "test")
	}
	if config.action != ActionAdd {
		t.Errorf("inputConvConfig.action = %q, want %q", config.action, ActionAdd)
	}
}

func TestInputConvConfig_UnmarshalJSON_InvalidAction(t *testing.T) {
	// Clear cache and register creator
	inputConfigCreatorCache = make(map[string]inputConfigCreator)

	creator := func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typ: "test", action: action, description: "test"}, nil
	}

	RegisterInputConfigCreator("test", creator)

	// Test invalid action
	jsonData := `{"type": "test", "action": "invalid", "args": {}}`
	var config inputConvConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err == nil {
		t.Error("inputConvConfig.UnmarshalJSON() with invalid action expected error, got nil")
	}
}

func TestInputConvConfig_UnmarshalJSON_InvalidJSON(t *testing.T) {
	jsonData := `{invalid json}`
	var config inputConvConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err == nil {
		t.Error("inputConvConfig.UnmarshalJSON() with invalid JSON expected error, got nil")
	}
}

func TestInputConvConfig_UnmarshalJSON_UnknownType(t *testing.T) {
	// Clear cache
	inputConfigCreatorCache = make(map[string]inputConfigCreator)

	jsonData := `{"type": "unknown", "action": "add", "args": {}}`
	var config inputConvConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err == nil {
		t.Error("inputConvConfig.UnmarshalJSON() with unknown type expected error, got nil")
	}
}

func TestOutputConvConfig_UnmarshalJSON(t *testing.T) {
	// Clear cache and register creator
	outputConfigCreatorCache = make(map[string]outputConfigCreator)

	creator := func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typ: "test", action: action, description: "test"}, nil
	}

	RegisterOutputConfigCreator("test", creator)

	// Test valid JSON
	jsonData := `{"type": "test", "action": "output", "args": {}}`
	var config outputConvConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		t.Errorf("outputConvConfig.UnmarshalJSON() error = %v, want nil", err)
	}
	if config.iType != "test" {
		t.Errorf("outputConvConfig.iType = %q, want %q", config.iType, "test")
	}
	if config.action != ActionOutput {
		t.Errorf("outputConvConfig.action = %q, want %q", config.action, ActionOutput)
	}
}

func TestOutputConvConfig_UnmarshalJSON_DefaultAction(t *testing.T) {
	// Clear cache and register creator
	outputConfigCreatorCache = make(map[string]outputConfigCreator)

	creator := func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typ: "test", action: action, description: "test"}, nil
	}

	RegisterOutputConfigCreator("test", creator)

	// Test without action (should default to "output")
	jsonData := `{"type": "test", "args": {}}`
	var config outputConvConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		t.Errorf("outputConvConfig.UnmarshalJSON() error = %v, want nil", err)
	}
	if config.action != ActionOutput {
		t.Errorf("outputConvConfig.action = %q, want %q (default)", config.action, ActionOutput)
	}
}

func TestOutputConvConfig_UnmarshalJSON_InvalidAction(t *testing.T) {
	// Clear cache and register creator
	outputConfigCreatorCache = make(map[string]outputConfigCreator)

	creator := func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typ: "test", action: action, description: "test"}, nil
	}

	RegisterOutputConfigCreator("test", creator)

	// Test invalid action
	jsonData := `{"type": "test", "action": "invalid", "args": {}}`
	var config outputConvConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err == nil {
		t.Error("outputConvConfig.UnmarshalJSON() with invalid action expected error, got nil")
	}
}

func TestOutputConvConfig_UnmarshalJSON_InvalidJSON(t *testing.T) {
	jsonData := `{invalid json}`
	var config outputConvConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err == nil {
		t.Error("outputConvConfig.UnmarshalJSON() with invalid JSON expected error, got nil")
	}
}

func TestOutputConvConfig_UnmarshalJSON_UnknownType(t *testing.T) {
	// Clear cache
	outputConfigCreatorCache = make(map[string]outputConfigCreator)

	jsonData := `{"type": "unknown", "action": "output", "args": {}}`
	var config outputConvConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err == nil {
		t.Error("outputConvConfig.UnmarshalJSON() with unknown type expected error, got nil")
	}
}

func TestConfig_UnmarshalJSON(t *testing.T) {
	// Clear caches and register creators
	inputConfigCreatorCache = make(map[string]inputConfigCreator)
	outputConfigCreatorCache = make(map[string]outputConfigCreator)

	inputCreator := func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typ: "testin", action: action, description: "test input"}, nil
	}
	outputCreator := func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typ: "testout", action: action, description: "test output"}, nil
	}

	RegisterInputConfigCreator("testin", inputCreator)
	RegisterOutputConfigCreator("testout", outputCreator)

	// Test full config
	jsonData := `{
		"input": [
			{"type": "testin", "action": "add", "args": {}}
		],
		"output": [
			{"type": "testout", "action": "output", "args": {}}
		]
	}`

	var cfg config
	err := json.Unmarshal([]byte(jsonData), &cfg)
	if err != nil {
		t.Errorf("config.UnmarshalJSON() error = %v, want nil", err)
	}

	if len(cfg.Input) != 1 {
		t.Errorf("config.Input length = %d, want 1", len(cfg.Input))
	}
	if len(cfg.Output) != 1 {
		t.Errorf("config.Output length = %d, want 1", len(cfg.Output))
	}
}
