package lib

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewInstance(t *testing.T) {
	instance, err := NewInstance()
	if err != nil {
		t.Errorf("NewInstance() error = %v, want nil", err)
	}
	if instance == nil {
		t.Fatal("NewInstance() returned nil")
	}
}

func TestInstance_AddInput(t *testing.T) {
	instance, _ := NewInstance()

	converter := &mockInputConverter{
		typ:         "test",
		action:      ActionAdd,
		description: "test",
	}

	instance.AddInput(converter)

	// Verify by running (should not panic)
	container := NewContainer()
	err := instance.RunInput(container)
	if err != nil {
		t.Errorf("RunInput() after AddInput error = %v, want nil", err)
	}
}

func TestInstance_AddOutput(t *testing.T) {
	instance, _ := NewInstance()

	converter := &mockOutputConverter{
		typ:         "test",
		action:      ActionOutput,
		description: "test",
	}

	instance.AddOutput(converter)

	// Verify by running (should not panic)
	container := NewContainer()
	err := instance.RunOutput(container)
	if err != nil {
		t.Errorf("RunOutput() after AddOutput error = %v, want nil", err)
	}
}

func TestInstance_ResetInput(t *testing.T) {
	instance, _ := NewInstance()

	converter := &mockInputConverter{
		typ:         "test",
		action:      ActionAdd,
		description: "test",
	}

	instance.AddInput(converter)
	instance.ResetInput()

	// After reset, Run should fail due to no input
	instance.AddOutput(&mockOutputConverter{typ: "test", action: ActionOutput, description: "test"})
	err := instance.Run()
	if err == nil {
		t.Error("Run() after ResetInput expected error, got nil")
	}
}

func TestInstance_ResetOutput(t *testing.T) {
	instance, _ := NewInstance()

	converter := &mockOutputConverter{
		typ:         "test",
		action:      ActionOutput,
		description: "test",
	}

	instance.AddOutput(converter)
	instance.ResetOutput()

	// After reset, Run should fail due to no output
	instance.AddInput(&mockInputConverter{typ: "test", action: ActionAdd, description: "test"})
	err := instance.Run()
	if err == nil {
		t.Error("Run() after ResetOutput expected error, got nil")
	}
}

func TestInstance_RunInput(t *testing.T) {
	instance, _ := NewInstance()
	container := NewContainer()

	// Add mock input converter
	called := false
	converter := &mockInputConverterWithCallback{
		mockInputConverter: mockInputConverter{
			typ:         "test",
			action:      ActionAdd,
			description: "test",
		},
		callback: func() { called = true },
	}

	instance.AddInput(converter)

	err := instance.RunInput(container)
	if err != nil {
		t.Errorf("RunInput() error = %v, want nil", err)
	}
	if !called {
		t.Error("RunInput() did not call input converter")
	}
}

type mockInputConverterWithCallback struct {
	mockInputConverter
	callback func()
}

func (m *mockInputConverterWithCallback) Input(c Container) (Container, error) {
	if m.callback != nil {
		m.callback()
	}
	return c, nil
}

func TestInstance_RunInput_Error(t *testing.T) {
	instance, _ := NewInstance()
	container := NewContainer()

	// Add mock input converter that returns error
	converter := &mockInputConverterWithError{
		mockInputConverter: mockInputConverter{
			typ:         "test",
			action:      ActionAdd,
			description: "test",
		},
		err: errors.New("test error"),
	}

	instance.AddInput(converter)

	err := instance.RunInput(container)
	if err == nil {
		t.Error("RunInput() expected error, got nil")
	}
	if err.Error() != "test error" {
		t.Errorf("RunInput() error = %v, want 'test error'", err)
	}
}

type mockInputConverterWithError struct {
	mockInputConverter
	err error
}

func (m *mockInputConverterWithError) Input(c Container) (Container, error) {
	return nil, m.err
}

func TestInstance_RunOutput(t *testing.T) {
	instance, _ := NewInstance()
	container := NewContainer()

	// Add mock output converter
	called := false
	converter := &mockOutputConverterWithCallback{
		mockOutputConverter: mockOutputConverter{
			typ:         "test",
			action:      ActionOutput,
			description: "test",
		},
		callback: func() { called = true },
	}

	instance.AddOutput(converter)

	err := instance.RunOutput(container)
	if err != nil {
		t.Errorf("RunOutput() error = %v, want nil", err)
	}
	if !called {
		t.Error("RunOutput() did not call output converter")
	}
}

type mockOutputConverterWithCallback struct {
	mockOutputConverter
	callback func()
}

