package lib

import (
	"encoding/json"
	"testing"
)

func TestRegisterInputConfigCreator(t *testing.T) {
	// Test registering a new input config creator
	testID := "test_input_config_creator_" + t.Name()
	fn := func(action Action, data json.RawMessage) (InputConverter, error) {
		return nil, nil
	}

	err := RegisterInputConfigCreator(testID, fn)
	if err != nil {
		t.Fatalf("RegisterInputConfigCreator failed: %v", err)
	}

	// Test registering duplicate
	err = RegisterInputConfigCreator(testID, fn)
	if err == nil {
		t.Error("RegisterInputConfigCreator should return error for duplicate")
	}
}

func TestRegisterInputConfigCreator_CaseInsensitive(t *testing.T) {
	testID := "TEST_INPUT_CONFIG_CASE_" + t.Name()
	fn := func(action Action, data json.RawMessage) (InputConverter, error) {
		return nil, nil
	}

	err := RegisterInputConfigCreator(testID, fn)
	if err != nil {
		t.Fatalf("RegisterInputConfigCreator failed: %v", err)
	}

	// Try to register with lowercase
	err = RegisterInputConfigCreator("test_input_config_case_"+t.Name(), fn)
	if err == nil {
		t.Error("RegisterInputConfigCreator should be case-insensitive")
	}
}

func TestCreateInputConfig_NotFound(t *testing.T) {
	_, err := createInputConfig("nonexistent_input_config", ActionAdd, nil)
	if err == nil {
		t.Error("createInputConfig should return error for unknown type")
	}
}

func TestRegisterOutputConfigCreator(t *testing.T) {
	// Test registering a new output config creator
	testID := "test_output_config_creator_" + t.Name()
	fn := func(action Action, data json.RawMessage) (OutputConverter, error) {
		return nil, nil
	}

	err := RegisterOutputConfigCreator(testID, fn)
	if err != nil {
		t.Fatalf("RegisterOutputConfigCreator failed: %v", err)
	}

	// Test registering duplicate
	err = RegisterOutputConfigCreator(testID, fn)
	if err == nil {
		t.Error("RegisterOutputConfigCreator should return error for duplicate")
	}
}

func TestRegisterOutputConfigCreator_CaseInsensitive(t *testing.T) {
	testID := "TEST_OUTPUT_CONFIG_CASE_" + t.Name()
	fn := func(action Action, data json.RawMessage) (OutputConverter, error) {
		return nil, nil
	}

	err := RegisterOutputConfigCreator(testID, fn)
	if err != nil {
		t.Fatalf("RegisterOutputConfigCreator failed: %v", err)
	}

	// Try to register with lowercase
	err = RegisterOutputConfigCreator("test_output_config_case_"+t.Name(), fn)
	if err == nil {
		t.Error("RegisterOutputConfigCreator should be case-insensitive")
	}
}

func TestCreateOutputConfig_NotFound(t *testing.T) {
	_, err := createOutputConfig("nonexistent_output_config", ActionOutput, nil)
	if err == nil {
		t.Error("createOutputConfig should return error for unknown type")
	}
}

// MockInputConverter for testing
type mockInputConverter struct {
	typeName    string
	action      Action
	description string
}

func (m *mockInputConverter) GetType() string        { return m.typeName }
func (m *mockInputConverter) GetAction() Action      { return m.action }
func (m *mockInputConverter) GetDescription() string { return m.description }
func (m *mockInputConverter) Input(c Container) (Container, error) {
	return c, nil
}

// MockOutputConverter for testing
type mockOutputConverter struct {
	typeName    string
	action      Action
	description string
}

func (m *mockOutputConverter) GetType() string        { return m.typeName }
func (m *mockOutputConverter) GetAction() Action      { return m.action }
func (m *mockOutputConverter) GetDescription() string { return m.description }
func (m *mockOutputConverter) Output(c Container) error {
	return nil
}

func TestInputConvConfigUnmarshalJSON(t *testing.T) {
	// Register a mock input config creator
	testType := "mock_input_" + t.Name()
	RegisterInputConfigCreator(testType, func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{
			typeName: testType,
			action:   action,
		}, nil
	})

	// Test valid unmarshal
	jsonData := []byte(`{"type":"` + testType + `","action":"add","args":{}}`)
	var config inputConvConfig
	err := json.Unmarshal(jsonData, &config)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if config.iType != testType {
		t.Errorf("config.iType = %s, want %s", config.iType, testType)
	}
	if config.action != ActionAdd {
		t.Errorf("config.action = %s, want %s", config.action, ActionAdd)
	}
}

