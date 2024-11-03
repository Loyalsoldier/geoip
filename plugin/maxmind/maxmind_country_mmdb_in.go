package maxmind

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/oschwald/geoip2-golang"
	"github.com/oschwald/maxminddb-golang"
)

const (
	TypeMaxmindMMDBIn = "maxmindMMDB"
	DescMaxmindMMDBIn = "Convert MaxMind mmdb database to other formats"
)

func init() {
	lib.RegisterInputConfigCreator(TypeMaxmindMMDBIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newMMDBIn(TypeMaxmindMMDBIn, DescMaxmindMMDBIn, action, data)
	})
	lib.RegisterInputConverter(TypeMaxmindMMDBIn, &MMDBIn{
		Description: DescMaxmindMMDBIn,
	})
}

type MMDBIn struct {
	Type        string
	Action      lib.Action
	Description string
	URI         string
	Want        map[string]bool
	OnlyIPType  lib.IPType
}

func (m *MMDBIn) GetType() string {
	return m.Type
}

func (m *MMDBIn) GetAction() lib.Action {
	return m.Action
}

func (m *MMDBIn) GetDescription() string {
	return m.Description
}

func (m *MMDBIn) Input(container lib.Container) (lib.Container, error) {
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

func (m *MMDBIn) generateEntries(content []byte, entries map[string]*lib.Entry) error {
	db, err := maxminddb.FromBytes(content)
	if err != nil {
		return err
	}
	defer db.Close()

	networks := db.Networks(maxminddb.SkipAliasedNetworks)
	for networks.Next() {
		var name string
		var subnet *net.IPNet
		var err error

		switch m.Type {
		case TypeMaxmindMMDBIn, TypeDBIPCountryMMDBIn:
			var record geoip2.Country
			subnet, err = networks.Network(&record)
			if err != nil {
				return err
			}

			switch {
			case strings.TrimSpace(record.Country.IsoCode) != "":
				name = strings.ToUpper(strings.TrimSpace(record.Country.IsoCode))
			case strings.TrimSpace(record.RegisteredCountry.IsoCode) != "":
				name = strings.ToUpper(strings.TrimSpace(record.RegisteredCountry.IsoCode))
			case strings.TrimSpace(record.RepresentedCountry.IsoCode) != "":
				name = strings.ToUpper(strings.TrimSpace(record.RepresentedCountry.IsoCode))
			}

		case TypeIPInfoCountryMMDBIn:
			record := struct {
				Country string `maxminddb:"country"`
			}{}
			subnet, err = networks.Network(&record)
			if err != nil {
				return err
			}
			name = strings.ToUpper(strings.TrimSpace(record.Country))

		default:
			return lib.ErrNotSupportedFormat
		}

		if name == "" || subnet == nil {
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
