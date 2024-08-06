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
	typeSRSOut = "singboxSRS"
	descSRSOut = "Convert data to sing-box SRS format"
)

var (
	defaultOutputDir = filepath.Join("./", "output", "srs")
)

func init() {
	lib.RegisterOutputConfigCreator(typeSRSOut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return newSRSOut(action, data)
	})
	lib.RegisterOutputConverter(typeSRSOut, &srsOut{
		Description: descSRSOut,
	})
}

func newSRSOut(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
	var tmp struct {
		OutputDir  string     `json:"outputDir"`
		Want       []string   `json:"wantedList"`
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

	return &srsOut{
		Type:        typeSRSOut,
		Action:      action,
		Description: descSRSOut,
		OutputDir:   tmp.OutputDir,
		Want:        tmp.Want,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

type srsOut struct {
	Type        string
	Action      lib.Action
	Description string
	OutputDir   string
	Want        []string
	OnlyIPType  lib.IPType
}

func (s *srsOut) GetType() string {
	return s.Type
}

func (s *srsOut) GetAction() lib.Action {
	return s.Action
}

func (s *srsOut) GetDescription() string {
	return s.Description
}

func (s *srsOut) Output(container lib.Container) error {
	// Filter want list
	wantList := make([]string, 0, 50)
	for _, want := range s.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList = append(wantList, want)
		}
	}

	switch len(wantList) {
	case 0:
		list := make([]string, 0, 300)
		for entry := range container.Loop() {
			list = append(list, entry.GetName())
		}

		// Sort the list
		slices.Sort(list)

		for _, name := range list {
			entry, found := container.GetEntry(name)
			if !found {
				log.Printf("❌ entry %s not found", name)
				continue
			}
			if err := s.run(entry); err != nil {
				return err
			}
		}

	default:
		// Sort the list
		slices.Sort(wantList)

		for _, name := range wantList {
			entry, found := container.GetEntry(name)
			if !found {
				log.Printf("❌ entry %s not found", name)
				continue
			}

			if err := s.run(entry); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *srsOut) run(entry *lib.Entry) error {
	ruleset, err := s.generateRuleSet(entry)
	if err != nil {
		return err
	}

	filename := strings.ToLower(entry.GetName()) + ".srs"
	if err := s.writeFile(filename, ruleset); err != nil {
		return err
	}

	return nil
}

func (s *srsOut) generateRuleSet(entry *lib.Entry) (*option.PlainRuleSet, error) {
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

	return nil, fmt.Errorf("entry %s has no CIDR", entry.GetName())
}

func (s *srsOut) writeFile(filename string, ruleset *option.PlainRuleSet) error {
	if err := os.MkdirAll(s.OutputDir, 0755); err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(s.OutputDir, filename))
	if err != nil {
		return err
	}
	defer f.Close()

	err = srs.Write(f, *ruleset)
	if err != nil {
		return err
	}

	log.Printf("✅ [%s] %s --> %s", s.Type, filename, s.OutputDir)

	return nil
}
