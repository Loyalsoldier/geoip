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
	TypeMaxmindMMDBIn = "maxmindMMDB"
	DescMaxmindMMDBIn = "Convert MaxMind mmdb database to other formats"
)

var (
	defaultMMDBFile = filepath.Join("./", "geolite2", "GeoLite2-Country.mmdb")
)

func init() {
	lib.RegisterInputConfigCreator(TypeMaxmindMMDBIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newMaxmindMMDBIn(action, data)
	})
	lib.RegisterInputConverter(TypeMaxmindMMDBIn, &MaxmindMMDBIn{
		Description: DescMaxmindMMDBIn,
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

	// Filter want list
	wantList := make(map[string]bool)
	for _, want := range tmp.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList[want] = true
		}
	}

	return &MaxmindMMDBIn{
		Type:        TypeMaxmindMMDBIn,
		Action:      action,
		Description: DescMaxmindMMDBIn,
		URI:         tmp.URI,
		Want:        wantList,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

type MaxmindMMDBIn struct {
	Type        string
	Action      lib.Action
	Description string
	URI         string
	Want        map[string]bool
	OnlyIPType  lib.IPType
}

func (m *MaxmindMMDBIn) GetType() string {
	return m.Type
}

func (m *MaxmindMMDBIn) GetAction() lib.Action {
	return m.Action
}

func (m *MaxmindMMDBIn) GetDescription() string {
	return m.Description
}

func (m *MaxmindMMDBIn) Input(container lib.Container) (lib.Container, error) {
	var content []byte
	var err error
	switch {
	case strings.HasPrefix(strings.ToLower(m.URI), "http://"), strings.HasPrefix(strings.ToLower(m.URI), "https://"):
		content, err = lib.GetRemoteURLContent(m.URI)
	default:
		content, err = os.ReadFile(m.URI)
	}
	if err != nil {
		return nil, err
	}

	entries := make(map[string]*lib.Entry, 300)
	err = m.generateEntries(content, entries)
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] no entry is generated", m.Type, m.Action)
	}

	var ignoreIPType lib.IgnoreIPOption
	switch m.OnlyIPType {
	case lib.IPv4:
		ignoreIPType = lib.IgnoreIPv6
	case lib.IPv6:
		ignoreIPType = lib.IgnoreIPv4
	}

	for _, entry := range entries {
		switch m.Action {
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

func (m *MaxmindMMDBIn) generateEntries(content []byte, entries map[string]*lib.Entry) error {
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

		if len(m.Want) > 0 && !m.Want[name] {
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
