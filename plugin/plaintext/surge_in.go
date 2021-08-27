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
	typeSurgeRuleSetIn = "surgeRuleSet"
	descSurgeRuleSetIn = "Convert Surge RuleSet to other formats (just processing IP & CIDR lines)"
)

func init() {
	lib.RegisterInputConfigCreator(typeSurgeRuleSetIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newTextIn(typeSurgeRuleSetIn, action, data)
	})
	lib.RegisterInputConverter(typeSurgeRuleSetIn, &textIn{
		Description: descSurgeRuleSetIn,
	})
}
