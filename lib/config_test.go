package lib

import (
	"encoding/json"
	"strings"
	"testing"
)

const (
	mockConfigInputType  = "mockConfigInput"
	mockConfigOutputType = "mockConfigOutput"
)

// registerMockConfigCreators registers mock config creators and removes them
// once the test finishes.
func registerMockConfigCreators(t *testing.T) {
	t.Helper()

	err := RegisterInputConfigCreator(mockConfigInputType, func(action Action, data json.RawMessage) (InputConverter, error) {
		args, err := parseMockArgs(data)
		if err != nil {
			return nil, err
		}
		return &mockInputConverter{typ: mockConfigInputType, action: action, cidrs: args.CIDRs}, nil
	})
	if err != nil {
		t.Fatalf("RegisterInputConfigCreator failed: %v", err)
	}

	err = RegisterOutputConfigCreator(mockConfigOutputType, func(action Action, data json.RawMessage) (OutputConverter, error) {
		if _, err := parseMockArgs(data); err != nil {
			return nil, err
		}
		return &mockOutputConverter{typ: mockConfigOutputType, action: action}, nil
	})
	if err != nil {
		t.Fatalf("RegisterOutputConfigCreator failed: %v", err)
	}

	t.Cleanup(func() {
		delete(inputConfigCreatorCache, strings.ToLower(mockConfigInputType))
		delete(outputConfigCreatorCache, strings.ToLower(mockConfigOutputType))
	})
}

func TestRegisterConfigCreator(t *testing.T) {
	inputCreator := func(Action, json.RawMessage) (InputConverter, error) { return &mockInputConverter{}, nil }
	outputCreator := func(Action, json.RawMessage) (OutputConverter, error) { return &mockOutputConverter{}, nil }

	tests := []struct {
		name     string
		register func(id string) error
		cleanup  func(id string)
	}{
		{
			name:     "input",
			register: func(id string) error { return RegisterInputConfigCreator(id, inputCreator) },
			cleanup:  func(id string) { delete(inputConfigCreatorCache, strings.ToLower(id)) },
		},
		{
			name:     "output",
			register: func(id string) error { return RegisterOutputConfigCreator(id, outputCreator) },
			cleanup:  func(id string) { delete(outputConfigCreatorCache, strings.ToLower(id)) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const id = "TestRegisterConfigCreator"
			t.Cleanup(func() { tt.cleanup(id) })

			if err := tt.register(id); err != nil {
				t.Fatalf("first registration failed: %v", err)
			}

			// IDs are case-insensitive, so registering again must fail.
			for _, dup := range []string{id, strings.ToLower(id), strings.ToUpper(id)} {
				err := tt.register(dup)
				if err == nil || err.Error() != "config creator has already been registered" {
					t.Fatalf("duplicated registration of %q: error = %v", dup, err)
				}
			}
		})
	}
}

func TestCreateConfig(t *testing.T) {
	registerMockConfigCreators(t)

	type created interface {
		Typer
		Actioner
	}

	kinds := []struct {
		name   string
		baseID string
		create func(id string, action Action, data json.RawMessage) (created, error)
	}{
		{
			name:   "input",
			baseID: mockConfigInputType,
			create: func(id string, action Action, data json.RawMessage) (created, error) {
				conv, err := createInputConfig(id, action, data)
				if conv == nil {
					return nil, err
				}
				return conv, err
			},
		},
		{
			name:   "output",
			baseID: mockConfigOutputType,
			create: func(id string, action Action, data json.RawMessage) (created, error) {
				conv, err := createOutputConfig(id, action, data)
				if conv == nil {
					return nil, err
				}
				return conv, err
			},
		},
	}

	tests := []struct {
		name    string
		id      func(base string) string
		action  Action
		data    string
		wantErr string
	}{
		{name: "exact id", id: func(b string) string { return b }, action: ActionAdd},
		{name: "uppercase id", id: strings.ToUpper, action: ActionRemove},
		{name: "lowercase id", id: strings.ToLower, action: ActionOutput},
		{name: "with args", id: func(b string) string { return b }, action: ActionAdd, data: `{"cidrs":["1.1.1.1"]}`},
		{name: "unknown id", id: func(string) string { return "unknown" }, action: ActionAdd, wantErr: "unknown config type"},
		{name: "creator error", id: func(b string) string { return b }, action: ActionAdd, data: `{"fail":true}`, wantErr: "mock creator failure"},
		{name: "invalid args", id: func(b string) string { return b }, action: ActionAdd, data: `{`, wantErr: "unexpected end of JSON input"},
	}

	for _, kind := range kinds {
		for _, tt := range tests {
			t.Run(kind.name+" "+tt.name, func(t *testing.T) {
				conv, err := kind.create(tt.id(kind.baseID), tt.action, json.RawMessage(tt.data))
				if tt.wantErr != "" {
					if err == nil || err.Error() != tt.wantErr {
						t.Fatalf("error = %v, want %q", err, tt.wantErr)
					}
					if conv != nil {
						t.Fatal("converter should be nil on error")
					}
					return
				}
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if conv.GetType() != kind.baseID || conv.GetAction() != tt.action {
					t.Fatalf("got type %q action %q, want type %q action %q", conv.GetType(), conv.GetAction(), kind.baseID, tt.action)
				}
			})
		}
	}
}

