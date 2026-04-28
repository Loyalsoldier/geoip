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
	TypeTextIn = "text"
	DescTextIn = "Convert plaintext IP & CIDR to other formats"
)

func init() {
	lib.RegisterInputConfigCreator(TypeTextIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return NewTextInFromBytes(TypeTextIn, DescTextIn, action, data)
	})
	lib.RegisterInputConverter(TypeTextIn, &text_in{
		Description: DescTextIn,
	})
}

func NewTextIn(iType string, iDesc string, action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	t := &text_in{
		Type:        iType,
		Action:      action,
		Description: iDesc,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(t)
		}
	}

	return t
}

func WithNameAndURI(name, uri string) lib.InputOption {
	return func(t lib.InputConverter) {
		t.(*text_in).Name = strings.TrimSpace(name)
		t.(*text_in).URI = strings.TrimSpace(uri)
	}
}

func WithIPOrCIDR(ipOrCIDR []string) lib.InputOption {
	return func(t lib.InputConverter) {
		t.(*text_in).IPOrCIDR = ipOrCIDR
	}
}

func WithInputDir(dir string) lib.InputOption {
	return func(t lib.InputConverter) {
		t.(*text_in).InputDir = strings.TrimSpace(dir)
	}
}

func WithInputWantedList(lists []string) lib.InputOption {
	return func(t lib.InputConverter) {
		wantList := make(map[string]bool)
		for _, want := range lists {
			if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
				wantList[want] = true
			}
		}

		t.(*text_in).Want = wantList
	}
}

func WithInputOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(t lib.InputConverter) {
		t.(*text_in).OnlyIPType = onlyIPType
	}
}

func WithJSONPath(jsonPath []string) lib.InputOption {
	return func(t lib.InputConverter) {
		t.(*text_in).JSONPath = jsonPath
	}
}

func WithRemovePrefixesInLine(prefixes []string) lib.InputOption {
	return func(t lib.InputConverter) {
		t.(*text_in).RemovePrefixesInLine = prefixes
	}
}

func WithRemoveSuffixesInLine(suffixes []string) lib.InputOption {
	return func(t lib.InputConverter) {
		t.(*text_in).RemoveSuffixesInLine = suffixes
	}
}

func NewTextInFromBytes(iType string, iDesc string, action lib.Action, data []byte) (lib.InputConverter, error) {
	var tmp struct {
		Name       string     `json:"name"`
		URI        string     `json:"uri"`
		IPOrCIDR   []string   `json:"ipOrCIDR"`
		InputDir   string     `json:"inputDir"`
		Want       []string   `json:"wantedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`

		JSONPath             []string `json:"jsonPath"`
		RemovePrefixesInLine []string `json:"removePrefixesInLine"`
		RemoveSuffixesInLine []string `json:"removeSuffixesInLine"`
	}

	if strings.TrimSpace(iType) == "" {
		return nil, fmt.Errorf("type is required")
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	if iType != TypeTextIn && len(tmp.IPOrCIDR) > 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] ipOrCIDR is invalid for this input format", iType, action)
	}

	if iType == TypeJSONIn && len(tmp.JSONPath) == 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] missing jsonPath", iType, action)
	}

	if tmp.InputDir == "" {
		if tmp.Name == "" {
			return nil, fmt.Errorf("❌ [type %s | action %s] missing inputDir or name", iType, action)
		}
		if tmp.URI == "" && len(tmp.IPOrCIDR) == 0 {
			return nil, fmt.Errorf("❌ [type %s | action %s] missing uri or ipOrCIDR", iType, action)
		}
	} else if tmp.Name != "" || tmp.URI != "" || len(tmp.IPOrCIDR) > 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] inputDir is not allowed to be used with name or uri or ipOrCIDR", iType, action)
	}

	return NewTextIn(
		iType, iDesc, action,
		WithNameAndURI(tmp.Name, tmp.URI),
		WithIPOrCIDR(tmp.IPOrCIDR),
		WithInputDir(tmp.InputDir),
		WithInputWantedList(tmp.Want),
		WithInputOnlyIPType(tmp.OnlyIPType),
		WithJSONPath(tmp.JSONPath),
		WithRemovePrefixesInLine(tmp.RemovePrefixesInLine),
		WithRemoveSuffixesInLine(tmp.RemoveSuffixesInLine),
	), nil
}

