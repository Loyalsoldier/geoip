package maxmind

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	TypeGeoLite2ASNCSVIn = "maxmindGeoLite2ASNCSV"
	DescGeoLite2ASNCSVIn = "Convert MaxMind GeoLite2 ASN CSV data to other formats"
)

var (
	defaultGeoLite2ASNCSVIPv4File = filepath.Join("./", "geolite2", "GeoLite2-ASN-Blocks-IPv4.csv")
	defaultGeoLite2ASNCSVIPv6File = filepath.Join("./", "geolite2", "GeoLite2-ASN-Blocks-IPv6.csv")
)

func init() {
	lib.RegisterInputConfigCreator(TypeGeoLite2ASNCSVIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return NewGeoLite2ASNCSVInFromBytes(action, data)
	})
	lib.RegisterInputConverter(TypeGeoLite2ASNCSVIn, &geoLite2ASNCSVIn{
		Description: DescGeoLite2ASNCSVIn,
	})
}

func NewGeoLite2ASNCSVIn(action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	g := &geoLite2ASNCSVIn{
		Type:        TypeGeoLite2ASNCSVIn,
		Action:      action,
		Description: DescGeoLite2ASNCSVIn,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(g)
		}
	}

	validateInputOptions(g.Type, g.Action, g.OnlyIPType)
	// Default both IP files only when neither has been specified.
	if g.IPv4File == "" && g.IPv6File == "" {
		g.IPv4File = defaultGeoLite2ASNCSVIPv4File
		g.IPv6File = defaultGeoLite2ASNCSVIPv6File
	}

	wanted, ok := asnWantedList(g.wantedList)
	if !ok {
		log.Fatalf("❌ [type %s | action %s] invalid wantedList: must be an array of ASN strings or an object of ASN string arrays", g.Type, g.Action)
	}
	g.Want = make(map[string][]string) // map[asn][]listname or map[asn][]asn
	g.wantedList = nil

	for list, asnList := range wanted.TypeMap {
		list = strings.ToUpper(strings.TrimSpace(list))
		if list == "" {
			continue
		}

		for _, asn := range asnList {
			asn = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(asn)), "as")
			if asn == "" {
				continue
			}

			if listArr, found := g.Want[asn]; found {
				listArr = append(listArr, list)
				g.Want[asn] = listArr
			} else {
				g.Want[asn] = []string{list}
			}
		}
	}

	for _, asn := range wanted.TypeSlice {
		asn = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(asn)), "as")
		if asn == "" {
			continue
		}

		g.Want[asn] = []string{"AS" + asn}
	}

	return g
}

func asnWantedList(value any) (lib.WantedListExtended, bool) {
	switch value := value.(type) {
	case lib.WantedListExtended:
		return value, true
	case map[string][]string:
		return lib.WantedListExtended{TypeMap: value}, true
	case map[string]any:
		lists := make(map[string][]string, len(value))
		for name, raw := range value {
			list, ok := stringList(raw)
			if !ok {
				return lib.WantedListExtended{}, false
			}
			lists[name] = list
		}
		return lib.WantedListExtended{TypeMap: lists}, true
	default:
		list, ok := stringList(value)
		return lib.WantedListExtended{TypeSlice: list}, ok
	}
}

func NewGeoLite2ASNCSVInFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
	var tmp struct {
		IPv4File   string     `json:"ipv4"`
		IPv6File   string     `json:"ipv6"`
		Want       any        `json:"wantedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}
	return NewGeoLite2ASNCSVIn(
		action,
		WithIPv4File(tmp.IPv4File),
		WithIPv6File(tmp.IPv6File),
		WithInputWantedList(tmp.Want),
		WithInputOnlyIPType(tmp.OnlyIPType),
	), nil
}

type geoLite2ASNCSVIn struct {
	Type        string
	Action      lib.Action
	Description string
	IPv4File    string
	IPv6File    string
	Want        map[string][]string
	OnlyIPType  lib.IPType
	wantedList  any
}

func (g *geoLite2ASNCSVIn) GetType() string {
	return g.Type
}

func (g *geoLite2ASNCSVIn) GetAction() lib.Action {
	return g.Action
}

func (g *geoLite2ASNCSVIn) GetDescription() string {
	return g.Description
}

func (g *geoLite2ASNCSVIn) Input(container lib.Container) (lib.Container, error) {
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

	ignoreIPType := lib.GetIgnoreIPType(g.OnlyIPType)

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

func (g *geoLite2ASNCSVIn) process(file string, entries map[string]*lib.Entry) error {
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

		// Maxmind ASN CSV reference:
		// network,autonomous_system_number,autonomous_system_organization
		// 1.0.0.0/24,13335,CLOUDFLARENET
		// 1.0.4.0/22,38803,"Gtelecom Pty Ltd"
		// 1.0.16.0/24,2519,"ARTERIA Networks Corporation"

		switch len(g.Want) {
		case 0: // it means user wants all ASNs
			asn := "AS" + strings.TrimSpace(record[1]) // default list name is in "AS12345" format
			entry, got := entries[asn]
			if !got {
				entry = lib.NewEntry(asn)
			}
			if err := entry.AddPrefix(strings.TrimSpace(record[0])); err != nil {
				return err
			}
			entries[asn] = entry

		default: // it means user wants specific ASNs or customized lists with specific ASNs
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
	}

	return nil
}
