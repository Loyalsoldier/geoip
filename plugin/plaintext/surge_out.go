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
		return NewSurgeRuleSetOutFromBytes(action, data)
	})
	lib.RegisterOutputConverter(TypeSurgeRuleSetOut, &text_out{
		Description: DescSurgeRuleSetOut,
	})
}

func NewSurgeRuleSetOut(action lib.Action, opts ...lib.OutputOption) lib.OutputConverter {
	return newTextOut(TypeSurgeRuleSetOut, DescSurgeRuleSetOut, action, opts...)
}

func NewSurgeRuleSetOutFromBytes(action lib.Action, data []byte) (lib.OutputConverter, error) {
	return newTextOutFromBytes(TypeSurgeRuleSetOut, DescSurgeRuleSetOut, action, data)
}
