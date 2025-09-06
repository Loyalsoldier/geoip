package maxmind

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/oschwald/geoip2-golang"
	"github.com/oschwald/maxminddb-golang"
)

var (
	defaultGeoLite2CountryMMDBOutputName = "Country.mmdb"

	defaultMaxmindOutputDir = filepath.Join("./", "output", "maxmind")
	defaultDBIPOutputDir    = filepath.Join("./", "output", "db-ip")
	defaultIPInfoOutputDir  = filepath.Join("./", "output", "ipinfo")
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

func newGeoLite2CountryMMDBOut(iType string, iDesc string, action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
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

	if tmp.OutputName == "" {
		tmp.OutputName = defaultGeoLite2CountryMMDBOutputName
	}

	if tmp.OutputDir == "" {
		switch iType {
		case TypeGeoLite2CountryMMDBOut:
			tmp.OutputDir = defaultMaxmindOutputDir

		case TypeDBIPCountryMMDBOut:
			tmp.OutputDir = defaultDBIPOutputDir

		case TypeIPInfoCountryMMDBOut:
			tmp.OutputDir = defaultIPInfoOutputDir
		}
	}

	return &GeoLite2CountryMMDBOut{
		Type:        iType,
		Action:      action,
		Description: iDesc,
		OutputName:  tmp.OutputName,
		OutputDir:   tmp.OutputDir,
		Want:        tmp.Want,
		Overwrite:   tmp.Overwrite,
		Exclude:     tmp.Exclude,
		OnlyIPType:  tmp.OnlyIPType,

		SourceMMDBURI: tmp.SourceMMDBURI,
	}, nil
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

	db, err := maxminddb.FromBytes(content)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	infoList := make(map[string]any)
	networks := db.Networks(maxminddb.SkipAliasedNetworks)
	for networks.Next() {
		switch g.Type {
		case TypeGeoLite2CountryMMDBOut, TypeDBIPCountryMMDBOut:
			var record geoip2.Country
			_, err := networks.Network(&record)
			if err != nil {
				return nil, err
			}

			switch {
			case strings.TrimSpace(record.Country.IsoCode) != "":
				countryCode := strings.ToUpper(strings.TrimSpace(record.Country.IsoCode))
				if _, found := infoList[countryCode]; !found {
					infoList[countryCode] = geoip2.Country{
						Continent: record.Continent,
						Country:   record.Country,
					}
				}

			case strings.TrimSpace(record.RegisteredCountry.IsoCode) != "":
				countryCode := strings.ToUpper(strings.TrimSpace(record.RegisteredCountry.IsoCode))
				if _, found := infoList[countryCode]; !found {
					infoList[countryCode] = geoip2.Country{
						Continent: record.Continent,
						Country:   record.RegisteredCountry,
					}
				}

			case strings.TrimSpace(record.RepresentedCountry.IsoCode) != "":
				countryCode := strings.ToUpper(strings.TrimSpace(record.RepresentedCountry.IsoCode))
				if _, found := infoList[countryCode]; !found {
					infoList[countryCode] = geoip2.Country{
						Continent: record.Continent,
						Country: struct {
							Names             map[string]string `maxminddb:"names"`
							IsoCode           string            `maxminddb:"iso_code"`
							GeoNameID         uint              `maxminddb:"geoname_id"`
							IsInEuropeanUnion bool              `maxminddb:"is_in_european_union"`
						}{
							Names:             record.RepresentedCountry.Names,
							IsoCode:           record.RepresentedCountry.IsoCode,
							GeoNameID:         record.RepresentedCountry.GeoNameID,
							IsInEuropeanUnion: record.RepresentedCountry.IsInEuropeanUnion,
						},
					}
				}
			}

		case TypeIPInfoCountryMMDBOut:
			var record ipInfoLite
			_, err := networks.Network(&record)
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

	if networks.Err() != nil {
		return nil, networks.Err()
	}

	if len(infoList) == 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] no extra info found in the source MMDB file: %s", g.Type, g.Action, g.SourceMMDBURI)
	}

	return infoList, nil
}
