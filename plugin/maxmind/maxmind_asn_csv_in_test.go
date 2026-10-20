package maxmind

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	asnBlocksIPv4CSV = `network,autonomous_system_number,autonomous_system_organization
1.0.0.0/24,13335,CLOUDFLARENET
1.0.4.0/22,38803,"Gtelecom Pty Ltd"
 1.1.1.0/24 , 13335 ,CLOUDFLARENET
8.8.8.0/24,15169,GOOGLE
`

	asnBlocksIPv6CSV = `network,autonomous_system_number,autonomous_system_organization
2606:4700::/32,13335,CLOUDFLARENET
2001:4860::/32,15169,GOOGLE
2400:cb00::/32,13335,CLOUDFLARENET
`
)

// asnCSVLists are the lists of asnBlocksIPv4CSV and asnBlocksIPv6CSV when
// all ASNs are wanted.
func asnCSVLists() map[string][]string {
	return map[string][]string{
		"AS13335": {"1.0.0.0/24", "1.1.1.0/24", "2400:cb00::/32", "2606:4700::/32"},
		"AS38803": {"1.0.4.0/22"},
		"AS15169": {"8.8.8.0/24", "2001:4860::/32"},
	}
}

// asnCSVFiles writes the ASN CSV fixtures into a temporary directory.
func asnCSVFiles(t *testing.T) (ipv4, ipv6 string) {
	t.Helper()

	return writeTempFile(t, "asn-ipv4.csv", []byte(asnBlocksIPv4CSV)),
		writeTempFile(t, "asn-ipv6.csv", []byte(asnBlocksIPv6CSV))
}

// assertASNWantedList compares wanted ASN lists, ignoring the order of the
// list names of an ASN, which depends on map iteration.
func assertASNWantedList(t *testing.T, got, want map[string][]string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("wanted list has %d ASNs %v, want %d ASNs %v", len(got), got, len(want), want)
	}
	for asn, wantLists := range want {
		gotLists := slices.Sorted(slices.Values(got[asn]))
		wantLists = slices.Sorted(slices.Values(wantLists))
		if !slices.Equal(gotLists, wantLists) {
			t.Fatalf("wanted list of ASN %q = %v, want %v", asn, gotLists, wantLists)
		}
	}
}

func TestNewGeoLite2ASNCSVIn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		action lib.Action
		opts   []lib.InputOption
		want   *geolite2_asn_csv_in
	}{
		{
			name:   "defaults",
			action: lib.ActionAdd,
			want: &geolite2_asn_csv_in{
				IPv4File: defaultGeoLite2ASNCSVIPv4File,
				IPv6File: defaultGeoLite2ASNCSVIPv6File,
			},
		},
		{
			name:   "nil and empty options",
			action: lib.ActionRemove,
			opts:   []lib.InputOption{nil, WithIPv4File(" "), WithIPv6File("")},
			want: &geolite2_asn_csv_in{
				IPv4File: defaultGeoLite2ASNCSVIPv4File,
				IPv6File: defaultGeoLite2ASNCSVIPv6File,
			},
		},
		{
			name:   "all options",
			action: lib.ActionAdd,
			opts: []lib.InputOption{
				WithIPv4File("./ipv4.csv"),
				WithIPv6File("./ipv6.csv"),
				WithASNWantedList(lib.WantedListExtended{TypeSlice: []string{"AS13335"}}),
				WithInputOnlyIPType(lib.IPv6),
			},
			want: &geolite2_asn_csv_in{
				IPv4File:   "./ipv4.csv",
				IPv6File:   "./ipv6.csv",
				Want:       map[string][]string{"13335": {"AS13335"}},
				OnlyIPType: lib.IPv6,
			},
		},
		{
			name:   "only IPv4 file",
			action: lib.ActionAdd,
			opts:   []lib.InputOption{WithIPv4File("./ipv4.csv")},
			want:   &geolite2_asn_csv_in{IPv4File: "./ipv4.csv"},
		},
		{
			name:   "only IPv6 file",
			action: lib.ActionAdd,
			opts:   []lib.InputOption{WithIPv6File("https://example.com/ipv6.csv")},
			want:   &geolite2_asn_csv_in{IPv6File: "https://example.com/ipv6.csv"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			converter, err := NewGeoLite2ASNCSVIn(tt.action, tt.opts...)
			assertError(t, err, false, "", nil)

			if got := converter.GetType(); got != TypeGeoLite2ASNCSVIn {
				t.Errorf("GetType() = %q, want %q", got, TypeGeoLite2ASNCSVIn)
			}
			if got := converter.GetAction(); got != tt.action {
				t.Errorf("GetAction() = %q, want %q", got, tt.action)
			}
			if got := converter.GetDescription(); got != DescGeoLite2ASNCSVIn {
				t.Errorf("GetDescription() = %q, want %q", got, DescGeoLite2ASNCSVIn)
			}

			tt.want.Type = TypeGeoLite2ASNCSVIn
			tt.want.Action = tt.action
			tt.want.Description = DescGeoLite2ASNCSVIn
			if !reflect.DeepEqual(converter, tt.want) {
				t.Fatalf("converter = %#v, want %#v", converter, tt.want)
			}
		})
	}
}

