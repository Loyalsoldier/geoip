package maxmind

import (
	"encoding/json"

	"github.com/Loyalsoldier/geoip/lib"
)

/*
The types in this file extend the type `typeMaxmindMMDBOut`,
which make it possible to support more formats for the project.
*/

const (
	TypeIPInfoCountryMMDBOut = "ipinfoCountryMMDB"
	DescIPInfoCountryMMDBOut = "Convert data to IPInfo country mmdb database format"
)

func init() {
	lib.RegisterOutputConfigCreator(TypeIPInfoCountryMMDBOut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return NewIPInfoCountryMMDBOutFromBytes(action, data)
	})
	lib.RegisterOutputConverter(TypeIPInfoCountryMMDBOut, &country_mmdb_out{
		Description: DescIPInfoCountryMMDBOut,
	})
}

func NewIPInfoCountryMMDBOut(action lib.Action, opts ...lib.OutputOption) lib.OutputConverter {
	return newCountryMMDBOut(TypeIPInfoCountryMMDBOut, DescIPInfoCountryMMDBOut, action, opts...)
}

func NewIPInfoCountryMMDBOutFromBytes(action lib.Action, data []byte) (lib.OutputConverter, error) {
	return newCountryMMDBOutFromBytes(TypeIPInfoCountryMMDBOut, DescIPInfoCountryMMDBOut, action, data)
}
