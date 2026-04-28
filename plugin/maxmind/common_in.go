package maxmind

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

var (
	defaultGeoLite2CountryMMDBFile = filepath.Join("./", "geolite2", "GeoLite2-Country.mmdb")
	defaultDBIPCountryMMDBFile     = filepath.Join("./", "db-ip", "dbip-country-lite.mmdb")
	defaultIPInfoCountryMMDBFile   = filepath.Join("./", "ipinfo", "country.mmdb")
)

func NewGeoLite2CountryMMDBIn(iType string, iDesc string, action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	g := &geoLite2CountryMMDBIn{
		Type:        iType,
		Action:      action,
		Description: iDesc,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(g)
		}
	}

	return g
}

func WithGeoLite2CountryMMDBInURI(iType, uri string) lib.InputOption {
	return func(s lib.InputConverter) {
		uri = strings.TrimSpace(uri)
		if uri == "" {
			switch iType {
			case TypeGeoLite2CountryMMDBIn:
				uri = defaultGeoLite2CountryMMDBFile
			case TypeDBIPCountryMMDBIn:
				uri = defaultDBIPCountryMMDBFile
			case TypeIPInfoCountryMMDBIn:
				uri = defaultIPInfoCountryMMDBFile
			}
		}

		s.(*geoLite2CountryMMDBIn).URI = uri
	}
}

func WithGeoLite2CountryMMDBInWantedList(lists []string) lib.InputOption {
	return func(s lib.InputConverter) {
		wantList := make(map[string]bool)
		for _, want := range lists {
			if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
				wantList[want] = true
			}
		}

		s.(*geoLite2CountryMMDBIn).Want = wantList
	}
}

func WithGeoLite2CountryMMDBInOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(s lib.InputConverter) {
		s.(*geoLite2CountryMMDBIn).OnlyIPType = onlyIPType
	}
}

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

	return NewGeoLite2CountryMMDBIn(
		iType,
		iDesc,
		action,
		WithGeoLite2CountryMMDBInURI(iType, tmp.URI),
		WithGeoLite2CountryMMDBInWantedList(tmp.Want),
		WithGeoLite2CountryMMDBInOnlyIPType(tmp.OnlyIPType),
	), nil
}
