// Download the IPInfo Lite country database from here for free: https://ipinfo.io/lite
// The database is in MaxMind DB (MMDB) format.
// Check the database schema here: https://ipinfo.io/lite
// Or use `mmdbinspect` CLI to inspect the database: https://github.com/maxmind/mmdbinspect
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
	lib.RegisterInputConverter(TypeIPInfoCountryMMDBIn, &geolite2_country_mmdb_in{
		Description: DescIPInfoCountryMMDBIn,
	})
}

func NewIPInfoCountryMMDBIn(action lib.Action, opts ...lib.InputOption) (lib.InputConverter, error) {
	return newGeoLite2CountryMMDBIn(TypeIPInfoCountryMMDBIn, DescIPInfoCountryMMDBIn, action, opts...)
}

func NewIPInfoCountryMMDBInFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
	return newGeoLite2CountryMMDBInFromBytes(TypeIPInfoCountryMMDBIn, DescIPInfoCountryMMDBIn, action, data)
}
