package maxmind

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/oschwald/geoip2-golang/v2"
	"github.com/oschwald/maxminddb-golang/v2"
)

func NewGeoLite2CountryMMDBOut(iType string, iDesc string, action lib.Action, opts ...lib.OutputOption) lib.OutputConverter {
	g := &GeoLite2CountryMMDBOut{
		Type:        iType,
		Action:      action,
		Description: iDesc,
		OutputName:  defaultGeoLite2CountryMMDBOutputName,
	}

	switch iType {
	case TypeGeoLite2CountryMMDBOut:
		g.OutputDir = defaultMaxmindOutputDir
	case TypeDBIPCountryMMDBOut:
		g.OutputDir = defaultDBIPOutputDir
	case TypeIPInfoCountryMMDBOut:
		g.OutputDir = defaultIPInfoOutputDir
	}

	for _, opt := range opts {
		if opt != nil {
			opt(g)
		}
	}

	return g
}

func WithMMDBOutOutputName(name string) lib.OutputOption {
	return func(c lib.OutputConverter) {
		name = strings.TrimSpace(name)
		if name != "" {
			c.(*GeoLite2CountryMMDBOut).OutputName = name
		}
	}
}

func WithMMDBOutOutputDir(dir string) lib.OutputOption {
	return func(c lib.OutputConverter) {
		dir = strings.TrimSpace(dir)
		if dir != "" {
			c.(*GeoLite2CountryMMDBOut).OutputDir = dir
		}
	}
}

func WithMMDBOutWantedList(lists []string) lib.OutputOption {
	return func(c lib.OutputConverter) {
		c.(*GeoLite2CountryMMDBOut).Want = lists
	}
}

func WithMMDBOutOverwriteList(lists []string) lib.OutputOption {
	return func(c lib.OutputConverter) {
		c.(*GeoLite2CountryMMDBOut).Overwrite = lists
	}
}

func WithMMDBOutExcludedList(lists []string) lib.OutputOption {
	return func(c lib.OutputConverter) {
		c.(*GeoLite2CountryMMDBOut).Exclude = lists
	}
}

func WithMMDBOutOnlyIPType(onlyIPType lib.IPType) lib.OutputOption {
	return func(c lib.OutputConverter) {
		c.(*GeoLite2CountryMMDBOut).OnlyIPType = onlyIPType
	}
}

func WithMMDBOutSourceMMDBURI(uri string) lib.OutputOption {
	return func(c lib.OutputConverter) {
		c.(*GeoLite2CountryMMDBOut).SourceMMDBURI = uri
	}
}

var (
	defaultGeoLite2CountryMMDBOutputName = "Country.mmdb"

	defaultMaxmindOutputDir = filepath.Join("./", "output", "maxmind")
	defaultDBIPOutputDir    = filepath.Join("./", "output", "db-ip")
	defaultIPInfoOutputDir  = filepath.Join("./", "output", "ipinfo")
)

// Reference: https://github.com/oschwald/geoip2-golang/blob/HEAD/models.go
var (
	zeroDBIPLanguageNames      dbipLanguageNames
	zeroDBIPContinent          dbipContinent
	zeroDBIPCountryRecord      dbipCountryRecord
	zeroDBIPRepresentedCountry dbipRepresentedCountry
	zeroDBIPCountry            dbipCountry
)

// Reference: https://ipinfo.io/lite
type ipInfoLite struct {
	ASN           string `maxminddb:"asn"`
	ASName        string `maxminddb:"as_name"`
	ASDomain      string `maxminddb:"as_domain"`
	Continent     string `maxminddb:"continent"`
	ContinentCode string `maxminddb:"continent_code"`
	Country       string `maxminddb:"country"`
	CountryCode   string `maxminddb:"country_code"`
}

