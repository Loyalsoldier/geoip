package maxmind

import (
	"encoding/json"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/oschwald/geoip2-golang/v2"
	"github.com/oschwald/maxminddb-golang/v2"
)

// assertMMDBOut compares the fields of two MMDB output converters.
func assertMMDBOut(t *testing.T, got, want *geolite2_country_mmdb_out) {
	t.Helper()
	if got.Type != want.Type || got.Action != want.Action || got.Description != want.Description ||
		got.OutputName != want.OutputName || got.OutputDir != want.OutputDir ||
		!slices.Equal(got.Want, want.Want) || !slices.Equal(got.Overwrite, want.Overwrite) ||
		!slices.Equal(got.Exclude, want.Exclude) || got.OnlyIPType != want.OnlyIPType ||
		got.SourceMMDBURI != want.SourceMMDBURI {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestDBIPHasData(t *testing.T) {
	names := dbipLanguageNames{Persian: "چین"}

	tests := []struct {
		name string
		data interface{ HasData() bool }
		want bool
	}{
		{"empty names", dbipLanguageNames{}, false},
		{"names in a geolite2 language", dbipLanguageNames{English: "China"}, true},
		{"names in persian", dbipLanguageNames{Persian: "چین"}, true},
		{"names in korean", dbipLanguageNames{Korean: "중국"}, true},

		{"empty continent", dbipContinent{}, false},
		{"continent with code", dbipContinent{Code: "AS"}, true},
		{"continent with names", dbipContinent{Names: names}, true},

		{"empty country record", dbipCountryRecord{}, false},
		{"country record with iso code", dbipCountryRecord{ISOCode: "CN"}, true},
		{"country record with names", dbipCountryRecord{Names: names}, true},

		{"empty represented country", dbipRepresentedCountry{}, false},
		{"represented country with type", dbipRepresentedCountry{Type: "military"}, true},
		{"represented country with names", dbipRepresentedCountry{Names: names}, true},

		{"empty country", dbipCountry{}, false},
		{"country with continent", dbipCountry{Continent: dbipContinent{Names: names}}, true},
		{"country with represented country", dbipCountry{RepresentedCountry: dbipRepresentedCountry{Names: names}}, true},
		{"country with country", dbipCountry{Country: dbipCountryRecord{Names: names}}, true},
		{"country with registered country", dbipCountry{RegisteredCountry: dbipCountryRecord{Names: names}}, true},
		{"country with traits", dbipCountry{Traits: geoip2.CountryTraits{IsAnycast: true}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.data.HasData(); got != tt.want {
				t.Fatalf("HasData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDefaultMMDBOutputDir(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		want string
	}{
		{"geolite2", TypeGeoLite2CountryMMDBOut, defaultMaxmindOutputDir},
		{"dbip", TypeDBIPCountryMMDBOut, defaultDBIPOutputDir},
		{"ipinfo", TypeIPInfoCountryMMDBOut, defaultIPInfoOutputDir},
		{"unknown format", "unknown", defaultMaxmindOutputDir},
		{"empty format", "", defaultMaxmindOutputDir},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDefaultMMDBOutputDir(tt.typ); got != tt.want {
				t.Fatalf("getDefaultMMDBOutputDir(%q) = %q, want %q", tt.typ, got, tt.want)
			}
		})
	}
}

func TestNewGeoLite2CountryMMDBOut(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		opts []lib.OutputOption
		want *geolite2_country_mmdb_out
	}{
		{
			name: "geolite2 defaults",
			typ:  TypeGeoLite2CountryMMDBOut,
			want: &geolite2_country_mmdb_out{OutputName: defaultGeoLite2CountryMMDBOutputName, OutputDir: defaultMaxmindOutputDir},
		},
		{
			name: "dbip defaults",
			typ:  TypeDBIPCountryMMDBOut,
			want: &geolite2_country_mmdb_out{OutputName: defaultGeoLite2CountryMMDBOutputName, OutputDir: defaultDBIPOutputDir},
		},
		{
			name: "ipinfo defaults",
			typ:  TypeIPInfoCountryMMDBOut,
			want: &geolite2_country_mmdb_out{OutputName: defaultGeoLite2CountryMMDBOutputName, OutputDir: defaultIPInfoOutputDir},
		},
		{
			name: "nil option is ignored",
			typ:  TypeGeoLite2CountryMMDBOut,
			opts: []lib.OutputOption{nil},
			want: &geolite2_country_mmdb_out{OutputName: defaultGeoLite2CountryMMDBOutputName, OutputDir: defaultMaxmindOutputDir},
		},
		{
			name: "all options",
			typ:  TypeGeoLite2CountryMMDBOut,
			opts: []lib.OutputOption{
				WithOutputName(" custom.mmdb "),
				WithOutputDir(" ./dir "),
				WithOutputWantedList([]string{"cn", "us"}),
				WithOverwriteList([]string{"google"}),
				WithOutputExcludedList([]string{"private"}),
				WithOutputOnlyIPType(lib.IPv4),
				WithSourceMMDBURI(" ./source.mmdb "),
			},
			want: &geolite2_country_mmdb_out{
				OutputName:    "custom.mmdb",
				OutputDir:     "./dir",
				Want:          []string{"cn", "us"},
				Overwrite:     []string{"google"},
				Exclude:       []string{"private"},
				OnlyIPType:    lib.IPv4,
				SourceMMDBURI: "./source.mmdb",
			},
		},
		{
			name: "blank name and dir fall back to defaults",
			typ:  TypeIPInfoCountryMMDBOut,
			opts: []lib.OutputOption{
				WithOutputName("custom.mmdb"), WithOutputDir("./dir"), WithSourceMMDBURI("./source.mmdb"),
				WithOutputName(" "), WithOutputDir(" "), WithSourceMMDBURI(" "),
			},
			want: &geolite2_country_mmdb_out{OutputName: defaultGeoLite2CountryMMDBOutputName, OutputDir: defaultIPInfoOutputDir},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newGeoLite2CountryMMDBOut(tt.typ, "desc", lib.ActionOutput, tt.opts...)
			assertErr(t, err, nil)

			tt.want.Type, tt.want.Description, tt.want.Action = tt.typ, "desc", lib.ActionOutput
			assertMMDBOut(t, got.(*geolite2_country_mmdb_out), tt.want)
		})
	}
}

func TestNewGeoLite2CountryMMDBOutFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		typ     string
		data    string
		want    *geolite2_country_mmdb_out
		wantErr *errCheck
	}{
		{
			name: "no data",
			typ:  TypeGeoLite2CountryMMDBOut,
			want: &geolite2_country_mmdb_out{OutputName: defaultGeoLite2CountryMMDBOutputName, OutputDir: defaultMaxmindOutputDir},
		},
		{
			name: "empty object",
			typ:  TypeDBIPCountryMMDBOut,
			data: `{}`,
			want: &geolite2_country_mmdb_out{OutputName: defaultGeoLite2CountryMMDBOutputName, OutputDir: defaultDBIPOutputDir},
		},
		{
			name: "all arguments",
			typ:  TypeIPInfoCountryMMDBOut,
			data: `{
				"outputName": " custom.mmdb ",
				"outputDir": " ./dir ",
				"wantedList": ["cn", "us"],
				"overwriteList": ["google"],
				"excludedList": ["private"],
				"onlyIPType": "ipv6",
				"sourceMMDBURI": " ./source.mmdb "
			}`,
			want: &geolite2_country_mmdb_out{
				OutputName:    "custom.mmdb",
				OutputDir:     "./dir",
				Want:          []string{"cn", "us"},
				Overwrite:     []string{"google"},
				Exclude:       []string{"private"},
				OnlyIPType:    lib.IPv6,
				SourceMMDBURI: "./source.mmdb",
			},
		},
		{
			name:    "malformed json",
			typ:     TypeGeoLite2CountryMMDBOut,
			data:    `{`,
			wantErr: errAs[*json.SyntaxError](),
		},
		{
			name:    "argument of wrong type",
			typ:     TypeGeoLite2CountryMMDBOut,
			data:    `{"outputName": 1}`,
			wantErr: errAs[*json.UnmarshalTypeError](),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newGeoLite2CountryMMDBOutFromBytes(tt.typ, "desc", lib.ActionOutput, []byte(tt.data))
			if assertErr(t, err, tt.wantErr) {
				assertConverterErr(t, err, tt.typ, lib.ActionOutput)
				if got != nil {
					t.Fatalf("got converter %+v along with error %v", got, err)
				}
				return
			}

			tt.want.Type, tt.want.Description, tt.want.Action = tt.typ, "desc", lib.ActionOutput
			assertMMDBOut(t, got.(*geolite2_country_mmdb_out), tt.want)
		})
	}
}

