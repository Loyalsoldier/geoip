package lib

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// Mock converters for testing
type mockInputConverter struct {
	typ         string
	action      Action
	description string
}

func (m *mockInputConverter) GetType() string        { return m.typ }
func (m *mockInputConverter) GetAction() Action      { return m.action }
func (m *mockInputConverter) GetDescription() string { return m.description }
func (m *mockInputConverter) Input(c Container) (Container, error) {
	return c, nil
}

type mockOutputConverter struct {
	typ         string
	action      Action
	description string
}

func (m *mockOutputConverter) GetType() string        { return m.typ }
func (m *mockOutputConverter) GetAction() Action      { return m.action }
func (m *mockOutputConverter) GetDescription() string { return m.description }
func (m *mockOutputConverter) Output(c Container) error {
	return nil
}

func TestRegisterInputConverter(t *testing.T) {
	// Clear any existing converters
	inputConverterMap = make(map[string]InputConverter)

	converter := &mockInputConverter{
		typ:         "test",
		action:      ActionAdd,
		description: "Test input converter",
	}

	err := RegisterInputConverter("test", converter)
	if err != nil {
		t.Errorf("RegisterInputConverter() error = %v, want nil", err)
	}

	// Verify converter was registered
	if _, ok := inputConverterMap["test"]; !ok {
		t.Error("Converter not found in inputConverterMap")
	}
}

func TestRegisterInputConverter_Duplicate(t *testing.T) {
	// Clear any existing converters
	inputConverterMap = make(map[string]InputConverter)

	converter := &mockInputConverter{
		typ:         "test",
		action:      ActionAdd,
		description: "Test input converter",
	}

	// Register first time
	err := RegisterInputConverter("test", converter)
	if err != nil {
		t.Errorf("RegisterInputConverter() first call error = %v, want nil", err)
	}

	// Register duplicate
	err = RegisterInputConverter("test", converter)
	if err != ErrDuplicatedConverter {
		t.Errorf("RegisterInputConverter() duplicate error = %v, want %v", err, ErrDuplicatedConverter)
	}
}

func TestRegisterInputConverter_Trimming(t *testing.T) {
	// Clear any existing converters
	inputConverterMap = make(map[string]InputConverter)

	converter := &mockInputConverter{
		typ:         "test",
		action:      ActionAdd,
		description: "Test input converter",
	}

	// Register with spaces
	err := RegisterInputConverter("  test  ", converter)
	if err != nil {
		t.Errorf("RegisterInputConverter() error = %v, want nil", err)
	}

	// Verify trimmed name is used
	if _, ok := inputConverterMap["test"]; !ok {
		t.Error("Converter not found with trimmed name in inputConverterMap")
	}
}

func TestRegisterOutputConverter(t *testing.T) {
	// Clear any existing converters
	outputConverterMap = make(map[string]OutputConverter)

	converter := &mockOutputConverter{
		typ:         "test",
		action:      ActionOutput,
		description: "Test output converter",
	}

	err := RegisterOutputConverter("test", converter)
	if err != nil {
		t.Errorf("RegisterOutputConverter() error = %v, want nil", err)
	}

	// Verify converter was registered
	if _, ok := outputConverterMap["test"]; !ok {
		t.Error("Converter not found in outputConverterMap")
	}
}

func TestRegisterOutputConverter_Duplicate(t *testing.T) {
	// Clear any existing converters
	outputConverterMap = make(map[string]OutputConverter)

	converter := &mockOutputConverter{
		typ:         "test",
		action:      ActionOutput,
		description: "Test output converter",
	}

	// Register first time
	err := RegisterOutputConverter("test", converter)
	if err != nil {
		t.Errorf("RegisterOutputConverter() first call error = %v, want nil", err)
	}

	// Register duplicate
	err = RegisterOutputConverter("test", converter)
	if err != ErrDuplicatedConverter {
		t.Errorf("RegisterOutputConverter() duplicate error = %v, want %v", err, ErrDuplicatedConverter)
	}
}