// Reference: https://github.com/oschwald/geoip2-golang/blob/HEAD/models.go
type dbipLanguageNames struct {
	geoip2.Names

	// Persian localized name
	Persian string `json:"fa,omitzero" maxminddb:"fa"`
	// Korean localized name
	Korean string `json:"ko,omitzero" maxminddb:"ko"`
}

func (d dbipLanguageNames) HasData() bool {
	return d != zeroDBIPLanguageNames
}

// Reference: https://github.com/oschwald/geoip2-golang/blob/HEAD/models.go
type dbipContinent struct {
	geoip2.Continent

	Names dbipLanguageNames `json:"names,omitzero" maxminddb:"names"`
}

func (d dbipContinent) HasData() bool {
	return d != zeroDBIPContinent
}

// Reference: https://github.com/oschwald/geoip2-golang/blob/HEAD/models.go
type dbipCountryRecord struct {
	geoip2.CountryRecord

	Names dbipLanguageNames `json:"names,omitzero" maxminddb:"names"`
}

func (d dbipCountryRecord) HasData() bool {
	return d != zeroDBIPCountryRecord
}

// Reference: https://github.com/oschwald/geoip2-golang/blob/HEAD/models.go
type dbipRepresentedCountry struct {
	geoip2.RepresentedCountry

	Names dbipLanguageNames `json:"names,omitzero" maxminddb:"names"`
}

func (d dbipRepresentedCountry) HasData() bool {
	return d != zeroDBIPRepresentedCountry
}

// Reference: https://github.com/oschwald/geoip2-golang/blob/HEAD/models.go
type dbipCountry struct {
	Traits             geoip2.CountryTraits   `json:"traits,omitzero"              maxminddb:"traits"`
	Continent          dbipContinent          `json:"continent,omitzero"           maxminddb:"continent"`
	RepresentedCountry dbipRepresentedCountry `json:"represented_country,omitzero" maxminddb:"represented_country"`
	Country            dbipCountryRecord      `json:"country,omitzero"             maxminddb:"country"`
	RegisteredCountry  dbipCountryRecord      `json:"registered_country,omitzero"  maxminddb:"registered_country"`
}

func (d dbipCountry) HasData() bool {
	return d != zeroDBIPCountry
}

