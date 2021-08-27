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
	typeClashRuleSetClassicalIn = "clashRuleSetClassical"
	descClashClassicalIn        = "Convert classical type of Clash RuleSet to other formats (just processing IP & CIDR lines)"

	typeClashRuleSetIPCIDRIn = "clashRuleSet"
	descClashRuleSetIn       = "Convert ipcidr type of Clash RuleSet to other formats"
)

func init() {
	lib.RegisterInputConfigCreator(typeClashRuleSetClassicalIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newTextIn(typeClashRuleSetClassicalIn, action, data)
	})
	lib.RegisterInputConverter(typeClashRuleSetClassicalIn, &textIn{
		Description: descClashClassicalIn,
	})

	lib.RegisterInputConfigCreator(typeClashRuleSetIPCIDRIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newTextIn(typeClashRuleSetIPCIDRIn, action, data)
	})
	lib.RegisterInputConverter(typeClashRuleSetIPCIDRIn, &textIn{
		Description: descClashRuleSetIn,
	})
}
