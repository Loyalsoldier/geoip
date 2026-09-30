package special

import (
	"encoding/json"
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
	if c.Action != lib.ActionRemove {
		log.Fatalf("❌ [type %s] only supports `remove` action", c.Type)
	}
	if len(c.Want) == 0 {
		log.Fatalf("❌ [type %s] wantedList must be specified", c.Type)
	}
	validateInput(c, c.OnlyIPType)
	return c
}

func WithInputWantedList(lists []string) lib.InputOption {
	return func(c lib.InputConverter) {
		wantList := make(map[string]bool)
		for _, want := range lists {
			if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
				wantList[want] = true
			}
		}
		c.(*Cutter).Want = wantList
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

	return NewCutter(action,
		WithInputWantedList(tmp.Want),
		WithInputOnlyIPType(tmp.OnlyIPType),
	), nil
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
