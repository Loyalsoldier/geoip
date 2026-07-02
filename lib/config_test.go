package lib

import (
	"encoding/json"
	"testing"
)

// mockInputConverter implements InputConverter for testing
type mockInputConverter struct {
	typeName    string
	action      Action
	description string
	inputFn     func(Container) (Container, error)
}

func (m *mockInputConverter) GetType() string           { return m.typeName }
func (m *mockInputConverter) GetAction() Action          { return m.action }
func (m *mockInputConverter) GetDescription() string     { return m.description }
func (m *mockInputConverter) Input(c Container) (Container, error) {
	if m.inputFn != nil {
		return m.inputFn(c)
	}
	return c, nil
}

// mockOutputConverter implements OutputConverter for testing
type mockOutputConverter struct {
	typeName    string
	action      Action
	description string
	outputFn    func(Container) error
}

func (m *mockOutputConverter) GetType() string           { return m.typeName }
func (m *mockOutputConverter) GetAction() Action          { return m.action }
func (m *mockOutputConverter) GetDescription() string     { return m.description }
func (m *mockOutputConverter) Output(c Container) error {
	if m.outputFn != nil {
		return m.outputFn(c)
	}
	return nil
}

func TestRegisterInputConfigCreator(t *testing.T) {
	// Save and restore original cache
	origCache := inputConfigCreatorCache
	inputConfigCreatorCache = make(map[string]inputConfigCreator)
	defer func() { inputConfigCreatorCache = origCache }()

	creator := func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typeName: "test-input", action: action, description: "test"}, nil
	}

	// Register successfully
	if err := RegisterInputConfigCreator("test-input", creator); err != nil {
		t.Errorf("RegisterInputConfigCreator error = %v", err)
	}

	// Duplicate registration
	if err := RegisterInputConfigCreator("test-input", creator); err == nil {
		t.Error("expected error for duplicate registration")
	}

	// Case insensitive
	if err := RegisterInputConfigCreator("TEST-INPUT", creator); err == nil {
		t.Error("expected error for case-insensitive duplicate")
	}
}

func TestCreateInputConfig(t *testing.T) {
	origCache := inputConfigCreatorCache
	inputConfigCreatorCache = make(map[string]inputConfigCreator)
	defer func() { inputConfigCreatorCache = origCache }()

	creator := func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typeName: "test-input", action: action, description: "test"}, nil
	}
	RegisterInputConfigCreator("test-input", creator)

	// Successful creation
	ic, err := createInputConfig("test-input", ActionAdd, nil)
	if err != nil {
		t.Errorf("createInputConfig error = %v", err)
	}
	if ic.GetType() != "test-input" {
		t.Errorf("expected type 'test-input', got %q", ic.GetType())
	}

	// Unknown type
	_, err = createInputConfig("unknown", ActionAdd, nil)
	if err == nil {
		t.Error("expected error for unknown config type")
	}

	// Case insensitive
	ic, err = createInputConfig("TEST-INPUT", ActionAdd, nil)
	if err != nil {
		t.Errorf("createInputConfig case insensitive error = %v", err)
	}
	if ic.GetType() != "test-input" {
		t.Errorf("expected type 'test-input', got %q", ic.GetType())
	}
}

func TestRegisterOutputConfigCreator(t *testing.T) {
	origCache := outputConfigCreatorCache
	outputConfigCreatorCache = make(map[string]outputConfigCreator)
	defer func() { outputConfigCreatorCache = origCache }()

	creator := func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typeName: "test-output", action: action, description: "test"}, nil
	}

	// Register successfully
	if err := RegisterOutputConfigCreator("test-output", creator); err != nil {
		t.Errorf("RegisterOutputConfigCreator error = %v", err)
	}

	// Duplicate registration
	if err := RegisterOutputConfigCreator("test-output", creator); err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestCreateOutputConfig(t *testing.T) {
	origCache := outputConfigCreatorCache
	outputConfigCreatorCache = make(map[string]outputConfigCreator)
	defer func() { outputConfigCreatorCache = origCache }()

	creator := func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typeName: "test-output", action: action, description: "test"}, nil
	}
	RegisterOutputConfigCreator("test-output", creator)

	// Successful creation
	oc, err := createOutputConfig("test-output", ActionOutput, nil)
	if err != nil {
		t.Errorf("createOutputConfig error = %v", err)
	}
	if oc.GetType() != "test-output" {
		t.Errorf("expected type 'test-output', got %q", oc.GetType())
	}

	// Unknown type
	_, err = createOutputConfig("unknown", ActionOutput, nil)
	if err == nil {
		t.Error("expected error for unknown config type")
	}
}

