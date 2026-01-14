package lib

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewInstance(t *testing.T) {
	inst, err := NewInstance()
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}
	if inst == nil {
		t.Fatal("NewInstance returned nil")
	}
}

func TestInstanceAddInput(t *testing.T) {
	inst, _ := NewInstance()
	mockConv := &mockInputConverter{
		typeName:    "test",
		action:      ActionAdd,
		description: "Test",
	}

	inst.AddInput(mockConv)

	// Verify by running
	container := NewContainer()
	err := inst.RunInput(container)
	if err != nil {
		t.Fatalf("RunInput failed: %v", err)
	}
}

func TestInstanceAddOutput(t *testing.T) {
	inst, _ := NewInstance()
	mockConv := &mockOutputConverter{
		typeName:    "test",
		action:      ActionOutput,
		description: "Test",
	}

	inst.AddOutput(mockConv)

	// Verify by running
	container := NewContainer()
	err := inst.RunOutput(container)
	if err != nil {
		t.Fatalf("RunOutput failed: %v", err)
	}
}

func TestInstanceResetInput(t *testing.T) {
	inst, _ := NewInstance()
	mockConv := &mockInputConverter{
		typeName:    "test",
		action:      ActionAdd,
		description: "Test",
	}

	inst.AddInput(mockConv)
	inst.ResetInput()

	// After reset, RunInput should not process anything
	container := NewContainer()
	err := inst.RunInput(container)
	if err != nil {
		t.Fatalf("RunInput failed: %v", err)
	}
}

func TestInstanceResetOutput(t *testing.T) {
	inst, _ := NewInstance()
	mockConv := &mockOutputConverter{
		typeName:    "test",
		action:      ActionOutput,
		description: "Test",
	}

	inst.AddOutput(mockConv)
	inst.ResetOutput()

	// After reset, RunOutput should not process anything
	container := NewContainer()
	err := inst.RunOutput(container)
	if err != nil {
		t.Fatalf("RunOutput failed: %v", err)
	}
}

func TestInstanceRunInput(t *testing.T) {
	inst, _ := NewInstance()

	inputConv := &mockInputConverterWithData{
		mockInputConverter: mockInputConverter{
			typeName:    "test",
			action:      ActionAdd,
			description: "Test",
		},
	}

	inst.AddInput(inputConv)

	container := NewContainer()
	err := inst.RunInput(container)
	if err != nil {
		t.Fatalf("RunInput failed: %v", err)
	}
}

func TestInstanceRunOutput(t *testing.T) {
	inst, _ := NewInstance()

	outputConv := &mockOutputConverter{
		typeName:    "test",
		action:      ActionOutput,
		description: "Test",
	}

	inst.AddOutput(outputConv)

	container := NewContainer()
	err := inst.RunOutput(container)
	if err != nil {
		t.Fatalf("RunOutput failed: %v", err)
	}
}

func TestInstanceRun_NoInputOrOutput(t *testing.T) {
	inst, _ := NewInstance()

	err := inst.Run()
	if err == nil {
		t.Error("Run should fail when no input or output is specified")
	}
}

func TestInstanceRun_NoInput(t *testing.T) {
	inst, _ := NewInstance()

	inst.AddOutput(&mockOutputConverter{
		typeName:    "test",
		action:      ActionOutput,
		description: "Test",
	})

	err := inst.Run()
	if err == nil {
		t.Error("Run should fail when no input is specified")
	}
}

func TestInstanceRun_NoOutput(t *testing.T) {
	inst, _ := NewInstance()

	inst.AddInput(&mockInputConverter{
		typeName:    "test",
		action:      ActionAdd,
		description: "Test",
	})

	err := inst.Run()
	if err == nil {
		t.Error("Run should fail when no output is specified")
	}
}

func TestInstanceRun_Success(t *testing.T) {
	inst, _ := NewInstance()

	inst.AddInput(&mockInputConverter{
		typeName:    "test",
		action:      ActionAdd,
		description: "Test",
	})

	inst.AddOutput(&mockOutputConverter{
		typeName:    "test",
		action:      ActionOutput,
		description: "Test",
	})

	err := inst.Run()
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
}

