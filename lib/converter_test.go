package lib

import (
	"strings"
	"testing"
)

func TestRegisterInputConverter(t *testing.T) {
	resetInputConverters()
	if err := RegisterInputConverter("json", mockInputConverter{typ: "json", action: ActionAdd}); err != nil {
		t.Fatalf("RegisterInputConverter() error = %v", err)
	}
	if err := RegisterInputConverter("json", mockInputConverter{}); err != ErrDuplicatedConverter {
		t.Fatalf("expected ErrDuplicatedConverter, got %v", err)
	}
}

func TestRegisterOutputConverter(t *testing.T) {
	resetOutputConverters()
	if err := RegisterOutputConverter("txt", mockOutputConverter{typ: "txt", action: ActionOutput}); err != nil {
		t.Fatalf("RegisterOutputConverter() error = %v", err)
	}
	if err := RegisterOutputConverter("txt", mockOutputConverter{}); err != ErrDuplicatedConverter {
		t.Fatalf("expected ErrDuplicatedConverter, got %v", err)
	}
}

func TestListConverters(t *testing.T) {
	resetInputConverters()
	resetOutputConverters()

	_ = RegisterInputConverter("b", mockInputConverter{typ: "b", desc: "second"})
	_ = RegisterInputConverter("a", mockInputConverter{typ: "a", desc: "first"})
	_ = RegisterOutputConverter("x", mockOutputConverter{typ: "x", desc: "out"})

	out := captureOutput(t, func() {
		ListInputConverter()
		ListOutputConverter()
	})

	if !strings.Contains(out, "a") || !strings.Contains(out, "b") || !strings.Contains(out, "x") {
		t.Fatalf("unexpected output: %s", out)
	}
}
