package maxmind

import (
	"encoding/json"
	"fmt"
	"log"
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

func defaultMMDBOutputDirFor(iType string) string {
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

func NewGeoLite2CountryMMDBOut(iType string, iDesc string, action lib.Action, opts ...lib.OutputOption) lib.OutputConverter {
	g := &geolite2_country_mmdb_out{
		Type:        iType,
		Action:      action,
		Description: iDesc,
		OutputName:  defaultGeoLite2CountryMMDBOutputName,
		OutputDir:   defaultMMDBOutputDirFor(iType),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(g)
		}
	}

	if g.Type == "" {
		log.Fatalf("❌ [type %s | action %s] missing type", g.Type, g.Action)
	}

	return g
}

// WithOutputName sets the output name of the output converter.
func WithOutputName(name string) lib.OutputOption {
	return func(g lib.OutputConverter) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		g.(*geolite2_country_mmdb_out).OutputName = name
	}
}

// WithOutputDir sets the output directory of the output converter.
func WithOutputDir(dir string) lib.OutputOption {
	return func(g lib.OutputConverter) {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			return
		}
		g.(*geolite2_country_mmdb_out).OutputDir = dir
	}
}

// WithOutputWantedList sets the wanted list of the output converter.
func WithOutputWantedList(lists []string) lib.OutputOption {
	return func(g lib.OutputConverter) {
		g.(*geolite2_country_mmdb_out).Want = lists
	}
}

// WithOutputOverwriteList sets the overwrite list of the output converter.
func WithOutputOverwriteList(lists []string) lib.OutputOption {
	return func(g lib.OutputConverter) {
		g.(*geolite2_country_mmdb_out).Overwrite = lists
	}
}

// WithOutputExcludedList sets the excluded list of the output converter.
func WithOutputExcludedList(lists []string) lib.OutputOption {
	return func(g lib.OutputConverter) {
		g.(*geolite2_country_mmdb_out).Exclude = lists
	}
}

// WithOutputOnlyIPType sets the only IP type of the output converter.
func WithOutputOnlyIPType(onlyIPType lib.IPType) lib.OutputOption {
	return func(g lib.OutputConverter) {
		g.(*geolite2_country_mmdb_out).OnlyIPType = onlyIPType
	}
}

// WithSourceMMDBURI sets the source MMDB URI of the output converter.
func WithSourceMMDBURI(uri string) lib.OutputOption {
	return func(g lib.OutputConverter) {
		g.(*geolite2_country_mmdb_out).SourceMMDBURI = strings.TrimSpace(uri)
	}
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

	return NewGeoLite2CountryMMDBOut(iType, iDesc, action,
		WithOutputName(tmp.OutputName),
		WithOutputDir(tmp.OutputDir),
		WithOutputWantedList(tmp.Want),
		WithOutputOverwriteList(tmp.Overwrite),
		WithOutputExcludedList(tmp.Exclude),
		WithOutputOnlyIPType(tmp.OnlyIPType),
		WithSourceMMDBURI(tmp.SourceMMDBURI),
	), nil
}

func (g *geolite2_country_mmdb_out) GetExtraInfo() (map[string]any, error) {
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
