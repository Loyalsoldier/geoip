package special

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	typeCutter = "cutter"
	descCutter = "Remove data from previous steps"
)

func init() {
	lib.RegisterInputConfigCreator(typeCutter, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newCutter(action, data)
	})
	lib.RegisterInputConverter(typeCutter, &cutter{
		Description: descCutter,
	})
}

func newCutter(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
	var tmp struct {
		Want       []string   `json:"wantedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	if action != lib.ActionRemove {
		return nil, fmt.Errorf("type %s only supports `remove` action", typeCutter)
	}

	return &cutter{
		Type:        typeCutter,
		Action:      action,
		Description: descCutter,
		Want:        tmp.Want,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

type cutter struct {
	Type        string
	Action      lib.Action
	Description string
	Want        []string
	OnlyIPType  lib.IPType
}

func (c *cutter) GetType() string {
	return c.Type
}

func (c *cutter) GetAction() lib.Action {
	return c.Action
}

func (c *cutter) GetDescription() string {
	return c.Description
}

func (c *cutter) Input(container lib.Container) (lib.Container, error) {
	var ignoreIPType lib.IgnoreIPOption
	switch c.OnlyIPType {
	case lib.IPv4:
		ignoreIPType = lib.IgnoreIPv6
	case lib.IPv6:
		ignoreIPType = lib.IgnoreIPv4
	}

	// Filter want list
	wantList := make(map[string]bool)
	for _, want := range c.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList[want] = true
		}
	}

	for entry := range container.Loop() {
		name := entry.GetName()
		if len(wantList) > 0 && !wantList[name] {
			continue
		}
		container.Remove(name, ignoreIPType)
	}

	return container, nil
}
