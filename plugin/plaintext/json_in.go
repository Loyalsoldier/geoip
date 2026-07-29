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
		return NewJSONInFromBytes(action, data)
	})

	lib.RegisterInputConverter(TypeJSONIn, &text_in{
		Description: DescJSONIn,
	})
}

func NewJSONIn(action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	return newTextIn(TypeJSONIn, DescJSONIn, action, opts...)
}

func NewJSONInFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
	return newTextInFromBytes(TypeJSONIn, DescJSONIn, action, data)
}
