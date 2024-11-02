package lib

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/tailscale/hujson"
)

type Instance interface {
	InitConfig(configFile string) error
	InitConfigFromBytes(content []byte) error
	AddInput(InputConverter)
	AddOutput(OutputConverter)
	ResetInput()
	ResetOutput()
	RunInput(Container) error
	RunOutput(Container) error
	Run() error
}

type instance struct {
	input  []InputConverter
	output []OutputConverter
}

func NewInstance() (Instance, error) {
	return &instance{
		input:  make([]InputConverter, 0),
		output: make([]OutputConverter, 0),
	}, nil
}

func (i *instance) InitConfig(configFile string) error {
	var content []byte
	var err error
	configFile = strings.TrimSpace(configFile)
	if strings.HasPrefix(strings.ToLower(configFile), "http://") || strings.HasPrefix(strings.ToLower(configFile), "https://") {
		content, err = GetRemoteURLContent(configFile)
	} else {
		content, err = os.ReadFile(configFile)
	}
	if err != nil {
		return err
	}

	return i.InitConfigFromBytes(content)
}

func (i *instance) InitConfigFromBytes(content []byte) error {
	config := new(config)

	// Support JSON with comments and trailing commas
	content, _ = hujson.Standardize(content)

	if err := json.Unmarshal(content, &config); err != nil {
		return err
	}

	for _, input := range config.Input {
		i.input = append(i.input, input.converter)
	}

	for _, output := range config.Output {
		i.output = append(i.output, output.converter)
	}

	return nil
}

func (i *instance) AddInput(ic InputConverter) {
	i.input = append(i.input, ic)
}

func (i *instance) AddOutput(oc OutputConverter) {
	i.output = append(i.output, oc)
}

func (i *instance) ResetInput() {
	i.input = make([]InputConverter, 0)
}

func (i *instance) ResetOutput() {
	i.output = make([]OutputConverter, 0)
}

func (i *instance) RunInput(container Container) error {
	var err error
	for _, ic := range i.input {
		container, err = ic.Input(container)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *instance) RunOutput(container Container) error {
	for _, oc := range i.output {
		if err := oc.Output(container); err != nil {
			return err
		}
	}

	return nil
}

func (i *instance) Run() error {
	if len(i.input) == 0 || len(i.output) == 0 {
		return errors.New("input type and output type must be specified")
	}

	container := NewContainer()

	if err := i.RunInput(container); err != nil {
		return err
	}

	if err := i.RunOutput(container); err != nil {
		return err
	}

	return nil
}
