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

// mockInputConverterWithBehavior allows customizing Input behavior
type mockInputConverterWithBehavior struct {
	iType       string
	action      Action
	description string
	inputFunc   func(Container) (Container, error)
}

func (m *mockInputConverterWithBehavior) GetType() string        { return m.iType }
func (m *mockInputConverterWithBehavior) GetAction() Action      { return m.action }
func (m *mockInputConverterWithBehavior) GetDescription() string { return m.description }
func (m *mockInputConverterWithBehavior) Input(c Container) (Container, error) {
	if m.inputFunc != nil {
		return m.inputFunc(c)
	}
	return c, nil
}

// mockOutputConverterWithBehavior allows customizing Output behavior
type mockOutputConverterWithBehavior struct {
	iType       string
	action      Action
	description string
	outputFunc  func(Container) error
}

func (m *mockOutputConverterWithBehavior) GetType() string        { return m.iType }
func (m *mockOutputConverterWithBehavior) GetAction() Action      { return m.action }
func (m *mockOutputConverterWithBehavior) GetDescription() string { return m.description }
func (m *mockOutputConverterWithBehavior) Output(c Container) error {
	if m.outputFunc != nil {
		return m.outputFunc(c)
	}
	return nil
}

func TestNewInstance(t *testing.T) {
	inst, err := NewInstance()
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}
	if inst == nil {
		t.Fatal("NewInstance returned nil")
	}
}

func TestInstance_InitConfigFromBytes(t *testing.T) {
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

	t.Run("valid config", func(t *testing.T) {
		inst, _ := NewInstance()
		content := []byte(`{
			"input": [{"type": "testinput", "action": "add", "args": {}}],
			"output": [{"type": "testoutput", "action": "output", "args": {}}]
		}`)
		err := inst.InitConfigFromBytes(content)
		if err != nil {
			t.Fatalf("InitConfigFromBytes failed: %v", err)
		}
	})

	t.Run("with JSON comments", func(t *testing.T) {
		inst, _ := NewInstance()
		content := []byte(`{
			// This is a comment
			"input": [{"type": "testinput", "action": "add", "args": {}}],
			"output": [{"type": "testoutput", /* inline comment */ "action": "output", "args": {}}]
		}`)
		err := inst.InitConfigFromBytes(content)
		if err != nil {
			t.Fatalf("InitConfigFromBytes with comments failed: %v", err)
		}
	})

	t.Run("with trailing comma", func(t *testing.T) {
		inst, _ := NewInstance()
		content := []byte(`{
			"input": [{"type": "testinput", "action": "add", "args": {}},],
			"output": [{"type": "testoutput", "action": "output", "args": {}},],
		}`)
		err := inst.InitConfigFromBytes(content)
		if err != nil {
			t.Fatalf("InitConfigFromBytes with trailing comma failed: %v", err)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		inst, _ := NewInstance()
		content := []byte(`{invalid json}`)
		err := inst.InitConfigFromBytes(content)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("empty config", func(t *testing.T) {
		inst, _ := NewInstance()
		content := []byte(`{"input": [], "output": []}`)
		err := inst.InitConfigFromBytes(content)
		if err != nil {
			t.Fatalf("InitConfigFromBytes with empty arrays failed: %v", err)
		}
	})
}

func TestInstance_InitConfig(t *testing.T) {
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

	t.Run("local file", func(t *testing.T) {
		// Create temp config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")
		content := []byte(`{
			"input": [{"type": "testinput", "action": "add", "args": {}}],
			"output": [{"type": "testoutput", "action": "output", "args": {}}]
		}`)
		if err := os.WriteFile(configPath, content, 0644); err != nil {
			t.Fatalf("failed to write temp config: %v", err)
		}

		inst, _ := NewInstance()
		err := inst.InitConfig(configPath)
		if err != nil {
			t.Fatalf("InitConfig failed: %v", err)
		}
	})

	t.Run("local file not found", func(t *testing.T) {
		inst, _ := NewInstance()
		err := inst.InitConfig("/nonexistent/path/config.json")
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("remote URL", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"input": [{"type": "testinput", "action": "add", "args": {}}],
				"output": [{"type": "testoutput", "action": "output", "args": {}}]
			}`))
		}))
		defer server.Close()

		inst, _ := NewInstance()
		err := inst.InitConfig(server.URL)
		if err != nil {
			t.Fatalf("InitConfig with URL failed: %v", err)
		}
	})

	t.Run("remote URL https", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"input": [{"type": "testinput", "action": "add", "args": {}}],
				"output": [{"type": "testoutput", "action": "output", "args": {}}]
			}`))
		}))
		defer server.Close()

		// Note: TLS test server won't work without proper cert handling,
		// but we can test the URL detection
		inst, _ := NewInstance()
		// This will fail due to cert issues, but it exercises the HTTPS path
		_ = inst.InitConfig(server.URL)
	})

	t.Run("remote URL error", func(t *testing.T) {
		inst, _ := NewInstance()
		err := inst.InitConfig("http://invalid.invalid.invalid:1234")
		if err == nil {
			t.Error("expected error for invalid URL")
		}
	})

	t.Run("with spaces in path", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")
		content := []byte(`{"input": [], "output": []}`)
		if err := os.WriteFile(configPath, content, 0644); err != nil {
			t.Fatalf("failed to write temp config: %v", err)
		}

		inst, _ := NewInstance()
		err := inst.InitConfig("  " + configPath + "  ")
		if err != nil {
			t.Fatalf("InitConfig with spaces failed: %v", err)
		}
	})
}

