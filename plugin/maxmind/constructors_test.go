package maxmind

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestMMDBInputConstructors(t *testing.T) {
	tests := []struct {
		typ, desc, defaultURI string
		new                   func(lib.Action, ...lib.InputOption) (lib.InputConverter, error)
		fromJSON              func(lib.Action, []byte) (lib.InputConverter, error)
	}{
		{TypeGeoLite2CountryMMDBIn, DescGeoLite2CountryMMDBIn, filepath.Join("geolite2", "GeoLite2-Country.mmdb"), NewGeoLite2CountryMMDBIn, NewGeoLite2CountryMMDBInFromBytes},
		{TypeDBIPCountryMMDBIn, DescDBIPCountryMMDBIn, filepath.Join("db-ip", "dbip-country-lite.mmdb"), NewDBIPCountryMMDBIn, NewDBIPCountryMMDBInFromBytes},
		{TypeIPInfoCountryMMDBIn, DescIPInfoCountryMMDBIn, filepath.Join("ipinfo", "country.mmdb"), NewIPInfoCountryMMDBIn, NewIPInfoCountryMMDBInFromBytes},
	}
	// A blank URI option can be reused; the default belongs to the format, not the option.
	blankURI := WithURI("  ")
	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			for name, opts := range map[string][]lib.InputOption{
				"defaults": nil, "reset to defaults": {nil, WithURI("custom.mmdb"), blankURI},
			} {
				t.Run(name, func(t *testing.T) {
					in, err := tt.new(lib.ActionRemove, opts...)
					if err != nil {
						t.Fatal(err)
					}
					if in.GetType() != tt.typ || in.GetDescription() != tt.desc || in.GetAction() != lib.ActionRemove || in.(*geolite2_country_mmdb_in).URI != tt.defaultURI {
						t.Fatalf("unexpected defaults: %+v", in)
					}
				})
			}
			in, err := tt.fromJSON(lib.ActionAdd, []byte(`{"uri":" local.mmdb ","wantedList":[" de ","DE"," "],"onlyIPType":"ipv6"}`))
			if err != nil {
				t.Fatal(err)
			}
			g := in.(*geolite2_country_mmdb_in)
			if g.Type != tt.typ || g.Action != lib.ActionAdd || g.URI != "local.mmdb" || !maps.Equal(g.Want, map[string]bool{"DE": true}) || g.OnlyIPType != lib.IPv6 {
				t.Errorf("unexpected JSON configuration: %+v", g)
			}
			_, err = tt.fromJSON(lib.ActionAdd, []byte("{"))
			if !errors.As(err, new(*json.SyntaxError)) {
				t.Errorf("invalid JSON: got %v, want syntax error", err)
			}
		})
	}
	if _, err := newGeoLite2CountryMMDBIn("unknown", "", lib.ActionAdd); err == nil {
		t.Error("accepted a format with no URI or default")
	}
}

func TestMMDBOutputConstructors(t *testing.T) {
	tests := []struct {
		typ, desc, subdir string
		new               func(lib.Action, ...lib.OutputOption) (lib.OutputConverter, error)
		fromJSON          func(lib.Action, []byte) (lib.OutputConverter, error)
	}{
		{TypeGeoLite2CountryMMDBOut, DescGeoLite2CountryMMDBOut, "maxmind", NewGeoLite2CountryMMDBOut, NewGeoLite2CountryMMDBOutFromBytes},
		{TypeDBIPCountryMMDBOut, DescDBIPCountryMMDBOut, "db-ip", NewDBIPCountryMMDBOut, NewDBIPCountryMMDBOutFromBytes},
		{TypeIPInfoCountryMMDBOut, DescIPInfoCountryMMDBOut, "ipinfo", NewIPInfoCountryMMDBOut, NewIPInfoCountryMMDBOutFromBytes},
	}
	blankDir := WithOutputDir(" ")
	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			for name, opts := range map[string][]lib.OutputOption{
				"defaults": nil, "reset to defaults": {nil, WithOutputName("custom.mmdb"), WithOutputDir("custom"), blankDir, WithOutputName(" ")},
			} {
				t.Run(name, func(t *testing.T) {
					out, err := tt.new(lib.ActionOutput, opts...)
					if err != nil {
						t.Fatal(err)
					}
					g := out.(*geolite2_country_mmdb_out)
					if out.GetType() != tt.typ || out.GetDescription() != tt.desc || out.GetAction() != lib.ActionOutput || g.OutputDir != filepath.Join("output", tt.subdir) || g.OutputName != "Country.mmdb" {
						t.Fatalf("unexpected defaults: %+v", out)
					}
				})
			}
			out, err := tt.fromJSON(lib.ActionOutput, []byte(`{
				"outputName":" result.mmdb ", "outputDir":" custom ", "sourceMMDBURI":" source ",
				"wantedList":["DE","JP"], "overwriteList":["JP"], "excludedList":["US"], "onlyIPType":"ipv4"}`))
			if err != nil {
				t.Fatal(err)
			}
			g := out.(*geolite2_country_mmdb_out)
			if g.Type != tt.typ || g.OutputName != "result.mmdb" || g.OutputDir != "custom" || g.SourceMMDBURI != "source" || g.OnlyIPType != lib.IPv4 ||
				!slices.Equal(g.Want, []string{"DE", "JP"}) || !slices.Equal(g.Overwrite, []string{"JP"}) || !slices.Equal(g.Exclude, []string{"US"}) {
				t.Errorf("unexpected JSON configuration: %+v", g)
			}
			_, err = tt.fromJSON(lib.ActionOutput, []byte("{"))
			if !errors.As(err, new(*json.SyntaxError)) {
				t.Errorf("invalid JSON: got %v, want syntax error", err)
			}
		})
	}
}

