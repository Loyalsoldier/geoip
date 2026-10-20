package lib

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testConfig = `
// JSON with comments and trailing commas is supported
{
  "input": [
    {
      "type": "mockConfigInput",
      "action": "add",
      "args": { "cidrs": ["192.168.0.3/24", "192.168.1.5/24", "2001:db8::/33", "2001:db8:8000::/33",] },
    },
  ],
  "output": [
    { "type": "mockConfigOutput" }, /* default action is output */
  ],
}
`

func TestNewInstance(t *testing.T) {
	t.Parallel()

	inst, err := NewInstance()
	if err != nil {
		t.Fatalf("NewInstance() error: %v", err)
	}
	i, ok := inst.(*instance)
	if !ok {
		t.Fatalf("NewInstance() returned %T", inst)
	}
	if i.input == nil || len(i.input) != 0 {
		t.Fatalf("input = %v, want empty non-nil slice", i.input)
	}
	if i.output == nil || len(i.output) != 0 {
		t.Fatalf("output = %v, want empty non-nil slice", i.output)
	}
}

func TestInstanceAddAndReset(t *testing.T) {
	t.Parallel()

	inst, _ := NewInstance()
	i := inst.(*instance)

	in1, in2 := &mockInputConverter{typ: "in1"}, &mockInputConverter{typ: "in2"}
	out1, out2 := &mockOutputConverter{typ: "out1"}, &mockOutputConverter{typ: "out2"}

	inst.AddInput(in1)
	inst.AddInput(in2)
	inst.AddOutput(out1)
	inst.AddOutput(out2)

	if len(i.input) != 2 || i.input[0] != in1 || i.input[1] != in2 {
		t.Fatalf("input = %v, want [in1 in2]", i.input)
	}
	if len(i.output) != 2 || i.output[0] != out1 || i.output[1] != out2 {
		t.Fatalf("output = %v, want [out1 out2]", i.output)
	}

	inst.ResetInput()
	if i.input == nil || len(i.input) != 0 {
		t.Fatalf("after ResetInput, input = %v", i.input)
	}
	if len(i.output) != 2 {
		t.Fatalf("ResetInput should not touch output, got %v", i.output)
	}

	inst.ResetOutput()
	if i.output == nil || len(i.output) != 0 {
		t.Fatalf("after ResetOutput, output = %v", i.output)
	}
}

func assertTestConfigLoaded(t *testing.T, inst Instance) {
	t.Helper()
	i := inst.(*instance)
	if len(i.input) != 1 || len(i.output) != 1 {
		t.Fatalf("got %d inputs and %d outputs, want 1 and 1", len(i.input), len(i.output))
	}
	in, ok := i.input[0].(*mockInputConverter)
	if !ok {
		t.Fatalf("input type = %T", i.input[0])
	}
	if in.action != ActionAdd {
		t.Fatalf("input action = %q, want %q", in.action, ActionAdd)
	}
	assertStringSlice(t, in.cidrs, []string{"192.168.0.3/24", "192.168.1.5/24", "2001:db8::/33", "2001:db8:8000::/33"})

	out, ok := i.output[0].(*mockOutputConverter)
	if !ok {
		t.Fatalf("output type = %T", i.output[0])
	}
	if out.action != ActionOutput {
		t.Fatalf("output action = %q, want %q", out.action, ActionOutput)
	}
}

func TestInstanceInitConfigFromBytes(t *testing.T) {
	registerMockConfigCreators(t)

	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{name: "JSON with comments and trailing commas", content: testConfig},
		{name: "invalid JSON", content: `{"input": [`, wantErr: "unexpected end of JSON input"},
		{name: "wrong JSON type", content: `[]`, wantErr: "cannot unmarshal array"},
		{name: "unknown input type", content: `{"input":[{"type":"unknown","action":"add"}]}`, wantErr: "unknown config type"},
		{name: "invalid input action", content: `{"input":[{"type":"mockConfigInput","action":"bad"}]}`, wantErr: "invalid action bad"},
		{name: "unknown output type", content: `{"output":[{"type":"unknown"}]}`, wantErr: "unknown config type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst, _ := NewInstance()
			err := inst.InitConfigFromBytes([]byte(tt.content))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertTestConfigLoaded(t, inst)
		})
	}
}

func TestInstanceInitConfig(t *testing.T) {
	registerMockConfigCreators(t)

	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configFile, []byte(testConfig), 0644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/config.json":
			w.Write([]byte(testConfig))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	hostPort := strings.TrimPrefix(server.URL, "http://")

	tests := []struct {
		name    string
		uri     string
		wantErr string
	}{
		{name: "local file", uri: configFile},
		{name: "local file with spaces", uri: "  " + configFile + "\t"},
		{name: "missing local file", uri: filepath.Join(dir, "missing.json"), wantErr: "no such file or directory"},
		{name: "remote file", uri: server.URL + "/config.json"},
		{name: "remote file with uppercase scheme", uri: "HTTP://" + hostPort + "/config.json"},
		{name: "remote file not found", uri: server.URL + "/missing.json", wantErr: "404 Not Found"},
		{name: "https scheme is detected as remote", uri: "HTTPS://" + hostPort + "/config.json", wantErr: "http: server gave HTTP response to HTTPS client"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst, _ := NewInstance()
			err := inst.InitConfig(tt.uri)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertTestConfigLoaded(t, inst)
		})
	}
}

