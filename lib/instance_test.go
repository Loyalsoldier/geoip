package lib

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const validConfig = `{
	// comment
	"input": [{"type": "mockIn", "action": "add"},],
	"output": [{"type": "mockOut"}],
}`

func TestInstanceInitConfig(t *testing.T) {
	withConfigCreators(t)
	registerMockCreators(t)

	dir := t.TempDir()
	goodFile := filepath.Join(dir, "good.json")
	if err := os.WriteFile(goodFile, []byte(validConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := newTestServer(t)

	tests := []struct {
		name    string
		file    string
		wantErr bool
		wantIn  int
		wantOut int
	}{
		{"local file", "  " + goodFile + "  ", false, 1, 1},
		{"missing local file", filepath.Join(dir, "missing.json"), true, 0, 0},
		{"remote http non-200", srv.URL + "/notfound", true, 0, 0},
		{"remote HTTP invalid json", srv.URL + "/ok", true, 0, 0},
		{"remote https unreachable", "HTTPS://127.0.0.1:0/x", true, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst, _ := NewInstance()
			err := inst.InitConfig(tt.file)
			assertErr(t, err, nil, tt.wantErr)
			in := inst.(*instance)
			if len(in.input) != tt.wantIn || len(in.output) != tt.wantOut {
				t.Fatalf("got %d inputs %d outputs, want %d %d", len(in.input), len(in.output), tt.wantIn, tt.wantOut)
			}
		})
	}
}

func TestInstanceInitConfigFromBytes(t *testing.T) {
	withConfigCreators(t)
	registerMockCreators(t)

	tests := []struct {
		name    string
		content string
		wantErr bool
		wantIn  int
		wantOut int
	}{
		{"hujson", validConfig, false, 1, 1},
		{"empty object", `{}`, false, 0, 0},
		{"malformed", `{"input": [`, true, 0, 0},
		{"bad converter", `{"input":[{"type":"nope","action":"add"}]}`, true, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst, err := NewInstance()
			if err != nil {
				t.Fatal(err)
			}
			err = inst.InitConfigFromBytes([]byte(tt.content))
			assertErr(t, err, nil, tt.wantErr)
			in := inst.(*instance)
			if len(in.input) != tt.wantIn || len(in.output) != tt.wantOut {
				t.Fatalf("got %d inputs %d outputs, want %d %d", len(in.input), len(in.output), tt.wantIn, tt.wantOut)
			}
		})
	}
}

func TestInstanceAddReset(t *testing.T) {
	inst, _ := NewInstance()
	in := inst.(*instance)
	inst.AddInput(&mockInputConverter{})
	inst.AddInput(&mockInputConverter{})
	inst.AddOutput(&mockOutputConverter{})
	if len(in.input) != 2 || len(in.output) != 1 {
		t.Fatalf("got %d inputs %d outputs", len(in.input), len(in.output))
	}
	inst.ResetInput()
	if len(in.input) != 0 || len(in.output) != 1 {
		t.Fatalf("after ResetInput: %d inputs %d outputs", len(in.input), len(in.output))
	}
	inst.ResetOutput()
	if len(in.output) != 0 {
		t.Fatalf("after ResetOutput: %d outputs", len(in.output))
	}
}

func TestInstanceRun(t *testing.T) {
	errIn := errors.New("input error")
	errOut := errors.New("output error")

	tests := []struct {
		name    string
		inputs  []InputConverter
		outputs []*mockOutputConverter
		wantErr error
		anyErr  bool
	}{
		{"no input", nil, []*mockOutputConverter{{}}, nil, true},
		{"no output", []InputConverter{&mockInputConverter{}}, nil, nil, true},
		{"success", []InputConverter{
			&mockInputConverter{fn: func(c Container) (Container, error) {
				e := NewEntry("test")
				_ = e.AddPrefix("1.1.1.1")
				return c, c.Add(e)
			}},
			&mockInputConverter{},
		}, []*mockOutputConverter{{}, {}}, nil, false},
		{"input fails", []InputConverter{&mockInputConverter{err: errIn}}, []*mockOutputConverter{{}}, errIn, true},
		{"output fails", []InputConverter{&mockInputConverter{}}, []*mockOutputConverter{{err: errOut}, {}}, errOut, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst, _ := NewInstance()
			for _, ic := range tt.inputs {
				inst.AddInput(ic)
			}
			for _, oc := range tt.outputs {
				inst.AddOutput(oc)
			}
			err := inst.Run()
			assertErr(t, err, tt.wantErr, tt.anyErr)
			if !tt.anyErr {
				for i, oc := range tt.outputs {
					if oc.called != 1 {
						t.Errorf("output %d called %d times", i, oc.called)
					}
				}
			}
			if tt.wantErr == errOut && tt.outputs[1].called != 0 {
				t.Error("second output should not be called after first fails")
			}
		})
	}
}
