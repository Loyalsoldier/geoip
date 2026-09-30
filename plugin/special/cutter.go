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

	if c.Action != lib.ActionRemove {
		log.Fatalf("❌ [type %s] only supports `remove` action", c.Type)
	}

	if len(c.Want) == 0 {
		log.Fatalf("❌ [type %s] wantedList must be specified", c.Type)
	}

	return c
}

// WithInputWantedList sets the wanted list for cutter
func WithInputWantedList(lists []string) lib.InputOption {
	return func(i lib.InputConverter) {
		c, ok := i.(*cutter)
		if !ok {
			return
		}

		wantList := make(map[string]bool)
		for _, want := range lists {
			if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
				wantList[want] = true
			}
		}

		c.Want = wantList
	}
}

// WithInputOnlyIPType sets the only IP type for cutter, private and stdin
func WithInputOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(i lib.InputConverter) {
		switch c := i.(type) {
		case *cutter:
			c.OnlyIPType = onlyIPType
		case *private:
			c.OnlyIPType = onlyIPType
		case *stdin:
			c.OnlyIPType = onlyIPType
		}
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

	return NewCutter(
		action,
		WithInputWantedList(tmp.Want),
		WithInputOnlyIPType(tmp.OnlyIPType),
	), nil
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
