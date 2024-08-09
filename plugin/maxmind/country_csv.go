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
	typeCountryCSV = "maxmindGeoLite2CountryCSV"
	descCountryCSV = "Convert MaxMind GeoLite2 country CSV data to other formats"
)

var (
	defaultCCFile          = filepath.Join("./", "geolite2", "GeoLite2-Country-Locations-en.csv")
	defaultCountryIPv4File = filepath.Join("./", "geolite2", "GeoLite2-Country-Blocks-IPv4.csv")
	defaultCountryIPv6File = filepath.Join("./", "geolite2", "GeoLite2-Country-Blocks-IPv6.csv")
)

func init() {
	lib.RegisterInputConfigCreator(typeCountryCSV, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newGeoLite2CountryCSV(action, data)
	})
	lib.RegisterInputConverter(typeCountryCSV, &geoLite2CountryCSV{
		Description: descCountryCSV,
	})
}

func newGeoLite2CountryCSV(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
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

	if tmp.CountryCodeFile == "" {
		tmp.CountryCodeFile = defaultCCFile
	}

	if tmp.IPv4File == "" {
		tmp.IPv4File = defaultCountryIPv4File
	}

	if tmp.IPv6File == "" {
		tmp.IPv6File = defaultCountryIPv6File
	}

	return &geoLite2CountryCSV{
		Type:            typeCountryCSV,
		Action:          action,
		Description:     descCountryCSV,
		CountryCodeFile: tmp.CountryCodeFile,
		IPv4File:        tmp.IPv4File,
		IPv6File:        tmp.IPv6File,
		Want:            tmp.Want,
		OnlyIPType:      tmp.OnlyIPType,
	}, nil
}

type geoLite2CountryCSV struct {
	Type            string
	Action          lib.Action
	Description     string
	CountryCodeFile string
	IPv4File        string
	IPv6File        string
	Want            []string
	OnlyIPType      lib.IPType
}

func (g *geoLite2CountryCSV) GetType() string {
	return g.Type
}

func (g *geoLite2CountryCSV) GetAction() lib.Action {
	return g.Action
}

func (g *geoLite2CountryCSV) GetDescription() string {
	return g.Description
}

func (g *geoLite2CountryCSV) Input(container lib.Container) (lib.Container, error) {
	ccMap, err := g.getCountryCode()
	if err != nil {
		return nil, err
	}

	entries := make(map[string]*lib.Entry, 300)

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
		return nil, fmt.Errorf("❌ [type %s | action %s] no entry is generated", typeCountryCSV, g.Action)
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

func (g *geoLite2CountryCSV) getCountryCode() (map[string]string, error) {
	ccReader, err := os.Open(g.CountryCodeFile)
	if err != nil {
		return nil, err
	}
	defer ccReader.Close()

	reader := csv.NewReader(ccReader)
	lines, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	ccMap := make(map[string]string)
	for _, line := range lines[1:] {
		if len(line) < 5 {
			return nil, fmt.Errorf("❌ [type %s | action %s] invalid record: %v", typeCountryCSV, g.Action, line)
		}

		id := strings.TrimSpace(line[0])
		countryCode := strings.TrimSpace(line[4])
		if id == "" || countryCode == "" {
			continue
		}
		ccMap[id] = strings.ToUpper(countryCode)
	}

	if len(ccMap) == 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] invalid country code data", typeCountryCSV, g.Action)
	}

	return ccMap, nil
}

func (g *geoLite2CountryCSV) process(file string, ccMap map[string]string, entries map[string]*lib.Entry) error {
	if len(ccMap) == 0 {
		return fmt.Errorf("❌ [type %s | action %s] invalid country code data", typeCountryCSV, g.Action)
	}
	if entries == nil {
		entries = make(map[string]*lib.Entry, 300)
	}

	// Filter want list
	wantList := make(map[string]bool)
	for _, want := range g.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList[want] = true
		}
	}

	fReader, err := os.Open(file)
	if err != nil {
		return err
	}
	defer fReader.Close()

	reader := csv.NewReader(fReader)
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
			return fmt.Errorf("❌ [type %s | action %s] invalid record: %v", typeASNCSV, g.Action, record)
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
			if len(wantList) > 0 && !wantList[countryCode] {
				continue
			}
			cidrStr := strings.ToLower(strings.TrimSpace(record[0]))
			entry, found := entries[countryCode]
			if !found {
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
