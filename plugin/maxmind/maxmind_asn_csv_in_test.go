package maxmind

import (
	"encoding/csv"
	"encoding/json"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

// assertASNCSVIn compares the fields of two ASN CSV input converters.
func assertASNCSVIn(t *testing.T, got, want *geolite2_asn_csv_in) {
	t.Helper()
	if got.Type != want.Type || got.Action != want.Action || got.Description != want.Description ||
		got.IPv4File != want.IPv4File || got.IPv6File != want.IPv6File ||
		!maps.EqualFunc(got.Want, want.Want, slices.Equal[[]string]) || got.OnlyIPType != want.OnlyIPType {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestNewGeoLite2ASNCSVIn(t *testing.T) {
	tests := []struct {
		name string
		opts []lib.InputOption
		want *geolite2_asn_csv_in
	}{
		{
			name: "defaults",
			want: &geolite2_asn_csv_in{IPv4File: defaultGeoLite2ASNCSVIPv4File, IPv6File: defaultGeoLite2ASNCSVIPv6File},
		},
		{
			name: "nil option is ignored",
			opts: []lib.InputOption{nil},
			want: &geolite2_asn_csv_in{IPv4File: defaultGeoLite2ASNCSVIPv4File, IPv6File: defaultGeoLite2ASNCSVIPv6File},
		},
		{
			name: "blank files fall back to defaults",
			opts: []lib.InputOption{WithIPv4File(" "), WithIPv6File("")},
			want: &geolite2_asn_csv_in{IPv4File: defaultGeoLite2ASNCSVIPv4File, IPv6File: defaultGeoLite2ASNCSVIPv6File},
		},
		{
			name: "ipv4 file only",
			opts: []lib.InputOption{WithIPv4File(" ./ipv4.csv ")},
			want: &geolite2_asn_csv_in{IPv4File: "./ipv4.csv"},
		},
		{
			name: "ipv6 file only",
			opts: []lib.InputOption{WithIPv6File(" ./ipv6.csv ")},
			want: &geolite2_asn_csv_in{IPv6File: "./ipv6.csv"},
		},
		{
			name: "all options",
			opts: []lib.InputOption{
				WithIPv4File("./ipv4.csv"),
				WithIPv6File("./ipv6.csv"),
				WithGeoLite2ASNCSVWantedList([]string{"AS13335", "15169"}),
				WithInputOnlyIPType(lib.IPv4),
			},
			want: &geolite2_asn_csv_in{
				IPv4File:   "./ipv4.csv",
				IPv6File:   "./ipv6.csv",
				Want:       map[string][]string{"13335": {"AS13335"}, "15169": {"AS15169"}},
				OnlyIPType: lib.IPv4,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGeoLite2ASNCSVIn(lib.ActionRemove, tt.opts...)
			assertErr(t, err, nil)

			tt.want.Type, tt.want.Description, tt.want.Action = TypeGeoLite2ASNCSVIn, DescGeoLite2ASNCSVIn, lib.ActionRemove
			assertASNCSVIn(t, got.(*geolite2_asn_csv_in), tt.want)

			if got.GetType() != TypeGeoLite2ASNCSVIn || got.GetAction() != lib.ActionRemove || got.GetDescription() != DescGeoLite2ASNCSVIn {
				t.Fatalf("unexpected type %q, action %q or description %q", got.GetType(), got.GetAction(), got.GetDescription())
			}
		})
	}
}

func TestNewGeoLite2ASNCSVInFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    *geolite2_asn_csv_in
		wantErr *errCheck
	}{
		{
			name: "no data",
			want: &geolite2_asn_csv_in{IPv4File: defaultGeoLite2ASNCSVIPv4File, IPv6File: defaultGeoLite2ASNCSVIPv6File},
		},
		{
			name: "empty object",
			data: `{}`,
			want: &geolite2_asn_csv_in{IPv4File: defaultGeoLite2ASNCSVIPv4File, IPv6File: defaultGeoLite2ASNCSVIPv6File},
		},
		{
			name: "all arguments",
			data: `{"ipv4": " ./ipv4.csv ", "ipv6": " ./ipv6.csv ", "wantedList": ["AS13335", "15169"], "onlyIPType": "ipv6"}`,
			want: &geolite2_asn_csv_in{
				IPv4File:   "./ipv4.csv",
				IPv6File:   "./ipv6.csv",
				Want:       map[string][]string{"13335": {"AS13335"}, "15169": {"AS15169"}},
				OnlyIPType: lib.IPv6,
			},
		},
		{
			name: "wanted list of numbers",
			data: `{"ipv4": "./ipv4.csv", "wantedList": [13335, 15169]}`,
			want: &geolite2_asn_csv_in{
				IPv4File: "./ipv4.csv",
				Want:     map[string][]string{"13335": {"AS13335"}, "15169": {"AS15169"}},
			},
		},
		{
			name: "wanted lists of numbers",
			data: `{"ipv6": "./ipv6.csv", "wantedList": {"cloudflare": [13335], "cdn": [13335, 54113]}}`,
			want: &geolite2_asn_csv_in{
				IPv6File: "./ipv6.csv",
				Want:     map[string][]string{"13335": {"CDN", "CLOUDFLARE"}, "54113": {"CDN"}},
			},
		},
		{
			name: "wanted lists of strings",
			data: `{"wantedList": {"cloudflare": ["AS13335"], "cdn": ["as13335", "54113"]}}`,
			want: &geolite2_asn_csv_in{
				IPv4File: defaultGeoLite2ASNCSVIPv4File,
				IPv6File: defaultGeoLite2ASNCSVIPv6File,
				Want:     map[string][]string{"13335": {"CDN", "CLOUDFLARE"}, "54113": {"CDN"}},
			},
		},
		{
			name:    "malformed json",
			data:    `{`,
			wantErr: errAs[*json.SyntaxError](),
		},
		{
			name:    "argument of wrong type",
			data:    `{"ipv4": 4}`,
			wantErr: errAs[*json.UnmarshalTypeError](),
		},
		{
			name:    "wanted list of wrong type",
			data:    `{"wantedList": "AS13335"}`,
			wantErr: errContains("invalid JSON config for `wantedList`"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGeoLite2ASNCSVInFromBytes(lib.ActionAdd, []byte(tt.data))
			if assertErr(t, err, tt.wantErr) {
				assertConverterErr(t, err, TypeGeoLite2ASNCSVIn, lib.ActionAdd)
				if got != nil {
					t.Fatalf("got converter %+v along with error %v", got, err)
				}
				return
			}

			tt.want.Type, tt.want.Description, tt.want.Action = TypeGeoLite2ASNCSVIn, DescGeoLite2ASNCSVIn, lib.ActionAdd
			assertASNCSVIn(t, got.(*geolite2_asn_csv_in), tt.want)
		})
	}
}

// assertASNCSVWantedList compares two wanted lists, in which unset and empty
// values are the same.
func assertASNCSVWantedList(t *testing.T, got, want GeoLite2ASNCSVWantedList) {
	t.Helper()
	if !slices.Equal(got.TypeSliceNum, want.TypeSliceNum) || !slices.Equal(got.TypeSliceStr, want.TypeSliceStr) ||
		!maps.EqualFunc(got.TypeMapNum, want.TypeMapNum, slices.Equal[[]uint]) ||
		!maps.EqualFunc(got.TypeMapStr, want.TypeMapStr, slices.Equal[[]string]) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestGeoLite2ASNCSVWantedList_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    GeoLite2ASNCSVWantedList
		wantErr *errCheck
	}{
		{name: "numbers", data: `[13335, 15169]`, want: GeoLite2ASNCSVWantedList{TypeSliceNum: []uint{13335, 15169}}},
		{name: "strings", data: `["AS13335", "15169"]`, want: GeoLite2ASNCSVWantedList{TypeSliceStr: []string{"AS13335", "15169"}}},
		{
			name: "lists of numbers",
			data: `{"cloudflare": [13335], "google": [15169, 36040]}`,
			want: GeoLite2ASNCSVWantedList{TypeMapNum: map[string][]uint{"cloudflare": {13335}, "google": {15169, 36040}}},
		},
		{
			name: "lists of strings",
			data: `{"cloudflare": ["AS13335"], "google": ["AS15169", "36040"]}`,
			want: GeoLite2ASNCSVWantedList{TypeMapStr: map[string][]string{"cloudflare": {"AS13335"}, "google": {"AS15169", "36040"}}},
		},
		{name: "surrounding whitespace", data: " \n[13335]\t ", want: GeoLite2ASNCSVWantedList{TypeSliceNum: []uint{13335}}},
		{name: "empty array", data: `[]`},
		{name: "empty object", data: `{}`},
		{name: "null", data: `null`},
		{name: "no data", data: ``},
		{name: "blank data", data: " \n "},

		{name: "bare string", data: `"AS13335"`, wantErr: errContains("invalid JSON config for `wantedList`: \"AS13335\"")},
		{name: "bare number", data: `13335`, wantErr: errContains("invalid JSON config for `wantedList`")},
		{name: "negative number", data: `[-13335]`, wantErr: errContains("invalid JSON config for `wantedList`")},
		{name: "fractional number", data: `[13335.5]`, wantErr: errContains("invalid JSON config for `wantedList`")},
		{name: "numbers mixed with strings", data: `[13335, "AS15169"]`, wantErr: errContains("invalid JSON config for `wantedList`")},
		{name: "nested array", data: `[[13335]]`, wantErr: errContains("invalid JSON config for `wantedList`")},
		{name: "list that is not an array", data: `{"cloudflare": 13335}`, wantErr: errContains("invalid JSON config for `wantedList`")},
		{name: "list of numbers mixed with strings", data: `{"cloudflare": [13335, "AS15169"]}`, wantErr: errContains("invalid JSON config for `wantedList`")},
		{name: "lists of different types", data: `{"cloudflare": [13335], "google": ["AS15169"]}`, wantErr: errContains("invalid JSON config for `wantedList`")},
		{name: "malformed json", data: `[13335`, wantErr: errContains("invalid JSON config for `wantedList`")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Whatever is left from a previous use must not survive.
			got := GeoLite2ASNCSVWantedList{
				TypeSliceNum: []uint{1},
				TypeSliceStr: []string{"AS1"},
				TypeMapNum:   map[string][]uint{"stale": {1}},
				TypeMapStr:   map[string][]string{"stale": {"AS1"}},
			}

			err := got.UnmarshalJSON([]byte(tt.data))
			if assertErr(t, err, tt.wantErr) {
				return
			}
			assertASNCSVWantedList(t, got, tt.want)
		})
	}
}

