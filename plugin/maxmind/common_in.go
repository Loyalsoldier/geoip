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

// WithURI is only for MMDB input formats
func WithURI(uri string) lib.InputOption {
	return func(g lib.InputConverter) {
		g.(*geolite2_country_mmdb_in).URI = strings.TrimSpace(uri)
	}
}

// WithIPv4File is only for GeoLite2 CSV input formats
func WithIPv4File(file string) lib.InputOption {
	return func(i lib.InputConverter) {
		switch g := i.(type) {
		case *geolite2_asn_csv_in:
			g.IPv4File = strings.TrimSpace(file)
		case *geolite2_country_csv_in:
			g.IPv4File = strings.TrimSpace(file)
		default:
			log.Fatalf("❌ [type %s | action %s] option WithIPv4File is not supported", i.GetType(), i.GetAction())
		}
	}
}

// WithIPv6File is only for GeoLite2 CSV input formats
func WithIPv6File(file string) lib.InputOption {
	return func(i lib.InputConverter) {
		switch g := i.(type) {
		case *geolite2_asn_csv_in:
			g.IPv6File = strings.TrimSpace(file)
		case *geolite2_country_csv_in:
			g.IPv6File = strings.TrimSpace(file)
		default:
			log.Fatalf("❌ [type %s | action %s] option WithIPv6File is not supported", i.GetType(), i.GetAction())
		}
	}
}

// For maxmindGeoLite2ASNCSV input format, the values of lists are ASNs.
func WithInputWantedList(lists []string) lib.InputOption {
	return func(i lib.InputConverter) {
		if g, ok := i.(*geolite2_asn_csv_in); ok {
			g.addWantedASNList(lists)
			return
		}

		wantList := make(map[string]bool)
		for _, want := range lists {
			if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
				wantList[want] = true
			}
		}

		switch g := i.(type) {
		case *geolite2_country_mmdb_in:
			g.Want = wantList
		case *geolite2_country_csv_in:
			g.Want = wantList
		default:
			log.Fatalf("❌ [type %s | action %s] option WithInputWantedList is not supported", i.GetType(), i.GetAction())
		}
	}
}

func WithInputOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(i lib.InputConverter) {
		switch g := i.(type) {
		case *geolite2_country_mmdb_in:
			g.OnlyIPType = onlyIPType
		case *geolite2_asn_csv_in:
			g.OnlyIPType = onlyIPType
		case *geolite2_country_csv_in:
			g.OnlyIPType = onlyIPType
		default:
			log.Fatalf("❌ [type %s | action %s] option WithInputOnlyIPType is not supported", i.GetType(), i.GetAction())
		}
	}
}