func TestInstanceRunInput(t *testing.T) {
	t.Parallel()

	errInput := errors.New("input failure")

	tests := []struct {
		name      string
		inputs    []*mockInputConverter
		wantErr   error
		wantCalls []int
		want      map[string][]string
	}{
		{
			name:      "no input",
			wantCalls: []int{},
			want:      map[string][]string{},
		},
		{
			name: "multiple inputs",
			inputs: []*mockInputConverter{
				{typ: "a", cidrs: []string{"192.168.0.3/24", "2001:db8::/33"}},
				{typ: "a", cidrs: []string{"192.168.1.5/24", "2001:db8:8000::/33"}},
				{typ: "b", cidrs: []string{"::ffff:10.0.0.0/120"}},
			},
			wantCalls: []int{1, 1, 1},
			want: map[string][]string{
				"A": {"192.168.0.0/23", "2001:db8::/32"},
				"B": {"10.0.0.0/24"},
			},
		},
		{
			name: "stops at first error",
			inputs: []*mockInputConverter{
				{typ: "a", cidrs: []string{"1.1.1.1"}},
				{typ: "b", err: errInput},
				{typ: "c", cidrs: []string{"3.3.3.3"}},
			},
			wantErr:   errInput,
			wantCalls: []int{1, 1, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inst, _ := NewInstance()
			for _, in := range tt.inputs {
				inst.AddInput(in)
			}

			c := NewContainer()
			err := inst.RunInput(c)
			assertErrorIs(t, err, tt.wantErr)

			calls := make([]int, 0, len(tt.inputs))
			for _, in := range tt.inputs {
				calls = append(calls, in.calls)
			}
			if len(calls) != len(tt.wantCalls) {
				t.Fatalf("calls = %v, want %v", calls, tt.wantCalls)
			}
			for i := range calls {
				if calls[i] != tt.wantCalls[i] {
					t.Fatalf("calls = %v, want %v", calls, tt.wantCalls)
				}
			}

			if tt.wantErr != nil {
				return
			}
			if c.Len() != len(tt.want) {
				t.Fatalf("Len() = %d, want %d", c.Len(), len(tt.want))
			}
			for name, want := range tt.want {
				assertStringSlice(t, containerText(t, c, name), want)
			}
		})
	}
}

func TestInstanceRunOutput(t *testing.T) {
	t.Parallel()

	errOutput := errors.New("output failure")

	tests := []struct {
		name      string
		outputs   []*mockOutputConverter
		wantErr   error
		wantCalls []int
	}{
		{name: "no output", wantCalls: []int{}},
		{
			name:      "multiple outputs",
			outputs:   []*mockOutputConverter{{typ: "a"}, {typ: "b"}},
			wantCalls: []int{1, 1},
		},
		{
			name:      "stops at first error",
			outputs:   []*mockOutputConverter{{typ: "a"}, {typ: "b", err: errOutput}, {typ: "c"}},
			wantErr:   errOutput,
			wantCalls: []int{1, 1, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inst, _ := NewInstance()
			for _, out := range tt.outputs {
				inst.AddOutput(out)
			}

			c := NewContainer()
			assertErrorIs(t, inst.RunOutput(c), tt.wantErr)

			if len(tt.outputs) != len(tt.wantCalls) {
				t.Fatalf("got %d outputs, want %d", len(tt.outputs), len(tt.wantCalls))
			}
			for i, out := range tt.outputs {
				if out.calls != tt.wantCalls[i] {
					t.Fatalf("output %d called %d times, want %d", i, out.calls, tt.wantCalls[i])
				}
				if out.calls > 0 && out.got != c {
					t.Fatalf("output %d received a different container", i)
				}
			}
		})
	}
}

func TestInstanceRun(t *testing.T) {
	t.Parallel()

	errInput := errors.New("input failure")
	errOutput := errors.New("output failure")

	tests := []struct {
		name       string
		input      *mockInputConverter
		output     *mockOutputConverter
		wantErr    string
		wantOutput bool
	}{
		{name: "no input and no output", wantErr: "input type and output type must be specified"},
		{name: "no output", input: &mockInputConverter{typ: "a"}, wantErr: "input type and output type must be specified"},
		{name: "no input", output: &mockOutputConverter{typ: "a"}, wantErr: "input type and output type must be specified"},
		{
			name:    "input error",
			input:   &mockInputConverter{typ: "a", err: errInput},
			output:  &mockOutputConverter{typ: "b"},
			wantErr: errInput.Error(),
		},
		{
			name:       "output error",
			input:      &mockInputConverter{typ: "a", cidrs: []string{"1.1.1.1"}},
			output:     &mockOutputConverter{typ: "b", err: errOutput},
			wantErr:    errOutput.Error(),
			wantOutput: true,
		},
		{
			name:       "success",
			input:      &mockInputConverter{typ: "a", cidrs: []string{"192.168.0.3/24", "192.168.1.5/24", "::ffff:10.0.0.1"}},
			output:     &mockOutputConverter{typ: "b"},
			wantOutput: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inst, _ := NewInstance()
			if tt.input != nil {
				inst.AddInput(tt.input)
			}
			if tt.output != nil {
				inst.AddOutput(tt.output)
			}

			err := inst.Run()
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.output == nil {
				return
			}
			if called := tt.output.calls > 0; called != tt.wantOutput {
				t.Fatalf("output called = %v, want %v", called, tt.wantOutput)
			}
			if tt.wantErr == "" {
				assertStringSlice(t, containerText(t, tt.output.got, "a"), []string{"10.0.0.1/32", "192.168.0.0/23"})
			}
		})
	}
}
