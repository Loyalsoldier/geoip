package ipipnet

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/netip"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/ipipdotnet/ipdb-go"
)

const (
	TypeIPDBIn = "ipipnetIPDB"
	DescIPDBIn = "Convert IPIP.net ipdb city database to other formats"
)

var mappedIPv4 = netip.MustParseAddr("::ffff:0:0")

func init() {
	lib.RegisterInputConfigCreator(TypeIPDBIn, newIPDBIn)
	lib.RegisterInputConverter(TypeIPDBIn, &IPDBIn{Description: DescIPDBIn})
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
	fieldIndex  int
	fieldTag    string
}

func newIPDBIn(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
	var config struct {
		URI        string     `json:"uri"`
		Field      string     `json:"field"`
		Language   string     `json:"language"`
		Want       []string   `json:"wantedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, err
		}
	}
	if config.URI == "" {
		return nil, fmt.Errorf("❌ [type %s | action %s] uri must be specified in config", TypeIPDBIn, action)
	}

	field := strings.TrimSpace(config.Field)
	cityType := reflect.TypeFor[ipdb.CityInfo]()
	fieldIndex := -1
	fieldTag := ""
	for i := range cityType.NumField() {
		f := cityType.Field(i)
		if f.Type.Kind() == reflect.String && (field == f.Name || field == f.Tag.Get("json")) {
			fieldIndex, fieldTag = i, f.Tag.Get("json")
			break
		}
	}
	if fieldIndex < 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] field must name a string field of ipdb.CityInfo", TypeIPDBIn, action)
	}

	wantList := make(map[string]bool)
	for _, want := range config.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList[want] = true
		}
	}
	return &IPDBIn{
		Type:        TypeIPDBIn,
		Action:      action,
		Description: DescIPDBIn,
		URI:         config.URI,
		Field:       field,
		Language:    config.Language,
		Want:        wantList,
		OnlyIPType:  config.OnlyIPType,
		fieldIndex:  fieldIndex,
		fieldTag:    fieldTag,
	}, nil
}

func (g *IPDBIn) GetType() string        { return g.Type }
func (g *IPDBIn) GetAction() lib.Action  { return g.Action }
func (g *IPDBIn) GetDescription() string { return g.Description }

func (g *IPDBIn) Input(container lib.Container) (lib.Container, error) {
	var content []byte
	var err error
	switch {
	case strings.HasPrefix(strings.ToLower(g.URI), "http://"), strings.HasPrefix(strings.ToLower(g.URI), "https://"):
		content, err = lib.GetRemoteURLContent(g.URI)
	default:
		content, err = os.ReadFile(g.URI)
	}
	if err != nil {
		return nil, err
	}

	entries := make(map[string]*lib.Entry)
	if err := g.generateEntries(content, entries); err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] no entry is generated", g.Type, g.Action)
	}

	ignoreIPType := lib.GetIgnoreIPType(g.OnlyIPType)
	for _, entry := range entries {
		switch g.Action {
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

type ipdbMetadata struct {
	NodeCount int            `json:"node_count"`
	TotalSize int            `json:"total_size"`
	Languages map[string]int `json:"languages"`
	Fields    []string       `json:"fields"`
}

type ipdbTree struct {
	data      []byte
	nodeCount int
	visiting  map[uint32]bool
	steps     int
	maxSteps  int
}

func parseIPDBTree(content []byte) (*ipdbTree, *ipdbMetadata, error) {
	if len(content) < 4 {
		return nil, nil, fmt.Errorf("invalid ipdb header")
	}
	metaSize := uint64(binary.BigEndian.Uint32(content[:4]))
	if metaSize > uint64(len(content)-4) {
		return nil, nil, fmt.Errorf("invalid ipdb metadata size")
	}
	var meta ipdbMetadata
	if err := json.Unmarshal(content[4:4+int(metaSize)], &meta); err != nil {
		return nil, nil, err
	}
	data := content[4+int(metaSize):]
	if meta.TotalSize != len(data) || meta.NodeCount <= 0 || meta.NodeCount > len(data)/8 || uint64(meta.NodeCount) > uint64(^uint32(0)) || len(meta.Languages) == 0 || len(meta.Fields) == 0 {
		return nil, nil, fmt.Errorf("invalid ipdb metadata or index size")
	}
	for i := 0; i < meta.NodeCount*8; i += 4 {
		if uint64(binary.BigEndian.Uint32(data[i:i+4])) > uint64(int(^uint(0)>>1)) {
			return nil, nil, fmt.Errorf("invalid ipdb index pointer at %d", i)
		}
	}
	maxSteps := 10_000_000
	if len(data) < (maxSteps-256)/16 {
		maxSteps = len(data)*16 + 256
	}
	return &ipdbTree{data: data, nodeCount: meta.NodeCount, visiting: make(map[uint32]bool), maxSteps: maxSteps}, &meta, nil
}

func (t *ipdbTree) child(node uint32, bit int) uint32 {
	offset := int(node)*8 + bit*4
	return binary.BigEndian.Uint32(t.data[offset : offset+4])
}

func (t *ipdbTree) v4Root() uint32 {
	var node uint32
	for i := 0; i < 96 && int(node) < t.nodeCount; i++ {
		bit := 0
		if i >= 80 {
			bit = 1
		}
		node = t.child(node, bit)
	}
	return node
}

func (t *ipdbTree) checkRecord(node uint32) error {
	offset := int(node) - t.nodeCount
	recordBase := t.nodeCount * 8
	if offset > len(t.data)-recordBase-2 {
		return fmt.Errorf("invalid ipdb record pointer %d", node)
	}
	pos := recordBase + offset
	size := int(binary.BigEndian.Uint16(t.data[pos : pos+2]))
	if size > len(t.data)-pos-2 {
		return fmt.Errorf("invalid ipdb record length at %d", pos)
	}
	return nil
}

func splitIPDBPrefix(prefix netip.Prefix) (netip.Prefix, netip.Prefix) {
	depth := prefix.Bits()
	left := netip.PrefixFrom(prefix.Addr(), depth+1)
	if prefix.Addr().Is4() {
		addr := prefix.Addr().As4()
		addr[depth/8] |= 1 << (7 - depth%8)
		return left, netip.PrefixFrom(netip.AddrFrom4(addr), depth+1)
	}
	addr := prefix.Addr().As16()
	addr[depth/8] |= 1 << (7 - depth%8)
	return left, netip.PrefixFrom(netip.AddrFrom16(addr), depth+1)
}

func (t *ipdbTree) walk(node uint32, prefix netip.Prefix, visit func(uint32, netip.Prefix) error) error {
	t.steps++
	if t.steps > t.maxSteps {
		return fmt.Errorf("ipdb index expands to too many CIDRs")
	}
	if prefix.Addr().Is6() && prefix.Bits() == 96 && prefix.Addr() == mappedIPv4 {
		return nil
	}
	switch {
	case int(node) == t.nodeCount:
		return nil
	case int(node) > t.nodeCount:
		if err := t.checkRecord(node); err != nil {
			return err
		}
		if prefix.Addr().Is6() && prefix.Bits() < 96 && prefix.Contains(mappedIPv4) {
			left, right := splitIPDBPrefix(prefix)
			if err := t.walk(node, left, visit); err != nil {
				return err
			}
			return t.walk(node, right, visit)
		}
		return visit(node, prefix)
	case prefix.Bits() == prefix.Addr().BitLen():
		return fmt.Errorf("invalid ipdb index depth at node %d", node)
	case t.visiting[node]:
		return fmt.Errorf("invalid ipdb index cycle at node %d", node)
	}

	t.visiting[node] = true
	defer delete(t.visiting, node)
	left, right := splitIPDBPrefix(prefix)
	if err := t.walk(t.child(node, 0), left, visit); err != nil {
		return err
	}
	return t.walk(t.child(node, 1), right, visit)
}

func (g *IPDBIn) generateEntries(content []byte, entries map[string]*lib.Entry) error {
	tree, meta, err := parseIPDBTree(content)
	if err != nil {
		return err
	}
	present := false
	for _, field := range meta.Fields {
		if field == g.fieldTag {
			present = true
			break
		}
	}
	if !present {
		return fmt.Errorf("❌ [type %s | action %s] field %q is not present in database", g.Type, g.Action, g.fieldTag)
	}

	db, err := ipdb.NewCityFromBytes(content)
	if err != nil {
		return err
	}
	language := g.Language
	if language == "" {
		languages := db.Languages()
		sort.Strings(languages)
		language = languages[0]
		if _, ok := meta.Languages["CN"]; ok {
			language = "CN"
		}
	}
	if offset, ok := meta.Languages[language]; !ok || offset < 0 || offset > int(^uint(0)>>1)-len(meta.Fields) {
		return fmt.Errorf("❌ [type %s | action %s] unsupported language %q", g.Type, g.Action, language)
	}

	names := make(map[uint32]string)
	visit := func(node uint32, prefix netip.Prefix) error {
		name, cached := names[node]
		if !cached {
			info, err := db.FindInfo(prefix.Addr().String(), language)
			if err != nil {
				return fmt.Errorf("ipdb lookup %s: %w", prefix, err)
			}
			name = strings.ToUpper(strings.TrimSpace(reflect.ValueOf(info).Elem().Field(g.fieldIndex).String()))
			names[node] = name
		}
		if name == "" || len(g.Want) > 0 && !g.Want[name] {
			return nil
		}
		entry, ok := entries[name]
		if !ok {
			entry = lib.NewEntry(name)
			entries[name] = entry
		}
		return entry.AddPrefix(prefix)
	}

	if db.IsIPv4() && g.OnlyIPType != lib.IPv6 {
		if err := tree.walk(tree.v4Root(), netip.PrefixFrom(netip.IPv4Unspecified(), 0), visit); err != nil {
			return err
		}
	}
	if db.IsIPv6() && g.OnlyIPType != lib.IPv4 {
		if err := tree.walk(0, netip.PrefixFrom(netip.IPv6Unspecified(), 0), visit); err != nil {
			return err
		}
	}
	return nil
}