func TestInputConvConfigUnmarshalJSON(t *testing.T) {
	origCache := inputConfigCreatorCache
	inputConfigCreatorCache = make(map[string]inputConfigCreator)
	defer func() { inputConfigCreatorCache = origCache }()

	creator := func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typeName: "test-input", action: action, description: "test"}, nil
	}
	RegisterInputConfigCreator("test-input", creator)

	// Valid JSON
	data := []byte(`{"type":"test-input","action":"add","args":{}}`)
	icc := &inputConvConfig{}
	if err := icc.UnmarshalJSON(data); err != nil {
		t.Errorf("UnmarshalJSON error = %v", err)
	}
	if icc.iType != "test-input" {
		t.Errorf("expected type 'test-input', got %q", icc.iType)
	}
	if icc.action != ActionAdd {
		t.Errorf("expected action 'add', got %q", icc.action)
	}

	// Invalid action
	data2 := []byte(`{"type":"test-input","action":"invalid","args":{}}`)
	icc2 := &inputConvConfig{}
	if err := icc2.UnmarshalJSON(data2); err == nil {
		t.Error("expected error for invalid action")
	}

	// Invalid JSON
	icc3 := &inputConvConfig{}
	if err := icc3.UnmarshalJSON([]byte(`{invalid}`)); err == nil {
		t.Error("expected error for invalid JSON")
	}

	// Unknown type
	data4 := []byte(`{"type":"unknown","action":"add","args":{}}`)
	icc4 := &inputConvConfig{}
	if err := icc4.UnmarshalJSON(data4); err == nil {
		t.Error("expected error for unknown type")
	}
}

func TestOutputConvConfigUnmarshalJSON(t *testing.T) {
	origCache := outputConfigCreatorCache
	outputConfigCreatorCache = make(map[string]outputConfigCreator)
	defer func() { outputConfigCreatorCache = origCache }()

	creator := func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typeName: "test-output", action: action, description: "test"}, nil
	}
	RegisterOutputConfigCreator("test-output", creator)

	// Valid JSON
	data := []byte(`{"type":"test-output","action":"output","args":{}}`)
	occ := &outputConvConfig{}
	if err := occ.UnmarshalJSON(data); err != nil {
		t.Errorf("UnmarshalJSON error = %v", err)
	}
	if occ.iType != "test-output" {
		t.Errorf("expected type 'test-output', got %q", occ.iType)
	}
	if occ.action != ActionOutput {
		t.Errorf("expected action 'output', got %q", occ.action)
	}

	// Default action (empty action defaults to "output")
	data2 := []byte(`{"type":"test-output","args":{}}`)
	occ2 := &outputConvConfig{}
	if err := occ2.UnmarshalJSON(data2); err != nil {
		t.Errorf("UnmarshalJSON default action error = %v", err)
	}
	if occ2.action != ActionOutput {
		t.Errorf("expected default action 'output', got %q", occ2.action)
	}

	// Invalid action
	data3 := []byte(`{"type":"test-output","action":"invalid","args":{}}`)
	occ3 := &outputConvConfig{}
	if err := occ3.UnmarshalJSON(data3); err == nil {
		t.Error("expected error for invalid action")
	}

	// Invalid JSON
	occ4 := &outputConvConfig{}
	if err := occ4.UnmarshalJSON([]byte(`{invalid}`)); err == nil {
		t.Error("expected error for invalid JSON")
	}

	// Unknown type
	data5 := []byte(`{"type":"unknown","action":"output","args":{}}`)
	occ5 := &outputConvConfig{}
	if err := occ5.UnmarshalJSON(data5); err == nil {
		t.Error("expected error for unknown type")
	}
}
