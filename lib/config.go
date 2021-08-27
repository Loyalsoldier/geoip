package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	inputConfigCreatorCache  = make(map[string]inputConfigCreator)
	outputConfigCreatorCache = make(map[string]outputConfigCreator)
)

type inputConfigCreator func(Action, json.RawMessage) (InputConverter, error)

type outputConfigCreator func(Action, json.RawMessage) (OutputConverter, error)

func RegisterInputConfigCreator(id string, fn inputConfigCreator) error {
	id = strings.ToLower(id)
	if _, found := inputConfigCreatorCache[id]; found {
		return errors.New("config creator has already been registered")
	}
	inputConfigCreatorCache[id] = fn
	return nil
}

func createInputConfig(id string, action Action, data json.RawMessage) (InputConverter, error) {
	id = strings.ToLower(id)
	fn, found := inputConfigCreatorCache[id]
	if !found {
		return nil, errors.New("unknown config type")
	}
	return fn(action, data)
}

func RegisterOutputConfigCreator(id string, fn outputConfigCreator) error {
	id = strings.ToLower(id)
	if _, found := outputConfigCreatorCache[id]; found {
		return errors.New("config creator has already been registered")
	}
	outputConfigCreatorCache[id] = fn
	return nil
}

func createOutputConfig(id string, action Action, data json.RawMessage) (OutputConverter, error) {
	id = strings.ToLower(id)
	fn, found := outputConfigCreatorCache[id]
	if !found {
		return nil, errors.New("unknown config type")
	}
	return fn(action, data)
}

type config struct {
	Input  []*inputConvConfig  `json:"input"`
	Output []*outputConvConfig `json:"output"`
}

type inputConvConfig struct {
	iType     string
	action    Action
	converter InputConverter
}

func (i *inputConvConfig) UnmarshalJSON(data []byte) error {
	var temp struct {
		Type   string          `json:"type"`
		Action Action          `json:"action"`
		Args   json.RawMessage `json:"args"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	if !ActionsRegistry[temp.Action] {
		return fmt.Errorf("invalid action %s in type %s", temp.Action, temp.Type)
	}

	config, err := createInputConfig(temp.Type, temp.Action, temp.Args)
	if err != nil {
		return err
	}

	i.iType = config.GetType()
	i.action = config.GetAction()
	i.converter = config

	return nil
}

type outputConvConfig struct {
	iType     string
	action    Action
	converter OutputConverter
}

func (i *outputConvConfig) UnmarshalJSON(data []byte) error {
	var temp struct {
		Type   string          `json:"type"`
		Action Action          `json:"action"`
		Args   json.RawMessage `json:"args"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	if temp.Action == "" {
		temp.Action = ActionOutput
	}

	if !ActionsRegistry[temp.Action] {
		return fmt.Errorf("invalid action %s in type %s", temp.Action, temp.Type)
	}

	config, err := createOutputConfig(temp.Type, temp.Action, temp.Args)
	if err != nil {
		return err
	}

	i.iType = config.GetType()
	i.action = config.GetAction()
	i.converter = config

	return nil
}
