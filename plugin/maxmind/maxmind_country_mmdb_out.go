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
	"github.com/oschwald/geoip2-golang"
)

const (
	TypeMaxmindMMDBOut = "maxmindMMDB"
	DescMaxmindMMDBOut = "Convert data to MaxMind mmdb database format"
)

func init() {
	lib.RegisterOutputConfigCreator(TypeMaxmindMMDBOut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return newMMDBOut(TypeMaxmindMMDBOut, DescMaxmindMMDBOut, action, data)
	})
	lib.RegisterOutputConverter(TypeMaxmindMMDBOut, &MMDBOut{
		Description: DescMaxmindMMDBOut,
	})
}

type MMDBOut struct {
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

func (m *MMDBOut) GetType() string {
	return m.Type
}

func (m *MMDBOut) GetAction() lib.Action {
	return m.Action
}

func (m *MMDBOut) GetDescription() string {
	return m.Description
}

func (m *MMDBOut) Output(container lib.Container) error {
	dbName := ""
	dbDesc := ""
	recordSize := 28

	switch m.Type {
	case TypeMaxmindMMDBOut:
		dbName = "GeoLite2-Country"
		dbDesc = "Customized GeoLite2 Country database"

	case TypeDBIPCountryMMDBOut:
		dbName = "DBIP-Country-Lite"
		dbDesc = "Customized DB-IP Country Lite database"

	case TypeIPInfoCountryMMDBOut:
		dbName = "IPInfo-Country"
		dbDesc = "Customized IPInfo Country database"
		recordSize = 32
	}

	writer, err := mmdbwriter.New(
		mmdbwriter.Options{
			DatabaseType:            dbName,
			Description:             map[string]string{"en": dbDesc},
			RecordSize:              recordSize,
			IncludeReservedNetworks: true,
		},
	)
	if err != nil {
		return err
	}

	// Get extra info
	extraInfo, err := m.GetExtraInfo()
	if err != nil {
		return err
	}

	updated := false
	for _, name := range m.filterAndSortList(container) {
		entry, found := container.GetEntry(name)
		if !found {
			log.Printf("❌ entry %s not found\n", name)
			continue
		}

		if err := m.marshalData(writer, entry, extraInfo); err != nil {
			return err
		}

		updated = true
	}

	if updated {
		return m.writeFile(m.OutputName, writer)
	}

	return nil
}

func (m *MMDBOut) filterAndSortList(container lib.Container) []string {
	/*
		Note: The IPs and/or CIDRs of the latter list will overwrite those of the former one
		when duplicated data found due to MaxMind mmdb file format constraint.

		Be sure to place the name of the most important list at last
		when writing wantedList and overwriteList in config file.

		The order of names in wantedList has a higher priority than which of the overwriteList.
	*/

	excludeMap := make(map[string]bool)
	for _, exclude := range m.Exclude {
		if exclude = strings.ToUpper(strings.TrimSpace(exclude)); exclude != "" {
			excludeMap[exclude] = true
		}
	}

	wantList := make([]string, 0, len(m.Want))
	for _, want := range m.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" && !excludeMap[want] {
			wantList = append(wantList, want)
		}
	}

	if len(wantList) > 0 {
		return wantList
	}

	overwriteList := make([]string, 0, len(m.Overwrite))
	overwriteMap := make(map[string]bool)
	for _, overwrite := range m.Overwrite {
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

func (m *MMDBOut) marshalData(writer *mmdbwriter.Tree, entry *lib.Entry, extraInfo map[string]interface{}) error {
	var entryCidr []string
	var err error
	switch m.OnlyIPType {
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
	switch strings.TrimSpace(m.SourceMMDBURI) {
	case "": // No need to get extra info
		switch m.Type {
		case TypeMaxmindMMDBOut, TypeDBIPCountryMMDBOut:
			record = mmdbtype.Map{
				"country": mmdbtype.Map{
					"iso_code": mmdbtype.String(entry.GetName()),
				},
			}

		case TypeIPInfoCountryMMDBOut:
			record = mmdbtype.Map{
				"country": mmdbtype.String(entry.GetName()),
			}

		default:
			return lib.ErrNotSupportedFormat
		}

	default: // Get extra info
		switch m.Type {
		case TypeMaxmindMMDBOut:
			info, found := extraInfo[entry.GetName()].(geoip2.Country)
			if !found {
				log.Printf("⚠️ [type %s | action %s] not found extra info for list %s\n", m.Type, m.Action, entry.GetName())

				record = mmdbtype.Map{
					"country": mmdbtype.Map{
						"iso_code": mmdbtype.String(entry.GetName()),
					},
				}
			} else if info.Continent.Code != "" {
				record = mmdbtype.Map{
					"continent": mmdbtype.Map{
						"names": mmdbtype.Map{
							"de":    mmdbtype.String(info.Continent.Names["de"]),
							"en":    mmdbtype.String(info.Continent.Names["en"]),
							"es":    mmdbtype.String(info.Continent.Names["es"]),
							"fr":    mmdbtype.String(info.Continent.Names["fr"]),
							"ja":    mmdbtype.String(info.Continent.Names["ja"]),
							"pt-BR": mmdbtype.String(info.Continent.Names["pt-BR"]),
							"ru":    mmdbtype.String(info.Continent.Names["ru"]),
							"zh-CN": mmdbtype.String(info.Continent.Names["zh-CN"]),
						},
						"code":       mmdbtype.String(info.Continent.Code),
						"geoname_id": mmdbtype.Uint32(info.Continent.GeoNameID),
					},
					"country": mmdbtype.Map{
						"names": mmdbtype.Map{
							"de":    mmdbtype.String(info.Country.Names["de"]),
							"en":    mmdbtype.String(info.Country.Names["en"]),
							"es":    mmdbtype.String(info.Country.Names["es"]),
							"fr":    mmdbtype.String(info.Country.Names["fr"]),
							"ja":    mmdbtype.String(info.Country.Names["ja"]),
							"pt-BR": mmdbtype.String(info.Country.Names["pt-BR"]),
							"ru":    mmdbtype.String(info.Country.Names["ru"]),
							"zh-CN": mmdbtype.String(info.Country.Names["zh-CN"]),
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
							"de":    mmdbtype.String(info.Country.Names["de"]),
							"en":    mmdbtype.String(info.Country.Names["en"]),
							"es":    mmdbtype.String(info.Country.Names["es"]),
							"fr":    mmdbtype.String(info.Country.Names["fr"]),
							"ja":    mmdbtype.String(info.Country.Names["ja"]),
							"pt-BR": mmdbtype.String(info.Country.Names["pt-BR"]),
							"ru":    mmdbtype.String(info.Country.Names["ru"]),
							"zh-CN": mmdbtype.String(info.Country.Names["zh-CN"]),
						},
						"iso_code":             mmdbtype.String(entry.GetName()),
						"geoname_id":           mmdbtype.Uint32(info.Country.GeoNameID),
						"is_in_european_union": mmdbtype.Bool(info.Country.IsInEuropeanUnion),
					},
				}
			}

		case TypeDBIPCountryMMDBOut:
			info, found := extraInfo[entry.GetName()].(geoip2.Country)
			if !found {
				log.Printf("⚠️ [type %s | action %s] not found extra info for list %s\n", m.Type, m.Action, entry.GetName())

				record = mmdbtype.Map{
					"country": mmdbtype.Map{
						"iso_code": mmdbtype.String(entry.GetName()),
					},
				}
			} else if info.Continent.Code != "" {
				record = mmdbtype.Map{
					"continent": mmdbtype.Map{
						"names": mmdbtype.Map{
							"de":    mmdbtype.String(info.Continent.Names["de"]),
							"en":    mmdbtype.String(info.Continent.Names["en"]),
							"es":    mmdbtype.String(info.Continent.Names["es"]),
							"fa":    mmdbtype.String(info.Continent.Names["fa"]),
							"fr":    mmdbtype.String(info.Continent.Names["fr"]),
							"ja":    mmdbtype.String(info.Continent.Names["ja"]),
							"ko":    mmdbtype.String(info.Continent.Names["ko"]),
							"pt-BR": mmdbtype.String(info.Continent.Names["pt-BR"]),
							"ru":    mmdbtype.String(info.Continent.Names["ru"]),
							"zh-CN": mmdbtype.String(info.Continent.Names["zh-CN"]),
						},
						"code":       mmdbtype.String(info.Continent.Code),
						"geoname_id": mmdbtype.Uint32(info.Continent.GeoNameID),
					},
					"country": mmdbtype.Map{
						"names": mmdbtype.Map{
							"de":    mmdbtype.String(info.Country.Names["de"]),
							"en":    mmdbtype.String(info.Country.Names["en"]),
							"es":    mmdbtype.String(info.Country.Names["es"]),
							"fa":    mmdbtype.String(info.Country.Names["fa"]),
							"fr":    mmdbtype.String(info.Country.Names["fr"]),
							"ja":    mmdbtype.String(info.Country.Names["ja"]),
							"ko":    mmdbtype.String(info.Country.Names["ko"]),
							"pt-BR": mmdbtype.String(info.Country.Names["pt-BR"]),
							"ru":    mmdbtype.String(info.Country.Names["ru"]),
							"zh-CN": mmdbtype.String(info.Country.Names["zh-CN"]),
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
							"de":    mmdbtype.String(info.Country.Names["de"]),
							"en":    mmdbtype.String(info.Country.Names["en"]),
							"es":    mmdbtype.String(info.Country.Names["es"]),
							"fa":    mmdbtype.String(info.Country.Names["fa"]),
							"fr":    mmdbtype.String(info.Country.Names["fr"]),
							"ja":    mmdbtype.String(info.Country.Names["ja"]),
							"ko":    mmdbtype.String(info.Country.Names["ko"]),
							"pt-BR": mmdbtype.String(info.Country.Names["pt-BR"]),
							"ru":    mmdbtype.String(info.Country.Names["ru"]),
							"zh-CN": mmdbtype.String(info.Country.Names["zh-CN"]),
						},
						"iso_code":             mmdbtype.String(entry.GetName()),
						"geoname_id":           mmdbtype.Uint32(info.Country.GeoNameID),
						"is_in_european_union": mmdbtype.Bool(info.Country.IsInEuropeanUnion),
					},
				}
			}

		case TypeIPInfoCountryMMDBOut:
			info, found := extraInfo[entry.GetName()].(struct {
				Continent     string `maxminddb:"continent"`
				ContinentName string `maxminddb:"continent_name"`
				Country       string `maxminddb:"country"`
				CountryName   string `maxminddb:"country_name"`
			})

			if !found {
				log.Printf("⚠️ [type %s | action %s] not found extra info for list %s\n", m.Type, m.Action, entry.GetName())

				record = mmdbtype.Map{
					"country": mmdbtype.String(entry.GetName()),
				}
			} else {
				record = mmdbtype.Map{
					"continent":      mmdbtype.String(info.Continent),
					"continent_name": mmdbtype.String(info.ContinentName),
					"country":        mmdbtype.String(entry.GetName()),
					"country_name":   mmdbtype.String(info.CountryName),
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

func (m *MMDBOut) writeFile(filename string, writer *mmdbwriter.Tree) error {
	if err := os.MkdirAll(m.OutputDir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(m.OutputDir, filename), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	_, err = writer.WriteTo(f)
	if err != nil {
		return err
	}

	log.Printf("✅ [%s] %s --> %s", m.Type, filename, m.OutputDir)

	return nil
}
