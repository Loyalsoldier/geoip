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
	inst, err := NewInstance()
	if err != nil {
		t.Fatalf("NewInstance error = %v", err)
	}
	if inst == nil {
		t.Fatal("NewInstance returned nil")
	}
}

func TestInstanceAddInputOutput(t *testing.T) {
	inst, _ := NewInstance()

	ic := &mockInputConverter{typeName: "test", action: ActionAdd, description: "test"}
	oc := &mockOutputConverter{typeName: "test", action: ActionOutput, description: "test"}

	inst.AddInput(ic)
	inst.AddOutput(oc)

	// Verify through Run (it should work since both exist)
	err := inst.Run()
	if err != nil {
		t.Errorf("Run error = %v", err)
	}
}

func TestInstanceResetInputOutput(t *testing.T) {
	inst, _ := NewInstance()

	ic := &mockInputConverter{typeName: "test", action: ActionAdd, description: "test"}
	oc := &mockOutputConverter{typeName: "test", action: ActionOutput, description: "test"}

	inst.AddInput(ic)
	inst.AddOutput(oc)

	inst.ResetInput()
	inst.ResetOutput()

	// Should fail because both are now empty
	err := inst.Run()
	if err == nil {
		t.Error("expected error after reset")
	}
}

func TestInstanceRunNoInput(t *testing.T) {
	inst, _ := NewInstance()
	oc := &mockOutputConverter{typeName: "test", action: ActionOutput, description: "test"}
	inst.AddOutput(oc)

	err := inst.Run()
	if err == nil {
		t.Error("expected error when no input")
	}
}

func TestInstanceRunNoOutput(t *testing.T) {
	inst, _ := NewInstance()
	ic := &mockInputConverter{typeName: "test", action: ActionAdd, description: "test"}
	inst.AddInput(ic)

	err := inst.Run()
	if err == nil {
		t.Error("expected error when no output")
	}
}

func TestInstanceRunInputError(t *testing.T) {
	inst, _ := NewInstance()

	ic := &mockInputConverter{
		typeName: "test", action: ActionAdd, description: "test",
		inputFn: func(c Container) (Container, error) {
			return nil, errors.New("input error")
		},
	}
	oc := &mockOutputConverter{typeName: "test", action: ActionOutput, description: "test"}

	inst.AddInput(ic)
	inst.AddOutput(oc)

	err := inst.Run()
	if err == nil {
		t.Error("expected input error")
	}
	if err.Error() != "input error" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInstanceRunOutputError(t *testing.T) {
	inst, _ := NewInstance()

	ic := &mockInputConverter{typeName: "test", action: ActionAdd, description: "test"}
	oc := &mockOutputConverter{
		typeName: "test", action: ActionOutput, description: "test",
		outputFn: func(c Container) error {
			return errors.New("output error")
		},
	}

	inst.AddInput(ic)
	inst.AddOutput(oc)

	err := inst.Run()
	if err == nil {
		t.Error("expected output error")
	}
	if err.Error() != "output error" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInstanceRunInput(t *testing.T) {
	inst, _ := NewInstance()

	ic := &mockInputConverter{typeName: "test", action: ActionAdd, description: "test"}
	inst.AddInput(ic)

	container := NewContainer()
	if err := inst.RunInput(container); err != nil {
		t.Errorf("RunInput error = %v", err)
	}
}

func TestInstanceRunOutput(t *testing.T) {
	inst, _ := NewInstance()

	oc := &mockOutputConverter{typeName: "test", action: ActionOutput, description: "test"}
	inst.AddOutput(oc)

	container := NewContainer()
	if err := inst.RunOutput(container); err != nil {
		t.Errorf("RunOutput error = %v", err)
	}
}

func TestInstanceInitConfigFromBytes(t *testing.T) {
	origInputCache := inputConfigCreatorCache
	origOutputCache := outputConfigCreatorCache
	inputConfigCreatorCache = make(map[string]inputConfigCreator)
	outputConfigCreatorCache = make(map[string]outputConfigCreator)
	defer func() {
		inputConfigCreatorCache = origInputCache
		outputConfigCreatorCache = origOutputCache
	}()

	RegisterInputConfigCreator("test-input", func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typeName: "test-input", action: action, description: "test"}, nil
	})
	RegisterOutputConfigCreator("test-output", func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typeName: "test-output", action: action, description: "test"}, nil
	})

	inst, _ := NewInstance()

	configJSON := `{
		"input": [{"type": "test-input", "action": "add", "args": {}}],
		"output": [{"type": "test-output", "action": "output", "args": {}}]
	}`

	if err := inst.InitConfigFromBytes([]byte(configJSON)); err != nil {
		t.Errorf("InitConfigFromBytes error = %v", err)
	}

	if err := inst.Run(); err != nil {
		t.Errorf("Run after InitConfigFromBytes error = %v", err)
	}
}

func TestInstanceInitConfigFromBytesWithComments(t *testing.T) {
	origInputCache := inputConfigCreatorCache
	origOutputCache := outputConfigCreatorCache
	inputConfigCreatorCache = make(map[string]inputConfigCreator)
	outputConfigCreatorCache = make(map[string]outputConfigCreator)
	defer func() {
		inputConfigCreatorCache = origInputCache
		outputConfigCreatorCache = origOutputCache
	}()

	RegisterInputConfigCreator("test-input", func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typeName: "test-input", action: action, description: "test"}, nil
	})
	RegisterOutputConfigCreator("test-output", func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typeName: "test-output", action: action, description: "test"}, nil
	})

	inst, _ := NewInstance()

	// JSON with comments and trailing commas (hujson format)
	configJSON := `{
		// This is a comment
		"input": [{"type": "test-input", "action": "add", "args": {}}],
		"output": [{"type": "test-output", "action": "output", "args": {}},],
	}`

	if err := inst.InitConfigFromBytes([]byte(configJSON)); err != nil {
		t.Errorf("InitConfigFromBytes with comments error = %v", err)
	}
}

