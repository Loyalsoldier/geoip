package maxmind

import (
	"maps"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestGetDefaultMMDBInputURI(t *testing.T) {
	tests := []struct {
		iType string
		want  string
	}{
		{TypeGeoLite2CountryMMDBIn, defaultGeoLite2CountryMMDBFile},
		{TypeDBIPCountryMMDBIn, defaultDBIPCountryMMDBFile},
		{TypeIPInfoCountryMMDBIn, defaultIPInfoCountryMMDBFile},
		{"unknown", ""},
	}
	for _, tt := range tests {
		t.Run(tt.iType, func(t *testing.T) {
			if got := getDefaultMMDBInputURI(tt.iType); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewMMDBIn(t *testing.T) {
	tests := []struct {
		name     string
		newFn    func(lib.Action, ...lib.InputOption) (lib.InputConverter, error)
		opts     []lib.InputOption
		wantType string
		wantDesc string
		wantURI  string
		wantWant map[string]bool
		wantOnly lib.IPType
	}{
		{"maxmind default", NewGeoLite2CountryMMDBIn, nil, TypeGeoLite2CountryMMDBIn, DescGeoLite2CountryMMDBIn, defaultGeoLite2CountryMMDBFile, nil, ""},
		{"dbip default", NewDBIPCountryMMDBIn, []lib.InputOption{nil}, TypeDBIPCountryMMDBIn, DescDBIPCountryMMDBIn, defaultDBIPCountryMMDBFile, nil, ""},
		{"ipinfo default", NewIPInfoCountryMMDBIn, []lib.InputOption{WithURI("  ")}, TypeIPInfoCountryMMDBIn, DescIPInfoCountryMMDBIn, defaultIPInfoCountryMMDBFile, nil, ""},
		{
			"maxmind with options", NewGeoLite2CountryMMDBIn,
			[]lib.InputOption{WithURI(" a.mmdb "), WithInputWantedList([]string{" cn ", "", "Us"}), WithInputOnlyIPType(lib.IPv4), WithIPv4File("x"), WithIPv6File("y")},
			TypeGeoLite2CountryMMDBIn, DescGeoLite2CountryMMDBIn, "a.mmdb", map[string]bool{"CN": true, "US": true}, lib.IPv4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := tt.newFn(lib.ActionAdd, tt.opts...)
			assertErr(t, err, false)
			g := conv.(*geolite2_country_mmdb_in)
			if g.GetType() != tt.wantType || g.GetDescription() != tt.wantDesc || g.GetAction() != lib.ActionAdd {
				t.Errorf("got type=%q desc=%q action=%q", g.GetType(), g.GetDescription(), g.GetAction())
			}
			if g.URI != tt.wantURI {
				t.Errorf("URI = %q, want %q", g.URI, tt.wantURI)
			}
			if !maps.Equal(g.Want, tt.wantWant) {
				t.Errorf("Want = %v, want %v", g.Want, tt.wantWant)
			}
			if g.OnlyIPType != tt.wantOnly {
				t.Errorf("OnlyIPType = %q, want %q", g.OnlyIPType, tt.wantOnly)
			}
		})
	}
}

func TestNewMMDBIn_EmptyURI(t *testing.T) {
	_, err := newGeoLite2CountryMMDBIn("unknown", "desc", lib.ActionAdd)
	assertErr(t, err, true)
}

func TestNewMMDBInFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		newFn   func(lib.Action, []byte) (lib.InputConverter, error)
		data    string
		wantURI string
		want    map[string]bool
		only    lib.IPType
		wantErr bool
	}{
		{"maxmind empty", NewGeoLite2CountryMMDBInFromBytes, "", defaultGeoLite2CountryMMDBFile, nil, "", false},
		{"dbip uri", NewDBIPCountryMMDBInFromBytes, `{"uri":"d.mmdb","wantedList":["cn"],"onlyIPType":"ipv6"}`, "d.mmdb", map[string]bool{"CN": true}, lib.IPv6, false},
		{"ipinfo empty object", NewIPInfoCountryMMDBInFromBytes, `{}`, defaultIPInfoCountryMMDBFile, nil, "", false},
		{"invalid json", NewGeoLite2CountryMMDBInFromBytes, `{`, "", nil, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := tt.newFn(lib.ActionAdd, []byte(tt.data))
			assertErr(t, err, tt.wantErr)
			if tt.wantErr {
				return
			}
			g := conv.(*geolite2_country_mmdb_in)
			if g.URI != tt.wantURI || !maps.Equal(g.Want, tt.want) || g.OnlyIPType != tt.only {
				t.Errorf("got URI %q, wanted list %v, IP type %q; want %q, %v, %q", g.URI, g.Want, g.OnlyIPType, tt.wantURI, tt.want, tt.only)
			}
		})
	}
}

func TestConfigCreatorsRegistered(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{"maxmind mmdb", `{"input":[{"type":"maxmindMMDB","action":"add"}],"output":[{"type":"maxmindMMDB"}]}`},
		{"dbip mmdb", `{"input":[{"type":"dbipCountryMMDB","action":"add"}],"output":[{"type":"dbipCountryMMDB"}]}`},
		{"ipinfo mmdb", `{"input":[{"type":"ipinfoCountryMMDB","action":"add"}],"output":[{"type":"ipinfoCountryMMDB"}]}`},
		{"country csv", `{"input":[{"type":"maxmindGeoLite2CountryCSV","action":"add"}]}`},
		{"asn csv", `{"input":[{"type":"maxmindGeoLite2ASNCSV","action":"add"}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst, err := lib.NewInstance()
			assertErr(t, err, false)
			assertErr(t, inst.InitConfigFromBytes([]byte(tt.config)), false)
		})
	}
}
