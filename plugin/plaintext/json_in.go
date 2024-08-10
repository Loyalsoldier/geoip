package plaintext

import (
	"encoding/json"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	typeJSONIn = "json"
	descJSONIn = "Convert JSON data to other formats"
)

func init() {
	lib.RegisterInputConfigCreator(typeJSONIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newTextIn(typeJSONIn, action, data)
	})

	lib.RegisterInputConverter(typeJSONIn, &textIn{
		Description: descJSONIn,
	})
}
