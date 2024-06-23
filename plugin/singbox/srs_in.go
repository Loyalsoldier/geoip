package singbox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/sagernet/sing-box/common/srs"
)

const (
	typeSRSIn = "singboxSRS"
	descSRSIn = "Convert sing-box SRS data to other formats"
)

func init() {
	lib.RegisterInputConfigCreator(typeSRSIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newSRSIn(action, data)
	})
	lib.RegisterInputConverter(typeSRSIn, &srsIn{
		Description: descSRSIn,
	})
}

func newSRSIn(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
	var tmp struct {
		Name       string     `json:"name"`
		URI        string     `json:"uri"`
		InputDir   string     `json:"inputDir"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	if tmp.Name == "" && tmp.URI == "" && tmp.InputDir == "" {
		return nil, fmt.Errorf("type %s | action %s missing inputdir or name or uri", typeSRSIn, action)
	}

	if (tmp.Name != "" && tmp.URI == "") || (tmp.Name == "" && tmp.URI != "") {
		return nil, fmt.Errorf("type %s | action %s name & uri must be specified together", typeSRSIn, action)
	}

	return &srsIn{
		Type:        typeSRSIn,
		Action:      action,
		Description: descSRSIn,
		Name:        tmp.Name,
		URI:         tmp.URI,
		InputDir:    tmp.InputDir,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

type srsIn struct {
	Type        string
	Action      lib.Action
	Description string
	Name        string
	URI         string
	InputDir    string
	OnlyIPType  lib.IPType
}

func (s *srsIn) GetType() string {
	return s.Type
}

func (s *srsIn) GetAction() lib.Action {
	return s.Action
}

func (s *srsIn) GetDescription() string {
	return s.Description
}

func (s *srsIn) Input(container lib.Container) (lib.Container, error) {
	entries := make(map[string]*lib.Entry)
	var err error

	switch {
	case s.InputDir != "":
		err = s.walkDir(s.InputDir, entries)
	case s.Name != "" && s.URI != "":
		switch {
		case strings.HasPrefix(s.URI, "http://"), strings.HasPrefix(s.URI, "https://"):
			err = s.walkRemoteFile(s.URI, s.Name, entries)
		default:
			err = s.walkLocalFile(s.URI, s.Name, entries)
		}
	default:
		return nil, fmt.Errorf("config missing argument inputDir or name or uri")
	}

	if err != nil {
		return nil, err
	}

	var ignoreIPType lib.IgnoreIPOption
	switch s.OnlyIPType {
	case lib.IPv4:
		ignoreIPType = lib.IgnoreIPv6
	case lib.IPv6:
		ignoreIPType = lib.IgnoreIPv4
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("type %s | action %s no entry are generated", s.Type, s.Action)
	}

	for _, entry := range entries {
		switch s.Action {
		case lib.ActionAdd:
			if err := container.Add(entry, ignoreIPType); err != nil {
				return nil, err
			}
		case lib.ActionRemove:
			container.Remove(entry.GetName(), ignoreIPType)
		}
	}

	return container, nil
}

func (s *srsIn) walkDir(dir string, entries map[string]*lib.Entry) error {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		if err := s.walkLocalFile(path, "", entries); err != nil {
			return err
		}

		return nil
	})

	return err
}

func (s *srsIn) walkLocalFile(path, name string, entries map[string]*lib.Entry) error {
	name = strings.TrimSpace(name)
	var filename string
	if name != "" {
		filename = name
	} else {
		filename = filepath.Base(path)
	}

	// check filename
	if !regexp.MustCompile(`^[a-zA-Z0-9_.\-]+$`).MatchString(filename) {
		return fmt.Errorf("filename %s cannot be entry name, please remove special characters in it", filename)
	}
	dotIndex := strings.LastIndex(filename, ".")
	if dotIndex > 0 {
		filename = filename[:dotIndex]
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := s.generateEntries(filename, file, entries); err != nil {
		return err
	}

	return nil
}

func (s *srsIn) walkRemoteFile(url, name string, entries map[string]*lib.Entry) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to get remote file %s, http status code %d", url, resp.StatusCode)
	}

	if err := s.generateEntries(name, resp.Body, entries); err != nil {
		return err
	}

	return nil
}

func (s *srsIn) generateEntries(name string, reader io.Reader, entries map[string]*lib.Entry) error {
	entry := lib.NewEntry(name)
	if theEntry, found := entries[entry.GetName()]; found {
		fmt.Printf("⚠️ [type %s | action %s] found duplicated entry: %s. Process anyway\n", typeSRSIn, s.Action, name)
		entry = theEntry
	}

	plainRuleSet, err := srs.Read(reader, true)
	if err != nil {
		return err
	}

	for _, rule := range plainRuleSet.Rules {
		for _, cidrStr := range rule.DefaultOptions.IPCIDR {
			switch s.Action {
			case lib.ActionAdd:
				if err := entry.AddPrefix(cidrStr); err != nil {
					return err
				}
			case lib.ActionRemove:
				if err := entry.RemovePrefix(cidrStr); err != nil {
					return err
				}
			}
		}
	}

	entries[entry.GetName()] = entry

	return nil
}
