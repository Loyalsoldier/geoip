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
		return newGeoLite2CountryMMDBOut(TypeIPInfoCountryMMDBOut, DescIPInfoCountryMMDBOut, action, data)
	})
	lib.RegisterOutputConverter(TypeIPInfoCountryMMDBOut, &GeoLite2CountryMMDBOut{
		Description: DescIPInfoCountryMMDBOut,
	})
}
