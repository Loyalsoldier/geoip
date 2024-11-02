package plaintext

import (
	"encoding/json"

	"github.com/Loyalsoldier/geoip/lib"
)

/*
The types in this file extend the type `typeTextOut`,
which make it possible to support more formats for the project.
*/

const (
	TypeClashRuleSetClassicalOut = "clashRuleSetClassical"
	DescClashClassicalOut        = "Convert data to classical type of Clash RuleSet"

	TypeClashRuleSetIPCIDROut = "clashRuleSet"
	DescClashRuleSetOut       = "Convert data to ipcidr type of Clash RuleSet"
)

func init() {
	lib.RegisterOutputConfigCreator(TypeClashRuleSetClassicalOut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return newTextOut(TypeClashRuleSetClassicalOut, DescClashClassicalOut, action, data)
	})
	lib.RegisterOutputConverter(TypeClashRuleSetClassicalOut, &TextOut{
		Description: DescClashClassicalOut,
	})

	lib.RegisterOutputConfigCreator(TypeClashRuleSetIPCIDROut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return newTextOut(TypeClashRuleSetIPCIDROut, DescClashRuleSetOut, action, data)
	})
	lib.RegisterOutputConverter(TypeClashRuleSetIPCIDROut, &TextOut{
		Description: DescClashRuleSetOut,
	})
}
