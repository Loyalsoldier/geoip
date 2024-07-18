package special

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	typeLookup = "lookup"
	descLookup = "Lookup specified IP or CIDR from various formats of data"
)

func init() {
	lib.RegisterOutputConfigCreator(typeLookup, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return newLookup(action, data)
	})
	lib.RegisterOutputConverter(typeLookup, &lookup{
		Description: descLookup,
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
		return nil, fmt.Errorf("type %s | action %s: please specify an IP or a CIDR as search target", typeLookup, action)
	}

	return &lookup{
		Type:        typeLookup,
		Action:      action,
		Description: descLookup,
		Search:      tmp.Search,
		SearchList:  tmp.SearchList,
	}, nil
}

type lookup struct {
	Type        string
	Action      lib.Action
	Description string
	Search      string
	SearchList  []string
}

func (l *lookup) GetType() string {
	return l.Type
}

func (l *lookup) GetAction() lib.Action {
	return l.Action
}

func (l *lookup) GetDescription() string {
	return l.Description
}

func (l *lookup) Output(container lib.Container) error {
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
		fmt.Println(strings.ToLower(strings.Join(lists, ",")))
	}

	return nil
}
