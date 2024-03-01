package special

import (
	"encoding/json"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	entryNameReserved = "Reserved"
	typeReserved      = "Reserved"
	descReserved      = "Convert LAN and Reserved network CIDR to other formats"
)

var ReservedCIDRs = []string{
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
	lib.RegisterInputConfigCreator(typeReserved, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newReserved(action, data)
	})
	lib.RegisterInputConverter(typeReserved, &Reserved{
		Description: descReserved,
	})
}

func newReserved(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
	return &Reserved{
		Type:        typeReserved,
		Action:      action,
		Description: descReserved,
	}, nil
}

type Reserved struct {
	Type        string
	Action      lib.Action
	Description string
}

func (p *Reserved) GetType() string {
	return p.Type
}

func (p *Reserved) GetAction() lib.Action {
	return p.Action
}

func (p *Reserved) GetDescription() string {
	return p.Description
}

func (p *Reserved) Input(container lib.Container) (lib.Container, error) {
	entry := lib.NewEntry(entryNameReserved)
	for _, cidr := range ReservedCIDRs {
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
		container.Remove(entryNameReserved)
	default:
		return nil, lib.ErrUnknownAction
	}

	return container, nil
}