func TestWithGeoLite2ASNCSVWantedList(t *testing.T) {
	tests := []struct {
		name   string
		option lib.InputOption
		want   map[string][]string
	}{
		{
			name:   "number",
			option: WithGeoLite2ASNCSVWantedList(uint(13335)),
			want:   map[string][]string{"13335": {"AS13335"}},
		},
		{
			name:   "numbers",
			option: WithGeoLite2ASNCSVWantedList([]uint{13335, 15169, 13335}),
			want:   map[string][]string{"13335": {"AS13335"}, "15169": {"AS15169"}},
		},
		{
			name:   "no numbers",
			option: WithGeoLite2ASNCSVWantedList([]uint{}),
			want:   map[string][]string{},
		},
		{
			name:   "string",
			option: WithGeoLite2ASNCSVWantedList("13335"),
			want:   map[string][]string{"13335": {"AS13335"}},
		},
		{
			name:   "string with prefix",
			option: WithGeoLite2ASNCSVWantedList(" as13335 "),
			want:   map[string][]string{"13335": {"AS13335"}},
		},
		{
			name:   "blank string",
			option: WithGeoLite2ASNCSVWantedList("  "),
			want:   map[string][]string{},
		},
		{
			name:   "prefix only",
			option: WithGeoLite2ASNCSVWantedList("AS"),
			want:   map[string][]string{},
		},
		{
			name:   "strings",
			option: WithGeoLite2ASNCSVWantedList([]string{"AS13335", "15169", "", "as13335", " As15169 "}),
			want:   map[string][]string{"13335": {"AS13335"}, "15169": {"AS15169"}},
		},
		{
			name: "lists of numbers",
			option: WithGeoLite2ASNCSVWantedList(map[string][]uint{
				"cloudflare": {13335, 13335},
				" cdn ":      {13335, 54113},
				"":           {15169},
			}),
			want: map[string][]string{"13335": {"CDN", "CLOUDFLARE"}, "54113": {"CDN"}},
		},
		{
			name: "lists of strings",
			option: WithGeoLite2ASNCSVWantedList(map[string][]string{
				"cloudflare": {"AS13335", "13335", ""},
				" cdn ":      {"as13335", "AS54113"},
				"  ":         {"AS15169"},
			}),
			want: map[string][]string{"13335": {"CDN", "CLOUDFLARE"}, "54113": {"CDN"}},
		},
		{
			name: "parsed json config",
			option: WithGeoLite2ASNCSVWantedList(GeoLite2ASNCSVWantedList{
				TypeSliceNum: []uint{13335},
				TypeSliceStr: []string{"AS15169", "as13335"},
				TypeMapNum:   map[string][]uint{"cdn": {13335, 54113}},
				TypeMapStr:   map[string][]string{"cdn": {"AS54113"}, "google": {"AS15169"}},
			}),
			want: map[string][]string{
				"13335": {"AS13335", "CDN"},
				"15169": {"AS15169", "GOOGLE"},
				"54113": {"CDN"},
			},
		},
		{
			name:   "empty parsed json config",
			option: WithGeoLite2ASNCSVWantedList(GeoLite2ASNCSVWantedList{}),
			want:   map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A wanted list replaces the previous one.
			g := &geolite2_asn_csv_in{Want: map[string][]string{"1": {"STALE"}}}
			tt.option(g)

			if !maps.EqualFunc(g.Want, tt.want, slices.Equal[[]string]) {
				t.Fatalf("got %v, want %v", g.Want, tt.want)
			}
		})
	}
}

