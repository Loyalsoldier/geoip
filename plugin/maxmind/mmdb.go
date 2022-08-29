package maxmind

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

const (
	typeMaxmindMMDB = "maxmindMMDB"
	descMaxmindMMDB = "Convert data to MaxMind mmdb database format"
)

var (
	defaultOutputName = "Country.mmdb"
	defaultOutputDir  = filepath.Join("./", "output", "maxmind")
)

func init() {
	lib.RegisterOutputConfigCreator(typeMaxmindMMDB, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return newMMDB(action, data)
	})
	lib.RegisterOutputConverter(typeMaxmindMMDB, &mmdb{
		Description: descMaxmindMMDB,
	})
}

func newMMDB(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
	var tmp struct {
		OutputName string     `json:"outputName"`
		OutputDir  string     `json:"outputDir"`
		Want       []string   `json:"wantedList"`
		Overwrite  []string   `json:"overwriteList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	if tmp.OutputName == "" {
		tmp.OutputName = defaultOutputName
	}

	if tmp.OutputDir == "" {
		tmp.OutputDir = defaultOutputDir
	}

	return &mmdb{
		Type:        typeMaxmindMMDB,
		Action:      action,
		Description: descMaxmindMMDB,
		OutputName:  tmp.OutputName,
		OutputDir:   tmp.OutputDir,
		Want:        tmp.Want,
		Overwrite:   tmp.Overwrite,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

type mmdb struct {
	Type        string
	Action      lib.Action
	Description string
	OutputName  string
	OutputDir   string
	Want        []string
	Overwrite   []string
	OnlyIPType  lib.IPType
}

func (m *mmdb) GetType() string {
	return m.Type
}

func (m *mmdb) GetAction() lib.Action {
	return m.Action
}

func (m *mmdb) GetDescription() string {
	return m.Description
}

func (m *mmdb) Output(container lib.Container) error {
	writer, err := mmdbwriter.New(
		mmdbwriter.Options{
			DatabaseType:            "GeoLite2-Country",
			Description:             map[string]string{"en": "Customized GeoLite2 Country database"},
			RecordSize:              24,
			IncludeReservedNetworks: true,
		},
	)
	if err != nil {
		return err
	}

	updated := false
	for _, name := range m.getEntryNameListInOrder(container) {
		entry, found := container.GetEntry(name)
		if !found {
			log.Printf("❌ entry %s not found", name)
			continue
		}
		if err := m.marshalData(writer, entry); err != nil {
			return err
		}
		updated = true
	}

	if updated {
		if err := m.writeFile(m.OutputName, writer); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("type %s | action %s failed to write file", m.Type, m.Action)
	}

	return nil
}

func (m *mmdb) getEntryNameListInOrder(container lib.Container) []string {
	/*
		Note: The IPs and/or CIDRs of the latter list will overwrite those of the former one
		when duplicated data found due to MaxMind mmdb file format constraint.

		Be sure to place the name of the most important list at last
		when writing wantedList and overwriteList in config file.

		The order of names in wantedList has a higher priority than which of the overwriteList.
	*/

	wantList := make([]string, 0, 200)
	for _, want := range m.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList = append(wantList, want)
		}
	}

	if len(wantList) > 0 {
		return wantList
	}

	overwriteList := make([]string, 0, 200)
	overwriteMap := make(map[string]bool)
	for _, overwrite := range m.Overwrite {
		if overwrite = strings.ToUpper(strings.TrimSpace(overwrite)); overwrite != "" {
			overwriteList = append(overwriteList, overwrite)
			overwriteMap[overwrite] = true
		}
	}

	list := make([]string, 0, 200)
	for entry := range container.Loop() {
		name := entry.GetName()
		_, found := overwriteMap[name]
		if found {
			continue
		}
		list = append(list, name)
	}

	// Make sure the names in overwriteList are written at last
	list = append(list, overwriteList...)

	return list
}

func (m *mmdb) marshalData(writer *mmdbwriter.Tree, entry *lib.Entry) error {
	var entryCidr []string
	var err error
	switch m.OnlyIPType {
	case lib.IPv4:
		entryCidr, err = entry.MarshalText(lib.IgnoreIPv6)
	case lib.IPv6:
		entryCidr, err = entry.MarshalText(lib.IgnoreIPv4)
	default:
		entryCidr, err = entry.MarshalText()
	}
	if err != nil {
		return err
	}

	record := mmdbtype.Map{
		"country": mmdbtype.Map{
			"iso_code": mmdbtype.String(entry.GetName()),
		},
	}

	for _, cidr := range entryCidr {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return err
		}
		if err := writer.Insert(network, record); err != nil {
			return err
		}
	}

	return nil
}

func (m *mmdb) writeFile(filename string, writer *mmdbwriter.Tree) error {
	if err := os.MkdirAll(m.OutputDir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(m.OutputDir, filename), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	_, err = writer.WriteTo(f)
	if err != nil {
		return err
	}

	log.Printf("✅ [%s] %s --> %s", m.Type, filename, m.OutputDir)

	return nil
}
