package v2ray

import (
	"encoding/json"
	"fmt"
	"io"
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
		return newGeoIPDatIn(action, data)
	})
	lib.RegisterInputConverter(TypeGeoIPDatIn, &GeoIPDatIn{
		Description: DescGeoIPDatIn,
	})
}

func newGeoIPDatIn(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
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

	if tmp.URI == "" {
		return nil, fmt.Errorf("❌ [type %s | action %s] uri must be specified in config", TypeGeoIPDatIn, action)
	}

	// Filter want list
	wantList := make(map[string]bool)
	for _, want := range tmp.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList[want] = true
		}
	}

	return &GeoIPDatIn{
		Type:        TypeGeoIPDatIn,
		Action:      action,
		Description: DescGeoIPDatIn,
		URI:         tmp.URI,
		Want:        wantList,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

type GeoIPDatIn struct {
	Type        string
	Action      lib.Action
	Description string
	URI         string
	Want        map[string]bool
	OnlyIPType  lib.IPType
}

func (g *GeoIPDatIn) GetType() string {
	return g.Type
}

func (g *GeoIPDatIn) GetAction() lib.Action {
	return g.Action
}

func (g *GeoIPDatIn) GetDescription() string {
	return g.Description
}

func (g *GeoIPDatIn) Input(container lib.Container) (lib.Container, error) {
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

func (g *GeoIPDatIn) walkLocalFile(path string, entries map[string]*lib.Entry) error {
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

func (g *GeoIPDatIn) walkRemoteFile(url string, entries map[string]*lib.Entry) error {
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

func (g *GeoIPDatIn) generateEntries(reader io.Reader, entries map[string]*lib.Entry) error {
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
