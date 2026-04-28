package mihomo

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/klauspost/compress/zstd"
	"go4.org/netipx"
)

const (
	TypeMRSOut = "mihomoMRS"
	DescMRSOut = "Convert data to mihomo MRS format"
)

var (
	defaultOutputDir = filepath.Join("./", "output", "mrs")
)

func init() {
	lib.RegisterOutputConfigCreator(TypeMRSOut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return NewMRSOutFromBytes(action, data)
	})
	lib.RegisterOutputConverter(TypeMRSOut, &mrs_out{
		Description: DescMRSOut,
	})
}

type mrs_out struct {
	Type        string
	Action      lib.Action
	Description string
	OutputDir   string
	Want        []string
	Exclude     []string
	OnlyIPType  lib.IPType
}

func NewMRSOut(action lib.Action, opts ...lib.OutputOption) lib.OutputConverter {
	m := &mrs_out{
		Type:        TypeMRSOut,
		Action:      action,
		Description: DescMRSOut,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(m)
		}
	}

	return m
}

func WithOutputDir(dir string) lib.OutputOption {
	return func(m lib.OutputConverter) {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			dir = defaultOutputDir
		}

		m.(*mrs_out).OutputDir = dir
	}
}

func WithOutputWantedList(lists []string) lib.OutputOption {
	return func(m lib.OutputConverter) {
		m.(*mrs_out).Want = lists
	}
}

func WithOutputExcludedList(lists []string) lib.OutputOption {
	return func(m lib.OutputConverter) {
		m.(*mrs_out).Exclude = lists
	}
}

func WithOutputOnlyIPType(onlyIPType lib.IPType) lib.OutputOption {
	return func(m lib.OutputConverter) {
		m.(*mrs_out).OnlyIPType = onlyIPType
	}
}

func NewMRSOutFromBytes(action lib.Action, data []byte) (lib.OutputConverter, error) {
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

	return NewMRSOut(
		action,
		WithOutputDir(tmp.OutputDir),
		WithOutputWantedList(tmp.Want),
		WithOutputExcludedList(tmp.Exclude),
		WithOutputOnlyIPType(tmp.OnlyIPType),
	), nil
}

func (m *mrs_out) GetType() string {
	return m.Type
}

func (m *mrs_out) GetAction() lib.Action {
	return m.Action
}

func (m *mrs_out) GetDescription() string {
	return m.Description
}

func (m *mrs_out) Output(container lib.Container) error {
	for _, name := range m.filterAndSortList(container) {
		entry, found := container.GetEntry(name)
		if !found {
			log.Printf("❌ entry %s not found\n", name)
			continue
		}

		if err := m.generate(entry); err != nil {
			return err
		}
	}

	return nil
}

func (m *mrs_out) filterAndSortList(container lib.Container) []string {
	excludeMap := make(map[string]bool)
	for _, exclude := range m.Exclude {
		if exclude = strings.ToUpper(strings.TrimSpace(exclude)); exclude != "" {
			excludeMap[exclude] = true
		}
	}

	wantList := make([]string, 0, len(m.Want))
	for _, want := range m.Want {
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

func (m *mrs_out) generate(entry *lib.Entry) error {
	ipRanges, err := entry.MarshalIPRange(lib.GetIgnoreIPType(m.OnlyIPType))
	if err != nil {
		return err
	}

	if len(ipRanges) == 0 {
		return fmt.Errorf("❌ [type %s | action %s] entry %s has no CIDR", m.Type, m.Action, entry.GetName())
	}

	filename := strings.ToLower(entry.GetName()) + ".mrs"
	if err := m.writeFile(filename, ipRanges); err != nil {
		return err
	}

	return nil
}

func (m *mrs_out) writeFile(filename string, ipRanges []netipx.IPRange) error {
	if err := os.MkdirAll(m.OutputDir, 0755); err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(m.OutputDir, filename))
	if err != nil {
		return err
	}
	defer f.Close()

	err = m.convertToMrs(ipRanges, f)
	if err != nil {
		return err
	}

	log.Printf("✅ [%s] %s --> %s", m.Type, filename, m.OutputDir)

	return nil
}

func (m *mrs_out) convertToMrs(ipRanges []netipx.IPRange, w io.Writer) (err error) {
	encoder, err := zstd.NewWriter(w)
	if err != nil {
		return err
	}
	defer encoder.Close()

	// header
	_, err = encoder.Write(mrsMagicBytes[:])
	if err != nil {
		return err
	}

	// behavior
	_, err = encoder.Write([]byte{1}) // RuleBehavior IPCIDR = 1
	if err != nil {
		return err
	}

	// count
	count := int64(len(ipRanges))
	err = binary.Write(encoder, binary.BigEndian, count)
	if err != nil {
		return err
	}

	// extra (reserved for future using)
	var extra []byte
	err = binary.Write(encoder, binary.BigEndian, int64(len(extra)))
	if err != nil {
		return err
	}
	_, err = encoder.Write(extra)
	if err != nil {
		return err
	}

	//
	// rule
	//
	// version
	_, err = encoder.Write([]byte{1})
	if err != nil {
		return err
	}

	// rule length
	err = binary.Write(encoder, binary.BigEndian, int64(len(ipRanges)))
	if err != nil {
		return err
	}

	for _, ipRange := range ipRanges {
		err = binary.Write(encoder, binary.BigEndian, ipRange.From().As16())
		if err != nil {
			return err
		}

		err = binary.Write(encoder, binary.BigEndian, ipRange.To().As16())
		if err != nil {
			return err
		}
	}

	return nil
}