// TestGeoLite2CountryMMDBOut_GetExtraInfo verifies how the source MMDB is
// loaded and which lists get extra info out of it. What the extra info is made
// of is verified by TestGeoLite2CountryMMDBOut_Output, by means of the records
// of the generated databases.
func TestGeoLite2CountryMMDBOut_GetExtraInfo(t *testing.T) {
	countryMMDB := buildMMDB(t, countryMMDBRecords(dbipLanguages)...)
	ipinfoMMDB := buildMMDB(t, ipinfoMMDBRecords()...)
	// Each format expects records of its own schema, which a string never is.
	mismatchedMMDB := buildMMDB(t, mmdbRecord{"1.0.1.0/24", mmdbtype.String("CN")})

	dir := t.TempDir()
	countryFile := writeTestFile(t, dir, "country.mmdb", countryMMDB)
	ipinfoFile := writeTestFile(t, dir, "ipinfo.mmdb", ipinfoMMDB)
	mismatchedFile := writeTestFile(t, dir, "mismatched.mmdb", mismatchedMMDB)
	emptyFile := writeTestFile(t, dir, "empty.mmdb", buildMMDB(t))
	countrylessFile := writeTestFile(t, dir, "countryless.mmdb", buildMMDB(t,
		mmdbRecord{"1.0.1.0/24", mmdbtype.Map{"continent": continentData("AS", 6255147, nil)}},
	))
	codelessFile := writeTestFile(t, dir, "codeless.mmdb", buildMMDB(t,
		mmdbRecord{"1.0.1.0/24", ipinfoData(ipinfoPlace("AS", "Asia", "", "China"), "AS4134")},
		mmdbRecord{"1.0.2.0/23", ipinfoData(ipinfoPlace("AS", "Asia", " ", "China"), "AS4837")},
		mmdbRecord{"8.8.8.0/24", mmdbtype.Map{"asn": mmdbtype.String("AS15169")}},
	))
	corruptedFile := writeTestFile(t, dir, "corrupted.mmdb", corruptMMDB(t, countryMMDB))
	invalidFile := writeTestFile(t, dir, "invalid.mmdb", []byte("not an mmdb database"))

	server := newTestServer(t, map[string][]byte{
		"/country.mmdb": countryMMDB,
		"/invalid.mmdb": []byte("not an mmdb database"),
	})

	tests := []struct {
		name string
		typ  string
		uri  string
		// wantLists are the names of the lists that have extra info.
		wantLists []string
		wantErr   *errCheck
	}{
		{name: "no source", typ: TypeGeoLite2CountryMMDBOut, uri: ""},
		{name: "blank source", typ: TypeGeoLite2CountryMMDBOut, uri: "  "},
		{name: "no source of unknown format", typ: "unknown", uri: ""},

		{name: "geolite2", typ: TypeGeoLite2CountryMMDBOut, uri: countryFile, wantLists: []string{"CN", "DE", "US"}},
		{name: "dbip", typ: TypeDBIPCountryMMDBOut, uri: countryFile, wantLists: []string{"CN", "DE", "US"}},
		{name: "ipinfo", typ: TypeIPInfoCountryMMDBOut, uri: ipinfoFile, wantLists: []string{"CN", "DE", "US"}},
		{name: "remote source", typ: TypeGeoLite2CountryMMDBOut, uri: server.URL + "/country.mmdb", wantLists: []string{"CN", "DE", "US"}},
		{name: "remote source with upper case scheme", typ: TypeDBIPCountryMMDBOut, uri: upperCaseScheme(server) + "/country.mmdb", wantLists: []string{"CN", "DE", "US"}},

		{name: "missing file", typ: TypeGeoLite2CountryMMDBOut, uri: filepath.Join(dir, "missing.mmdb"), wantErr: errIs(os.ErrNotExist)},
		{name: "missing remote file", typ: TypeGeoLite2CountryMMDBOut, uri: server.URL + "/missing.mmdb", wantErr: errContains("404 Not Found")},
		{name: "invalid https url", typ: TypeGeoLite2CountryMMDBOut, uri: invalidHTTPSURL, wantErr: errAs[*url.Error]()},
		{name: "invalid file", typ: TypeGeoLite2CountryMMDBOut, uri: invalidFile, wantErr: errAs[maxminddb.InvalidDatabaseError]()},
		{name: "invalid remote file", typ: TypeGeoLite2CountryMMDBOut, uri: server.URL + "/invalid.mmdb", wantErr: errAs[maxminddb.InvalidDatabaseError]()},
		{name: "corrupted file", typ: TypeGeoLite2CountryMMDBOut, uri: corruptedFile, wantErr: errAs[maxminddb.InvalidDatabaseError]()},
		{name: "geolite2 schema mismatch", typ: TypeGeoLite2CountryMMDBOut, uri: mismatchedFile, wantErr: errAs[maxminddb.UnmarshalTypeError]()},
		{name: "dbip schema mismatch", typ: TypeDBIPCountryMMDBOut, uri: mismatchedFile, wantErr: errAs[maxminddb.UnmarshalTypeError]()},
		{name: "ipinfo schema mismatch", typ: TypeIPInfoCountryMMDBOut, uri: mismatchedFile, wantErr: errAs[maxminddb.UnmarshalTypeError]()},
		{name: "unknown format", typ: "unknown", uri: countryFile, wantErr: errIs(lib.ErrNotSupportedFormat)},
		{name: "source without networks", typ: TypeGeoLite2CountryMMDBOut, uri: emptyFile, wantErr: errContains("no extra info found in the source MMDB file: " + emptyFile)},
		{name: "geolite2 source without countries", typ: TypeGeoLite2CountryMMDBOut, uri: countrylessFile, wantErr: errContains("no extra info found")},
		{name: "dbip source without countries", typ: TypeDBIPCountryMMDBOut, uri: countrylessFile, wantErr: errContains("no extra info found")},
		{name: "ipinfo source without country codes", typ: TypeIPInfoCountryMMDBOut, uri: codelessFile, wantErr: errContains("no extra info found")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_country_mmdb_out{Type: tt.typ, Action: lib.ActionOutput, SourceMMDBURI: tt.uri}

			got, err := g.GetExtraInfo()
			if assertErr(t, err, tt.wantErr) {
				if got != nil {
					t.Fatalf("got extra info %v along with error %v", got, err)
				}
				return
			}

			gotLists := slices.Sorted(maps.Keys(got))
			if len(gotLists) != len(tt.wantLists) || !slices.Equal(gotLists, tt.wantLists) {
				t.Fatalf("lists with extra info mismatch: got %v, want %v", gotLists, tt.wantLists)
			}
			if tt.wantLists == nil && got != nil {
				t.Fatalf("got extra info %v without source", got)
			}
		})
	}
}
