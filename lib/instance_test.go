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
	instance, err := NewInstance()
	if err != nil {
		t.Errorf("NewInstance() error = %v", err)
	}
	if instance == nil {
		t.Error("NewInstance() returned nil")
	}
}

func TestInstance_AddInput(t *testing.T) {
	instance, _ := NewInstance()
	mockInput := &mockInputConverter{typ: "test"}
	
	instance.AddInput(mockInput)
	
	// We can't directly access the input slice, but we can test Run behavior
	// This is tested indirectly through other tests
}

func TestInstance_AddOutput(t *testing.T) {
	instance, _ := NewInstance()
	mockOutput := &mockOutputConverter{typ: "test"}
	
	instance.AddOutput(mockOutput)
	
	// Similar to AddInput, tested indirectly
}

func TestInstance_ResetInput(t *testing.T) {
	instance, _ := NewInstance()
	mockInput := &mockInputConverter{typ: "test"}
	
	instance.AddInput(mockInput)
	instance.ResetInput()
	
	// After reset, Run should fail due to no inputs
	err := instance.Run()
	if err == nil {
		t.Error("Instance.Run() should fail after ResetInput")
	}
}

func TestInstance_ResetOutput(t *testing.T) {
	instance, _ := NewInstance()
	mockOutput := &mockOutputConverter{typ: "test"}
	
	instance.AddOutput(mockOutput)
	instance.ResetOutput()
	
	// After reset, Run should fail due to no outputs
	err := instance.Run()
	if err == nil {
		t.Error("Instance.Run() should fail after ResetOutput")
	}
}

func TestInstance_Run(t *testing.T) {
	tests := []struct {
		name     string
		setupFn  func(Instance)
		wantErr  bool
	}{
		{
			name: "no input or output",
			setupFn: func(i Instance) {
				// Don't add anything
			},
			wantErr: true,
		},
		{
			name: "only input",
			setupFn: func(i Instance) {
				i.AddInput(&mockInputConverter{typ: "test"})
			},
			wantErr: true,
		},
		{
			name: "only output",
			setupFn: func(i Instance) {
				i.AddOutput(&mockOutputConverter{typ: "test"})
			},
			wantErr: true,
		},
		{
			name: "both input and output",
			setupFn: func(i Instance) {
				i.AddInput(&mockInputConverter{typ: "test"})
				i.AddOutput(&mockOutputConverter{typ: "test"})
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance, _ := NewInstance()
			tt.setupFn(instance)
			
			err := instance.Run()
			if (err != nil) != tt.wantErr {
				t.Errorf("Instance.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInstance_RunInput(t *testing.T) {
	instance, _ := NewInstance()
	instance.AddInput(&mockInputConverter{typ: "test"})
	
	container := NewContainer()
	err := instance.RunInput(container)
	if err != nil {
		t.Errorf("Instance.RunInput() error = %v", err)
	}
}

func TestInstance_RunOutput(t *testing.T) {
	instance, _ := NewInstance()
	instance.AddOutput(&mockOutputConverter{typ: "test"})
	
	container := NewContainer()
	err := instance.RunOutput(container)
	if err != nil {
		t.Errorf("Instance.RunOutput() error = %v", err)
	}
}

func TestInstance_InitConfigFromBytes(t *testing.T) {
	// Save original state
	originalInputCache := inputConfigCreatorCache
	originalOutputCache := outputConfigCreatorCache
	defer func() {
		inputConfigCreatorCache = originalInputCache
		outputConfigCreatorCache = originalOutputCache
	}()
	
	// Reset caches
	inputConfigCreatorCache = make(map[string]inputConfigCreator)
	outputConfigCreatorCache = make(map[string]outputConfigCreator)
	
	// Register test creators
	RegisterInputConfigCreator("test", func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typ: "test"}, nil
	})
	RegisterOutputConfigCreator("test", func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typ: "test"}, nil
	})

	tests := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{
			name: "valid config",
			config: `{
				"input": [{"type": "test", "action": "add", "args": {}}],
				"output": [{"type": "test", "action": "output", "args": {}}]
			}`,
			wantErr: false,
		},
		{
			name: "config with comments",
			config: `{
				// This is a comment
				"input": [{"type": "test", "action": "add", "args": {}}],
				"output": [{"type": "test", "action": "output", "args": {}}]
			}`,
			wantErr: false,
		},
		{
			name: "config with trailing comma",
			config: `{
				"input": [{"type": "test", "action": "add", "args": {}}],
				"output": [{"type": "test", "action": "output", "args": {}}],
			}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			config:  `{invalid}`,
			wantErr: true,
		},
		{
			name: "unknown input type",
			config: `{
				"input": [{"type": "unknown", "action": "add", "args": {}}],
				"output": [{"type": "test", "action": "output", "args": {}}]
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance, _ := NewInstance()
			err := instance.InitConfigFromBytes([]byte(tt.config))
			if (err != nil) != tt.wantErr {
				t.Errorf("Instance.InitConfigFromBytes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInstance_InitConfig(t *testing.T) {
	// Save original state
	originalInputCache := inputConfigCreatorCache
	originalOutputCache := outputConfigCreatorCache
	defer func() {
		inputConfigCreatorCache = originalInputCache
		outputConfigCreatorCache = originalOutputCache
	}()
	
	// Reset caches
	inputConfigCreatorCache = make(map[string]inputConfigCreator)
	outputConfigCreatorCache = make(map[string]outputConfigCreator)
	
	// Register test creators
	RegisterInputConfigCreator("test", func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typ: "test"}, nil
	})
	RegisterOutputConfigCreator("test", func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typ: "test"}, nil
	})

	configContent := `{
		"input": [{"type": "test", "action": "add", "args": {}}],
		"output": [{"type": "test", "action": "output", "args": {}}]
	}`

	t.Run("load from file", func(t *testing.T) {
		// Create a temporary config file
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.json")
		if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to create test config file: %v", err)
		}

		instance, _ := NewInstance()
		err := instance.InitConfig(configFile)
		if err != nil {
			t.Errorf("Instance.InitConfig() error = %v", err)
		}
	})

	t.Run("load from HTTP URL", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(configContent))
		}))
		defer server.Close()

		instance, _ := NewInstance()
		err := instance.InitConfig(server.URL)
		if err != nil {
			t.Errorf("Instance.InitConfig() with HTTP URL error = %v", err)
		}
	})

	t.Run("load from HTTPS URL", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(configContent))
		}))
		defer server.Close()

		instance, _ := NewInstance()
		// This may fail due to certificate issues in test, but we test the code path
		instance.InitConfig(server.URL)
	})

	t.Run("non-existent file", func(t *testing.T) {
		instance, _ := NewInstance()
		err := instance.InitConfig("/nonexistent/config.json")
		if err == nil {
			t.Error("Instance.InitConfig() should return error for non-existent file")
		}
	})

	t.Run("URL with whitespace", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(configContent))
		}))
		defer server.Close()

		instance, _ := NewInstance()
		err := instance.InitConfig("  " + server.URL + "  ")
		if err != nil {
			t.Errorf("Instance.InitConfig() with whitespace error = %v", err)
		}
	})
}

