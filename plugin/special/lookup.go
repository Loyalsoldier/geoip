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

func NewLookup(action lib.Action, opts ...lib.OutputOption) lib.OutputConverter {
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

	return l
}

func WithSearch(search string) lib.OutputOption {
	return func(l lib.OutputConverter) {
		l.(*lookup).Search = strings.TrimSpace(search)
	}
}

func WithSearchList(searchList []string) lib.OutputOption {
	return func(l lib.OutputConverter) {
		l.(*lookup).SearchList = searchList
	}
}

func NewLookupFromBytes(action lib.Action, data []byte) (lib.OutputConverter, error) {
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

	return NewLookup(
		action,
		WithSearch(tmp.Search),
		WithSearchList(tmp.SearchList),
	), nil
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
		slices.Sort(lists)
		fmt.Println(strings.ToLower(strings.Join(lists, ",")))
	} else {
		fmt.Println("false")
	}

	return nil
}