func TestCSVInputConstructors(t *testing.T) {
	t.Run("country defaults", func(t *testing.T) {
		in, err := NewGeoLite2CountryCSVIn(lib.ActionAdd, nil)
		if err != nil {
			t.Fatal(err)
		}
		g := in.(*geolite2_country_csv_in)
		if in.GetType() != TypeGeoLite2CountryCSVIn || in.GetAction() != lib.ActionAdd || in.GetDescription() != DescGeoLite2CountryCSVIn ||
			g.CountryCodeFile != filepath.Join("geolite2", "GeoLite2-Country-Locations-en.csv") || g.IPv4File != filepath.Join("geolite2", "GeoLite2-Country-Blocks-IPv4.csv") || g.IPv6File != filepath.Join("geolite2", "GeoLite2-Country-Blocks-IPv6.csv") {
			t.Errorf("unexpected defaults: %+v", g)
		}
	})
	t.Run("ASN defaults", func(t *testing.T) {
		in, err := NewGeoLite2ASNCSVIn(lib.ActionAdd, nil)
		if err != nil {
			t.Fatal(err)
		}
		g := in.(*geolite2_asn_csv_in)
		if in.GetType() != TypeGeoLite2ASNCSVIn || in.GetAction() != lib.ActionAdd || in.GetDescription() != DescGeoLite2ASNCSVIn ||
			g.IPv4File != filepath.Join("geolite2", "GeoLite2-ASN-Blocks-IPv4.csv") || g.IPv6File != filepath.Join("geolite2", "GeoLite2-ASN-Blocks-IPv6.csv") {
			t.Errorf("unexpected defaults: %+v", g)
		}
	})
	t.Run("country JSON with only IPv6 file", func(t *testing.T) {
		in, err := NewGeoLite2CountryCSVInFromBytes(lib.ActionRemove, []byte(`{"country":" locations.csv ","ipv6":" v6.csv ","wantedList":[" de "," "],"onlyIPType":"ipv6"}`))
		if err != nil {
			t.Fatal(err)
		}
		g := in.(*geolite2_country_csv_in)
		if g.CountryCodeFile != "locations.csv" || g.IPv4File != "" || g.IPv6File != "v6.csv" || !maps.Equal(g.Want, map[string]bool{"DE": true}) || g.OnlyIPType != lib.IPv6 || g.Action != lib.ActionRemove {
			t.Errorf("unexpected JSON configuration: %+v", g)
		}
	})
	t.Run("ASN JSON with only IPv4 file", func(t *testing.T) {
		in, err := NewGeoLite2ASNCSVInFromBytes(lib.ActionRemove, []byte(`{"ipv4":" v4.csv ","wantedList":{"example":[" AS64496 "]},"onlyIPType":"ipv4"}`))
		if err != nil {
			t.Fatal(err)
		}
		g := in.(*geolite2_asn_csv_in)
		if g.IPv4File != "v4.csv" || g.IPv6File != "" || !reflect.DeepEqual(g.Want, map[string][]string{"64496": {"EXAMPLE"}}) || g.OnlyIPType != lib.IPv4 || g.Action != lib.ActionRemove {
			t.Errorf("unexpected JSON configuration: %+v", g)
		}
	})
	for name, newConverter := range map[string]func(lib.Action, []byte) (lib.InputConverter, error){
		"country": NewGeoLite2CountryCSVInFromBytes, "ASN": NewGeoLite2ASNCSVInFromBytes,
	} {
		t.Run(name+" invalid JSON", func(t *testing.T) {
			_, err := newConverter(lib.ActionAdd, []byte("{"))
			if !errors.As(err, new(*json.SyntaxError)) {
				t.Errorf("invalid JSON: got %v, want syntax error", err)
			}
		})
	}
}

func TestRegisteredConfigCreators(t *testing.T) {
	tests := []struct{ direction, typ string }{
		{"input", TypeGeoLite2CountryCSVIn}, {"input", TypeGeoLite2ASNCSVIn},
		{"input", TypeGeoLite2CountryMMDBIn}, {"input", TypeDBIPCountryMMDBIn}, {"input", TypeIPInfoCountryMMDBIn},
		{"output", TypeGeoLite2CountryMMDBOut}, {"output", TypeDBIPCountryMMDBOut}, {"output", TypeIPInfoCountryMMDBOut},
	}
	for _, tt := range tests {
		t.Run(tt.direction+"/"+tt.typ, func(t *testing.T) {
			instance, err := lib.NewInstance()
			if err != nil {
				t.Fatal(err)
			}
			action := lib.ActionAdd
			if tt.direction == "output" {
				action = lib.ActionOutput
			}
			config := fmt.Sprintf(`{%q:[{"type":%q,"action":%q}]}`, tt.direction, tt.typ, action)
			if err := instance.InitConfigFromBytes([]byte(config)); err != nil {
				t.Fatal(err)
			}
		})
	}
}