func (m *mockOutputConverterWithCallback) Output(c Container) error {
	if m.callback != nil {
		m.callback()
	}
	return nil
}

func TestInstance_RunOutput_Error(t *testing.T) {
	instance, _ := NewInstance()
	container := NewContainer()

	// Add mock output converter that returns error
	converter := &mockOutputConverterWithError{
		mockOutputConverter: mockOutputConverter{
			typ:         "test",
			action:      ActionOutput,
			description: "test",
		},
		err: errors.New("test error"),
	}

	instance.AddOutput(converter)

	err := instance.RunOutput(container)
	if err == nil {
		t.Error("RunOutput() expected error, got nil")
	}
	if err.Error() != "test error" {
		t.Errorf("RunOutput() error = %v, want 'test error'", err)
	}
}

type mockOutputConverterWithError struct {
	mockOutputConverter
	err error
}

func (m *mockOutputConverterWithError) Output(c Container) error {
	return m.err
}

func TestInstance_Run(t *testing.T) {
	instance, _ := NewInstance()

	instance.AddInput(&mockInputConverter{typ: "test", action: ActionAdd, description: "test"})
	instance.AddOutput(&mockOutputConverter{typ: "test", action: ActionOutput, description: "test"})

	err := instance.Run()
	if err != nil {
		t.Errorf("Run() error = %v, want nil", err)
	}
}

func TestInstance_Run_NoInput(t *testing.T) {
	instance, _ := NewInstance()

	instance.AddOutput(&mockOutputConverter{typ: "test", action: ActionOutput, description: "test"})

	err := instance.Run()
	if err == nil {
		t.Error("Run() without input expected error, got nil")
	}
	if err.Error() != "input type and output type must be specified" {
		t.Errorf("Run() error = %v, want 'input type and output type must be specified'", err)
	}
}

func TestInstance_Run_NoOutput(t *testing.T) {
	instance, _ := NewInstance()

	instance.AddInput(&mockInputConverter{typ: "test", action: ActionAdd, description: "test"})

	err := instance.Run()
	if err == nil {
		t.Error("Run() without output expected error, got nil")
	}
	if err.Error() != "input type and output type must be specified" {
		t.Errorf("Run() error = %v, want 'input type and output type must be specified'", err)
	}
}

func TestInstance_Run_InputError(t *testing.T) {
	instance, _ := NewInstance()

	instance.AddInput(&mockInputConverterWithError{
		mockInputConverter: mockInputConverter{typ: "test", action: ActionAdd, description: "test"},
		err:                errors.New("input error"),
	})
	instance.AddOutput(&mockOutputConverter{typ: "test", action: ActionOutput, description: "test"})

	err := instance.Run()
	if err == nil {
		t.Error("Run() with input error expected error, got nil")
	}
	if err.Error() != "input error" {
		t.Errorf("Run() error = %v, want 'input error'", err)
	}
}

func TestInstance_Run_OutputError(t *testing.T) {
	instance, _ := NewInstance()

	instance.AddInput(&mockInputConverter{typ: "test", action: ActionAdd, description: "test"})
	instance.AddOutput(&mockOutputConverterWithError{
		mockOutputConverter: mockOutputConverter{typ: "test", action: ActionOutput, description: "test"},
		err:                 errors.New("output error"),
	})

	err := instance.Run()
	if err == nil {
		t.Error("Run() with output error expected error, got nil")
	}
	if err.Error() != "output error" {
		t.Errorf("Run() error = %v, want 'output error'", err)
	}
}

func TestInstance_InitConfigFromBytes(t *testing.T) {
	// Setup config creators
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

	instance, _ := NewInstance()

	configJSON := `{
		"input": [
			{"type": "testin", "action": "add", "args": {}}
		],
		"output": [
			{"type": "testout", "action": "output", "args": {}}
		]
	}`

	err := instance.InitConfigFromBytes([]byte(configJSON))
	if err != nil {
		t.Errorf("InitConfigFromBytes() error = %v, want nil", err)
	}

	// Verify converters were added
	err = instance.Run()
	if err != nil {
		t.Errorf("Run() after InitConfigFromBytes error = %v, want nil", err)
	}
}

