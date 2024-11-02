package plaintext

import (
	"encoding/json"
	"log"
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
		return newTextOut(TypeTextOut, DescTextOut, action, data)
	})
	lib.RegisterOutputConverter(TypeTextOut, &TextOut{
		Description: DescTextOut,
	})
}

func (t *TextOut) GetType() string {
	return t.Type
}

func (t *TextOut) GetAction() lib.Action {
	return t.Action
}

func (t *TextOut) GetDescription() string {
	return t.Description
}

func (t *TextOut) Output(container lib.Container) error {
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

func (t *TextOut) filterAndSortList(container lib.Container) []string {
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
