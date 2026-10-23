package maxmind

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	countryLocationsCSV = `geoname_id,locale_code,continent_code,continent_name,country_iso_code,country_name,is_in_european_union
6255147,en,AS,Asia,,,0
1814991,en,AS,Asia, cn ,China,0
6252001,en,NA,"North America",US,"United States",0
3175395,en,EU,Europe,IT,Italy,1
2921044,en,EU,Europe,DE,Germany,1
,en,EU,Europe,FR,France,1
`

	countryBlocksIPv4CSV = `network,geoname_id,registered_country_geoname_id,represented_country_geoname_id,is_anonymous_proxy,is_satellite_provider,is_anycast
1.0.0.0/24,1814991,1814991,,0,0,
1.0.1.0/24,6255147, 1814991 ,,0,0,
2.0.0.0/16,,6252001,,0,0,
3.0.0.0/16,,,3175395,0,0,
4.0.0.0/16,,,,1,0,
5.0.0.0/16,999,888,777,0,0,
6.0.0.0/16,3175395,6252001,,0,0,
`

	countryBlocksIPv6CSV = `network,geoname_id,registered_country_geoname_id,represented_country_geoname_id,is_anonymous_proxy,is_satellite_provider,is_anycast
2402:4E00::/32,1814991,1814991,,0,0,
2001:db8::/32,2921044,2921044,,0,0,
2400:cb00::/32,,6252001,,0,0,
`
)

// countryCSVCodes is the country code map of countryLocationsCSV.
func countryCSVCodes() map[string]string {
	return map[string]string{
		"1814991": "CN",
		"6252001": "US",
		"3175395": "IT",
		"2921044": "DE",
	}
}

// countryCSVLists are the lists of countryBlocksIPv4CSV and countryBlocksIPv6CSV.
func countryCSVLists() map[string][]string {
	return map[string][]string{
		"CN": {"1.0.0.0/23", "2402:4e00::/32"},
		"US": {"2.0.0.0/16", "2400:cb00::/32"},
		"IT": {"3.0.0.0/16", "6.0.0.0/16"},
		"DE": {"2001:db8::/32"},
	}
}

// countryCSVFiles writes the country CSV fixtures into a temporary directory.
func countryCSVFiles(t *testing.T) (country, ipv4, ipv6 string) {
	t.Helper()

	return writeTempFile(t, "locations.csv", []byte(countryLocationsCSV)),
		writeTempFile(t, "ipv4.csv", []byte(countryBlocksIPv4CSV)),
		writeTempFile(t, "ipv6.csv", []byte(countryBlocksIPv6CSV))
}

