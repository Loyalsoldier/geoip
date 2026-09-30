package special

import (
	"log"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

func WithInputWantedList(lists []string) lib.InputOption {
	return func(s lib.InputConverter) {
		wantList := make(map[string]bool)
		for _, want := range lists {
			if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
				wantList[want] = true
			}
		}

		switch s := s.(type) {
		case *cutter:
			s.Want = wantList
		default:
			log.Fatalf("❌ WithInputWantedList is not supported by %T", s)
		}
	}
}

func WithInputOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(s lib.InputConverter) {
		ipType := lib.IPType(strings.ToLower(strings.TrimSpace(string(onlyIPType))))
		switch s := s.(type) {
		case *stdin:
			s.OnlyIPType = ipType
		case *private:
			s.OnlyIPType = ipType
		case *cutter:
			s.OnlyIPType = ipType
		default:
			log.Fatalf("❌ WithInputOnlyIPType is not supported by %T", s)
		}
	}
}
