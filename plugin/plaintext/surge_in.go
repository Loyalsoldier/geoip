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
	TypeSurgeRuleSetIn = "surgeRuleSet"
	DescSurgeRuleSetIn = "Convert Surge RuleSet to other formats (just processing IP & CIDR lines)"
)

func init() {
	lib.RegisterInputConfigCreator(TypeSurgeRuleSetIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newTextIn(TypeSurgeRuleSetIn, DescSurgeRuleSetIn, action, data)
	})
	lib.RegisterInputConverter(TypeSurgeRuleSetIn, &TextIn{
		Description: DescSurgeRuleSetIn,
	})
}
