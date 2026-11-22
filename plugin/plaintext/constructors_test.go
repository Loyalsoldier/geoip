package plaintext

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestInputConstructors(t *testing.T) {
	opts := []lib.InputOption{WithNameAndURI("cn", "a.txt"), WithJSONPath([]string{"a"})}
	data := []byte(`{"name":"cn","uri":"a.txt","jsonPath":["a"]}`)
	tests := []struct {
		name      string
		typ       string
		desc      string
		newFn     func(lib.Action, ...lib.InputOption) (lib.InputConverter, error)
		fromBytes func(lib.Action, []byte) (lib.InputConverter, error)
	}{
		{"text", TypeTextIn, DescTextIn, NewTextIn, NewTextInFromBytes},
		{"json", TypeJSONIn, DescJSONIn, NewJSONIn, NewJSONInFromBytes},
		{"clash classical", TypeClashRuleSetClassicalIn, DescClashRuleSetClassicalIn, NewClashRuleSetClassicalIn, NewClashRuleSetClassicalInFromBytes},
		{"clash ipcidr", TypeClashRuleSetIPCIDRIn, DescClashRuleSetIPCIDRIn, NewClashRuleSetIPCIDRIn, NewClashRuleSetIPCIDRInFromBytes},
		{"surge", TypeSurgeRuleSetIn, DescSurgeRuleSetIn, NewSurgeRuleSetIn, NewSurgeRuleSetInFromBytes},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, fn := range []func() (lib.InputConverter, error){
				func() (lib.InputConverter, error) { return tt.newFn(lib.ActionAdd, opts...) },
				func() (lib.InputConverter, error) { return tt.fromBytes(lib.ActionAdd, data) },
			} {
				got, err := fn()
				if err != nil {
					t.Fatal(err)
				}
				if got.GetType() != tt.typ || got.GetDescription() != tt.desc || got.GetAction() != lib.ActionAdd {
					t.Fatalf("unexpected converter: %+v", got)
				}
			}
		})
	}
}

func TestOutputConstructors(t *testing.T) {
	tests := []struct {
		name      string
		typ       string
		desc      string
		newFn     func(lib.Action, ...lib.OutputOption) (lib.OutputConverter, error)
		fromBytes func(lib.Action, []byte) (lib.OutputConverter, error)
	}{
		{"text", TypeTextOut, DescTextOut, NewTextOut, NewTextOutFromBytes},
		{"clash classical", TypeClashRuleSetClassicalOut, DescClashRuleSetClassicalOut, NewClashRuleSetClassicalOut, NewClashRuleSetClassicalOutFromBytes},
		{"clash ipcidr", TypeClashRuleSetIPCIDROut, DescClashRuleSetIPCIDROut, NewClashRuleSetIPCIDROut, NewClashRuleSetIPCIDROutFromBytes},
		{"surge", TypeSurgeRuleSetOut, DescSurgeRuleSetOut, NewSurgeRuleSetOut, NewSurgeRuleSetOutFromBytes},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, fn := range []func() (lib.OutputConverter, error){
				func() (lib.OutputConverter, error) { return tt.newFn(lib.ActionOutput, WithOutputExtension(".list")) },
				func() (lib.OutputConverter, error) {
					return tt.fromBytes(lib.ActionOutput, []byte(`{"outputExtension":".list"}`))
				},
			} {
				got, err := fn()
				if err != nil {
					t.Fatal(err)
				}
				if got.GetType() != tt.typ || got.GetDescription() != tt.desc || got.GetAction() != lib.ActionOutput {
					t.Fatalf("unexpected converter: %+v", got)
				}
				o := got.(*text_out)
				if dir := o.OutputDir; dir != getDefaultOutputDir(tt.typ) {
					t.Fatalf("OutputDir = %q, want %q", dir, getDefaultOutputDir(tt.typ))
				}
				if o.OutputExt != ".list" {
					t.Fatalf("OutputExt = %q, want .list", o.OutputExt)
				}
			}
		})
	}
}

func TestRegisteredConfigCreators(t *testing.T) {
	tests := []struct{ in, out, source, want string }{
		{"text", "text", "192.0.2.1\n", "192.0.2.1/32\n"},
		{"json", "text", `{"cidrs":["192.0.2.1"]}`, "192.0.2.1/32\n"},
		{"clashRuleSetClassical", "clashRuleSetClassical", "payload:\n  - IP-CIDR,192.0.2.1/32,no-resolve\n", "payload:\n  - IP-CIDR,192.0.2.1/32\n"},
		{"clashRuleSet", "clashRuleSet", "payload:\n  - 192.0.2.1/32\n", "payload:\n  - '192.0.2.1/32'\n"},
		{"surgeRuleSet", "surgeRuleSet", "IP-CIDR,192.0.2.1/32,no-resolve\n", "IP-CIDR,192.0.2.1/32\n"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Chdir(t.TempDir())
			writeTestFile(t, ".", "input", tt.source)
			instance, err := lib.NewInstance()
			if err != nil {
				t.Fatal(err)
			}
			config := fmt.Sprintf(`{
				"input":[{"type":%q,"action":"add","args":{"name":"sample","uri":"input","jsonPath":["cidrs"]}}],
				"output":[{"type":%q,"args":{"outputDir":"out"}}]
			}`, tt.in, tt.out)
			if err := instance.InitConfigFromBytes([]byte(config)); err != nil {
				t.Fatal(err)
			}
			if err := instance.Run(); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join("out", "sample.txt"))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Fatalf("output = %q, want %q", got, tt.want)
			}
		})
	}
}
