package maxmind

import (
	"errors"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

func countryRec(iso string) mmdbtype.Map {
	return mmdbtype.Map{"country": mmdbtype.Map{"iso_code": mmdbtype.String(iso)}}
}

// geoLite2Fixture covers country / registered / represented / no-code records.
func geoLite2Fixture() []mmdbRecord {
	return []mmdbRecord{
		{"1.0.0.0/24", countryRec(" cn ")},
		{"8.8.8.0/24", mmdbtype.Map{"registered_country": mmdbtype.Map{"iso_code": mmdbtype.String("us")}}},
		{"9.9.9.0/24", mmdbtype.Map{"represented_country": mmdbtype.Map{"iso_code": mmdbtype.String("JP")}}},
		{"4.4.4.0/24", mmdbtype.Map{"continent": mmdbtype.Map{"code": mmdbtype.String("NA")}}},
		{"2001:4860::/32", countryRec("CN")},
		{"2400:cb00::/32", countryRec("US")},
	}
}

func ipinfoFixture() []mmdbRecord {
	return []mmdbRecord{
		{"1.0.0.0/24", mmdbtype.Map{"country_code": mmdbtype.String("cn"), "asn": mmdbtype.String("AS1")}},
		{"8.8.8.0/24", mmdbtype.Map{"country_code": mmdbtype.String("US")}},
		{"4.4.4.0/24", mmdbtype.Map{"country_code": mmdbtype.String("")}},
		{"2400:cb00::/32", mmdbtype.Map{"country_code": mmdbtype.String("US")}},
	}
}

func TestGetDefaultMMDBInputURI(t *testing.T) {
	tests := []struct {
		typ  string
		want string
	}{
		{TypeGeoLite2CountryMMDBIn, defaultGeoLite2CountryMMDBFile},
		{TypeDBIPCountryMMDBIn, defaultDBIPCountryMMDBFile},
		{TypeIPInfoCountryMMDBIn, defaultIPInfoCountryMMDBFile},
		{"unknown", ""},
	}
	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			if got := getDefaultMMDBInputURI(tt.typ); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewMMDBIn(t *testing.T) {
	type ctor func(lib.Action, ...lib.InputOption) (lib.InputConverter, error)
	tests := []struct {
		name     string
		ctor     ctor
		opts     []lib.InputOption
		wantType string
		wantDesc string
		wantURI  string
		wantWant map[string]bool
		wantOnly lib.IPType
	}{
		{"geolite2 default", NewGeoLite2CountryMMDBIn, nil, TypeGeoLite2CountryMMDBIn, DescGeoLite2CountryMMDBIn, defaultGeoLite2CountryMMDBFile, nil, ""},
		{"dbip default", NewDBIPCountryMMDBIn, nil, TypeDBIPCountryMMDBIn, DescDBIPCountryMMDBIn, defaultDBIPCountryMMDBFile, nil, ""},
		{"ipinfo default", NewIPInfoCountryMMDBIn, nil, TypeIPInfoCountryMMDBIn, DescIPInfoCountryMMDBIn, defaultIPInfoCountryMMDBFile, nil, ""},
		{
			"geolite2 with options and nil option", NewGeoLite2CountryMMDBIn,
			[]lib.InputOption{nil, WithURI("  /tmp/x.mmdb "), WithInputWantedList([]string{" cn", "", "us ", "  "}), WithInputOnlyIPType(lib.IPv4)},
			TypeGeoLite2CountryMMDBIn, DescGeoLite2CountryMMDBIn, "/tmp/x.mmdb", map[string]bool{"CN": true, "US": true}, lib.IPv4,
		},
		{
			"empty uri falls back to default", NewDBIPCountryMMDBIn,
			[]lib.InputOption{WithURI("   ")},
			TypeDBIPCountryMMDBIn, DescDBIPCountryMMDBIn, defaultDBIPCountryMMDBFile, nil, "",
		},
		{
			"file options ignored on mmdb input", NewIPInfoCountryMMDBIn,
			[]lib.InputOption{WithIPv4File("a"), WithIPv6File("b")},
			TypeIPInfoCountryMMDBIn, DescIPInfoCountryMMDBIn, defaultIPInfoCountryMMDBFile, nil, "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := tt.ctor(lib.ActionAdd, tt.opts...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			g := conv.(*geolite2_country_mmdb_in)
			if g.GetType() != tt.wantType || g.GetDescription() != tt.wantDesc || g.GetAction() != lib.ActionAdd {
				t.Errorf("got type=%q desc=%q action=%q", g.GetType(), g.GetDescription(), g.GetAction())
			}
			if g.URI != tt.wantURI {
				t.Errorf("URI: got %q, want %q", g.URI, tt.wantURI)
			}
			if len(g.Want) != len(tt.wantWant) {
				t.Errorf("Want: got %v, want %v", g.Want, tt.wantWant)
			}
			for k := range tt.wantWant {
				if !g.Want[k] {
					t.Errorf("Want missing %s: %v", k, g.Want)
				}
			}
			if g.OnlyIPType != tt.wantOnly {
				t.Errorf("OnlyIPType: got %q, want %q", g.OnlyIPType, tt.wantOnly)
			}
		})
	}
}

func TestNewMMDBInEmptyURI(t *testing.T) {
	tests := []struct {
		name string
		typ  string
	}{
		{"unknown type has no default", "unknownType"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newGeoLite2CountryMMDBIn(tt.typ, "d", lib.ActionAdd, WithURI(""))
			assertErrContains(t, err, "uri must be specified")
		})
	}
}

