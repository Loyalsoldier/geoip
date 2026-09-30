package ipipnet

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"net/netip"
	"slices"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/ipipdotnet/ipdb-go"
)

func makeTestIPDB(t testing.TB, meta ipdb.MetaData, nodes [][2]uint32, records ...string) []byte {
	t.Helper()
	var data []byte
	for _, node := range nodes {
		data = binary.BigEndian.AppendUint32(data, node[0])
		data = binary.BigEndian.AppendUint32(data, node[1])
	}
	data = append(data, make([]byte, 16)...)
	for _, record := range records {
		data = binary.BigEndian.AppendUint16(data, uint16(len(record)))
		data = append(data, record...)
	}
	meta.NodeCount = len(nodes)
	meta.TotalSize = len(data)
	header, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	content := binary.BigEndian.AppendUint32(nil, uint32(len(header)))
	content = append(content, header...)
	return append(content, data...)
}

func TestIPDBLanguageOffsets(t *testing.T) {
	for _, tc := range []struct {
		name   string
		offset int
	}{
		{"negative", -1},
		{"overflow", math.MaxInt},
		{"past record", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("generateEntries panicked: %v", r)
				}
			}()
			content := makeTestIPDB(t, ipdb.MetaData{
				IPVersion: ipdb.IPv4,
				Fields:    []string{"country_code"},
				Languages: map[string]int{"CN": tc.offset},
			}, [][2]uint32{{17, 17}}, "CN")
			input := &IPDBIn{Field: "country_code", Language: "CN"}
			if err := input.generateEntries(content, make(map[string]*lib.Entry)); err == nil {
				t.Fatal("expected an error for an invalid language offset")
			}
		})
	}
}

func TestIPDBInvalidRecordPointer(t *testing.T) {
	content := makeTestIPDB(t, ipdb.MetaData{
		IPVersion: ipdb.IPv4,
		Fields:    []string{"country_code"},
		Languages: map[string]int{"CN": 0},
	}, [][2]uint32{{math.MaxUint32, math.MaxUint32}}, "CN")
	input := &IPDBIn{Field: "country_code", Language: "CN"}
	if err := input.generateEntries(content, make(map[string]*lib.Entry)); err == nil {
		t.Fatal("expected an error for an out-of-range record pointer")
	}
}

func TestIPDBGenerateEntries(t *testing.T) {
	// The IPv4 subtree starts after the 96-bit mapped prefix.
	nodes := make([][2]uint32, 97)
	noData := uint32(len(nodes))
	firstRecord := noData + 16
	secondRecord := firstRecord + 2 + uint32(len("cn\t北京\tCN\tBeijing"))
	for bit := range 96 {
		nodes[bit] = [2]uint32{noData, noData}
		branch := 0
		if bit >= 80 {
			branch = 1
		}
		nodes[bit][branch] = uint32(bit + 1)
	}
	nodes[0][1] = secondRecord
	nodes[96] = [2]uint32{firstRecord, secondRecord}
	content := makeTestIPDB(t, ipdb.MetaData{
		IPVersion: ipdb.IPv4 | ipdb.IPv6,
		Fields:    []string{"country_code", "city_name"},
		Languages: map[string]int{"CN": 0, "EN": 2},
	}, nodes, "cn\t北京\tCN\tBeijing", "us\t纽约\tUS\tNew York")

	for _, tc := range []struct {
		name  string
		args  string
		want4 map[string][]netip.Prefix
		want6 map[string][]netip.Prefix
	}{
		{
			name: "default field",
			args: `{"uri":"unused"}`,
			want4: map[string][]netip.Prefix{
				"CN": {netip.MustParsePrefix("0.0.0.0/1")},
				"US": {netip.MustParsePrefix("128.0.0.0/1")},
			},
			want6: map[string][]netip.Prefix{"US": {netip.MustParsePrefix("8000::/1")}},
		},
		{
			name:  "field language and wanted list",
			args:  `{"uri":"unused","field":" CITY_NAME ","language":"en","wantedList":[" beijing "]}`,
			want4: map[string][]netip.Prefix{"BEIJING": {netip.MustParsePrefix("0.0.0.0/1")}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			converter, err := newIPDBIn(lib.ActionAdd, json.RawMessage(tc.args))
			if err != nil {
				t.Fatal(err)
			}
			entries := make(map[string]*lib.Entry)
			if err := converter.(*IPDBIn).generateEntries(content, entries); err != nil {
				t.Fatal(err)
			}
			if len(entries) != len(tc.want4) {
				t.Fatalf("got %d entries, want %d", len(entries), len(tc.want4))
			}
			for name, want := range tc.want4 {
				entry, ok := entries[name]
				if !ok {
					t.Fatalf("missing entry %s", name)
				}
				set, err := entry.GetIPv4Set()
				if err != nil {
					t.Fatal(err)
				}
				if got := set.Prefixes(); !slices.Equal(got, want) {
					t.Errorf("%s IPv4 = %v, want %v", name, got, want)
				}
				set, err = entry.GetIPv6Set()
				if want := tc.want6[name]; len(want) > 0 {
					if err != nil {
						t.Fatal(err)
					}
					if got := set.Prefixes(); !slices.Equal(got, want) {
						t.Errorf("%s IPv6 = %v, want %v", name, got, want)
					}
				} else if err == nil && len(set.Prefixes()) > 0 {
					t.Errorf("%s has unexpected IPv6 networks: %v", name, set.Prefixes())
				}
			}
		})
	}
}