func TestInputConvConfigUnmarshalJSON_InvalidAction(t *testing.T) {
	jsonData := []byte(`{"type":"sometype","action":"invalid_action","args":{}}`)
	var config inputConvConfig
	err := json.Unmarshal(jsonData, &config)
	if err == nil {
		t.Error("UnmarshalJSON should fail for invalid action")
	}
}

func TestInputConvConfigUnmarshalJSON_InvalidJSON(t *testing.T) {
	jsonData := []byte(`{invalid json}`)
	var config inputConvConfig
	err := json.Unmarshal(jsonData, &config)
	if err == nil {
		t.Error("UnmarshalJSON should fail for invalid JSON")
	}
}

func TestInputConvConfigUnmarshalJSON_UnknownType(t *testing.T) {
	jsonData := []byte(`{"type":"unknown_type_123","action":"add","args":{}}`)
	var config inputConvConfig
	err := json.Unmarshal(jsonData, &config)
	if err == nil {
		t.Error("UnmarshalJSON should fail for unknown type")
	}
}

func TestOutputConvConfigUnmarshalJSON(t *testing.T) {
	// Register a mock output config creator
	testType := "mock_output_" + t.Name()
	RegisterOutputConfigCreator(testType, func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{
			typeName: testType,
			action:   action,
		}, nil
	})

	// Test valid unmarshal
	jsonData := []byte(`{"type":"` + testType + `","action":"output","args":{}}`)
	var config outputConvConfig
	err := json.Unmarshal(jsonData, &config)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if config.iType != testType {
		t.Errorf("config.iType = %s, want %s", config.iType, testType)
	}
	if config.action != ActionOutput {
		t.Errorf("config.action = %s, want %s", config.action, ActionOutput)
	}
}

func TestOutputConvConfigUnmarshalJSON_DefaultAction(t *testing.T) {
	// Register a mock output config creator
	testType := "mock_output_default_" + t.Name()
	RegisterOutputConfigCreator(testType, func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{
			typeName: testType,
			action:   action,
		}, nil
	})

	// Test unmarshal without action (should default to "output")
	jsonData := []byte(`{"type":"` + testType + `","args":{}}`)
	var config outputConvConfig
	err := json.Unmarshal(jsonData, &config)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if config.action != ActionOutput {
		t.Errorf("config.action = %s, want %s (default)", config.action, ActionOutput)
	}
}

func TestOutputConvConfigUnmarshalJSON_InvalidAction(t *testing.T) {
	jsonData := []byte(`{"type":"sometype","action":"invalid_action","args":{}}`)
	var config outputConvConfig
	err := json.Unmarshal(jsonData, &config)
	if err == nil {
		t.Error("UnmarshalJSON should fail for invalid action")
	}
}

func TestOutputConvConfigUnmarshalJSON_InvalidJSON(t *testing.T) {
	jsonData := []byte(`{invalid json}`)
	var config outputConvConfig
	err := json.Unmarshal(jsonData, &config)
	if err == nil {
		t.Error("UnmarshalJSON should fail for invalid JSON")
	}
}

func TestOutputConvConfigUnmarshalJSON_UnknownType(t *testing.T) {
	jsonData := []byte(`{"type":"unknown_type_456","action":"output","args":{}}`)
	var config outputConvConfig
	err := json.Unmarshal(jsonData, &config)
	if err == nil {
		t.Error("UnmarshalJSON should fail for unknown type")
	}
}

func TestConfigStruct(t *testing.T) {
	// Register mock converters for this test
	inputType := "config_test_input_" + t.Name()
	outputType := "config_test_output_" + t.Name()

	RegisterInputConfigCreator(inputType, func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{
			typeName: inputType,
			action:   action,
		}, nil
	})

	RegisterOutputConfigCreator(outputType, func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{
			typeName: outputType,
			action:   action,
		}, nil
	})

	// Test unmarshaling full config
	jsonData := []byte(`{
		"input": [
			{"type":"` + inputType + `","action":"add","args":{}}
		],
		"output": [
			{"type":"` + outputType + `","action":"output","args":{}}
		]
	}`)

	var cfg config
	err := json.Unmarshal(jsonData, &cfg)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if len(cfg.Input) != 1 {
		t.Errorf("len(cfg.Input) = %d, want 1", len(cfg.Input))
	}
	if len(cfg.Output) != 1 {
		t.Errorf("len(cfg.Output) = %d, want 1", len(cfg.Output))
	}
}
