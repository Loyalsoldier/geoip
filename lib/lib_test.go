package lib

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

type mockConverter struct {
	typ      string
	action   Action
	desc     string
	inErr    error
	outErr   error
	inCalled int
	outCalls int
}

func (m *mockConverter) GetType() string        { return m.typ }
func (m *mockConverter) GetAction() Action      { return m.action }
func (m *mockConverter) GetDescription() string { return m.desc }
func (m *mockConverter) Input(c Container) (Container, error) {
	m.inCalled++
	return c, m.inErr
}
func (m *mockConverter) Output(Container) error {
	m.outCalls++
	return m.outErr
}

func TestIgnoreOptions(t *testing.T) {
	tests := []struct {
		in   IPType
		want IPType
		nil_ bool
	}{
		{IPv4, IPv6, false},
		{IPv6, IPv4, false},
		{"", "", true},
		{"other", "", true},
	}
	for _, tt := range tests {
		t.Run(string(tt.in), func(t *testing.T) {
			opt := GetIgnoreIPType(tt.in)
			if tt.nil_ {
				if opt != nil {
					t.Error("expected nil")
				}
				return
			}
			if opt() != tt.want {
				t.Errorf("got %s, want %s", opt(), tt.want)
			}
		})
	}
}

func TestWantedListExtendedUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		data      string
		wantSlice []string
		wantMap   map[string][]string
		wantErr   bool
	}{
		{"slice", `["a","b"]`, []string{"a", "b"}, map[string][]string{}, false},
		{"map", `{"x":["a"],"y":["b","c"]}`, []string{}, map[string][]string{"x": {"a"}, "y": {"b", "c"}}, false},
		{"invalid", `123`, nil, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w WantedListExtended
			err := json.Unmarshal([]byte(tt.data), &w)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v", err)
			}
			if tt.wantErr {
				return
			}
			compareStrings(t, w.TypeSlice, tt.wantSlice)
			if len(w.TypeMap) != len(tt.wantMap) {
				t.Fatalf("map len %d, want %d", len(w.TypeMap), len(tt.wantMap))
			}
			for k, v := range tt.wantMap {
				compareStrings(t, w.TypeMap[k], v)
			}
		})
	}

	t.Run("empty data", func(t *testing.T) {
		var w WantedListExtended
		if err := w.UnmarshalJSON(nil); err != nil || w.TypeSlice != nil || w.TypeMap != nil {
			t.Error("expected no-op")
		}
	})
}

func TestGetRemoteURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ok" {
			_, _ = w.Write([]byte("hello"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{"ok", srv.URL + "/ok", "hello", false},
		{"404", srv.URL + "/missing", "", true},
		{"bad url", "http://\x00invalid", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := GetRemoteURLContent(tt.url)
			if (err != nil) != tt.wantErr || string(b) != tt.want {
				t.Errorf("content: %q %v", b, err)
			}
			rc, err := GetRemoteURLReader(tt.url)
			if (err != nil) != tt.wantErr {
				t.Fatalf("reader err: %v", err)
			}
			if err == nil {
				defer rc.Close()
				b, _ := io.ReadAll(rc)
				if string(b) != tt.want {
					t.Errorf("reader: %q", b)
				}
			}
		})
	}
}

func TestConverterRegistry(t *testing.T) {
	m := &mockConverter{typ: "mock", desc: "mock converter"}
	if err := RegisterInputConverter(" testMockIn ", m); err != nil {
		t.Fatal(err)
	}
	if err := RegisterInputConverter("testMockIn", m); !errors.Is(err, ErrDuplicatedConverter) {
		t.Errorf("err = %v", err)
	}
	if err := RegisterOutputConverter(" testMockOut ", m); err != nil {
		t.Fatal(err)
	}
	if err := RegisterOutputConverter("testMockOut", m); !errors.Is(err, ErrDuplicatedConverter) {
		t.Errorf("err = %v", err)
	}
	_ = RegisterInputConverter("testMockIn2", m)
	_ = RegisterOutputConverter("testMockOut2", m)
	ListInputConverter()
	ListOutputConverter()
}

func registerMockCreators(t *testing.T) {
	t.Helper()
	_ = RegisterInputConfigCreator("mockIn", func(a Action, data json.RawMessage) (InputConverter, error) {
		if string(data) == `"fail"` {
			return nil, errors.New("fail")
		}
		return &mockConverter{typ: "mockIn", action: a}, nil
	})
	_ = RegisterOutputConfigCreator("mockOut", func(a Action, data json.RawMessage) (OutputConverter, error) {
		if string(data) == `"fail"` {
			return nil, errors.New("fail")
		}
		return &mockConverter{typ: "mockOut", action: a}, nil
	})
}

