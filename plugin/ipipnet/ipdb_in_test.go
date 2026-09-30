package ipipnet

import (
	"encoding/binary"
	"encoding/json"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func ipdbFixture(t *testing.T) []byte {
	t.Helper()
	const nodes = 97
	data := make([]byte, nodes*8+1)
	for i := 0; i < nodes; i++ {
		for bit := 0; bit < 2; bit++ {
			binary.BigEndian.PutUint32(data[i*8+bit*4:], nodes)
		}
	}
	for i := 0; i < 96; i++ {
		bit := 0
		if i >= 80 {
			bit = 1
		}
		binary.BigEndian.PutUint32(data[i*8+bit*4:], uint32(i+1))
	}
	addRecord := func(value string) uint32 {
		ptr := uint32(nodes + len(data) - nodes*8)
		payload := []byte(value)
		data = binary.BigEndian.AppendUint16(data, uint16(len(payload)))
		data = append(data, payload...)
		return ptr
	}
	cn := addRecord("CN\t中国\tCN\tChina")
	us := addRecord("US\t美国\tUS\tUnited States")
	v6 := addRecord("CN\t中国\tCN\tChina")
	binary.BigEndian.PutUint32(data[96*8:], cn)
	binary.BigEndian.PutUint32(data[96*8+4:], us)
	binary.BigEndian.PutUint32(data[4:], v6)

	meta, err := json.Marshal(struct {
		IPVersion uint16         `json:"ip_version"`
		NodeCount int            `json:"node_count"`
		TotalSize int            `json:"total_size"`
		Languages map[string]int `json:"languages"`
		Fields    []string       `json:"fields"`
	}{
		IPVersion: 3, NodeCount: nodes, TotalSize: len(data),
		Languages: map[string]int{"CN": 0, "EN": 2},
		Fields:    []string{"country_code", "country_name"},
	})
	if err != nil {
		t.Fatal(err)
	}
	content := binary.BigEndian.AppendUint32(nil, uint32(len(meta)))
	return append(append(content, meta...), data...)
}

func changeIPDBMetadata(t *testing.T, content []byte, change func(map[string]any)) []byte {
	t.Helper()
	metaSize := int(binary.BigEndian.Uint32(content[:4]))
	var meta map[string]any
	if err := json.Unmarshal(content[4:4+metaSize], &meta); err != nil {
		t.Fatal(err)
	}
	change(meta)
	header, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	result := binary.BigEndian.AppendUint32(nil, uint32(len(header)))
	return append(append(result, header...), content[4+metaSize:]...)
}

func fixtureInput(t *testing.T, content []byte, args string, action lib.Action) *IPDBIn {
	t.Helper()
	path := filepath.Join(t.TempDir(), "city.ipdb")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	conf := `{"uri":` + stringMustJSON(t, path) + `,` + args + `}`
	input, err := newIPDBIn(action, json.RawMessage(conf))
	if err != nil {
		t.Fatal(err)
	}
	return input.(*IPDBIn)
}

func stringMustJSON(t *testing.T, s string) string {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func assertLookup(t *testing.T, container lib.Container, ip, name string, found bool) {
	t.Helper()
	names, ok, err := container.Lookup(ip)
	if err != nil {
		t.Fatal(err)
	}
	if ok != found || (found && (len(names) != 1 || names[0] != name)) {
		t.Fatalf("lookup %s: names=%v, found=%v; want %s, %v", ip, names, ok, name, found)
	}
}

func TestIPDBInput(t *testing.T) {
	content := ipdbFixture(t)
	input := fixtureInput(t, content, `"field":"country_code"`, lib.ActionAdd)
	container := lib.NewContainer()
	if _, err := input.Input(container); err != nil {
		t.Fatal(err)
	}
	if container.Len() != 2 {
		t.Fatalf("got %d entries, want 2", container.Len())
	}
	assertLookup(t, container, "1.2.3.4", "CN", true)
	assertLookup(t, container, "129.2.3.4", "US", true)
	assertLookup(t, container, "9000::1", "CN", true)
	assertLookup(t, container, "2001::1", "", false)
	assertLookup(t, container, "::ffff:1.2.3.4", "CN", true)

	cn, _ := container.GetEntry("CN")
	v4, err := cn.GetIPv4Set()
	if err != nil || !v4.ContainsPrefix(netip.MustParsePrefix("0.0.0.0/1")) {
		t.Fatalf("missing IPv4 CIDR: %v", err)
	}
	v6, err := cn.GetIPv6Set()
	if err != nil || !v6.ContainsPrefix(netip.MustParsePrefix("8000::/1")) {
		t.Fatalf("missing IPv6 CIDR: %v", err)
	}
}

func TestIPDBOptionsAndRemoval(t *testing.T) {
	content := ipdbFixture(t)
	input := fixtureInput(t, content, `"field":"CountryName","language":"EN","wantedList":["china"],"onlyIPType":"ipv4"`, lib.ActionAdd)
	container := lib.NewContainer()
	if _, err := input.Input(container); err != nil {
		t.Fatal(err)
	}
	assertLookup(t, container, "1.2.3.4", "CHINA", true)
	assertLookup(t, container, "9000::1", "", false)
	assertLookup(t, container, "129.2.3.4", "", false)

	remove := fixtureInput(t, content, `"field":"CountryName","language":"EN","wantedList":["CHINA"],"onlyIPType":"ipv4"`, lib.ActionRemove)
	if _, err := remove.Input(container); err != nil {
		t.Fatal(err)
	}
	assertLookup(t, container, "1.2.3.4", "", false)

	defaultLanguage := fixtureInput(t, content, `"field":"country_name","onlyIPType":"ipv6"`, lib.ActionAdd)
	if _, err := defaultLanguage.Input(container); err != nil {
		t.Fatal(err)
	}
	assertLookup(t, container, "9000::1", "中国", true)
}

func TestIPDBMappedLeaf(t *testing.T) {
	content := ipdbFixture(t)
	tree, _, err := parseIPDBTree(content)
	if err != nil {
		t.Fatal(err)
	}
	leaf := tree.child(96, 0)
	var prefixes []netip.Prefix
	if err := tree.walk(leaf, netip.PrefixFrom(netip.IPv6Unspecified(), 0), func(_ uint32, prefix netip.Prefix) error {
		prefixes = append(prefixes, prefix)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(prefixes) != 96 {
		t.Fatalf("got %d prefixes, want 96 after excluding mapped /96", len(prefixes))
	}
	for _, prefix := range prefixes {
		if prefix.Contains(mappedIPv4) {
			t.Fatalf("mapped IPv4 included in IPv6 CIDR %s", prefix)
		}
	}
}

func TestIPDBInvalidConfigAndData(t *testing.T) {
	for _, field := range []string{"", "asn_info", "not_a_field"} {
		if _, err := newIPDBIn(lib.ActionAdd, json.RawMessage(`{"uri":"unused","field":`+stringMustJSON(t, field)+`}`)); err == nil {
			t.Errorf("accepted invalid field %q", field)
		}
	}

	content := ipdbFixture(t)
	for _, args := range []string{
		`"field":"region_name"`,
		`"field":"country_code","language":"FR"`,
		`"field":"country_code","wantedList":["JP"]`,
	} {
		input := fixtureInput(t, content, args, lib.ActionAdd)
		if _, err := input.Input(lib.NewContainer()); err == nil {
			t.Errorf("accepted invalid input %s", args)
		}
	}
	badSize := append([]byte(nil), content...)
	binary.BigEndian.PutUint32(badSize[:4], uint32(len(content)))
	badPointer := append([]byte(nil), content...)
	dataStart := 4 + int(binary.BigEndian.Uint32(content[:4]))
	binary.BigEndian.PutUint32(badPointer[dataStart+4:], ^uint32(0))
	badLength := append([]byte(nil), content...)
	tree, _, err := parseIPDBTree(badLength)
	if err != nil {
		t.Fatal(err)
	}
	recordPos := dataStart + tree.nodeCount*8 + int(tree.child(96, 0)) - tree.nodeCount
	binary.BigEndian.PutUint16(badLength[recordPos:], 65535)
	badCycle := append([]byte(nil), content...)
	binary.BigEndian.PutUint32(badCycle[dataStart+4:], 0)
	badExpansion := append([]byte(nil), content...)
	for i := 0; i < 20; i++ {
		binary.BigEndian.PutUint32(badExpansion[dataStart+i*8:], uint32(i+1))
		binary.BigEndian.PutUint32(badExpansion[dataStart+i*8+4:], uint32(i+1))
	}
	binary.BigEndian.PutUint32(badExpansion[dataStart+20*8:], tree.child(96, 0))
	binary.BigEndian.PutUint32(badExpansion[dataStart+20*8+4:], tree.child(96, 0))
	badLanguage := changeIPDBMetadata(t, content, func(meta map[string]any) {
		meta["languages"] = map[string]int{"CN": int(^uint(0) >> 1)}
	})
	for name, value := range map[string][]byte{
		"short": content[:3], "metadata": badSize, "pointer": badPointer,
		"record": badLength, "cycle": badCycle, "expansion": badExpansion, "language": badLanguage,
	} {
		t.Run(name, func(t *testing.T) {
			input := fixtureInput(t, value, `"field":"country_code"`, lib.ActionAdd)
			if _, err := input.Input(lib.NewContainer()); err == nil {
				t.Fatal("accepted invalid database")
			}
		})
	}
}

func TestIPDBMissingURI(t *testing.T) {
	_, err := newIPDBIn(lib.ActionAdd, nil)
	if err == nil || !strings.Contains(err.Error(), "uri") {
		t.Fatalf("missing required URI error: %v", err)
	}
}
