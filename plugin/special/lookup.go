package special

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	TypeLookup = "lookup"
	DescLookup = "Lookup specified IP or CIDR from various formats of data"
)

func init() {
	lib.RegisterOutputConfigCreator(TypeLookup, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return newLookup(action, data)
	})
	lib.RegisterOutputConverter(TypeLookup, &Lookup{
		Description: DescLookup,
	})
}

func newLookup(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
	var tmp struct {
		Search     string   `json:"search"`
		SearchList []string `json:"searchList"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	tmp.Search = strings.TrimSpace(tmp.Search)
	if tmp.Search == "" {
		return nil, fmt.Errorf("❌ [type %s | action %s] please specify an IP or a CIDR as search target", TypeLookup, action)
	}

	return &Lookup{
		Type:        TypeLookup,
		Action:      action,
		Description: DescLookup,
		Search:      tmp.Search,
		SearchList:  tmp.SearchList,
	}, nil
}

type Lookup struct {
	Type        string
	Action      lib.Action
	Description string
	Search      string
	SearchList  []string
}

func (l *Lookup) GetType() string {
	return l.Type
}

func (l *Lookup) GetAction() lib.Action {
	return l.Action
}

func (l *Lookup) GetDescription() string {
	return l.Description
}

func (l *Lookup) Output(container lib.Container) error {
	switch strings.Contains(l.Search, "/") {
	case true: // CIDR
		if _, err := netip.ParsePrefix(l.Search); err != nil {
			return errors.New("invalid IP or CIDR")
		}

	case false: // IP
		if _, err := netip.ParseAddr(l.Search); err != nil {
			return errors.New("invalid IP or CIDR")
		}
	}

	lists, found, _ := container.Lookup(l.Search, l.SearchList...)
	if found {
		slices.Sort(lists)
		fmt.Println(strings.ToLower(strings.Join(lists, ",")))
	} else {
		fmt.Println("false")
	}

	return nil
}
