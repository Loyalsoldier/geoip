package v2ray

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/v2fly/v2ray-core/v4/app/router"
	"github.com/v2fly/v2ray-core/v4/infra/conf/rule"
	"google.golang.org/protobuf/proto"
)

const (
	typeGeoIPdat = "v2rayGeoIPDat"
	descGeoIPdat = "Convert data to V2Ray GeoIP dat format"
)

var (
	defaultOutputName = "geoip.dat"
	defaultOutputDir  = filepath.Join("./", "output", "dat")
)

func init() {
	lib.RegisterOutputConfigCreator(typeGeoIPdat, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return newGeoIPDat(action, data)
	})
	lib.RegisterOutputConverter(typeGeoIPdat, &geoIPDat{
		Description: descGeoIPdat,
	})
}

func newGeoIPDat(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
	var tmp struct {
		OutputName     string     `json:"outputName"`
		OutputDir      string     `json:"outputDir"`
		Want           []string   `json:"wantedList"`
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

	return &geoIPDat{
		Type:           typeGeoIPdat,
		Action:         action,
		Description:    descGeoIPdat,
		OutputName:     tmp.OutputName,
		OutputDir:      tmp.OutputDir,
		Want:           tmp.Want,
		OneFilePerList: tmp.OneFilePerList,
		OnlyIPType:     tmp.OnlyIPType,
	}, nil
}

type geoIPDat struct {
	Type           string
	Action         lib.Action
	Description    string
	OutputName     string
	OutputDir      string
	Want           []string
	OneFilePerList bool
	OnlyIPType     lib.IPType
}

func (g *geoIPDat) GetType() string {
	return g.Type
}

func (g *geoIPDat) GetAction() lib.Action {
	return g.Action
}

func (g *geoIPDat) GetDescription() string {
	return g.Description
}

func (g *geoIPDat) Output(container lib.Container) error {
	// Filter want list
	wantList := make(map[string]bool)
	for _, want := range g.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList[want] = true
		}
	}

	geoIPList := new(router.GeoIPList)
	geoIPList.Entry = make([]*router.GeoIP, 0, 300)
	updated := false
	switch len(wantList) {
	case 0:
		for entry := range container.Loop() {
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

	default:
		for name := range wantList {
			entry, found := container.GetEntry(name)
			if !found {
				log.Printf("entry %s not found", name)
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
	}

	// Sort to make reproducible builds
	g.sort(geoIPList)

	if !g.OneFilePerList && updated {
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

func (g *geoIPDat) generateGeoIP(entry *lib.Entry) (*router.GeoIP, error) {
	var entryCidr []string
	var err error
	switch g.OnlyIPType {
	case lib.IPv4:
		entryCidr, err = entry.MarshalText(lib.IgnoreIPv6)
	case lib.IPv6:
		entryCidr, err = entry.MarshalText(lib.IgnoreIPv4)
	default:
		entryCidr, err = entry.MarshalText()
	}
	if err != nil {
		return nil, err
	}

	v2rayCIDR := make([]*router.CIDR, 0, 1024)
	for _, cidrStr := range entryCidr {
		cidr, err := rule.ParseIP(cidrStr)
		if err != nil {
			return nil, err
		}
		v2rayCIDR = append(v2rayCIDR, cidr)
	}

	if len(v2rayCIDR) > 0 {
		return &router.GeoIP{
			CountryCode: entry.GetName(),
			Cidr:        v2rayCIDR,
		}, nil
	}

	return nil, fmt.Errorf("entry %s has no CIDR", entry.GetName())
}

// Sort by country code to make reproducible builds
func (g *geoIPDat) sort(list *router.GeoIPList) {
	sort.SliceStable(list.Entry, func(i, j int) bool {
		return list.Entry[i].CountryCode < list.Entry[j].CountryCode
	})
}

func (g *geoIPDat) writeFile(filename string, geoIPBytes []byte) error {
	if err := os.MkdirAll(g.OutputDir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(g.OutputDir, filename), geoIPBytes, 0644); err != nil {
		return err
	}

	log.Printf("✅ [%s] %s --> %s", g.Type, filename, g.OutputDir)

	return nil
}
