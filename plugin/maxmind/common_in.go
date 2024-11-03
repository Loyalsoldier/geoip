package maxmind

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

var (
	defaultGeoLite2MMDBFile      = filepath.Join("./", "geolite2", "GeoLite2-Country.mmdb")
	defaultDBIPCountryMMDBFile   = filepath.Join("./", "db-ip", "dbip-country-lite.mmdb")
	defaultIPInfoCountryMMDBFile = filepath.Join("./", "ipinfo", "country.mmdb")
)

func newMMDBIn(iType string, iDesc string, action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
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
		case TypeMaxmindMMDBIn:
			tmp.URI = defaultGeoLite2MMDBFile

		case TypeDBIPCountryMMDBIn:
			tmp.URI = defaultDBIPCountryMMDBFile

		case TypeIPInfoCountryMMDBIn:
			tmp.URI = defaultIPInfoCountryMMDBFile
		}
	}

	// Filter want list
	wantList := make(map[string]bool)
	for _, want := range tmp.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList[want] = true
		}
	}

	return &MMDBIn{
		Type:        iType,
		Action:      action,
		Description: iDesc,
		URI:         tmp.URI,
		Want:        wantList,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}
