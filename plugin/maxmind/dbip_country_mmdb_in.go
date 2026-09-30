package maxmind

import (
	"encoding/json"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	TypeDBIPCountryMMDBIn = "dbipCountryMMDB"
	DescDBIPCountryMMDBIn = "Convert DB-IP country mmdb database to other formats"
)

func init() {
	lib.RegisterInputConfigCreator(TypeDBIPCountryMMDBIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return NewDBIPCountryMMDBInFromBytes(action, data)
	})
	lib.RegisterInputConverter(TypeDBIPCountryMMDBIn, &geoLite2CountryMMDBIn{
		Description: DescDBIPCountryMMDBIn,
	})
}

func NewDBIPCountryMMDBIn(action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	return newGeoLite2CountryMMDBIn(TypeDBIPCountryMMDBIn, DescDBIPCountryMMDBIn, action, opts...)
}

func NewDBIPCountryMMDBInFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
	return newCountryMMDBInFromBytes(action, data, NewDBIPCountryMMDBIn)
}
