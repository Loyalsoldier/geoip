package special

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
	lib.RegisterOutputConverter(TypeLookup, &Lookup{
		Description: DescLookup,
	})
}

func NewLookup(action lib.Action, opts ...lib.OutputOption) lib.OutputConverter {
	l := &Lookup{
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
		log.Fatalf("❌ [type %s | action %s] please specify an IP or a CIDR as search target", l.Type, l.Action)
	}
	var err error
	if strings.Contains(l.Search, "/") {
		_, err = netip.ParsePrefix(l.Search)
	} else {
		_, err = netip.ParseAddr(l.Search)
	}
	if err != nil {
		log.Fatalf("❌ [type %s | action %s] invalid IP or CIDR: %s", l.Type, l.Action, l.Search)
	}
	validateOutput(l, "")
	return l
}

func WithSearch(search string) lib.OutputOption {
	return func(l lib.OutputConverter) {
		l.(*Lookup).Search = strings.TrimSpace(search)
	}
}

func WithSearchList(lists []string) lib.OutputOption {
	return func(l lib.OutputConverter) {
		l.(*Lookup).SearchList = lists
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

	return NewLookup(action,
		WithSearch(tmp.Search),
		WithSearchList(tmp.SearchList),
	), nil
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

	lists, found, err := container.Lookup(l.Search, l.SearchList...)
	if err != nil {
		return err
	}

	if found {
		slices.Sort(lists)
		fmt.Println(strings.ToLower(strings.Join(lists, ",")))
	} else {
		fmt.Println("false")
	}

	return nil
}
