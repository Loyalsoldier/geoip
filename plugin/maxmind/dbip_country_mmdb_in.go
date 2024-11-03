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
	TypeDBIPCountryMMDBIn = "dbipCountryMMDB"
	DescDBIPCountryMMDBIn = "Convert DB-IP country mmdb database to other formats"
)

func init() {
	lib.RegisterInputConfigCreator(TypeDBIPCountryMMDBIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newMMDBIn(TypeDBIPCountryMMDBIn, DescDBIPCountryMMDBIn, action, data)
	})
	lib.RegisterInputConverter(TypeDBIPCountryMMDBIn, &MMDBIn{
		Description: DescDBIPCountryMMDBIn,
	})
}
