package plaintext

import (
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestOutputDirOptionReuse(t *testing.T) {
	formats := []struct {
		newConverter func(lib.Action, ...lib.OutputOption) lib.OutputConverter
		defaultDir   string
	}{
		{NewTextOut, defaultOutputDirForTextOut},
		{NewClashRuleSetIPCIDROut, defaultOutputDirForClashRuleSetIPCIDROut},
		{NewClashRuleSetClassicalOut, defaultOutputDirForClashRuleSetClassicalOut},
		{NewSurgeRuleSetOut, defaultOutputDirForSurgeRuleSetOut},
	}
	for _, dir := range []string{"", " \t ", " custom-output "} {
		opt := WithOutputDir(dir)
		for _, format := range formats {
			converter := format.newConverter(lib.ActionOutput, opt).(*text_out)
			want := strings.TrimSpace(dir)
			if want == "" {
				want = format.defaultDir
			}
			if converter.OutputDir != want {
				t.Errorf("%s with directory %q: got %q, want %q", converter.Type, dir, converter.OutputDir, want)
			}
		}
	}
}
