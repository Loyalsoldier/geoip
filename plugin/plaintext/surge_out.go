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
	typeSurgeRuleSetOut = "surgeRuleSet"
	descSurgeRuleSetOut = "Convert data to Surge RuleSet"
)

func init() {
	lib.RegisterOutputConfigCreator(typeSurgeRuleSetOut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return newTextOut(typeSurgeRuleSetOut, action, data)
	})
	lib.RegisterOutputConverter(typeSurgeRuleSetOut, &textOut{
		Description: descSurgeRuleSetOut,
	})
}