func TestWithASNWantedList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want lib.WantedListExtended
		// wantList is the expected wanted list, keyed by ASN without the "AS" prefix.
		wantList map[string][]string
	}{
		{
			name:     "empty",
			wantList: map[string][]string{},
		},
		{
			name: "ASN slice",
			want: lib.WantedListExtended{
				TypeSlice: []string{"AS13335", " as15169 ", "38803", "", "  ", "As", "aS209242"},
			},
			wantList: map[string][]string{
				"13335":  {"AS13335"},
				"15169":  {"AS15169"},
				"38803":  {"AS38803"},
				"209242": {"AS209242"},
			},
		},
		{
			name: "custom lists",
			want: lib.WantedListExtended{
				TypeMap: map[string][]string{
					" cloudflare ": {"AS13335", "as209242", "", "as"},
					"cdn":          {"13335", " AS15169 "},
					"  ":           {"1"},
					"empty":        {},
				},
			},
			wantList: map[string][]string{
				"13335":  {"CLOUDFLARE", "CDN"},
				"209242": {"CLOUDFLARE"},
				"15169":  {"CDN"},
			},
		},
		{
			name: "ASN slice overrides custom lists",
			want: lib.WantedListExtended{
				TypeSlice: []string{"AS13335"},
				TypeMap:   map[string][]string{"cdn": {"AS13335", "AS15169"}},
			},
			wantList: map[string][]string{
				"13335": {"AS13335"},
				"15169": {"CDN"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := &geolite2_asn_csv_in{Want: map[string][]string{"1": {"OLD"}}}
			WithASNWantedList(tt.want)(g)
			if g.Want == nil {
				t.Fatal("Want is nil, want a non-nil map")
			}
			assertASNWantedList(t, g.Want, tt.wantList)
		})
	}
}

