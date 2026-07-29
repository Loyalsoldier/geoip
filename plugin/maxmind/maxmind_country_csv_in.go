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
	lib.RegisterInputConverter(TypeGeoLite2CountryCSVIn, &geolite2_country_csv_in{
		Description: DescGeoLite2CountryCSVIn,
	})
}

func NewGeoLite2CountryCSVIn(action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	g := &geolite2_country_csv_in{
		Type:            TypeGeoLite2CountryCSVIn,
		Action:          action,
		Description:     DescGeoLite2CountryCSVIn,
		CountryCodeFile: defaultGeoLite2CountryCodeFile,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(g)
		}
	}

	// When both of IP files are not specified,
	// it means user wants to use the default ones
	if g.IPv4File == "" && g.IPv6File == "" {
		g.IPv4File = defaultGeoLite2CountryIPv4File
		g.IPv6File = defaultGeoLite2CountryIPv6File
	}

	return g
}

// WithCountryCSVCountryCodeFile sets the country code CSV file of the input converter.
func WithCountryCSVCountryCodeFile(file string) lib.InputOption {
	return func(g lib.InputConverter) {
		file = strings.TrimSpace(file)
		if file == "" {
			return
		}
		g.(*geolite2_country_csv_in).CountryCodeFile = file
	}
}

// WithCountryCSVIPv4File sets the IPv4 CSV file of the input converter.
func WithCountryCSVIPv4File(file string) lib.InputOption {
	return func(g lib.InputConverter) {
		file = strings.TrimSpace(file)
		if file == "" {
			return
		}
		g.(*geolite2_country_csv_in).IPv4File = file
	}
}

// WithCountryCSVIPv6File sets the IPv6 CSV file of the input converter.
func WithCountryCSVIPv6File(file string) lib.InputOption {
	return func(g lib.InputConverter) {
		file = strings.TrimSpace(file)
		if file == "" {
			return
		}
		g.(*geolite2_country_csv_in).IPv6File = file
	}
}

// WithCountryCSVWantedList sets the wanted list of the input converter.
func WithCountryCSVWantedList(lists []string) lib.InputOption {
	return func(g lib.InputConverter) {
		wantList := make(map[string]bool)
		for _, want := range lists {
			if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
				wantList[want] = true
			}
		}
		g.(*geolite2_country_csv_in).Want = wantList
	}
}

// WithCountryCSVOnlyIPType sets the only IP type of the input converter.
func WithCountryCSVOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(g lib.InputConverter) {
		g.(*geolite2_country_csv_in).OnlyIPType = onlyIPType
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

	return NewGeoLite2CountryCSVIn(action,
		WithCountryCSVCountryCodeFile(tmp.CountryCodeFile),
		WithCountryCSVIPv4File(tmp.IPv4File),
		WithCountryCSVIPv6File(tmp.IPv6File),
		WithCountryCSVWantedList(tmp.Want),
		WithCountryCSVOnlyIPType(tmp.OnlyIPType),
	), nil
}

type geolite2_country_csv_in struct {
	Type            string
	Action          lib.Action
	Description     string
	CountryCodeFile string
	IPv4File        string
	IPv6File        string
	Want            map[string]bool
	OnlyIPType      lib.IPType
}

func (g *geolite2_country_csv_in) GetType() string {
	return g.Type
}

func (g *geolite2_country_csv_in) GetAction() lib.Action {
	return g.Action
}

func (g *geolite2_country_csv_in) GetDescription() string {
	return g.Description
}

func (g *geolite2_country_csv_in) Input(container lib.Container) (lib.Container, error) {
	ccMap, err := g.getCountryCode()
	if err != nil {
		return nil, err
	}

	entries := make(map[string]*lib.Entry, len(ccMap))

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

func (g *geolite2_country_csv_in) getCountryCode() (map[string]string, error) {
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

		if len(g.Want) > 0 && !g.Want[countryCode] {
			continue
		}

		ccMap[id] = countryCode
	}

	if len(ccMap) == 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] invalid country code data", g.Type, g.Action)
	}

	return ccMap, nil
}

func (g *geolite2_country_csv_in) process(file string, ccMap map[string]string, entries map[string]*lib.Entry) error {
	if len(ccMap) == 0 {
		return fmt.Errorf("❌ [type %s | action %s] invalid country code data", g.Type, g.Action)
	}
	if entries == nil {
		entries = make(map[string]*lib.Entry, len(ccMap))
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

		ccID := ""
		switch {
		case strings.TrimSpace(record[1]) != "":
			ccID = strings.TrimSpace(record[1])
		case strings.TrimSpace(record[2]) != "":
			ccID = strings.TrimSpace(record[2])
		case strings.TrimSpace(record[3]) != "":
			ccID = strings.TrimSpace(record[3])
		default:
			continue
		}

		if countryCode, found := ccMap[ccID]; found {
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
	}

	return nil
}
