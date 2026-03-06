package lib

import (
	"encoding/json"
	"testing"
)

func TestRegisterInputConfigCreator(t *testing.T) {
	resetConfigCreators()

	if err := RegisterInputConfigCreator("sample", func(a Action, data json.RawMessage) (InputConverter, error) {
		return mockInputConverter{typ: "sample", action: a}, nil
	}); err != nil {
		t.Fatalf("RegisterInputConfigCreator() error = %v", err)
	}

	if err := RegisterInputConfigCreator("sample", nil); err == nil {
		t.Fatalf("expected error for duplicated creator")
	}
}

func TestCreateInputConfig(t *testing.T) {
	resetConfigCreators()

	if _, err := createInputConfig("unknown", ActionAdd, nil); err == nil {
		t.Fatalf("expected error for unknown config type")
	}

	_ = RegisterInputConfigCreator("known", func(a Action, data json.RawMessage) (InputConverter, error) {
		return mockInputConverter{typ: "known", action: a}, nil
	})

	cfg, err := createInputConfig("known", ActionRemove, nil)
	if err != nil {
		t.Fatalf("createInputConfig() error = %v", err)
	}
	if cfg.GetAction() != ActionRemove || cfg.GetType() != "known" {
		t.Fatalf("unexpected converter: %v %v", cfg.GetType(), cfg.GetAction())
	}
}

func TestRegisterOutputConfigCreator(t *testing.T) {
	resetConfigCreators()

	if err := RegisterOutputConfigCreator("sample", func(a Action, data json.RawMessage) (OutputConverter, error) {
		return mockOutputConverter{typ: "sample", action: a}, nil
	}); err != nil {
		t.Fatalf("RegisterOutputConfigCreator() error = %v", err)
	}

	if err := RegisterOutputConfigCreator("sample", nil); err == nil {
		t.Fatalf("expected error for duplicated creator")
	}
}

func TestCreateOutputConfig(t *testing.T) {
	resetConfigCreators()

	if _, err := createOutputConfig("unknown", ActionAdd, nil); err == nil {
		t.Fatalf("expected error for unknown config type")
	}

	_ = RegisterOutputConfigCreator("known", func(a Action, data json.RawMessage) (OutputConverter, error) {
		return mockOutputConverter{typ: "known", action: a}, nil
	})

	cfg, err := createOutputConfig("known", ActionOutput, nil)
	if err != nil {
		t.Fatalf("createOutputConfig() error = %v", err)
	}
	if cfg.GetAction() != ActionOutput || cfg.GetType() != "known" {
		t.Fatalf("unexpected converter: %v %v", cfg.GetType(), cfg.GetAction())
	}
}

func TestInputConvConfigUnmarshalJSON(t *testing.T) {
	resetConfigCreators()
	_ = RegisterInputConfigCreator("stub", func(a Action, data json.RawMessage) (InputConverter, error) {
		return mockInputConverter{typ: "stub", action: a}, nil
	})

	jsonData := []byte(`{"type":"stub","action":"add","args":{}}`)
	var cfg inputConvConfig
	if err := cfg.UnmarshalJSON(jsonData); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if cfg.iType != "stub" || cfg.action != ActionAdd {
		t.Fatalf("unexpected values: %v %v", cfg.iType, cfg.action)
	}

	if err := cfg.UnmarshalJSON([]byte(`{"type":"stub","action":"invalid"}`)); err == nil {
		t.Fatalf("expected error for invalid action")
	}

	if err := cfg.UnmarshalJSON([]byte(`{"type":"unknown","action":"add"}`)); err == nil {
		t.Fatalf("expected error for unknown type")
	}

	if err := cfg.UnmarshalJSON([]byte(`{`)); err == nil {
		t.Fatalf("expected json error")
	}
}

func TestOutputConvConfigUnmarshalJSON(t *testing.T) {
	resetConfigCreators()
	_ = RegisterOutputConfigCreator("stub", func(a Action, data json.RawMessage) (OutputConverter, error) {
		return mockOutputConverter{typ: "stub", action: a}, nil
	})

	jsonData := []byte(`{"type":"stub","args":{}}`)
	var cfg outputConvConfig
	if err := cfg.UnmarshalJSON(jsonData); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if cfg.iType != "stub" || cfg.action != ActionOutput {
		t.Fatalf("unexpected values: %v %v", cfg.iType, cfg.action)
	}

	if err := cfg.UnmarshalJSON([]byte(`{"type":"stub","action":"invalid"}`)); err == nil {
		t.Fatalf("expected error for invalid action")
	}

	if err := cfg.UnmarshalJSON([]byte(`{"type":"unknown","action":"add"}`)); err == nil {
		t.Fatalf("expected error for unknown type")
	}

	if err := cfg.UnmarshalJSON([]byte(`{`)); err == nil {
		t.Fatalf("expected json error")
	}
}
