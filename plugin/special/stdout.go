package special

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	typeStdout = "stdout"
	descStdout = "Convert data to plaintext CIDR format and output to standard output"
)

func init() {
	lib.RegisterOutputConfigCreator(typeStdout, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return newStdout(action, data)
	})
	lib.RegisterOutputConverter(typeStdout, &stdout{
		Description: descStdout,
	})
}

func newStdout(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
	var tmp struct {
		Want       []string   `json:"wantedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	// Filter want list
	wantList := make(map[string]bool)
	for _, want := range tmp.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList[want] = true
		}
	}

	return &stdout{
		Type:        typeStdout,
		Action:      action,
		Description: descStdout,
		Want:        wantList,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

type stdout struct {
	Type        string
	Action      lib.Action
	Description string
	Want        map[string]bool
	OnlyIPType  lib.IPType
}

func (s *stdout) GetType() string {
	return s.Type
}

func (s *stdout) GetAction() lib.Action {
	return s.Action
}

func (s *stdout) GetDescription() string {
	return s.Description
}

func (s *stdout) Output(container lib.Container) error {
	for entry := range container.Loop() {
		if len(s.Want) > 0 && !s.Want[entry.GetName()] {
			continue
		}

		cidrList, err := s.generateCIDRList(entry)
		if err != nil {
			continue
		}
		for _, cidr := range cidrList {
			io.WriteString(os.Stdout, cidr+"\n")
		}
	}

	return nil
}

func (s *stdout) generateCIDRList(entry *lib.Entry) ([]string, error) {
	var entryList []string
	var err error
	switch s.OnlyIPType {
	case lib.IPv4:
		entryList, err = entry.MarshalText(lib.IgnoreIPv6)
	case lib.IPv6:
		entryList, err = entry.MarshalText(lib.IgnoreIPv4)
	default:
		entryList, err = entry.MarshalText()
	}
	if err != nil {
		return nil, err
	}

	if len(entryList) == 0 {
		return nil, errors.New("empty CIDR list")
	}

	return entryList, nil
}
