package plaintext

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/tidwall/gjson"
	"gopkg.in/yaml.v2"
)

type TextIn struct {
	Type        string
	Action      lib.Action
	Description string
	Name        string
	URI         string
	IPOrCIDR    []string
	InputDir    string
	Want        map[string]bool
	OnlyIPType  lib.IPType

	JSONPath             []string
	RemovePrefixesInLine []string
	RemoveSuffixesInLine []string
}

func (t *TextIn) scanFile(reader io.Reader, entry *lib.Entry) error {
	var err error
	switch t.Type {
	case TypeTextIn:
		err = t.scanFileForTextIn(reader, entry)
	case TypeJSONIn:
		err = t.scanFileForJSONIn(reader, entry)
	case TypeClashRuleSetClassicalIn:
		err = t.scanFileForClashClassicalRuleSetIn(reader, entry)
	case TypeClashRuleSetIPCIDRIn:
		err = t.scanFileForClashIPCIDRRuleSetIn(reader, entry)
	case TypeSurgeRuleSetIn:
		err = t.scanFileForSurgeRuleSetIn(reader, entry)
	default:
		return lib.ErrNotSupportedFormat
	}

	return err
}

func (t *TextIn) scanFileForTextIn(reader io.Reader, entry *lib.Entry) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		line, _, _ = strings.Cut(line, "#")
		line, _, _ = strings.Cut(line, "//")
		line, _, _ = strings.Cut(line, "/*")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		line = strings.ToLower(line)
		for _, prefix := range t.RemovePrefixesInLine {
			line = strings.TrimSpace(strings.TrimPrefix(line, strings.ToLower(strings.TrimSpace(prefix))))
		}
		for _, suffix := range t.RemoveSuffixesInLine {
			line = strings.TrimSpace(strings.TrimSuffix(line, strings.ToLower(strings.TrimSpace(suffix))))
		}
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

func (t *TextIn) readClashRuleSetYAMLFile(reader io.Reader) ([]string, error) {
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

func (t *TextIn) scanFileForClashIPCIDRRuleSetIn(reader io.Reader, entry *lib.Entry) error {
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

func (t *TextIn) scanFileForClashClassicalRuleSetIn(reader io.Reader, entry *lib.Entry) error {
	payload, err := t.readClashRuleSetYAMLFile(reader)
	if err != nil {
		return err
	}

	for _, line := range payload {
		line = strings.ToLower(strings.TrimSpace(line))
		if line == "" {
			continue
		}

		// Examples:
		// IP-CIDR,162.208.16.0/24
		// IP-CIDR6,2a0b:e40:1::/48
		// IP-CIDR,162.208.16.0/24,no-resolve
		// IP-CIDR6,2a0b:e40:1::/48,no-resolve
		if strings.HasPrefix(line, "ip-cidr,") || strings.HasPrefix(line, "ip-cidr6,") {
			parts := strings.Split(line, ",")
			if len(parts) < 2 {
				continue
			}
			line = strings.TrimSpace(parts[1])
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

func (t *TextIn) scanFileForSurgeRuleSetIn(reader io.Reader, entry *lib.Entry) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		line, _, _ = strings.Cut(line, "#")
		line, _, _ = strings.Cut(line, "//")
		line, _, _ = strings.Cut(line, "/*")
		line = strings.ToLower(strings.TrimSpace(line))
		if line == "" {
			continue
		}

		// Examples:
		// IP-CIDR,162.208.16.0/24
		// IP-CIDR6,2a0b:e40:1::/48
		// IP-CIDR,162.208.16.0/24,no-resolve
		// IP-CIDR6,2a0b:e40:1::/48,no-resolve
		if strings.HasPrefix(line, "ip-cidr,") || strings.HasPrefix(line, "ip-cidr6,") {
			parts := strings.Split(line, ",")
			if len(parts) < 2 {
				continue
			}
			line = strings.TrimSpace(parts[1])
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

func (t *TextIn) scanFileForJSONIn(reader io.Reader, entry *lib.Entry) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	if !gjson.ValidBytes(data) {
		return fmt.Errorf("❌ [type %s | action %s] invalid JSON data", t.Type, t.Action)
	}

	// JSON Path syntax:
	// https://github.com/tidwall/gjson/blob/master/SYNTAX.md
	for _, path := range t.JSONPath {
		path = strings.TrimSpace(path)

		result := gjson.GetBytes(data, path)
		for _, cidr := range result.Array() {
			if err := entry.AddPrefix(cidr.String()); err != nil {
				return err
			}
		}
	}

	return nil
}
