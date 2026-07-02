package lib

import (
	"encoding/json"
	"testing"
)

func TestRegisterInputConfigCreator(t *testing.T) {
	// Save original state
	originalCache := inputConfigCreatorCache
	defer func() { inputConfigCreatorCache = originalCache }()
	
	// Reset cache for testing
	inputConfigCreatorCache = make(map[string]inputConfigCreator)

	creator := func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typ: "test"}, nil
	}

	// Test successful registration
	err := RegisterInputConfigCreator("test", creator)
	if err != nil {
		t.Errorf("RegisterInputConfigCreator() error = %v, want nil", err)
	}

	// Test duplicate registration
	err = RegisterInputConfigCreator("test", creator)
	if err == nil {
		t.Error("RegisterInputConfigCreator() should return error for duplicate")
	}

	// Test case insensitive registration
	err = RegisterInputConfigCreator("TEST", creator)
	if err == nil {
		t.Error("RegisterInputConfigCreator() should return error for duplicate (case insensitive)")
	}
}

func TestCreateInputConfig(t *testing.T) {
	// Save original state
	originalCache := inputConfigCreatorCache
	defer func() { inputConfigCreatorCache = originalCache }()
	
	// Reset cache for testing
	inputConfigCreatorCache = make(map[string]inputConfigCreator)

	creator := func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typ: "test"}, nil
	}

	RegisterInputConfigCreator("test", creator)

	tests := []struct {
		name    string
		id      string
		action  Action
		wantErr bool
	}{
		{
			name:    "valid type",
			id:      "test",
			action:  ActionAdd,
			wantErr: false,
		},
		{
			name:    "valid type uppercase",
			id:      "TEST",
			action:  ActionAdd,
			wantErr: false,
		},
		{
			name:    "unknown type",
			id:      "unknown",
			action:  ActionAdd,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createInputConfig(tt.id, tt.action, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("createInputConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("createInputConfig() returned nil converter")
			}
		})
	}
}

func TestRegisterOutputConfigCreator(t *testing.T) {
	// Save original state
	originalCache := outputConfigCreatorCache
	defer func() { outputConfigCreatorCache = originalCache }()
	
	// Reset cache for testing
	outputConfigCreatorCache = make(map[string]outputConfigCreator)

	creator := func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typ: "test"}, nil
	}

	// Test successful registration
	err := RegisterOutputConfigCreator("test", creator)
	if err != nil {
		t.Errorf("RegisterOutputConfigCreator() error = %v, want nil", err)
	}

	// Test duplicate registration
	err = RegisterOutputConfigCreator("test", creator)
	if err == nil {
		t.Error("RegisterOutputConfigCreator() should return error for duplicate")
	}
}

func TestCreateOutputConfig(t *testing.T) {
	// Save original state
	originalCache := outputConfigCreatorCache
	defer func() { outputConfigCreatorCache = originalCache }()
	
	// Reset cache for testing
	outputConfigCreatorCache = make(map[string]outputConfigCreator)

	creator := func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typ: "test"}, nil
	}

	RegisterOutputConfigCreator("test", creator)

	tests := []struct {
		name    string
		id      string
		action  Action
		wantErr bool
	}{
		{
			name:    "valid type",
			id:      "test",
			action:  ActionOutput,
			wantErr: false,
		},
		{
			name:    "unknown type",
			id:      "unknown",
			action:  ActionOutput,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createOutputConfig(tt.id, tt.action, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("createOutputConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("createOutputConfig() returned nil converter")
			}
		})
	}
}

func TestInputConvConfig_UnmarshalJSON(t *testing.T) {
	// Save original state
	originalCache := inputConfigCreatorCache
	defer func() { inputConfigCreatorCache = originalCache }()
	
	// Reset cache for testing
	inputConfigCreatorCache = make(map[string]inputConfigCreator)

	creator := func(action Action, data json.RawMessage) (InputConverter, error) {
		return &mockInputConverter{typ: "test"}, nil
	}
	RegisterInputConfigCreator("test", creator)

	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "valid config",
			json:    `{"type": "test", "action": "add", "args": {}}`,
			wantErr: false,
		},
		{
			name:    "invalid action",
			json:    `{"type": "test", "action": "invalid", "args": {}}`,
			wantErr: true,
		},
		{
			name:    "unknown type",
			json:    `{"type": "unknown", "action": "add", "args": {}}`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			json:    `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg inputConvConfig
			err := json.Unmarshal([]byte(tt.json), &cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("inputConvConfig.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOutputConvConfig_UnmarshalJSON(t *testing.T) {
	// Save original state
	originalCache := outputConfigCreatorCache
	defer func() { outputConfigCreatorCache = originalCache }()
	
	// Reset cache for testing
	outputConfigCreatorCache = make(map[string]outputConfigCreator)

	creator := func(action Action, data json.RawMessage) (OutputConverter, error) {
		return &mockOutputConverter{typ: "test"}, nil
	}
	RegisterOutputConfigCreator("test", creator)

	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "valid config",
			json:    `{"type": "test", "action": "output", "args": {}}`,
			wantErr: false,
		},
		{
			name:    "default action",
			json:    `{"type": "test", "args": {}}`,
			wantErr: false,
		},
		{
			name:    "invalid action",
			json:    `{"type": "test", "action": "invalid", "args": {}}`,
			wantErr: true,
		},
		{
			name:    "unknown type",
			json:    `{"type": "unknown", "action": "output", "args": {}}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg outputConvConfig
			err := json.Unmarshal([]byte(tt.json), &cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("outputConvConfig.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
