package special

import (
	"log"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

func WithName(name string) lib.InputOption {
	return func(s lib.InputConverter) {
		s.(*stdin).Name = strings.TrimSpace(name)
	}
}

func WithInputWantedList(lists []string) lib.InputOption {
	return func(c lib.InputConverter) {
		wantList := make(map[string]bool)
		for _, want := range lists {
			if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
				wantList[want] = true
			}
		}
		c.(*cutter).Want = wantList
	}
}

func WithInputOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(c lib.InputConverter) {
		switch c := c.(type) {
		case *stdin:
			c.OnlyIPType = onlyIPType
		case *cutter:
			c.OnlyIPType = onlyIPType
		case *private:
			c.OnlyIPType = onlyIPType
		}
	}
}

func WithOutputWantedList(lists []string) lib.OutputOption {
	return func(s lib.OutputConverter) {
		s.(*stdout).Want = lists
	}
}

func WithOutputExcludedList(lists []string) lib.OutputOption {
	return func(s lib.OutputConverter) {
		s.(*stdout).Exclude = lists
	}
}

func WithOutputOnlyIPType(onlyIPType lib.IPType) lib.OutputOption {
	return func(s lib.OutputConverter) {
		s.(*stdout).OnlyIPType = onlyIPType
	}
}

func WithSearch(search string) lib.OutputOption {
	return func(l lib.OutputConverter) {
		l.(*lookup).Search = strings.TrimSpace(search)
	}
}

func WithSearchList(lists []string) lib.OutputOption {
	return func(l lib.OutputConverter) {
		l.(*lookup).SearchList = lists
	}
}

func validateInputAction(typ string, action lib.Action) {
	if action != lib.ActionAdd && action != lib.ActionRemove {
		log.Fatalf("❌ [type %s | action %s] only supports `add` or `remove` action", typ, action)
	}
}

func validateOutputAction(typ string, action lib.Action) {
	if action != lib.ActionOutput {
		log.Fatalf("❌ [type %s | action %s] only supports `output` action", typ, action)
	}
}

func validateOnlyIPType(typ string, action lib.Action, onlyIPType lib.IPType) {
	switch onlyIPType {
	case "", lib.IPv4, lib.IPv6:
	default:
		log.Fatalf("❌ [type %s | action %s] invalid onlyIPType %q, must be ipv4 or ipv6", typ, action, onlyIPType)
	}
}
