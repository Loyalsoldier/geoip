package special

import (
	"encoding/json"
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
		return NewLookupFromBytes(action, data)
	})
	lib.RegisterOutputConverter(TypeLookup, &lookup{
		Description: DescLookup,
	})
}

type lookup struct {
	Type        string
	Action      lib.Action
	Description string
	Search      string
	SearchList  []string
}

func NewLookup(action lib.Action, opts ...lib.OutputOption) (lib.OutputConverter, error) {
	l := &lookup{
		Type:        TypeLookup,
		Action:      action,
		Description: DescLookup,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(l)
		}
	}

	if l.Search == "" {
		return nil, fmt.Errorf("❌ [type %s | action %s] search must be specified", l.Type, l.Action)
	}

	return l, nil
}

func WithLookupSearch(search string) lib.OutputOption {
	return func(l lib.OutputConverter) {
		l.(*lookup).Search = strings.TrimSpace(search)
	}
}

func WithLookupSearchList(lists []string) lib.OutputOption {
	return func(l lib.OutputConverter) {
		l.(*lookup).SearchList = lists
	}
}

func NewLookupFromBytes(action lib.Action, data []byte) (lib.OutputConverter, error) {
	var tmp struct {
		Search     string   `json:"search"`
		SearchList []string `json:"searchList"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, fmt.Errorf("❌ [type %s | action %s] %w", TypeLookup, action, err)
		}
	}

	return NewLookup(
		action,
		WithLookupSearch(tmp.Search),
		WithLookupSearchList(tmp.SearchList),
	)
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
			return fmt.Errorf("❌ [type %s | action %s] invalid IP or CIDR", l.Type, l.Action)
		}

	case false: // IP
		if _, err := netip.ParseAddr(l.Search); err != nil {
			return fmt.Errorf("❌ [type %s | action %s] invalid IP or CIDR", l.Type, l.Action)
		}
	}

	lists, found, err := container.Lookup(l.Search, l.SearchList...)
	if err != nil {
		return fmt.Errorf("❌ [type %s | action %s] %w", l.Type, l.Action, err)
	}

	if found {
		slices.Sort(lists)
		fmt.Println(strings.ToLower(strings.Join(lists, ",")))
	} else {
		fmt.Println("false")
	}

	return nil
}
