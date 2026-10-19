package plaintext

import (
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestOutputDirOptionReuse(t *testing.T) {
	for _, dir := range []string{"", " \t", " custom-output "} {
		t.Run(dir, func(t *testing.T) {
			opt := WithOutputDir(dir)
			for _, tc := range []struct {
				name string
				new  func(lib.Action, ...lib.OutputOption) (lib.OutputConverter, error)
				want string
			}{
				{TypeTextOut, NewTextOut, defaultOutputDirForTextOut},
				{TypeClashRuleSetClassicalOut, NewClashRuleSetClassicalOut, defaultOutputDirForClashRuleSetClassicalOut},
				{TypeClashRuleSetIPCIDROut, NewClashRuleSetIPCIDROut, defaultOutputDirForClashRuleSetIPCIDROut},
				{TypeSurgeRuleSetOut, NewSurgeRuleSetOut, defaultOutputDirForSurgeRuleSetOut},
			} {
				t.Run(tc.name, func(t *testing.T) {
					out, err := tc.new(lib.ActionOutput, opt)
					if err != nil {
						t.Fatal(err)
					}
					want := tc.want
					if strings.TrimSpace(dir) != "" {
						want = strings.TrimSpace(dir)
					}
					if got := out.(*text_out).OutputDir; got != want {
						t.Fatalf("outputDir = %q, want %q", got, want)
					}
				})
			}
		})
	}
}
