package special

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	TypeCutter = "cutter"
	DescCutter = "Remove data from previous steps"
)

func init() {
	lib.RegisterInputConfigCreator(TypeCutter, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newCutter(action, data)
	})
	lib.RegisterInputConverter(TypeCutter, &Cutter{
		Description: DescCutter,
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
		return nil, fmt.Errorf("❌ [type %s] only supports `remove` action", TypeCutter)
	}

	// Filter want list
	wantList := make(map[string]bool)
	for _, want := range tmp.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList[want] = true
		}
	}

	if len(wantList) == 0 {
		return nil, fmt.Errorf("❌ [type %s] wantedList must be specified", TypeCutter)
	}

	return &Cutter{
		Type:        TypeCutter,
		Action:      action,
		Description: DescCutter,
		Want:        wantList,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

type Cutter struct {
	Type        string
	Action      lib.Action
	Description string
	Want        map[string]bool
	OnlyIPType  lib.IPType
}

func (c *Cutter) GetType() string {
	return c.Type
}

func (c *Cutter) GetAction() lib.Action {
	return c.Action
}

func (c *Cutter) GetDescription() string {
	return c.Description
}

func (c *Cutter) Input(container lib.Container) (lib.Container, error) {
	var ignoreIPType lib.IgnoreIPOption
	switch c.OnlyIPType {
	case lib.IPv4:
		ignoreIPType = lib.IgnoreIPv6
	case lib.IPv6:
		ignoreIPType = lib.IgnoreIPv4
	}

	for entry := range container.Loop() {
		if len(c.Want) > 0 && !c.Want[entry.GetName()] {
			continue
		}

		if err := container.Remove(entry, lib.CaseRemoveEntry, ignoreIPType); err != nil {
			return nil, err
		}
	}

	return container, nil
}
