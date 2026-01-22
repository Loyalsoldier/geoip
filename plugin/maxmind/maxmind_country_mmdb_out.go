package maxmind

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/oschwald/geoip2-golang/v2"
)

const (
	TypeGeoLite2CountryMMDBOut = "maxmindMMDB"
	DescGeoLite2CountryMMDBOut = "Convert data to MaxMind mmdb database format"
)

func init() {
	lib.RegisterOutputConfigCreator(TypeGeoLite2CountryMMDBOut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return newGeoLite2CountryMMDBOut(TypeGeoLite2CountryMMDBOut, DescGeoLite2CountryMMDBOut, action, data)
	})
	lib.RegisterOutputConverter(TypeGeoLite2CountryMMDBOut, &GeoLite2CountryMMDBOut{
		Description: DescGeoLite2CountryMMDBOut,
	})
}

type GeoLite2CountryMMDBOut struct {
	Type        string
	Action      lib.Action
	Description string
	OutputName  string
	OutputDir   string
	Want        []string
	Overwrite   []string
	Exclude     []string
	OnlyIPType  lib.IPType

	SourceMMDBURI string
}

func (g *GeoLite2CountryMMDBOut) GetType() string {
	return g.Type
}

func (g *GeoLite2CountryMMDBOut) GetAction() lib.Action {
	return g.Action
}

func (g *GeoLite2CountryMMDBOut) GetDescription() string {
	return g.Description
}

func (g *GeoLite2CountryMMDBOut) Output(container lib.Container) error {
	dbName := ""
	dbDesc := ""
	dbLanguages := []string{"en"}
	recordSize := 24

	switch g.Type {
	case TypeGeoLite2CountryMMDBOut:
		dbName = "GeoLite2-Country"
		dbDesc = "Customized GeoLite2 Country database"
		dbLanguages = []string{"de", "en", "es", "fr", "ja", "pt-BR", "ru", "zh-CN"}

	case TypeDBIPCountryMMDBOut:
		dbName = "DBIP-Country-Lite"
		dbDesc = "Customized DB-IP Country Lite database"
		dbLanguages = []string{"de", "en", "es", "fr", "ja", "pt-BR", "ru", "zh-CN", "fa", "ko"}

	case TypeIPInfoCountryMMDBOut:
		dbName = "IPInfo-Lite"
		dbDesc = "Customized IPInfo Lite database"
		recordSize = 32
	}

	writer, err := mmdbwriter.New(
		mmdbwriter.Options{
			DatabaseType:            dbName,
			Description:             map[string]string{"en": dbDesc},
			Languages:               dbLanguages,
			RecordSize:              recordSize,
			IncludeReservedNetworks: true,
		},
	)
	if err != nil {
		return err
	}

	// Get extra info
	extraInfo, err := g.GetExtraInfo()
	if err != nil {
		return err
	}

	updated := false
	for _, name := range g.filterAndSortList(container) {
		entry, found := container.GetEntry(name)
		if !found {
			log.Printf("❌ entry %s not found\n", name)
			continue
		}

		if err := g.marshalData(writer, entry, extraInfo); err != nil {
			return err
		}

		updated = true
	}

	if updated {
		return g.writeFile(g.OutputName, writer)
	}

	return nil
}

func (g *GeoLite2CountryMMDBOut) filterAndSortList(container lib.Container) []string {
	/*
		Note: The IPs and/or CIDRs of the latter list will overwrite those of the former one
		when duplicated data found due to MaxMind mmdb file format constraint.

		Be sure to place the name of the most important list at last
		when writing wantedList and overwriteList in config file.

		The order of names in wantedList has a higher priority than which of the overwriteList.
	*/

	excludeMap := make(map[string]bool)
	for _, exclude := range g.Exclude {
		if exclude = strings.ToUpper(strings.TrimSpace(exclude)); exclude != "" {
			excludeMap[exclude] = true
		}
	}

	wantList := make([]string, 0, len(g.Want))
	for _, want := range g.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" && !excludeMap[want] {
			wantList = append(wantList, want)
		}
	}

	if len(wantList) > 0 {
		return wantList
	}

	overwriteList := make([]string, 0, len(g.Overwrite))
	overwriteMap := make(map[string]bool)
	for _, overwrite := range g.Overwrite {
		if overwrite = strings.ToUpper(strings.TrimSpace(overwrite)); overwrite != "" && !excludeMap[overwrite] {
			overwriteList = append(overwriteList, overwrite)
			overwriteMap[overwrite] = true
		}
	}

	list := make([]string, 0, 300)
	for entry := range container.Loop() {
		name := entry.GetName()
		if excludeMap[name] || overwriteMap[name] {
			continue
		}
		list = append(list, name)
	}

	// Sort the lists
	slices.Sort(list)

	// Make sure the names in overwriteList are written at last
	list = append(list, overwriteList...)

	return list
}

