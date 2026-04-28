package special

import (
	"encoding/json"
	"fmt"
	"log"
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
	lib.RegisterInputConverter(TypeCutter, &Cutter{
		Description: DescCutter,
	})
}

func NewCutter(action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	c := &Cutter{
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

func WithCutterWantedList(lists []string) lib.InputOption {
	return func(c lib.InputConverter) {
		wantList := make(map[string]bool)
		for _, want := range lists {
			if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
				wantList[want] = true
			}
		}
		if len(wantList) == 0 {
			log.Fatalf("❌ [type %s] wantedList must be specified", TypeCutter)
		}
		c.(*Cutter).Want = wantList
	}
}

func WithCutterOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(c lib.InputConverter) {
		c.(*Cutter).OnlyIPType = onlyIPType
	}
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

	return NewCutter(action, WithCutterWantedList(tmp.Want), WithCutterOnlyIPType(tmp.OnlyIPType)), nil
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
