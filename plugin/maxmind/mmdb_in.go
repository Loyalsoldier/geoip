package maxmind

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/oschwald/maxminddb-golang"
)

const (
	typeMaxmindMMDBIn = "maxmindMMDB"
	descMaxmindMMDBIn = "Convert MaxMind mmdb database to other formats"
)

var (
	defaultMMDBFile = filepath.Join("./", "geolite2", "GeoLite2-Country.mmdb")
	tempMMDBPath    = filepath.Join("./", "tmp")
	tempMMDBFile    = filepath.Join(tempMMDBPath, "input.mmdb")
)

func init() {
	lib.RegisterInputConfigCreator(typeMaxmindMMDBIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newMaxmindMMDBIn(action, data)
	})
	lib.RegisterInputConverter(typeMaxmindMMDBIn, &maxmindMMDBIn{
		Description: descMaxmindMMDBIn,
	})
}

func newMaxmindMMDBIn(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
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
		tmp.URI = defaultMMDBFile
	}

	return &maxmindMMDBIn{
		Type:        typeMaxmindMMDBIn,
		Action:      action,
		Description: descMaxmindMMDBIn,
		URI:         tmp.URI,
		Want:        tmp.Want,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

type maxmindMMDBIn struct {
	Type        string
	Action      lib.Action
	Description string
	URI         string
	Want        []string
	OnlyIPType  lib.IPType
}

func (g *maxmindMMDBIn) GetType() string {
	return g.Type
}

func (g *maxmindMMDBIn) GetAction() lib.Action {
	return g.Action
}

func (g *maxmindMMDBIn) GetDescription() string {
	return g.Description
}

func (g *maxmindMMDBIn) Input(container lib.Container) (lib.Container, error) {
	var fd io.ReadCloser
	var err error
	switch {
	case strings.HasPrefix(g.URI, "http://"), strings.HasPrefix(g.URI, "https://"):
		fd, err = g.downloadFile(g.URI)
	default:
		fd, err = os.Open(g.URI)
	}

	if err != nil {
		return nil, err
	}

	err = g.moveFile(fd)
	if err != nil {
		return nil, err
	}

	entries := make(map[string]*lib.Entry)
	err = g.generateEntries(entries)
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] no entry is newly generated", typeMaxmindMMDBIn, g.Action)
	}

	var ignoreIPType lib.IgnoreIPOption
	switch g.OnlyIPType {
	case lib.IPv4:
		ignoreIPType = lib.IgnoreIPv6
	case lib.IPv6:
		ignoreIPType = lib.IgnoreIPv4
	}

	// Filter want list
	wantList := make(map[string]bool)
	for _, want := range g.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList[want] = true
		}
	}

	for _, entry := range entries {
		name := entry.GetName()
		if len(wantList) > 0 && !wantList[name] {
			continue
		}

		switch g.Action {
		case lib.ActionAdd:
			if err := container.Add(entry, ignoreIPType); err != nil {
				return nil, err
			}
		case lib.ActionRemove:
			container.Remove(name, ignoreIPType)
		}
	}

	return container, nil
}

func (g *maxmindMMDBIn) downloadFile(url string) (io.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get remote file %s, http status code %d", url, resp.StatusCode)
	}

	return resp.Body, nil
}

func (g *maxmindMMDBIn) moveFile(src io.ReadCloser) error {
	defer src.Close()

	err := os.MkdirAll(tempMMDBPath, 0755)
	if err != nil {
		return err
	}

	out, err := os.Create(tempMMDBFile)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)

	return err
}

func (g *maxmindMMDBIn) generateEntries(entries map[string]*lib.Entry) error {
	db, err := maxminddb.Open(tempMMDBFile)
	if err != nil {
		return err
	}
	defer db.Close()

	record := struct {
		Country struct {
			IsoCode string `maxminddb:"iso_code"`
		} `maxminddb:"country"`
	}{}

	networks := db.Networks(maxminddb.SkipAliasedNetworks)
	for networks.Next() {
		subnet, err := networks.Network(&record)
		if err != nil {
			continue
		}

		var entry *lib.Entry
		name := strings.ToUpper(record.Country.IsoCode)
		if theEntry, found := entries[name]; found {
			entry = theEntry
		} else {
			entry = lib.NewEntry(name)
		}

		switch g.Action {
		case lib.ActionAdd:
			if err := entry.AddPrefix(subnet); err != nil {
				return err
			}
		case lib.ActionRemove:
			if err := entry.RemovePrefix(subnet.String()); err != nil {
				return err
			}
		}

		entries[name] = entry
	}

	if networks.Err() != nil {
		return networks.Err()
	}

	return nil
}