func TestNewMMDBInFromBytes(t *testing.T) {
	type ctor func(lib.Action, []byte) (lib.InputConverter, error)
	tests := []struct {
		name     string
		ctor     ctor
		data     string
		wantErr  bool
		wantType string
		wantURI  string
		wantWant []string
		wantOnly lib.IPType
	}{
		{"geolite2 empty data", NewGeoLite2CountryMMDBInFromBytes, "", false, TypeGeoLite2CountryMMDBIn, defaultGeoLite2CountryMMDBFile, nil, ""},
		{"dbip full", NewDBIPCountryMMDBInFromBytes, `{"uri":"x.mmdb","wantedList":["cn"],"onlyIPType":"ipv6"}`, false, TypeDBIPCountryMMDBIn, "x.mmdb", []string{"CN"}, lib.IPv6},
		{"ipinfo empty object", NewIPInfoCountryMMDBInFromBytes, `{}`, false, TypeIPInfoCountryMMDBIn, defaultIPInfoCountryMMDBFile, nil, ""},
		{"geolite2 invalid json", NewGeoLite2CountryMMDBInFromBytes, `{`, true, "", "", nil, ""},
		{"dbip invalid json", NewDBIPCountryMMDBInFromBytes, `[]`, true, "", "", nil, ""},
		{"ipinfo invalid json", NewIPInfoCountryMMDBInFromBytes, `{"uri":1}`, true, "", "", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := tt.ctor(lib.ActionRemove, []byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			g := conv.(*geolite2_country_mmdb_in)
			if g.Type != tt.wantType || g.URI != tt.wantURI || g.OnlyIPType != tt.wantOnly || g.Action != lib.ActionRemove {
				t.Errorf("got %+v", g)
			}
			if len(g.Want) != len(tt.wantWant) {
				t.Errorf("Want: got %v, want %v", g.Want, tt.wantWant)
			}
			for _, w := range tt.wantWant {
				if !g.Want[w] {
					t.Errorf("Want missing %s", w)
				}
			}
		})
	}
}

