package lib

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestRegisterInputConfigCreator(t *testing.T) {
	// Store original state
	original := make(map[string]inputConfigCreator)
	for k, v := range inputConfigCreatorCache {
		original[k] = v
	}
	defer func() {
		inputConfigCreatorCache = original
	}()

	// Clear for testing
	inputConfigCreatorCache = make(map[string]inputConfigCreator)

	t.Run("success", func(t *testing.T) {
		fn := func(action Action, data json.RawMessage) (InputConverter, error) {
			return &mockInputConverter{iType: "test-config", action: action}, nil
		}
		err := RegisterInputConfigCreator("test-config", fn)
		if err != nil {
			t.Fatalf("RegisterInputConfigCreator failed: %v", err)
		}
		if inputConfigCreatorCache["test-config"] == nil {
			t.Error("config creator not registered")
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		fn := func(action Action, data json.RawMessage) (InputConverter, error) {
			return nil, nil
		}
		err := RegisterInputConfigCreator("dup-config", fn)
		if err != nil {
			t.Fatalf("first registration failed: %v", err)
		}
		err = RegisterInputConfigCreator("dup-config", fn)
		if err == nil || err.Error() != "config creator has already been registered" {
			t.Errorf("expected duplicate error, got %v", err)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		fn := func(action Action, data json.RawMessage) (InputConverter, error) {
			return nil, nil
		}
		err := RegisterInputConfigCreator("CaseTest", fn)
		if err != nil {
			t.Fatalf("RegisterInputConfigCreator failed: %v", err)
		}
		// Should be stored as lowercase
		if inputConfigCreatorCache["casetest"] == nil {
			t.Error("config creator not stored as lowercase")
		}
	})
}

func TestCreateInputConfig(t *testing.T) {
	// Store original state
	original := make(map[string]inputConfigCreator)
	for k, v := range inputConfigCreatorCache {
		original[k] = v
	}
	defer func() {
		inputConfigCreatorCache = original
	}()

	// Clear for testing
	inputConfigCreatorCache = make(map[string]inputConfigCreator)

	t.Run("success", func(t *testing.T) {
		inputConfigCreatorCache["mytype"] = func(action Action, data json.RawMessage) (InputConverter, error) {
			return &mockInputConverter{iType: "mytype", action: action}, nil
		}
		conv, err := createInputConfig("mytype", ActionAdd, nil)
		if err != nil {
			t.Fatalf("createInputConfig failed: %v", err)
		}
		if conv.GetType() != "mytype" {
			t.Errorf("expected type 'mytype', got %q", conv.GetType())
		}
		if conv.GetAction() != ActionAdd {
			t.Errorf("expected action 'add', got %q", conv.GetAction())
		}
	})

	t.Run("unknown type", func(t *testing.T) {
		_, err := createInputConfig("unknown", ActionAdd, nil)
		if err == nil || err.Error() != "unknown config type" {
			t.Errorf("expected unknown config type error, got %v", err)
		}
	})

	t.Run("case insensitive lookup", func(t *testing.T) {
		inputConfigCreatorCache["lowercase"] = func(action Action, data json.RawMessage) (InputConverter, error) {
			return &mockInputConverter{iType: "lowercase"}, nil
		}
		conv, err := createInputConfig("LOWERCASE", ActionAdd, nil)
		if err != nil {
			t.Fatalf("createInputConfig failed: %v", err)
		}
		if conv.GetType() != "lowercase" {
			t.Errorf("expected type 'lowercase', got %q", conv.GetType())
		}
	})

	t.Run("creator returns error", func(t *testing.T) {
		inputConfigCreatorCache["errortype"] = func(action Action, data json.RawMessage) (InputConverter, error) {
			return nil, errors.New("creator error")
		}
		_, err := createInputConfig("errortype", ActionAdd, nil)
		if err == nil || err.Error() != "creator error" {
			t.Errorf("expected creator error, got %v", err)
		}
	})
}

func TestRegisterOutputConfigCreator(t *testing.T) {
	// Store original state
	original := make(map[string]outputConfigCreator)
	for k, v := range outputConfigCreatorCache {
		original[k] = v
	}
	defer func() {
		outputConfigCreatorCache = original
	}()

	// Clear for testing
	outputConfigCreatorCache = make(map[string]outputConfigCreator)

	t.Run("success", func(t *testing.T) {
		fn := func(action Action, data json.RawMessage) (OutputConverter, error) {
			return &mockOutputConverter{iType: "test-output-config", action: action}, nil
		}
		err := RegisterOutputConfigCreator("test-output-config", fn)
		if err != nil {
			t.Fatalf("RegisterOutputConfigCreator failed: %v", err)
		}
		if outputConfigCreatorCache["test-output-config"] == nil {
			t.Error("config creator not registered")
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		fn := func(action Action, data json.RawMessage) (OutputConverter, error) {
			return nil, nil
		}
		err := RegisterOutputConfigCreator("dup-output-config", fn)
		if err != nil {
			t.Fatalf("first registration failed: %v", err)
		}
		err = RegisterOutputConfigCreator("dup-output-config", fn)
		if err == nil || err.Error() != "config creator has already been registered" {
			t.Errorf("expected duplicate error, got %v", err)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		fn := func(action Action, data json.RawMessage) (OutputConverter, error) {
			return nil, nil
		}
		err := RegisterOutputConfigCreator("OutputCaseTest", fn)
		if err != nil {
			t.Fatalf("RegisterOutputConfigCreator failed: %v", err)
		}
		// Should be stored as lowercase
		if outputConfigCreatorCache["outputcasetest"] == nil {
			t.Error("config creator not stored as lowercase")
		}
	})
}

func TestCreateOutputConfig(t *testing.T) {
	// Store original state
	original := make(map[string]outputConfigCreator)
	for k, v := range outputConfigCreatorCache {
		original[k] = v
	}
	defer func() {
		outputConfigCreatorCache = original
	}()

	// Clear for testing
	outputConfigCreatorCache = make(map[string]outputConfigCreator)

	t.Run("success", func(t *testing.T) {
		outputConfigCreatorCache["myoutputtype"] = func(action Action, data json.RawMessage) (OutputConverter, error) {
			return &mockOutputConverter{iType: "myoutputtype", action: action}, nil
		}
		conv, err := createOutputConfig("myoutputtype", ActionOutput, nil)
		if err != nil {
			t.Fatalf("createOutputConfig failed: %v", err)
		}
		if conv.GetType() != "myoutputtype" {
			t.Errorf("expected type 'myoutputtype', got %q", conv.GetType())
		}
		if conv.GetAction() != ActionOutput {
			t.Errorf("expected action 'output', got %q", conv.GetAction())
		}
	})

	t.Run("unknown type", func(t *testing.T) {
		_, err := createOutputConfig("unknown-output", ActionOutput, nil)
		if err == nil || err.Error() != "unknown config type" {
			t.Errorf("expected unknown config type error, got %v", err)
		}
	})

	t.Run("case insensitive lookup", func(t *testing.T) {
		outputConfigCreatorCache["lowercaseoutput"] = func(action Action, data json.RawMessage) (OutputConverter, error) {
			return &mockOutputConverter{iType: "lowercaseoutput"}, nil
		}
		conv, err := createOutputConfig("LOWERCASEOUTPUT", ActionOutput, nil)
		if err != nil {
			t.Fatalf("createOutputConfig failed: %v", err)
		}
		if conv.GetType() != "lowercaseoutput" {
			t.Errorf("expected type 'lowercaseoutput', got %q", conv.GetType())
		}
	})

	t.Run("creator returns error", func(t *testing.T) {
		outputConfigCreatorCache["erroroutputtype"] = func(action Action, data json.RawMessage) (OutputConverter, error) {
			return nil, errors.New("output creator error")
		}
		_, err := createOutputConfig("erroroutputtype", ActionOutput, nil)
		if err == nil || err.Error() != "output creator error" {
			t.Errorf("expected output creator error, got %v", err)
		}
	})
}

func TestInputConvConfig_UnmarshalJSON(t *testing.T) {
	// Store original state
	original := make(map[string]inputConfigCreator)
	for k, v := range inputConfigCreatorCache {
		original[k] = v
	}
	defer func() {
		inputConfigCreatorCache = original
	}()

	// Clear for testing
	inputConfigCreatorCache = make(map[string]inputConfigCreator)

	// Register a test creator
	inputConfigCreatorCache["testinput"] = func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{iType: "testinput", action: action, description: "Test"}, nil
	}

	t.Run("success", func(t *testing.T) {
		data := []byte(`{"type": "testinput", "action": "add", "args": {}}`)
		var cfg inputConvConfig
		err := json.Unmarshal(data, &cfg)
		if err != nil {
			t.Fatalf("UnmarshalJSON failed: %v", err)
		}
		if cfg.iType != "testinput" {
			t.Errorf("expected type 'testinput', got %q", cfg.iType)
		}
		if cfg.action != ActionAdd {
			t.Errorf("expected action 'add', got %q", cfg.action)
		}
		if cfg.converter == nil {
			t.Error("expected converter to be set")
		}
	})

	t.Run("invalid action", func(t *testing.T) {
		data := []byte(`{"type": "testinput", "action": "invalid", "args": {}}`)
		var cfg inputConvConfig
		err := json.Unmarshal(data, &cfg)
		if err == nil {
			t.Error("expected error for invalid action")
		}
	})

	t.Run("unknown type", func(t *testing.T) {
		data := []byte(`{"type": "unknowntype", "action": "add", "args": {}}`)
		var cfg inputConvConfig
		err := json.Unmarshal(data, &cfg)
		if err == nil {
			t.Error("expected error for unknown type")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		// Invalid JSON that passes the outer parse but fails inner parse
		data := []byte(`{invalid}`)
		var cfg inputConvConfig
		err := json.Unmarshal(data, &cfg)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("invalid nested JSON", func(t *testing.T) {
		// JSON that is valid at the outer level but has invalid type value
		data := []byte(`{"type": 123, "action": "add", "args": {}}`)
		var cfg inputConvConfig
		err := json.Unmarshal(data, &cfg)
		if err == nil {
			t.Error("expected error for invalid type field")
		}
	})

	t.Run("with remove action", func(t *testing.T) {
		data := []byte(`{"type": "testinput", "action": "remove", "args": {}}`)
		var cfg inputConvConfig
		err := json.Unmarshal(data, &cfg)
		if err != nil {
			t.Fatalf("UnmarshalJSON failed: %v", err)
		}
		if cfg.action != ActionRemove {
			t.Errorf("expected action 'remove', got %q", cfg.action)
		}
	})
}

func TestOutputConvConfig_UnmarshalJSON(t *testing.T) {
	// Store original state
	original := make(map[string]outputConfigCreator)
	for k, v := range outputConfigCreatorCache {
		original[k] = v
	}
	defer func() {
		outputConfigCreatorCache = original
	}()

	// Clear for testing
	outputConfigCreatorCache = make(map[string]outputConfigCreator)

	// Register a test creator
	outputConfigCreatorCache["testoutput"] = func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{iType: "testoutput", action: action, description: "Test"}, nil
	}

	t.Run("success", func(t *testing.T) {
		data := []byte(`{"type": "testoutput", "action": "output", "args": {}}`)
		var cfg outputConvConfig
		err := json.Unmarshal(data, &cfg)
		if err != nil {
			t.Fatalf("UnmarshalJSON failed: %v", err)
		}
		if cfg.iType != "testoutput" {
			t.Errorf("expected type 'testoutput', got %q", cfg.iType)
		}
		if cfg.action != ActionOutput {
			t.Errorf("expected action 'output', got %q", cfg.action)
		}
		if cfg.converter == nil {
			t.Error("expected converter to be set")
		}
	})

	t.Run("empty action defaults to output", func(t *testing.T) {
		data := []byte(`{"type": "testoutput", "args": {}}`)
		var cfg outputConvConfig
		err := json.Unmarshal(data, &cfg)
		if err != nil {
			t.Fatalf("UnmarshalJSON failed: %v", err)
		}
		if cfg.action != ActionOutput {
			t.Errorf("expected default action 'output', got %q", cfg.action)
		}
	})

	t.Run("invalid action", func(t *testing.T) {
		data := []byte(`{"type": "testoutput", "action": "invalid", "args": {}}`)
		var cfg outputConvConfig
		err := json.Unmarshal(data, &cfg)
		if err == nil {
			t.Error("expected error for invalid action")
		}
	})

	t.Run("unknown type", func(t *testing.T) {
		data := []byte(`{"type": "unknownoutputtype", "action": "output", "args": {}}`)
		var cfg outputConvConfig
		err := json.Unmarshal(data, &cfg)
		if err == nil {
			t.Error("expected error for unknown type")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		data := []byte(`{invalid}`)
		var cfg outputConvConfig
		err := json.Unmarshal(data, &cfg)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("invalid nested JSON", func(t *testing.T) {
		// JSON that has invalid type value (number instead of string)
		data := []byte(`{"type": 123, "action": "output", "args": {}}`)
		var cfg outputConvConfig
		err := json.Unmarshal(data, &cfg)
		if err == nil {
			t.Error("expected error for invalid type field")
		}
	})

	t.Run("with add action", func(t *testing.T) {
		data := []byte(`{"type": "testoutput", "action": "add", "args": {}}`)
		var cfg outputConvConfig
		err := json.Unmarshal(data, &cfg)
		if err != nil {
			t.Fatalf("UnmarshalJSON failed: %v", err)
		}
		if cfg.action != ActionAdd {
			t.Errorf("expected action 'add', got %q", cfg.action)
		}
	})
}

func TestConfig_FullUnmarshal(t *testing.T) {
	// Store original state
	originalInput := make(map[string]inputConfigCreator)
	for k, v := range inputConfigCreatorCache {
		originalInput[k] = v
	}
	originalOutput := make(map[string]outputConfigCreator)
	for k, v := range outputConfigCreatorCache {
		originalOutput[k] = v
	}
	defer func() {
		inputConfigCreatorCache = originalInput
		outputConfigCreatorCache = originalOutput
	}()

	// Clear for testing
	inputConfigCreatorCache = make(map[string]inputConfigCreator)
	outputConfigCreatorCache = make(map[string]outputConfigCreator)

	// Register test creators
	inputConfigCreatorCache["testinput"] = func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{iType: "testinput", action: action}, nil
	}
	outputConfigCreatorCache["testoutput"] = func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{iType: "testoutput", action: action}, nil
	}

	t.Run("full config", func(t *testing.T) {
		data := []byte(`{
			"input": [
				{"type": "testinput", "action": "add", "args": {}}
			],
			"output": [
				{"type": "testoutput", "action": "output", "args": {}}
			]
		}`)
		var cfg config
		err := json.Unmarshal(data, &cfg)
		if err != nil {
			t.Fatalf("UnmarshalJSON failed: %v", err)
		}
		if len(cfg.Input) != 1 {
			t.Errorf("expected 1 input, got %d", len(cfg.Input))
		}
		if len(cfg.Output) != 1 {
			t.Errorf("expected 1 output, got %d", len(cfg.Output))
		}
	})
}
