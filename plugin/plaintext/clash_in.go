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
		return NewClashRuleSetClassicalInFromBytes(action, data)
	})
	lib.RegisterInputConverter(TypeClashRuleSetClassicalIn, &text_in{
		Description: DescClashRuleSetClassicalIn,
	})

	lib.RegisterInputConfigCreator(TypeClashRuleSetIPCIDRIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return NewClashRuleSetIPCIDRInFromBytes(action, data)
	})
	lib.RegisterInputConverter(TypeClashRuleSetIPCIDRIn, &text_in{
		Description: DescClashRuleSetIPCIDRIn,
	})
}

func NewClashRuleSetClassicalIn(action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	return mustNewTextIn(TypeClashRuleSetClassicalIn, DescClashRuleSetClassicalIn, action, opts...)
}

func NewClashRuleSetClassicalInFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
	return newTextInFromBytes(TypeClashRuleSetClassicalIn, DescClashRuleSetClassicalIn, action, data)
}

func NewClashRuleSetIPCIDRIn(action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	return mustNewTextIn(TypeClashRuleSetIPCIDRIn, DescClashRuleSetIPCIDRIn, action, opts...)
}

func NewClashRuleSetIPCIDRInFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
	return newTextInFromBytes(TypeClashRuleSetIPCIDRIn, DescClashRuleSetIPCIDRIn, action, data)
}
