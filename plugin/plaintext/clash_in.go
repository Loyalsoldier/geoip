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
	DescClashClassicalIn        = "Convert classical type of Clash RuleSet to other formats (just processing IP & CIDR lines)"

	TypeClashRuleSetIPCIDRIn = "clashRuleSet"
	DescClashRuleSetIn       = "Convert ipcidr type of Clash RuleSet to other formats"
)

func init() {
	lib.RegisterInputConfigCreator(TypeClashRuleSetClassicalIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newTextIn(TypeClashRuleSetClassicalIn, DescClashClassicalIn, action, data)
	})
	lib.RegisterInputConverter(TypeClashRuleSetClassicalIn, &TextIn{
		Description: DescClashClassicalIn,
	})

	lib.RegisterInputConfigCreator(TypeClashRuleSetIPCIDRIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newTextIn(TypeClashRuleSetIPCIDRIn, DescClashRuleSetIn, action, data)
	})
	lib.RegisterInputConverter(TypeClashRuleSetIPCIDRIn, &TextIn{
		Description: DescClashRuleSetIn,
	})
}