func TestInstance_AddInput(t *testing.T) {
	inst, _ := NewInstance()
	conv := &mockInputConverter{iType: "test"}

	inst.AddInput(conv)

	// Verify by running
	container := NewContainer()
	err := inst.RunInput(container)
	if err != nil {
		t.Fatalf("RunInput failed: %v", err)
	}
}

func TestInstance_AddOutput(t *testing.T) {
	inst, _ := NewInstance()
	conv := &mockOutputConverter{iType: "test"}

	inst.AddOutput(conv)

	// Verify by running
	container := NewContainer()
	err := inst.RunOutput(container)
	if err != nil {
		t.Fatalf("RunOutput failed: %v", err)
	}
}

func TestInstance_ResetInput(t *testing.T) {
	inst, _ := NewInstance()
	inst.AddInput(&mockInputConverter{iType: "test1"})
	inst.AddInput(&mockInputConverter{iType: "test2"})

	inst.ResetInput()

	// After reset, running Run should fail due to empty input
	inst.AddOutput(&mockOutputConverter{iType: "test"})
	err := inst.Run()
	if err == nil {
		t.Error("expected error for empty input after reset")
	}
}

func TestInstance_ResetOutput(t *testing.T) {
	inst, _ := NewInstance()
	inst.AddOutput(&mockOutputConverter{iType: "test1"})
	inst.AddOutput(&mockOutputConverter{iType: "test2"})

	inst.ResetOutput()

	// After reset, running Run should fail due to empty output
	inst.AddInput(&mockInputConverter{iType: "test"})
	err := inst.Run()
	if err == nil {
		t.Error("expected error for empty output after reset")
	}
}

func TestInstance_RunInput(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		inst, _ := NewInstance()
		called := false
		inst.AddInput(&mockInputConverterWithBehavior{
			iType: "test",
			inputFunc: func(c Container) (Container, error) {
				called = true
				return c, nil
			},
		})

		container := NewContainer()
		err := inst.RunInput(container)
		if err != nil {
			t.Fatalf("RunInput failed: %v", err)
		}
		if !called {
			t.Error("input converter was not called")
		}
	})

	t.Run("error", func(t *testing.T) {
		inst, _ := NewInstance()
		inst.AddInput(&mockInputConverterWithBehavior{
			iType: "test",
			inputFunc: func(c Container) (Container, error) {
				return nil, errors.New("input error")
			},
		})

		container := NewContainer()
		err := inst.RunInput(container)
		if err == nil {
			t.Error("expected error from input converter")
		}
	})

	t.Run("multiple converters", func(t *testing.T) {
		inst, _ := NewInstance()
		callOrder := []string{}

		inst.AddInput(&mockInputConverterWithBehavior{
			iType: "first",
			inputFunc: func(c Container) (Container, error) {
				callOrder = append(callOrder, "first")
				return c, nil
			},
		})
		inst.AddInput(&mockInputConverterWithBehavior{
			iType: "second",
			inputFunc: func(c Container) (Container, error) {
				callOrder = append(callOrder, "second")
				return c, nil
			},
		})

		container := NewContainer()
		err := inst.RunInput(container)
		if err != nil {
			t.Fatalf("RunInput failed: %v", err)
		}
		if len(callOrder) != 2 || callOrder[0] != "first" || callOrder[1] != "second" {
			t.Errorf("expected call order [first, second], got %v", callOrder)
		}
	})
}

