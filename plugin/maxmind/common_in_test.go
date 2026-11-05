package maxmind

import (
	"reflect"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestGetDefaultMMDBInputURI(t *testing.T) {
	tests := []struct {
		name  string
		iType string
		want  string
	}{
		{"GeoLite2", TypeGeoLite2CountryMMDBIn, defaultGeoLite2CountryMMDBFile},
		{"DBIP", TypeDBIPCountryMMDBIn, defaultDBIPCountryMMDBFile},
		{"IPInfo", TypeIPInfoCountryMMDBIn, defaultIPInfoCountryMMDBFile},
		{"Unknown", "unknown", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDefaultMMDBInputURI(tt.iType); got != tt.want {
				t.Errorf("getDefaultMMDBInputURI(%q) = %q, want %q", tt.iType, got, tt.want)
			}
		})
	}
}

func TestNewGeoLite2CountryMMDBIn(t *testing.T) {
	tests := []struct {
		name    string
		iType   string
		opts    []lib.InputOption
		want    *geolite2_country_mmdb_in
		wantErr bool
	}{
		{
			name:  "Default URI",
			iType: TypeGeoLite2CountryMMDBIn,
			want: &geolite2_country_mmdb_in{
				Type: TypeGeoLite2CountryMMDBIn, Action: lib.ActionAdd, Description: "desc",
				URI: defaultGeoLite2CountryMMDBFile,
			},
		},
		{
			name:  "Nil option and all options",
			iType: TypeDBIPCountryMMDBIn,
			opts: []lib.InputOption{
				nil,
				WithURI("  custom.mmdb  "),
				WithInputWantedList([]string{" cn ", "", "us"}),
				WithInputOnlyIPType(lib.IPv4),
			},
			want: &geolite2_country_mmdb_in{
				Type: TypeDBIPCountryMMDBIn, Action: lib.ActionAdd, Description: "desc",
				URI: "custom.mmdb", Want: map[string]bool{"CN": true, "US": true}, OnlyIPType: lib.IPv4,
			},
		},
		{
			name:  "Empty URI falls back to default",
			iType: TypeIPInfoCountryMMDBIn,
			opts:  []lib.InputOption{WithURI("   ")},
			want: &geolite2_country_mmdb_in{
				Type: TypeIPInfoCountryMMDBIn, Action: lib.ActionAdd, Description: "desc",
				URI: defaultIPInfoCountryMMDBFile,
			},
		},
		{
			name:    "Unknown type without URI",
			iType:   "unknown",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newGeoLite2CountryMMDBIn(tt.iType, "desc", lib.ActionAdd, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNewMMDBInFromBytes(t *testing.T) {
	type ctor func(lib.Action, []byte) (lib.InputConverter, error)
	ctors := []struct {
		name       string
		fn         ctor
		iType      string
		desc       string
		defaultURI string
	}{
		{"GeoLite2", NewGeoLite2CountryMMDBInFromBytes, TypeGeoLite2CountryMMDBIn, DescGeoLite2CountryMMDBIn, defaultGeoLite2CountryMMDBFile},
		{"DBIP", NewDBIPCountryMMDBInFromBytes, TypeDBIPCountryMMDBIn, DescDBIPCountryMMDBIn, defaultDBIPCountryMMDBFile},
		{"IPInfo", NewIPInfoCountryMMDBInFromBytes, TypeIPInfoCountryMMDBIn, DescIPInfoCountryMMDBIn, defaultIPInfoCountryMMDBFile},
	}
	for _, c := range ctors {
		tests := []struct {
			name    string
			data    []byte
			want    *geolite2_country_mmdb_in
			wantErr bool
		}{
			{
				name: "Empty data",
				want: &geolite2_country_mmdb_in{Type: c.iType, Action: lib.ActionAdd, Description: c.desc, URI: c.defaultURI, Want: map[string]bool{}},
			},
			{
				name: "Full data",
				data: []byte(`{"uri":"a.mmdb","wantedList":["cn","US"],"onlyIPType":"ipv6"}`),
				want: &geolite2_country_mmdb_in{
					Type: c.iType, Action: lib.ActionAdd, Description: c.desc, URI: "a.mmdb",
					Want: map[string]bool{"CN": true, "US": true}, OnlyIPType: lib.IPv6,
				},
			},
			{name: "Invalid JSON", data: []byte(`{`), wantErr: true},
		}
		for _, tt := range tests {
			t.Run(c.name+"/"+tt.name, func(t *testing.T) {
				got, err := c.fn(lib.ActionAdd, tt.data)
				if (err != nil) != tt.wantErr {
					t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
				}
				if tt.wantErr {
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("got %+v, want %+v", got, tt.want)
				}
			})
		}
	}
}

func TestNewMMDBIn(t *testing.T) {
	tests := []struct {
		name  string
		fn    func(lib.Action, ...lib.InputOption) (lib.InputConverter, error)
		iType string
		desc  string
	}{
		{"GeoLite2", NewGeoLite2CountryMMDBIn, TypeGeoLite2CountryMMDBIn, DescGeoLite2CountryMMDBIn},
		{"DBIP", NewDBIPCountryMMDBIn, TypeDBIPCountryMMDBIn, DescDBIPCountryMMDBIn},
		{"IPInfo", NewIPInfoCountryMMDBIn, TypeIPInfoCountryMMDBIn, DescIPInfoCountryMMDBIn},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fn(lib.ActionRemove, WithURI("x.mmdb"))
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if got.GetType() != tt.iType || got.GetAction() != lib.ActionRemove || got.GetDescription() != tt.desc {
				t.Errorf("got type=%q action=%q desc=%q", got.GetType(), got.GetAction(), got.GetDescription())
			}
			if got.(*geolite2_country_mmdb_in).URI != "x.mmdb" {
				t.Errorf("URI = %q, want x.mmdb", got.(*geolite2_country_mmdb_in).URI)
			}
		})
	}
}

func TestInputOptionsOnDifferentTypes(t *testing.T) {
	tests := []struct {
		name string
		conv lib.InputConverter
		opts []lib.InputOption
		want lib.InputConverter
	}{
		{
			name: "Country CSV",
			conv: &geolite2_country_csv_in{},
			opts: []lib.InputOption{
				WithInputWantedList([]string{"cn", " "}),
				WithInputOnlyIPType(lib.IPv4),
				WithIPv4File(" v4.csv "),
				WithIPv6File(" v6.csv "),
			},
			want: &geolite2_country_csv_in{
				Want: map[string]bool{"CN": true}, OnlyIPType: lib.IPv4, IPv4File: "v4.csv", IPv6File: "v6.csv",
			},
		},
		{
			name: "ASN CSV",
			conv: &geolite2_asn_csv_in{},
			opts: []lib.InputOption{
				WithInputWantedList([]string{"cn"}), // ignored for ASN CSV
				WithInputOnlyIPType(lib.IPv6),
				WithIPv4File(" v4.csv "),
				WithIPv6File(" v6.csv "),
			},
			want: &geolite2_asn_csv_in{OnlyIPType: lib.IPv6, IPv4File: "v4.csv", IPv6File: "v6.csv"},
		},
		{
			name: "MMDB ignores file options",
			conv: &geolite2_country_mmdb_in{},
			opts: []lib.InputOption{WithIPv4File("v4.csv"), WithIPv6File("v6.csv")},
			want: &geolite2_country_mmdb_in{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, opt := range tt.opts {
				opt(tt.conv)
			}
			if !reflect.DeepEqual(tt.conv, tt.want) {
				t.Errorf("got %+v, want %+v", tt.conv, tt.want)
			}
		})
	}
}
