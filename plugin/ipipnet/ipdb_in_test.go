package ipipnet

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
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

func makeDualStackTestIPDB(t testing.TB) []byte {
	t.Helper()
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
	return makeTestIPDB(t, ipdb.MetaData{
		IPVersion: ipdb.IPv4 | ipdb.IPv6,
		Fields:    []string{"country_code", "city_name"},
		Languages: map[string]int{"CN": 0, "EN": 2},
	}, nodes, "cn\t北京\tCN\tBeijing", "us\t纽约\tUS\tNew York")
}

func TestIPDBGenerateEntries(t *testing.T) {
	content := makeDualStackTestIPDB(t)
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

func TestIPDBInput(t *testing.T) {
	content := makeDualStackTestIPDB(t)
	path := filepath.Join(t.TempDir(), "city.ipdb")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer server.Close()

	for source, uri := range map[string]string{"local": path, "remote": server.URL} {
		for _, action := range []lib.Action{lib.ActionAdd, lib.ActionRemove} {
			for _, onlyIPType := range []lib.IPType{"", lib.IPv4, lib.IPv6} {
				t.Run(source+"/"+string(action)+"/"+string(onlyIPType), func(t *testing.T) {
					container := lib.NewContainer()
					input := &IPDBIn{
						Type: TypeIPDBIn, Action: lib.ActionAdd, URI: uri,
						Field: "country_code", Language: "CN",
					}
					if action == lib.ActionRemove {
						if _, err := input.Input(container); err != nil {
							t.Fatal(err)
						}
					}
					input.Action = action
					input.OnlyIPType = onlyIPType
					if _, err := input.Input(container); err != nil {
						t.Fatal(err)
					}
					for _, addr := range []string{"1.1.1.1", "128.0.0.1", "8000::1"} {
						want := onlyIPType == "" || (netip.MustParseAddr(addr).Is4() == (onlyIPType == lib.IPv4))
						if action == lib.ActionRemove {
							want = !want
						}
						_, found, err := container.Lookup(addr)
						if err != nil {
							t.Fatal(err)
						}
						if found != want {
							t.Errorf("Lookup(%s) = %v, want %v", addr, found, want)
						}
					}
				})
			}
		}
	}
}

func TestIPDBMalformedDatabase(t *testing.T) {
	valid := makeTestIPDB(t, ipdb.MetaData{
		IPVersion: ipdb.IPv4,
		Fields:    []string{"country_code"},
		Languages: map[string]int{"CN": 0},
	}, [][2]uint32{{17, 17}}, "CN")
	dataStart := 4 + int(binary.BigEndian.Uint32(valid[:4]))
	for _, tc := range []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{"empty", func(b []byte) []byte { return nil }},
		{"short header", func(b []byte) []byte { return b[:3] }},
		{"short metadata", func(b []byte) []byte { return b[:dataStart-1] }},
		{"short node table", func(b []byte) []byte { return b[:dataStart+7] }},
		{"invalid metadata", func(b []byte) []byte { b[4] = '!'; return b }},
		{"record header", func(b []byte) []byte {
			binary.BigEndian.PutUint32(b[dataStart:dataStart+4], uint32(len(b)-dataStart-8))
			return b
		}},
		{"record body", func(b []byte) []byte {
			binary.BigEndian.PutUint16(b[dataStart+24:dataStart+26], math.MaxUint16)
			return b
		}},
		{"cycle", func(b []byte) []byte {
			clear(b[dataStart : dataStart+8])
			return b
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := &IPDBIn{Field: "country_code", Language: "CN"}
			if err := input.generateEntries(tc.mutate(slices.Clone(valid)), make(map[string]*lib.Entry)); err == nil {
				t.Fatal("expected an error for a malformed database")
			}
		})
	}
}

func TestIPDBRemoveOnlyIPType(t *testing.T) {
	content := makeDualStackTestIPDB(t)
	dataStart := 4 + int(binary.BigEndian.Uint32(content[:4]))
	// Leave CN as an IPv4-only category and US as an IPv6-only category.
	binary.BigEndian.PutUint32(content[dataStart+96*8+4:], 97)
	path := filepath.Join(t.TempDir(), "city.ipdb")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		family lib.IPType
		name   string
		prefix string
	}{
		{lib.IPv4, "CN", "0.0.0.0/1"},
		{lib.IPv6, "US", "8000::/1"},
	} {
		t.Run(string(tc.family), func(t *testing.T) {
			container := lib.NewContainer()
			entry := lib.NewEntry(tc.name)
			if err := entry.AddPrefix(tc.prefix); err != nil {
				t.Fatal(err)
			}
			if err := container.Add(entry); err != nil {
				t.Fatal(err)
			}
			input := &IPDBIn{
				Type: TypeIPDBIn, Action: lib.ActionRemove, URI: path,
				Field: "country_code", Language: "CN", OnlyIPType: tc.family,
			}
			if _, err := input.Input(container); err != nil {
				t.Fatal(err)
			}
			if _, found, err := container.Lookup(tc.prefix); err != nil || found {
				t.Fatalf("removed prefix still present: found=%v, err=%v", found, err)
			}
		})
	}
}

