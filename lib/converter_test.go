package lib

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestRegisterInputConverter(t *testing.T) {
	testName := "test_input_conv_" + t.Name()
	mockConv := &mockInputConverter{
		typeName:    testName,
		action:      ActionAdd,
		description: "Test input converter",
	}

	err := RegisterInputConverter(testName, mockConv)
	if err != nil {
		t.Fatalf("RegisterInputConverter failed: %v", err)
	}

	// Test registering duplicate
	err = RegisterInputConverter(testName, mockConv)
	if err != ErrDuplicatedConverter {
		t.Errorf("RegisterInputConverter duplicate error = %v, want %v", err, ErrDuplicatedConverter)
	}
}

func TestRegisterInputConverter_TrimSpace(t *testing.T) {
	testName := "  test_input_conv_space_" + t.Name() + "  "
	mockConv := &mockInputConverter{
		typeName:    testName,
		action:      ActionAdd,
		description: "Test input converter",
	}

	err := RegisterInputConverter(testName, mockConv)
	if err != nil {
		t.Fatalf("RegisterInputConverter failed: %v", err)
	}

	// Test registering duplicate with trimmed name
	err = RegisterInputConverter("test_input_conv_space_"+t.Name(), mockConv)
	if err != ErrDuplicatedConverter {
		t.Errorf("RegisterInputConverter should detect duplicate after trim")
	}
}

func TestRegisterOutputConverter(t *testing.T) {
	testName := "test_output_conv_" + t.Name()
	mockConv := &mockOutputConverter{
		typeName:    testName,
		action:      ActionOutput,
		description: "Test output converter",
	}

	err := RegisterOutputConverter(testName, mockConv)
	if err != nil {
		t.Fatalf("RegisterOutputConverter failed: %v", err)
	}

	// Test registering duplicate
	err = RegisterOutputConverter(testName, mockConv)
	if err != ErrDuplicatedConverter {
		t.Errorf("RegisterOutputConverter duplicate error = %v, want %v", err, ErrDuplicatedConverter)
	}
}

func TestRegisterOutputConverter_TrimSpace(t *testing.T) {
	testName := "  test_output_conv_space_" + t.Name() + "  "
	mockConv := &mockOutputConverter{
		typeName:    testName,
		action:      ActionOutput,
		description: "Test output converter",
	}

	err := RegisterOutputConverter(testName, mockConv)
	if err != nil {
		t.Fatalf("RegisterOutputConverter failed: %v", err)
	}

	// Test registering duplicate with trimmed name
	err = RegisterOutputConverter("test_output_conv_space_"+t.Name(), mockConv)
	if err != ErrDuplicatedConverter {
		t.Errorf("RegisterOutputConverter should detect duplicate after trim")
	}
}

func TestListInputConverter(t *testing.T) {
	// Register a converter to ensure there's at least one
	testName := "list_input_conv_" + t.Name()
	mockConv := &mockInputConverter{
		typeName:    testName,
		action:      ActionAdd,
		description: "List test input converter",
	}
	RegisterInputConverter(testName, mockConv)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ListInputConverter()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if len(output) == 0 {
		t.Error("ListInputConverter should produce output")
	}
}

func TestListOutputConverter(t *testing.T) {
	// Register a converter to ensure there's at least one
	testName := "list_output_conv_" + t.Name()
	mockConv := &mockOutputConverter{
		typeName:    testName,
		action:      ActionOutput,
		description: "List test output converter",
	}
	RegisterOutputConverter(testName, mockConv)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ListOutputConverter()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if len(output) == 0 {
		t.Error("ListOutputConverter should produce output")
	}
}
