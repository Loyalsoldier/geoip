package maxmind

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/oschwald/maxminddb-golang/v2"
)

func TestGeoLite2CountryMMDBIn_generateEntries(t *testing.T) {
	geoLite2MMDB := buildMMDB(t, countryMMDBRecords(geoLite2Languages)...)
	dbipMMDB := buildMMDB(t, countryMMDBRecords(dbipLanguages)...)
	ipinfoMMDB := buildMMDB(t, ipinfoMMDBRecords()...)
	// Each format expects records of its own schema, which a string never is.
	mismatchedMMDB := buildMMDB(t, mmdbRecord{"1.0.1.0/24", mmdbtype.String("CN")})

	tests := []struct {
		name       string
		typ        string
		content    []byte
		wantedList map[string]bool
		// entries are the entries generated before the database is processed.
		entries map[string]*lib.Entry
		want    map[string][]string
		wantErr *errCheck
	}{
		{name: "geolite2", typ: TypeGeoLite2CountryMMDBIn, content: geoLite2MMDB, want: countryLists},
		{name: "dbip", typ: TypeDBIPCountryMMDBIn, content: dbipMMDB, want: countryLists},
		{name: "ipinfo", typ: TypeIPInfoCountryMMDBIn, content: ipinfoMMDB, want: countryLists},
		{
			name:       "geolite2 wanted list",
			typ:        TypeGeoLite2CountryMMDBIn,
			content:    geoLite2MMDB,
			wantedList: map[string]bool{"CN": true, "DE": true, "JP": true},
			want:       map[string][]string{"CN": countryLists["CN"], "DE": countryLists["DE"]},
		},
		{
			name:       "ipinfo wanted list",
			typ:        TypeIPInfoCountryMMDBIn,
			content:    ipinfoMMDB,
			wantedList: map[string]bool{"US": true},
			want:       map[string][]string{"US": countryLists["US"]},
		},
		{
			name:       "nothing wanted is found",
			typ:        TypeDBIPCountryMMDBIn,
			content:    dbipMMDB,
			wantedList: map[string]bool{"JP": true},
			want:       map[string][]string{},
		},
		{
			name:    "merges into existing entries",
			typ:     TypeGeoLite2CountryMMDBIn,
			content: geoLite2MMDB,
			entries: map[string]*lib.Entry{"CN": newTestEntry(t, "CN", "1.0.8.0/21"), "JP": newTestEntry(t, "JP", "1.0.16.0/20")},
			want: map[string][]string{
				"CN": {"1.0.1.0/24", "1.0.2.0/23", "1.0.8.0/21", "2001:250::/35"},
				"DE": countryLists["DE"],
				"JP": {"1.0.16.0/20"},
				"US": countryLists["US"],
			},
		},
		{
			name: "ipinfo networks without country code are skipped",
			typ:  TypeIPInfoCountryMMDBIn,
			content: buildMMDB(t,
				mmdbRecord{"1.0.1.0/24", ipinfoData(ipinfoPlace("AS", "Asia", "CN", "China"), "AS4134")},
				mmdbRecord{"1.0.2.0/23", mmdbtype.Map{"asn": mmdbtype.String("AS4837")}},
				mmdbRecord{"1.0.8.0/21", ipinfoData(ipinfoPlace("AS", "Asia", " ", "China"), "AS4134")},
			),
			want: map[string][]string{"CN": {"1.0.1.0/24"}},
		},
		{
			name: "ipv4 only database",
			typ:  TypeGeoLite2CountryMMDBIn,
			content: buildMMDBOfVersion(t, 4,
				mmdbRecord{"1.0.1.0/24", mmdbtype.Map{"country": countryData("CN", 1814991, false, nil)}},
				mmdbRecord{"8.8.8.0/24", mmdbtype.Map{"country": countryData("US", 6252001, false, nil)}},
			),
			want: map[string][]string{"CN": {"1.0.1.0/24"}, "US": {"8.8.8.0/24"}},
		},
		{name: "database without networks", typ: TypeGeoLite2CountryMMDBIn, content: buildMMDB(t), want: map[string][]string{}},

		{name: "no content", typ: TypeGeoLite2CountryMMDBIn, content: nil, wantErr: errAs[maxminddb.InvalidDatabaseError]()},
		{name: "invalid content", typ: TypeGeoLite2CountryMMDBIn, content: []byte("not an mmdb database"), wantErr: errAs[maxminddb.InvalidDatabaseError]()},
		{name: "corrupted database", typ: TypeGeoLite2CountryMMDBIn, content: corruptMMDB(t, geoLite2MMDB), wantErr: errAs[maxminddb.InvalidDatabaseError]()},
		{name: "geolite2 schema mismatch", typ: TypeGeoLite2CountryMMDBIn, content: mismatchedMMDB, wantErr: errAs[maxminddb.UnmarshalTypeError]()},
		{name: "dbip schema mismatch", typ: TypeDBIPCountryMMDBIn, content: mismatchedMMDB, wantErr: errAs[maxminddb.UnmarshalTypeError]()},
		{name: "ipinfo schema mismatch", typ: TypeIPInfoCountryMMDBIn, content: mismatchedMMDB, wantErr: errAs[maxminddb.UnmarshalTypeError]()},
		{name: "unknown format", typ: "unknown", content: geoLite2MMDB, wantErr: errIs(lib.ErrNotSupportedFormat)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_country_mmdb_in{Type: tt.typ, Action: lib.ActionAdd, Want: tt.wantedList}
			entries := tt.entries
			if entries == nil {
				entries = make(map[string]*lib.Entry)
			}

			err := g.generateEntries(tt.content, entries)
			if assertErr(t, err, tt.wantErr) {
				return
			}
			assertEntries(t, entries, tt.want)
		})
	}
}

