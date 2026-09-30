package maxmind

import (
	"encoding/json"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	TypeDBIPCountryMMDBOut = "dbipCountryMMDB"
	DescDBIPCountryMMDBOut = "Convert data to DB-IP country mmdb database format"
)

func init() {
	lib.RegisterOutputConfigCreator(TypeDBIPCountryMMDBOut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return NewDBIPCountryMMDBOutFromBytes(action, data)
	})
	lib.RegisterOutputConverter(TypeDBIPCountryMMDBOut, &geoLite2CountryMMDBOut{
		Description: DescDBIPCountryMMDBOut,
	})
}

func NewDBIPCountryMMDBOut(action lib.Action, opts ...lib.OutputOption) lib.OutputConverter {
	return newCountryMMDBOut(TypeDBIPCountryMMDBOut, DescDBIPCountryMMDBOut, action, opts...)
}

func NewDBIPCountryMMDBOutFromBytes(action lib.Action, data []byte) (lib.OutputConverter, error) {
	return newCountryMMDBOutFromBytes(action, data, NewDBIPCountryMMDBOut)
}
