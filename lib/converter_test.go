package lib

import (
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestRegisterAndListInputConverter(t *testing.T) {
	orig := inputConverterMap
	inputConverterMap = make(map[string]InputConverter)
	t.Cleanup(func() { inputConverterMap = orig })

	tests := []struct {
		name    string
		key     string
		wantErr error
	}{
		{"register b", "b", nil},
		{"register a with spaces", "  a  ", nil},
		{"duplicate b", "b", ErrDuplicatedConverter},
		{"duplicate a trimmed", "a", ErrDuplicatedConverter},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RegisterInputConverter(tt.key, &mockInputConverter{desc: "desc-" + strings.TrimSpace(tt.key)})
			assertErr(t, err, tt.wantErr, tt.wantErr != nil)
		})
	}

	out := captureStdout(t, ListInputConverter)
	want := "All available input formats:\n  - a (desc-a)\n  - b (desc-b)\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

func TestRegisterAndListOutputConverter(t *testing.T) {
	orig := outputConverterMap
	outputConverterMap = make(map[string]OutputConverter)
	t.Cleanup(func() { outputConverterMap = orig })

	tests := []struct {
		name    string
		key     string
		wantErr error
	}{
		{"register y", "y", nil},
		{"register x with spaces", " x ", nil},
		{"duplicate y", "y", ErrDuplicatedConverter},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RegisterOutputConverter(tt.key, &mockOutputConverter{desc: "desc-" + strings.TrimSpace(tt.key)})
			assertErr(t, err, tt.wantErr, tt.wantErr != nil)
		})
	}

	out := captureStdout(t, ListOutputConverter)
	want := "All available output formats:\n  - x (desc-x)\n  - y (desc-y)\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}
