package plaintext

import (
	"encoding/json"
	"log"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	TypeTextOut = "text"
	DescTextOut = "Convert data to plaintext CIDR format"
)

func init() {
	lib.RegisterOutputConfigCreator(TypeTextOut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return NewTextOutFromBytes(TypeTextOut, DescTextOut, action, data)
	})
	lib.RegisterOutputConverter(TypeTextOut, &text_out{
		Description: DescTextOut,
	})
}

func NewTextOut(oType string, oDesc string, action lib.Action, opts ...lib.OutputOption) lib.OutputConverter {
	t := &text_out{
		Type:        oType,
		Action:      action,
		Description: oDesc,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(t)
		}
	}

	return t
}

func WithTextOutputDir(dir string) lib.OutputOption {
	return func(t lib.OutputConverter) {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			// Use default based on type (set in NewTextOutFromBytes)
			return
		}
		t.(*text_out).OutputDir = dir
	}
}

func WithTextOutputExt(ext string) lib.OutputOption {
	return func(t lib.OutputConverter) {
		ext = strings.TrimSpace(ext)
		if ext == "" {
			ext = ".txt"
		}
		t.(*text_out).OutputExt = ext
	}
}

func WithTextOutputWantedList(lists []string) lib.OutputOption {
	return func(t lib.OutputConverter) {
		t.(*text_out).Want = lists
	}
}

func WithTextOutputExcludedList(lists []string) lib.OutputOption {
	return func(t lib.OutputConverter) {
		t.(*text_out).Exclude = lists
	}
}

func WithTextOutputOnlyIPType(onlyIPType lib.IPType) lib.OutputOption {
	return func(t lib.OutputConverter) {
		t.(*text_out).OnlyIPType = onlyIPType
	}
}

func WithTextAddPrefixInLine(prefix string) lib.OutputOption {
	return func(t lib.OutputConverter) {
		t.(*text_out).AddPrefixInLine = prefix
	}
}

func WithTextAddSuffixInLine(suffix string) lib.OutputOption {
	return func(t lib.OutputConverter) {
		t.(*text_out).AddSuffixInLine = suffix
	}
}

func NewTextOutFromBytes(oType string, oDesc string, action lib.Action, data []byte) (lib.OutputConverter, error) {
	var tmp struct {
		OutputDir  string     `json:"outputDir"`
		OutputExt  string     `json:"outputExtension"`
		Want       []string   `json:"wantedList"`
		Exclude    []string   `json:"excludedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`

		AddPrefixInLine string `json:"addPrefixInLine"`
		AddSuffixInLine string `json:"addSuffixInLine"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	// Set default output directory based on type
	if tmp.OutputDir == "" {
		switch oType {
		case TypeTextOut:
			tmp.OutputDir = filepath.Join("./", "output", "text")
		case TypeClashRuleSetClassicalOut:
			tmp.OutputDir = filepath.Join("./", "output", "clash", "classical")
		case TypeClashRuleSetIPCIDROut:
			tmp.OutputDir = filepath.Join("./", "output", "clash", "ipcidr")
		case TypeSurgeRuleSetOut:
			tmp.OutputDir = filepath.Join("./", "output", "surge")
		}
	}

	if tmp.OutputExt == "" {
		tmp.OutputExt = ".txt"
	}

	return NewTextOut(
		oType,
		oDesc,
		action,
		WithTextOutputDir(tmp.OutputDir),
		WithTextOutputExt(tmp.OutputExt),
		WithTextOutputWantedList(tmp.Want),
		WithTextOutputExcludedList(tmp.Exclude),
		WithTextOutputOnlyIPType(tmp.OnlyIPType),
		WithTextAddPrefixInLine(tmp.AddPrefixInLine),
		WithTextAddSuffixInLine(tmp.AddSuffixInLine),
	), nil
}

func (t *text_out) GetType() string {
	return t.Type
}

func (t *text_out) GetAction() lib.Action {
	return t.Action
}

func (t *text_out) GetDescription() string {
	return t.Description
}

func (t *text_out) Output(container lib.Container) error {
	for _, name := range t.filterAndSortList(container) {
		entry, found := container.GetEntry(name)
		if !found {
			log.Printf("❌ entry %s not found\n", name)
			continue
		}

		data, err := t.marshalBytes(entry)
		if err != nil {
			return err
		}

		filename := strings.ToLower(entry.GetName()) + t.OutputExt
		if err := t.writeFile(filename, data); err != nil {
			return err
		}
	}

	return nil
}

func (t *text_out) filterAndSortList(container lib.Container) []string {
	excludeMap := make(map[string]bool)
	for _, exclude := range t.Exclude {
		if exclude = strings.ToUpper(strings.TrimSpace(exclude)); exclude != "" {
			excludeMap[exclude] = true
		}
	}

	wantList := make([]string, 0, len(t.Want))
	for _, want := range t.Want {
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
