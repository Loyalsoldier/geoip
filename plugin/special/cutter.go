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
		return NewCutterFromBytes(action, data)
	})
	lib.RegisterInputConverter(TypeCutter, &cutter{
		Description: DescCutter,
	})
}

type cutter struct {
	Type        string
	Action      lib.Action
	Description string
	Want        map[string]bool
	OnlyIPType  lib.IPType
}

func NewCutter(action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	c := &cutter{
		Type:        TypeCutter,
		Action:      action,
		Description: DescCutter,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}

	return c
}

func NewCutterFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
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

	return NewCutter(
		action,
		WithCutterWantedList(wantList),
		WithCutterOnlyIPType(tmp.OnlyIPType),
	), nil
}

func WithCutterWantedList(wantList map[string]bool) lib.InputOption {
	return func(c lib.InputConverter) {
		c.(*cutter).Want = wantList
	}
}

func WithCutterOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(c lib.InputConverter) {
		c.(*cutter).OnlyIPType = onlyIPType
	}
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
	ignoreIPType := lib.GetIgnoreIPType(c.OnlyIPType)

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
