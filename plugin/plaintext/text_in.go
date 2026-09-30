package plaintext

import (
	"encoding/json"
	"fmt"
	"log"
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
		return NewTextInFromBytes(action, data)
	})
	lib.RegisterInputConverter(TypeTextIn, &textIn{
		Description: DescTextIn,
	})
}

func NewTextIn(action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	return newTextIn(TypeTextIn, DescTextIn, action, opts...)
}

func newTextIn(iType, iDesc string, action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	t := &textIn{
		Type:        iType,
		Action:      action,
		Description: iDesc,
		Want:        make(map[string]bool),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(t)
		}
	}

	if t.Action != lib.ActionAdd && t.Action != lib.ActionRemove {
		log.Fatalf("❌ [type %s | action %s] only supports add or remove action", t.Type, t.Action)
	}
	if t.OnlyIPType != "" && t.OnlyIPType != lib.IPv4 && t.OnlyIPType != lib.IPv6 {
		log.Fatalf("❌ [type %s | action %s] invalid onlyIPType: %s", t.Type, t.Action, t.OnlyIPType)
	}
	if t.Type != TypeTextIn && (len(t.IPOrCIDR) > 0 || t.inlineSource) {
		log.Fatalf("❌ [type %s | action %s] ipOrCIDR is invalid for this input format", t.Type, t.Action)
	}
	if t.Type == TypeJSONIn {
		if len(t.JSONPath) == 0 {
			log.Fatalf("❌ [type %s | action %s] missing jsonPath", t.Type, t.Action)
		}
		for _, path := range t.JSONPath {
			if strings.TrimSpace(path) == "" {
				log.Fatalf("❌ [type %s | action %s] jsonPath must not contain empty paths", t.Type, t.Action)
			}
		}
	}

	if t.InputDir != "" {
		if t.Name != "" || t.URI != "" || len(t.IPOrCIDR) > 0 || t.inlineSource {
			log.Fatalf("❌ [type %s | action %s] inputDir is not allowed to be used with name or uri or ipOrCIDR", t.Type, t.Action)
		}
	} else {
		if t.Name == "" {
			log.Fatalf("❌ [type %s | action %s] missing inputDir or name", t.Type, t.Action)
		}
		if t.inlineSource {
			if len(t.IPOrCIDR) == 0 {
				log.Fatalf("❌ [type %s | action %s] name and ipOrCIDR must be specified together", t.Type, t.Action)
			}
		} else if t.URI == "" {
			log.Fatalf("❌ [type %s | action %s] name and uri must be specified together", t.Type, t.Action)
		}
	}

	return t
}

func WithNameAndURI(name, uri string) lib.InputOption {
	return func(c lib.InputConverter) {
		t := c.(*textIn)
		t.Name = strings.TrimSpace(name)
		t.URI = strings.TrimSpace(uri)
		t.inlineSource = false
	}
}

// WithNameAndIPOrCIDR selects a named inline text source without a URI.
func WithNameAndIPOrCIDR(name string, ipOrCIDR []string) lib.InputOption {
	return func(c lib.InputConverter) {
		t := c.(*textIn)
		t.Name = strings.TrimSpace(name)
		t.URI = ""
		t.IPOrCIDR = ipOrCIDR
		t.inlineSource = true
	}
}

// WithIPOrCIDR supplements a named text URI source with inline addresses.
func WithIPOrCIDR(ipOrCIDR []string) lib.InputOption {
	return func(t lib.InputConverter) {
		t.(*textIn).IPOrCIDR = ipOrCIDR
	}
}

func WithInputDir(dir string) lib.InputOption {
	return func(t lib.InputConverter) {
		t.(*textIn).InputDir = strings.TrimSpace(dir)
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
		t.(*textIn).Want = wantList
	}
}

func WithInputOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(t lib.InputConverter) {
		t.(*textIn).OnlyIPType = lib.IPType(strings.ToLower(strings.TrimSpace(string(onlyIPType))))
	}
}

func WithJSONPath(paths []string) lib.InputOption {
	return func(t lib.InputConverter) {
		t.(*textIn).JSONPath = paths
	}
}

func WithRemovePrefixesInLine(prefixes []string) lib.InputOption {
	return func(t lib.InputConverter) {
		t.(*textIn).RemovePrefixesInLine = prefixes
	}
}

func WithRemoveSuffixesInLine(suffixes []string) lib.InputOption {
	return func(t lib.InputConverter) {
		t.(*textIn).RemoveSuffixesInLine = suffixes
	}
}

func NewTextInFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
	return newTextInFromBytes(NewTextIn, action, data)
}

func newTextInFromBytes(constructor func(lib.Action, ...lib.InputOption) lib.InputConverter, action lib.Action, data []byte) (lib.InputConverter, error) {
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

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	source := WithNameAndURI(tmp.Name, tmp.URI)
	if strings.TrimSpace(tmp.URI) == "" && len(tmp.IPOrCIDR) > 0 {
		source = WithNameAndIPOrCIDR(tmp.Name, tmp.IPOrCIDR)
	}

	return constructor(
		action,
		source,
		WithIPOrCIDR(tmp.IPOrCIDR),
		WithInputDir(tmp.InputDir),
		WithInputWantedList(tmp.Want),
		WithInputOnlyIPType(tmp.OnlyIPType),
		WithJSONPath(tmp.JSONPath),
		WithRemovePrefixesInLine(tmp.RemovePrefixesInLine),
		WithRemoveSuffixesInLine(tmp.RemoveSuffixesInLine),
	), nil
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

func (t *textIn) walkRemoteFile(url, name string, entries map[string]*lib.Entry) error {
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

func (t *textIn) appendIPOrCIDR(ipOrCIDR []string, name string, entries map[string]*lib.Entry) error {
	if len(ipOrCIDR) == 0 {
		return nil
	}

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
