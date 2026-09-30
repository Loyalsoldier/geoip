package special

import (
	"log"

	"github.com/Loyalsoldier/geoip/lib"
)

func WithInputOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(i lib.InputConverter) {
		switch s := i.(type) {
		case *cutter:
			s.OnlyIPType = onlyIPType
		case *private:
			s.OnlyIPType = onlyIPType
		case *stdin:
			s.OnlyIPType = onlyIPType
		default:
			log.Fatalf("❌ [type %s | action %s] option WithInputOnlyIPType is not supported", i.GetType(), i.GetAction())
		}
	}
}