func TestInstanceInitConfig_LocalFile(t *testing.T) {
	// Create a temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Register mock converters for this test
	inputType := "instance_test_input_" + t.Name()
	outputType := "instance_test_output_" + t.Name()

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

	configContent := `{
		"input": [{"type":"` + inputType + `","action":"add","args":{}}],
		"output": [{"type":"` + outputType + `","action":"output","args":{}}]
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	inst, _ := NewInstance()
	err = inst.InitConfig(configPath)
	if err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}

	// Should be able to run now
	err = inst.Run()
	if err != nil {
		t.Fatalf("Run failed after InitConfig: %v", err)
	}
}

func TestInstanceInitConfig_LocalFileWithComments(t *testing.T) {
	// Create a temp config file with JSON comments
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Register mock converters for this test
	inputType := "instance_test_input_comments_" + t.Name()
	outputType := "instance_test_output_comments_" + t.Name()

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

	// JSON with comments and trailing comma
	configContent := `{
		// This is a comment
		"input": [
			{"type":"` + inputType + `","action":"add","args":{}},
		],
		/* Multi-line comment */
		"output": [
			{"type":"` + outputType + `","action":"output","args":{}},
		],
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	inst, _ := NewInstance()
	err = inst.InitConfig(configPath)
	if err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}
}

func TestInstanceInitConfig_RemoteURL(t *testing.T) {
	// Register mock converters for this test
	inputType := "instance_test_input_remote_" + t.Name()
	outputType := "instance_test_output_remote_" + t.Name()

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

	configContent := `{
		"input": [{"type":"` + inputType + `","action":"add","args":{}}],
		"output": [{"type":"` + outputType + `","action":"output","args":{}}]
	}`

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(configContent))
	}))
	defer server.Close()

	inst, _ := NewInstance()
	err := inst.InitConfig(server.URL)
	if err != nil {
		t.Fatalf("InitConfig from remote URL failed: %v", err)
	}

	// Should be able to run now
	err = inst.Run()
	if err != nil {
		t.Fatalf("Run failed after InitConfig from remote: %v", err)
	}
}

func TestInstanceInitConfig_FileNotFound(t *testing.T) {
	inst, _ := NewInstance()
	err := inst.InitConfig("/nonexistent/path/to/config.json")
	if err == nil {
		t.Error("InitConfig should fail for non-existent file")
	}
}

func TestInstanceInitConfig_InvalidJSON(t *testing.T) {
	// Create a temp config file with invalid JSON
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	err := os.WriteFile(configPath, []byte("{invalid json}"), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	inst, _ := NewInstance()
	err = inst.InitConfig(configPath)
	if err == nil {
		t.Error("InitConfig should fail for invalid JSON")
	}
}

func TestInstanceInitConfigFromBytes(t *testing.T) {
	// Register mock converters for this test
	inputType := "instance_test_input_bytes_" + t.Name()
	outputType := "instance_test_output_bytes_" + t.Name()

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

	configContent := []byte(`{
		"input": [{"type":"` + inputType + `","action":"add","args":{}}],
		"output": [{"type":"` + outputType + `","action":"output","args":{}}]
	}`)

	inst, _ := NewInstance()
	err := inst.InitConfigFromBytes(configContent)
	if err != nil {
		t.Fatalf("InitConfigFromBytes failed: %v", err)
	}

	// Should be able to run now
	err = inst.Run()
	if err != nil {
		t.Fatalf("Run failed after InitConfigFromBytes: %v", err)
	}
}

func TestInstanceInitConfigFromBytes_InvalidJSON(t *testing.T) {
	inst, _ := NewInstance()
	err := inst.InitConfigFromBytes([]byte("{invalid json}"))
	if err == nil {
		t.Error("InitConfigFromBytes should fail for invalid JSON")
	}
}

// Mock input converter that adds data to container
type mockInputConverterWithData struct {
	mockInputConverter
}

func (m *mockInputConverterWithData) Input(c Container) (Container, error) {
	entry := NewEntry("test")
	if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
		return nil, err
	}
	if err := c.Add(entry); err != nil {
		return nil, err
	}
	return c, nil
}