func TestGeoLite2ASNCSVIn_process(t *testing.T) {
	dir := t.TempDir()
	ipv4File := writeTestFile(t, dir, "ipv4.csv", []byte(asnIPv4CSV))
	ipv6File := writeTestFile(t, dir, "ipv6.csv", []byte(asnIPv6CSV))

	file := func(name, content string) string {
		return writeTestFile(t, dir, name, []byte(content))
	}

	server := newTestServer(t, map[string][]byte{"/ipv4.csv": []byte(asnIPv4CSV)})

	wantIPv4 := map[string][]string{
		"AS13335": {"1.0.0.0/24", "1.1.1.0/24"},
		"AS15169": {"8.8.8.0/24"},
		"AS38803": {"1.0.4.0/22"},
	}

	tests := []struct {
		name string
		file string
		want map[string][]string
		// entries are the entries generated before the file is processed.
		entries    map[string]*lib.Entry
		wantErr    *errCheck
		wantedList map[string][]string
	}{
		{name: "ipv4 file", file: ipv4File, want: wantIPv4},
		{
			name: "ipv6 file",
			file: ipv6File,
			want: map[string][]string{"AS13335": {"2606:4700::/32"}, "AS15169": {"2001:4860::/32"}},
		},
		{
			name:    "merges into existing entries",
			file:    ipv6File,
			entries: map[string]*lib.Entry{"AS13335": newTestEntry(t, "AS13335", "1.0.0.0/24"), "AS1": newTestEntry(t, "AS1", "192.0.2.0/24")},
			want: map[string][]string{
				"AS1":     {"192.0.2.0/24"},
				"AS13335": {"1.0.0.0/24", "2606:4700::/32"},
				"AS15169": {"2001:4860::/32"},
			},
		},
		{
			name:       "wanted asn",
			file:       ipv4File,
			wantedList: map[string][]string{"13335": {"AS13335"}, "64512": {"AS64512"}},
			want:       map[string][]string{"AS13335": {"1.0.0.0/24", "1.1.1.0/24"}},
		},
		{
			name:       "wanted lists",
			file:       ipv4File,
			wantedList: map[string][]string{"13335": {"CDN", "CLOUDFLARE"}, "15169": {"CDN"}},
			want: map[string][]string{
				"CDN":        {"1.0.0.0/24", "1.1.1.0/24", "8.8.8.0/24"},
				"CLOUDFLARE": {"1.0.0.0/24", "1.1.1.0/24"},
			},
		},
		{
			name:       "wanted lists merge into existing entries",
			file:       ipv6File,
			wantedList: map[string][]string{"13335": {"CLOUDFLARE"}},
			entries:    map[string]*lib.Entry{"CLOUDFLARE": newTestEntry(t, "CLOUDFLARE", "1.0.0.0/24")},
			want:       map[string][]string{"CLOUDFLARE": {"1.0.0.0/24", "2606:4700::/32"}},
		},
		{
			name:       "nothing wanted is found",
			file:       ipv4File,
			wantedList: map[string][]string{"64512": {"AS64512"}},
			want:       map[string][]string{},
		},
		{
			name: "fields are trimmed",
			file: file("spaces.csv", "network,autonomous_system_number,autonomous_system_organization\n 1.0.0.0/24 , 13335 ,CLOUDFLARENET\n"),
			want: map[string][]string{"AS13335": {"1.0.0.0/24"}},
		},
		{
			name:       "fields are trimmed when looking for wanted asn",
			file:       file("spaces.csv", "network,autonomous_system_number,autonomous_system_organization\n 1.0.0.0/24 , 13335 ,CLOUDFLARENET\n"),
			wantedList: map[string][]string{"13335": {"CLOUDFLARE"}},
			want:       map[string][]string{"CLOUDFLARE": {"1.0.0.0/24"}},
		},
		{
			name: "organization is optional",
			file: file("two-columns.csv", "network,autonomous_system_number\n1.0.0.0/24,13335\n"),
			want: map[string][]string{"AS13335": {"1.0.0.0/24"}},
		},
		{
			name: "header only",
			file: file("header.csv", "network,autonomous_system_number,autonomous_system_organization\n"),
			want: map[string][]string{},
		},
		{name: "empty file", file: file("empty.csv", ""), want: map[string][]string{}},
		{name: "remote file", file: server.URL + "/ipv4.csv", want: wantIPv4},
		{name: "remote file with upper case scheme", file: upperCaseScheme(server) + "/ipv4.csv", want: wantIPv4},

		{name: "missing file", file: filepath.Join(dir, "missing.csv"), wantErr: errIs(os.ErrNotExist)},
		{name: "missing remote file", file: server.URL + "/missing.csv", wantErr: errContains("404 Not Found")},
		{name: "invalid https url", file: invalidHTTPSURL, wantErr: errAs[*url.Error]()},
		{
			name:    "record without asn",
			file:    file("one-column.csv", "network\n1.0.0.0/24\n"),
			wantErr: errContains("invalid record: [1.0.0.0/24]"),
		},
		{
			name:    "record with missing fields",
			file:    file("missing-fields.csv", "network,autonomous_system_number,autonomous_system_organization\n1.0.0.0/24,13335\n"),
			wantErr: errIs(csv.ErrFieldCount),
		},
		{
			name:    "malformed csv",
			file:    file("malformed.csv", "network,autonomous_system_number,autonomous_system_organization\n1.0.0.0/24,13335,CLOUD\"FLARE\n"),
			wantErr: errIs(csv.ErrBareQuote),
		},
		{
			name:    "invalid network",
			file:    file("invalid-network.csv", "network,autonomous_system_number,autonomous_system_organization\n1.0.0.0/33,13335,CLOUDFLARENET\n"),
			wantErr: errIs(lib.ErrInvalidCIDR),
		},
		{
			name:       "invalid network of wanted asn",
			file:       file("invalid-network.csv", "network,autonomous_system_number,autonomous_system_organization\n1.0.0.0/33,13335,CLOUDFLARENET\n"),
			wantedList: map[string][]string{"13335": {"CLOUDFLARE"}},
			wantErr:    errIs(lib.ErrInvalidCIDR),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_asn_csv_in{Type: TypeGeoLite2ASNCSVIn, Action: lib.ActionAdd, Want: tt.wantedList}
			entries := tt.entries
			if entries == nil {
				entries = make(map[string]*lib.Entry)
			}

			err := g.process(tt.file, entries)
			if assertErr(t, err, tt.wantErr) {
				return
			}
			assertEntries(t, entries, tt.want)
		})
	}

	t.Run("without entries to fill", func(t *testing.T) {
		g := &geolite2_asn_csv_in{Type: TypeGeoLite2ASNCSVIn, Action: lib.ActionAdd}
		assertErr(t, g.process(ipv4File, nil), nil)
	})
}