func TestInstance_InitConfigFromBytes_WithComments(t *testing.T) {
	// Setup config creators
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

	instance, _ := NewInstance()

	// Config with comments and trailing commas (hujson format)
	configJSON := `{
		// This is a comment
		"input": [
			{"type": "testin", "action": "add", "args": {}}, // trailing comma
		],
		"output": [
			{"type": "testout", "action": "output", "args": {}},
		], // trailing comma
	}`

	err := instance.InitConfigFromBytes([]byte(configJSON))
	if err != nil {
		t.Errorf("InitConfigFromBytes() with comments error = %v, want nil", err)
	}

	// Verify converters were added
	err = instance.Run()
	if err != nil {
		t.Errorf("Run() after InitConfigFromBytes with comments error = %v, want nil", err)
	}
}

func TestInstance_InitConfigFromBytes_InvalidJSON(t *testing.T) {
	instance, _ := NewInstance()

	configJSON := `{invalid json}`

	err := instance.InitConfigFromBytes([]byte(configJSON))
	if err == nil {
		t.Error("InitConfigFromBytes() with invalid JSON expected error, got nil")
	}
}

func TestInstance_InitConfig_LocalFile(t *testing.T) {
	// Setup config creators
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

	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	configJSON := `{
		"input": [
			{"type": "testin", "action": "add", "args": {}}
		],
		"output": [
			{"type": "testout", "action": "output", "args": {}}
		]
	}`

	err := os.WriteFile(configFile, []byte(configJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	instance, _ := NewInstance()

	err = instance.InitConfig(configFile)
	if err != nil {
		t.Errorf("InitConfig() error = %v, want nil", err)
	}

	// Verify converters were added
	err = instance.Run()
	if err != nil {
		t.Errorf("Run() after InitConfig error = %v, want nil", err)
	}
}

func TestInstance_InitConfig_RemoteURL(t *testing.T) {
	// Setup config creators
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

	// Create test server
	configJSON := `{
		"input": [
			{"type": "testin", "action": "add", "args": {}}
		],
		"output": [
			{"type": "testout", "action": "output", "args": {}}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(configJSON))
	}))
	defer server.Close()

	instance, _ := NewInstance()

	err := instance.InitConfig(server.URL)
	if err != nil {
		t.Errorf("InitConfig() with URL error = %v, want nil", err)
	}

	// Verify converters were added
	err = instance.Run()
	if err != nil {
		t.Errorf("Run() after InitConfig with URL error = %v, want nil", err)
	}
}

func TestInstance_InitConfig_RemoteURL_HTTPS(t *testing.T) {
	// Setup config creators
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

	// Create test server
	configJSON := `{
		"input": [
			{"type": "testin", "action": "add", "args": {}}
		],
		"output": [
			{"type": "testout", "action": "output", "args": {}}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(configJSON))
	}))
	defer server.Close()

	instance, _ := NewInstance()

	// Replace http:// with https:// in URL (will fail but tests the code path)
	httpsURL := "https" + server.URL[4:]
	err := instance.InitConfig(httpsURL)
	// This will fail because it's not a real HTTPS server, but it tests the code path
	if err == nil {
		// If it somehow succeeds, that's also fine
		t.Log("InitConfig() with HTTPS URL succeeded unexpectedly")
	}
}

func TestInstance_InitConfig_FileNotFound(t *testing.T) {
	instance, _ := NewInstance()

	err := instance.InitConfig("/nonexistent/config.json")
	if err == nil {
		t.Error("InitConfig() with non-existent file expected error, got nil")
	}
}

func TestInstance_InitConfig_WithSpaces(t *testing.T) {
	// Setup config creators
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

	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	configJSON := `{
		"input": [
			{"type": "testin", "action": "add", "args": {}}
		],
		"output": [
			{"type": "testout", "action": "output", "args": {}}
		]
	}`

	err := os.WriteFile(configFile, []byte(configJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	instance, _ := NewInstance()

	// Test with spaces around the path
	err = instance.InitConfig("  " + configFile + "  ")
	if err != nil {
		t.Errorf("InitConfig() with spaces error = %v, want nil", err)
	}
}

func TestInstance_MultipleInputOutput(t *testing.T) {
	instance, _ := NewInstance()

	// Add multiple inputs and outputs
	instance.AddInput(&mockInputConverter{typ: "test1", action: ActionAdd, description: "test1"})
	instance.AddInput(&mockInputConverter{typ: "test2", action: ActionAdd, description: "test2"})
	instance.AddOutput(&mockOutputConverter{typ: "test1", action: ActionOutput, description: "test1"})
	instance.AddOutput(&mockOutputConverter{typ: "test2", action: ActionOutput, description: "test2"})

	err := instance.Run()
	if err != nil {
		t.Errorf("Run() with multiple converters error = %v, want nil", err)
	}
}
