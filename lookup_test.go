package main

import (
	"testing"

	"github.com/Loyalsoldier/geoip/plugin/mihomo"
	"github.com/Loyalsoldier/geoip/plugin/plaintext"
	"github.com/Loyalsoldier/geoip/plugin/singbox"
)

func TestGetInputForLookupWithDirectory(t *testing.T) {
	formats := []string{
		mihomo.TypeMRSIn,
		singbox.TypeSRSIn,
		plaintext.TypeTextIn,
		plaintext.TypeClashRuleSetIPCIDRIn,
		plaintext.TypeClashRuleSetClassicalIn,
		plaintext.TypeSurgeRuleSetIn,
	}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			input := getInputForLookup(format, "", t.TempDir())
			if got := input.GetType(); got != format {
				t.Fatalf("GetType() = %q, want %q", got, format)
			}
		})
	}
}
