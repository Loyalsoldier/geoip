package special

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	typeStdin = "stdin"
	descStdin = "Accept plaintext IP & CIDR from standard input, separated by newline"
)

func init() {
	lib.RegisterInputConfigCreator(typeStdin, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newStdin(action, data)
	})
	lib.RegisterInputConverter(typeStdin, &stdin{
		Description: descStdin,
	})
}

func newStdin(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
	var tmp struct {
		Name       string     `json:"name"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	return &stdin{
		Type:        typeStdin,
		Action:      action,
		Description: descStdin,
		Name:        tmp.Name,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

type stdin struct {
	Type        string
	Action      lib.Action
	Description string
	Name        string
	OnlyIPType  lib.IPType
}

func (s *stdin) GetType() string {
	return s.Type
}

func (s *stdin) GetAction() lib.Action {
	return s.Action
}

func (s *stdin) GetDescription() string {
	return s.Description
}

func (s *stdin) Input(container lib.Container) (lib.Container, error) {
	entry := lib.NewEntry(s.Name)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		line, _, _ = strings.Cut(line, "#")
		line, _, _ = strings.Cut(line, "//")
		line, _, _ = strings.Cut(line, "/*")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch s.Action {
		case lib.ActionAdd:
			if err := entry.AddPrefix(line); err != nil {
				continue
			}
		case lib.ActionRemove:
			if err := entry.RemovePrefix(line); err != nil {
				continue
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var ignoreIPType lib.IgnoreIPOption
	switch s.OnlyIPType {
	case lib.IPv4:
		ignoreIPType = lib.IgnoreIPv6
	case lib.IPv6:
		ignoreIPType = lib.IgnoreIPv4
	}

	if err := container.Add(entry, ignoreIPType); err != nil {
		return nil, err
	}

	return container, nil
}