func TestConfigCreatorRegistry(t *testing.T) {
	registerMockCreators(t)
	if err := RegisterInputConfigCreator("MOCKIN", nil); err == nil {
		t.Error("expected duplicate error")
	}
	if err := RegisterOutputConfigCreator("MOCKOUT", nil); err == nil {
		t.Error("expected duplicate error")
	}
	if _, err := createInputConfig("unknown", ActionAdd, nil); err == nil {
		t.Error("expected unknown error")
	}
	if _, err := createOutputConfig("unknown", ActionOutput, nil); err == nil {
		t.Error("expected unknown error")
	}
}

func TestInstanceInitConfigFromBytes(t *testing.T) {
	registerMockCreators(t)
	tests := []struct {
		name    string
		config  string
		wantIn  int
		wantOut int
		wantErr bool
	}{
		{"valid", `{"input":[{"type":"mockIn","action":"add"}],"output":[{"type":"mockOut"}]}`, 1, 1, false},
		{"hujson", "{\n// comment\n\"input\":[{\"type\":\"mockIn\",\"action\":\"remove\",},],\"output\":[{\"type\":\"mockOut\",\"action\":\"output\"}],}", 1, 1, false},
		{"invalid json", `{`, 0, 0, true},
		{"input invalid action", `{"input":[{"type":"mockIn","action":"bad"}]}`, 0, 0, true},
		{"input missing action", `{"input":[{"type":"mockIn"}]}`, 0, 0, true},
		{"input unknown type", `{"input":[{"type":"none","action":"add"}]}`, 0, 0, true},
		{"input bad shape", `{"input":[1]}`, 0, 0, true},
		{"input creator fail", `{"input":[{"type":"mockIn","action":"add","args":"fail"}]}`, 0, 0, true},
		{"output invalid action", `{"output":[{"type":"mockOut","action":"bad"}]}`, 0, 0, true},
		{"output unknown type", `{"output":[{"type":"none"}]}`, 0, 0, true},
		{"output bad shape", `{"output":[1]}`, 0, 0, true},
		{"output creator fail", `{"output":[{"type":"mockOut","args":"fail"}]}`, 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst, _ := NewInstance()
			err := inst.InitConfigFromBytes([]byte(tt.config))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v", err)
			}
			i := inst.(*instance)
			if !tt.wantErr && (len(i.input) != tt.wantIn || len(i.output) != tt.wantOut) {
				t.Errorf("got %d/%d", len(i.input), len(i.output))
			}
		})
	}
}

func TestInstanceInitConfig(t *testing.T) {
	registerMockCreators(t)
	content := `{"input":[{"type":"mockIn","action":"add"}],"output":[{"type":"mockOut"}]}`
	dir := t.TempDir()
	file := filepath.Join(dir, "config.json")
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(content))
	}))
	defer srv.Close()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"local file", " " + file + " ", false},
		{"remote", srv.URL, false},
		{"missing file", filepath.Join(dir, "missing.json"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst, _ := NewInstance()
			if err := inst.InitConfig(tt.path); (err != nil) != tt.wantErr {
				t.Errorf("err = %v", err)
			}
		})
	}
}

func TestInstanceRun(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name    string
		in      []*mockConverter
		out     []*mockConverter
		wantErr bool
	}{
		{"no input", nil, []*mockConverter{{}}, true},
		{"no output", []*mockConverter{{}}, nil, true},
		{"ok", []*mockConverter{{}, {}}, []*mockConverter{{}, {}}, false},
		{"input error", []*mockConverter{{inErr: boom}}, []*mockConverter{{}}, true},
		{"output error", []*mockConverter{{}}, []*mockConverter{{outErr: boom}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst, err := NewInstance()
			if err != nil {
				t.Fatal(err)
			}
			for _, m := range tt.in {
				inst.AddInput(m)
			}
			for _, m := range tt.out {
				inst.AddOutput(m)
			}
			if err := inst.Run(); (err != nil) != tt.wantErr {
				t.Fatalf("err = %v", err)
			}
		})
	}

	t.Run("reset", func(t *testing.T) {
		inst, _ := NewInstance()
		inst.AddInput(&mockConverter{})
		inst.AddOutput(&mockConverter{})
		inst.ResetInput()
		inst.ResetOutput()
		i := inst.(*instance)
		if len(i.input) != 0 || len(i.output) != 0 {
			t.Error("reset failed")
		}
	})
}
