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
		err = t.scanFileForClashClassicalRuleSetIn(reader, entry)
	case typeClashRuleSetIPCIDRIn:
		err = t.scanFileForClashIPCIDRRuleSetIn(reader, entry)
	case typeSurgeRuleSetIn:
		err = t.scanFileForSurgeRuleSetIn(reader, entry)
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
		line, _, _ = strings.Cut(line, "#")
		line, _, _ = strings.Cut(line, "//")
		line, _, _ = strings.Cut(line, "/*")
		line = strings.TrimSpace(line)
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

func (t *textIn) readClashRuleSetYAMLFile(reader io.Reader) ([]string, error) {
	var payload struct {
		Payload []string `yaml:"payload"`
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, &payload); err != nil {
		return nil, err
	}

	return payload.Payload, nil
}

func (t *textIn) scanFileForClashIPCIDRRuleSetIn(reader io.Reader, entry *lib.Entry) error {
	payload, err := t.readClashRuleSetYAMLFile(reader)
	if err != nil {
		return err
	}

	for _, cidrStr := range payload {
		cidrStr = strings.TrimSpace(cidrStr)
		if cidrStr == "" {
			continue
		}
		if err := entry.AddPrefix(cidrStr); err != nil {
			return err
		}
	}

	return nil
}

func (t *textIn) scanFileForClashClassicalRuleSetIn(reader io.Reader, entry *lib.Entry) error {
	payload, err := t.readClashRuleSetYAMLFile(reader)
	if err != nil {
		return err
	}

	for _, line := range payload {
		line = strings.ToLower(strings.TrimSpace(line))
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "ip-cidr,") || strings.HasPrefix(line, "ip-cidr6,") {
			_, line, _ = strings.Cut(line, ",")
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if err := entry.AddPrefix(line); err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *textIn) scanFileForSurgeRuleSetIn(reader io.Reader, entry *lib.Entry) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "ip-cidr,") || strings.HasPrefix(line, "ip-cidr6,") {
			line, _, _ = strings.Cut(line, "#")
			line, _, _ = strings.Cut(line, "//")
			line, _, _ = strings.Cut(line, "/*")
			_, line, _ = strings.Cut(line, ",")
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if err := entry.AddPrefix(line); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
