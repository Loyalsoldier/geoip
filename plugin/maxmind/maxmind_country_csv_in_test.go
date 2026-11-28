package maxmind

import (
	"encoding/csv"
	"encoding/json"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

// assertCountryCSVIn compares the fields of two country CSV input converters.
func assertCountryCSVIn(t *testing.T, got, want *geolite2_country_csv_in) {
	t.Helper()
	if got.Type != want.Type || got.Action != want.Action || got.Description != want.Description ||
		got.CountryCodeFile != want.CountryCodeFile || got.IPv4File != want.IPv4File || got.IPv6File != want.IPv6File ||
		!maps.Equal(got.Want, want.Want) || got.OnlyIPType != want.OnlyIPType {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestNewGeoLite2CountryCSVIn(t *testing.T) {
	tests := []struct {
		name string
		opts []lib.InputOption
		want *geolite2_country_csv_in
	}{
		{
			name: "defaults",
			want: &geolite2_country_csv_in{
				CountryCodeFile: defaultGeoLite2CountryCodeFile,
				IPv4File:        defaultGeoLite2CountryIPv4File,
				IPv6File:        defaultGeoLite2CountryIPv6File,
			},
		},
		{
			name: "nil option is ignored",
			opts: []lib.InputOption{nil},
			want: &geolite2_country_csv_in{
				CountryCodeFile: defaultGeoLite2CountryCodeFile,
				IPv4File:        defaultGeoLite2CountryIPv4File,
				IPv6File:        defaultGeoLite2CountryIPv6File,
			},
		},
		{
			name: "blank files fall back to defaults",
			opts: []lib.InputOption{WithCountryCodeFile(" "), WithIPv4File(" "), WithIPv6File("")},
			want: &geolite2_country_csv_in{
				CountryCodeFile: defaultGeoLite2CountryCodeFile,
				IPv4File:        defaultGeoLite2CountryIPv4File,
				IPv6File:        defaultGeoLite2CountryIPv6File,
			},
		},
		{
			name: "ipv4 file only",
			opts: []lib.InputOption{WithIPv4File(" ./ipv4.csv ")},
			want: &geolite2_country_csv_in{CountryCodeFile: defaultGeoLite2CountryCodeFile, IPv4File: "./ipv4.csv"},
		},
		{
			name: "ipv6 file only",
			opts: []lib.InputOption{WithIPv6File(" ./ipv6.csv ")},
			want: &geolite2_country_csv_in{CountryCodeFile: defaultGeoLite2CountryCodeFile, IPv6File: "./ipv6.csv"},
		},
		{
			name: "all options",
			opts: []lib.InputOption{
				WithCountryCodeFile(" ./country.csv "),
				WithIPv4File("./ipv4.csv"),
				WithIPv6File("./ipv6.csv"),
				WithInputWantedList([]string{" cn ", "", "US", "cn"}),
				WithInputOnlyIPType(lib.IPv6),
			},
			want: &geolite2_country_csv_in{
				CountryCodeFile: "./country.csv",
				IPv4File:        "./ipv4.csv",
				IPv6File:        "./ipv6.csv",
				Want:            map[string]bool{"CN": true, "US": true},
				OnlyIPType:      lib.IPv6,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGeoLite2CountryCSVIn(lib.ActionRemove, tt.opts...)
			assertErr(t, err, nil)

			tt.want.Type, tt.want.Description, tt.want.Action = TypeGeoLite2CountryCSVIn, DescGeoLite2CountryCSVIn, lib.ActionRemove
			assertCountryCSVIn(t, got.(*geolite2_country_csv_in), tt.want)

			if got.GetType() != TypeGeoLite2CountryCSVIn || got.GetAction() != lib.ActionRemove || got.GetDescription() != DescGeoLite2CountryCSVIn {
				t.Fatalf("unexpected type %q, action %q or description %q", got.GetType(), got.GetAction(), got.GetDescription())
			}
		})
	}
}

func TestNewGeoLite2CountryCSVInFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    *geolite2_country_csv_in
		wantErr *errCheck
	}{
		{
			name: "no data",
			want: &geolite2_country_csv_in{
				CountryCodeFile: defaultGeoLite2CountryCodeFile,
				IPv4File:        defaultGeoLite2CountryIPv4File,
				IPv6File:        defaultGeoLite2CountryIPv6File,
			},
		},
		{
			name: "empty object",
			data: `{}`,
			want: &geolite2_country_csv_in{
				CountryCodeFile: defaultGeoLite2CountryCodeFile,
				IPv4File:        defaultGeoLite2CountryIPv4File,
				IPv6File:        defaultGeoLite2CountryIPv6File,
			},
		},
		{
			name: "all arguments",
			data: `{
				"country": " ./country.csv ",
				"ipv4": " ./ipv4.csv ",
				"ipv6": " ./ipv6.csv ",
				"wantedList": ["cn", " us "],
				"onlyIPType": "ipv4"
			}`,
			want: &geolite2_country_csv_in{
				CountryCodeFile: "./country.csv",
				IPv4File:        "./ipv4.csv",
				IPv6File:        "./ipv6.csv",
				Want:            map[string]bool{"CN": true, "US": true},
				OnlyIPType:      lib.IPv4,
			},
		},
		{
			name:    "malformed json",
			data:    `{`,
			wantErr: errAs[*json.SyntaxError](),
		},
		{
			name:    "argument of wrong type",
			data:    `{"country": ["./country.csv"]}`,
			wantErr: errAs[*json.UnmarshalTypeError](),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGeoLite2CountryCSVInFromBytes(lib.ActionAdd, []byte(tt.data))
			if assertErr(t, err, tt.wantErr) {
				assertConverterErr(t, err, TypeGeoLite2CountryCSVIn, lib.ActionAdd)
				if got != nil {
					t.Fatalf("got converter %+v along with error %v", got, err)
				}
				return
			}

			tt.want.Type, tt.want.Description, tt.want.Action = TypeGeoLite2CountryCSVIn, DescGeoLite2CountryCSVIn, lib.ActionAdd
			assertCountryCSVIn(t, got.(*geolite2_country_csv_in), tt.want)
		})
	}
}

