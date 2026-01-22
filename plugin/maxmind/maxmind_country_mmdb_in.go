package maxmind

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/oschwald/geoip2-golang/v2"
	"github.com/oschwald/maxminddb-golang/v2"
)

const (
	TypeGeoLite2CountryMMDBIn = "maxmindMMDB"
	DescGeoLite2CountryMMDBIn = "Convert MaxMind mmdb database to other formats"
)

func init() {
	lib.RegisterInputConfigCreator(TypeGeoLite2CountryMMDBIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newGeoLite2CountryMMDBIn(TypeGeoLite2CountryMMDBIn, DescGeoLite2CountryMMDBIn, action, data)
	})
	lib.RegisterInputConverter(TypeGeoLite2CountryMMDBIn, &GeoLite2CountryMMDBIn{
		Description: DescGeoLite2CountryMMDBIn,
	})
}

type GeoLite2CountryMMDBIn struct {
	Type        string
	Action      lib.Action
	Description string
	URI         string
	Want        map[string]bool
	OnlyIPType  lib.IPType
}

func (g *GeoLite2CountryMMDBIn) GetType() string {
	return g.Type
}

func (g *GeoLite2CountryMMDBIn) GetAction() lib.Action {
	return g.Action
}

func (g *GeoLite2CountryMMDBIn) GetDescription() string {
	return g.Description
}

func (g *GeoLite2CountryMMDBIn) Input(container lib.Container) (lib.Container, error) {
	var content []byte
	var err error
	switch {
	case strings.HasPrefix(strings.ToLower(g.URI), "http://"), strings.HasPrefix(strings.ToLower(g.URI), "https://"):
		content, err = lib.GetRemoteURLContent(g.URI)
	default:
		content, err = os.ReadFile(g.URI)
	}
	if err != nil {
		return nil, err
	}

	entries := make(map[string]*lib.Entry, 300)
	err = g.generateEntries(content, entries)
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] no entry is generated", g.Type, g.Action)
	}

	var ignoreIPType lib.IgnoreIPOption
	switch g.OnlyIPType {
	case lib.IPv4:
		ignoreIPType = lib.IgnoreIPv6
	case lib.IPv6:
		ignoreIPType = lib.IgnoreIPv4
	}

	for _, entry := range entries {
		switch g.Action {
		case lib.ActionAdd:
			if err := container.Add(entry, ignoreIPType); err != nil {
				return nil, err
			}
		case lib.ActionRemove:
			if err := container.Remove(entry, lib.CaseRemovePrefix, ignoreIPType); err != nil {
				return nil, err
			}
		default:
			return nil, lib.ErrUnknownAction
		}
	}

	return container, nil
}

func (g *GeoLite2CountryMMDBIn) generateEntries(content []byte, entries map[string]*lib.Entry) error {
	db, err := maxminddb.OpenBytes(content)
	if err != nil {
		return err
	}
	defer db.Close()

	for network := range db.Networks() {
		var name string
		var err error

		switch g.Type {
		case TypeGeoLite2CountryMMDBIn, TypeDBIPCountryMMDBIn:
			var record geoip2.Country
			err = network.Decode(&record)
			if err != nil {
				return err
			}

			switch {
			case strings.TrimSpace(record.Country.ISOCode) != "":
				name = strings.ToUpper(strings.TrimSpace(record.Country.ISOCode))
			case strings.TrimSpace(record.RegisteredCountry.ISOCode) != "":
				name = strings.ToUpper(strings.TrimSpace(record.RegisteredCountry.ISOCode))
			case strings.TrimSpace(record.RepresentedCountry.ISOCode) != "":
				name = strings.ToUpper(strings.TrimSpace(record.RepresentedCountry.ISOCode))
			}

		case TypeIPInfoCountryMMDBIn:
			var record ipInfoLite
			err = network.Decode(&record)
			if err != nil {
				return err
			}
			name = strings.ToUpper(strings.TrimSpace(record.CountryCode))

		default:
			return lib.ErrNotSupportedFormat
		}

		if name == "" || !network.Found() {
			continue
		}

		if len(g.Want) > 0 && !g.Want[name] {
			continue
		}

		entry, found := entries[name]
		if !found {
			entry = lib.NewEntry(name)
		}

		if err := entry.AddPrefix(network.Prefix()); err != nil {
			return err
		}

		entries[name] = entry
	}

	return nil
}