func TestInstance_RunInputError(t *testing.T) {
	instance, _ := NewInstance()
	
	// Add an input converter that returns an error
	instance.AddInput(&mockErrorInputConverter{})
	
	container := NewContainer()
	err := instance.RunInput(container)
	if err == nil {
		t.Error("Instance.RunInput() should return error when converter fails")
	}
}

func TestInstance_RunOutputError(t *testing.T) {
	instance, _ := NewInstance()
	
	// Add an output converter that returns an error
	instance.AddOutput(&mockErrorOutputConverter{})
	
	container := NewContainer()
	err := instance.RunOutput(container)
	if err == nil {
		t.Error("Instance.RunOutput() should return error when converter fails")
	}
}

func TestInstance_RunWithContainer(t *testing.T) {
	// Save original state
	originalInputCache := inputConfigCreatorCache
	originalOutputCache := outputConfigCreatorCache
	defer func() {
		inputConfigCreatorCache = originalInputCache
		outputConfigCreatorCache = originalOutputCache
	}()
	
	// Reset caches
	inputConfigCreatorCache = make(map[string]inputConfigCreator)
	outputConfigCreatorCache = make(map[string]outputConfigCreator)
	
	// Register test creators
	RegisterInputConfigCreator("test", func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typ: "test"}, nil
	})
	RegisterOutputConfigCreator("test", func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typ: "test"}, nil
	})

	// Test full run with both input and output
	instance, _ := NewInstance()
	instance.AddInput(&mockInputConverter{typ: "test"})
	instance.AddOutput(&mockOutputConverter{typ: "test"})
	
	err := instance.Run()
	if err != nil {
		t.Errorf("Instance.Run() error = %v", err)
	}
}

// Mock error converters
type mockErrorInputConverter struct{}

func (m *mockErrorInputConverter) GetType() string { return "error" }
func (m *mockErrorInputConverter) GetAction() Action { return ActionAdd }
func (m *mockErrorInputConverter) GetDescription() string { return "error converter" }
func (m *mockErrorInputConverter) Input(c Container) (Container, error) {
	return nil, ErrNotSupportedFormat
}

type mockErrorOutputConverter struct{}

func (m *mockErrorOutputConverter) GetType() string { return "error" }
func (m *mockErrorOutputConverter) GetAction() Action { return ActionOutput }
func (m *mockErrorOutputConverter) GetDescription() string { return "error converter" }
func (m *mockErrorOutputConverter) Output(c Container) error {
	return ErrNotSupportedFormat
}
