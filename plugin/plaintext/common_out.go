package plaintext

import (
	"bytes"
	"encoding/json"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

var (
	defaultOutputDirForTextOut                  = filepath.Join("./", "output", "text")
	defaultOutputDirForClashRuleSetClassicalOut = filepath.Join("./", "output", "clash", "classical")
	defaultOutputDirForClashRuleSetIPCIDROut    = filepath.Join("./", "output", "clash", "ipcidr")
	defaultOutputDirForSurgeRuleSetOut          = filepath.Join("./", "output", "surge")
)

type text_out struct {
	Type        string
	Action      lib.Action
	Description string
	OutputDir   string
	OutputExt   string
	Want        []string
	Exclude     []string
	OnlyIPType  lib.IPType

	AddPrefixInLine string
	AddSuffixInLine string
}

func NewTextOut(iType string, iDesc string, action lib.Action, opts ...lib.OutputOption) lib.OutputConverter {
	t := &text_out{
		Type:        iType,
		Action:      action,
		Description: iDesc,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(t)
		}
	}

	return t
}

func WithOutputDir(dir string, iType string) lib.OutputOption {
	return func(t lib.OutputConverter) {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			switch iType {
			case TypeTextOut:
				dir = defaultOutputDirForTextOut
			case TypeClashRuleSetClassicalOut:
				dir = defaultOutputDirForClashRuleSetClassicalOut
			case TypeClashRuleSetIPCIDROut:
				dir = defaultOutputDirForClashRuleSetIPCIDROut
			case TypeSurgeRuleSetOut:
				dir = defaultOutputDirForSurgeRuleSetOut
			}
		}

		t.(*text_out).OutputDir = dir
	}
}

func WithOutputExtension(ext string) lib.OutputOption {
	return func(t lib.OutputConverter) {
		ext = strings.TrimSpace(ext)
		if ext == "" {
			ext = ".txt"
		}

		t.(*text_out).OutputExt = ext
	}
}

func WithOutputWantedList(lists []string) lib.OutputOption {
	return func(t lib.OutputConverter) {
		t.(*text_out).Want = lists
	}
}

func WithOutputExcludedList(lists []string) lib.OutputOption {
	return func(t lib.OutputConverter) {
		t.(*text_out).Exclude = lists
	}
}

func WithOutputOnlyIPType(onlyIPType lib.IPType) lib.OutputOption {
	return func(t lib.OutputConverter) {
		t.(*text_out).OnlyIPType = onlyIPType
	}
}

func WithAddPrefixInLine(prefix string) lib.OutputOption {
	return func(t lib.OutputConverter) {
		t.(*text_out).AddPrefixInLine = prefix
	}
}

func WithAddSuffixInLine(suffix string) lib.OutputOption {
	return func(t lib.OutputConverter) {
		t.(*text_out).AddSuffixInLine = suffix
	}
}

func NewTextOutFromBytes(iType string, iDesc string, action lib.Action, data []byte) (lib.OutputConverter, error) {
	var tmp struct {
		OutputDir  string     `json:"outputDir"`
		OutputExt  string     `json:"outputExtension"`
		Want       []string   `json:"wantedList"`
		Exclude    []string   `json:"excludedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`

		AddPrefixInLine string `json:"addPrefixInLine"`
		AddSuffixInLine string `json:"addSuffixInLine"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	return NewTextOut(
		iType, iDesc, action,
		WithOutputDir(tmp.OutputDir, iType),
		WithOutputExtension(tmp.OutputExt),
		WithOutputWantedList(tmp.Want),
		WithOutputExcludedList(tmp.Exclude),
		WithOutputOnlyIPType(tmp.OnlyIPType),
		WithAddPrefixInLine(tmp.AddPrefixInLine),
		WithAddSuffixInLine(tmp.AddSuffixInLine),
	), nil
}

func (t *text_out) marshalBytes(entry *lib.Entry) ([]byte, error) {
	entryCidr, err := entry.MarshalText(lib.GetIgnoreIPType(t.OnlyIPType))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	switch t.Type {
	case TypeTextOut:
		err = t.marshalBytesForTextOut(&buf, entryCidr)
	case TypeClashRuleSetClassicalOut:
		err = t.marshalBytesForClashRuleSetClassicalOut(&buf, entryCidr)
	case TypeClashRuleSetIPCIDROut:
		err = t.marshalBytesForClashRuleSetIPCIDROut(&buf, entryCidr)
	case TypeSurgeRuleSetOut:
		err = t.marshalBytesForSurgeRuleSetOut(&buf, entryCidr)
	default:
		return nil, lib.ErrNotSupportedFormat
	}
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (t *text_out) marshalBytesForTextOut(buf *bytes.Buffer, entryCidr []string) error {
	for _, cidr := range entryCidr {
		if t.AddPrefixInLine != "" {
			buf.WriteString(t.AddPrefixInLine)
		}
		buf.WriteString(cidr)
		if t.AddSuffixInLine != "" {
			buf.WriteString(t.AddSuffixInLine)
		}
		buf.WriteString("\n")
	}
	return nil
}

func (t *text_out) marshalBytesForClashRuleSetClassicalOut(buf *bytes.Buffer, entryCidr []string) error {
	buf.WriteString("payload:\n")
	for _, cidr := range entryCidr {
		ip, _, err := net.ParseCIDR(cidr)
		if err != nil {
			return err
		}
		if ip.To4() != nil {
			buf.WriteString("  - IP-CIDR,")
		} else {
			buf.WriteString("  - IP-CIDR6,")
		}
		buf.WriteString(cidr)
		buf.WriteString("\n")
	}

	return nil
}

func (t *text_out) marshalBytesForClashRuleSetIPCIDROut(buf *bytes.Buffer, entryCidr []string) error {
	buf.WriteString("payload:\n")
	for _, cidr := range entryCidr {
		buf.WriteString("  - '")
		buf.WriteString(cidr)
		buf.WriteString("'\n")
	}

	return nil
}

func (t *text_out) marshalBytesForSurgeRuleSetOut(buf *bytes.Buffer, entryCidr []string) error {
	for _, cidr := range entryCidr {
		ip, _, err := net.ParseCIDR(cidr)
		if err != nil {
			return err
		}
		if ip.To4() != nil {
			buf.WriteString("IP-CIDR,")
		} else {
			buf.WriteString("IP-CIDR6,")
		}
		buf.WriteString(cidr)
		if t.AddSuffixInLine != "" {
			buf.WriteString(t.AddSuffixInLine)
		}
		buf.WriteString("\n")
	}

	return nil
}

func (t *text_out) writeFile(filename string, data []byte) error {
	if err := os.MkdirAll(t.OutputDir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(t.OutputDir, filename), data, 0644); err != nil {
		return err
	}

	log.Printf("✅ [%s] %s --> %s", t.Type, filename, t.OutputDir)

	return nil
}
