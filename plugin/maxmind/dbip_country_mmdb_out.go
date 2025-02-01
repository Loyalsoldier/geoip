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
	TypeDBIPCountryMMDBOut = "dbipCountryMMDB"
	DescDBIPCountryMMDBOut = "Convert data to DB-IP country mmdb database format"
)

func init() {
	lib.RegisterOutputConfigCreator(TypeDBIPCountryMMDBOut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return newGeoLite2CountryMMDBOut(TypeDBIPCountryMMDBOut, DescDBIPCountryMMDBOut, action, data)
	})
	lib.RegisterOutputConverter(TypeDBIPCountryMMDBOut, &GeoLite2CountryMMDBOut{
		Description: DescDBIPCountryMMDBOut,
	})
}
