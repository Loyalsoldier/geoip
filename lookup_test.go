package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestLookupRequiresSingleFileURI(t *testing.T) {
	if format := os.Getenv("GEOIP_TEST_LOOKUP_FORMAT"); format != "" {
		getInputForLookup(format, "", os.Getenv("GEOIP_TEST_LOOKUP_URI"), "")
		return
	}

	for format := range supportedInputFormatsDisallowedDirFlag {
		for _, uri := range []string{"", " \t "} {
			t.Run(format+"/"+uri, func(t *testing.T) {
				cmd := exec.Command(os.Args[0], "-test.run=^TestLookupRequiresSingleFileURI$")
				cmd.Env = append(os.Environ(), "GEOIP_TEST_LOOKUP_FORMAT="+format, "GEOIP_TEST_LOOKUP_URI="+uri)
				output, err := cmd.CombinedOutput()
				exitErr, ok := err.(*exec.ExitError)
				if !ok || exitErr.ExitCode() != 1 || !strings.Contains(string(output), "missing uri") {
					t.Fatalf("expected missing URI validation, got error %v and output %s", err, output)
				}
			})
		}
	}
}

func TestLookupInputSources(t *testing.T) {
	for format := range supportedInputFormats {
		t.Run(format+"/file", func(t *testing.T) {
			input := getInputForLookup(format, "true", "input-file", "")
			if !strings.EqualFold(input.GetType(), format) {
				t.Fatalf("got input type %q, want %q", input.GetType(), format)
			}
		})
		if !supportedInputFormatsDisallowedDirFlag[format] {
			t.Run(format+"/directory", func(t *testing.T) {
				input := getInputForLookup(format, "", "", "input-directory")
				if !strings.EqualFold(input.GetType(), format) {
					t.Fatalf("got input type %q, want %q", input.GetType(), format)
				}
			})
		}
	}
}
