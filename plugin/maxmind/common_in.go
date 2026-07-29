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

func newCountryMMDBIn(iType, iDesc string, action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	g := &country_mmdb_in{
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
		switch iType {
		case TypeGeoLite2CountryMMDBIn:
			g.URI = defaultGeoLite2CountryMMDBFile
		case TypeDBIPCountryMMDBIn:
			g.URI = defaultDBIPCountryMMDBFile
		case TypeIPInfoCountryMMDBIn:
			g.URI = defaultIPInfoCountryMMDBFile
		}
	}

	return g
}

func WithInputURI(uri string) lib.InputOption {
	return func(g lib.InputConverter) {
		g.(*country_mmdb_in).URI = strings.TrimSpace(uri)
	}
}

func WithInputWantedList(lists []string) lib.InputOption {
	return func(g lib.InputConverter) {
		switch g := g.(type) {
		case *country_mmdb_in:
			g.Want = filterInputWantedList(lists)
		case *country_csv_in:
			g.Want = filterInputWantedList(lists)
		case *asn_csv_in:
			g.Want = filterASNInputWantedList(lib.WantedListExtended{TypeSlice: lists})
		default:
			panic("unsupported input converter")
		}
	}
}

func WithInputWantedListExtended(lists lib.WantedListExtended) lib.InputOption {
	return func(g lib.InputConverter) {
		g.(*asn_csv_in).Want = filterASNInputWantedList(lists)
	}
}

func WithCountryCodeFile(file string) lib.InputOption {
	return func(g lib.InputConverter) {
		g.(*country_csv_in).CountryCodeFile = strings.TrimSpace(file)
	}
}

func WithIPv4File(file string) lib.InputOption {
	return func(g lib.InputConverter) {
		switch g := g.(type) {
		case *country_csv_in:
			g.IPv4File = strings.TrimSpace(file)
		case *asn_csv_in:
			g.IPv4File = strings.TrimSpace(file)
		default:
			panic("unsupported input converter")
		}
	}
}

func WithIPv6File(file string) lib.InputOption {
	return func(g lib.InputConverter) {
		switch g := g.(type) {
		case *country_csv_in:
			g.IPv6File = strings.TrimSpace(file)
		case *asn_csv_in:
			g.IPv6File = strings.TrimSpace(file)
		default:
			panic("unsupported input converter")
		}
	}
}

func WithInputOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(g lib.InputConverter) {
		switch g := g.(type) {
		case *country_mmdb_in:
			g.OnlyIPType = onlyIPType
		case *country_csv_in:
			g.OnlyIPType = onlyIPType
		case *asn_csv_in:
			g.OnlyIPType = onlyIPType
		default:
			panic("unsupported input converter")
		}
	}
}

func filterInputWantedList(lists []string) map[string]bool {
	wantList := make(map[string]bool)
	for _, want := range lists {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList[want] = true
		}
	}
	return wantList
}

func newCountryMMDBInFromBytes(iType, iDesc string, action lib.Action, data []byte) (lib.InputConverter, error) {
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

	return newCountryMMDBIn(
		iType,
		iDesc,
		action,
		WithInputURI(tmp.URI),
		WithInputWantedList(tmp.Want),
		WithInputOnlyIPType(tmp.OnlyIPType),
	), nil
}
