package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/plugin/maxmind"
	"github.com/Loyalsoldier/geoip/plugin/v2ray"
)

func TestLookupRejectsEmptyURI(t *testing.T) {
	if format := os.Getenv("GEOIP_TEST_LOOKUP_FORMAT"); format != "" {
		getInputForLookup(format, "", os.Getenv("GEOIP_TEST_LOOKUP_URI"), "")
		t.Fatal("lookup accepted an empty URI")
	}

	for _, format := range []string{
		maxmind.TypeGeoLite2CountryMMDBIn,
		maxmind.TypeDBIPCountryMMDBIn,
		maxmind.TypeIPInfoCountryMMDBIn,
		v2ray.TypeGeoIPDatIn,
	} {
		for _, uri := range []string{"", " \t"} {
			t.Run(format+"/"+uri, func(t *testing.T) {
				cmd := exec.Command(os.Args[0], "-test.run=^TestLookupRejectsEmptyURI$")
				cmd.Env = append(os.Environ(), "GEOIP_TEST_LOOKUP_FORMAT="+format, "GEOIP_TEST_LOOKUP_URI="+uri)
				output, err := cmd.CombinedOutput()
				if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
					t.Fatalf("expected exit code 1, got %v: %s", err, output)
				}
				want := fmt.Sprintf("format %s requires a non-empty \"uri\" flag", strings.ToLower(format))
				if !strings.Contains(string(output), want) {
					t.Fatalf("output = %q, want %q", output, want)
				}
			})
		}
	}
}
