package maxmind

import (
	"github.com/Loyalsoldier/geoip/lib"
)

type geoLite2CountryMMDBIn struct {
	Type        string
	Action      lib.Action
	Description string
	URI         string
	Want        map[string]bool
	OnlyIPType  lib.IPType
}
