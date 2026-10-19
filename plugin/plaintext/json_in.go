package plaintext

import (
	"encoding/json"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	TypeJSONIn = "json"
	DescJSONIn = "Convert JSON data to other formats"
)

func init() {
	lib.RegisterInputConfigCreator(TypeJSONIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newTextInFromBytes(TypeJSONIn, DescJSONIn, action, data)
	})

	lib.RegisterInputConverter(TypeJSONIn, &textIn{
		Description: DescJSONIn,
	})
}

func NewJSONIn(action lib.Action, opts ...lib.InputOption) (lib.InputConverter, error) {
	return newTextIn(TypeJSONIn, DescJSONIn, action, opts...)
}
