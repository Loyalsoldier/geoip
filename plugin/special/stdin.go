package special

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	TypeStdin = "stdin"
	DescStdin = "Accept plaintext IP & CIDR from standard input, separated by newline"
)

func init() {
	lib.RegisterInputConfigCreator(TypeStdin, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return NewStdinFromBytes(action, data)
	})
	lib.RegisterInputConverter(TypeStdin, &stdin{
		Description: DescStdin,
	})
}

type stdin struct {
	Type        string
	Action      lib.Action
	Description string
	Name        string
	OnlyIPType  lib.IPType
}

func NewStdin(action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	s := &stdin{
		Type:        TypeStdin,
		Action:      action,
		Description: DescStdin,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}

	return s
}

func WithStdinName(name string) lib.InputOption {
	return func(s lib.InputConverter) {
		name = strings.TrimSpace(name)
		if name == "" {
			log.Fatalf("❌ [type %s | action %s] missing name", TypeStdin, s.(*stdin).Action)
		}

		s.(*stdin).Name = name
	}
}

func WithStdinOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(s lib.InputConverter) {
		s.(*stdin).OnlyIPType = onlyIPType
	}
}

func NewStdinFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
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
		return nil, fmt.Errorf("❌ [type %s | action %s] missing name", TypeStdin, action)
	}

	return NewStdin(
		action,
		WithStdinName(tmp.Name),
		WithStdinOnlyIPType(tmp.OnlyIPType),
	), nil
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

	ignoreIPType := lib.GetIgnoreIPType(s.OnlyIPType)

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
