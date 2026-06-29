package lib

import (
	"bytes"
	"io"
	"os"
	"testing"
)

type mockInputConverter struct {
	typ      string
	action   Action
	desc     string
	err      error
	inputFn  func(Container) (Container, error)
}

func (m mockInputConverter) GetType() string {
	return m.typ
}

func (m mockInputConverter) GetAction() Action {
	return m.action
}

func (m mockInputConverter) GetDescription() string {
	if m.desc != "" {
		return m.desc
	}
	return "mock input converter"
}

func (m mockInputConverter) Input(c Container) (Container, error) {
	if m.inputFn != nil {
		return m.inputFn(c)
	}
	return c, m.err
}

type mockOutputConverter struct {
	typ     string
	action  Action
	desc    string
	err     error
	outFn   func(Container) error
}

func (m mockOutputConverter) GetType() string {
	return m.typ
}

func (m mockOutputConverter) GetAction() Action {
	return m.action
}

func (m mockOutputConverter) GetDescription() string {
	if m.desc != "" {
		return m.desc
	}
	return "mock output converter"
}

func (m mockOutputConverter) Output(c Container) error {
	if m.outFn != nil {
		return m.outFn(c)
	}
	return m.err
}

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()

	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = stdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()

	return buf.String()
}

func resetInputConverters() {
	inputConverterMap = make(map[string]InputConverter)
}

func resetOutputConverters() {
	outputConverterMap = make(map[string]OutputConverter)
}

func resetConfigCreators() {
	inputConfigCreatorCache = make(map[string]inputConfigCreator)
	outputConfigCreatorCache = make(map[string]outputConfigCreator)
}
