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
	TypeSRSIn = "singboxSRS"
	DescSRSIn = "Convert sing-box SRS data to other formats"
)

func init() {
	lib.RegisterInputConfigCreator(TypeSRSIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newSRSIn(action, data)
	})
	lib.RegisterInputConverter(TypeSRSIn, &SRSIn{
		Description: DescSRSIn,
	})
}

func newSRSIn(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
	var tmp struct {
		Name       string     `json:"name"`
		URI        string     `json:"uri"`
		InputDir   string     `json:"inputDir"`
		Want       []string   `json:"wantedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	if tmp.Name == "" && tmp.URI == "" && tmp.InputDir == "" {
		return nil, fmt.Errorf("❌ [type %s | action %s] missing inputdir or name or uri", TypeSRSIn, action)
	}

	if (tmp.Name != "" && tmp.URI == "") || (tmp.Name == "" && tmp.URI != "") {
		return nil, fmt.Errorf("❌ [type %s | action %s] name & uri must be specified together", TypeSRSIn, action)
	}

	// Filter want list
	wantList := make(map[string]bool)
	for _, want := range tmp.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList[want] = true
		}
	}

	return &SRSIn{
		Type:        TypeSRSIn,
		Action:      action,
		Description: DescSRSIn,
		Name:        tmp.Name,
		URI:         tmp.URI,
		InputDir:    tmp.InputDir,
		Want:        wantList,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

type SRSIn struct {
	Type        string
	Action      lib.Action
	Description string
	Name        string
	URI         string
	InputDir    string
	Want        map[string]bool
	OnlyIPType  lib.IPType
}

func (s *SRSIn) GetType() string {
	return s.Type
}

func (s *SRSIn) GetAction() lib.Action {
	return s.Action
}

func (s *SRSIn) GetDescription() string {
	return s.Description
}

func (s *SRSIn) Input(container lib.Container) (lib.Container, error) {
	entries := make(map[string]*lib.Entry)
	var err error

	switch {
	case s.InputDir != "":
		err = s.walkDir(s.InputDir, entries)
	case s.Name != "" && s.URI != "":
		switch {
		case strings.HasPrefix(strings.ToLower(s.URI), "http://"), strings.HasPrefix(strings.ToLower(s.URI), "https://"):
			err = s.walkRemoteFile(s.URI, s.Name, entries)
		default:
			err = s.walkLocalFile(s.URI, s.Name, entries)
		}
	default:
		return nil, fmt.Errorf("❌ [type %s | action %s] config missing argument inputDir or name or uri", s.Type, s.Action)
	}

	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] no entry is generated", s.Type, s.Action)
	}

	var ignoreIPType lib.IgnoreIPOption
	switch s.OnlyIPType {
	case lib.IPv4:
		ignoreIPType = lib.IgnoreIPv6
	case lib.IPv6:
		ignoreIPType = lib.IgnoreIPv4
	}

	for _, entry := range entries {
		switch s.Action {
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

func (s *SRSIn) walkDir(dir string, entries map[string]*lib.Entry) error {
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

func (s *SRSIn) walkLocalFile(path, name string, entries map[string]*lib.Entry) error {
	entryName := ""
	name = strings.TrimSpace(name)
	if name != "" {
		entryName = name
	} else {
		entryName = filepath.Base(path)

		// check filename
		if !regexp.MustCompile(`^[a-zA-Z0-9_.\-]+$`).MatchString(entryName) {
			return fmt.Errorf("❌ [type %s | action %s] filename %s cannot be entry name, please remove special characters in it", s.Type, s.Action, entryName)
		}

		// remove file extension but not hidden files of which filename starts with "."
		dotIndex := strings.LastIndex(entryName, ".")
		if dotIndex > 0 {
			entryName = entryName[:dotIndex]
		}
	}

	entryName = strings.ToUpper(entryName)
	if _, found := entries[entryName]; found {
		return fmt.Errorf("❌ [type %s | action %s] found duplicated list %s", s.Type, s.Action, entryName)
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := s.generateEntries(entryName, file, entries); err != nil {
		return err
	}

	return nil
}

func (s *SRSIn) walkRemoteFile(url, name string, entries map[string]*lib.Entry) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("❌ [type %s | action %s] failed to get remote file %s, http status code %d", s.Type, s.Action, url, resp.StatusCode)
	}

	if err := s.generateEntries(name, resp.Body, entries); err != nil {
		return err
	}

	return nil
}

func (s *SRSIn) generateEntries(name string, reader io.Reader, entries map[string]*lib.Entry) error {
	name = strings.ToUpper(name)

	if len(s.Want) > 0 && !s.Want[name] {
		return nil
	}

	entry, found := entries[name]
	if !found {
		entry = lib.NewEntry(name)
	}

	plainRuleSet, err := srs.Read(reader, true)
	if err != nil {
		return err
	}

	for _, rule := range plainRuleSet.Rules {
		for _, cidrStr := range rule.DefaultOptions.IPCIDR {
			if err := entry.AddPrefix(cidrStr); err != nil {
				return err
			}
		}
	}

	entries[name] = entry
	return nil
}
