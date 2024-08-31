package special

import (
	"bufio"
	"encoding/json"
	"fmt"
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

	if tmp.Name == "" {
		return nil, fmt.Errorf("❌ [type %s | action %s] missing name", typeStdin, action)
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

		if err := entry.AddPrefix(line); err != nil {
			continue
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

	switch s.Action {
	case lib.ActionAdd:
		if err := container.Add(entry, ignoreIPType); err != nil {
			return nil, err
		}
	case lib.ActionRemove:
		if err := container.Remove(entry, lib.CaseRemovePrefix, ignoreIPType); err != nil {
			return nil, err
		}
	default:
		return nil, lib.ErrUnknownAction
	}

	return container, nil
}
