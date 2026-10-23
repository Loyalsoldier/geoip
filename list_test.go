package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/plugin/maxmind"
	"github.com/Loyalsoldier/geoip/plugin/mihomo"
	"github.com/Loyalsoldier/geoip/plugin/plaintext"
	"github.com/Loyalsoldier/geoip/plugin/singbox"
	"github.com/Loyalsoldier/geoip/plugin/special"
	"github.com/Loyalsoldier/geoip/plugin/v2ray"
)

// wantListOutput is the exact standard output of the list command: all
// registered converters sorted by name. The blank line printed between the
// two sections by the builtin println goes to standard error, not stdout.
func wantListOutput() string {
	inputs := [][2]string{
		{plaintext.TypeClashRuleSetIPCIDRIn, plaintext.DescClashRuleSetIPCIDRIn},
		{plaintext.TypeClashRuleSetClassicalIn, plaintext.DescClashRuleSetClassicalIn},
		{special.TypeCutter, special.DescCutter},
		{maxmind.TypeDBIPCountryMMDBIn, maxmind.DescDBIPCountryMMDBIn},
		{maxmind.TypeIPInfoCountryMMDBIn, maxmind.DescIPInfoCountryMMDBIn},
		{plaintext.TypeJSONIn, plaintext.DescJSONIn},
		{maxmind.TypeGeoLite2ASNCSVIn, maxmind.DescGeoLite2ASNCSVIn},
		{maxmind.TypeGeoLite2CountryCSVIn, maxmind.DescGeoLite2CountryCSVIn},
		{maxmind.TypeGeoLite2CountryMMDBIn, maxmind.DescGeoLite2CountryMMDBIn},
		{mihomo.TypeMRSIn, mihomo.DescMRSIn},
		{special.TypePrivate, special.DescPrivate},
		{singbox.TypeSRSIn, singbox.DescSRSIn},
		{special.TypeStdin, special.DescStdin},
		{plaintext.TypeSurgeRuleSetIn, plaintext.DescSurgeRuleSetIn},
		{"test", "Convert specific CIDR to other formats (for test only)"},
		{plaintext.TypeTextIn, plaintext.DescTextIn},
		{v2ray.TypeGeoIPDatIn, v2ray.DescGeoIPDatIn},
	}
	outputs := [][2]string{
		{plaintext.TypeClashRuleSetIPCIDROut, plaintext.DescClashRuleSetIPCIDROut},
		{plaintext.TypeClashRuleSetClassicalOut, plaintext.DescClashRuleSetClassicalOut},
		{maxmind.TypeDBIPCountryMMDBOut, maxmind.DescDBIPCountryMMDBOut},
		{maxmind.TypeIPInfoCountryMMDBOut, maxmind.DescIPInfoCountryMMDBOut},
		{special.TypeLookup, special.DescLookup},
		{maxmind.TypeGeoLite2CountryMMDBOut, maxmind.DescGeoLite2CountryMMDBOut},
		{mihomo.TypeMRSOut, mihomo.DescMRSOut},
		{singbox.TypeSRSOut, singbox.DescSRSOut},
		{special.TypeStdout, special.DescStdout},
		{plaintext.TypeSurgeRuleSetOut, plaintext.DescSurgeRuleSetOut},
		{plaintext.TypeTextOut, plaintext.DescTextOut},
		{v2ray.TypeGeoIPDatOut, v2ray.DescGeoIPDatOut},
	}

	var b strings.Builder
	b.WriteString("All available input formats:\n")
	for _, in := range inputs {
		fmt.Fprintf(&b, "  - %s (%s)\n", in[0], in[1])
	}
	b.WriteString("All available output formats:\n")
	for _, out := range outputs {
		fmt.Fprintf(&b, "  - %s (%s)\n", out[0], out[1])
	}

	return b.String()
}

func TestListCmd(t *testing.T) {
	want := wantListOutput()

	tests := []struct {
		name string
		args []string
	}{
		{name: "list", args: []string{"list"}},
		{name: "alias l", args: []string{"l"}},
		{name: "alias ls", args: []string{"ls"}},
		{name: "extra args are ignored", args: []string{"list", "foo"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, cobraOutput, err := executeCommand(t, tt.args...)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if cobraOutput != "" {
				t.Errorf("cobra output = %q, want empty", cobraOutput)
			}
			assertLines(t, stdout, want)
		})
	}
}