func TestGeoLite2ASNCSVIn_Input(t *testing.T) {
	dir := t.TempDir()
	ipv4File := writeTestFile(t, dir, "ipv4.csv", []byte(asnIPv4CSV))
	ipv6File := writeTestFile(t, dir, "ipv6.csv", []byte(asnIPv6CSV))
	missingFile := filepath.Join(dir, "missing.csv")

	server := newTestServer(t, map[string][]byte{
		"/ipv4.csv": []byte(asnIPv4CSV),
		"/ipv6.csv": []byte(asnIPv6CSV),
	})

	// The ASes that have both IPv4 and IPv6 networks.
	dualStack := map[string][]string{"13335": {"AS13335"}, "15169": {"AS15169"}}
	cloudflare := map[string][]string{"13335": {"CLOUDFLARE"}}
	cloudflareLists := map[string][]string{"CLOUDFLARE": {"1.0.0.0/23", "1.1.1.0/24", "2400:cb00::/32", "2606:4700::/32"}}

	newInput := func(g geolite2_asn_csv_in) lib.InputConverter {
		g.Type = TypeGeoLite2ASNCSVIn
		return &g
	}

	runInputCases(t, []inputCase{
		{
			name:  "add all asns",
			input: newInput(geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: ipv4File, IPv6File: ipv6File}),
			want:  asnLists,
		},
		{
			name:  "add ipv4 file only",
			input: newInput(geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: ipv4File}),
			want: map[string][]string{
				"AS13335": {"1.0.0.0/24", "1.1.1.0/24"},
				"AS15169": {"8.8.8.0/24"},
				"AS38803": {"1.0.4.0/22"},
			},
		},
		{
			name:  "add ipv6 file only",
			input: newInput(geolite2_asn_csv_in{Action: lib.ActionAdd, IPv6File: ipv6File}),
			want:  map[string][]string{"AS13335": {"2606:4700::/32"}, "AS15169": {"2001:4860::/32"}},
		},
		{
			name:  "add remote files",
			input: newInput(geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: server.URL + "/ipv4.csv", IPv6File: server.URL + "/ipv6.csv"}),
			want:  asnLists,
		},
		{
			name: "add wanted lists",
			input: newInput(geolite2_asn_csv_in{
				Action: lib.ActionAdd, IPv4File: ipv4File, IPv6File: ipv6File,
				Want: map[string][]string{"13335": {"CDN", "CLOUDFLARE"}, "15169": {"CDN"}},
			}),
			want: map[string][]string{
				"CDN":        {"1.0.0.0/24", "1.1.1.0/24", "8.8.8.0/24", "2001:4860::/32", "2606:4700::/32"},
				"CLOUDFLARE": {"1.0.0.0/24", "1.1.1.0/24", "2606:4700::/32"},
			},
		},
		{
			name:  "add ipv4 only",
			input: newInput(geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: ipv4File, IPv6File: ipv6File, Want: dualStack, OnlyIPType: lib.IPv4}),
			want:  map[string][]string{"AS13335": {"1.0.0.0/24", "1.1.1.0/24"}, "AS15169": {"8.8.8.0/24"}},
		},
		{
			name:  "add ipv6 only",
			input: newInput(geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: ipv4File, IPv6File: ipv6File, Want: dualStack, OnlyIPType: lib.IPv6}),
			want:  map[string][]string{"AS13335": {"2606:4700::/32"}, "AS15169": {"2001:4860::/32"}},
		},
		{
			name:      "add to existing entries",
			input:     newInput(geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: ipv4File, IPv6File: ipv6File, Want: cloudflare}),
			container: newTestContainer(t, map[string][]string{"CLOUDFLARE": {"1.0.0.0/24", "104.16.0.0/13"}, "GOOGLE": {"8.8.4.0/24"}}),
			want: map[string][]string{
				"CLOUDFLARE": {"1.0.0.0/24", "1.1.1.0/24", "104.16.0.0/13", "2606:4700::/32"},
				"GOOGLE":     {"8.8.4.0/24"},
			},
		},
		{
			name:      "remove",
			input:     newInput(geolite2_asn_csv_in{Action: lib.ActionRemove, IPv4File: ipv4File, IPv6File: ipv6File, Want: cloudflare}),
			container: newTestContainer(t, cloudflareLists),
			want:      map[string][]string{"CLOUDFLARE": {"1.0.1.0/24", "2400:cb00::/32"}},
		},
		{
			name:      "remove ipv4 only",
			input:     newInput(geolite2_asn_csv_in{Action: lib.ActionRemove, IPv4File: ipv4File, IPv6File: ipv6File, Want: cloudflare, OnlyIPType: lib.IPv4}),
			container: newTestContainer(t, cloudflareLists),
			want:      map[string][]string{"CLOUDFLARE": {"1.0.1.0/24", "2400:cb00::/32", "2606:4700::/32"}},
		},
		{
			name:      "remove ipv6 only",
			input:     newInput(geolite2_asn_csv_in{Action: lib.ActionRemove, IPv4File: ipv4File, IPv6File: ipv6File, Want: cloudflare, OnlyIPType: lib.IPv6}),
			container: newTestContainer(t, cloudflareLists),
			want:      map[string][]string{"CLOUDFLARE": {"1.0.0.0/23", "1.1.1.0/24", "2400:cb00::/32"}},
		},
		{
			name:    "remove from missing entry",
			input:   newInput(geolite2_asn_csv_in{Action: lib.ActionRemove, IPv4File: ipv4File, IPv6File: ipv6File, Want: cloudflare}),
			wantErr: errContains("entry CLOUDFLARE not found"),
		},
		{
			name:    "unknown action",
			input:   newInput(geolite2_asn_csv_in{Action: lib.Action("unknown"), IPv4File: ipv4File, IPv6File: ipv6File}),
			wantErr: errIs(lib.ErrUnknownAction),
		},
		{
			name:      "container fails to add",
			input:     newInput(geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: ipv4File, IPv6File: ipv6File}),
			container: &mockContainer{Container: lib.NewContainer(), addErr: errAdd},
			wantErr:   errIs(errAdd),
		},
		{
			name:      "container fails to remove",
			input:     newInput(geolite2_asn_csv_in{Action: lib.ActionRemove, IPv4File: ipv4File, IPv6File: ipv6File}),
			container: &mockContainer{Container: lib.NewContainer(), removeErr: errRemove},
			wantErr:   errIs(errRemove),
		},
		{
			name:    "missing ipv4 file",
			input:   newInput(geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: missingFile, IPv6File: ipv6File}),
			wantErr: errIs(os.ErrNotExist),
		},
		{
			name:    "missing ipv6 file",
			input:   newInput(geolite2_asn_csv_in{Action: lib.ActionAdd, IPv4File: ipv4File, IPv6File: missingFile}),
			wantErr: errIs(os.ErrNotExist),
		},
		{
			name: "no entry is generated",
			input: newInput(geolite2_asn_csv_in{
				Action: lib.ActionAdd, IPv4File: ipv4File, IPv6File: ipv6File,
				Want: map[string][]string{"64512": {"AS64512"}},
			}),
			wantErr: errContains("no entry is generated"),
		},
		{
			name:    "no file",
			input:   newInput(geolite2_asn_csv_in{Action: lib.ActionAdd}),
			wantErr: errContains("no entry is generated"),
		},
	})
}
