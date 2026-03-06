package lib

import (
	"bytes"
	"os"
	"testing"
)

func TestRegisterInputConverter(t *testing.T) {
	origMap := inputConverterMap
	inputConverterMap = make(map[string]InputConverter)
	defer func() { inputConverterMap = origMap }()

	mock := &mockInputConverter{typeName: "test-ic", action: ActionAdd, description: "Test Input"}

	// Register successfully
	if err := RegisterInputConverter("test-ic", mock); err != nil {
		t.Errorf("RegisterInputConverter error = %v", err)
	}

	// Duplicate registration
	if err := RegisterInputConverter("test-ic", mock); err != ErrDuplicatedConverter {
		t.Errorf("expected ErrDuplicatedConverter, got %v", err)
	}
}

func TestRegisterOutputConverter(t *testing.T) {
	origMap := outputConverterMap
	outputConverterMap = make(map[string]OutputConverter)
	defer func() { outputConverterMap = origMap }()

	mock := &mockOutputConverter{typeName: "test-oc", action: ActionOutput, description: "Test Output"}

	// Register successfully
	if err := RegisterOutputConverter("test-oc", mock); err != nil {
		t.Errorf("RegisterOutputConverter error = %v", err)
	}

	// Duplicate registration
	if err := RegisterOutputConverter("test-oc", mock); err != ErrDuplicatedConverter {
		t.Errorf("expected ErrDuplicatedConverter, got %v", err)
	}
}

func TestListInputConverter(t *testing.T) {
	origMap := inputConverterMap
	inputConverterMap = make(map[string]InputConverter)
	defer func() { inputConverterMap = origMap }()

	mock := &mockInputConverter{typeName: "test-ic", action: ActionAdd, description: "Test Input"}
	RegisterInputConverter("test-ic", mock)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ListInputConverter()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if len(output) == 0 {
		t.Error("ListInputConverter should produce output")
	}
}

func TestListOutputConverter(t *testing.T) {
	origMap := outputConverterMap
	outputConverterMap = make(map[string]OutputConverter)
	defer func() { outputConverterMap = origMap }()

	mock := &mockOutputConverter{typeName: "test-oc", action: ActionOutput, description: "Test Output"}
	RegisterOutputConverter("test-oc", mock)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ListOutputConverter()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if len(output) == 0 {
		t.Error("ListOutputConverter should produce output")
	}
}