func TestGeoLite2CountryMMDBIn_Input(t *testing.T) {
	countryMMDB := buildMMDB(t, countryMMDBRecords(geoLite2Languages)...)
	ipinfoMMDB := buildMMDB(t, ipinfoMMDBRecords()...)

	dir := t.TempDir()
	countryFile := writeTestFile(t, dir, "country.mmdb", countryMMDB)
	ipinfoFile := writeTestFile(t, dir, "ipinfo.mmdb", ipinfoMMDB)
	invalidFile := writeTestFile(t, dir, "invalid.mmdb", []byte("not an mmdb database"))

	server := newTestServer(t, map[string][]byte{
		"/country.mmdb": countryMMDB,
		"/invalid.mmdb": []byte("not an mmdb database"),
	})

	china := map[string]bool{"CN": true}
	chinaLists := map[string][]string{"CN": {"1.0.0.0/22", "2001:250::/35", "2001:da8::/32"}}

	const (
		geoLite2 = TypeGeoLite2CountryMMDBIn
		dbip     = TypeDBIPCountryMMDBIn
		ipinfo   = TypeIPInfoCountryMMDBIn
	)

	runInputCases(t, []inputCase{
		{
			name:  "add geolite2",
			input: &geolite2_country_mmdb_in{Type: geoLite2, Action: lib.ActionAdd, URI: countryFile},
			want:  countryLists,
		},
		{
			name:  "add dbip",
			input: &geolite2_country_mmdb_in{Type: dbip, Action: lib.ActionAdd, URI: countryFile},
			want:  countryLists,
		},
		{
			name:  "add ipinfo",
			input: &geolite2_country_mmdb_in{Type: ipinfo, Action: lib.ActionAdd, URI: ipinfoFile},
			want:  countryLists,
		},
		{
			name:  "add remote file",
			input: &geolite2_country_mmdb_in{Type: geoLite2, Action: lib.ActionAdd, URI: server.URL + "/country.mmdb"},
			want:  countryLists,
		},
		{
			name:  "add remote file with upper case scheme",
			input: &geolite2_country_mmdb_in{Type: dbip, Action: lib.ActionAdd, URI: upperCaseScheme(server) + "/country.mmdb"},
			want:  countryLists,
		},
		{
			name: "add wanted list",
			input: &geolite2_country_mmdb_in{
				Type: geoLite2, Action: lib.ActionAdd, URI: countryFile,
				Want: map[string]bool{"CN": true, "DE": true, "JP": true},
			},
			want: map[string][]string{"CN": countryLists["CN"], "DE": countryLists["DE"]},
		},
		{
			name:  "add ipv4 only",
			input: &geolite2_country_mmdb_in{Type: geoLite2, Action: lib.ActionAdd, URI: countryFile, OnlyIPType: lib.IPv4},
			want:  countryListsIPv4,
		},
		{
			// DE has no IPv6 network, so that there is no list of it.
			name:  "add ipv6 only",
			input: &geolite2_country_mmdb_in{Type: ipinfo, Action: lib.ActionAdd, URI: ipinfoFile, OnlyIPType: lib.IPv6},
			want:  countryListsIPv6,
		},
		{
			name:      "add to existing entries",
			input:     &geolite2_country_mmdb_in{Type: geoLite2, Action: lib.ActionAdd, URI: countryFile, Want: china},
			container: newTestContainer(t, map[string][]string{"CN": {"1.0.1.0/24", "1.0.8.0/21"}, "JP": {"1.0.16.0/20"}}),
			want: map[string][]string{
				"CN": {"1.0.1.0/24", "1.0.2.0/23", "1.0.8.0/21", "2001:250::/35"},
				"JP": {"1.0.16.0/20"},
			},
		},
		{
			name:      "remove",
			input:     &geolite2_country_mmdb_in{Type: geoLite2, Action: lib.ActionRemove, URI: countryFile, Want: china},
			container: newTestContainer(t, chinaLists),
			want:      map[string][]string{"CN": {"1.0.0.0/24", "2001:da8::/32"}},
		},
		{
			name:      "remove ipv4 only",
			input:     &geolite2_country_mmdb_in{Type: dbip, Action: lib.ActionRemove, URI: countryFile, Want: china, OnlyIPType: lib.IPv4},
			container: newTestContainer(t, chinaLists),
			want:      map[string][]string{"CN": {"1.0.0.0/24", "2001:250::/35", "2001:da8::/32"}},
		},
		{
			name:      "remove ipv6 only",
			input:     &geolite2_country_mmdb_in{Type: ipinfo, Action: lib.ActionRemove, URI: ipinfoFile, Want: china, OnlyIPType: lib.IPv6},
			container: newTestContainer(t, chinaLists),
			want:      map[string][]string{"CN": {"1.0.0.0/22", "2001:da8::/32"}},
		},
		{
			name:    "remove from missing entry",
			input:   &geolite2_country_mmdb_in{Type: geoLite2, Action: lib.ActionRemove, URI: countryFile, Want: china},
			wantErr: errContains("entry CN not found"),
		},
		{
			name:    "unknown action",
			input:   &geolite2_country_mmdb_in{Type: geoLite2, Action: lib.Action("unknown"), URI: countryFile},
			wantErr: errIs(lib.ErrUnknownAction),
		},
		{
			name:      "container fails to add",
			input:     &geolite2_country_mmdb_in{Type: geoLite2, Action: lib.ActionAdd, URI: countryFile},
			container: &mockContainer{Container: lib.NewContainer(), addErr: errAdd},
			wantErr:   errIs(errAdd),
		},
		{
			name:      "container fails to remove",
			input:     &geolite2_country_mmdb_in{Type: geoLite2, Action: lib.ActionRemove, URI: countryFile},
			container: &mockContainer{Container: lib.NewContainer(), removeErr: errRemove},
			wantErr:   errIs(errRemove),
		},
		{
			name:    "missing file",
			input:   &geolite2_country_mmdb_in{Type: geoLite2, Action: lib.ActionAdd, URI: filepath.Join(dir, "missing.mmdb")},
			wantErr: errIs(os.ErrNotExist),
		},
		{
			name:    "missing remote file",
			input:   &geolite2_country_mmdb_in{Type: geoLite2, Action: lib.ActionAdd, URI: server.URL + "/missing.mmdb"},
			wantErr: errContains("404 Not Found"),
		},
		{
			name:    "invalid https url",
			input:   &geolite2_country_mmdb_in{Type: geoLite2, Action: lib.ActionAdd, URI: invalidHTTPSURL},
			wantErr: errAs[*url.Error](),
		},
		{
			name:    "invalid file",
			input:   &geolite2_country_mmdb_in{Type: geoLite2, Action: lib.ActionAdd, URI: invalidFile},
			wantErr: errAs[maxminddb.InvalidDatabaseError](),
		},
		{
			name:    "invalid remote file",
			input:   &geolite2_country_mmdb_in{Type: geoLite2, Action: lib.ActionAdd, URI: server.URL + "/invalid.mmdb"},
			wantErr: errAs[maxminddb.InvalidDatabaseError](),
		},
		{
			name:    "unknown format",
			input:   &geolite2_country_mmdb_in{Type: "unknown", Action: lib.ActionAdd, URI: countryFile},
			wantErr: errIs(lib.ErrNotSupportedFormat),
		},
		{
			name:    "no entry is generated",
			input:   &geolite2_country_mmdb_in{Type: geoLite2, Action: lib.ActionAdd, URI: countryFile, Want: map[string]bool{"JP": true}},
			wantErr: errContains("no entry is generated"),
		},
	})
}
