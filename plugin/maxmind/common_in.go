package maxmind

import (
	"encoding/json"
	"path/filepath"

	"github.com/Loyalsoldier/geoip/lib"
)

var (
	defaultGeoLite2CountryMMDBFile = filepath.Join("./", "geolite2", "GeoLite2-Country.mmdb")
	defaultDBIPCountryMMDBFile     = filepath.Join("./", "db-ip", "dbip-country-lite.mmdb")
	defaultIPInfoCountryMMDBFile   = filepath.Join("./", "ipinfo", "country.mmdb")
)

func NewGeoLite2CountryMMDBInFromBytes(iType string, iDesc string, action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
	var tmp struct {
		URI        string     `json:"uri"`
		Want       []string   `json:"wantedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	if tmp.URI == "" {
		switch iType {
		case TypeGeoLite2CountryMMDBIn:
			tmp.URI = defaultGeoLite2CountryMMDBFile

		case TypeDBIPCountryMMDBIn:
			tmp.URI = defaultDBIPCountryMMDBFile

		case TypeIPInfoCountryMMDBIn:
			tmp.URI = defaultIPInfoCountryMMDBFile
		}
	}

	return NewGeoLite2CountryMMDBIn(
		iType, iDesc, action,
		WithURI(tmp.URI),
		WithInputWantedList(tmp.Want),
		WithInputOnlyIPType(tmp.OnlyIPType),
	), nil
}
