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
		return NewSurgeRuleSetInFromBytes(action, data)
	})
	lib.RegisterInputConverter(TypeSurgeRuleSetIn, &text_in{
		Description: DescSurgeRuleSetIn,
	})
}

func NewSurgeRuleSetIn(action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	return newTextIn(TypeSurgeRuleSetIn, DescSurgeRuleSetIn, action, opts...)
}

func NewSurgeRuleSetInFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
	return newTextInFromBytes(TypeSurgeRuleSetIn, DescSurgeRuleSetIn, action, data)
}
