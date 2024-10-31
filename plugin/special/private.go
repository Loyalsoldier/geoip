package special

import (
	"encoding/json"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	entryNamePrivate = "private"
	TypePrivate      = "private"
	DescPrivate      = "Convert LAN and private network CIDR to other formats"
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
	lib.RegisterInputConfigCreator(TypePrivate, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newPrivate(action, data)
	})
	lib.RegisterInputConverter(TypePrivate, &Private{
		Description: DescPrivate,
	})
}

func newPrivate(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
	var tmp struct {
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	return &Private{
		Type:        TypePrivate,
		Action:      action,
		Description: DescPrivate,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

type Private struct {
	Type        string
	Action      lib.Action
	Description string
	OnlyIPType  lib.IPType
}

func (p *Private) GetType() string {
	return p.Type
}

func (p *Private) GetAction() lib.Action {
	return p.Action
}

func (p *Private) GetDescription() string {
	return p.Description
}

func (p *Private) Input(container lib.Container) (lib.Container, error) {
	entry, found := container.GetEntry(entryNamePrivate)
	if !found {
		entry = lib.NewEntry(entryNamePrivate)
	}

	for _, cidr := range privateCIDRs {
		if err := entry.AddPrefix(cidr); err != nil {
			return nil, err
		}
	}

	var ignoreIPType lib.IgnoreIPOption
	switch p.OnlyIPType {
	case lib.IPv4:
		ignoreIPType = lib.IgnoreIPv6
	case lib.IPv6:
		ignoreIPType = lib.IgnoreIPv4
	}

	switch p.Action {
	case lib.ActionAdd:
		if err := container.Add(entry, ignoreIPType); err != nil {
			return nil, err
		}
	case lib.ActionRemove:
		if err := container.Remove(entry, lib.CaseRemovePrefix, ignoreIPType); err != nil {
			return nil, err
		}
	default:
		return nil, lib.ErrUnknownAction
	}

	return container, nil
}