func TestRegisterOutputConverter_Trimming(t *testing.T) {
	// Clear any existing converters
	outputConverterMap = make(map[string]OutputConverter)

	converter := &mockOutputConverter{
		typ:         "test",
		action:      ActionOutput,
		description: "Test output converter",
	}

	// Register with spaces
	err := RegisterOutputConverter("  test  ", converter)
	if err != nil {
		t.Errorf("RegisterOutputConverter() error = %v, want nil", err)
	}

	// Verify trimmed name is used
	if _, ok := outputConverterMap["test"]; !ok {
		t.Error("Converter not found with trimmed name in outputConverterMap")
	}
}

func TestListInputConverter(t *testing.T) {
	// Clear and setup converters
	inputConverterMap = make(map[string]InputConverter)

	converter1 := &mockInputConverter{
		typ:         "test1",
		action:      ActionAdd,
		description: "Test input converter 1",
	}
	converter2 := &mockInputConverter{
		typ:         "test2",
		action:      ActionAdd,
		description: "Test input converter 2",
	}

	RegisterInputConverter("test1", converter1)
	RegisterInputConverter("test2", converter2)

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

	// Verify output contains expected strings
	if !strings.Contains(output, "All available input formats:") {
		t.Error("ListInputConverter() output missing header")
	}
	if !strings.Contains(output, "test1") {
		t.Error("ListInputConverter() output missing test1")
	}
	if !strings.Contains(output, "test2") {
		t.Error("ListInputConverter() output missing test2")
	}
	if !strings.Contains(output, "Test input converter 1") {
		t.Error("ListInputConverter() output missing description 1")
	}
	if !strings.Contains(output, "Test input converter 2") {
		t.Error("ListInputConverter() output missing description 2")
	}
}

func TestListInputConverter_Empty(t *testing.T) {
	// Clear converters
	inputConverterMap = make(map[string]InputConverter)

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

	// Verify output contains header
	if !strings.Contains(output, "All available input formats:") {
		t.Error("ListInputConverter() output missing header")
	}
}

func TestListOutputConverter(t *testing.T) {
	// Clear and setup converters
	outputConverterMap = make(map[string]OutputConverter)

	converter1 := &mockOutputConverter{
		typ:         "test1",
		action:      ActionOutput,
		description: "Test output converter 1",
	}
	converter2 := &mockOutputConverter{
		typ:         "test2",
		action:      ActionOutput,
		description: "Test output converter 2",
	}

	RegisterOutputConverter("test1", converter1)
	RegisterOutputConverter("test2", converter2)

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

	// Verify output contains expected strings
	if !strings.Contains(output, "All available output formats:") {
		t.Error("ListOutputConverter() output missing header")
	}
	if !strings.Contains(output, "test1") {
		t.Error("ListOutputConverter() output missing test1")
	}
	if !strings.Contains(output, "test2") {
		t.Error("ListOutputConverter() output missing test2")
	}
	if !strings.Contains(output, "Test output converter 1") {
		t.Error("ListOutputConverter() output missing description 1")
	}
	if !strings.Contains(output, "Test output converter 2") {
		t.Error("ListOutputConverter() output missing description 2")
	}
}

func TestListOutputConverter_Empty(t *testing.T) {
	// Clear converters
	outputConverterMap = make(map[string]OutputConverter)

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

	// Verify output contains header
	if !strings.Contains(output, "All available output formats:") {
		t.Error("ListOutputConverter() output missing header")
	}
}

func TestListConverters_Sorted(t *testing.T) {
	// Clear and setup converters in non-alphabetical order
	inputConverterMap = make(map[string]InputConverter)

	converterZ := &mockInputConverter{typ: "z", action: ActionAdd, description: "Z"}
	converterA := &mockInputConverter{typ: "a", action: ActionAdd, description: "A"}
	converterM := &mockInputConverter{typ: "m", action: ActionAdd, description: "M"}

	RegisterInputConverter("z", converterZ)
	RegisterInputConverter("a", converterA)
	RegisterInputConverter("m", converterM)

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

	// Find positions of each converter in output
	posA := strings.Index(output, "- a")
	posM := strings.Index(output, "- m")
	posZ := strings.Index(output, "- z")

	if posA == -1 || posM == -1 || posZ == -1 {
		t.Error("ListInputConverter() missing one or more converters")
	}

	// Verify alphabetical order
	if !(posA < posM && posM < posZ) {
		t.Error("ListInputConverter() not sorted alphabetically")
	}
}
