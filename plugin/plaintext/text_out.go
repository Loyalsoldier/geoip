package plaintext

import (
	"encoding/json"
	"log"
	"slices"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	typeTextOut = "text"
	descTextOut = "Convert data to plaintext CIDR format"
)

func init() {
	lib.RegisterOutputConfigCreator(typeTextOut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return newTextOut(typeTextOut, action, data)
	})
	lib.RegisterOutputConverter(typeTextOut, &textOut{
		Description: descTextOut,
	})
}

func (t *textOut) GetType() string {
	return t.Type
}

func (t *textOut) GetAction() lib.Action {
	return t.Action
}

func (t *textOut) GetDescription() string {
	return t.Description
}

func (t *textOut) Output(container lib.Container) error {
	// Filter want list
	wantList := make([]string, 0, 50)
	for _, want := range t.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList = append(wantList, want)
		}
	}

	switch len(wantList) {
	case 0:
		list := make([]string, 0, 300)
		for entry := range container.Loop() {
			list = append(list, entry.GetName())
		}

		// Sort the list
		slices.Sort(list)

		for _, name := range list {
			entry, found := container.GetEntry(name)
			if !found {
				log.Printf("❌ entry %s not found", name)
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

	default:
		// Sort the list
		slices.Sort(wantList)

		for _, name := range wantList {
			entry, found := container.GetEntry(name)
			if !found {
				log.Printf("❌ entry %s not found", name)
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
	}

	return nil
}
