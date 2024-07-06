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
		return newTest(action, data)
	})
	lib.RegisterInputConverter(typeTest, &test{
		Description: descTest,
	})
}

func newTest(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
	return &test{
		Type:        typeTest,
		Action:      action,
		Description: descTest,
	}, nil
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
