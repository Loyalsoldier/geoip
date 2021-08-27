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
	typeClashRuleSetClassicalOut = "clashRuleSetClassical"
	descClashClassicalOut        = "Convert data to classical type of Clash RuleSet"

	typeClashRuleSetIPCIDROut = "clashRuleSet"
	descClashRuleSetOut       = "Convert data to ipcidr type of Clash RuleSet"
)

func init() {
	lib.RegisterOutputConfigCreator(typeClashRuleSetClassicalOut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return newTextOut(typeClashRuleSetClassicalOut, action, data)
	})
	lib.RegisterOutputConverter(typeClashRuleSetClassicalOut, &textOut{
		Description: descClashClassicalOut,
	})

	lib.RegisterOutputConfigCreator(typeClashRuleSetIPCIDROut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return newTextOut(typeClashRuleSetIPCIDROut, action, data)
	})
	lib.RegisterOutputConverter(typeClashRuleSetIPCIDROut, &textOut{
		Description: descClashRuleSetOut,
	})
}
