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
	TypeGeoLite2CountryCSVIn = "maxmindGeoLite2CountryCSV"
	DescGeoLite2CountryCSVIn = "Convert MaxMind GeoLite2 country CSV data to other formats"
)

var (
	defaultGeoLite2CountryCodeFile = filepath.Join("./", "geolite2", "GeoLite2-Country-Locations-en.csv")
	defaultGeoLite2CountryIPv4File = filepath.Join("./", "geolite2", "GeoLite2-Country-Blocks-IPv4.csv")
	defaultGeoLite2CountryIPv6File = filepath.Join("./", "geolite2", "GeoLite2-Country-Blocks-IPv6.csv")
)

func init() {
	lib.RegisterInputConfigCreator(TypeGeoLite2CountryCSVIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return NewGeoLite2CountryCSVInFromBytes(action, data)
	})
	lib.RegisterInputConverter(TypeGeoLite2CountryCSVIn, &geoLite2CountryCSVIn{
		Description: DescGeoLite2CountryCSVIn,
	})
}

func NewGeoLite2CountryCSVIn(action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	g := &geoLite2CountryCSVIn{
		Type:        TypeGeoLite2CountryCSVIn,
		Action:      action,
		Description: DescGeoLite2CountryCSVIn,
		Want:        make(map[string]bool),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(g)
		}
	}

	if g.CountryCodeFile == "" {
		g.CountryCodeFile = defaultGeoLite2CountryCodeFile
	}

	// Apply both defaults only after options, preserving single-family inputs.
	if g.IPv4File == "" && g.IPv6File == "" {
		g.IPv4File = defaultGeoLite2CountryIPv4File
		g.IPv6File = defaultGeoLite2CountryIPv6File
	}

	validateInput(g, g.OnlyIPType)
	return g
}

func WithCountryCodeFile(file string) lib.InputOption {
	return func(converter lib.InputConverter) {
		switch g := converter.(type) {
		case *geoLite2CountryCSVIn:
			g.CountryCodeFile = strings.TrimSpace(file)
		default:
			log.Fatalf("❌ maxmind WithCountryCodeFile does not support %T", converter)
		}
	}
}

func NewGeoLite2CountryCSVInFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
	var tmp struct {
		CountryCodeFile string     `json:"country"`
		IPv4File        string     `json:"ipv4"`
		IPv6File        string     `json:"ipv6"`
		Want            []string   `json:"wantedList"`
		OnlyIPType      lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	return NewGeoLite2CountryCSVIn(
		action,
		WithCountryCodeFile(tmp.CountryCodeFile),
		WithIPv4File(tmp.IPv4File),
		WithIPv6File(tmp.IPv6File),
		WithInputWantedList(tmp.Want),
		WithInputOnlyIPType(tmp.OnlyIPType),
	), nil
}

type geoLite2CountryCSVIn struct {
	Type            string
	Action          lib.Action
	Description     string
	CountryCodeFile string
	IPv4File        string
	IPv6File        string
	Want            map[string]bool
	OnlyIPType      lib.IPType
}

func (g *geoLite2CountryCSVIn) GetType() string {
	return g.Type
}

func (g *geoLite2CountryCSVIn) GetAction() lib.Action {
	return g.Action
}

func (g *geoLite2CountryCSVIn) GetDescription() string {
	return g.Description
}

func (g *geoLite2CountryCSVIn) Input(container lib.Container) (lib.Container, error) {
	ccMap, err := g.getCountryCode()
	if err != nil {
		return nil, err
	}

	entries := make(map[string]*lib.Entry)

	if g.IPv4File != "" {
		if err := g.process(g.IPv4File, ccMap, entries); err != nil {
			return nil, err
		}
	}

	if g.IPv6File != "" {
		if err := g.process(g.IPv6File, ccMap, entries); err != nil {
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

func (g *geoLite2CountryCSVIn) getCountryCode() (map[string]string, error) {
	var f io.ReadCloser
	var err error
	switch {
	case strings.HasPrefix(strings.ToLower(g.CountryCodeFile), "http://"), strings.HasPrefix(strings.ToLower(g.CountryCodeFile), "https://"):
		f, err = lib.GetRemoteURLReader(g.CountryCodeFile)
	default:
		f, err = os.Open(g.CountryCodeFile)
	}

	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	lines, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] empty country code file: %s", g.Type, g.Action, g.CountryCodeFile)
	}

	ccMap := make(map[string]string)
	for _, line := range lines[1:] {
		if len(line) < 5 {
			return nil, fmt.Errorf("❌ [type %s | action %s] invalid record: %v", g.Type, g.Action, line)
		}

		id := strings.TrimSpace(line[0])
		countryCode := strings.ToUpper(strings.TrimSpace(line[4]))
		if id == "" || countryCode == "" {
			continue
		}

		ccMap[id] = countryCode
	}

	if len(ccMap) == 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] invalid country code data", g.Type, g.Action)
	}

	return ccMap, nil
}

func (g *geoLite2CountryCSVIn) process(file string, ccMap map[string]string, entries map[string]*lib.Entry) error {
	if len(ccMap) == 0 {
		return fmt.Errorf("❌ [type %s | action %s] invalid country code data", g.Type, g.Action)
	}
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

		if len(record) < 4 {
			return fmt.Errorf("❌ [type %s | action %s] invalid record: %v", g.Type, g.Action, record)
		}

		// Use the first ID that maps to a country code in the order of
		// geoname_id, registered_country_geoname_id, represented_country_geoname_id,
		// as geoname_id may refer to a continent rather than a country.
		countryCode := ""
		for _, ccID := range record[1:4] {
			if cc, found := ccMap[strings.TrimSpace(ccID)]; found {
				countryCode = cc
				break
			}
		}
		if countryCode == "" {
			continue
		}

		if len(g.Want) > 0 && !g.Want[countryCode] {
			continue
		}

		cidrStr := strings.ToLower(strings.TrimSpace(record[0]))
		entry, got := entries[countryCode]
		if !got {
			entry = lib.NewEntry(countryCode)
		}

		if err := entry.AddPrefix(cidrStr); err != nil {
			return err
		}

		entries[countryCode] = entry
	}

	return nil
}
