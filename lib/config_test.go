package lib

import (
	"encoding/json"
	"errors"
	"testing"
)

// withConfigCreators isolates the global config creator caches for a test.
func withConfigCreators(t *testing.T) {
	t.Helper()
	origIn, origOut := inputConfigCreatorCache, outputConfigCreatorCache
	inputConfigCreatorCache = make(map[string]inputConfigCreator)
	outputConfigCreatorCache = make(map[string]outputConfigCreator)
	t.Cleanup(func() {
		inputConfigCreatorCache, outputConfigCreatorCache = origIn, origOut
	})
}

var errCreator = errors.New("creator error")

func registerMockCreators(t *testing.T) {
	t.Helper()
	if err := RegisterInputConfigCreator("MockIn", func(a Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typ: "mockIn", action: a}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := RegisterInputConfigCreator("failIn", func(Action, json.RawMessage) (InputConverter, error) {
		return nil, errCreator
	}); err != nil {
		t.Fatal(err)
	}
	if err := RegisterOutputConfigCreator("MockOut", func(a Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typ: "mockOut", action: a}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := RegisterOutputConfigCreator("failOut", func(Action, json.RawMessage) (OutputConverter, error) {
		return nil, errCreator
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRegisterConfigCreatorDuplicate(t *testing.T) {
	withConfigCreators(t)
	registerMockCreators(t)

	tests := []struct {
		name string
		fn   func() error
	}{
		{"input duplicate case-insensitive", func() error {
			return RegisterInputConfigCreator("MOCKIN", nil)
		}},
		{"output duplicate case-insensitive", func() error {
			return RegisterOutputConfigCreator("mockout", nil)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); err == nil {
				t.Fatal("expected duplicate registration error")
			}
		})
	}
}

func TestCreateConfig(t *testing.T) {
	withConfigCreators(t)
	registerMockCreators(t)

	t.Run("input", func(t *testing.T) {
		tests := []struct {
			name    string
			id      string
			wantErr bool
		}{
			{"known", "mockin", false},
			{"known upper", "MOCKIN", false},
			{"unknown", "nope", true},
			{"creator fails", "failIn", true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c, err := createInputConfig(tt.id, ActionAdd, nil)
				assertErr(t, err, nil, tt.wantErr)
				if !tt.wantErr && (c.GetType() != "mockIn" || c.GetAction() != ActionAdd) {
					t.Fatalf("unexpected converter %+v", c)
				}
			})
		}
	})

	t.Run("output", func(t *testing.T) {
		tests := []struct {
			name    string
			id      string
			wantErr bool
		}{
			{"known", "mockout", false},
			{"unknown", "nope", true},
			{"creator fails", "FAILOUT", true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c, err := createOutputConfig(tt.id, ActionOutput, nil)
				assertErr(t, err, nil, tt.wantErr)
				if !tt.wantErr && (c.GetType() != "mockOut" || c.GetAction() != ActionOutput) {
					t.Fatalf("unexpected converter %+v", c)
				}
			})
		}
	})
}

func TestInputConvConfigUnmarshalJSON(t *testing.T) {
	withConfigCreators(t)
	registerMockCreators(t)

	tests := []struct {
		name       string
		data       string
		wantErr    bool
		wantAction Action
	}{
		{"valid add", `{"type":"mockIn","action":"add","args":{}}`, false, ActionAdd},
		{"valid remove", `{"type":"mockIn","action":"remove"}`, false, ActionRemove},
		{"malformed json", `{`, true, ""},
		{"wrong field type", `{"type":123,"action":"add"}`, true, ""},
		{"missing action", `{"type":"mockIn"}`, true, ""},
		{"invalid action", `{"type":"mockIn","action":"bogus"}`, true, ""},
		{"unknown type", `{"type":"nope","action":"add"}`, true, ""},
		{"creator error", `{"type":"failIn","action":"add"}`, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c inputConvConfig
			err := json.Unmarshal([]byte(tt.data), &c)
			assertErr(t, err, nil, tt.wantErr)
			if tt.wantErr {
				return
			}
			if c.iType != "mockIn" || c.action != tt.wantAction || c.converter == nil {
				t.Fatalf("unexpected config %+v", c)
			}
		})
	}
}

func TestOutputConvConfigUnmarshalJSON(t *testing.T) {
	withConfigCreators(t)
	registerMockCreators(t)

	tests := []struct {
		name       string
		data       string
		wantErr    bool
		wantAction Action
	}{
		{"default action is output", `{"type":"mockOut"}`, false, ActionOutput},
		{"explicit action", `{"type":"mockOut","action":"add"}`, false, ActionAdd},
		{"malformed json", `[`, true, ""},
		{"wrong field type", `{"type":"mockOut","action":1}`, true, ""},
		{"invalid action", `{"type":"mockOut","action":"bogus"}`, true, ""},
		{"unknown type", `{"type":"nope"}`, true, ""},
		{"creator error", `{"type":"failOut"}`, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c outputConvConfig
			err := json.Unmarshal([]byte(tt.data), &c)
			assertErr(t, err, nil, tt.wantErr)
			if tt.wantErr {
				return
			}
			if c.iType != "mockOut" || c.action != tt.wantAction || c.converter == nil {
				t.Fatalf("unexpected config %+v", c)
			}
		})
	}
}