func TestGeoLite2CountryCSVIn_getCountryCode(t *testing.T) {
	const header = "geoname_id,locale_code,continent_code,continent_name,country_iso_code,country_name,is_in_european_union\n"

	dir := t.TempDir()
	file := func(name, content string) string {
		return writeTestFile(t, dir, name, []byte(content))
	}

	server := newTestServer(t, map[string][]byte{"/country.csv": []byte(countryCodeCSV)})

	countryCodes := map[string]string{"1814991": "CN", "2921044": "DE", "6252001": "US"}

	tests := []struct {
		name    string
		file    string
		want    map[string]string
		wantErr *errCheck
	}{
		{name: "local file", file: file("country.csv", countryCodeCSV), want: countryCodes},
		{name: "remote file", file: server.URL + "/country.csv", want: countryCodes},
		{name: "remote file with upper case scheme", file: upperCaseScheme(server) + "/country.csv", want: countryCodes},
		{
			name: "fields are trimmed and country codes are upper-cased",
			file: file("spaces.csv", header+" 1814991 ,en,AS,Asia, cn ,China,0\n"),
			want: map[string]string{"1814991": "CN"},
		},
		{
			name: "locations without id or country code are skipped",
			file: file("blanks.csv", header+",en,AS,Asia,CN,China,0\n 6255148 ,en,EU,Europe, ,,0\n6252001,en,NA,\"North America\",US,\"United States\",0\n"),
			want: map[string]string{"6252001": "US"},
		},
		{
			name: "fields after the country code are optional",
			file: file("five-columns.csv", "geoname_id,locale_code,continent_code,continent_name,country_iso_code\n1814991,en,AS,Asia,CN\n"),
			want: map[string]string{"1814991": "CN"},
		},

		{name: "missing file", file: filepath.Join(dir, "missing.csv"), wantErr: errIs(os.ErrNotExist)},
		{name: "missing remote file", file: server.URL + "/missing.csv", wantErr: errContains("404 Not Found")},
		{name: "invalid https url", file: invalidHTTPSURL, wantErr: errAs[*url.Error]()},
		{name: "empty file", file: file("empty.csv", ""), wantErr: errContains("empty country code file: " + filepath.Join(dir, "empty.csv"))},
		{name: "header only", file: file("header.csv", header), wantErr: errContains("invalid country code data")},
		{
			name:    "no location of a country",
			file:    file("continents.csv", header+"6255147,en,AS,Asia,,,0\n6255148,en,EU,Europe,,,0\n"),
			wantErr: errContains("invalid country code data"),
		},
		{
			name:    "location without country code field",
			file:    file("four-columns.csv", "geoname_id,locale_code,continent_code,continent_name\n1814991,en,AS,Asia\n"),
			wantErr: errContains("invalid record: [1814991 en AS Asia]"),
		},
		{
			name:    "location with missing fields",
			file:    file("missing-fields.csv", header+"1814991,en,AS,Asia,CN\n"),
			wantErr: errIs(csv.ErrFieldCount),
		},
		{
			name:    "malformed csv",
			file:    file("malformed.csv", header+"1814991,en,AS,Asia,CN,Chi\"na,0\n"),
			wantErr: errIs(csv.ErrBareQuote),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_country_csv_in{Type: TypeGeoLite2CountryCSVIn, Action: lib.ActionAdd, CountryCodeFile: tt.file}

			got, err := g.getCountryCode()
			if assertErr(t, err, tt.wantErr) {
				if got != nil {
					t.Fatalf("got country codes %v along with error %v", got, err)
				}
				return
			}
			if len(got) != len(tt.want) || !maps.Equal(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGeoLite2CountryCSVIn_process(t *testing.T) {
	const header = "network,geoname_id,registered_country_geoname_id,represented_country_geoname_id,is_anonymous_proxy,is_satellite_provider,is_anycast\n"

	dir := t.TempDir()
	ipv4File := writeTestFile(t, dir, "ipv4.csv", []byte(countryIPv4CSV))
	ipv6File := writeTestFile(t, dir, "ipv6.csv", []byte(countryIPv6CSV))

	file := func(name, content string) string {
		return writeTestFile(t, dir, name, []byte(content))
	}

	server := newTestServer(t, map[string][]byte{"/ipv4.csv": []byte(countryIPv4CSV)})

	countryCodes := map[string]string{"1814991": "CN", "2921044": "DE", "6252001": "US"}

	tests := []struct {
		name string
		file string
		// countryCodes default to the ones of the country code fixture.
		countryCodes map[string]string
		wantedList   map[string]bool
		// entries are the entries generated before the file is processed.
		entries map[string]*lib.Entry
		want    map[string][]string
		wantErr *errCheck
	}{
		{name: "ipv4 file", file: ipv4File, want: countryListsIPv4},
		{name: "ipv6 file", file: ipv6File, want: countryListsIPv6},
		{name: "remote file", file: server.URL + "/ipv4.csv", want: countryListsIPv4},
		{name: "remote file with upper case scheme", file: upperCaseScheme(server) + "/ipv4.csv", want: countryListsIPv4},
		{
			name:       "wanted list",
			file:       ipv4File,
			wantedList: map[string]bool{"CN": true, "DE": true, "JP": true},
			want:       map[string][]string{"CN": {"1.0.1.0/24", "1.0.2.0/23"}, "DE": {"5.1.0.0/16"}},
		},
		{
			name:       "nothing wanted is found",
			file:       ipv4File,
			wantedList: map[string]bool{"JP": true},
			want:       map[string][]string{},
		},
		{
			name:    "merges into existing entries",
			file:    ipv6File,
			entries: map[string]*lib.Entry{"CN": newTestEntry(t, "CN", "1.0.1.0/24"), "JP": newTestEntry(t, "JP", "1.0.16.0/20")},
			want: map[string][]string{
				"CN": {"1.0.1.0/24", "2001:250::/35"},
				"JP": {"1.0.16.0/20"},
				"US": {"2001:4860::/32"},
			},
		},
		{
			name: "fields are trimmed",
			file: file("spaces.csv", header+" 1.0.1.0/24 , 1814991 ,,,0,0,\n 2001:250::/35 ,, 1814991 ,,0,0,\n 5.1.0.0/16 ,,, 2921044 ,0,0,\n"),
			want: map[string][]string{"CN": {"1.0.1.0/24", "2001:250::/35"}, "DE": {"5.1.0.0/16"}},
		},
		{
			name: "upper case network",
			file: file("upper-case.csv", header+"2001:DB8:ABCD::/48,1814991,,,0,0,\n"),
			want: map[string][]string{"CN": {"2001:db8:abcd::/48"}},
		},
		{
			name: "fields after the represented country are optional",
			file: file("four-columns.csv", "network,geoname_id,registered_country_geoname_id,represented_country_geoname_id\n1.0.1.0/24,1814991,,\n"),
			want: map[string][]string{"CN": {"1.0.1.0/24"}},
		},
		{name: "header only", file: file("header.csv", header), want: map[string][]string{}},
		{name: "empty file", file: file("empty.csv", ""), want: map[string][]string{}},

		{name: "missing file", file: filepath.Join(dir, "missing.csv"), wantErr: errIs(os.ErrNotExist)},
		{name: "missing remote file", file: server.URL + "/missing.csv", wantErr: errContains("404 Not Found")},
		{name: "invalid https url", file: invalidHTTPSURL, wantErr: errAs[*url.Error]()},
		{name: "no country codes", file: ipv4File, countryCodes: map[string]string{}, wantErr: errContains("invalid country code data")},
		{
			name:    "network without represented country field",
			file:    file("three-columns.csv", "network,geoname_id,registered_country_geoname_id\n1.0.1.0/24,1814991,1814991\n"),
			wantErr: errContains("invalid record: [1.0.1.0/24 1814991 1814991]"),
		},
		{
			name:    "network with missing fields",
			file:    file("missing-fields.csv", header+"1.0.1.0/24,1814991,1814991\n"),
			wantErr: errIs(csv.ErrFieldCount),
		},
		{
			name:    "malformed csv",
			file:    file("malformed.csv", header+"1.0.1.0/24,18\"14991,,,0,0,\n"),
			wantErr: errIs(csv.ErrBareQuote),
		},
		{
			name:    "invalid network",
			file:    file("invalid-network.csv", header+"1.0.1.0/33,1814991,,,0,0,\n"),
			wantErr: errIs(lib.ErrInvalidCIDR),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_country_csv_in{Type: TypeGeoLite2CountryCSVIn, Action: lib.ActionAdd, Want: tt.wantedList}
			ccMap := tt.countryCodes
			if ccMap == nil {
				ccMap = countryCodes
			}
			entries := tt.entries
			if entries == nil {
				entries = make(map[string]*lib.Entry)
			}

			err := g.process(tt.file, ccMap, entries)
			if assertErr(t, err, tt.wantErr) {
				return
			}
			assertEntries(t, entries, tt.want)
		})
	}

	t.Run("without entries to fill", func(t *testing.T) {
		g := &geolite2_country_csv_in{Type: TypeGeoLite2CountryCSVIn, Action: lib.ActionAdd}
		assertErr(t, g.process(ipv4File, countryCodes, nil), nil)
	})
}

func TestGeoLite2CountryCSVIn_Input(t *testing.T) {
	dir := t.TempDir()
	countryCodeFile := writeTestFile(t, dir, "country.csv", []byte(countryCodeCSV))
	ipv4File := writeTestFile(t, dir, "ipv4.csv", []byte(countryIPv4CSV))
	ipv6File := writeTestFile(t, dir, "ipv6.csv", []byte(countryIPv6CSV))
	missingFile := filepath.Join(dir, "missing.csv")

	server := newTestServer(t, map[string][]byte{
		"/country.csv": []byte(countryCodeCSV),
		"/ipv4.csv":    []byte(countryIPv4CSV),
		"/ipv6.csv":    []byte(countryIPv6CSV),
	})

	// The countries that have both IPv4 and IPv6 networks.
	dualStack := map[string]bool{"CN": true, "US": true}
	china := map[string]bool{"CN": true}
	chinaLists := map[string][]string{"CN": {"1.0.0.0/22", "2001:250::/35", "2001:da8::/32"}}

	newInput := func(g geolite2_country_csv_in) lib.InputConverter {
		g.Type = TypeGeoLite2CountryCSVIn
		if g.CountryCodeFile == "" {
			g.CountryCodeFile = countryCodeFile
		}
		return &g
	}

	runInputCases(t, []inputCase{
		{
			name:  "add all countries",
			input: newInput(geolite2_country_csv_in{Action: lib.ActionAdd, IPv4File: ipv4File, IPv6File: ipv6File}),
			want:  countryLists,
		},
		{
			name:  "add ipv4 file only",
			input: newInput(geolite2_country_csv_in{Action: lib.ActionAdd, IPv4File: ipv4File}),
			want:  countryListsIPv4,
		},
		{
			name:  "add ipv6 file only",
			input: newInput(geolite2_country_csv_in{Action: lib.ActionAdd, IPv6File: ipv6File}),
			want:  countryListsIPv6,
		},
		{
			name: "add remote files",
			input: newInput(geolite2_country_csv_in{
				Action: lib.ActionAdd, CountryCodeFile: server.URL + "/country.csv",
				IPv4File: server.URL + "/ipv4.csv", IPv6File: server.URL + "/ipv6.csv",
			}),
			want: countryLists,
		},
		{
			name: "add wanted list",
			input: newInput(geolite2_country_csv_in{
				Action: lib.ActionAdd, IPv4File: ipv4File, IPv6File: ipv6File,
				Want: map[string]bool{"CN": true, "DE": true, "JP": true},
			}),
			want: map[string][]string{"CN": countryLists["CN"], "DE": countryLists["DE"]},
		},
		{
			name:  "add ipv4 only",
			input: newInput(geolite2_country_csv_in{Action: lib.ActionAdd, IPv4File: ipv4File, IPv6File: ipv6File, Want: dualStack, OnlyIPType: lib.IPv4}),
			want:  map[string][]string{"CN": countryListsIPv4["CN"], "US": countryListsIPv4["US"]},
		},
		{
			name:  "add ipv6 only",
			input: newInput(geolite2_country_csv_in{Action: lib.ActionAdd, IPv4File: ipv4File, IPv6File: ipv6File, Want: dualStack, OnlyIPType: lib.IPv6}),
			want:  countryListsIPv6,
		},
		{
			name:      "add to existing entries",
			input:     newInput(geolite2_country_csv_in{Action: lib.ActionAdd, IPv4File: ipv4File, IPv6File: ipv6File, Want: china}),
			container: newTestContainer(t, map[string][]string{"CN": {"1.0.1.0/24", "1.0.8.0/21"}, "JP": {"1.0.16.0/20"}}),
			want: map[string][]string{
				"CN": {"1.0.1.0/24", "1.0.2.0/23", "1.0.8.0/21", "2001:250::/35"},
				"JP": {"1.0.16.0/20"},
			},
		},
		{
			name:      "remove",
			input:     newInput(geolite2_country_csv_in{Action: lib.ActionRemove, IPv4File: ipv4File, IPv6File: ipv6File, Want: china}),
			container: newTestContainer(t, chinaLists),
			want:      map[string][]string{"CN": {"1.0.0.0/24", "2001:da8::/32"}},
		},
		{
			name:      "remove ipv4 only",
			input:     newInput(geolite2_country_csv_in{Action: lib.ActionRemove, IPv4File: ipv4File, IPv6File: ipv6File, Want: china, OnlyIPType: lib.IPv4}),
			container: newTestContainer(t, chinaLists),
			want:      map[string][]string{"CN": {"1.0.0.0/24", "2001:250::/35", "2001:da8::/32"}},
		},
		{
			name:      "remove ipv6 only",
			input:     newInput(geolite2_country_csv_in{Action: lib.ActionRemove, IPv4File: ipv4File, IPv6File: ipv6File, Want: china, OnlyIPType: lib.IPv6}),
			container: newTestContainer(t, chinaLists),
			want:      map[string][]string{"CN": {"1.0.0.0/22", "2001:da8::/32"}},
		},
		{
			name:    "remove from missing entry",
			input:   newInput(geolite2_country_csv_in{Action: lib.ActionRemove, IPv4File: ipv4File, IPv6File: ipv6File, Want: china}),
			wantErr: errContains("entry CN not found"),
		},
		{
			name:    "unknown action",
			input:   newInput(geolite2_country_csv_in{Action: lib.Action("unknown"), IPv4File: ipv4File, IPv6File: ipv6File}),
			wantErr: errIs(lib.ErrUnknownAction),
		},
		{
			name:      "container fails to add",
			input:     newInput(geolite2_country_csv_in{Action: lib.ActionAdd, IPv4File: ipv4File, IPv6File: ipv6File}),
			container: &mockContainer{Container: lib.NewContainer(), addErr: errAdd},
			wantErr:   errIs(errAdd),
		},
		{
			name:      "container fails to remove",
			input:     newInput(geolite2_country_csv_in{Action: lib.ActionRemove, IPv4File: ipv4File, IPv6File: ipv6File}),
			container: &mockContainer{Container: lib.NewContainer(), removeErr: errRemove},
			wantErr:   errIs(errRemove),
		},
		{
			name:    "missing country code file",
			input:   newInput(geolite2_country_csv_in{Action: lib.ActionAdd, CountryCodeFile: missingFile, IPv4File: ipv4File, IPv6File: ipv6File}),
			wantErr: errIs(os.ErrNotExist),
		},
		{
			name:    "missing ipv4 file",
			input:   newInput(geolite2_country_csv_in{Action: lib.ActionAdd, IPv4File: missingFile, IPv6File: ipv6File}),
			wantErr: errIs(os.ErrNotExist),
		},
		{
			name:    "missing ipv6 file",
			input:   newInput(geolite2_country_csv_in{Action: lib.ActionAdd, IPv4File: ipv4File, IPv6File: missingFile}),
			wantErr: errIs(os.ErrNotExist),
		},
		{
			name: "no entry is generated",
			input: newInput(geolite2_country_csv_in{
				Action: lib.ActionAdd, IPv4File: ipv4File, IPv6File: ipv6File,
				Want: map[string]bool{"JP": true},
			}),
			wantErr: errContains("no entry is generated"),
		},
		{
			name:    "no file",
			input:   newInput(geolite2_country_csv_in{Action: lib.ActionAdd}),
			wantErr: errContains("no entry is generated"),
		},
	})
}
