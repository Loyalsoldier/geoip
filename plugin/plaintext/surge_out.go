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
	TypeSurgeRuleSetOut = "surgeRuleSet"
	DescSurgeRuleSetOut = "Convert data to Surge RuleSet"
)

func init() {
	lib.RegisterOutputConfigCreator(TypeSurgeRuleSetOut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return newTextOut(TypeSurgeRuleSetOut, DescSurgeRuleSetOut, action, data)
	})
	lib.RegisterOutputConverter(TypeSurgeRuleSetOut, &TextOut{
		Description: DescSurgeRuleSetOut,
	})
}
