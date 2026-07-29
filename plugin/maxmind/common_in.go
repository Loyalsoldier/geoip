package maxmind

import (
	"encoding/json"
	"log"
	"path/filepath"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

var (
	defaultGeoLite2CountryMMDBFile = filepath.Join("./", "geolite2", "GeoLite2-Country.mmdb")
	defaultDBIPCountryMMDBFile     = filepath.Join("./", "db-ip", "dbip-country-lite.mmdb")
	defaultIPInfoCountryMMDBFile   = filepath.Join("./", "ipinfo", "country.mmdb")
)

func defaultMMDBFileFor(iType string) string {
	switch iType {
	case TypeGeoLite2CountryMMDBIn:
		return defaultGeoLite2CountryMMDBFile
	case TypeDBIPCountryMMDBIn:
		return defaultDBIPCountryMMDBFile
	case TypeIPInfoCountryMMDBIn:
		return defaultIPInfoCountryMMDBFile
	default:
		return ""
	}
}

func NewGeoLite2CountryMMDBIn(iType string, iDesc string, action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	g := &geolite2_country_mmdb_in{
		Type:        strings.TrimSpace(iType),
		Action:      action,
		Description: iDesc,
	}
	g.URI = defaultMMDBFileFor(g.Type)

	for _, opt := range opts {
		if opt != nil {
			opt(g)
		}
	}

	if g.Type == "" {
		log.Fatalf("❌ [action %s] missing type", g.Action)
	}

	if g.URI == "" {
		log.Fatalf("❌ [type %s | action %s] missing uri", g.Type, g.Action)
	}

	return g
}

func WithMMDBInURI(uri string) lib.InputOption {
	return func(g lib.InputConverter) {
		if uri = strings.TrimSpace(uri); uri == "" {
			return
		}

		g.(*geolite2_country_mmdb_in).URI = uri
	}
}

func WithMMDBInWantedList(lists []string) lib.InputOption {
	return func(g lib.InputConverter) {
		wantList := make(map[string]bool)
		for _, want := range lists {
			if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
				wantList[want] = true
			}
		}

		g.(*geolite2_country_mmdb_in).Want = wantList
	}
}

func WithMMDBInOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(g lib.InputConverter) {
		g.(*geolite2_country_mmdb_in).OnlyIPType = onlyIPType
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
		WithMMDBInURI(tmp.URI),
		WithMMDBInWantedList(tmp.Want),
		WithMMDBInOnlyIPType(tmp.OnlyIPType),
	), nil
}
