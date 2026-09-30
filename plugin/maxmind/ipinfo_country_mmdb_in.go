package maxmind

import (
	"encoding/json"

	"github.com/Loyalsoldier/geoip/lib"
)

/*
The types in this file extend the type `typeMaxmindMMDBIn`,
which make it possible to support more formats for the project.
*/

const (
	TypeIPInfoCountryMMDBIn = "ipinfoCountryMMDB"
	DescIPInfoCountryMMDBIn = "Convert IPInfo country mmdb database to other formats"
)

func init() {
	lib.RegisterInputConfigCreator(TypeIPInfoCountryMMDBIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return NewIPInfoCountryMMDBInFromBytes(action, data)
	})
	lib.RegisterInputConverter(TypeIPInfoCountryMMDBIn, &GeoLite2CountryMMDBIn{
		Description: DescIPInfoCountryMMDBIn,
	})
}

func NewIPInfoCountryMMDBIn(action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	return newGeoLite2CountryMMDBIn(TypeIPInfoCountryMMDBIn, DescIPInfoCountryMMDBIn, action, opts...)
}

func NewIPInfoCountryMMDBInFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
	return newGeoLite2CountryMMDBInFromBytes(NewIPInfoCountryMMDBIn, action, data)
}