func TestNewGeoLite2CountryCSVIn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		action lib.Action
		opts   []lib.InputOption
		want   *geolite2_country_csv_in
	}{
		{
			name:   "defaults",
			action: lib.ActionAdd,
			want: &geolite2_country_csv_in{
				CountryCodeFile: defaultGeoLite2CountryCodeFile,
				IPv4File:        defaultGeoLite2CountryIPv4File,
				IPv6File:        defaultGeoLite2CountryIPv6File,
			},
		},
		{
			name:   "nil and empty options",
			action: lib.ActionRemove,
			opts:   []lib.InputOption{nil, WithCountryCodeFile(" "), WithIPv4File(""), WithIPv6File(" ")},
			want: &geolite2_country_csv_in{
				CountryCodeFile: defaultGeoLite2CountryCodeFile,
				IPv4File:        defaultGeoLite2CountryIPv4File,
				IPv6File:        defaultGeoLite2CountryIPv6File,
			},
		},
		{
			name:   "all options",
			action: lib.ActionAdd,
			opts: []lib.InputOption{
				WithCountryCodeFile(" ./country.csv "),
				WithIPv4File("./ipv4.csv"),
				WithIPv6File("./ipv6.csv"),
				WithInputWantedList([]string{"cn", " "}),
				WithInputOnlyIPType(lib.IPv4),
			},
			want: &geolite2_country_csv_in{
				CountryCodeFile: "./country.csv",
				IPv4File:        "./ipv4.csv",
				IPv6File:        "./ipv6.csv",
				Want:            map[string]bool{"CN": true},
				OnlyIPType:      lib.IPv4,
			},
		},
		{
			name:   "only IPv4 file",
			action: lib.ActionAdd,
			opts:   []lib.InputOption{WithIPv4File("./ipv4.csv")},
			want: &geolite2_country_csv_in{
				CountryCodeFile: defaultGeoLite2CountryCodeFile,
				IPv4File:        "./ipv4.csv",
			},
		},
		{
			name:   "only IPv6 file",
			action: lib.ActionAdd,
			opts:   []lib.InputOption{WithIPv6File("https://example.com/ipv6.csv")},
			want: &geolite2_country_csv_in{
				CountryCodeFile: defaultGeoLite2CountryCodeFile,
				IPv6File:        "https://example.com/ipv6.csv",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			converter, err := NewGeoLite2CountryCSVIn(tt.action, tt.opts...)
			assertError(t, err, false, "", nil)

			if got := converter.GetType(); got != TypeGeoLite2CountryCSVIn {
				t.Errorf("GetType() = %q, want %q", got, TypeGeoLite2CountryCSVIn)
			}
			if got := converter.GetAction(); got != tt.action {
				t.Errorf("GetAction() = %q, want %q", got, tt.action)
			}
			if got := converter.GetDescription(); got != DescGeoLite2CountryCSVIn {
				t.Errorf("GetDescription() = %q, want %q", got, DescGeoLite2CountryCSVIn)
			}

			tt.want.Type = TypeGeoLite2CountryCSVIn
			tt.want.Action = tt.action
			tt.want.Description = DescGeoLite2CountryCSVIn
			if !reflect.DeepEqual(converter, tt.want) {
				t.Fatalf("converter = %#v, want %#v", converter, tt.want)
			}
		})
	}
}

