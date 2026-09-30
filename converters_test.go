package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestLookupConverterConstruction(t *testing.T) {
	for format := range supportedInputFormats {
		t.Run(format, func(t *testing.T) {
			input := getInputForLookup(format, "true", "input.file", "")
			if !strings.EqualFold(input.GetType(), format) || input.GetAction() != lib.ActionAdd || input.GetDescription() == "" {
				t.Fatalf("unexpected converter metadata: %s, %s, %s", input.GetType(), input.GetAction(), input.GetDescription())
			}
		})
	}

	for _, format := range []string{"text", "clashRuleSet", "clashRuleSetClassical", "surgeRuleSet", "mihomoMRS", "singboxSRS"} {
		t.Run(format+"-directory", func(t *testing.T) {
			input := getInputForLookup(format, "true", "", t.TempDir())
			if input.GetType() != format {
				t.Fatalf("unexpected converter type %q", input.GetType())
			}
		})
	}
}

func TestLookupTextDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("192.0.2.0/24\n"), 0600); err != nil {
		t.Fatal(err)
	}
	input := getInputForLookup("text", "true", "", dir)
	container, err := input.Input(lib.NewContainer())
	if err != nil {
		t.Fatal(err)
	}
	got := captureConverterOutput(t, func() error {
		return getOutputForLookup("192.0.2.1", "test").Output(container)
	})
	if strings.TrimSpace(got) != "test" {
		t.Fatalf("unexpected lookup output %q", got)
	}
}

func TestMergeConverters(t *testing.T) {
	input := getInputForMerge()
	if input.GetType() != "stdin" || input.GetAction() != lib.ActionAdd {
		t.Fatal("incorrect merge input metadata")
	}
	container := lib.NewContainer()
	entry := lib.NewEntry("temp")
	for _, prefix := range []string{"192.0.2.0/24", "2001:db8::/32"} {
		if err := entry.AddPrefix(prefix); err != nil {
			t.Fatal(err)
		}
	}
	if err := container.Add(entry); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		ipType string
		want   string
	}{
		{"", "192.0.2.0/24\n2001:db8::/32\n"},
		{"ipv4", "192.0.2.0/24\n"},
		{"ipv6", "2001:db8::/32\n"},
	} {
		t.Run(tc.ipType, func(t *testing.T) {
			got := captureConverterOutput(t, func() error {
				return getOutputForMerge(tc.ipType).Output(container)
			})
			if got != tc.want {
				t.Fatalf("merge output %q, want %q", got, tc.want)
			}
		})
	}
}

func captureConverterOutput(t *testing.T, output func() error) string {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	previous := os.Stdout
	os.Stdout = file
	defer func() { os.Stdout = previous }()
	if err := output(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