func (t *text_in) GetType() string {
	return t.Type
}

func (t *text_in) GetAction() lib.Action {
	return t.Action
}

func (t *text_in) GetDescription() string {
	return t.Description
}

func (t *text_in) Input(container lib.Container) (lib.Container, error) {
	entries := make(map[string]*lib.Entry)
	var err error

	switch {
	case t.InputDir != "":
		err = t.walkDir(t.InputDir, entries)

	case t.Name != "" && t.URI != "":
		switch {
		case strings.HasPrefix(strings.ToLower(t.URI), "http://"), strings.HasPrefix(strings.ToLower(t.URI), "https://"):
			err = t.walkRemoteFile(t.URI, t.Name, entries)
		default:
			err = t.walkLocalFile(t.URI, t.Name, entries)
		}
		if err != nil {
			return nil, err
		}

		fallthrough

	case t.Name != "" && len(t.IPOrCIDR) > 0:
		err = t.appendIPOrCIDR(t.IPOrCIDR, t.Name, entries)

	default:
		return nil, fmt.Errorf("❌ [type %s | action %s] config missing argument inputDir or name or uri or ipOrCIDR", t.Type, t.Action)
	}

	if err != nil {
		return nil, err
	}

	ignoreIPType := lib.GetIgnoreIPType(t.OnlyIPType)

	if len(entries) == 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] no entry is generated", t.Type, t.Action)
	}

	for _, entry := range entries {
		switch t.Action {
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

func (t *text_in) walkDir(dir string, entries map[string]*lib.Entry) error {
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

func (t *text_in) walkLocalFile(path, name string, entries map[string]*lib.Entry) error {
	entryName := ""
	name = strings.TrimSpace(name)
	if name != "" {
		entryName = name
	} else {
		entryName = filepath.Base(path)

		// check filename
		if !regexp.MustCompile(`^[a-zA-Z0-9_.\-]+$`).MatchString(entryName) {
			return fmt.Errorf("❌ [type %s | action %s] filename %s cannot be entry name, please remove special characters in it", t.Type, t.Action, entryName)
		}

		// remove file extension but not hidden files of which filename starts with "."
		dotIndex := strings.LastIndex(entryName, ".")
		if dotIndex > 0 {
			entryName = entryName[:dotIndex]
		}
	}

	entryName = strings.ToUpper(entryName)

	if len(t.Want) > 0 && !t.Want[entryName] {
		return nil
	}
	if _, found := entries[entryName]; found {
		return fmt.Errorf("❌ [type %s | action %s] found duplicated list %s", t.Type, t.Action, entryName)
	}

	entry := lib.NewEntry(entryName)
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := t.scanFile(file, entry); err != nil {
		return err
	}

	entries[entryName] = entry

	return nil
}

func (t *text_in) walkRemoteFile(url, name string, entries map[string]*lib.Entry) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("❌ [type %s | action %s] failed to get remote file %s, http status code %d", t.Type, t.Action, url, resp.StatusCode)
	}

	name = strings.ToUpper(name)

	if len(t.Want) > 0 && !t.Want[name] {
		return nil
	}

	entry := lib.NewEntry(name)
	if err := t.scanFile(resp.Body, entry); err != nil {
		return err
	}

	entries[name] = entry

	return nil
}

func (t *text_in) appendIPOrCIDR(ipOrCIDR []string, name string, entries map[string]*lib.Entry) error {
	name = strings.ToUpper(name)

	entry, found := entries[name]
	if !found {
		entry = lib.NewEntry(name)
	}

	for _, cidr := range ipOrCIDR {
		if err := entry.AddPrefix(strings.TrimSpace(cidr)); err != nil {
			return err
		}
	}

	entries[name] = entry

	return nil
}
