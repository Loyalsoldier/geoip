package special

import (
	"encoding/json"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	entryNamePrivate = "private"
	typePrivate      = "private"
	descPrivate      = "Convert LAN and private network CIDR to other formats"
)

var privateCIDRs = []string{
	"0.0.0.0/8",
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"192.88.99.0/24",
	"192.168.0.0/16",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"255.255.255.255/32",
	"::/128",
	"::1/128",
	"fc00::/7",
	"ff00::/8",
	"fe80::/10",
}

func init() {
	lib.RegisterInputConfigCreator(typePrivate, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newPrivate(action, data)
	})
	lib.RegisterInputConverter(typePrivate, &private{
		Description: descPrivate,
	})
}

func newPrivate(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
	return &private{
		Type:        typePrivate,
		Action:      action,
		Description: descPrivate,
	}, nil
}

type private struct {
	Type        string
	Action      lib.Action
	Description string
}

func (p *private) GetType() string {
	return p.Type
}

func (p *private) GetAction() lib.Action {
	return p.Action
}

func (p *private) GetDescription() string {
	return p.Description
}

func (p *private) Input(container lib.Container) (lib.Container, error) {
	entry := lib.NewEntry(entryNamePrivate)
	for _, cidr := range privateCIDRs {
		if err := entry.AddPrefix(cidr); err != nil {
			return nil, err
		}
	}

	switch p.Action {
	case lib.ActionAdd:
		if err := container.Add(entry); err != nil {
			return nil, err
		}
	case lib.ActionRemove:
		container.Remove(entryNamePrivate)
	default:
		return nil, lib.ErrUnknownAction
	}

	return container, nil
}
