package maxmind

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

var (
	defaultGeoLite2CountryMMDBFile = filepath.Join("./", "geolite2", "GeoLite2-Country.mmdb")
	defaultDBIPCountryMMDBFile     = filepath.Join("./", "db-ip", "dbip-country-lite.mmdb")
	defaultIPInfoCountryMMDBFile   = filepath.Join("./", "ipinfo", "country.mmdb")
)

func defaultMMDBFileForType(iType string) string {
	switch iType {
	case TypeGeoLite2CountryMMDBIn:
		return defaultGeoLite2CountryMMDBFile
	case TypeDBIPCountryMMDBIn:
		return defaultDBIPCountryMMDBFile
	case TypeIPInfoCountryMMDBIn:
		return defaultIPInfoCountryMMDBFile
	}
	return ""
}

func newGeoLite2CountryMMDBIn(iType string, iDesc string, action lib.Action, opts ...lib.InputOption) lib.InputConverter {
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

	if g.URI == "" {
		g.URI = defaultMMDBFileForType(g.Type)
	}

	return g
}

// NewGeoLite2CountryMMDBIn creates an input converter of type maxmindMMDB
func NewGeoLite2CountryMMDBIn(action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	return newGeoLite2CountryMMDBIn(TypeGeoLite2CountryMMDBIn, DescGeoLite2CountryMMDBIn, action, opts...)
}

// NewDBIPCountryMMDBIn creates an input converter of type dbipCountryMMDB
func NewDBIPCountryMMDBIn(action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	return newGeoLite2CountryMMDBIn(TypeDBIPCountryMMDBIn, DescDBIPCountryMMDBIn, action, opts...)
}

// NewIPInfoCountryMMDBIn creates an input converter of type ipinfoCountryMMDB
func NewIPInfoCountryMMDBIn(action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	return newGeoLite2CountryMMDBIn(TypeIPInfoCountryMMDBIn, DescIPInfoCountryMMDBIn, action, opts...)
}

func WithMMDBInURI(uri string) lib.InputOption {
	return func(g lib.InputConverter) {
		g.(*geoLite2CountryMMDBIn).URI = strings.TrimSpace(uri)
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

		g.(*geoLite2CountryMMDBIn).Want = wantList
	}
}

func WithMMDBInOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(g lib.InputConverter) {
		g.(*geoLite2CountryMMDBIn).OnlyIPType = onlyIPType
	}
}

func newGeoLite2CountryMMDBInFromBytes(iType string, iDesc string, action lib.Action, data []byte) (lib.InputConverter, error) {
	var tmp struct {
		URI        string     `json:"uri"`
		Want       []string   `json:"wantedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, fmt.Errorf("❌ [type %s | action %s] %w", iType, action, err)
		}
	}

	return newGeoLite2CountryMMDBIn(
		iType,
		iDesc,
		action,
		WithMMDBInURI(tmp.URI),
		WithMMDBInWantedList(tmp.Want),
		WithMMDBInOnlyIPType(tmp.OnlyIPType),
	), nil
}
