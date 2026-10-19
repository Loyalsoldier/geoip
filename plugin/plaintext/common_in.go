package plaintext

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/tidwall/gjson"
	"gopkg.in/yaml.v2"
)

type text_in struct {
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

func newTextIn(iType string, iDesc string, action lib.Action, opts ...lib.InputOption) (lib.InputConverter, error) {
	t := &text_in{
		Type:        iType,
		Action:      action,
		Description: iDesc,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(t)
		}
	}

	if t.Type != TypeTextIn && len(t.IPOrCIDR) > 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] ipOrCIDR is invalid for this input format", t.Type, t.Action)
	}

	if t.Type == TypeJSONIn && len(t.JSONPath) == 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] missing jsonPath", t.Type, t.Action)
	}

	if t.InputDir == "" {
		if t.Name == "" {
			return nil, fmt.Errorf("❌ [type %s | action %s] missing inputDir or name", t.Type, t.Action)
		}
		if t.URI == "" && len(t.IPOrCIDR) == 0 {
			return nil, fmt.Errorf("❌ [type %s | action %s] missing uri or ipOrCIDR", t.Type, t.Action)
		}
	} else if t.Name != "" || t.URI != "" || len(t.IPOrCIDR) > 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] inputDir is not allowed to be used with name or uri or ipOrCIDR", t.Type, t.Action)
	}

	return t, nil
}

func newTextInFromBytes(iType string, iDesc string, action lib.Action, data []byte) (lib.InputConverter, error) {
	var tmp struct {
		Name       string     `json:"name"`
		URI        string     `json:"uri"`
		IPOrCIDR   []string   `json:"ipOrCIDR"`
		InputDir   string     `json:"inputDir"`
		Want       []string   `json:"wantedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`

		JSONPath             []string `json:"jsonPath"`
		RemovePrefixesInLine []string `json:"removePrefixesInLine"`
		RemoveSuffixesInLine []string `json:"removeSuffixesInLine"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, fmt.Errorf("❌ [type %s | action %s] %w", iType, action, err)
		}
	}

	return newTextIn(
		iType,
		iDesc,
		action,
		WithNameAndURI(tmp.Name, tmp.URI),
		WithIPOrCIDR(tmp.IPOrCIDR),
		WithInputDir(tmp.InputDir),
		WithInputWantedList(tmp.Want),
		WithInputOnlyIPType(tmp.OnlyIPType),
		WithJSONPath(tmp.JSONPath),
		WithRemovePrefixesInLine(tmp.RemovePrefixesInLine),
		WithRemoveSuffixesInLine(tmp.RemoveSuffixesInLine),
	)
}

// WithNameAndURI sets the list name and the URI of the file to read.
// For text input, uri can be empty if ipOrCIDR is specified with WithIPOrCIDR.
func WithNameAndURI(name, uri string) lib.InputOption {
	return func(t lib.InputConverter) {
		t.(*text_in).Name = strings.TrimSpace(name)
		t.(*text_in).URI = strings.TrimSpace(uri)
	}
}

// WithIPOrCIDR is only valid for text input, and must be used with WithNameAndURI.
func WithIPOrCIDR(ipOrCIDR []string) lib.InputOption {
	return func(t lib.InputConverter) {
		t.(*text_in).IPOrCIDR = ipOrCIDR
	}
}

func WithInputDir(dir string) lib.InputOption {
	return func(t lib.InputConverter) {
		t.(*text_in).InputDir = strings.TrimSpace(dir)
	}
}

func WithInputWantedList(lists []string) lib.InputOption {
	return func(t lib.InputConverter) {
		wantList := make(map[string]bool)
		for _, want := range lists {
			if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
				wantList[want] = true
			}
		}

		t.(*text_in).Want = wantList
	}
}

func WithInputOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(t lib.InputConverter) {
		t.(*text_in).OnlyIPType = onlyIPType
	}
}

// WithJSONPath is required for JSON input.
func WithJSONPath(jsonPath []string) lib.InputOption {
	return func(t lib.InputConverter) {
		t.(*text_in).JSONPath = jsonPath
	}
}

func WithRemovePrefixesInLine(prefixes []string) lib.InputOption {
	return func(t lib.InputConverter) {
		t.(*text_in).RemovePrefixesInLine = prefixes
	}
}

func WithRemoveSuffixesInLine(suffixes []string) lib.InputOption {
	return func(t lib.InputConverter) {
		t.(*text_in).RemoveSuffixesInLine = suffixes
	}
}

func (t *text_in) scanFile(reader io.Reader, entry *lib.Entry) error {
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

func (t *text_in) scanFileForTextIn(reader io.Reader, entry *lib.Entry) error {
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

func (t *text_in) readClashRuleSetYAMLFile(reader io.Reader) ([]string, error) {
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

func (t *text_in) scanFileForClashIPCIDRRuleSetIn(reader io.Reader, entry *lib.Entry) error {
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

func (t *text_in) scanFileForClashClassicalRuleSetIn(reader io.Reader, entry *lib.Entry) error {
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

func (t *text_in) scanFileForSurgeRuleSetIn(reader io.Reader, entry *lib.Entry) error {
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

func (t *text_in) scanFileForJSONIn(reader io.Reader, entry *lib.Entry) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	if !gjson.ValidBytes(data) {
		return fmt.Errorf("invalid JSON data")
	}

	// JSON Path syntax:
	// https://github.com/tidwall/gjson/blob/master/SYNTAX.md
	for _, path := range t.JSONPath {
		path = strings.TrimSpace(path)

		result := gjson.GetBytes(data, path)
		if err := t.processJSONResult(result, entry); err != nil {
			return fmt.Errorf("failed to process JSON: %v", err)
		}
	}

	return nil
}

func (t *text_in) processJSONResult(result gjson.Result, entry *lib.Entry) error {
	switch {
	case !result.Exists():
		return fmt.Errorf("invalid IP address or CIDR (value not exist), please check your specified JSON path or JSON source")

	case result.Type == gjson.String:
		cidr := strings.TrimSpace(result.String())
		if cidr == "" {
			return fmt.Errorf("empty string, please check your specified JSON path or JSON source")
		}
		if err := entry.AddPrefix(cidr); err != nil {
			return err
		}

	case result.IsArray():
		for _, item := range result.Array() {
			if err := t.processJSONResult(item, entry); err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf("invalid IP address or CIDR, please check your specified JSON path or JSON source")
	}

	return nil
}
