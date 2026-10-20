package maxmind

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestGetDefaultMMDBInputURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		iType string
		want  string
	}{
		{name: "GeoLite2 country", iType: TypeGeoLite2CountryMMDBIn, want: filepath.Join("geolite2", "GeoLite2-Country.mmdb")},
		{name: "DB-IP country", iType: TypeDBIPCountryMMDBIn, want: filepath.Join("db-ip", "dbip-country-lite.mmdb")},
		{name: "IPInfo country", iType: TypeIPInfoCountryMMDBIn, want: filepath.Join("ipinfo", "country.mmdb")},
		{name: "unknown type", iType: "unknown", want: ""},
		{name: "empty type", iType: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := getDefaultMMDBInputURI(tt.iType); got != tt.want {
				t.Errorf("getDefaultMMDBInputURI(%q) = %q, want %q", tt.iType, got, tt.want)
			}
		})
	}
}

func TestNewGeoLite2CountryMMDBInInternal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		iType       string
		iDesc       string
		action      lib.Action
		opts        []lib.InputOption
		want        *geolite2_country_mmdb_in
		wantErr     bool
		errContains string
	}{
		{
			name:   "GeoLite2 default URI",
			iType:  TypeGeoLite2CountryMMDBIn,
			iDesc:  DescGeoLite2CountryMMDBIn,
			action: lib.ActionAdd,
			want: &geolite2_country_mmdb_in{
				Type:        TypeGeoLite2CountryMMDBIn,
				Action:      lib.ActionAdd,
				Description: DescGeoLite2CountryMMDBIn,
				URI:         defaultGeoLite2CountryMMDBFile,
			},
		},
		{
			name:   "DB-IP default URI",
			iType:  TypeDBIPCountryMMDBIn,
			iDesc:  DescDBIPCountryMMDBIn,
			action: lib.ActionRemove,
			want: &geolite2_country_mmdb_in{
				Type:        TypeDBIPCountryMMDBIn,
				Action:      lib.ActionRemove,
				Description: DescDBIPCountryMMDBIn,
				URI:         defaultDBIPCountryMMDBFile,
			},
		},
		{
			name:   "IPInfo default URI",
			iType:  TypeIPInfoCountryMMDBIn,
			iDesc:  DescIPInfoCountryMMDBIn,
			action: lib.ActionAdd,
			want: &geolite2_country_mmdb_in{
				Type:        TypeIPInfoCountryMMDBIn,
				Action:      lib.ActionAdd,
				Description: DescIPInfoCountryMMDBIn,
				URI:         defaultIPInfoCountryMMDBFile,
			},
		},
		{
			name:   "nil options are ignored and others are applied",
			iType:  TypeGeoLite2CountryMMDBIn,
			iDesc:  DescGeoLite2CountryMMDBIn,
			action: lib.ActionAdd,
			opts: []lib.InputOption{
				nil,
				WithURI("  ./custom.mmdb  "),
				nil,
				WithInputWantedList([]string{" cn ", "Us", "", "  "}),
				WithInputOnlyIPType(lib.IPv4),
			},
			want: &geolite2_country_mmdb_in{
				Type:        TypeGeoLite2CountryMMDBIn,
				Action:      lib.ActionAdd,
				Description: DescGeoLite2CountryMMDBIn,
				URI:         "./custom.mmdb",
				Want:        map[string]bool{"CN": true, "US": true},
				OnlyIPType:  lib.IPv4,
			},
		},
		{
			name:   "later URI option overrides former one",
			iType:  TypeDBIPCountryMMDBIn,
			iDesc:  DescDBIPCountryMMDBIn,
			action: lib.ActionAdd,
			opts:   []lib.InputOption{WithURI("./first.mmdb"), WithURI("   ")},
			want: &geolite2_country_mmdb_in{
				Type:        TypeDBIPCountryMMDBIn,
				Action:      lib.ActionAdd,
				Description: DescDBIPCountryMMDBIn,
				URI:         defaultDBIPCountryMMDBFile,
			},
		},
		{
			name:   "unknown type with URI",
			iType:  "unknown",
			iDesc:  "desc",
			action: lib.ActionAdd,
			opts:   []lib.InputOption{WithURI("./custom.mmdb")},
			want: &geolite2_country_mmdb_in{
				Type:        "unknown",
				Action:      lib.ActionAdd,
				Description: "desc",
				URI:         "./custom.mmdb",
			},
		},
		{
			name:        "unknown type without URI",
			iType:       "unknown",
			iDesc:       "desc",
			action:      lib.ActionAdd,
			wantErr:     true,
			errContains: "[type unknown | action add] uri must be specified",
		},
		{
			name:        "unknown type with blank URI",
			iType:       "unknown",
			iDesc:       "desc",
			action:      lib.ActionRemove,
			opts:        []lib.InputOption{WithURI("   ")},
			wantErr:     true,
			errContains: "[type unknown | action remove] uri must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := newGeoLite2CountryMMDBIn(tt.iType, tt.iDesc, tt.action, tt.opts...)
			assertError(t, err, tt.wantErr, tt.errContains, nil)
			if tt.wantErr {
				if got != nil {
					t.Fatalf("expected nil converter on error, got %#v", got)
				}
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("newGeoLite2CountryMMDBIn() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestNewGeoLite2CountryMMDBInFromBytesInternal(t *testing.T) {
	t.Parallel()

	types := []struct {
		iType      string
		iDesc      string
		defaultURI string
	}{
		{TypeGeoLite2CountryMMDBIn, DescGeoLite2CountryMMDBIn, defaultGeoLite2CountryMMDBFile},
		{TypeDBIPCountryMMDBIn, DescDBIPCountryMMDBIn, defaultDBIPCountryMMDBFile},
		{TypeIPInfoCountryMMDBIn, DescIPInfoCountryMMDBIn, defaultIPInfoCountryMMDBFile},
	}

	tests := []struct {
		name           string
		action         lib.Action
		data           []byte
		wantURI        string // empty means the default URI of the type
		wantWant       map[string]bool
		wantOnlyIPType lib.IPType
		wantErr        bool
	}{
		{
			name:     "nil data",
			action:   lib.ActionAdd,
			data:     nil,
			wantWant: map[string]bool{},
		},
		{
			name:     "empty data",
			action:   lib.ActionAdd,
			data:     []byte{},
			wantWant: map[string]bool{},
		},
		{
			name:     "empty JSON object",
			action:   lib.ActionRemove,
			data:     []byte(`{}`),
			wantWant: map[string]bool{},
		},
		{
			name:     "blank URI falls back to default",
			action:   lib.ActionAdd,
			data:     []byte(`{"uri": "   "}`),
			wantWant: map[string]bool{},
		},
		{
			name:           "all args",
			action:         lib.ActionAdd,
			data:           []byte(`{"uri": " ./data/custom.mmdb ", "wantedList": [" cn ", "us", "", "   ", "CN"], "onlyIPType": "ipv6"}`),
			wantURI:        "./data/custom.mmdb",
			wantWant:       map[string]bool{"CN": true, "US": true},
			wantOnlyIPType: lib.IPv6,
		},
		{
			name:           "remote URI",
			action:         lib.ActionRemove,
			data:           []byte(`{"uri": "https://example.com/country.mmdb", "onlyIPType": "ipv4"}`),
			wantURI:        "https://example.com/country.mmdb",
			wantWant:       map[string]bool{},
			wantOnlyIPType: lib.IPv4,
		},
		{
			name:    "invalid JSON",
			action:  lib.ActionAdd,
			data:    []byte(`{"uri": `),
			wantErr: true,
		},
		{
			name:    "invalid wantedList type",
			action:  lib.ActionAdd,
			data:    []byte(`{"wantedList": "cn"}`),
			wantErr: true,
		},
		{
			name:    "invalid uri type",
			action:  lib.ActionRemove,
			data:    []byte(`{"uri": 123}`),
			wantErr: true,
		},
	}

	for _, typ := range types {
		for _, tt := range tests {
			t.Run(typ.iType+"/"+tt.name, func(t *testing.T) {
				t.Parallel()

				got, err := newGeoLite2CountryMMDBInFromBytes(typ.iType, typ.iDesc, tt.action, tt.data)
				assertError(t, err, tt.wantErr, "[type "+typ.iType+" | action "+string(tt.action)+"]", nil)
				if tt.wantErr {
					if got != nil {
						t.Fatalf("expected nil converter on error, got %#v", got)
					}
					return
				}

				wantURI := tt.wantURI
				if wantURI == "" {
					wantURI = typ.defaultURI
				}
				want := &geolite2_country_mmdb_in{
					Type:        typ.iType,
					Action:      tt.action,
					Description: typ.iDesc,
					URI:         wantURI,
					Want:        tt.wantWant,
					OnlyIPType:  tt.wantOnlyIPType,
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("newGeoLite2CountryMMDBInFromBytes() = %#v, want %#v", got, want)
				}
			})
		}
	}

	t.Run("unknown type without URI", func(t *testing.T) {
		t.Parallel()

		got, err := newGeoLite2CountryMMDBInFromBytes("unknown", "desc", lib.ActionAdd, nil)
		assertError(t, err, true, "uri must be specified", nil)
		if got != nil {
			t.Fatalf("expected nil converter on error, got %#v", got)
		}
	})
}

func TestWithURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		iType string
		uri   string
		want  string
	}{
		{name: "local path", iType: TypeGeoLite2CountryMMDBIn, uri: "./a.mmdb", want: "./a.mmdb"},
		{name: "trimmed path", iType: TypeGeoLite2CountryMMDBIn, uri: " \t./a.mmdb\n ", want: "./a.mmdb"},
		{name: "remote URL", iType: TypeDBIPCountryMMDBIn, uri: "https://example.com/a.mmdb", want: "https://example.com/a.mmdb"},
		{name: "empty GeoLite2", iType: TypeGeoLite2CountryMMDBIn, uri: "", want: defaultGeoLite2CountryMMDBFile},
		{name: "empty DB-IP", iType: TypeDBIPCountryMMDBIn, uri: "  ", want: defaultDBIPCountryMMDBFile},
		{name: "empty IPInfo", iType: TypeIPInfoCountryMMDBIn, uri: "", want: defaultIPInfoCountryMMDBFile},
		{name: "empty unknown type", iType: "unknown", uri: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := &geolite2_country_mmdb_in{Type: tt.iType, URI: "previous"}
			WithURI(tt.uri)(g)
			if g.URI != tt.want {
				t.Errorf("URI = %q, want %q", g.URI, tt.want)
			}
		})
	}
}

