package ipipnet

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"maps"
	"net/netip"
	"os"
	"reflect"
	"slices"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/ipipdotnet/ipdb-go"
)

const (
	TypeIPDBIn = "ipipnetIPDB"
	DescIPDBIn = "Convert ipip.net ipdb database to other formats"
)

const (
	defaultField    = "country_code"
	defaultLanguage = "CN"
)

// v4MappedPrefix is the first 12 bytes of IPv4-mapped IPv6 addresses (::ffff:0:0/96),
// under which ipdb stores IPv4 networks.
var v4MappedPrefix = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff}

// cityInfoFields maps the JSON tag of each string field in ipdb.CityInfo to its field index,
// e.g. "country_code" -> index of ipdb.CityInfo.CountryCode
var cityInfoFields = func() map[string]int {
	fields := make(map[string]int)
	t := reflect.TypeFor[ipdb.CityInfo]()
	for i := range t.NumField() {
		f := t.Field(i)
		if f.Type.Kind() != reflect.String {
			continue
		}
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}
		fields[name] = i
	}
	return fields
}()

func init() {
	lib.RegisterInputConfigCreator(TypeIPDBIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newIPDBIn(action, data)
	})
	lib.RegisterInputConverter(TypeIPDBIn, &IPDBIn{
		Description: DescIPDBIn,
	})
}

