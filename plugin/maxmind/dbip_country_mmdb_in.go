// Download the DB-IP IP to Country Lite database from here for free: https://db-ip.com/db/download/ip-to-country-lite
// The database is in MaxMind DB (MMDB) format.
// Check the database schema here: https://db-ip.com/db/format/ip-to-country-lite/mmdb.html
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
	TypeDBIPCountryMMDBIn = "dbipCountryMMDB"
	DescDBIPCountryMMDBIn = "Convert DB-IP country mmdb database to other formats"
)

func init() {
	lib.RegisterInputConfigCreator(TypeDBIPCountryMMDBIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return NewDBIPCountryMMDBInFromBytes(action, data)
	})
	lib.RegisterInputConverter(TypeDBIPCountryMMDBIn, &geolite2_country_mmdb_in{
		Description: DescDBIPCountryMMDBIn,
	})
}

func NewDBIPCountryMMDBIn(action lib.Action, opts ...lib.InputOption) (lib.InputConverter, error) {
	return newGeoLite2CountryMMDBIn(TypeDBIPCountryMMDBIn, DescDBIPCountryMMDBIn, action, opts...)
}

func NewDBIPCountryMMDBInFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
	return newGeoLite2CountryMMDBInFromBytes(TypeDBIPCountryMMDBIn, DescDBIPCountryMMDBIn, action, data)
}
