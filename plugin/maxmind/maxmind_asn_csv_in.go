package maxmind

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	TypeASNCSV = "maxmindGeoLite2ASNCSV"
	DescASNCSV = "Convert MaxMind GeoLite2 ASN CSV data to other formats"
)

var (
	defaultASNIPv4File = filepath.Join("./", "geolite2", "GeoLite2-ASN-Blocks-IPv4.csv")
	defaultASNIPv6File = filepath.Join("./", "geolite2", "GeoLite2-ASN-Blocks-IPv6.csv")
)

func init() {
	lib.RegisterInputConfigCreator(TypeASNCSV, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newGeoLite2ASNCSV(action, data)
	})
	lib.RegisterInputConverter(TypeASNCSV, &GeoLite2ASNCSV{
		Description: DescASNCSV,
	})
}

func newGeoLite2ASNCSV(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
	var tmp struct {
		IPv4File   string              `json:"ipv4"`
		IPv6File   string              `json:"ipv6"`
		Want       map[string][]string `json:"wantedList"`
		OnlyIPType lib.IPType          `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	if tmp.IPv4File == "" {
		tmp.IPv4File = defaultASNIPv4File
	}

	if tmp.IPv6File == "" {
		tmp.IPv6File = defaultASNIPv6File
	}

	// Filter want list
	wantList := make(map[string][]string) // map[asn][]listname
	for list, asnList := range tmp.Want {
		list = strings.ToUpper(strings.TrimSpace(list))
		if list == "" {
			continue
		}

		for _, asn := range asnList {
			asn = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(asn)), "as")
			if asn == "" {
				continue
			}

			if listArr, found := wantList[asn]; found {
				listArr = append(listArr, list)
				wantList[asn] = listArr
			} else {
				wantList[asn] = []string{list}
			}
		}
	}

	if len(wantList) == 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] wantedList must be specified in config", TypeASNCSV, action)
	}

	return &GeoLite2ASNCSV{
		Type:        TypeASNCSV,
		Action:      action,
		Description: DescASNCSV,
		IPv4File:    tmp.IPv4File,
		IPv6File:    tmp.IPv6File,
		Want:        wantList,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

type GeoLite2ASNCSV struct {
	Type        string
	Action      lib.Action
	Description string
	IPv4File    string
	IPv6File    string
	Want        map[string][]string
	OnlyIPType  lib.IPType
}

func (g *GeoLite2ASNCSV) GetType() string {
	return g.Type
}

func (g *GeoLite2ASNCSV) GetAction() lib.Action {
	return g.Action
}

func (g *GeoLite2ASNCSV) GetDescription() string {
	return g.Description
}

func (g *GeoLite2ASNCSV) Input(container lib.Container) (lib.Container, error) {
	entries := make(map[string]*lib.Entry)

	if g.IPv4File != "" {
		if err := g.process(g.IPv4File, entries); err != nil {
			return nil, err
		}
	}

	if g.IPv6File != "" {
		if err := g.process(g.IPv6File, entries); err != nil {
			return nil, err
		}
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

func (g *GeoLite2ASNCSV) process(file string, entries map[string]*lib.Entry) error {
	if entries == nil {
		entries = make(map[string]*lib.Entry)
	}

	var f io.ReadCloser
	var err error
	switch {
	case strings.HasPrefix(strings.ToLower(file), "http://"), strings.HasPrefix(strings.ToLower(file), "https://"):
		f, err = lib.GetRemoteURLReader(file)
	default:
		f, err = os.Open(file)
	}

	if err != nil {
		return err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.Read() // skip header

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if len(record) < 2 {
			return fmt.Errorf("❌ [type %s | action %s] invalid record: %v", g.Type, g.Action, record)
		}

		if listArr, found := g.Want[strings.TrimSpace(record[1])]; found {
			for _, listName := range listArr {
				entry, got := entries[listName]
				if !got {
					entry = lib.NewEntry(listName)
				}
				if err := entry.AddPrefix(strings.TrimSpace(record[0])); err != nil {
					return err
				}
				entries[listName] = entry
			}
		}
	}

	return nil
}
