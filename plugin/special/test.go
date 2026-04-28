package special

import (
	"encoding/json"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	entryNameTest = "test"
	typeTest      = "test"
	descTest      = "Convert specific CIDR to other formats (for test only)"
)

var testCIDRs = []string{
	"127.0.0.0/8",
}

func init() {
	lib.RegisterInputConfigCreator(typeTest, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return NewTestFromBytes(action, data)
	})
	lib.RegisterInputConverter(typeTest, &test{
		Description: descTest,
	})
}

func NewTest(action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	t := &test{
		Type:        typeTest,
		Action:      action,
		Description: descTest,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(t)
		}
	}

	return t
}

func NewTestFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
	return NewTest(action), nil
}

type test struct {
	Type        string
	Action      lib.Action
	Description string
}

func (t *test) GetType() string {
	return t.Type
}

func (t *test) GetAction() lib.Action {
	return t.Action
}

func (t *test) GetDescription() string {
	return t.Description
}

func (t *test) Input(container lib.Container) (lib.Container, error) {
	entry := lib.NewEntry(entryNameTest)
	for _, cidr := range testCIDRs {
		if err := entry.AddPrefix(cidr); err != nil {
			return nil, err
		}
	}

	switch t.Action {
	case lib.ActionAdd:
		if err := container.Add(entry); err != nil {
			return nil, err
		}
	case lib.ActionRemove:
		if err := container.Remove(entry, lib.CaseRemovePrefix); err != nil {
			return nil, err
		}
	default:
		return nil, lib.ErrUnknownAction
	}

	return container, nil
}
