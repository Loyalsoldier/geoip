package special

import (
	"encoding/json"
	"fmt"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	entryNamePrivate = "private"
	TypePrivate      = "private"
	DescPrivate      = "Convert LAN and private network CIDR to other formats"
)

// References:
// - https://github.com/Loyalsoldier/geoip/issues/234
// - https://github.com/Loyalsoldier/geoip/pull/412
// - https://www.iana.org/assignments/iana-ipv4-special-registry
// - https://www.iana.org/assignments/iana-ipv6-special-registry
var privateCIDRs = []string{
	"0.0.0.0/8",
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"192.31.196.0/24",
	"192.52.193.0/24",
	"192.88.99.0/24",
	"192.168.0.0/16",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"255.255.255.255/32",
	"::/127",
	"64:ff9b::/96",
	"64:ff9b:1::/48",
	"100::/64",
	"2001::/23",
	"2001:db8::/32",
	"2620:4f:8000::/48",
	"3fff::/20",
	"5f00::/16",
	"fc00::/7",
	"fe80::/10",
	"ff00::/8",
}

func init() {
	lib.RegisterInputConfigCreator(TypePrivate, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return NewPrivateFromBytes(action, data)
	})
	lib.RegisterInputConverter(TypePrivate, &private{
		Description: DescPrivate,
	})
}

type private struct {
	Type        string
	Action      lib.Action
	Description string
	OnlyIPType  lib.IPType
}

func NewPrivate(action lib.Action, opts ...lib.InputOption) (lib.InputConverter, error) {
	p := &private{
		Type:        TypePrivate,
		Action:      action,
		Description: DescPrivate,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}

	return p, nil
}

func WithPrivateOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(p lib.InputConverter) {
		p.(*private).OnlyIPType = onlyIPType
	}
}

func NewPrivateFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
	var tmp struct {
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, fmt.Errorf("❌ [type %s | action %s] %w", TypePrivate, action, err)
		}
	}

	return NewPrivate(
		action,
		WithPrivateOnlyIPType(tmp.OnlyIPType),
	)
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
			return nil, fmt.Errorf("❌ [type %s | action %s] %w", p.Type, p.Action, err)
		}
	}

	ignoreIPType := lib.GetIgnoreIPType(p.OnlyIPType)

	switch p.Action {
	case lib.ActionAdd:
		if err := container.Add(entry, ignoreIPType); err != nil {
			return nil, fmt.Errorf("❌ [type %s | action %s] %w", p.Type, p.Action, err)
		}
	case lib.ActionRemove:
		if err := container.Remove(entry, lib.CaseRemovePrefix, ignoreIPType); err != nil {
			return nil, fmt.Errorf("❌ [type %s | action %s] %w", p.Type, p.Action, err)
		}
	default:
		return nil, fmt.Errorf("❌ [type %s | action %s] %w", p.Type, p.Action, lib.ErrUnknownAction)
	}

	return container, nil
}
