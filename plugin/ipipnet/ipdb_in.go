package ipipnet

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"os"
	"reflect"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/ipipdotnet/ipdb-go"
)

const (
	TypeIPDBIn = "ipipnetIPDB"
	DescIPDBIn = "Convert ipip.net IPDB to other formats"
)

const (
	defaultField    = "country_code"
	defaultLanguage = "CN"
)

func init() {
	lib.RegisterInputConfigCreator(TypeIPDBIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return NewIPDBInFromBytes(action, data)
	})
	lib.RegisterInputConverter(TypeIPDBIn, &ipdb_in{
		Description: DescIPDBIn,
	})
}

type ipdb_in struct {
	Type        string
	Action      lib.Action
	Description string
	URI         string
	Field       string
	Language    string
	Want        map[string]bool
	OnlyIPType  lib.IPType
}

func NewIPDBIn(action lib.Action, opts ...lib.InputOption) (lib.InputConverter, error) {
	i := &ipdb_in{
		Type:        TypeIPDBIn,
		Action:      action,
		Description: DescIPDBIn,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(i)
		}
	}

	if i.URI == "" {
		return nil, fmt.Errorf("❌ [type %s | action %s] uri must be specified", i.Type, i.Action)
	}

	if i.Field == "" {
		i.Field = defaultField
	}
	if cityInfoFieldIndex(i.Field) < 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] unsupported field: %s", i.Type, i.Action, i.Field)
	}

	if i.Language == "" {
		i.Language = defaultLanguage
	}

	return i, nil
}

func WithURI(uri string) lib.InputOption {
	return func(i lib.InputConverter) {
		i.(*ipdb_in).URI = strings.TrimSpace(uri)
	}
}

func WithField(field string) lib.InputOption {
	return func(i lib.InputConverter) {
		i.(*ipdb_in).Field = strings.ToLower(strings.TrimSpace(field))
	}
}

func WithLanguage(language string) lib.InputOption {
	return func(i lib.InputConverter) {
		i.(*ipdb_in).Language = strings.ToUpper(strings.TrimSpace(language))
	}
}

func WithInputWantedList(lists []string) lib.InputOption {
	return func(i lib.InputConverter) {
		wantList := make(map[string]bool)
		for _, want := range lists {
			if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
				wantList[want] = true
			}
		}

		i.(*ipdb_in).Want = wantList
	}
}

func WithInputOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(i lib.InputConverter) {
		i.(*ipdb_in).OnlyIPType = onlyIPType
	}
}

func NewIPDBInFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
	var tmp struct {
		URI        string     `json:"uri"`
		Field      string     `json:"field"`
		Language   string     `json:"language"`
		Want       []string   `json:"wantedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, fmt.Errorf("❌ [type %s | action %s] %w", TypeIPDBIn, action, err)
		}
	}

	return NewIPDBIn(
		action,
		WithURI(tmp.URI),
		WithField(tmp.Field),
		WithLanguage(tmp.Language),
		WithInputWantedList(tmp.Want),
		WithInputOnlyIPType(tmp.OnlyIPType),
	)
}

// cityInfoFieldIndex returns the index of the string field in ipdb.CityInfo
// whose json tag equals name, or -1 if not found.
func cityInfoFieldIndex(name string) int {
	t := reflect.TypeOf(ipdb.CityInfo{})
	for idx := 0; idx < t.NumField(); idx++ {
		f := t.Field(idx)
		if f.Type.Kind() == reflect.String && f.Tag.Get("json") == name {
			return idx
		}
	}
	return -1
}

func (i *ipdb_in) GetType() string {
	return i.Type
}

func (i *ipdb_in) GetAction() lib.Action {
	return i.Action
}

func (i *ipdb_in) GetDescription() string {
	return i.Description
}

func (i *ipdb_in) Input(container lib.Container) (lib.Container, error) {
	var content []byte
	var err error

	switch {
	case strings.HasPrefix(strings.ToLower(i.URI), "http://"), strings.HasPrefix(strings.ToLower(i.URI), "https://"):
		content, err = readRemoteFile(i.URI)
	default:
		content, err = os.ReadFile(i.URI)
	}
	if err != nil {
		return nil, fmt.Errorf("❌ [type %s | action %s] %w", i.Type, i.Action, err)
	}

	entries := make(map[string]*lib.Entry)
	if err := i.generateEntries(content, entries); err != nil {
		return nil, fmt.Errorf("❌ [type %s | action %s] %w", i.Type, i.Action, err)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] no entry is generated", i.Type, i.Action)
	}

	ignoreIPType := lib.GetIgnoreIPType(i.OnlyIPType)

	for _, entry := range entries {
		switch i.Action {
		case lib.ActionAdd:
			if err := container.Add(entry, ignoreIPType); err != nil {
				return nil, fmt.Errorf("❌ [type %s | action %s] %w", i.Type, i.Action, err)
			}
		case lib.ActionRemove:
			if err := container.Remove(entry, lib.CaseRemovePrefix, ignoreIPType); err != nil {
				return nil, fmt.Errorf("❌ [type %s | action %s] %w", i.Type, i.Action, err)
			}
		default:
			return nil, fmt.Errorf("❌ [type %s | action %s] %w", i.Type, i.Action, lib.ErrUnknownAction)
		}
	}

	return container, nil
}

func readRemoteFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get remote file %s, http status code %d", url, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// tree holds the raw binary search tree of an IPDB file. The ipdb-go library
// does not expose a way to iterate the tree, so it is walked here directly.
type tree struct {
	nodeCount int
	data      []byte
}

func newTree(content []byte) (*tree, error) {
	if len(content) < 4 {
		return nil, fmt.Errorf("invalid ipdb file: too small")
	}
	metaLength := int(binary.BigEndian.Uint32(content[0:4]))
	if len(content) < 4+metaLength {
		return nil, fmt.Errorf("invalid ipdb file: bad metadata length")
	}
	var meta struct {
		NodeCount int `json:"node_count"`
	}
	if err := json.Unmarshal(content[4:4+metaLength], &meta); err != nil {
		return nil, err
	}
	data := content[4+metaLength:]
	if meta.NodeCount <= 0 || len(data) < meta.NodeCount*8 {
		return nil, fmt.Errorf("invalid ipdb file: bad node count")
	}
	return &tree{nodeCount: meta.NodeCount, data: data}, nil
}

func (t *tree) readNode(node, bit int) int {
	off := node*8 + bit*4
	return int(binary.BigEndian.Uint32(t.data[off : off+4]))
}

// walk visits every prefix that points to a data record. A node value greater
// than nodeCount is a record pointer, equal to nodeCount means "no data".
func (t *tree) walk(node int, ip [16]byte, depth, maxDepth int, skip func(node, depth int) bool, fn func(ip [16]byte, depth, record int) error) error {
	if node > t.nodeCount {
		return fn(ip, depth, node)
	}
	if node == t.nodeCount || depth >= maxDepth || skip(node, depth) {
		return nil
	}
	for bit := 0; bit < 2; bit++ {
		next := ip
		if bit == 1 {
			next[depth>>3] |= 1 << uint(7-depth%8)
		}
		if err := t.walk(t.readNode(node, bit), next, depth+1, maxDepth, skip, fn); err != nil {
			return err
		}
	}
	return nil
}

func (i *ipdb_in) generateEntries(content []byte, entries map[string]*lib.Entry) error {
	db, err := ipdb.NewCityFromBytes(content)
	if err != nil {
		return err
	}
	t, err := newTree(content)
	if err != nil {
		return err
	}

	fieldIdx := cityInfoFieldIndex(i.Field)
	// The same record pointer always resolves to the same CityInfo, so cache names.
	names := make(map[int]string)

	handle := func(addr netip.Addr, bits, record int) error {
		name, found := names[record]
		if !found {
			info, err := db.FindInfo(addr.String(), i.Language)
			if err != nil {
				return fmt.Errorf("failed to find info of %s: %w", addr, err)
			}
			name = strings.ToUpper(strings.TrimSpace(reflect.ValueOf(info).Elem().Field(fieldIdx).String()))
			names[record] = name
		}
		if name == "" || (len(i.Want) > 0 && !i.Want[name]) {
			return nil
		}

		entry, found := entries[name]
		if !found {
			entry = lib.NewEntry(name)
			entries[name] = entry
		}
		return entry.AddPrefix(netip.PrefixFrom(addr, bits))
	}

	// Locate the IPv4 subtree (::ffff:0:0/96 style: 80 zero bits then 16 one bits),
	// the same way ipdb-go does.
	v4offset := 0
	for d := 0; d < 96 && v4offset < t.nodeCount; d++ {
		bit := 0
		if d >= 80 {
			bit = 1
		}
		v4offset = t.readNode(v4offset, bit)
	}

	noSkip := func(int, int) bool { return false }

	if db.IsIPv4() && i.OnlyIPType != lib.IPv6 && v4offset < t.nodeCount {
		err := t.walk(v4offset, [16]byte{}, 0, 32, noSkip, func(ip [16]byte, depth, record int) error {
			return handle(netip.AddrFrom4([4]byte(ip[:4])), depth, record)
		})
		if err != nil {
			return err
		}
	}

	if db.IsIPv6() && i.OnlyIPType != lib.IPv4 {
		// Skip the IPv4 subtree, which has already been handled above.
		skip := func(node, depth int) bool { return depth == 96 && node == v4offset }
		err := t.walk(0, [16]byte{}, 0, 128, skip, func(ip [16]byte, depth, record int) error {
			return handle(netip.AddrFrom16(ip), depth, record)
		})
		if err != nil {
			return err
		}
	}

	return nil
}