func TestMMDBInInput(t *testing.T) {
	geoPath := writeMMDB(t, "GeoLite2-Country", geoLite2Fixture())
	geoBytes := buildMMDB(t, "GeoLite2-Country", geoLite2Fixture())
	ipinfoPath := writeMMDB(t, "ipinfo", ipinfoFixture())
	emptyPath := writeMMDB(t, "empty", []mmdbRecord{{"4.4.4.0/24", mmdbtype.Map{"x": mmdbtype.String("y")}}})
	badDecodePath := writeMMDB(t, "bad", []mmdbRecord{{"1.0.0.0/24", mmdbtype.Map{"country": mmdbtype.String("not a map")}}})
	badIPInfoPath := writeMMDB(t, "bad", []mmdbRecord{{"1.0.0.0/24", mmdbtype.Map{"country_code": mmdbtype.Uint32(1)}}})
	notMMDB := writeFile(t, "bad.mmdb", []byte("this is not an mmdb"))

	srv := serveFiles(t, map[string][]byte{"/geo.mmdb": geoBytes})

	allGeo := map[string][]string{
		"CN": {"1.0.0.0/24", "2001:4860::/32"},
		"US": {"8.8.8.0/24", "2400:cb00::/32"},
		"JP": {"9.9.9.0/24"},
	}

	tests := []struct {
		name    string
		typ     string
		uri     string
		action  lib.Action
		want    []string
		only    lib.IPType
		pre     map[string][]string
		wantErr string
		expect  map[string][]string
	}{
		{name: "geolite2 all", typ: TypeGeoLite2CountryMMDBIn, uri: geoPath, action: lib.ActionAdd, expect: allGeo},
		{name: "dbip all", typ: TypeDBIPCountryMMDBIn, uri: geoPath, action: lib.ActionAdd, expect: allGeo},
		{name: "geolite2 remote http", typ: TypeGeoLite2CountryMMDBIn, uri: srv.URL + "/geo.mmdb", action: lib.ActionAdd, expect: allGeo},
		{name: "geolite2 remote 404", typ: TypeGeoLite2CountryMMDBIn, uri: srv.URL + "/missing.mmdb", action: lib.ActionAdd, wantErr: "404"},
		{name: "geolite2 wanted list", typ: TypeGeoLite2CountryMMDBIn, uri: geoPath, action: lib.ActionAdd, want: []string{"cn", "jp"},
			expect: map[string][]string{"CN": {"1.0.0.0/24", "2001:4860::/32"}, "JP": {"9.9.9.0/24"}}},
		{name: "geolite2 only ipv4", typ: TypeGeoLite2CountryMMDBIn, uri: geoPath, action: lib.ActionAdd, only: lib.IPv4,
			expect: map[string][]string{"CN": {"1.0.0.0/24"}, "US": {"8.8.8.0/24"}, "JP": {"9.9.9.0/24"}}},
		{name: "geolite2 only ipv6", typ: TypeGeoLite2CountryMMDBIn, uri: geoPath, action: lib.ActionAdd, only: lib.IPv6,
			expect: map[string][]string{"CN": {"2001:4860::/32"}, "US": {"2400:cb00::/32"}, "JP": nil}},
		{name: "geolite2 remove", typ: TypeGeoLite2CountryMMDBIn, uri: geoPath, action: lib.ActionRemove, want: []string{"CN"},
			pre:    map[string][]string{"CN": {"1.0.0.0/23", "2001:4860::/32", "2001:4861::/32"}},
			expect: map[string][]string{"CN": {"1.0.1.0/24", "2001:4861::/32"}}},
		{name: "geolite2 remove only ipv4", typ: TypeGeoLite2CountryMMDBIn, uri: geoPath, action: lib.ActionRemove, want: []string{"CN"}, only: lib.IPv4,
			pre:    map[string][]string{"CN": {"1.0.0.0/23", "2001:4860::/32"}},
			expect: map[string][]string{"CN": {"1.0.1.0/24", "2001:4860::/32"}}},
		{name: "geolite2 remove missing entry", typ: TypeGeoLite2CountryMMDBIn, uri: geoPath, action: lib.ActionRemove, want: []string{"CN"}, wantErr: "not found"},
		{name: "unknown action", typ: TypeGeoLite2CountryMMDBIn, uri: geoPath, action: lib.Action("bogus"), wantErr: lib.ErrUnknownAction.Error()},
		{name: "ipinfo all", typ: TypeIPInfoCountryMMDBIn, uri: ipinfoPath, action: lib.ActionAdd,
			expect: map[string][]string{"CN": {"1.0.0.0/24"}, "US": {"8.8.8.0/24", "2400:cb00::/32"}}},
		{name: "ipinfo wanted", typ: TypeIPInfoCountryMMDBIn, uri: ipinfoPath, action: lib.ActionAdd, want: []string{"us"},
			expect: map[string][]string{"US": {"8.8.8.0/24", "2400:cb00::/32"}}},
		{name: "missing file", typ: TypeGeoLite2CountryMMDBIn, uri: "/nonexistent/x.mmdb", action: lib.ActionAdd, wantErr: "no such file"},
		{name: "not an mmdb", typ: TypeGeoLite2CountryMMDBIn, uri: notMMDB, action: lib.ActionAdd, wantErr: "❌"},
		{name: "no entry generated", typ: TypeGeoLite2CountryMMDBIn, uri: emptyPath, action: lib.ActionAdd, wantErr: "no entry is generated"},
		{name: "wanted list matches nothing", typ: TypeGeoLite2CountryMMDBIn, uri: geoPath, action: lib.ActionAdd, want: []string{"ZZ"}, wantErr: "no entry is generated"},
		{name: "geolite2 decode error", typ: TypeGeoLite2CountryMMDBIn, uri: badDecodePath, action: lib.ActionAdd, wantErr: "❌"},
		{name: "ipinfo decode error", typ: TypeIPInfoCountryMMDBIn, uri: badIPInfoPath, action: lib.ActionAdd, wantErr: "❌"},
		{name: "unsupported type", typ: "unsupported", uri: geoPath, action: lib.ActionAdd, wantErr: lib.ErrNotSupportedFormat.Error()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := newGeoLite2CountryMMDBIn(tt.typ, "desc", tt.action,
				WithURI(tt.uri), WithInputWantedList(tt.want), WithInputOnlyIPType(tt.only))
			if err != nil {
				t.Fatalf("constructor: %v", err)
			}
			c := containerWith(t, tt.pre)
			got, err := conv.Input(c)
			if tt.wantErr != "" {
				assertErrContains(t, err, tt.wantErr)
				return
			}
			if err != nil {
				t.Fatalf("Input: %v", err)
			}
			assertEntries(t, got, tt.expect)
		})
	}
}

