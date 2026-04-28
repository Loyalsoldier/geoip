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

type textOut struct {
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
	t := &textOut{
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

func WithTextOutOutputDir(iType, dir string) lib.OutputOption {
	return func(s lib.OutputConverter) {
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

		s.(*textOut).OutputDir = dir
	}
}

func WithTextOutOutputExt(ext string) lib.OutputOption {
	return func(s lib.OutputConverter) {
		ext = strings.TrimSpace(ext)
		if ext == "" {
			ext = ".txt"
		}

		s.(*textOut).OutputExt = ext
	}
}

func WithTextOutWantedList(lists []string) lib.OutputOption {
	return func(s lib.OutputConverter) {
		s.(*textOut).Want = lists
	}
}

func WithTextOutExcludedList(lists []string) lib.OutputOption {
	return func(s lib.OutputConverter) {
		s.(*textOut).Exclude = lists
	}
}

func WithTextOutOnlyIPType(onlyIPType lib.IPType) lib.OutputOption {
	return func(s lib.OutputConverter) {
		s.(*textOut).OnlyIPType = onlyIPType
	}
}

func WithTextOutAddPrefixInLine(prefix string) lib.OutputOption {
	return func(s lib.OutputConverter) {
		s.(*textOut).AddPrefixInLine = prefix
	}
}

func WithTextOutAddSuffixInLine(suffix string) lib.OutputOption {
	return func(s lib.OutputConverter) {
		s.(*textOut).AddSuffixInLine = suffix
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
		iType,
		iDesc,
		action,
		WithTextOutOutputDir(iType, tmp.OutputDir),
		WithTextOutOutputExt(tmp.OutputExt),
		WithTextOutWantedList(tmp.Want),
		WithTextOutExcludedList(tmp.Exclude),
		WithTextOutOnlyIPType(tmp.OnlyIPType),
		WithTextOutAddPrefixInLine(tmp.AddPrefixInLine),
		WithTextOutAddSuffixInLine(tmp.AddSuffixInLine),
	), nil
}

func (t *textOut) marshalBytes(entry *lib.Entry) ([]byte, error) {
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

func (t *textOut) marshalBytesForTextOut(buf *bytes.Buffer, entryCidr []string) error {
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

func (t *textOut) marshalBytesForClashRuleSetClassicalOut(buf *bytes.Buffer, entryCidr []string) error {
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

func (t *textOut) marshalBytesForClashRuleSetIPCIDROut(buf *bytes.Buffer, entryCidr []string) error {
	buf.WriteString("payload:\n")
	for _, cidr := range entryCidr {
		buf.WriteString("  - '")
		buf.WriteString(cidr)
		buf.WriteString("'\n")
	}

	return nil
}

func (t *textOut) marshalBytesForSurgeRuleSetOut(buf *bytes.Buffer, entryCidr []string) error {
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

func (t *textOut) writeFile(filename string, data []byte) error {
	if err := os.MkdirAll(t.OutputDir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(t.OutputDir, filename), data, 0644); err != nil {
		return err
	}

	log.Printf("✅ [%s] %s --> %s", t.Type, filename, t.OutputDir)

	return nil
}