func TestWithInputWantedList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		lists []string
		want  map[string]bool
	}{
		{name: "nil list", lists: nil, want: map[string]bool{}},
		{name: "empty list", lists: []string{}, want: map[string]bool{}},
		{name: "only blank names", lists: []string{"", "  ", "\t"}, want: map[string]bool{}},
		{name: "normalized names", lists: []string{" cn ", "Us", "jp", "CN"}, want: map[string]bool{"CN": true, "US": true, "JP": true}},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/mmdb input", func(t *testing.T) {
			t.Parallel()

			g := &geolite2_country_mmdb_in{Want: map[string]bool{"OLD": true}}
			WithInputWantedList(tt.lists)(g)
			if !reflect.DeepEqual(g.Want, tt.want) {
				t.Errorf("Want = %v, want %v", g.Want, tt.want)
			}
		})

		t.Run(tt.name+"/country CSV input", func(t *testing.T) {
			t.Parallel()

			g := &geolite2_country_csv_in{Want: map[string]bool{"OLD": true}}
			WithInputWantedList(tt.lists)(g)
			if !reflect.DeepEqual(g.Want, tt.want) {
				t.Errorf("Want = %v, want %v", g.Want, tt.want)
			}
		})
	}

	t.Run("ASN CSV input is not affected", func(t *testing.T) {
		t.Parallel()

		g := &geolite2_asn_csv_in{Want: map[string][]string{"13335": {"AS13335"}}}
		WithInputWantedList([]string{"cn"})(g)
		want := map[string][]string{"13335": {"AS13335"}}
		if !reflect.DeepEqual(g.Want, want) {
			t.Errorf("Want = %v, want %v", g.Want, want)
		}
	})

	t.Run("foreign converter is not affected", func(t *testing.T) {
		t.Parallel()

		WithInputWantedList([]string{"cn"})(fakeInputConverter{})
	})
}

