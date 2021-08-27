package plaintext

import (
	"bufio"
	"io"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"gopkg.in/yaml.v2"
)

type textIn struct {
	Type        string
	Action      lib.Action
	Description string
	Name        string
	URI         string
	InputDir    string
	OnlyIPType  lib.IPType
}

func (t *textIn) scanFile(reader io.Reader, entry *lib.Entry) error {
	var err error
	switch t.Type {
	case typeTextIn:
		err = t.scanFileForTextIn(reader, entry)
	case typeClashRuleSetClassicalIn:
		err = t.scanFileForClashClassicalRuleSetInAndSurgeIn(reader, entry)
	case typeClashRuleSetIPCIDRIn:
		err = t.scanFileForClashRuleSetIn(reader, entry)
	case typeSurgeRuleSetIn:
		err = t.scanFileForClashClassicalRuleSetInAndSurgeIn(reader, entry)
	default:
		return lib.ErrNotSupportedFormat
	}

	return err
}

func (t *textIn) scanFileForTextIn(reader io.Reader, entry *lib.Entry) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if err := entry.AddPrefix(line); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (t *textIn) scanFileForClashRuleSetIn(reader io.Reader, entry *lib.Entry) error {
	var payload struct {
		Payload []string `yaml:"payload"`
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, &payload); err != nil {
		return err
	}

	for _, cidrStr := range payload.Payload {
		if err := entry.AddPrefix(strings.TrimSpace(cidrStr)); err != nil {
			return err
		}
	}

	return nil
}

func (t *textIn) scanFileForClashClassicalRuleSetInAndSurgeIn(reader io.Reader, entry *lib.Entry) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "ip-cidr,"), strings.HasPrefix(line, "ip-cidr6,"):
			parts := strings.Split(line, ",")
			if len(parts) > 1 {
				if err := entry.AddPrefix(strings.TrimSpace(parts[1])); err != nil {
					return err
				}
			}
		default:
			continue
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
