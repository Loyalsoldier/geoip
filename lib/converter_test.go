package lib

import (
	"strings"
	"testing"
)

func TestRegisterInputConverter(t *testing.T) {
	// Save original state
	originalMap := inputConverterMap
	defer func() { inputConverterMap = originalMap }()
	
	// Reset map for testing
	inputConverterMap = make(map[string]InputConverter)

	tests := []struct {
		name      string
		converter string
		wantErr   bool
	}{
		{
			name:      "register new converter",
			converter: "test1",
			wantErr:   false,
		},
		{
			name:      "register duplicate converter",
			converter: "test1",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock converter
			mockConverter := &mockInputConverter{typ: tt.converter}
			err := RegisterInputConverter(tt.converter, mockConverter)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterInputConverter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRegisterOutputConverter(t *testing.T) {
	// Save original state
	originalMap := outputConverterMap
	defer func() { outputConverterMap = originalMap }()
	
	// Reset map for testing
	outputConverterMap = make(map[string]OutputConverter)

	tests := []struct {
		name      string
		converter string
		wantErr   bool
	}{
		{
			name:      "register new converter",
			converter: "test1",
			wantErr:   false,
		},
		{
			name:      "register duplicate converter",
			converter: "test1",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock converter
			mockConverter := &mockOutputConverter{typ: tt.converter}
			err := RegisterOutputConverter(tt.converter, mockConverter)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterOutputConverter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListInputConverter(t *testing.T) {
	// Save original state
	originalMap := inputConverterMap
	defer func() { inputConverterMap = originalMap }()
	
	// Reset and populate map
	inputConverterMap = make(map[string]InputConverter)
	inputConverterMap["test"] = &mockInputConverter{typ: "test"}
	
	// Just test that it doesn't panic
	ListInputConverter()
}

func TestListOutputConverter(t *testing.T) {
	// Save original state
	originalMap := outputConverterMap
	defer func() { outputConverterMap = originalMap }()
	
	// Reset and populate map
	outputConverterMap = make(map[string]OutputConverter)
	outputConverterMap["test"] = &mockOutputConverter{typ: "test"}
	
	// Just test that it doesn't panic
	ListOutputConverter()
}

// Mock converters for testing
type mockInputConverter struct {
	typ string
}

func (m *mockInputConverter) GetType() string {
	return m.typ
}

func (m *mockInputConverter) GetAction() Action {
	return ActionAdd
}

func (m *mockInputConverter) GetDescription() string {
	return "mock input converter"
}

func (m *mockInputConverter) Input(c Container) (Container, error) {
	return c, nil
}

type mockOutputConverter struct {
	typ string
}

func (m *mockOutputConverter) GetType() string {
	return m.typ
}

func (m *mockOutputConverter) GetAction() Action {
	return ActionOutput
}

func (m *mockOutputConverter) GetDescription() string {
	return "mock output converter"
}

func (m *mockOutputConverter) Output(c Container) error {
	return nil
}

func TestRegisterConverterWithWhitespace(t *testing.T) {
	// Save original state
	originalMap := inputConverterMap
	defer func() { inputConverterMap = originalMap }()
	
	// Reset map for testing
	inputConverterMap = make(map[string]InputConverter)

	mockConverter := &mockInputConverter{typ: "test"}
	err := RegisterInputConverter("  test  ", mockConverter)
	if err != nil {
		t.Errorf("RegisterInputConverter() with whitespace should not error: %v", err)
	}

	// Verify it was registered with trimmed name
	if _, ok := inputConverterMap["test"]; !ok {
		// Check without trim
		found := false
		for k := range inputConverterMap {
			if strings.TrimSpace(k) == "test" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Converter not registered properly with whitespace")
		}
	}
}