func TestWithInputOnlyIPType(t *testing.T) {
	t.Parallel()

	for _, ipType := range []lib.IPType{lib.IPv4, lib.IPv6, ""} {
		t.Run(string(ipType)+"/mmdb input", func(t *testing.T) {
			t.Parallel()

			g := &geolite2_country_mmdb_in{OnlyIPType: "previous"}
			WithInputOnlyIPType(ipType)(g)
			if g.OnlyIPType != ipType {
				t.Errorf("OnlyIPType = %q, want %q", g.OnlyIPType, ipType)
			}
		})

		t.Run(string(ipType)+"/country CSV input", func(t *testing.T) {
			t.Parallel()

			g := &geolite2_country_csv_in{OnlyIPType: "previous"}
			WithInputOnlyIPType(ipType)(g)
			if g.OnlyIPType != ipType {
				t.Errorf("OnlyIPType = %q, want %q", g.OnlyIPType, ipType)
			}
		})

		t.Run(string(ipType)+"/ASN CSV input", func(t *testing.T) {
			t.Parallel()

			g := &geolite2_asn_csv_in{OnlyIPType: "previous"}
			WithInputOnlyIPType(ipType)(g)
			if g.OnlyIPType != ipType {
				t.Errorf("OnlyIPType = %q, want %q", g.OnlyIPType, ipType)
			}
		})
	}

	t.Run("foreign converter is not affected", func(t *testing.T) {
		t.Parallel()

		WithInputOnlyIPType(lib.IPv4)(fakeInputConverter{})
	})
}

func TestWithIPFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file string
		want string
	}{
		{name: "local file", file: "./a.csv", want: "./a.csv"},
		{name: "trimmed file", file: "  ./a.csv\t", want: "./a.csv"},
		{name: "remote file", file: "https://example.com/a.csv", want: "https://example.com/a.csv"},
		{name: "empty file", file: "   ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/country CSV input", func(t *testing.T) {
			t.Parallel()

			g := &geolite2_country_csv_in{IPv4File: "old4", IPv6File: "old6"}
			WithIPv4File(tt.file)(g)
			if g.IPv4File != tt.want || g.IPv6File != "old6" {
				t.Fatalf("after WithIPv4File: IPv4File = %q, IPv6File = %q; want %q, %q", g.IPv4File, g.IPv6File, tt.want, "old6")
			}

			g = &geolite2_country_csv_in{IPv4File: "old4", IPv6File: "old6"}
			WithIPv6File(tt.file)(g)
			if g.IPv6File != tt.want || g.IPv4File != "old4" {
				t.Fatalf("after WithIPv6File: IPv4File = %q, IPv6File = %q; want %q, %q", g.IPv4File, g.IPv6File, "old4", tt.want)
			}
		})

		t.Run(tt.name+"/ASN CSV input", func(t *testing.T) {
			t.Parallel()

			g := &geolite2_asn_csv_in{IPv4File: "old4", IPv6File: "old6"}
			WithIPv4File(tt.file)(g)
			if g.IPv4File != tt.want || g.IPv6File != "old6" {
				t.Fatalf("after WithIPv4File: IPv4File = %q, IPv6File = %q; want %q, %q", g.IPv4File, g.IPv6File, tt.want, "old6")
			}

			g = &geolite2_asn_csv_in{IPv4File: "old4", IPv6File: "old6"}
			WithIPv6File(tt.file)(g)
			if g.IPv6File != tt.want || g.IPv4File != "old4" {
				t.Fatalf("after WithIPv6File: IPv4File = %q, IPv6File = %q; want %q, %q", g.IPv4File, g.IPv6File, "old4", tt.want)
			}
		})
	}

	t.Run("mmdb input is not affected", func(t *testing.T) {
		t.Parallel()

		g := &geolite2_country_mmdb_in{URI: "old"}
		WithIPv4File("./a.csv")(g)
		WithIPv6File("./b.csv")(g)
		want := &geolite2_country_mmdb_in{URI: "old"}
		if !reflect.DeepEqual(g, want) {
			t.Fatalf("converter = %#v, want %#v", g, want)
		}
	})

	t.Run("foreign converter is not affected", func(t *testing.T) {
		t.Parallel()

		WithIPv4File("./a.csv")(fakeInputConverter{})
		WithIPv6File("./b.csv")(fakeInputConverter{})
	})
}