func TestNewGeoLite2CountryCSVInFromBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    string
		want    *geolite2_country_csv_in
		wantErr bool
	}{
		{
			name: "empty args",
			want: &geolite2_country_csv_in{
				CountryCodeFile: defaultGeoLite2CountryCodeFile,
				IPv4File:        defaultGeoLite2CountryIPv4File,
				IPv6File:        defaultGeoLite2CountryIPv6File,
				Want:            map[string]bool{},
			},
		},
		{
			name: "all args",
			data: `{"country": "./c.csv", "ipv4": "./4.csv", "ipv6": "./6.csv", "wantedList": ["cn", "Us"], "onlyIPType": "ipv6"}`,
			want: &geolite2_country_csv_in{
				CountryCodeFile: "./c.csv",
				IPv4File:        "./4.csv",
				IPv6File:        "./6.csv",
				Want:            map[string]bool{"CN": true, "US": true},
				OnlyIPType:      lib.IPv6,
			},
		},
		{
			name: "only IPv6 file",
			data: `{"ipv6": "./6.csv"}`,
			want: &geolite2_country_csv_in{
				CountryCodeFile: defaultGeoLite2CountryCodeFile,
				IPv6File:        "./6.csv",
				Want:            map[string]bool{},
			},
		},
		{name: "invalid JSON", data: `{`, wantErr: true},
		{name: "invalid field type", data: `{"ipv4": 4}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewGeoLite2CountryCSVInFromBytes(lib.ActionAdd, []byte(tt.data))
			assertError(t, err, tt.wantErr, "[type "+TypeGeoLite2CountryCSVIn+" | action add]", nil)
			if tt.wantErr {
				if got != nil {
					t.Fatalf("expected nil converter on error, got %#v", got)
				}
				return
			}

			tt.want.Type = TypeGeoLite2CountryCSVIn
			tt.want.Action = lib.ActionAdd
			tt.want.Description = DescGeoLite2CountryCSVIn
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("converter = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestWithCountryCodeFile(t *testing.T) {
	t.Parallel()

	g := &geolite2_country_csv_in{CountryCodeFile: "old"}
	WithCountryCodeFile("  ./country.csv\n")(g)
	if g.CountryCodeFile != "./country.csv" {
		t.Fatalf("CountryCodeFile = %q, want %q", g.CountryCodeFile, "./country.csv")
	}
}

func TestGeoLite2CountryCSVIn_Config(t *testing.T) {
	t.Parallel()

	assertInputRegistered(t, TypeGeoLite2CountryCSVIn)

	country, ipv4, ipv6 := countryCSVFiles(t)

	t.Run("valid args", func(t *testing.T) {
		t.Parallel()

		instance, err := newInstanceFromConfig(t, []map[string]any{{
			"type":   TypeGeoLite2CountryCSVIn,
			"action": lib.ActionAdd,
			"args":   map[string]any{"country": country, "ipv4": ipv4, "ipv6": ipv6, "wantedList": []string{"cn"}},
		}}, nil)
		assertError(t, err, false, "", nil)

		container := lib.NewContainer()
		assertError(t, instance.RunInput(container), false, "", nil)
		assertContainer(t, container, filterLists(countryCSVLists(), "", "CN"))
	})

	t.Run("invalid args", func(t *testing.T) {
		t.Parallel()

		_, err := newInstanceFromConfig(t, []map[string]any{{
			"type":   TypeGeoLite2CountryCSVIn,
			"action": lib.ActionAdd,
			"args":   []string{"invalid"},
		}}, nil)
		assertError(t, err, true, "[type "+TypeGeoLite2CountryCSVIn+" | action add]", nil)
	})
}

func TestGeoLite2CountryCSVIn_Input(t *testing.T) {
	t.Parallel()

	country, ipv4, ipv6 := countryCSVFiles(t)
	missingFile := filepath.Join(t.TempDir(), "missing.csv")
	srv := newFileServer(t, map[string]string{
		"/locations.csv": countryLocationsCSV,
		"/ipv4.csv":      countryBlocksIPv4CSV,
		"/ipv6.csv":      countryBlocksIPv6CSV,
	})
	lists := countryCSVLists()

	files := func(opts ...lib.InputOption) []lib.InputOption {
		return append([]lib.InputOption{WithCountryCodeFile(country), WithIPv4File(ipv4), WithIPv6File(ipv6)}, opts...)
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
			name:   "add all lists from local files",
			action: lib.ActionAdd,
			opts:   files(),
			want:   lists,
		},
		{
			name:   "add all lists from remote files",
			action: lib.ActionAdd,
			opts: []lib.InputOption{
				WithCountryCodeFile(srv.URL + "/locations.csv"),
				WithIPv4File(srv.URL + "/ipv4.csv"),
				WithIPv6File("HTTP://" + strings.TrimPrefix(srv.URL, "http://") + "/ipv6.csv"),
			},
			want: lists,
		},
		{
			name:   "add from IPv4 file only",
			action: lib.ActionAdd,
			opts:   []lib.InputOption{WithCountryCodeFile(country), WithIPv4File(ipv4)},
			want:   filterLists(lists, lib.IPv4, "CN", "US", "IT"),
		},
		{
			name:   "add from IPv6 file only",
			action: lib.ActionAdd,
			opts:   []lib.InputOption{WithCountryCodeFile(country), WithIPv6File(ipv6)},
			want:   filterLists(lists, lib.IPv6, "CN", "US", "DE"),
		},
		{
			name:   "add wanted lists",
			action: lib.ActionAdd,
			opts:   files(WithInputWantedList([]string{" cn ", "de", "not-exist"})),
			want:   filterLists(lists, "", "CN", "DE"),
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
			opts:   files(WithInputWantedList([]string{"CN"})),
			container: func(t *testing.T) lib.Container {
				return newContainer(t, map[string][]string{
					"CN":      {extraIPv4, "1.0.0.0/24", extraIPv6},
					otherList: {otherCIDR},
				})
			},
			want: map[string][]string{
				"CN":      {"1.0.0.0/23", extraIPv4, "2402:4e00::/32", extraIPv6},
				otherList: {otherCIDR},
			},
		},
		{
			name:      "remove all lists",
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
			opts:        files(WithInputWantedList([]string{"CN"})),
			wantErr:     true,
			errContains: "entry CN not found",
		},
		{
			name:        "unknown action",
			action:      lib.Action("unknown"),
			opts:        files(),
			wantErr:     true,
			errContains: "[type " + TypeGeoLite2CountryCSVIn + " | action unknown]",
			target:      lib.ErrUnknownAction,
		},
		{
			name:        "wanted list matches nothing",
			action:      lib.ActionAdd,
			opts:        files(WithInputWantedList([]string{"not-exist"})),
			wantErr:     true,
			errContains: "no entry is generated",
		},
		{
			name:        "country code file error",
			action:      lib.ActionAdd,
			opts:        files(WithCountryCodeFile(missingFile)),
			wantErr:     true,
			errContains: "[type " + TypeGeoLite2CountryCSVIn + " | action add]",
			target:      os.ErrNotExist,
		},
		{
			name:        "IPv4 file error",
			action:      lib.ActionAdd,
			opts:        files(WithIPv4File(missingFile)),
			wantErr:     true,
			errContains: "[type " + TypeGeoLite2CountryCSVIn + " | action add]",
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

			converter, err := NewGeoLite2CountryCSVIn(tt.action, tt.opts...)
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

func TestGeoLite2CountryCSVIn_getCountryCode(t *testing.T) {
	t.Parallel()

	srv := newFileServer(t, map[string]string{"/locations.csv": countryLocationsCSV})
	header := "geoname_id,locale_code,continent_code,continent_name,country_iso_code,country_name,is_in_european_union\n"

	tests := []struct {
		name string
		// either content (written to a local file) or uri is used
		content     string
		uri         string
		want        map[string]string
		wantErr     bool
		errContains string
		target      error
	}{
		{
			name:    "local file",
			content: countryLocationsCSV,
			want:    countryCSVCodes(),
		},
		{
			name: "remote file",
			uri:  srv.URL + "/locations.csv",
			want: countryCSVCodes(),
		},
		{
			name: "remote file with upper scheme",
			uri:  "HTTP://" + strings.TrimPrefix(srv.URL, "http://") + "/locations.csv",
			want: countryCSVCodes(),
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
			name:        "empty file",
			content:     "",
			wantErr:     true,
			errContains: "empty country code file",
		},
		{
			name:        "header only",
			content:     header,
			wantErr:     true,
			errContains: "invalid country code data",
		},
		{
			name:        "no country code",
			content:     header + "6255147,en,AS,Asia,,,0\n,en,EU,Europe,FR,France,1\n",
			wantErr:     true,
			errContains: "invalid country code data",
		},
		{
			name:        "record with too few fields",
			content:     "geoname_id,locale_code,continent_code,continent_name\n1814991,en,AS,Asia\n",
			wantErr:     true,
			errContains: "invalid record: [1814991 en AS Asia]",
		},
		{
			name:    "record with wrong number of fields",
			content: header + "1814991,en,AS,Asia,CN\n",
			wantErr: true,
			target:  csv.ErrFieldCount,
		},
		{
			name:    "malformed CSV",
			content: header + "1814991,en,AS,\"Asia,CN,China,0\n",
			wantErr: true,
			target:  csv.ErrQuote,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uri := tt.uri
			if uri == "" {
				uri = writeTempFile(t, "locations.csv", []byte(tt.content))
			}

			g := &geolite2_country_csv_in{Type: TypeGeoLite2CountryCSVIn, CountryCodeFile: uri}
			got, err := g.getCountryCode()
			assertError(t, err, tt.wantErr, tt.errContains, tt.target)
			if tt.wantErr {
				if got != nil {
					t.Fatalf("getCountryCode() = %v, want nil on error", got)
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Fatalf("getCountryCode() returned %d codes %v, want %d codes %v", len(got), got, len(tt.want), tt.want)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("getCountryCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGeoLite2CountryCSVIn_process(t *testing.T) {
	t.Parallel()

	srv := newFileServer(t, map[string]string{"/ipv4.csv": countryBlocksIPv4CSV})
	header := "network,geoname_id,registered_country_geoname_id,represented_country_geoname_id\n"

	tests := []struct {
		name string
		// either content (written to a local file) or uri is used
		content     string
		uri         string
		ccMap       map[string]string
		want        []string
		entries     map[string][]string
		wantEntries map[string][]string
		wantErr     bool
		errContains string
		target      error
	}{
		{
			name:        "IPv4 file",
			content:     countryBlocksIPv4CSV,
			ccMap:       countryCSVCodes(),
			wantEntries: filterLists(countryCSVLists(), lib.IPv4, "CN", "US", "IT"),
		},
		{
			name:        "IPv6 file",
			content:     countryBlocksIPv6CSV,
			ccMap:       countryCSVCodes(),
			wantEntries: filterLists(countryCSVLists(), lib.IPv6, "CN", "US", "DE"),
		},
		{
			name:        "remote file",
			uri:         srv.URL + "/ipv4.csv",
			ccMap:       countryCSVCodes(),
			wantEntries: filterLists(countryCSVLists(), lib.IPv4, "CN", "US", "IT"),
		},
		{
			name:        "wanted lists",
			content:     countryBlocksIPv4CSV,
			ccMap:       countryCSVCodes(),
			want:        []string{"it"},
			wantEntries: filterLists(countryCSVLists(), lib.IPv4, "IT"),
		},
		{
			name:        "existing entries are extended",
			content:     countryBlocksIPv6CSV,
			ccMap:       countryCSVCodes(),
			entries:     filterLists(countryCSVLists(), lib.IPv4, "CN", "US", "IT"),
			wantEntries: countryCSVLists(),
		},
		{
			name:        "first known ID is used",
			content:     header + "1.0.0.0/24,999, 6252001 ,1814991\n1.0.1.0/24,,,1814991\n",
			ccMap:       countryCSVCodes(),
			wantEntries: map[string][]string{"US": {"1.0.0.0/24"}, "CN": {"1.0.1.0/24"}},
		},
		{
			name:        "IPv4-mapped IPv6 CIDR is stored as IPv4",
			content:     header + "::FFFF:1.0.0.0/120,1814991,,\n",
			ccMap:       countryCSVCodes(),
			wantEntries: map[string][]string{"CN": {"1.0.0.0/24"}},
		},
		{
			name:        "header only",
			content:     header,
			ccMap:       countryCSVCodes(),
			wantEntries: map[string][]string{},
		},
		{
			name:        "empty file",
			content:     "",
			ccMap:       countryCSVCodes(),
			wantEntries: map[string][]string{},
		},
		{
			name:        "empty country code map",
			content:     countryBlocksIPv4CSV,
			ccMap:       map[string]string{},
			wantErr:     true,
			errContains: "invalid country code data",
		},
		{
			name:        "remote file not found",
			uri:         srv.URL + "/not-found.csv",
			ccMap:       countryCSVCodes(),
			wantErr:     true,
			errContains: "404 Not Found",
		},
		{
			name:    "remote connection aborted",
			uri:     failingServerURL(t),
			ccMap:   countryCSVCodes(),
			wantErr: true,
		},
		{
			name:    "missing local file",
			uri:     filepath.Join(t.TempDir(), "missing.csv"),
			ccMap:   countryCSVCodes(),
			wantErr: true,
			target:  os.ErrNotExist,
		},
		{
			name:        "record with too few fields",
			content:     "network,geoname_id,registered_country_geoname_id\n1.0.0.0/24,1814991,1814991\n",
			ccMap:       countryCSVCodes(),
			wantErr:     true,
			errContains: "invalid record: [1.0.0.0/24 1814991 1814991]",
		},
		{
			name:    "record with wrong number of fields",
			content: header + "1.0.0.0/24,1814991\n",
			ccMap:   countryCSVCodes(),
			wantErr: true,
			target:  csv.ErrFieldCount,
		},
		{
			name:    "invalid CIDR",
			content: header + "1.0.0.0/33,1814991,,\n",
			ccMap:   countryCSVCodes(),
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

			g := &geolite2_country_csv_in{Type: TypeGeoLite2CountryCSVIn}
			WithInputWantedList(tt.want)(g)

			entries := make(map[string]*lib.Entry)
			for name, cidrs := range tt.entries {
				entries[name] = newEntry(t, name, cidrs...)
			}

			err := g.process(uri, tt.ccMap, entries)
			assertError(t, err, tt.wantErr, tt.errContains, tt.target)
			if tt.wantErr {
				return
			}
			assertEntries(t, entries, tt.wantEntries)
		})
	}

	t.Run("nil entries", func(t *testing.T) {
		t.Parallel()

		g := &geolite2_country_csv_in{Type: TypeGeoLite2CountryCSVIn}
		file := writeTempFile(t, "blocks.csv", []byte(countryBlocksIPv4CSV))
		assertError(t, g.process(file, countryCSVCodes(), nil), false, "", nil)
	})
}
