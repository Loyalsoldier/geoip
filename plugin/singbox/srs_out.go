package singbox

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/sagernet/sing-box/common/srs"
	"github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

const (
	TypeSRSOut = "singboxSRS"
	DescSRSOut = "Convert data to sing-box SRS format"
)

var (
	defaultOutputDir = filepath.Join("./", "output", "srs")
)

func init() {
	lib.RegisterOutputConfigCreator(TypeSRSOut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return newSRSOut(action, data)
	})
	lib.RegisterOutputConverter(TypeSRSOut, &SRSOut{
		Description: DescSRSOut,
	})
}

func newSRSOut(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
	var tmp struct {
		OutputDir  string     `json:"outputDir"`
		Want       []string   `json:"wantedList"`
		Exclude    []string   `json:"excludedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	if tmp.OutputDir == "" {
		tmp.OutputDir = defaultOutputDir
	}

	return &SRSOut{
		Type:        TypeSRSOut,
		Action:      action,
		Description: DescSRSOut,
		OutputDir:   tmp.OutputDir,
		Want:        tmp.Want,
		Exclude:     tmp.Exclude,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

type SRSOut struct {
	Type        string
	Action      lib.Action
	Description string
	OutputDir   string
	Want        []string
	Exclude     []string
	OnlyIPType  lib.IPType
}

func (s *SRSOut) GetType() string {
	return s.Type
}

func (s *SRSOut) GetAction() lib.Action {
	return s.Action
}

func (s *SRSOut) GetDescription() string {
	return s.Description
}

func (s *SRSOut) Output(container lib.Container) error {
	for _, name := range s.filterAndSortList(container) {
		entry, found := container.GetEntry(name)
		if !found {
			log.Printf("❌ entry %s not found\n", name)
			continue
		}

		if err := s.generate(entry); err != nil {
			return err
		}
	}

	return nil
}

func (s *SRSOut) filterAndSortList(container lib.Container) []string {
	excludeMap := make(map[string]bool)
	for _, exclude := range s.Exclude {
		if exclude = strings.ToUpper(strings.TrimSpace(exclude)); exclude != "" {
			excludeMap[exclude] = true
		}
	}

	wantList := make([]string, 0, len(s.Want))
	for _, want := range s.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" && !excludeMap[want] {
			wantList = append(wantList, want)
		}
	}

	if len(wantList) > 0 {
		// Sort the list
		slices.Sort(wantList)
		return wantList
	}

	list := make([]string, 0, 300)
	for entry := range container.Loop() {
		name := entry.GetName()
		if excludeMap[name] {
			continue
		}
		list = append(list, name)
	}

	// Sort the list
	slices.Sort(list)

	return list
}

func (s *SRSOut) generate(entry *lib.Entry) error {
	ruleset, err := s.marshalRuleSet(entry)
	if err != nil {
		return err
	}

	filename := strings.ToLower(entry.GetName()) + ".srs"
	if err := s.writeFile(filename, ruleset); err != nil {
		return err
	}

	return nil
}

func (s *SRSOut) marshalRuleSet(entry *lib.Entry) (*option.PlainRuleSet, error) {
	var entryCidr []string
	var err error
	switch s.OnlyIPType {
	case lib.IPv4:
		entryCidr, err = entry.MarshalText(lib.IgnoreIPv6)
	case lib.IPv6:
		entryCidr, err = entry.MarshalText(lib.IgnoreIPv4)
	default:
		entryCidr, err = entry.MarshalText()
	}
	if err != nil {
		return nil, err
	}

	var headlessRule option.DefaultHeadlessRule
	headlessRule.IPCIDR = entryCidr

	var plainRuleSet option.PlainRuleSet
	plainRuleSet.Rules = []option.HeadlessRule{
		{
			Type:           constant.RuleTypeDefault,
			DefaultOptions: headlessRule,
		},
	}

	if len(headlessRule.IPCIDR) > 0 {
		return &plainRuleSet, nil
	}

	return nil, fmt.Errorf("❌ [type %s | action %s] entry %s has no CIDR", s.Type, s.Action, entry.GetName())
}

func (s *SRSOut) writeFile(filename string, ruleset *option.PlainRuleSet) error {
	if err := os.MkdirAll(s.OutputDir, 0755); err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(s.OutputDir, filename))
	if err != nil {
		return err
	}
	defer f.Close()

	err = srs.Write(f, *ruleset, constant.RuleSetVersion1)
	if err != nil {
		return err
	}

	log.Printf("✅ [%s] %s --> %s", s.Type, filename, s.OutputDir)

	return nil
}
