package plaintext

import (
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestInputConstructors(t *testing.T) {
	opts := []lib.InputOption{WithNameAndURI("cn", "a.txt"), WithJSONPath([]string{"a"})}
	data := []byte(`{"name":"cn","uri":"a.txt","jsonPath":["a"]}`)
	tests := []struct {
		name      string
		typ, desc string
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
		typ, desc string
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
				func() (lib.OutputConverter, error) { return tt.newFn(lib.ActionOutput, WithOutputDir("")) },
				func() (lib.OutputConverter, error) { return tt.fromBytes(lib.ActionOutput, nil) },
			} {
				got, err := fn()
				if err != nil {
					t.Fatal(err)
				}
				if got.GetType() != tt.typ || got.GetDescription() != tt.desc || got.GetAction() != lib.ActionOutput {
					t.Fatalf("unexpected converter: %+v", got)
				}
				if dir := got.(*text_out).OutputDir; dir != getDefaultOutputDir(tt.typ) {
					t.Fatalf("OutputDir = %q, want %q", dir, getDefaultOutputDir(tt.typ))
				}
			}
		})
	}
}

// TestRegisteredConfigCreators exercises the config creators registered in init().
func TestRegisteredConfigCreators(t *testing.T) {
	types := []struct{ in, out string }{
		{TypeTextIn, TypeTextOut},
		{TypeJSONIn, TypeTextOut},
		{TypeClashRuleSetClassicalIn, TypeClashRuleSetClassicalOut},
		{TypeClashRuleSetIPCIDRIn, TypeClashRuleSetIPCIDROut},
		{TypeSurgeRuleSetIn, TypeSurgeRuleSetOut},
	}
	for _, tt := range types {
		t.Run(tt.in, func(t *testing.T) {
			instance, err := lib.NewInstance()
			if err != nil {
				t.Fatal(err)
			}
			config := `{"input":[{"type":"` + tt.in + `","action":"add","args":{"name":"cn","uri":"a","jsonPath":["a"]}}],` +
				`"output":[{"type":"` + tt.out + `","action":"output"}]}`
			if err := instance.InitConfigFromBytes([]byte(config)); err != nil {
				t.Fatal(err)
			}
		})
	}
}
