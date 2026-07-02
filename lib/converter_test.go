package lib

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// mockInputConverter implements InputConverter for testing
type mockInputConverter struct {
	iType       string
	action      Action
	description string
}

func (m *mockInputConverter) GetType() string         { return m.iType }
func (m *mockInputConverter) GetAction() Action       { return m.action }
func (m *mockInputConverter) GetDescription() string  { return m.description }
func (m *mockInputConverter) Input(Container) (Container, error) { return nil, nil }

// mockOutputConverter implements OutputConverter for testing
type mockOutputConverter struct {
	iType       string
	action      Action
	description string
}

func (m *mockOutputConverter) GetType() string        { return m.iType }
func (m *mockOutputConverter) GetAction() Action      { return m.action }
func (m *mockOutputConverter) GetDescription() string { return m.description }
func (m *mockOutputConverter) Output(Container) error { return nil }

func TestRegisterInputConverter(t *testing.T) {
	// Store original state
	original := make(map[string]InputConverter)
	for k, v := range inputConverterMap {
		original[k] = v
	}
	defer func() {
		inputConverterMap = original
	}()

	// Clear for testing
	inputConverterMap = make(map[string]InputConverter)

	t.Run("success", func(t *testing.T) {
		conv := &mockInputConverter{
			iType:       "test-input",
			description: "Test input converter",
		}
		err := RegisterInputConverter("test-input", conv)
		if err != nil {
			t.Fatalf("RegisterInputConverter failed: %v", err)
		}
		if inputConverterMap["test-input"] != conv {
			t.Error("converter not registered correctly")
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		conv := &mockInputConverter{
			iType:       "dup-input",
			description: "Duplicate input converter",
		}
		err := RegisterInputConverter("dup-input", conv)
		if err != nil {
			t.Fatalf("first registration failed: %v", err)
		}
		err = RegisterInputConverter("dup-input", conv)
		if err != ErrDuplicatedConverter {
			t.Errorf("expected ErrDuplicatedConverter, got %v", err)
		}
	})

	t.Run("with spaces", func(t *testing.T) {
		conv := &mockInputConverter{
			iType:       "spaced-input",
			description: "Spaced input converter",
		}
		err := RegisterInputConverter("  spaced-input  ", conv)
		if err != nil {
			t.Fatalf("RegisterInputConverter failed: %v", err)
		}
		if inputConverterMap["spaced-input"] == nil {
			t.Error("converter not registered with trimmed name")
		}
	})
}

func TestRegisterOutputConverter(t *testing.T) {
	// Store original state
	original := make(map[string]OutputConverter)
	for k, v := range outputConverterMap {
		original[k] = v
	}
	defer func() {
		outputConverterMap = original
	}()

	// Clear for testing
	outputConverterMap = make(map[string]OutputConverter)

	t.Run("success", func(t *testing.T) {
		conv := &mockOutputConverter{
			iType:       "test-output",
			description: "Test output converter",
		}
		err := RegisterOutputConverter("test-output", conv)
		if err != nil {
			t.Fatalf("RegisterOutputConverter failed: %v", err)
		}
		if outputConverterMap["test-output"] != conv {
			t.Error("converter not registered correctly")
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		conv := &mockOutputConverter{
			iType:       "dup-output",
			description: "Duplicate output converter",
		}
		err := RegisterOutputConverter("dup-output", conv)
		if err != nil {
			t.Fatalf("first registration failed: %v", err)
		}
		err = RegisterOutputConverter("dup-output", conv)
		if err != ErrDuplicatedConverter {
			t.Errorf("expected ErrDuplicatedConverter, got %v", err)
		}
	})

	t.Run("with spaces", func(t *testing.T) {
		conv := &mockOutputConverter{
			iType:       "spaced-output",
			description: "Spaced output converter",
		}
		err := RegisterOutputConverter("  spaced-output  ", conv)
		if err != nil {
			t.Fatalf("RegisterOutputConverter failed: %v", err)
		}
		if outputConverterMap["spaced-output"] == nil {
			t.Error("converter not registered with trimmed name")
		}
	})
}

func TestListInputConverter(t *testing.T) {
	// Store original state
	original := make(map[string]InputConverter)
	for k, v := range inputConverterMap {
		original[k] = v
	}
	defer func() {
		inputConverterMap = original
	}()

	// Clear and add test converters
	inputConverterMap = make(map[string]InputConverter)
	inputConverterMap["alpha"] = &mockInputConverter{description: "Alpha desc"}
	inputConverterMap["beta"] = &mockInputConverter{description: "Beta desc"}

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

	if output == "" {
		t.Error("expected output, got empty string")
	}
	// Check that both converters are listed
	if !bytes.Contains([]byte(output), []byte("alpha")) {
		t.Error("expected 'alpha' in output")
	}
	if !bytes.Contains([]byte(output), []byte("beta")) {
		t.Error("expected 'beta' in output")
	}
}

func TestListOutputConverter(t *testing.T) {
	// Store original state
	original := make(map[string]OutputConverter)
	for k, v := range outputConverterMap {
		original[k] = v
	}
	defer func() {
		outputConverterMap = original
	}()

	// Clear and add test converters
	outputConverterMap = make(map[string]OutputConverter)
	outputConverterMap["gamma"] = &mockOutputConverter{description: "Gamma desc"}
	outputConverterMap["delta"] = &mockOutputConverter{description: "Delta desc"}

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

	if output == "" {
		t.Error("expected output, got empty string")
	}
	// Check that both converters are listed
	if !bytes.Contains([]byte(output), []byte("gamma")) {
		t.Error("expected 'gamma' in output")
	}
	if !bytes.Contains([]byte(output), []byte("delta")) {
		t.Error("expected 'delta' in output")
	}
}
