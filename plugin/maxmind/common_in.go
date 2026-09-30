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

// WithURI sets the URI of MMDB file for input converters of MMDB format
func WithURI(uri string) lib.InputOption {
	return func(i lib.InputConverter) {
		if g, ok := i.(*geolite2_country_mmdb_in); ok {
			g.URI = strings.TrimSpace(uri)
		}
	}
}

// WithIPv4File sets the IPv4 CSV file for input converters of CSV format
func WithIPv4File(file string) lib.InputOption {
	return func(i lib.InputConverter) {
		switch g := i.(type) {
		case *geolite2_asn_csv_in:
			g.IPv4File = strings.TrimSpace(file)
		case *geolite2_country_csv_in:
			g.IPv4File = strings.TrimSpace(file)
		}
	}
}

// WithIPv6File sets the IPv6 CSV file for input converters of CSV format
func WithIPv6File(file string) lib.InputOption {
	return func(i lib.InputConverter) {
		switch g := i.(type) {
		case *geolite2_asn_csv_in:
			g.IPv6File = strings.TrimSpace(file)
		case *geolite2_country_csv_in:
			g.IPv6File = strings.TrimSpace(file)
		}
	}
}

// WithCountryCodeFile sets the country code file for GeoLite2 country CSV input converter
func WithCountryCodeFile(file string) lib.InputOption {
	return func(i lib.InputConverter) {
		if g, ok := i.(*geolite2_country_csv_in); ok {
			g.CountryCodeFile = strings.TrimSpace(file)
		}
	}
}

// WithInputWantedList sets the wanted list for input converters except GeoLite2 ASN CSV,
// which uses WithASNWantedList instead
func WithInputWantedList(lists []string) lib.InputOption {
	return func(i lib.InputConverter) {
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
		}
	}
}

// WithASNWantedList sets the wanted list for GeoLite2 ASN CSV input converter
func WithASNWantedList(lists lib.WantedListExtended) lib.InputOption {
	return func(i lib.InputConverter) {
		g, ok := i.(*geolite2_asn_csv_in)
		if !ok {
			return
		}

		wantList := make(map[string][]string) // map[asn][]listname or map[asn][]asn

		for list, asnList := range lists.TypeMap {
			list = strings.ToUpper(strings.TrimSpace(list))
			if list == "" {
				continue
			}

			for _, asn := range asnList {
				asn = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(asn)), "as")
				if asn == "" {
					continue
				}

				wantList[asn] = append(wantList[asn], list)
			}
		}

		for _, asn := range lists.TypeSlice {
			asn = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(asn)), "as")
			if asn == "" {
				continue
			}

			wantList[asn] = []string{"AS" + asn}
		}

		g.Want = wantList
	}
}

// WithInputOnlyIPType sets the only IP type for input converters
func WithInputOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(i lib.InputConverter) {
		switch g := i.(type) {
		case *geolite2_country_mmdb_in:
			g.OnlyIPType = onlyIPType
		case *geolite2_asn_csv_in:
			g.OnlyIPType = onlyIPType
		case *geolite2_country_csv_in:
			g.OnlyIPType = onlyIPType
		}
	}
}
