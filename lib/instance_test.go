package lib

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func setupConfigCreators() {
	resetConfigCreators()
	_ = RegisterInputConfigCreator("stubinput", func(a Action, data json.RawMessage) (InputConverter, error) {
		return mockInputConverter{typ: "stubinput", action: a}, nil
	})
	_ = RegisterOutputConfigCreator("stuboutput", func(a Action, data json.RawMessage) (OutputConverter, error) {
		return mockOutputConverter{typ: "stuboutput", action: a}, nil
	})
}

func TestInitConfigFromBytes(t *testing.T) {
	setupConfigCreators()
	inst, _ := NewInstance()

	content := []byte(`
	{
		// comment
		"input": [
			{"type": "stubinput", "action": "add", "args": {}},
		],
		"output": [
			{"type": "stuboutput", "args": {}},
		],
	}
	`)

	if err := inst.InitConfigFromBytes(content); err != nil {
		t.Fatalf("InitConfigFromBytes() error = %v", err)
	}
	if len(inst.(*instance).input) != 1 || len(inst.(*instance).output) != 1 {
		t.Fatalf("expected converters to be loaded")
	}

	if err := inst.InitConfigFromBytes([]byte(`{`)); err == nil {
		t.Fatalf("expected JSON error")
	}
}

func TestInitConfigLocalAndRemote(t *testing.T) {
	setupConfigCreators()
	inst, _ := NewInstance()

	data := `{"input":[{"type":"stubinput","action":"add","args":{}}],"output":[{"type":"stuboutput","args":{}}]}`

	tmp, err := os.CreateTemp("", "config*.json")
	if err != nil {
		t.Fatalf("CreateTemp error: %v", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(data); err != nil {
		t.Fatalf("write temp file error: %v", err)
	}
	tmp.Close()

	if err := inst.InitConfig(tmp.Name()); err != nil {
		t.Fatalf("InitConfig(local) error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(data))
	}))
	defer server.Close()

	if err := inst.InitConfig(server.URL); err != nil {
		t.Fatalf("InitConfig(remote) error = %v", err)
	}

	if err := inst.InitConfig("non-existent.json"); err == nil {
		t.Fatalf("expected error for missing file")
	}

	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer errorServer.Close()
	if err := inst.InitConfig(errorServer.URL); err == nil {
		t.Fatalf("expected error for remote failure")
	}
}

func TestInstanceRun(t *testing.T) {
	inst, _ := NewInstance()

	if err := inst.Run(); err == nil {
		t.Fatalf("expected error when no input/output configured")
	}

	inputErr := errors.New("input fail")
	inst.AddInput(mockInputConverter{inputFn: func(c Container) (Container, error) {
		return c, inputErr
	}})
	inst.AddOutput(mockOutputConverter{})
	if err := inst.Run(); !errors.Is(err, inputErr) {
		t.Fatalf("expected input error, got %v", err)
	}

	inst.ResetInput()
	inst.ResetOutput()

	outputErr := errors.New("output fail")
	inst.AddInput(mockInputConverter{inputFn: func(c Container) (Container, error) {
		return c, nil
	}})
	inst.AddOutput(mockOutputConverter{outFn: func(c Container) error {
		return outputErr
	}})
	if err := inst.Run(); !errors.Is(err, outputErr) {
		t.Fatalf("expected output error, got %v", err)
	}

	inst.ResetInput()
	inst.ResetOutput()

	inputCalled := false
	outputCalled := false
	inst.AddInput(mockInputConverter{inputFn: func(c Container) (Container, error) {
		inputCalled = true
		return c, nil
	}})
	inst.AddOutput(mockOutputConverter{outFn: func(c Container) error {
		outputCalled = true
		return nil
	}})

	if err := inst.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !inputCalled || !outputCalled {
		t.Fatalf("expected both input and output to be called")
	}
}
