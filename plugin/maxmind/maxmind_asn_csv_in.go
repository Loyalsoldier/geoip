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

type geoLite2ASNCSVIn struct {
	Type        string
	Action      lib.Action
	Description string
	IPv4File    string
	IPv6File    string
	Want        map[string][]string
	OnlyIPType  lib.IPType
}

func NewGeoLite2ASNCSVIn(action lib.Action, opts ...lib.InputOption) (lib.InputConverter, error) {
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

	// When both of IP files are not specified,
	// it means user wants to use the default ones
	if g.IPv4File == "" && g.IPv6File == "" {
		g.IPv4File = defaultGeoLite2ASNCSVIPv4File
		g.IPv6File = defaultGeoLite2ASNCSVIPv6File
	}

	return g, nil
}

func WithASNCSVIPv4File(file string) lib.InputOption {
	return func(g lib.InputConverter) {
		g.(*geoLite2ASNCSVIn).IPv4File = strings.TrimSpace(file)
	}
}

func WithASNCSVIPv6File(file string) lib.InputOption {
	return func(g lib.InputConverter) {
		g.(*geoLite2ASNCSVIn).IPv6File = strings.TrimSpace(file)
	}
}

func WithASNCSVWantedList(want lib.WantedListExtended) lib.InputOption {
	return func(g lib.InputConverter) {
		wantList := make(map[string][]string) // map[asn][]listname or map[asn][]asn

		for list, asnList := range want.TypeMap {
			list = strings.ToUpper(strings.TrimSpace(list))
			if list == "" {
				continue
			}

			for _, asn := range asnList {
				asn = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(asn)), "as")
				if asn == "" {
					continue
				}

				wantList[asn] = append(wantList[asn], list)
			}
		}

		for _, asn := range want.TypeSlice {
			asn = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(asn)), "as")
			if asn == "" {
				continue
			}

			wantList[asn] = []string{"AS" + asn}
		}

		g.(*geoLite2ASNCSVIn).Want = wantList
	}
}

func WithASNCSVOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(g lib.InputConverter) {
		g.(*geoLite2ASNCSVIn).OnlyIPType = onlyIPType
	}
}

func NewGeoLite2ASNCSVInFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
	var tmp struct {
		IPv4File   string                 `json:"ipv4"`
		IPv6File   string                 `json:"ipv6"`
		Want       lib.WantedListExtended `json:"wantedList"`
		OnlyIPType lib.IPType             `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, fmt.Errorf("❌ [type %s | action %s] %w", TypeGeoLite2ASNCSVIn, action, err)
		}
	}

	return NewGeoLite2ASNCSVIn(
		action,
		WithASNCSVIPv4File(tmp.IPv4File),
		WithASNCSVIPv6File(tmp.IPv6File),
		WithASNCSVWantedList(tmp.Want),
		WithASNCSVOnlyIPType(tmp.OnlyIPType),
	)
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
			return nil, fmt.Errorf("❌ [type %s | action %s] %w", g.Type, g.Action, err)
		}
	}

	if g.IPv6File != "" {
		if err := g.process(g.IPv6File, entries); err != nil {
			return nil, fmt.Errorf("❌ [type %s | action %s] %w", g.Type, g.Action, err)
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
				return nil, fmt.Errorf("❌ [type %s | action %s] %w", g.Type, g.Action, err)
			}
		case lib.ActionRemove:
			if err := container.Remove(entry, lib.CaseRemovePrefix, ignoreIPType); err != nil {
				return nil, fmt.Errorf("❌ [type %s | action %s] %w", g.Type, g.Action, err)
			}
		default:
			return nil, fmt.Errorf("❌ [type %s | action %s] %w", g.Type, g.Action, lib.ErrUnknownAction)
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
			return fmt.Errorf("invalid record: %v", record)
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
