package maxmind

import (
	"encoding/json"
	"log"
	"path/filepath"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

func WithMMDBInURI(uri string) lib.InputOption {
	return func(c lib.InputConverter) {
		uri = strings.TrimSpace(uri)
		if uri == "" {
			log.Fatalf("❌ [type %s | action %s] missing uri", c.GetType(), c.GetAction())
		}
		c.(*GeoLite2CountryMMDBIn).URI = uri
	}
}

func WithMMDBInWantedList(lists []string) lib.InputOption {
	return func(c lib.InputConverter) {
		wantList := make(map[string]bool)
		for _, want := range lists {
			if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
				wantList[want] = true
			}
		}
		c.(*GeoLite2CountryMMDBIn).Want = wantList
	}
}

func WithMMDBInOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(c lib.InputConverter) {
		c.(*GeoLite2CountryMMDBIn).OnlyIPType = onlyIPType
	}
}

var (
	defaultGeoLite2CountryMMDBFile = filepath.Join("./", "geolite2", "GeoLite2-Country.mmdb")
	defaultDBIPCountryMMDBFile     = filepath.Join("./", "db-ip", "dbip-country-lite.mmdb")
	defaultIPInfoCountryMMDBFile   = filepath.Join("./", "ipinfo", "country.mmdb")
)

func NewGeoLite2CountryMMDBInFromBytes(iType string, iDesc string, action lib.Action, data []byte) (lib.InputConverter, error) {
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
		iType,
		iDesc,
		action,
		WithMMDBInURI(tmp.URI),
		WithMMDBInWantedList(tmp.Want),
		WithMMDBInOnlyIPType(tmp.OnlyIPType),
	), nil
}