func TestMMDBInGenerateEntriesUnsupported(t *testing.T) {
	content := buildMMDB(t, "x", geoLite2Fixture())
	tests := []struct {
		name string
		typ  string
	}{
		{"empty type", ""},
		{"output-only type", "someOutputType"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_country_mmdb_in{Type: tt.typ}
			err := g.generateEntries(content, map[string]*lib.Entry{})
			if !errors.Is(err, lib.ErrNotSupportedFormat) {
				t.Fatalf("got %v, want ErrNotSupportedFormat", err)
			}
		})
	}
}

func TestMMDBInRegistered(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		data string
	}{
		{"geolite2", TypeGeoLite2CountryMMDBIn, `{"uri":"a"}`},
		{"dbip", TypeDBIPCountryMMDBIn, `{"uri":"a"}`},
		{"ipinfo", TypeIPInfoCountryMMDBIn, `{"uri":"a"}`},
		{"country csv", TypeGeoLite2CountryCSVIn, `{"country":"a","ipv4":"b"}`},
		{"asn csv", TypeGeoLite2ASNCSVIn, `{"ipv4":"b"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := `{"input":[{"type":"` + tt.typ + `","action":"add","args":` + tt.data + `}],"output":[]}`
			inst, err := lib.NewInstance()
			if err != nil {
				t.Fatal(err)
			}
			if err := inst.InitConfigFromBytes([]byte(cfg)); err != nil {
				t.Fatalf("InitConfigFromBytes: %v", err)
			}
			badCfg := `{"input":[{"type":"` + tt.typ + `","action":"add","args":"bad"}],"output":[]}`
			inst2, _ := lib.NewInstance()
			if err := inst2.InitConfigFromBytes([]byte(badCfg)); err == nil {
				t.Fatal("expected error for invalid args")
			}
		})
	}
}