func TestIPDBExpansionLimit(t *testing.T) {
	nodes := make([][2]uint32, 13)
	firstRecord := uint32(len(nodes)) + 16
	nodes[0] = [2]uint32{firstRecord, 1}
	for node := 1; node < len(nodes)-1; node++ {
		nodes[node] = [2]uint32{uint32(node + 1), uint32(node + 1)}
	}
	nodes[len(nodes)-1] = [2]uint32{firstRecord, firstRecord + 4}
	content := makeTestIPDB(t, ipdb.MetaData{
		IPVersion: ipdb.IPv6,
		Fields:    []string{"country_code"},
		Languages: map[string]int{"CN": 0},
	}, nodes, "CN", "US")
	input := &IPDBIn{Field: "country_code", Language: "CN"}
	if err := input.generateEntries(content, make(map[string]*lib.Entry)); err == nil {
		t.Fatal("expected excessive expansion of shared nodes to be rejected")
	}
}

func TestIPDBIPv4Alias(t *testing.T) {
	content := makeDualStackTestIPDB(t)
	dataStart := 4 + int(binary.BigEndian.Uint32(content[:4]))
	secondRecord := uint32(97 + 16 + 2 + len("cn\t北京\tCN\tBeijing"))
	binary.BigEndian.PutUint32(content[dataStart+4:], 96)
	binary.BigEndian.PutUint32(content[dataStart+12:], secondRecord)
	input := &IPDBIn{Field: "country_code", Language: "CN"}
	entries := make(map[string]*lib.Entry)
	if err := input.generateEntries(content, entries); err != nil {
		t.Fatal(err)
	}
	entry, ok := entries["US"]
	if !ok {
		t.Fatal("missing US entry")
	}
	set, err := entry.GetIPv6Set()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := set.Prefixes(), []netip.Prefix{netip.MustParsePrefix("4000::/2")}; !slices.Equal(got, want) {
		t.Fatalf("IPv6 prefixes = %v, want %v", got, want)
	}
}

func TestIPDBEarlyIPv4Record(t *testing.T) {
	content := makeTestIPDB(t, ipdb.MetaData{
		IPVersion: ipdb.IPv4 | ipdb.IPv6,
		Fields:    []string{"country_code"},
		Languages: map[string]int{"CN": 0},
	}, [][2]uint32{{17, 21}}, "CN", "US")
	input := &IPDBIn{Field: "country_code", Language: "CN"}
	entries := make(map[string]*lib.Entry)
	if err := input.generateEntries(content, entries); err != nil {
		t.Fatal(err)
	}
	entry, ok := entries["CN"]
	if !ok {
		t.Fatal("missing CN entry")
	}
	set4, err := entry.GetIPv4Set()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := set4.Prefixes(), []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0")}; !slices.Equal(got, want) {
		t.Fatalf("IPv4 prefixes = %v, want %v", got, want)
	}
	set6, err := entry.GetIPv6Set()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := set6.Prefixes(), []netip.Prefix{netip.MustParsePrefix("::/1")}; !slices.Equal(got, want) {
		t.Fatalf("IPv6 prefixes = %v, want %v", got, want)
	}
}

func FuzzIPDBGenerateEntries(f *testing.F) {
	f.Add([]byte{})
	f.Add(makeDualStackTestIPDB(f))
	f.Add(makeTestIPDB(f, ipdb.MetaData{
		IPVersion: ipdb.IPv4 | ipdb.IPv6,
		Fields:    []string{"country_code"},
		Languages: map[string]int{"CN": 0},
	}, [][2]uint32{{17, 21}}, "CN", "US"))
	f.Fuzz(func(t *testing.T, content []byte) {
		input := &IPDBIn{Field: "country_code", Language: "CN"}
		_ = input.generateEntries(content, make(map[string]*lib.Entry))
	})
}