func (g *GeoLite2CountryMMDBOut) marshalData(writer *mmdbwriter.Tree, entry *lib.Entry, extraInfo map[string]any) error {
	var entryCidr []string
	var err error
	switch g.OnlyIPType {
	case lib.IPv4:
		entryCidr, err = entry.MarshalText(lib.IgnoreIPv6)
	case lib.IPv6:
		entryCidr, err = entry.MarshalText(lib.IgnoreIPv4)
	default:
		entryCidr, err = entry.MarshalText()
	}
	if err != nil {
		return err
	}

	var record mmdbtype.DataType
	switch strings.TrimSpace(g.SourceMMDBURI) {
	case "": // No need to get extra info
		switch g.Type {
		case TypeGeoLite2CountryMMDBOut, TypeDBIPCountryMMDBOut:
			record = mmdbtype.Map{
				"country": mmdbtype.Map{
					"iso_code": mmdbtype.String(entry.GetName()),
				},
			}

		case TypeIPInfoCountryMMDBOut:
			record = mmdbtype.Map{
				"country_code": mmdbtype.String(entry.GetName()),
			}

		default:
			return lib.ErrNotSupportedFormat
		}

	default: // Get extra info
		switch g.Type {
		case TypeGeoLite2CountryMMDBOut:
			info, found := extraInfo[entry.GetName()].(geoip2.Country)
			if !found {
				log.Printf("⚠️ [type %s | action %s] not found extra info for list %s\n", g.Type, g.Action, entry.GetName())

				record = mmdbtype.Map{
					"country": mmdbtype.Map{
						"iso_code": mmdbtype.String(entry.GetName()),
					},
				}
			} else if info.Continent.Code != "" {
				record = mmdbtype.Map{
					"continent": mmdbtype.Map{
						"names": mmdbtype.Map{
							"de":    mmdbtype.String(info.Continent.Names.German),
							"en":    mmdbtype.String(info.Continent.Names.English),
							"es":    mmdbtype.String(info.Continent.Names.Spanish),
							"fr":    mmdbtype.String(info.Continent.Names.French),
							"ja":    mmdbtype.String(info.Continent.Names.Japanese),
							"pt-BR": mmdbtype.String(info.Continent.Names.BrazilianPortuguese),
							"ru":    mmdbtype.String(info.Continent.Names.Russian),
							"zh-CN": mmdbtype.String(info.Continent.Names.SimplifiedChinese),
						},
						"code":       mmdbtype.String(info.Continent.Code),
						"geoname_id": mmdbtype.Uint32(info.Continent.GeoNameID),
					},
					"country": mmdbtype.Map{
						"names": mmdbtype.Map{
							"de":    mmdbtype.String(info.Country.Names.German),
							"en":    mmdbtype.String(info.Country.Names.English),
							"es":    mmdbtype.String(info.Country.Names.Spanish),
							"fr":    mmdbtype.String(info.Country.Names.French),
							"ja":    mmdbtype.String(info.Country.Names.Japanese),
							"pt-BR": mmdbtype.String(info.Country.Names.BrazilianPortuguese),
							"ru":    mmdbtype.String(info.Country.Names.Russian),
							"zh-CN": mmdbtype.String(info.Country.Names.SimplifiedChinese),
						},
						"iso_code":             mmdbtype.String(entry.GetName()),
						"geoname_id":           mmdbtype.Uint32(info.Country.GeoNameID),
						"is_in_european_union": mmdbtype.Bool(info.Country.IsInEuropeanUnion),
					},
				}
			} else {
				record = mmdbtype.Map{
					"country": mmdbtype.Map{
						"names": mmdbtype.Map{
							"de":    mmdbtype.String(info.Country.Names.German),
							"en":    mmdbtype.String(info.Country.Names.English),
							"es":    mmdbtype.String(info.Country.Names.Spanish),
							"fr":    mmdbtype.String(info.Country.Names.French),
							"ja":    mmdbtype.String(info.Country.Names.Japanese),
							"pt-BR": mmdbtype.String(info.Country.Names.BrazilianPortuguese),
							"ru":    mmdbtype.String(info.Country.Names.Russian),
							"zh-CN": mmdbtype.String(info.Country.Names.SimplifiedChinese),
						},
						"iso_code":             mmdbtype.String(entry.GetName()),
						"geoname_id":           mmdbtype.Uint32(info.Country.GeoNameID),
						"is_in_european_union": mmdbtype.Bool(info.Country.IsInEuropeanUnion),
					},
				}
			}

		case TypeDBIPCountryMMDBOut:
			info, found := extraInfo[entry.GetName()].(dbipCountry)
			if !found {
				log.Printf("⚠️ [type %s | action %s] not found extra info for list %s\n", g.Type, g.Action, entry.GetName())

				record = mmdbtype.Map{
					"country": mmdbtype.Map{
						"iso_code": mmdbtype.String(entry.GetName()),
					},
				}
			} else if info.Continent.Code != "" {
				record = mmdbtype.Map{
					"continent": mmdbtype.Map{
						"names": mmdbtype.Map{
							"de":    mmdbtype.String(info.Continent.Names.German),
							"en":    mmdbtype.String(info.Continent.Names.English),
							"es":    mmdbtype.String(info.Continent.Names.Spanish),
							"fa":    mmdbtype.String(info.Continent.Names.Persian),
							"fr":    mmdbtype.String(info.Continent.Names.French),
							"ja":    mmdbtype.String(info.Continent.Names.Japanese),
							"ko":    mmdbtype.String(info.Continent.Names.Korean),
							"pt-BR": mmdbtype.String(info.Continent.Names.BrazilianPortuguese),
							"ru":    mmdbtype.String(info.Continent.Names.Russian),
							"zh-CN": mmdbtype.String(info.Continent.Names.SimplifiedChinese),
						},
						"code":       mmdbtype.String(info.Continent.Code),
						"geoname_id": mmdbtype.Uint32(info.Continent.GeoNameID),
					},
					"country": mmdbtype.Map{
						"names": mmdbtype.Map{
							"de":    mmdbtype.String(info.Country.Names.German),
							"en":    mmdbtype.String(info.Country.Names.English),
							"es":    mmdbtype.String(info.Country.Names.Spanish),
							"fa":    mmdbtype.String(info.Country.Names.Persian),
							"fr":    mmdbtype.String(info.Country.Names.French),
							"ja":    mmdbtype.String(info.Country.Names.Japanese),
							"ko":    mmdbtype.String(info.Country.Names.Korean),
							"pt-BR": mmdbtype.String(info.Country.Names.BrazilianPortuguese),
							"ru":    mmdbtype.String(info.Country.Names.Russian),
							"zh-CN": mmdbtype.String(info.Country.Names.SimplifiedChinese),
						},
						"iso_code":             mmdbtype.String(entry.GetName()),
						"geoname_id":           mmdbtype.Uint32(info.Country.GeoNameID),
						"is_in_european_union": mmdbtype.Bool(info.Country.IsInEuropeanUnion),
					},
				}
			} else {
				record = mmdbtype.Map{
					"country": mmdbtype.Map{
						"names": mmdbtype.Map{
							"de":    mmdbtype.String(info.Country.Names.German),
							"en":    mmdbtype.String(info.Country.Names.English),
							"es":    mmdbtype.String(info.Country.Names.Spanish),
							"fa":    mmdbtype.String(info.Country.Names.Persian),
							"fr":    mmdbtype.String(info.Country.Names.French),
							"ja":    mmdbtype.String(info.Country.Names.Japanese),
							"ko":    mmdbtype.String(info.Country.Names.Korean),
							"pt-BR": mmdbtype.String(info.Country.Names.BrazilianPortuguese),
							"ru":    mmdbtype.String(info.Country.Names.Russian),
							"zh-CN": mmdbtype.String(info.Country.Names.SimplifiedChinese),
						},
						"iso_code":             mmdbtype.String(entry.GetName()),
						"geoname_id":           mmdbtype.Uint32(info.Country.GeoNameID),
						"is_in_european_union": mmdbtype.Bool(info.Country.IsInEuropeanUnion),
					},
				}
			}

		case TypeIPInfoCountryMMDBOut:
			info, found := extraInfo[entry.GetName()].(ipInfoLite)
			if !found {
				log.Printf("⚠️ [type %s | action %s] not found extra info for list %s\n", g.Type, g.Action, entry.GetName())

				record = mmdbtype.Map{
					"country_code": mmdbtype.String(entry.GetName()),
				}
			} else {
				record = mmdbtype.Map{
					"as_domain":      mmdbtype.String(info.ASDomain),
					"as_name":        mmdbtype.String(info.ASName),
					"asn":            mmdbtype.String(info.ASN),
					"continent":      mmdbtype.String(info.Continent),
					"continent_code": mmdbtype.String(info.ContinentCode),
					"country":        mmdbtype.String(info.Country),
					"country_code":   mmdbtype.String(entry.GetName()),
				}
			}

		default:
			return lib.ErrNotSupportedFormat
		}
	}

	for _, cidr := range entryCidr {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return err
		}
		if err := writer.Insert(network, record); err != nil {
			return err
		}
	}

	return nil
}

func (g *GeoLite2CountryMMDBOut) writeFile(filename string, writer *mmdbwriter.Tree) error {
	if err := os.MkdirAll(g.OutputDir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(g.OutputDir, filename), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	_, err = writer.WriteTo(f)
	if err != nil {
		return err
	}

	log.Printf("✅ [%s] %s --> %s", g.Type, filename, g.OutputDir)

	return nil
}