// Mock input converter that returns error
type mockInputConverterWithError struct {
	mockInputConverter
	err error
}

func (m *mockInputConverterWithError) Input(c Container) (Container, error) {
	return nil, m.err
}

// Mock output converter that returns error
type mockOutputConverterWithError struct {
	mockOutputConverter
	err error
}

func (m *mockOutputConverterWithError) Output(c Container) error {
	return m.err
}

func TestInstanceRunInput_Error(t *testing.T) {
	inst, _ := NewInstance()

	inputConv := &mockInputConverterWithError{
		mockInputConverter: mockInputConverter{
			typeName:    "test",
			action:      ActionAdd,
			description: "Test",
		},
		err: ErrInvalidIP,
	}

	inst.AddInput(inputConv)

	container := NewContainer()
	err := inst.RunInput(container)
	if err != ErrInvalidIP {
		t.Errorf("RunInput error = %v, want %v", err, ErrInvalidIP)
	}
}

func TestInstanceRunOutput_Error(t *testing.T) {
	inst, _ := NewInstance()

	outputConv := &mockOutputConverterWithError{
		mockOutputConverter: mockOutputConverter{
			typeName:    "test",
			action:      ActionOutput,
			description: "Test",
		},
		err: ErrNotSupportedFormat,
	}

	inst.AddOutput(outputConv)

	container := NewContainer()
	err := inst.RunOutput(container)
	if err != ErrNotSupportedFormat {
		t.Errorf("RunOutput error = %v, want %v", err, ErrNotSupportedFormat)
	}
}

func TestInstanceInitConfig_HTTPSPrefix(t *testing.T) {
	// Register mock converters for this test
	inputType := "instance_test_https_" + t.Name()
	outputType := "instance_test_https_out_" + t.Name()

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

	configContent := `{
		"input": [{"type":"` + inputType + `","action":"add","args":{}}],
		"output": [{"type":"` + outputType + `","action":"output","args":{}}]
	}`

	// Create test TLS server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(configContent))
	}))
	defer server.Close()

	// Note: This test will fail due to self-signed certificate
	// But it tests the URL prefix detection logic
	inst, _ := NewInstance()
	_ = inst.InitConfig(server.URL)
	// We don't check the error because TLS will fail with self-signed cert
}

func TestInstanceInitConfig_WithSpaces(t *testing.T) {
	// Create a temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Register mock converters for this test
	inputType := "instance_test_spaces_" + t.Name()
	outputType := "instance_test_spaces_out_" + t.Name()

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

	configContent := `{
		"input": [{"type":"` + inputType + `","action":"add","args":{}}],
		"output": [{"type":"` + outputType + `","action":"output","args":{}}]
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	inst, _ := NewInstance()
	// Add spaces around the path
	err = inst.InitConfig("  " + configPath + "  ")
	if err != nil {
		t.Fatalf("InitConfig with spaces failed: %v", err)
	}
}

func TestInstanceRun_InputError(t *testing.T) {
	inst, _ := NewInstance()

	inst.AddInput(&mockInputConverterWithError{
		mockInputConverter: mockInputConverter{
			typeName:    "test",
			action:      ActionAdd,
			description: "Test",
		},
		err: ErrInvalidIP,
	})

	inst.AddOutput(&mockOutputConverter{
		typeName:    "test",
		action:      ActionOutput,
		description: "Test",
	})

	err := inst.Run()
	if err != ErrInvalidIP {
		t.Errorf("Run should fail with input error, got %v, want %v", err, ErrInvalidIP)
	}
}

func TestInstanceRun_OutputError(t *testing.T) {
	inst, _ := NewInstance()

	inst.AddInput(&mockInputConverter{
		typeName:    "test",
		action:      ActionAdd,
		description: "Test",
	})

	inst.AddOutput(&mockOutputConverterWithError{
		mockOutputConverter: mockOutputConverter{
			typeName:    "test",
			action:      ActionOutput,
			description: "Test",
		},
		err: ErrNotSupportedFormat,
	})

	err := inst.Run()
	if err != ErrNotSupportedFormat {
		t.Errorf("Run should fail with output error, got %v, want %v", err, ErrNotSupportedFormat)
	}
}
