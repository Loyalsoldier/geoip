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

func newCountryMMDBIn(iType, iDesc string, action lib.Action, opts ...lib.InputOption) lib.InputConverter {
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

	validateInputOptions(g.Type, g.Action, g.OnlyIPType)
	if g.URI == "" {
		switch g.Type {
		case TypeGeoLite2CountryMMDBIn:
			g.URI = defaultGeoLite2CountryMMDBFile
		case TypeDBIPCountryMMDBIn:
			g.URI = defaultDBIPCountryMMDBFile
		case TypeIPInfoCountryMMDBIn:
			g.URI = defaultIPInfoCountryMMDBFile
		}
	}
	g.Want = countryWantedList(g.Type, g.Action, g.wantedList)
	g.wantedList = nil
	return g
}

func newCountryMMDBInFromBytes(action lib.Action, data []byte, constructor func(lib.Action, ...lib.InputOption) lib.InputConverter) (lib.InputConverter, error) {
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

	return constructor(
		action,
		WithURI(tmp.URI),
		WithInputWantedList(tmp.Want),
		WithInputOnlyIPType(tmp.OnlyIPType),
	), nil
}

func WithURI(uri string) lib.InputOption {
	return func(g lib.InputConverter) {
		g.(*geoLite2CountryMMDBIn).URI = strings.TrimSpace(uri)
	}
}

// WithInputWantedList accepts country names, or ASN strings and category-to-ASN maps
// for ASN CSV inputs.
func WithInputWantedList(lists any) lib.InputOption {
	return func(g lib.InputConverter) {
		switch g := g.(type) {
		case *geoLite2CountryMMDBIn:
			g.wantedList = lists
		case *geoLite2CountryCSVIn:
			g.wantedList = lists
		case *geoLite2ASNCSVIn:
			g.wantedList = lists
		}
	}
}

func WithInputOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(g lib.InputConverter) {
		switch g := g.(type) {
		case *geoLite2CountryMMDBIn:
			g.OnlyIPType = onlyIPType
		case *geoLite2CountryCSVIn:
			g.OnlyIPType = onlyIPType
		case *geoLite2ASNCSVIn:
			g.OnlyIPType = onlyIPType
		}
	}
}

func WithIPv4File(file string) lib.InputOption {
	return func(g lib.InputConverter) {
		switch g := g.(type) {
		case *geoLite2CountryCSVIn:
			g.IPv4File = strings.TrimSpace(file)
		case *geoLite2ASNCSVIn:
			g.IPv4File = strings.TrimSpace(file)
		}
	}
}

func WithIPv6File(file string) lib.InputOption {
	return func(g lib.InputConverter) {
		switch g := g.(type) {
		case *geoLite2CountryCSVIn:
			g.IPv6File = strings.TrimSpace(file)
		case *geoLite2ASNCSVIn:
			g.IPv6File = strings.TrimSpace(file)
		}
	}
}

func validateInputOptions(iType string, action lib.Action, onlyIPType lib.IPType) {
	if action != lib.ActionAdd && action != lib.ActionRemove {
		log.Fatalf("❌ [type %s | action %s] invalid action: must be add or remove", iType, action)
	}
	validateOnlyIPType(iType, action, onlyIPType)
}

func validateOnlyIPType(iType string, action lib.Action, onlyIPType lib.IPType) {
	switch onlyIPType {
	case "", lib.IPv4, lib.IPv6:
	default:
		log.Fatalf("❌ [type %s | action %s] invalid onlyIPType %q: must be ipv4 or ipv6", iType, action, onlyIPType)
	}
}

func countryWantedList(iType string, action lib.Action, lists any) map[string]bool {
	list, ok := stringList(lists)
	if !ok {
		log.Fatalf("❌ [type %s | action %s] invalid wantedList: must be an array of strings", iType, action)
	}
	wantList := make(map[string]bool)
	for _, want := range list {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList[want] = true
		}
	}
	return wantList
}

func stringList(value any) ([]string, bool) {
	switch value := value.(type) {
	case nil:
		return nil, true
	case []string:
		return value, true
	case []any:
		list := make([]string, 0, len(value))
		for _, item := range value {
			str, ok := item.(string)
			if !ok {
				return nil, false
			}
			list = append(list, str)
		}
		return list, true
	default:
		return nil, false
	}
}
