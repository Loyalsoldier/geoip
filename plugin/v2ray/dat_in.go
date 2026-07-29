package v2ray

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"google.golang.org/protobuf/proto"
)

const (
	TypeGeoIPDatIn = "v2rayGeoIPDat"
	DescGeoIPDatIn = "Convert V2Ray GeoIP dat to other formats"
)

func init() {
	lib.RegisterInputConfigCreator(TypeGeoIPDatIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return NewGeoIPDatInFromBytes(action, data)
	})
	lib.RegisterInputConverter(TypeGeoIPDatIn, &geoip_dat_in{
		Description: DescGeoIPDatIn,
	})
}

type geoip_dat_in struct {
	Type        string
	Action      lib.Action
	Description string
	URI         string
	Want        map[string]bool
	OnlyIPType  lib.IPType
}

func NewGeoIPDatIn(action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	g := &geoip_dat_in{
		Type:        TypeGeoIPDatIn,
		Action:      action,
		Description: DescGeoIPDatIn,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(g)
		}
	}

	if g.URI == "" {
		log.Fatalf("❌ [type %s | action %s] missing uri", g.Type, g.Action)
	}

	return g
}

func WithInputURI(uri string) lib.InputOption {
	return func(g lib.InputConverter) {
		g.(*geoip_dat_in).URI = strings.TrimSpace(uri)
	}
}

func WithInputWantedList(lists []string) lib.InputOption {
	return func(g lib.InputConverter) {
		wantList := make(map[string]bool)
		for _, want := range lists {
			if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
				wantList[want] = true
			}
		}

		g.(*geoip_dat_in).Want = wantList
	}
}

func WithInputOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(g lib.InputConverter) {
		g.(*geoip_dat_in).OnlyIPType = onlyIPType
	}
}

func NewGeoIPDatInFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
	var tmp struct {
		URI        string     `json:"uri"`
		Want       []string   `json:"wantedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	return NewGeoIPDatIn(
		action,
		WithInputURI(tmp.URI),
		WithInputWantedList(tmp.Want),
		WithInputOnlyIPType(tmp.OnlyIPType),
	), nil
}

func (g *geoip_dat_in) GetType() string {
	return g.Type
}

func (g *geoip_dat_in) GetAction() lib.Action {
	return g.Action
}

func (g *geoip_dat_in) GetDescription() string {
	return g.Description
}

func (g *geoip_dat_in) Input(container lib.Container) (lib.Container, error) {
	entries := make(map[string]*lib.Entry)
	var err error

	switch {
	case strings.HasPrefix(strings.ToLower(g.URI), "http://"), strings.HasPrefix(strings.ToLower(g.URI), "https://"):
		err = g.walkRemoteFile(g.URI, entries)
	default:
		err = g.walkLocalFile(g.URI, entries)
	}

	if err != nil {
		return nil, err
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

func (g *geoip_dat_in) walkLocalFile(path string, entries map[string]*lib.Entry) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := g.generateEntries(file, entries); err != nil {
		return err
	}

	return nil
}

func (g *geoip_dat_in) walkRemoteFile(url string, entries map[string]*lib.Entry) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("❌ [type %s | action %s] failed to get remote file %s, http status code %d", g.Type, g.Action, url, resp.StatusCode)
	}

	if err := g.generateEntries(resp.Body, entries); err != nil {
		return err
	}

	return nil
}

func (g *geoip_dat_in) generateEntries(reader io.Reader, entries map[string]*lib.Entry) error {
	geoipBytes, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	var geoipList GeoIPList
	if err := proto.Unmarshal(geoipBytes, &geoipList); err != nil {
		return err
	}

	for _, geoip := range geoipList.Entry {
		name := strings.ToUpper(strings.TrimSpace(geoip.CountryCode))

		if len(g.Want) > 0 && !g.Want[name] {
			continue
		}

		entry, found := entries[name]
		if !found {
			entry = lib.NewEntry(name)
		}

		for _, v2rayCIDR := range geoip.Cidr {
			ipStr := net.IP(v2rayCIDR.GetIp()).String() + "/" + fmt.Sprint(v2rayCIDR.GetPrefix())
			if err := entry.AddPrefix(ipStr); err != nil {
				return err
			}
		}

		entries[name] = entry
	}

	return nil
}
