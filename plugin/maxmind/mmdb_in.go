package maxmind

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/oschwald/maxminddb-golang"
)

const (
	typeMaxmindMMDBIn = "maxmindMMDB"
	descMaxmindMMDBIn = "Convert MaxMind mmdb database to other formats"
)

var (
	defaultMMDBFile = filepath.Join("./", "geolite2", "GeoLite2-Country.mmdb")
)

func init() {
	lib.RegisterInputConfigCreator(typeMaxmindMMDBIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newMaxmindMMDBIn(action, data)
	})
	lib.RegisterInputConverter(typeMaxmindMMDBIn, &maxmindMMDBIn{
		Description: descMaxmindMMDBIn,
	})
}

func newMaxmindMMDBIn(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
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

	if tmp.URI == "" {
		tmp.URI = defaultMMDBFile
	}

	return &maxmindMMDBIn{
		Type:        typeMaxmindMMDBIn,
		Action:      action,
		Description: descMaxmindMMDBIn,
		URI:         tmp.URI,
		Want:        tmp.Want,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

type maxmindMMDBIn struct {
	Type        string
	Action      lib.Action
	Description string
	URI         string
	Want        []string
	OnlyIPType  lib.IPType
}

func (g *maxmindMMDBIn) GetType() string {
	return g.Type
}

func (g *maxmindMMDBIn) GetAction() lib.Action {
	return g.Action
}

func (g *maxmindMMDBIn) GetDescription() string {
	return g.Description
}

func (g *maxmindMMDBIn) Input(container lib.Container) (lib.Container, error) {
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
		return nil, fmt.Errorf("❌ [type %s | action %s] no entry is generated", typeMaxmindMMDBIn, g.Action)
	}

	var ignoreIPType lib.IgnoreIPOption
	switch g.OnlyIPType {
	case lib.IPv4:
		ignoreIPType = lib.IgnoreIPv6
	case lib.IPv6:
		ignoreIPType = lib.IgnoreIPv4
	}

	// Filter want list
	wantList := make(map[string]bool)
	for _, want := range g.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList[want] = true
		}
	}

	for _, entry := range entries {
		name := entry.GetName()
		if len(wantList) > 0 && !wantList[name] {
			continue
		}

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

func (g *maxmindMMDBIn) generateEntries(content []byte, entries map[string]*lib.Entry) error {
	db, err := maxminddb.FromBytes(content)
	if err != nil {
		return err
	}
	defer db.Close()

	networks := db.Networks(maxminddb.SkipAliasedNetworks)
	for networks.Next() {
		record := struct {
			Country struct {
				IsoCode string `maxminddb:"iso_code"`
			} `maxminddb:"country"`
			RegisteredCountry struct {
				IsoCode string `maxminddb:"iso_code"`
			} `maxminddb:"registered_country"`
			RepresentedCountry struct {
				IsoCode string `maxminddb:"iso_code"`
			} `maxminddb:"represented_country"`
		}{}

		subnet, err := networks.Network(&record)
		if err != nil {
			continue
		}

		name := ""
		switch {
		case strings.TrimSpace(record.Country.IsoCode) != "":
			name = strings.ToUpper(strings.TrimSpace(record.Country.IsoCode))
		case strings.TrimSpace(record.RegisteredCountry.IsoCode) != "":
			name = strings.ToUpper(strings.TrimSpace(record.RegisteredCountry.IsoCode))
		case strings.TrimSpace(record.RepresentedCountry.IsoCode) != "":
			name = strings.ToUpper(strings.TrimSpace(record.RepresentedCountry.IsoCode))
		default:
			continue
		}

		entry, found := entries[name]
		if !found {
			entry = lib.NewEntry(name)
		}

		if err := entry.AddPrefix(subnet); err != nil {
			return err
		}

		entries[name] = entry
	}

	if networks.Err() != nil {
		return networks.Err()
	}

	return nil
}
