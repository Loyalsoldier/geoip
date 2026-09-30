package special

import (
	"log"

	"github.com/Loyalsoldier/geoip/lib"
)

func WithInputOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(c lib.InputConverter) {
		switch c := c.(type) {
		case *Cutter:
			c.OnlyIPType = onlyIPType
		case *Private:
			c.OnlyIPType = onlyIPType
		case *Stdin:
			c.OnlyIPType = onlyIPType
		default:
			log.Fatalf("❌ [type %s] onlyIPType is not supported", c.GetType())
		}
	}
}

func validateInput(c lib.InputConverter, onlyIPType lib.IPType) {
	if c.GetAction() != lib.ActionAdd && c.GetAction() != lib.ActionRemove {
		log.Fatalf("❌ [type %s | action %s] invalid input action", c.GetType(), c.GetAction())
	}
	if onlyIPType != "" && onlyIPType != lib.IPv4 && onlyIPType != lib.IPv6 {
		log.Fatalf("❌ [type %s | action %s] invalid onlyIPType: %s", c.GetType(), c.GetAction(), onlyIPType)
	}
}

func validateOutput(c lib.OutputConverter, onlyIPType lib.IPType) {
	if c.GetAction() != lib.ActionOutput {
		log.Fatalf("❌ [type %s | action %s] invalid output action", c.GetType(), c.GetAction())
	}
	if onlyIPType != "" && onlyIPType != lib.IPv4 && onlyIPType != lib.IPv6 {
		log.Fatalf("❌ [type %s | action %s] invalid onlyIPType: %s", c.GetType(), c.GetAction(), onlyIPType)
	}
}
