package lib

import (
	"errors"
	"strings"
	"testing"
)

// swapConverterMaps replaces the global converter maps for the duration of
// the test and restores them afterwards.
func swapConverterMaps(t *testing.T, in map[string]InputConverter, out map[string]OutputConverter) {
	t.Helper()
	origIn, origOut := inputConverterMap, outputConverterMap
	inputConverterMap, outputConverterMap = in, out
	t.Cleanup(func() {
		inputConverterMap, outputConverterMap = origIn, origOut
	})
}

func TestRegisterConverter(t *testing.T) {
	swapConverterMaps(t, make(map[string]InputConverter), make(map[string]OutputConverter))

	tests := []struct {
		name     string
		register func(name string) error
		has      func(name string) bool
	}{
		{
			name:     "input",
			register: func(name string) error { return RegisterInputConverter(name, &mockInputConverter{}) },
			has:      func(name string) bool { _, ok := inputConverterMap[name]; return ok },
		},
		{
			name:     "output",
			register: func(name string) error { return RegisterOutputConverter(name, &mockOutputConverter{}) },
			has:      func(name string) bool { _, ok := outputConverterMap[name]; return ok },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := []struct {
				name    string
				wantErr error
				wantKey string
			}{
				{name: "  conv  ", wantKey: "conv"},
				{name: "conv", wantErr: ErrDuplicatedConverter},
				{name: "\tconv\n", wantErr: ErrDuplicatedConverter},
				{name: "CONV", wantKey: "CONV"}, // names are case-sensitive
			}
			for _, step := range steps {
				err := tt.register(step.name)
				if !errors.Is(err, step.wantErr) {
					t.Fatalf("register(%q) error = %v, want %v", step.name, err, step.wantErr)
				}
				if step.wantKey != "" && !tt.has(step.wantKey) {
					t.Fatalf("register(%q) did not store key %q", step.name, step.wantKey)
				}
			}
		})
	}
}

func TestListConverter(t *testing.T) {
	tests := []struct {
		name    string
		inputs  map[string]InputConverter
		outputs map[string]OutputConverter
		list    func()
		want    string
	}{
		{
			name:    "empty input list",
			inputs:  map[string]InputConverter{},
			outputs: map[string]OutputConverter{},
			list:    ListInputConverter,
			want:    "All available input formats:\n",
		},
		{
			name: "sorted input list",
			inputs: map[string]InputConverter{
				"text":    &mockInputConverter{desc: "Convert text"},
				"cutter":  &mockInputConverter{desc: "Remove data"},
				"private": &mockInputConverter{desc: "Private CIDRs"},
			},
			outputs: map[string]OutputConverter{},
			list:    ListInputConverter,
			want: "All available input formats:\n" +
				"  - cutter (Remove data)\n" +
				"  - private (Private CIDRs)\n" +
				"  - text (Convert text)\n",
		},
		{
			name:    "empty output list",
			inputs:  map[string]InputConverter{},
			outputs: map[string]OutputConverter{},
			list:    ListOutputConverter,
			want:    "All available output formats:\n",
		},
		{
			name:   "sorted output list",
			inputs: map[string]InputConverter{},
			outputs: map[string]OutputConverter{
				"stdout": &mockOutputConverter{desc: "Print"},
				"lookup": &mockOutputConverter{desc: "Lookup IP"},
			},
			list: ListOutputConverter,
			want: "All available output formats:\n" +
				"  - lookup (Lookup IP)\n" +
				"  - stdout (Print)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			swapConverterMaps(t, tt.inputs, tt.outputs)
			got := captureStdout(t, tt.list)
			if got != tt.want {
				t.Fatalf("output mismatch:\ngot:\n%s\nwant:\n%s", got, tt.want)
			}
			if lines := strings.Count(got, "\n"); lines != 1+len(tt.inputs)+len(tt.outputs) {
				t.Fatalf("got %d lines", lines)
			}
		})
	}
}