func NewGeoLite2CountryMMDBOutFromBytes(iType string, iDesc string, action lib.Action, data []byte) (lib.OutputConverter, error) {
	var tmp struct {
		OutputName string     `json:"outputName"`
		OutputDir  string     `json:"outputDir"`
		Want       []string   `json:"wantedList"`
		Overwrite  []string   `json:"overwriteList"`
		Exclude    []string   `json:"excludedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`

		SourceMMDBURI string `json:"sourceMMDBURI"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	return NewGeoLite2CountryMMDBOut(
		iType,
		iDesc,
		action,
		WithMMDBOutOutputName(tmp.OutputName),
		WithMMDBOutOutputDir(tmp.OutputDir),
		WithMMDBOutWantedList(tmp.Want),
		WithMMDBOutOverwriteList(tmp.Overwrite),
		WithMMDBOutExcludedList(tmp.Exclude),
		WithMMDBOutOnlyIPType(tmp.OnlyIPType),
		WithMMDBOutSourceMMDBURI(tmp.SourceMMDBURI),
	), nil
}

func (g *GeoLite2CountryMMDBOut) GetExtraInfo() (map[string]any, error) {
	if strings.TrimSpace(g.SourceMMDBURI) == "" {
		return nil, nil
	}

	var content []byte
	var err error
	switch {
	case strings.HasPrefix(strings.ToLower(g.SourceMMDBURI), "http://"), strings.HasPrefix(strings.ToLower(g.SourceMMDBURI), "https://"):
		content, err = lib.GetRemoteURLContent(g.SourceMMDBURI)
	default:
		content, err = os.ReadFile(g.SourceMMDBURI)
	}
	if err != nil {
		return nil, err
	}

	db, err := maxminddb.OpenBytes(content)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	infoList := make(map[string]any)
	for network := range db.Networks() {
		switch g.Type {
		case TypeGeoLite2CountryMMDBOut:
			var record geoip2.Country
			err := network.Decode(&record)
			if err != nil {
				return nil, err
			}

			switch {
			case strings.TrimSpace(record.Country.ISOCode) != "":
				countryCode := strings.ToUpper(strings.TrimSpace(record.Country.ISOCode))
				if _, found := infoList[countryCode]; !found {
					infoList[countryCode] = geoip2.Country{
						Continent: record.Continent,
						Country:   record.Country,
					}
				}

			case strings.TrimSpace(record.RegisteredCountry.ISOCode) != "":
				countryCode := strings.ToUpper(strings.TrimSpace(record.RegisteredCountry.ISOCode))
				if _, found := infoList[countryCode]; !found {
					infoList[countryCode] = geoip2.Country{
						Continent: record.Continent,
						Country:   record.RegisteredCountry,
					}
				}

			case strings.TrimSpace(record.RepresentedCountry.ISOCode) != "":
				countryCode := strings.ToUpper(strings.TrimSpace(record.RepresentedCountry.ISOCode))
				if _, found := infoList[countryCode]; !found {
					infoList[countryCode] = geoip2.Country{
						Continent: record.Continent,
						Country: geoip2.CountryRecord{
							Names:             record.RepresentedCountry.Names,
							ISOCode:           record.RepresentedCountry.ISOCode,
							GeoNameID:         record.RepresentedCountry.GeoNameID,
							IsInEuropeanUnion: record.RepresentedCountry.IsInEuropeanUnion,
						},
					}
				}
			}

		case TypeDBIPCountryMMDBOut:
			var record dbipCountry
			err := network.Decode(&record)
			if err != nil {
				return nil, err
			}

			switch {
			case strings.TrimSpace(record.Country.ISOCode) != "":
				countryCode := strings.ToUpper(strings.TrimSpace(record.Country.ISOCode))
				if _, found := infoList[countryCode]; !found {
					infoList[countryCode] = dbipCountry{
						Continent: record.Continent,
						Country:   record.Country,
					}
				}

			case strings.TrimSpace(record.RegisteredCountry.ISOCode) != "":
				countryCode := strings.ToUpper(strings.TrimSpace(record.RegisteredCountry.ISOCode))
				if _, found := infoList[countryCode]; !found {
					infoList[countryCode] = dbipCountry{
						Continent: record.Continent,
						Country:   record.RegisteredCountry,
					}
				}

			case strings.TrimSpace(record.RepresentedCountry.ISOCode) != "":
				countryCode := strings.ToUpper(strings.TrimSpace(record.RepresentedCountry.ISOCode))
				if _, found := infoList[countryCode]; !found {
					infoList[countryCode] = dbipCountry{
						Continent: record.Continent,
						Country: dbipCountryRecord{
							CountryRecord: geoip2.CountryRecord{
								ISOCode:           record.RepresentedCountry.ISOCode,
								GeoNameID:         record.RepresentedCountry.GeoNameID,
								IsInEuropeanUnion: record.RepresentedCountry.IsInEuropeanUnion,
							},
							Names: record.RepresentedCountry.Names,
						},
					}
				}
			}

		case TypeIPInfoCountryMMDBOut:
			var record ipInfoLite
			err := network.Decode(&record)
			if err != nil {
				return nil, err
			}
			countryCode := strings.ToUpper(strings.TrimSpace(record.CountryCode))
			if _, found := infoList[countryCode]; !found {
				record.ASN = ""
				record.ASName = ""
				record.ASDomain = ""
				infoList[countryCode] = record
			}

		default:
			return nil, lib.ErrNotSupportedFormat
		}

	}

	if len(infoList) == 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] no extra info found in the source MMDB file: %s", g.Type, g.Action, g.SourceMMDBURI)
	}

	return infoList, nil
}
