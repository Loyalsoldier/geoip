package mihomo

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/klauspost/compress/zstd"
	"go4.org/netipx"
)

var mrsMagicBytes = [4]byte{'M', 'R', 'S', 1} // MRSv1

const (
	TypeMRSIn = "mihomoMRS"
	DescMRSIn = "Convert mihomo MRS data to other formats"
)

func init() {
	lib.RegisterInputConfigCreator(TypeMRSIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return NewMRSInFromBytes(action, data)
	})
	lib.RegisterInputConverter(TypeMRSIn, &mrs_in{
		Description: DescMRSIn,
	})
}

type mrs_in struct {
	Type        string
	Action      lib.Action
	Description string
	Name        string
	URI         string
	InputDir    string
	Want        map[string]bool
	OnlyIPType  lib.IPType
}

func NewMRSIn(action lib.Action, opts ...lib.InputOption) lib.InputConverter {
	m := &mrs_in{
		Type:        TypeMRSIn,
		Action:      action,
		Description: DescMRSIn,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(m)
		}
	}

	return m
}

func WithNameAndURI(name, uri string) lib.InputOption {
	return func(m lib.InputConverter) {
		name = strings.TrimSpace(name)
		uri = strings.TrimSpace(uri)
		if (name == "" || uri == "") && strings.TrimSpace(m.(*mrs_in).InputDir) == "" {
			log.Fatalf("❌ [type %s | action %s] missing name or uri or inputDir", TypeMRSIn, m.(*mrs_in).Action)
		}

		m.(*mrs_in).Name = name
		m.(*mrs_in).URI = uri
	}
}

func WithInputDir(dir string) lib.InputOption {
	return func(m lib.InputConverter) {
		dir = strings.TrimSpace(dir)
		if dir == "" && (strings.TrimSpace(m.(*mrs_in).Name) == "" || strings.TrimSpace(m.(*mrs_in).URI) == "") {
			log.Fatalf("❌ [type %s | action %s] missing name or uri or inputDir", TypeMRSIn, m.(*mrs_in).Action)
		}

		m.(*mrs_in).InputDir = dir
	}
}

func WithInputWantedList(lists []string) lib.InputOption {
	return func(m lib.InputConverter) {
		wantList := make(map[string]bool)
		for _, want := range lists {
			if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
				wantList[want] = true
			}
		}

		m.(*mrs_in).Want = wantList
	}
}

func WithInputOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(m lib.InputConverter) {
		m.(*mrs_in).OnlyIPType = onlyIPType
	}
}

func NewMRSInFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
	var tmp struct {
		Name       string     `json:"name"`
		URI        string     `json:"uri"`
		InputDir   string     `json:"inputDir"`
		Want       []string   `json:"wantedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	return NewMRSIn(
		action,
		WithNameAndURI(tmp.Name, tmp.URI),
		WithInputDir(tmp.InputDir),
		WithInputWantedList(tmp.Want),
		WithInputOnlyIPType(tmp.OnlyIPType),
	), nil
}

func (m *mrs_in) GetType() string {
	return m.Type
}

func (m *mrs_in) GetAction() lib.Action {
	return m.Action
}

func (m *mrs_in) GetDescription() string {
	return m.Description
}

func (m *mrs_in) Input(container lib.Container) (lib.Container, error) {
	entries := make(map[string]*lib.Entry)
	var err error

	switch {
	case m.InputDir != "":
		err = m.walkDir(m.InputDir, entries)
	case m.Name != "" && m.URI != "":
		switch {
		case strings.HasPrefix(strings.ToLower(m.URI), "http://"), strings.HasPrefix(strings.ToLower(m.URI), "https://"):
			err = m.walkRemoteFile(m.URI, m.Name, entries)
		default:
			err = m.walkLocalFile(m.URI, m.Name, entries)
		}
	default:
		return nil, fmt.Errorf("❌ [type %s | action %s] config missing argument inputDir or name or uri", m.Type, m.Action)
	}

	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] no entry is generated", m.Type, m.Action)
	}

	ignoreIPType := lib.GetIgnoreIPType(m.OnlyIPType)

	for _, entry := range entries {
		switch m.Action {
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
	}

	return container, nil
}

func (m *mrs_in) walkDir(dir string, entries map[string]*lib.Entry) error {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		if err := m.walkLocalFile(path, "", entries); err != nil {
			return err
		}

		return nil
	})

	return err
}

func (m *mrs_in) walkLocalFile(path, name string, entries map[string]*lib.Entry) error {
	entryName := ""
	name = strings.TrimSpace(name)
	if name != "" {
		entryName = name
	} else {
		entryName = filepath.Base(path)

		// check filename
		if !regexp.MustCompile(`^[a-zA-Z0-9_.\-]+$`).MatchString(entryName) {
			return fmt.Errorf("❌ [type %s | action %s] filename %s cannot be entry name, please remove special characters in it", m.Type, m.Action, entryName)
		}

		// remove file extension but not hidden files of which filename starts with "."
		dotIndex := strings.LastIndex(entryName, ".")
		if dotIndex > 0 {
			entryName = entryName[:dotIndex]
		}
	}

	entryName = strings.ToUpper(entryName)
	if _, found := entries[entryName]; found {
		return fmt.Errorf("❌ [type %s | action %s] found duplicated list %s", m.Type, m.Action, entryName)
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := m.generateEntries(entryName, file, entries); err != nil {
		return err
	}

	return nil
}

func (m *mrs_in) walkRemoteFile(url, name string, entries map[string]*lib.Entry) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("❌ [type %s | action %s] failed to get remote file %s, http status code %d", m.Type, m.Action, url, resp.StatusCode)
	}

	if err := m.generateEntries(name, resp.Body, entries); err != nil {
		return err
	}

	return nil
}

func (m *mrs_in) generateEntries(name string, reader io.Reader, entries map[string]*lib.Entry) error {
	name = strings.ToUpper(name)

	if len(m.Want) > 0 && !m.Want[name] {
		return nil
	}

	entry, found := entries[name]
	if !found {
		entry = lib.NewEntry(name)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	err = m.parseMRS(data, entry)
	if err != nil {
		return err
	}

	entries[name] = entry
	return nil
}

func (m *mrs_in) parseMRS(data []byte, entry *lib.Entry) error {
	reader, err := zstd.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer reader.Close()

	// header
	var header [4]byte
	_, err = io.ReadFull(reader, header[:])
	if err != nil {
		return err
	}
	if header != mrsMagicBytes {
		return fmt.Errorf("invalid MRS format")
	}

	// behavior
	var behavior [1]byte
	_, err = io.ReadFull(reader, behavior[:])
	if err != nil {
		return err
	}
	if behavior[0] != byte(1) { // RuleBehavior IPCIDR = 1
		return fmt.Errorf("invalid MRS IPCIDR data")
	}

	// count
	var count int64
	err = binary.Read(reader, binary.BigEndian, &count)
	if err != nil {
		return err
	}

	// extra (reserved for future using)
	var length int64
	err = binary.Read(reader, binary.BigEndian, &length)
	if err != nil {
		return err
	}
	if length < 0 {
		return fmt.Errorf("invalid MRS extra length")
	}
	if length > 0 {
		extra := make([]byte, length)
		_, err = io.ReadFull(reader, extra)
		if err != nil {
			return err
		}
	}

	//
	// rules
	//
	// version
	version := make([]byte, 1)
	_, err = io.ReadFull(reader, version)
	if err != nil {
		return err
	}
	if version[0] != 1 {
		return fmt.Errorf("invalid MRS rule version")
	}

	// rule length
	var ruleLength int64
	err = binary.Read(reader, binary.BigEndian, &ruleLength)
	if err != nil {
		return err
	}
	if ruleLength < 1 {
		return fmt.Errorf("invalid MRS rule length")
	}

	for i := int64(0); i < ruleLength; i++ {
		var a16 [16]byte
		err = binary.Read(reader, binary.BigEndian, &a16)
		if err != nil {
			return err
		}
		from := netip.AddrFrom16(a16).Unmap()

		err = binary.Read(reader, binary.BigEndian, &a16)
		if err != nil {
			return err
		}
		to := netip.AddrFrom16(a16).Unmap()

		iprange := netipx.IPRangeFrom(from, to)
		for _, prefix := range iprange.Prefixes() {
			if err := entry.AddPrefix(prefix); err != nil {
				return err
			}
		}
	}

	return nil
}
