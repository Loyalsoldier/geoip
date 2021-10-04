package plaintext

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	typeTextIn = "text"
	descTextIn = "Convert plaintext IP & CIDR to other formats"
)

func init() {
	lib.RegisterInputConfigCreator(typeTextIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newTextIn(typeTextIn, action, data)
	})
	lib.RegisterInputConverter(typeTextIn, &textIn{
		Description: descTextIn,
	})
}

func newTextIn(iType string, action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
	var tmp struct {
		Name       string     `json:"name"`
		URI        string     `json:"uri"`
		InputDir   string     `json:"inputDir"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if strings.TrimSpace(iType) == "" {
		return nil, fmt.Errorf("type is required")
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	if tmp.Name == "" && tmp.URI == "" && tmp.InputDir == "" {
		return nil, fmt.Errorf("type %s | action %s missing inputdir or name or uri", typeTextIn, action)
	}

	if (tmp.Name != "" && tmp.URI == "") || (tmp.Name == "" && tmp.URI != "") {
		return nil, fmt.Errorf("type %s | action %s name & uri must be specified together", typeTextIn, action)
	}

	return &textIn{
		Type:        iType,
		Action:      action,
		Description: descTextIn,
		Name:        tmp.Name,
		URI:         tmp.URI,
		InputDir:    tmp.InputDir,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

func (t *textIn) GetType() string {
	return t.Type
}

func (t *textIn) GetAction() lib.Action {
	return t.Action
}

func (t *textIn) GetDescription() string {
	return t.Description
}

func (t *textIn) Input(container lib.Container) (lib.Container, error) {
	entries := make(map[string]*lib.Entry)
	var err error

	switch {
	case t.InputDir != "":
		err = t.walkDir(t.InputDir, entries)
	case t.Name != "" && t.URI != "":
		switch {
		case strings.HasPrefix(t.URI, "http://"), strings.HasPrefix(t.URI, "https://"):
			err = t.walkRemoteFile(t.URI, t.Name, entries)
		default:
			err = t.walkLocalFile(t.URI, t.Name, entries)
		}
	default:
		return nil, fmt.Errorf("config missing argument inputDir or name or uri")
	}

	if err != nil {
		return nil, err
	}

	var ignoreIPType lib.IgnoreIPOption
	switch t.OnlyIPType {
	case lib.IPv4:
		ignoreIPType = lib.IgnoreIPv6
	case lib.IPv6:
		ignoreIPType = lib.IgnoreIPv4
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("type %s | action %s no entry are generated", t.Type, t.Action)
	}

	for _, entry := range entries {
		switch t.Action {
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

func (t *textIn) walkDir(dir string, entries map[string]*lib.Entry) error {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		if err := t.walkLocalFile(path, "", entries); err != nil {
			return err
		}

		return nil
	})

	return err
}

func (t *textIn) walkLocalFile(path, name string, entries map[string]*lib.Entry) error {
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

	if _, found := entries[filename]; found {
		return fmt.Errorf("found duplicated file %s", filename)
	}

	entry := lib.NewEntry(filename)
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := t.scanFile(file, entry); err != nil {
		return err
	}

	entries[filename] = entry

	return nil
}

func (t *textIn) walkRemoteFile(url, name string, entries map[string]*lib.Entry) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to get remote file %s, http status code %d", url, resp.StatusCode)
	}

	entry := lib.NewEntry(name)
	if err := t.scanFile(resp.Body, entry); err != nil {
		return err
	}

	entries[name] = entry
	return nil
}