func newIPDBIn(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
	var tmp struct {
		URI        string     `json:"uri"`
		Field      string     `json:"field"`
		Language   string     `json:"language"`
		Want       []string   `json:"wantedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	if tmp.URI == "" {
		return nil, fmt.Errorf("❌ [type %s | action %s] uri must be specified in config", TypeIPDBIn, action)
	}

	tmp.Field = strings.ToLower(strings.TrimSpace(tmp.Field))
	if tmp.Field == "" {
		tmp.Field = defaultField
	}
	if _, found := cityInfoFields[tmp.Field]; !found {
		return nil, fmt.Errorf("❌ [type %s | action %s] invalid field %q, must be one of: %s", TypeIPDBIn, action, tmp.Field, strings.Join(slices.Sorted(maps.Keys(cityInfoFields)), ", "))
	}

	tmp.Language = strings.TrimSpace(tmp.Language)
	if tmp.Language == "" {
		tmp.Language = defaultLanguage
	}

	// Filter want list
	wantList := make(map[string]bool)
	for _, want := range tmp.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList[want] = true
		}
	}

	return &IPDBIn{
		Type:        TypeIPDBIn,
		Action:      action,
		Description: DescIPDBIn,
		URI:         tmp.URI,
		Field:       tmp.Field,
		Language:    tmp.Language,
		Want:        wantList,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

type IPDBIn struct {
	Type        string
	Action      lib.Action
	Description string
	URI         string
	Field       string
	Language    string
	Want        map[string]bool
	OnlyIPType  lib.IPType
}

func (i *IPDBIn) GetType() string {
	return i.Type
}

func (i *IPDBIn) GetAction() lib.Action {
	return i.Action
}

func (i *IPDBIn) GetDescription() string {
	return i.Description
}

func (i *IPDBIn) Input(container lib.Container) (lib.Container, error) {
	entries := make(map[string]*lib.Entry)
	var err error

	switch {
	case strings.HasPrefix(strings.ToLower(i.URI), "http://"), strings.HasPrefix(strings.ToLower(i.URI), "https://"):
		err = i.walkRemoteFile(i.URI, entries)
	default:
		err = i.walkLocalFile(i.URI, entries)
	}

	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] no entry is generated", i.Type, i.Action)
	}

	ignoreIPType := lib.GetIgnoreIPType(i.OnlyIPType)

	for _, entry := range entries {
		switch i.Action {
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

func (i *IPDBIn) walkLocalFile(path string, entries map[string]*lib.Entry) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return i.generateEntries(content, entries)
}

func (i *IPDBIn) walkRemoteFile(url string, entries map[string]*lib.Entry) error {
	content, err := lib.GetRemoteURLContent(url)
	if err != nil {
		return err
	}

	return i.generateEntries(content, entries)
}

func (i *IPDBIn) generateEntries(content []byte, entries map[string]*lib.Entry) error {
	tree, err := newIPDBTree(content)
	if err != nil {
		return fmt.Errorf("❌ [type %s | action %s] %w", i.Type, i.Action, err)
	}

	db, err := ipdb.NewCityFromBytes(content)
	if err != nil {
		return err
	}

	if !slices.Contains(db.Fields(), i.Field) {
		return fmt.Errorf("❌ [type %s | action %s] field %q is not in the database, available fields: %s", i.Type, i.Action, i.Field, strings.Join(db.Fields(), ", "))
	}

	languages := db.Languages()
	slices.Sort(languages)
	idx := slices.IndexFunc(languages, func(l string) bool { return strings.EqualFold(l, i.Language) })
	if idx < 0 {
		return fmt.Errorf("❌ [type %s | action %s] language %q is not in the database, available languages: %s", i.Type, i.Action, i.Language, strings.Join(languages, ", "))
	}
	language := languages[idx]

	fieldIndex := cityInfoFields[i.Field]

	// Many networks point to the same record, so cache the entry name of each record
	recordNames := make(map[uint32]string)

	handler := func(prefix netip.Prefix, record uint32) error {
		name, found := recordNames[record]
		if !found {
			info, err := db.FindInfo(prefix.Addr().String(), language)
			if err != nil {
				return fmt.Errorf("❌ [type %s | action %s] failed to get info of %s: %w", i.Type, i.Action, prefix, err)
			}
			name = strings.ToUpper(strings.TrimSpace(reflect.ValueOf(info).Elem().Field(fieldIndex).String()))
			recordNames[record] = name
		}

		if name == "" {
			return nil
		}

		if len(i.Want) > 0 && !i.Want[name] {
			return nil
		}

		entry, found := entries[name]
		if !found {
			entry = lib.NewEntry(name)
			entries[name] = entry
		}

		return entry.AddPrefix(prefix)
	}

	if db.IsIPv4() {
		var ip [4]byte
		if err := tree.walk(tree.v4offset, ip[:], 0, handler); err != nil {
			return err
		}
	}

	if db.IsIPv6() {
		var ip [16]byte
		if err := tree.walk(0, ip[:], 0, handler); err != nil {
			return err
		}
	}

	return nil
}

// ipdbTree is the binary trie of an ipdb database. ipdb-go only supports looking up
// a single IP, so the trie is walked here to enumerate all networks in the database.
//
// The data section of an ipdb database starts with nodeCount nodes of 8 bytes, each
// containing two big-endian uint32 pointers for bit 0 and bit 1. A pointer p means:
//   - p < nodeCount: another node
//   - p == nodeCount: no data
//   - p > nodeCount: a record at data[p-nodeCount+nodeCount*8]
type ipdbTree struct {
	nodeCount uint32
	v4offset  uint32 // the node of ::ffff:0:0/96, i.e. the root of IPv4 networks
	data      []byte
}

func newIPDBTree(content []byte) (*ipdbTree, error) {
	if len(content) < 4 {
		return nil, fmt.Errorf("invalid ipdb database: file too small")
	}

	metaLength := uint64(binary.BigEndian.Uint32(content[:4]))
	if uint64(len(content)) < 4+metaLength {
		return nil, fmt.Errorf("invalid ipdb database: metadata out of range")
	}

	var meta struct {
		NodeCount int `json:"node_count"`
	}
	if err := json.Unmarshal(content[4:4+metaLength], &meta); err != nil {
		return nil, err
	}

	data := content[4+metaLength:]
	if meta.NodeCount <= 0 || uint64(meta.NodeCount) > uint64(len(data))/8 {
		return nil, fmt.Errorf("invalid ipdb database: node count %d out of range", meta.NodeCount)
	}

	t := &ipdbTree{
		nodeCount: uint32(meta.NodeCount),
		data:      data,
	}

	// Same as ipdb-go: follow 80 zero bits and 16 one bits to find ::ffff:0:0/96
	node := uint32(0)
	for i := 0; i < 96 && node < t.nodeCount; i++ {
		if i >= 80 {
			node = t.readNode(node, 1)
		} else {
			node = t.readNode(node, 0)
		}
	}
	t.v4offset = node

	return t, nil
}

func (t *ipdbTree) readNode(node uint32, bit int) uint32 {
	off := int(node)*8 + bit*4
	return binary.BigEndian.Uint32(t.data[off : off+4])
}

// walk calls handler with every network that has a record in the subtree of node,
// where node is reached by following the first depth bits of ip.
func (t *ipdbTree) walk(node uint32, ip []byte, depth int, handler func(netip.Prefix, uint32) error) error {
	isIPv6 := len(ip) == 16

	// IPv4-mapped IPv6 networks are handled as IPv4 networks
	if isIPv6 && depth == 96 && bytes.Equal(ip[:12], v4MappedPrefix) {
		return nil
	}

	switch {
	case node == t.nodeCount: // no data
		return nil

	case node > t.nodeCount: // record
		offset := uint64(node) - uint64(t.nodeCount) + uint64(t.nodeCount)*8
		if offset+2 > uint64(len(t.data)) {
			return fmt.Errorf("invalid ipdb database: record pointer %d out of range", node)
		}

		var addr netip.Addr
		if isIPv6 {
			addr = netip.AddrFrom16([16]byte(ip))
		} else {
			addr = netip.AddrFrom4([4]byte(ip))
		}
		return handler(netip.PrefixFrom(addr, depth), node)
	}

	// Skip IPv6 networks aliased to IPv4 networks
	if isIPv6 && node == t.v4offset {
		return nil
	}

	if depth >= len(ip)*8 {
		return fmt.Errorf("invalid ipdb database: tree is deeper than %d bits", len(ip)*8)
	}

	mask := byte(0x80) >> (depth % 8)
	for bit := range 2 {
		if bit == 1 {
			ip[depth/8] |= mask
		}
		if err := t.walk(t.readNode(node, bit), ip, depth+1, handler); err != nil {
			return err
		}
	}
	ip[depth/8] &^= mask

	return nil
}
