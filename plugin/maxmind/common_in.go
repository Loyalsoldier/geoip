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

func newGeoLite2CountryMMDBIn(iType, iDesc string, action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	g := &geoLite2CountryMMDBIn{
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
		switch g.Type {
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
	return func(converter lib.InputConverter) {
		switch g := converter.(type) {
		case *geoLite2CountryMMDBIn:
			g.URI = strings.TrimSpace(uri)
		default:
			log.Fatalf("❌ maxmind WithURI does not support %T", converter)
		}
	}
}

func WithInputWantedList(lists []string) lib.InputOption {
	return func(converter lib.InputConverter) {
		wantList := make(map[string]bool)
		for _, want := range lists {
			if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
				wantList[want] = true
			}
		}

		switch g := converter.(type) {
		case *geoLite2CountryMMDBIn:
			g.Want = wantList
		case *geoLite2CountryCSVIn:
			g.Want = wantList
		default:
			log.Fatalf("❌ maxmind WithInputWantedList does not support %T", converter)
		}
	}
}

func WithInputOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(converter lib.InputConverter) {
		ipType := lib.IPType(strings.ToLower(strings.TrimSpace(string(onlyIPType))))
		switch g := converter.(type) {
		case *geoLite2CountryMMDBIn:
			g.OnlyIPType = ipType
		case *geoLite2CountryCSVIn:
			g.OnlyIPType = ipType
		case *geoLite2ASNCSVIn:
			g.OnlyIPType = ipType
		default:
			log.Fatalf("❌ maxmind WithInputOnlyIPType does not support %T", converter)
		}
	}
}

func WithIPv4File(file string) lib.InputOption {
	return func(converter lib.InputConverter) {
		switch g := converter.(type) {
		case *geoLite2CountryCSVIn:
			g.IPv4File = strings.TrimSpace(file)
		case *geoLite2ASNCSVIn:
			g.IPv4File = strings.TrimSpace(file)
		default:
			log.Fatalf("❌ maxmind WithIPv4File does not support %T", converter)
		}
	}
}

func WithIPv6File(file string) lib.InputOption {
	return func(converter lib.InputConverter) {
		switch g := converter.(type) {
		case *geoLite2CountryCSVIn:
			g.IPv6File = strings.TrimSpace(file)
		case *geoLite2ASNCSVIn:
			g.IPv6File = strings.TrimSpace(file)
		default:
			log.Fatalf("❌ maxmind WithIPv6File does not support %T", converter)
		}
	}
}

func validateInput(g lib.InputConverter, onlyIPType lib.IPType) {
	if g.GetAction() != lib.ActionAdd && g.GetAction() != lib.ActionRemove {
		log.Fatalf("❌ [type %s | action %s] invalid action: expected add or remove", g.GetType(), g.GetAction())
	}
	validateOnlyIPType(g.GetType(), g.GetAction(), onlyIPType)
}

func validateOnlyIPType(iType string, action lib.Action, onlyIPType lib.IPType) {
	switch onlyIPType {
	case "", lib.IPv4, lib.IPv6:
	default:
		log.Fatalf("❌ [type %s | action %s] invalid onlyIPType %q: expected ipv4 or ipv6", iType, action, onlyIPType)
	}
}