func TestNewGeoLite2ASNCSVInFromBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     string
		want     *geolite2_asn_csv_in
		wantList map[string][]string
		wantErr  bool
	}{
		{
			name: "empty args",
			want: &geolite2_asn_csv_in{
				IPv4File: defaultGeoLite2ASNCSVIPv4File,
				IPv6File: defaultGeoLite2ASNCSVIPv6File,
			},
			wantList: map[string][]string{},
		},
		{
			name: "ASN slice",
			data: `{"ipv4": "./4.csv", "ipv6": "./6.csv", "wantedList": ["AS13335", "15169"], "onlyIPType": "ipv4"}`,
			want: &geolite2_asn_csv_in{
				IPv4File:   "./4.csv",
				IPv6File:   "./6.csv",
				OnlyIPType: lib.IPv4,
			},
			wantList: map[string][]string{"13335": {"AS13335"}, "15169": {"AS15169"}},
		},
		{
			name: "custom lists",
			data: `{"ipv4": "./4.csv", "wantedList": {"cloudflare": ["AS13335"], "google": ["AS15169", "as36040"]}}`,
			want: &geolite2_asn_csv_in{IPv4File: "./4.csv"},
			wantList: map[string][]string{
				"13335": {"CLOUDFLARE"},
				"15169": {"GOOGLE"},
				"36040": {"GOOGLE"},
			},
		},
		{name: "invalid JSON", data: `{"ipv4": `, wantErr: true},
		{name: "invalid wantedList", data: `{"wantedList": 5}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewGeoLite2ASNCSVInFromBytes(lib.ActionAdd, []byte(tt.data))
			assertError(t, err, tt.wantErr, "[type "+TypeGeoLite2ASNCSVIn+" | action add]", nil)
			if tt.wantErr {
				if got != nil {
					t.Fatalf("expected nil converter on error, got %#v", got)
				}
				return
			}

			g := got.(*geolite2_asn_csv_in)
			assertASNWantedList(t, g.Want, tt.wantList)

			tt.want.Type = TypeGeoLite2ASNCSVIn
			tt.want.Action = lib.ActionAdd
			tt.want.Description = DescGeoLite2ASNCSVIn
			tt.want.Want = g.Want // already checked above
			if !reflect.DeepEqual(g, tt.want) {
				t.Fatalf("converter = %#v, want %#v", g, tt.want)
			}
		})
	}
}

func TestGeoLite2ASNCSVIn_Config(t *testing.T) {
	t.Parallel()

	assertInputRegistered(t, TypeGeoLite2ASNCSVIn)

	ipv4, ipv6 := asnCSVFiles(t)

	t.Run("valid args", func(t *testing.T) {
		t.Parallel()

		instance, err := newInstanceFromConfig(t, []map[string]any{{
			"type":   TypeGeoLite2ASNCSVIn,
			"action": lib.ActionAdd,
			"args": map[string]any{
				"ipv4":       ipv4,
				"ipv6":       ipv6,
				"wantedList": map[string][]string{"google": {"AS15169"}},
				"onlyIPType": lib.IPv6,
			},
		}}, nil)
		assertError(t, err, false, "", nil)

		container := lib.NewContainer()
		assertError(t, instance.RunInput(container), false, "", nil)
		assertContainer(t, container, map[string][]string{"GOOGLE": {"2001:4860::/32"}})
	})

	t.Run("invalid args", func(t *testing.T) {
		t.Parallel()

		_, err := newInstanceFromConfig(t, []map[string]any{{
			"type":   TypeGeoLite2ASNCSVIn,
			"action": lib.ActionAdd,
			"args":   map[string]any{"wantedList": true},
		}}, nil)
		assertError(t, err, true, "[type "+TypeGeoLite2ASNCSVIn+" | action add]", nil)
	})
}

func TestGeoLite2ASNCSVIn_Input(t *testing.T) {
	t.Parallel()

	ipv4, ipv6 := asnCSVFiles(t)
	missingFile := filepath.Join(t.TempDir(), "missing.csv")
	srv := newFileServer(t, map[string]string{
		"/ipv4.csv": asnBlocksIPv4CSV,
		"/ipv6.csv": asnBlocksIPv6CSV,
	})
	lists := asnCSVLists()

	files := func(opts ...lib.InputOption) []lib.InputOption {
		return append([]lib.InputOption{WithIPv4File(ipv4), WithIPv6File(ipv6)}, opts...)
	}
	newFullContainer := func(t *testing.T) lib.Container {
		t.Helper()

		return fullContainer(t, lists)
	}

	tests := []struct {
		name        string
		action      lib.Action
		opts        []lib.InputOption
		container   func(t *testing.T) lib.Container
		want        map[string][]string
		wantErr     bool
		errContains string
		target      error
	}{
		{
			name:   "add all ASNs from local files",
			action: lib.ActionAdd,
			opts:   files(),
			want:   lists,
		},
		{
			name:   "add all ASNs from remote files",
			action: lib.ActionAdd,
			opts: []lib.InputOption{
				WithIPv4File(srv.URL + "/ipv4.csv"),
				WithIPv6File("HTTP://" + strings.TrimPrefix(srv.URL, "http://") + "/ipv6.csv"),
			},
			want: lists,
		},
		{
			name:   "add from IPv4 file only",
			action: lib.ActionAdd,
			opts:   []lib.InputOption{WithIPv4File(ipv4)},
			want:   filterLists(lists, lib.IPv4),
		},
		{
			name:   "add from IPv6 file only",
			action: lib.ActionAdd,
			opts:   []lib.InputOption{WithIPv6File(ipv6)},
			want:   filterLists(lists, lib.IPv6, "AS13335", "AS15169"),
		},
		{
			name:   "add wanted ASNs",
			action: lib.ActionAdd,
			opts:   files(WithASNWantedList(lib.WantedListExtended{TypeSlice: []string{"as13335", "AS64496"}})),
			want:   filterLists(lists, "", "AS13335"),
		},
		{
			name:   "add custom lists",
			action: lib.ActionAdd,
			opts: files(WithASNWantedList(lib.WantedListExtended{TypeMap: map[string][]string{
				"cloudflare": {"AS13335"},
				"cdn":        {"AS13335", "AS15169"},
			}})),
			want: map[string][]string{
				"CLOUDFLARE": lists["AS13335"],
				"CDN":        {"1.0.0.0/24", "1.1.1.0/24", "8.8.8.0/24", "2001:4860::/32", "2400:cb00::/32", "2606:4700::/32"},
			},
		},
		{
			name:   "add only IPv4",
			action: lib.ActionAdd,
			opts:   files(WithInputOnlyIPType(lib.IPv4)),
			want:   filterLists(lists, lib.IPv4),
		},
		{
			name:   "add only IPv6",
			action: lib.ActionAdd,
			opts:   files(WithInputOnlyIPType(lib.IPv6)),
			want:   filterLists(lists, lib.IPv6),
		},
		{
			name:   "add into existing container",
			action: lib.ActionAdd,
			opts:   files(WithASNWantedList(lib.WantedListExtended{TypeSlice: []string{"AS15169"}})),
			container: func(t *testing.T) lib.Container {
				return newContainer(t, map[string][]string{
					"AS15169": {extraIPv4, "8.8.8.0/24", extraIPv6},
					otherList: {otherCIDR},
				})
			},
			want: map[string][]string{
				"AS15169": {"8.8.8.0/24", extraIPv4, "2001:4860::/32", extraIPv6},
				otherList: {otherCIDR},
			},
		},
		{
			name:      "remove all ASNs",
			action:    lib.ActionRemove,
			opts:      files(),
			container: newFullContainer,
			want:      afterRemove(lists, ""),
		},
		{
			name:      "remove only IPv4",
			action:    lib.ActionRemove,
			opts:      files(WithInputOnlyIPType(lib.IPv4)),
			container: newFullContainer,
			want:      afterRemove(lists, lib.IPv4),
		},
		{
			name:      "remove only IPv6",
			action:    lib.ActionRemove,
			opts:      files(WithInputOnlyIPType(lib.IPv6)),
			container: newFullContainer,
			want:      afterRemove(lists, lib.IPv6),
		},
		{
			name:        "remove list not in container",
			action:      lib.ActionRemove,
			opts:        files(WithASNWantedList(lib.WantedListExtended{TypeSlice: []string{"AS38803"}})),
			wantErr:     true,
			errContains: "entry AS38803 not found",
		},
		{
			name:        "unknown action",
			action:      lib.Action("unknown"),
			opts:        files(),
			wantErr:     true,
			errContains: "[type " + TypeGeoLite2ASNCSVIn + " | action unknown]",
			target:      lib.ErrUnknownAction,
		},
		{
			name:        "wanted ASNs match nothing",
			action:      lib.ActionAdd,
			opts:        files(WithASNWantedList(lib.WantedListExtended{TypeSlice: []string{"AS64496"}})),
			wantErr:     true,
			errContains: "no entry is generated",
		},
		{
			name:        "IPv4 file error",
			action:      lib.ActionAdd,
			opts:        files(WithIPv4File(missingFile)),
			wantErr:     true,
			errContains: "[type " + TypeGeoLite2ASNCSVIn + " | action add]",
			target:      os.ErrNotExist,
		},
		{
			name:        "IPv6 file error",
			action:      lib.ActionAdd,
			opts:        files(WithIPv6File(srv.URL + "/not-found.csv")),
			wantErr:     true,
			errContains: "404 Not Found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			converter, err := NewGeoLite2ASNCSVIn(tt.action, tt.opts...)
			assertError(t, err, false, "", nil)

			container := lib.NewContainer()
			if tt.container != nil {
				container = tt.container(t)
			}

			got, err := converter.Input(container)
			assertError(t, err, tt.wantErr, tt.errContains, tt.target)
			if tt.wantErr {
				if got != nil {
					t.Fatalf("Input() returned non-nil container on error")
				}
				return
			}

			if got != container {
				t.Fatalf("Input() returned a different container")
			}
			assertContainer(t, got, tt.want)
		})
	}
}

func TestGeoLite2ASNCSVIn_process(t *testing.T) {
	t.Parallel()

	srv := newFileServer(t, map[string]string{"/ipv4.csv": asnBlocksIPv4CSV})
	header := "network,autonomous_system_number,autonomous_system_organization\n"
	cdn := lib.WantedListExtended{TypeMap: map[string][]string{"cdn": {"AS13335", "AS15169"}}}

	tests := []struct {
		name string
		// either content (written to a local file) or uri is used
		content     string
		uri         string
		want        lib.WantedListExtended
		entries     map[string][]string
		wantEntries map[string][]string
		wantErr     bool
		errContains string
		target      error
	}{
		{
			name:        "all ASNs of IPv4 file",
			content:     asnBlocksIPv4CSV,
			wantEntries: filterLists(asnCSVLists(), lib.IPv4),
		},
		{
			name:        "all ASNs of IPv6 file",
			content:     asnBlocksIPv6CSV,
			wantEntries: filterLists(asnCSVLists(), lib.IPv6, "AS13335", "AS15169"),
		},
		{
			name:        "all ASNs of remote file",
			uri:         srv.URL + "/ipv4.csv",
			wantEntries: filterLists(asnCSVLists(), lib.IPv4),
		},
		{
			name:        "wanted ASNs",
			content:     asnBlocksIPv4CSV,
			want:        lib.WantedListExtended{TypeSlice: []string{"AS38803"}},
			wantEntries: filterLists(asnCSVLists(), "", "AS38803"),
		},
		{
			name:        "custom lists",
			content:     asnBlocksIPv6CSV,
			want:        cdn,
			wantEntries: map[string][]string{"CDN": {"2001:4860::/32", "2400:cb00::/32", "2606:4700::/32"}},
		},
		{
			name:        "existing entries of all ASNs are extended",
			content:     asnBlocksIPv6CSV,
			entries:     filterLists(asnCSVLists(), lib.IPv4),
			wantEntries: asnCSVLists(),
		},
		{
			name:        "existing entries of custom lists are extended",
			content:     asnBlocksIPv6CSV,
			want:        cdn,
			entries:     map[string][]string{"CDN": {"1.0.0.0/24"}},
			wantEntries: map[string][]string{"CDN": {"1.0.0.0/24", "2001:4860::/32", "2400:cb00::/32", "2606:4700::/32"}},
		},
		{
			name:        "header only",
			content:     header,
			wantEntries: map[string][]string{},
		},
		{
			name:        "empty file",
			content:     "",
			wantEntries: map[string][]string{},
		},
		{
			name:        "remote file not found",
			uri:         srv.URL + "/not-found.csv",
			wantErr:     true,
			errContains: "404 Not Found",
		},
		{
			name:    "remote connection aborted",
			uri:     failingServerURL(t),
			wantErr: true,
		},
		{
			name:    "missing local file",
			uri:     filepath.Join(t.TempDir(), "missing.csv"),
			wantErr: true,
			target:  os.ErrNotExist,
		},
		{
			name:        "record with too few fields",
			content:     "network\n1.0.0.0/24\n",
			wantErr:     true,
			errContains: "invalid record: [1.0.0.0/24]",
		},
		{
			name:    "record with wrong number of fields",
			content: header + "1.0.0.0/24,13335\n",
			wantErr: true,
			target:  csv.ErrFieldCount,
		},
		{
			name:    "invalid CIDR of all ASNs",
			content: header + "1.0.0.0/33,13335,CLOUDFLARENET\n",
			wantErr: true,
			target:  lib.ErrInvalidCIDR,
		},
		{
			name:    "invalid CIDR of wanted ASNs",
			content: header + "1.0.0.0/33,13335,CLOUDFLARENET\n",
			want:    cdn,
			wantErr: true,
			target:  lib.ErrInvalidCIDR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uri := tt.uri
			if uri == "" {
				uri = writeTempFile(t, "blocks.csv", []byte(tt.content))
			}

			g := &geolite2_asn_csv_in{Type: TypeGeoLite2ASNCSVIn}
			WithASNWantedList(tt.want)(g)

			entries := make(map[string]*lib.Entry)
			for name, cidrs := range tt.entries {
				entries[name] = newEntry(t, name, cidrs...)
			}

			err := g.process(uri, entries)
			assertError(t, err, tt.wantErr, tt.errContains, tt.target)
			if tt.wantErr {
				return
			}
			assertEntries(t, entries, tt.wantEntries)
		})
	}

	t.Run("nil entries", func(t *testing.T) {
		t.Parallel()

		g := &geolite2_asn_csv_in{Type: TypeGeoLite2ASNCSVIn}
		file := writeTempFile(t, "blocks.csv", []byte(asnBlocksIPv4CSV))
		assertError(t, g.process(file, nil), false, "", nil)
	})
}
