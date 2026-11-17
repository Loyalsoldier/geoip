package maxmind

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
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
	lib.RegisterInputConverter(TypeGeoLite2ASNCSVIn, &geolite2_asn_csv_in{
		Description: DescGeoLite2ASNCSVIn,
	})
}

type geolite2_asn_csv_in struct {
	Type        string
	Action      lib.Action
	Description string
	IPv4File    string
	IPv6File    string
	Want        map[string][]string
	OnlyIPType  lib.IPType
}

func NewGeoLite2ASNCSVIn(action lib.Action, opts ...lib.InputOption) (lib.InputConverter, error) {
	g := &geolite2_asn_csv_in{
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

func NewGeoLite2ASNCSVInFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
	var tmp struct {
		IPv4File   string                   `json:"ipv4"`
		IPv6File   string                   `json:"ipv6"`
		Want       GeoLite2ASNCSVWantedList `json:"wantedList"`
		OnlyIPType lib.IPType               `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, fmt.Errorf("❌ [type %s | action %s] %w", TypeGeoLite2ASNCSVIn, action, err)
		}
	}

	return NewGeoLite2ASNCSVIn(
		action,
		WithIPv4File(tmp.IPv4File),
		WithIPv6File(tmp.IPv6File),
		WithGeoLite2ASNCSVWantedList(tmp.Want),
		WithInputOnlyIPType(tmp.OnlyIPType),
	)
}

func (g *geolite2_asn_csv_in) GetType() string {
	return g.Type
}

func (g *geolite2_asn_csv_in) GetAction() lib.Action {
	return g.Action
}

func (g *geolite2_asn_csv_in) GetDescription() string {
	return g.Description
}

func (g *geolite2_asn_csv_in) Input(container lib.Container) (lib.Container, error) {
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

func (g *geolite2_asn_csv_in) process(file string, entries map[string]*lib.Entry) error {
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

type GeoLite2ASNCSVWantedList struct {
	TypeSlice []string
	TypeMap   map[string][]string
}

func (g *GeoLite2ASNCSVWantedList) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)

	switch {
	case len(data) == 0:
		return errors.New("empty JSON")

	case bytes.Equal(data, []byte("null")):
		*g = GeoLite2ASNCSVWantedList{}
		return nil

	case data[0] == '[':
		var want []string
		if err := json.Unmarshal(data, &want); err != nil {
			return err
		}
		*g = GeoLite2ASNCSVWantedList{TypeSlice: want}
		return nil

	case data[0] == '{':
		var want map[string][]string
		if err := json.Unmarshal(data, &want); err != nil {
			return err
		}
		*g = GeoLite2ASNCSVWantedList{TypeMap: want}
		return nil

	default:
		return errors.New("expected JSON array or object")
	}
}

func normalizeGeoLite2ASNCSVWantedList(asn string, wantList map[string][]string, lists ...string) {
	asn = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(asn)), "as")
	if asn == "" {
		return
	}

	if len(lists) > 0 {
		for _, list := range lists {
			list = strings.ToUpper(strings.TrimSpace(list))
			if list != "" {
				wantList[asn] = append(wantList[asn], list)
			}
		}
	} else {
		wantList[asn] = append(wantList[asn], "AS"+asn)
	}
}

// WithGeoLite2ASNCSVWantedList is an input option for specifying the wanted list of ASNs or customized lists with specific ASNs for the `maxmindGeoLite2ASNCSV` input convertor.
// It accepts various types including int, string, []string, map[string][]string, and GeoLite2ASNCSVWantedList.
//
// For example:
// integer `12345`, string `12345` and string `AS12345` for a single ASN,
// string array or string slice `[]string{"AS12345", "AS67890"}` for multiple ASNs,
//
//	`map[string][]string{
//			"google": []string{"AS12345", "AS123"},
//			"apple":  []string{"AS67890"},
//	}` for customized lists with specific ASNs,
//
// or `GeoLite2ASNCSVWantedList` from parsing JSON configuration.
func WithGeoLite2ASNCSVWantedList[T int | string | []string | map[string][]string | GeoLite2ASNCSVWantedList](want T) lib.InputOption {
	return func(g lib.InputConverter) {
		wantList := make(map[string][]string) // map[asn][]listname or map[asn][]asn

		switch v := any(want).(type) {
		case int:
			normalizeGeoLite2ASNCSVWantedList(fmt.Sprintf("%d", v), wantList)

		case string:
			normalizeGeoLite2ASNCSVWantedList(v, wantList)

		case []string:
			for _, asn := range v {
				normalizeGeoLite2ASNCSVWantedList(asn, wantList)
			}

		case map[string][]string:
			for list, asnList := range v {
				for _, asn := range asnList {
					normalizeGeoLite2ASNCSVWantedList(asn, wantList, list)
				}
			}

		case GeoLite2ASNCSVWantedList:
			for _, asn := range v.TypeSlice {
				normalizeGeoLite2ASNCSVWantedList(asn, wantList)
			}
			for list, asnList := range v.TypeMap {
				for _, asn := range asnList {
					normalizeGeoLite2ASNCSVWantedList(asn, wantList, list)
				}
			}

		default:
			return
		}

		g.(*geolite2_asn_csv_in).Want = wantList
	}
}
