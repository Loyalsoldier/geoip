package maxmind

import (
	"encoding/json"
	"maps"
	"reflect"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

// assertMMDBIn compares the fields of two MMDB input converters.
func assertMMDBIn(t *testing.T, got, want *geolite2_country_mmdb_in) {
	t.Helper()
	if got.Type != want.Type || got.Action != want.Action || got.Description != want.Description ||
		got.URI != want.URI || !maps.Equal(got.Want, want.Want) || got.OnlyIPType != want.OnlyIPType {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestGetDefaultMMDBInputURI(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		want string
	}{
		{"geolite2", TypeGeoLite2CountryMMDBIn, defaultGeoLite2CountryMMDBFile},
		{"dbip", TypeDBIPCountryMMDBIn, defaultDBIPCountryMMDBFile},
		{"ipinfo", TypeIPInfoCountryMMDBIn, defaultIPInfoCountryMMDBFile},
		{"csv format has no mmdb file", TypeGeoLite2CountryCSVIn, ""},
		{"unknown format", "unknown", ""},
		{"empty format", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDefaultMMDBInputURI(tt.typ); got != tt.want {
				t.Fatalf("getDefaultMMDBInputURI(%q) = %q, want %q", tt.typ, got, tt.want)
			}
		})
	}
}

func TestNewGeoLite2CountryMMDBIn(t *testing.T) {
	tests := []struct {
		name    string
		typ     string
		opts    []lib.InputOption
		want    *geolite2_country_mmdb_in
		wantErr *errCheck
	}{
		{
			name: "geolite2 defaults",
			typ:  TypeGeoLite2CountryMMDBIn,
			want: &geolite2_country_mmdb_in{URI: defaultGeoLite2CountryMMDBFile},
		},
		{
			name: "dbip defaults",
			typ:  TypeDBIPCountryMMDBIn,
			want: &geolite2_country_mmdb_in{URI: defaultDBIPCountryMMDBFile},
		},
		{
			name: "ipinfo defaults",
			typ:  TypeIPInfoCountryMMDBIn,
			want: &geolite2_country_mmdb_in{URI: defaultIPInfoCountryMMDBFile},
		},
		{
			name: "nil option is ignored",
			typ:  TypeGeoLite2CountryMMDBIn,
			opts: []lib.InputOption{nil},
			want: &geolite2_country_mmdb_in{URI: defaultGeoLite2CountryMMDBFile},
		},
		{
			name: "all options",
			typ:  TypeDBIPCountryMMDBIn,
			opts: []lib.InputOption{
				WithURI(" ./custom.mmdb "),
				WithInputWantedList([]string{" cn ", "", "US", "cn"}),
				WithInputOnlyIPType(lib.IPv6),
			},
			want: &geolite2_country_mmdb_in{
				URI:        "./custom.mmdb",
				Want:       map[string]bool{"CN": true, "US": true},
				OnlyIPType: lib.IPv6,
			},
		},
		{
			name: "blank uri falls back to default",
			typ:  TypeIPInfoCountryMMDBIn,
			opts: []lib.InputOption{WithURI("./custom.mmdb"), WithURI("  ")},
			want: &geolite2_country_mmdb_in{URI: defaultIPInfoCountryMMDBFile},
		},
		{
			name: "blank wanted list wants everything",
			typ:  TypeGeoLite2CountryMMDBIn,
			opts: []lib.InputOption{WithInputWantedList([]string{"", "  "})},
			want: &geolite2_country_mmdb_in{URI: defaultGeoLite2CountryMMDBFile},
		},
		{
			name: "unknown format with uri",
			typ:  "unknown",
			opts: []lib.InputOption{WithURI("./custom.mmdb")},
			want: &geolite2_country_mmdb_in{URI: "./custom.mmdb"},
		},
		{
			name:    "unknown format without uri",
			typ:     "unknown",
			wantErr: errContains("uri must be specified"),
		},
		{
			name:    "unknown format with blank uri",
			typ:     "unknown",
			opts:    []lib.InputOption{WithURI(" ")},
			wantErr: errContains("uri must be specified"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newGeoLite2CountryMMDBIn(tt.typ, "desc", lib.ActionRemove, tt.opts...)
			if assertErr(t, err, tt.wantErr) {
				assertConverterErr(t, err, tt.typ, lib.ActionRemove)
				if got != nil {
					t.Fatalf("got converter %+v along with error %v", got, err)
				}
				return
			}

			tt.want.Type, tt.want.Description, tt.want.Action = tt.typ, "desc", lib.ActionRemove
			assertMMDBIn(t, got.(*geolite2_country_mmdb_in), tt.want)
		})
	}
}

func TestNewGeoLite2CountryMMDBInFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		typ     string
		data    string
		want    *geolite2_country_mmdb_in
		wantErr *errCheck
	}{
		{
			name: "no data",
			typ:  TypeGeoLite2CountryMMDBIn,
			want: &geolite2_country_mmdb_in{URI: defaultGeoLite2CountryMMDBFile},
		},
		{
			name: "empty object",
			typ:  TypeDBIPCountryMMDBIn,
			data: `{}`,
			want: &geolite2_country_mmdb_in{URI: defaultDBIPCountryMMDBFile},
		},
		{
			name: "all arguments",
			typ:  TypeIPInfoCountryMMDBIn,
			data: `{"uri": " https://example.com/country.mmdb ", "wantedList": ["cn", " us "], "onlyIPType": "ipv4"}`,
			want: &geolite2_country_mmdb_in{
				URI:        "https://example.com/country.mmdb",
				Want:       map[string]bool{"CN": true, "US": true},
				OnlyIPType: lib.IPv4,
			},
		},
		{
			name: "unknown arguments are ignored",
			typ:  TypeGeoLite2CountryMMDBIn,
			data: `{"name": "cn", "uri": "./custom.mmdb"}`,
			want: &geolite2_country_mmdb_in{URI: "./custom.mmdb"},
		},
		{
			name:    "malformed json",
			typ:     TypeGeoLite2CountryMMDBIn,
			data:    `{`,
			wantErr: errAs[*json.SyntaxError](),
		},
		{
			name:    "argument of wrong type",
			typ:     TypeGeoLite2CountryMMDBIn,
			data:    `{"wantedList": "cn"}`,
			wantErr: errAs[*json.UnmarshalTypeError](),
		},
		{
			name:    "unknown format without uri",
			typ:     "unknown",
			data:    `{}`,
			wantErr: errContains("uri must be specified"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newGeoLite2CountryMMDBInFromBytes(tt.typ, "desc", lib.ActionAdd, []byte(tt.data))
			if assertErr(t, err, tt.wantErr) {
				assertConverterErr(t, err, tt.typ, lib.ActionAdd)
				if got != nil {
					t.Fatalf("got converter %+v along with error %v", got, err)
				}
				return
			}

			tt.want.Type, tt.want.Description, tt.want.Action = tt.typ, "desc", lib.ActionAdd
			assertMMDBIn(t, got.(*geolite2_country_mmdb_in), tt.want)
		})
	}
}

