package v2ray

import (
	"encoding/json"
	"fmt"
	"log"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"google.golang.org/protobuf/proto"
)

const (
	TypeGeoIPdatOut = "v2rayGeoIPDat"
	DescGeoIPdatOut = "Convert data to V2Ray GeoIP dat format"
)

var (
	defaultOutputName = "geoip.dat"
	defaultOutputDir  = filepath.Join("./", "output", "dat")
)

func init() {
	lib.RegisterOutputConfigCreator(TypeGeoIPdatOut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return newGeoIPDat(action, data)
	})
	lib.RegisterOutputConverter(TypeGeoIPdatOut, &GeoIPDatOut{
		Description: DescGeoIPdatOut,
	})
}

func newGeoIPDat(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
	var tmp struct {
		OutputName     string     `json:"outputName"`
		OutputDir      string     `json:"outputDir"`
		Want           []string   `json:"wantedList"`
		Exclude        []string   `json:"excludedList"`
		OneFilePerList bool       `json:"oneFilePerList"`
		OnlyIPType     lib.IPType `json:"onlyIPType"`
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

	return &GeoIPDatOut{
		Type:           TypeGeoIPdatOut,
		Action:         action,
		Description:    DescGeoIPdatOut,
		OutputName:     tmp.OutputName,
		OutputDir:      tmp.OutputDir,
		Want:           tmp.Want,
		Exclude:        tmp.Exclude,
		OneFilePerList: tmp.OneFilePerList,
		OnlyIPType:     tmp.OnlyIPType,
	}, nil
}

type GeoIPDatOut struct {
	Type           string
	Action         lib.Action
	Description    string
	OutputName     string
	OutputDir      string
	Want           []string
	Exclude        []string
	OneFilePerList bool
	OnlyIPType     lib.IPType
}

func (g *GeoIPDatOut) GetType() string {
	return g.Type
}

func (g *GeoIPDatOut) GetAction() lib.Action {
	return g.Action
}

func (g *GeoIPDatOut) GetDescription() string {
	return g.Description
}

func (g *GeoIPDatOut) Output(container lib.Container) error {
	geoIPList := new(GeoIPList)
	geoIPList.Entry = make([]*GeoIP, 0, 300)
	updated := false

	for _, name := range g.filterAndSortList(container) {
		entry, found := container.GetEntry(name)
		if !found {
			log.Printf("❌ entry %s not found\n", name)
			continue
		}

		geoIP, err := g.generateGeoIP(entry)
		if err != nil {
			return err
		}
		geoIPList.Entry = append(geoIPList.Entry, geoIP)
		updated = true

		if g.OneFilePerList {
			geoIPBytes, err := proto.Marshal(geoIPList)
			if err != nil {
				return err
			}

			filename := strings.ToLower(entry.GetName()) + ".dat"
			if err := g.writeFile(filename, geoIPBytes); err != nil {
				return err
			}

			geoIPList.Entry = nil
		}
	}

	if !g.OneFilePerList && updated {
		// Sort to make reproducible builds
		g.sort(geoIPList)

		geoIPBytes, err := proto.Marshal(geoIPList)
		if err != nil {
			return err
		}
		if err := g.writeFile(g.OutputName, geoIPBytes); err != nil {
			return err
		}
	}

	return nil
}

func (g *GeoIPDatOut) filterAndSortList(container lib.Container) []string {
	excludeMap := make(map[string]bool)
	for _, exclude := range g.Exclude {
		if exclude = strings.ToUpper(strings.TrimSpace(exclude)); exclude != "" {
			excludeMap[exclude] = true
		}
	}

	wantList := make([]string, 0, len(g.Want))
	for _, want := range g.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" && !excludeMap[want] {
			wantList = append(wantList, want)
		}
	}

	if len(wantList) > 0 {
		// Sort the list
		slices.Sort(wantList)
		return wantList
	}

	list := make([]string, 0, 300)
	for entry := range container.Loop() {
		name := entry.GetName()
		if excludeMap[name] {
			continue
		}
		list = append(list, name)
	}

	// Sort the list
	slices.Sort(list)

	return list
}

func (g *GeoIPDatOut) generateGeoIP(entry *lib.Entry) (*GeoIP, error) {
	var entryCidr []netip.Prefix
	var err error
	switch g.OnlyIPType {
	case lib.IPv4:
		entryCidr, err = entry.MarshalPrefix(lib.IgnoreIPv6)
	case lib.IPv6:
		entryCidr, err = entry.MarshalPrefix(lib.IgnoreIPv4)
	default:
		entryCidr, err = entry.MarshalPrefix()
	}
	if err != nil {
		return nil, err
	}

	v2rayCIDR := make([]*CIDR, 0, len(entryCidr))
	for _, prefix := range entryCidr {
		v2rayCIDR = append(v2rayCIDR, &CIDR{
			Ip:     prefix.Addr().AsSlice(),
			Prefix: uint32(prefix.Bits()),
		})
	}

	if len(v2rayCIDR) > 0 {
		return &GeoIP{
			CountryCode: entry.GetName(),
			Cidr:        v2rayCIDR,
		}, nil
	}

	return nil, fmt.Errorf("❌ [type %s | action %s] entry %s has no CIDR", g.Type, g.Action, entry.GetName())
}

// Sort by country code to make reproducible builds
func (g *GeoIPDatOut) sort(list *GeoIPList) {
	sort.SliceStable(list.Entry, func(i, j int) bool {
		return list.Entry[i].CountryCode < list.Entry[j].CountryCode
	})
}

func (g *GeoIPDatOut) writeFile(filename string, geoIPBytes []byte) error {
	if err := os.MkdirAll(g.OutputDir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(g.OutputDir, filename), geoIPBytes, 0644); err != nil {
		return err
	}

	log.Printf("✅ [%s] %s --> %s", g.Type, filename, g.OutputDir)

	return nil
}
