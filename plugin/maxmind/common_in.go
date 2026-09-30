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

func newGeoLite2CountryMMDBIn(iType string, iDesc string, action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	g := &GeoLite2CountryMMDBIn{
		Type:        iType,
		Action:      action,
		Description: iDesc,
		Want:        make(map[string]bool),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(g)
		}
	}

	if g.URI == "" {
		switch iType {
		case TypeGeoLite2CountryMMDBIn:
			g.URI = defaultGeoLite2CountryMMDBFile

		case TypeDBIPCountryMMDBIn:
			g.URI = defaultDBIPCountryMMDBFile

		case TypeIPInfoCountryMMDBIn:
			g.URI = defaultIPInfoCountryMMDBFile
		}
	}

	validateInput(g, g.OnlyIPType)
	return g
}

func WithURI(uri string) lib.InputOption {
	return func(g lib.InputConverter) {
		g.(*GeoLite2CountryMMDBIn).URI = strings.TrimSpace(uri)
	}
}

func WithInputWantedList(lists []string) lib.InputOption {
	return func(g lib.InputConverter) {
		wantList := make(map[string]bool)
		for _, want := range lists {
			if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
				wantList[want] = true
			}
		}

		switch g := g.(type) {
		case *GeoLite2CountryMMDBIn:
			g.Want = wantList
		case *GeoLite2CountryCSVIn:
			g.Want = wantList
		case *GeoLite2ASNCSVIn:
			WithInputWantedListExtended(lib.WantedListExtended{TypeSlice: lists})(g)
		}
	}
}

func WithInputOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(g lib.InputConverter) {
		switch g := g.(type) {
		case *GeoLite2CountryMMDBIn:
			g.OnlyIPType = onlyIPType
		case *GeoLite2CountryCSVIn:
			g.OnlyIPType = onlyIPType
		case *GeoLite2ASNCSVIn:
			g.OnlyIPType = onlyIPType
		}
	}
}

func WithIPv4File(file string) lib.InputOption {
	return func(g lib.InputConverter) {
		switch g := g.(type) {
		case *GeoLite2CountryCSVIn:
			g.IPv4File = strings.TrimSpace(file)
		case *GeoLite2ASNCSVIn:
			g.IPv4File = strings.TrimSpace(file)
		}
	}
}

func WithIPv6File(file string) lib.InputOption {
	return func(g lib.InputConverter) {
		switch g := g.(type) {
		case *GeoLite2CountryCSVIn:
			g.IPv6File = strings.TrimSpace(file)
		case *GeoLite2ASNCSVIn:
			g.IPv6File = strings.TrimSpace(file)
		}
	}
}

func validateInput(g lib.InputConverter, onlyIPType lib.IPType) {
	if g.GetAction() != lib.ActionAdd && g.GetAction() != lib.ActionRemove {
		log.Fatalf("❌ [type %s | action %s] invalid action: must be add or remove", g.GetType(), g.GetAction())
	}
	validateOnlyIPType(g.GetType(), g.GetAction(), onlyIPType)
}

func validateOnlyIPType(iType string, action lib.Action, onlyIPType lib.IPType) {
	if onlyIPType != "" && onlyIPType != lib.IPv4 && onlyIPType != lib.IPv6 {
		log.Fatalf("❌ [type %s | action %s] invalid onlyIPType %q: must be ipv4 or ipv6", iType, action, onlyIPType)
	}
}

func newGeoLite2CountryMMDBInFromBytes(newIn func(lib.Action, ...lib.InputOption) lib.InputConverter, action lib.Action, data []byte) (lib.InputConverter, error) {
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

	return newIn(
		action,
		WithURI(tmp.URI),
		WithInputWantedList(tmp.Want),
		WithInputOnlyIPType(tmp.OnlyIPType),
	), nil
}