func TestInputConvConfigUnmarshalJSON(t *testing.T) {
	registerMockConfigCreators(t)

	tests := []struct {
		name       string
		data       string
		wantAction Action
		wantCIDRs  []string
		wantErr    string
	}{
		{
			name:       "add action",
			data:       `{"type":"mockConfigInput","action":"add","args":{"cidrs":["1.1.1.1"]}}`,
			wantAction: ActionAdd,
			wantCIDRs:  []string{"1.1.1.1"},
		},
		{
			name:       "remove action and case-insensitive type",
			data:       `{"type":"MOCKCONFIGINPUT","action":"remove"}`,
			wantAction: ActionRemove,
		},
		{
			name:       "output action",
			data:       `{"type":"mockConfigInput","action":"output"}`,
			wantAction: ActionOutput,
		},
		{name: "invalid JSON", data: `{"type":`, wantErr: "unexpected end of JSON input"},
		{name: "wrong JSON type", data: `[]`, wantErr: "cannot unmarshal array"},
		{name: "missing action", data: `{"type":"mockConfigInput"}`, wantErr: "invalid action  in type mockConfigInput"},
		{name: "invalid action", data: `{"type":"mockConfigInput","action":"bad"}`, wantErr: "invalid action bad in type mockConfigInput"},
		{name: "unknown type", data: `{"type":"unknown","action":"add"}`, wantErr: "unknown config type"},
		{name: "creator error", data: `{"type":"mockConfigInput","action":"add","args":{"fail":true}}`, wantErr: "mock creator failure"},
		{name: "invalid args", data: `{"type":"mockConfigInput","action":"add","args":{"cidrs":"x"}}`, wantErr: "cannot unmarshal string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg inputConvConfig
			err := json.Unmarshal([]byte(tt.data), &cfg)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				if cfg.converter != nil {
					t.Fatal("converter should be nil on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.iType != mockConfigInputType {
				t.Fatalf("iType = %q, want %q", cfg.iType, mockConfigInputType)
			}
			if cfg.action != tt.wantAction {
				t.Fatalf("action = %q, want %q", cfg.action, tt.wantAction)
			}
			mock, ok := cfg.converter.(*mockInputConverter)
			if !ok {
				t.Fatalf("converter type = %T", cfg.converter)
			}
			assertStringSlice(t, mock.cidrs, tt.wantCIDRs)
		})
	}
}

func TestOutputConvConfigUnmarshalJSON(t *testing.T) {
	registerMockConfigCreators(t)

	tests := []struct {
		name       string
		data       string
		wantAction Action
		wantErr    string
	}{
		{name: "default action", data: `{"type":"mockConfigOutput"}`, wantAction: ActionOutput},
		{name: "empty action", data: `{"type":"mockConfigOutput","action":""}`, wantAction: ActionOutput},
		{name: "output action", data: `{"type":"mockConfigOutput","action":"output"}`, wantAction: ActionOutput},
		{name: "add action", data: `{"type":"MockConfigOutput","action":"add"}`, wantAction: ActionAdd},
		{name: "with args", data: `{"type":"mockConfigOutput","args":{"cidrs":[]}}`, wantAction: ActionOutput},
		{name: "invalid JSON", data: `{`, wantErr: "unexpected end of JSON input"},
		{name: "wrong JSON type", data: `"str"`, wantErr: "cannot unmarshal string"},
		{name: "invalid action", data: `{"type":"mockConfigOutput","action":"bad"}`, wantErr: "invalid action bad in type mockConfigOutput"},
		{name: "unknown type", data: `{"type":"unknown"}`, wantErr: "unknown config type"},
		{name: "creator error", data: `{"type":"mockConfigOutput","args":{"fail":true}}`, wantErr: "mock creator failure"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg outputConvConfig
			err := json.Unmarshal([]byte(tt.data), &cfg)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				if cfg.converter != nil {
					t.Fatal("converter should be nil on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.iType != mockConfigOutputType {
				t.Fatalf("iType = %q, want %q", cfg.iType, mockConfigOutputType)
			}
			if cfg.action != tt.wantAction {
				t.Fatalf("action = %q, want %q", cfg.action, tt.wantAction)
			}
			if _, ok := cfg.converter.(*mockOutputConverter); !ok {
				t.Fatalf("converter type = %T", cfg.converter)
			}
		})
	}
}