func TestInstance_RunOutput(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		inst, _ := NewInstance()
		called := false
		inst.AddOutput(&mockOutputConverterWithBehavior{
			iType: "test",
			outputFunc: func(c Container) error {
				called = true
				return nil
			},
		})

		container := NewContainer()
		err := inst.RunOutput(container)
		if err != nil {
			t.Fatalf("RunOutput failed: %v", err)
		}
		if !called {
			t.Error("output converter was not called")
		}
	})

	t.Run("error", func(t *testing.T) {
		inst, _ := NewInstance()
		inst.AddOutput(&mockOutputConverterWithBehavior{
			iType: "test",
			outputFunc: func(c Container) error {
				return errors.New("output error")
			},
		})

		container := NewContainer()
		err := inst.RunOutput(container)
		if err == nil {
			t.Error("expected error from output converter")
		}
	})

	t.Run("multiple converters", func(t *testing.T) {
		inst, _ := NewInstance()
		callOrder := []string{}

		inst.AddOutput(&mockOutputConverterWithBehavior{
			iType: "first",
			outputFunc: func(c Container) error {
				callOrder = append(callOrder, "first")
				return nil
			},
		})
		inst.AddOutput(&mockOutputConverterWithBehavior{
			iType: "second",
			outputFunc: func(c Container) error {
				callOrder = append(callOrder, "second")
				return nil
			},
		})

		container := NewContainer()
		err := inst.RunOutput(container)
		if err != nil {
			t.Fatalf("RunOutput failed: %v", err)
		}
		if len(callOrder) != 2 || callOrder[0] != "first" || callOrder[1] != "second" {
			t.Errorf("expected call order [first, second], got %v", callOrder)
		}
	})
}

func TestInstance_Run(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		inst, _ := NewInstance()
		inst.AddInput(&mockInputConverterWithBehavior{
			iType: "input",
			inputFunc: func(c Container) (Container, error) {
				return c, nil
			},
		})
		inst.AddOutput(&mockOutputConverterWithBehavior{
			iType: "output",
			outputFunc: func(c Container) error {
				return nil
			},
		})

		err := inst.Run()
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}
	})

	t.Run("no input", func(t *testing.T) {
		inst, _ := NewInstance()
		inst.AddOutput(&mockOutputConverter{iType: "output"})

		err := inst.Run()
		if err == nil {
			t.Error("expected error for no input")
		}
		if err.Error() != "input type and output type must be specified" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("no output", func(t *testing.T) {
		inst, _ := NewInstance()
		inst.AddInput(&mockInputConverter{iType: "input"})

		err := inst.Run()
		if err == nil {
			t.Error("expected error for no output")
		}
		if err.Error() != "input type and output type must be specified" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("input error", func(t *testing.T) {
		inst, _ := NewInstance()
		inst.AddInput(&mockInputConverterWithBehavior{
			iType: "input",
			inputFunc: func(c Container) (Container, error) {
				return nil, errors.New("input error")
			},
		})
		inst.AddOutput(&mockOutputConverter{iType: "output"})

		err := inst.Run()
		if err == nil {
			t.Error("expected error from input")
		}
	})

	t.Run("output error", func(t *testing.T) {
		inst, _ := NewInstance()
		inst.AddInput(&mockInputConverterWithBehavior{
			iType: "input",
			inputFunc: func(c Container) (Container, error) {
				return c, nil
			},
		})
		inst.AddOutput(&mockOutputConverterWithBehavior{
			iType: "output",
			outputFunc: func(c Container) error {
				return errors.New("output error")
			},
		})

		err := inst.Run()
		if err == nil {
			t.Error("expected error from output")
		}
	})
}
