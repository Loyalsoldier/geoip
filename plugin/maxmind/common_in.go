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

func newGeoLite2CountryMMDBIn(iType string, iDesc string, action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	g := &geolite2_country_mmdb_in{
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
		switch g.Type {
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

func newGeoLite2CountryMMDBInFromBytes(iType string, iDesc string, action lib.Action, data []byte) (lib.InputConverter, error) {
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

	return newGeoLite2CountryMMDBIn(
		iType,
		iDesc,
		action,
		WithURI(tmp.URI),
		WithInputWantedList(tmp.Want),
		WithInputOnlyIPType(tmp.OnlyIPType),
	), nil
}

// WithURI is used by input formats maxmindMMDB, dbipCountryMMDB and ipinfoCountryMMDB
func WithURI(uri string) lib.InputOption {
	return func(g lib.InputConverter) {
		g.(*geolite2_country_mmdb_in).URI = strings.TrimSpace(uri)
	}
}

// WithIPv4File is used by input formats maxmindGeoLite2ASNCSV and maxmindGeoLite2CountryCSV
func WithIPv4File(file string) lib.InputOption {
	return func(g lib.InputConverter) {
		switch g := g.(type) {
		case *geolite2_asn_csv_in:
			g.IPv4File = strings.TrimSpace(file)
		case *geolite2_country_csv_in:
			g.IPv4File = strings.TrimSpace(file)
		}
	}
}

// WithIPv6File is used by input formats maxmindGeoLite2ASNCSV and maxmindGeoLite2CountryCSV
func WithIPv6File(file string) lib.InputOption {
	return func(g lib.InputConverter) {
		switch g := g.(type) {
		case *geolite2_asn_csv_in:
			g.IPv6File = strings.TrimSpace(file)
		case *geolite2_country_csv_in:
			g.IPv6File = strings.TrimSpace(file)
		}
	}
}

// WithInputWantedList is used by all input formats of this package.
// For input format maxmindGeoLite2ASNCSV, the lists are ASNs, eg. ["AS13335", "AS15169"].
func WithInputWantedList(lists []string) lib.InputOption {
	return func(g lib.InputConverter) {
		switch g := g.(type) {
		case *geolite2_country_mmdb_in:
			g.Want = newWantedList(lists)
		case *geolite2_country_csv_in:
			g.Want = newWantedList(lists)
		case *geolite2_asn_csv_in:
			g.Want = newASNWantedList(lib.WantedListExtended{TypeSlice: lists})
		}
	}
}

// WithInputOnlyIPType is used by all input formats of this package
func WithInputOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(g lib.InputConverter) {
		switch g := g.(type) {
		case *geolite2_country_mmdb_in:
			g.OnlyIPType = onlyIPType
		case *geolite2_country_csv_in:
			g.OnlyIPType = onlyIPType
		case *geolite2_asn_csv_in:
			g.OnlyIPType = onlyIPType
		}
	}
}

func newWantedList(lists []string) map[string]bool {
	wantList := make(map[string]bool)
	for _, want := range lists {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList[want] = true
		}
	}

	return wantList
}