// TestInputOptions verifies that each option is applied to the formats it is
// valid for, and that the options shared by several formats are ignored by
// the remaining formats of this package.
func TestInputOptions(t *testing.T) {
	wantedList := WithInputWantedList([]string{" cn ", "", "US", "cn"})
	want := map[string]bool{"CN": true, "US": true}

	tests := []struct {
		name   string
		option lib.InputOption
		target lib.InputConverter
		want   lib.InputConverter
	}{
		{
			name:   "uri of mmdb",
			option: WithURI(" ./custom.mmdb "),
			target: &geolite2_country_mmdb_in{Type: TypeGeoLite2CountryMMDBIn},
			want:   &geolite2_country_mmdb_in{Type: TypeGeoLite2CountryMMDBIn, URI: "./custom.mmdb"},
		},
		{
			name:   "blank uri of mmdb",
			option: WithURI(""),
			target: &geolite2_country_mmdb_in{Type: TypeDBIPCountryMMDBIn, URI: "./custom.mmdb"},
			want:   &geolite2_country_mmdb_in{Type: TypeDBIPCountryMMDBIn, URI: defaultDBIPCountryMMDBFile},
		},
		{
			name:   "wanted list of mmdb",
			option: wantedList,
			target: &geolite2_country_mmdb_in{},
			want:   &geolite2_country_mmdb_in{Want: want},
		},
		{
			name:   "wanted list of country csv",
			option: wantedList,
			target: &geolite2_country_csv_in{},
			want:   &geolite2_country_csv_in{Want: want},
		},
		{
			name:   "wanted list is ignored by asn csv",
			option: wantedList,
			target: &geolite2_asn_csv_in{},
			want:   &geolite2_asn_csv_in{},
		},
		{
			name:   "only ip type of mmdb",
			option: WithInputOnlyIPType(lib.IPv4),
			target: &geolite2_country_mmdb_in{},
			want:   &geolite2_country_mmdb_in{OnlyIPType: lib.IPv4},
		},
		{
			name:   "only ip type of country csv",
			option: WithInputOnlyIPType(lib.IPv6),
			target: &geolite2_country_csv_in{},
			want:   &geolite2_country_csv_in{OnlyIPType: lib.IPv6},
		},
		{
			name:   "only ip type of asn csv",
			option: WithInputOnlyIPType(lib.IPv4),
			target: &geolite2_asn_csv_in{},
			want:   &geolite2_asn_csv_in{OnlyIPType: lib.IPv4},
		},
		{
			name:   "ipv4 file of country csv",
			option: WithIPv4File(" ./ipv4.csv "),
			target: &geolite2_country_csv_in{},
			want:   &geolite2_country_csv_in{IPv4File: "./ipv4.csv"},
		},
		{
			name:   "ipv4 file of asn csv",
			option: WithIPv4File(" ./ipv4.csv "),
			target: &geolite2_asn_csv_in{},
			want:   &geolite2_asn_csv_in{IPv4File: "./ipv4.csv"},
		},
		{
			name:   "ipv4 file is ignored by mmdb",
			option: WithIPv4File("./ipv4.csv"),
			target: &geolite2_country_mmdb_in{},
			want:   &geolite2_country_mmdb_in{},
		},
		{
			name:   "ipv6 file of country csv",
			option: WithIPv6File(" ./ipv6.csv "),
			target: &geolite2_country_csv_in{},
			want:   &geolite2_country_csv_in{IPv6File: "./ipv6.csv"},
		},
		{
			name:   "ipv6 file of asn csv",
			option: WithIPv6File(" ./ipv6.csv "),
			target: &geolite2_asn_csv_in{},
			want:   &geolite2_asn_csv_in{IPv6File: "./ipv6.csv"},
		},
		{
			name:   "ipv6 file is ignored by mmdb",
			option: WithIPv6File("./ipv6.csv"),
			target: &geolite2_country_mmdb_in{},
			want:   &geolite2_country_mmdb_in{},
		},
		{
			name:   "country code file of country csv",
			option: WithCountryCodeFile(" ./country.csv "),
			target: &geolite2_country_csv_in{},
			want:   &geolite2_country_csv_in{CountryCodeFile: "./country.csv"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.option(tt.target)
			if !reflect.DeepEqual(tt.target, tt.want) {
				t.Fatalf("got %+v, want %+v", tt.target, tt.want)
			}
		})
	}
}