func TestInstanceInitConfigFromBytesInvalidJSON(t *testing.T) {
	inst, _ := NewInstance()
	if err := inst.InitConfigFromBytes([]byte(`{invalid json`)); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestInstanceInitConfigFromFile(t *testing.T) {
	origInputCache := inputConfigCreatorCache
	origOutputCache := outputConfigCreatorCache
	inputConfigCreatorCache = make(map[string]inputConfigCreator)
	outputConfigCreatorCache = make(map[string]outputConfigCreator)
	defer func() {
		inputConfigCreatorCache = origInputCache
		outputConfigCreatorCache = origOutputCache
	}()

	RegisterInputConfigCreator("test-input", func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typeName: "test-input", action: action, description: "test"}, nil
	})
	RegisterOutputConfigCreator("test-output", func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typeName: "test-output", action: action, description: "test"}, nil
	})

	// Write config to temp file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	configJSON := `{
		"input": [{"type": "test-input", "action": "add", "args": {}}],
		"output": [{"type": "test-output", "action": "output", "args": {}}]
	}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatal(err)
	}

	inst, _ := NewInstance()
	if err := inst.InitConfig(configPath); err != nil {
		t.Errorf("InitConfig error = %v", err)
	}
}

func TestInstanceInitConfigFromFileNotFound(t *testing.T) {
	inst, _ := NewInstance()
	if err := inst.InitConfig("/nonexistent/path/config.json"); err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestInstanceInitConfigFromURL(t *testing.T) {
	origInputCache := inputConfigCreatorCache
	origOutputCache := outputConfigCreatorCache
	inputConfigCreatorCache = make(map[string]inputConfigCreator)
	outputConfigCreatorCache = make(map[string]outputConfigCreator)
	defer func() {
		inputConfigCreatorCache = origInputCache
		outputConfigCreatorCache = origOutputCache
	}()

	RegisterInputConfigCreator("test-input", func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typeName: "test-input", action: action, description: "test"}, nil
	})
	RegisterOutputConfigCreator("test-output", func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typeName: "test-output", action: action, description: "test"}, nil
	})

	configJSON := `{
		"input": [{"type": "test-input", "action": "add", "args": {}}],
		"output": [{"type": "test-output", "action": "output", "args": {}}]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(configJSON))
	}))
	defer server.Close()

	inst, _ := NewInstance()
	if err := inst.InitConfig(server.URL); err != nil {
		t.Errorf("InitConfig from URL error = %v", err)
	}
}

func TestInstanceInitConfigFromURLError(t *testing.T) {
	inst, _ := NewInstance()
	if err := inst.InitConfig("http://invalid-host-that-does-not-exist.example.com"); err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestInstanceRunInputMultiple(t *testing.T) {
	inst, _ := NewInstance()

	callOrder := make([]string, 0)

	ic1 := &mockInputConverter{
		typeName: "test1", action: ActionAdd, description: "test1",
		inputFn: func(c Container) (Container, error) {
			callOrder = append(callOrder, "input1")
			return c, nil
		},
	}
	ic2 := &mockInputConverter{
		typeName: "test2", action: ActionAdd, description: "test2",
		inputFn: func(c Container) (Container, error) {
			callOrder = append(callOrder, "input2")
			return c, nil
		},
	}

	inst.AddInput(ic1)
	inst.AddInput(ic2)

	container := NewContainer()
	if err := inst.RunInput(container); err != nil {
		t.Errorf("RunInput error = %v", err)
	}

	if len(callOrder) != 2 || callOrder[0] != "input1" || callOrder[1] != "input2" {
		t.Errorf("unexpected call order: %v", callOrder)
	}
}

func TestInstanceRunOutputMultiple(t *testing.T) {
	inst, _ := NewInstance()

	callOrder := make([]string, 0)

	oc1 := &mockOutputConverter{
		typeName: "test1", action: ActionOutput, description: "test1",
		outputFn: func(c Container) error {
			callOrder = append(callOrder, "output1")
			return nil
		},
	}
	oc2 := &mockOutputConverter{
		typeName: "test2", action: ActionOutput, description: "test2",
		outputFn: func(c Container) error {
			callOrder = append(callOrder, "output2")
			return nil
		},
	}

	inst.AddOutput(oc1)
	inst.AddOutput(oc2)

	container := NewContainer()
	if err := inst.RunOutput(container); err != nil {
		t.Errorf("RunOutput error = %v", err)
	}

	if len(callOrder) != 2 || callOrder[0] != "output1" || callOrder[1] != "output2" {
		t.Errorf("unexpected call order: %v", callOrder)
	}
}

func TestInstanceRunInputDirectError(t *testing.T) {
	inst, _ := NewInstance()

	ic := &mockInputConverter{
		typeName: "test", action: ActionAdd, description: "test",
		inputFn: func(c Container) (Container, error) {
			return nil, errors.New("run input error")
		},
	}

	inst.AddInput(ic)

	container := NewContainer()
	if err := inst.RunInput(container); err == nil {
		t.Error("expected RunInput error")
	}
}

func TestInstanceRunOutputDirectError(t *testing.T) {
	inst, _ := NewInstance()

	oc := &mockOutputConverter{
		typeName: "test", action: ActionOutput, description: "test",
		outputFn: func(c Container) error {
			return errors.New("run output error")
		},
	}

	inst.AddOutput(oc)

	container := NewContainer()
	if err := inst.RunOutput(container); err == nil {
		t.Error("expected RunOutput error")
	}
}
