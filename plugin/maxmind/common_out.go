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

func defaultMMDBOutputDirForType(iType string) string {
	switch iType {
	case TypeGeoLite2CountryMMDBOut:
		return defaultMaxmindOutputDir
	case TypeDBIPCountryMMDBOut:
		return defaultDBIPOutputDir
	case TypeIPInfoCountryMMDBOut:
		return defaultIPInfoOutputDir
	}
	return ""
}

func newGeoLite2CountryMMDBOut(iType string, iDesc string, action lib.Action, opts ...lib.OutputOption) (lib.OutputConverter, error) {
	g := &geoLite2CountryMMDBOut{
		Type:        iType,
		Action:      action,
		Description: iDesc,
		OutputName:  defaultGeoLite2CountryMMDBOutputName,
		OutputDir:   defaultMMDBOutputDirForType(iType),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(g)
		}
	}

	return g, nil
}

// NewGeoLite2CountryMMDBOut creates an output converter of type maxmindMMDB
func NewGeoLite2CountryMMDBOut(action lib.Action, opts ...lib.OutputOption) (lib.OutputConverter, error) {
	return newGeoLite2CountryMMDBOut(TypeGeoLite2CountryMMDBOut, DescGeoLite2CountryMMDBOut, action, opts...)
}

// NewDBIPCountryMMDBOut creates an output converter of type dbipCountryMMDB
func NewDBIPCountryMMDBOut(action lib.Action, opts ...lib.OutputOption) (lib.OutputConverter, error) {
	return newGeoLite2CountryMMDBOut(TypeDBIPCountryMMDBOut, DescDBIPCountryMMDBOut, action, opts...)
}

// NewIPInfoCountryMMDBOut creates an output converter of type ipinfoCountryMMDB
func NewIPInfoCountryMMDBOut(action lib.Action, opts ...lib.OutputOption) (lib.OutputConverter, error) {
	return newGeoLite2CountryMMDBOut(TypeIPInfoCountryMMDBOut, DescIPInfoCountryMMDBOut, action, opts...)
}

func WithMMDBOutputName(name string) lib.OutputOption {
	return func(g lib.OutputConverter) {
		name = strings.TrimSpace(name)
		if name == "" {
			name = defaultGeoLite2CountryMMDBOutputName
		}

		g.(*geoLite2CountryMMDBOut).OutputName = name
	}
}

func WithMMDBOutputDir(dir string) lib.OutputOption {
	return func(g lib.OutputConverter) {
		dir := strings.TrimSpace(dir)
		if dir == "" {
			dir = defaultMMDBOutputDirForType(g.(*geoLite2CountryMMDBOut).Type)
		}

		g.(*geoLite2CountryMMDBOut).OutputDir = dir
	}
}

func WithMMDBOutWantedList(lists []string) lib.OutputOption {
	return func(g lib.OutputConverter) {
		g.(*geoLite2CountryMMDBOut).Want = lists
	}
}

func WithMMDBOutOverwriteList(lists []string) lib.OutputOption {
	return func(g lib.OutputConverter) {
		g.(*geoLite2CountryMMDBOut).Overwrite = lists
	}
}

func WithMMDBOutExcludedList(lists []string) lib.OutputOption {
	return func(g lib.OutputConverter) {
		g.(*geoLite2CountryMMDBOut).Exclude = lists
	}
}

func WithMMDBOutOnlyIPType(onlyIPType lib.IPType) lib.OutputOption {
	return func(g lib.OutputConverter) {
		g.(*geoLite2CountryMMDBOut).OnlyIPType = onlyIPType
	}
}

func WithSourceMMDBURI(uri string) lib.OutputOption {
	return func(g lib.OutputConverter) {
		g.(*geoLite2CountryMMDBOut).SourceMMDBURI = strings.TrimSpace(uri)
	}
}

func newGeoLite2CountryMMDBOutFromBytes(iType string, iDesc string, action lib.Action, data []byte) (lib.OutputConverter, error) {
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
			return nil, fmt.Errorf("❌ [type %s | action %s] %w", iType, action, err)
		}
	}

	return newGeoLite2CountryMMDBOut(
		iType,
		iDesc,
		action,
		WithMMDBOutputName(tmp.OutputName),
		WithMMDBOutputDir(tmp.OutputDir),
		WithMMDBOutWantedList(tmp.Want),
		WithMMDBOutOverwriteList(tmp.Overwrite),
		WithMMDBOutExcludedList(tmp.Exclude),
		WithMMDBOutOnlyIPType(tmp.OnlyIPType),
		WithSourceMMDBURI(tmp.SourceMMDBURI),
	)
}

func (g *geoLite2CountryMMDBOut) GetExtraInfo() (map[string]any, error) {
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

	err = db.Verify()
	if err != nil {
		return nil, err
	}

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
		return nil, fmt.Errorf("no extra info found in the source MMDB file: %s", g.SourceMMDBURI)
	}

	return infoList, nil
}
