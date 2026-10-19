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

func newGeoLite2CountryMMDBIn(iType string, iDesc string, action lib.Action, opts ...lib.InputOption) (lib.InputConverter, error) {
	g := &geolite2_country_mmdb_in{
		Type:        iType,
		Action:      action,
		Description: iDesc,
		URI:         getDefaultMMDBInputURI(iType),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(g)
		}
	}

	if g.URI == "" {
		return nil, fmt.Errorf("❌ [type %s | action %s] uri must be specified", g.Type, g.Action)
	}

	return g, nil
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
		WithURI(tmp.URI),
		WithInputWantedList(tmp.Want),
		WithInputOnlyIPType(tmp.OnlyIPType),
	)
}

func getDefaultMMDBInputURI(iType string) string {
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

// WithURI sets the URI of MMDB input. An empty uri means the default file
// of the corresponding input format.
func WithURI(uri string) lib.InputOption {
	uri = strings.TrimSpace(uri)
	return func(g lib.InputConverter) {
		inputURI := uri
		if inputURI == "" {
			inputURI = getDefaultMMDBInputURI(g.(*geolite2_country_mmdb_in).Type)
		}

		g.(*geolite2_country_mmdb_in).URI = inputURI
	}
}

// WithInputWantedList is valid for MMDB inputs and GeoLite2 country CSV input.
func WithInputWantedList(lists []string) lib.InputOption {
	return func(g lib.InputConverter) {
		wantList := make(map[string]bool)
		for _, want := range lists {
			if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
				wantList[want] = true
			}
		}

		switch g := g.(type) {
		case *geolite2_country_mmdb_in:
			g.Want = wantList
		case *geolite2_country_csv_in:
			g.Want = wantList
		}
	}
}

// WithInputOnlyIPType is valid for all input formats in this package.
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

// WithIPv4File is valid for GeoLite2 country CSV and ASN CSV inputs.
func WithIPv4File(file string) lib.InputOption {
	return func(g lib.InputConverter) {
		file = strings.TrimSpace(file)
		switch g := g.(type) {
		case *geolite2_country_csv_in:
			g.IPv4File = file
		case *geolite2_asn_csv_in:
			g.IPv4File = file
		}
	}
}

// WithIPv6File is valid for GeoLite2 country CSV and ASN CSV inputs.
func WithIPv6File(file string) lib.InputOption {
	return func(g lib.InputConverter) {
		file = strings.TrimSpace(file)
		switch g := g.(type) {
		case *geolite2_country_csv_in:
			g.IPv6File = file
		case *geolite2_asn_csv_in:
			g.IPv6File = file
		}
	}
}
