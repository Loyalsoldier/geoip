package plaintext

import (
	"encoding/json"

	"github.com/Loyalsoldier/geoip/lib"
)

/*
The types in this file extend the type `typeTextIn`,
which make it possible to support more formats for the project.
*/

const (
	TypeClashRuleSetClassicalIn = "clashRuleSetClassical"
	DescClashRuleSetClassicalIn = "Convert classical type of Clash RuleSet to other formats (just processing IP & CIDR lines)"

	TypeClashRuleSetIPCIDRIn = "clashRuleSet"
	DescClashRuleSetIPCIDRIn = "Convert ipcidr type of Clash RuleSet to other formats"
)

func init() {
	lib.RegisterInputConfigCreator(TypeClashRuleSetClassicalIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newTextIn(TypeClashRuleSetClassicalIn, DescClashRuleSetClassicalIn, action, data)
	})
	lib.RegisterInputConverter(TypeClashRuleSetClassicalIn, &TextIn{
		Description: DescClashRuleSetClassicalIn,
	})

	lib.RegisterInputConfigCreator(TypeClashRuleSetIPCIDRIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newTextIn(TypeClashRuleSetIPCIDRIn, DescClashRuleSetIPCIDRIn, action, data)
	})
	lib.RegisterInputConverter(TypeClashRuleSetIPCIDRIn, &TextIn{
		Description: DescClashRuleSetIPCIDRIn,
	})
}
