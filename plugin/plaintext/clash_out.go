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
	TypeClashRuleSetClassicalOut = "clashRuleSetClassical"
	DescClashRuleSetClassicalOut = "Convert data to classical type of Clash RuleSet"

	TypeClashRuleSetIPCIDROut = "clashRuleSet"
	DescClashRuleSetIPCIDROut = "Convert data to ipcidr type of Clash RuleSet"
)

func init() {
	lib.RegisterOutputConfigCreator(TypeClashRuleSetClassicalOut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return NewClashRuleSetClassicalOutFromBytes(action, data)
	})
	lib.RegisterOutputConverter(TypeClashRuleSetClassicalOut, &text_out{
		Description: DescClashRuleSetClassicalOut,
	})

	lib.RegisterOutputConfigCreator(TypeClashRuleSetIPCIDROut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return NewClashRuleSetIPCIDROutFromBytes(action, data)
	})
	lib.RegisterOutputConverter(TypeClashRuleSetIPCIDROut, &text_out{
		Description: DescClashRuleSetIPCIDROut,
	})
}

func NewClashRuleSetClassicalOut(action lib.Action, opts ...lib.OutputOption) lib.OutputConverter {
	return newTextOut(TypeClashRuleSetClassicalOut, DescClashRuleSetClassicalOut, action, opts...)
}

func NewClashRuleSetClassicalOutFromBytes(action lib.Action, data []byte) (lib.OutputConverter, error) {
	return newTextOutFromBytes(TypeClashRuleSetClassicalOut, DescClashRuleSetClassicalOut, action, data)
}

func NewClashRuleSetIPCIDROut(action lib.Action, opts ...lib.OutputOption) lib.OutputConverter {
	return newTextOut(TypeClashRuleSetIPCIDROut, DescClashRuleSetIPCIDROut, action, opts...)
}

func NewClashRuleSetIPCIDROutFromBytes(action lib.Action, data []byte) (lib.OutputConverter, error) {
	return newTextOutFromBytes(TypeClashRuleSetIPCIDROut, DescClashRuleSetIPCIDROut, action, data)
}
