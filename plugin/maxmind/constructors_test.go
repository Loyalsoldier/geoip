package maxmind

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

var inputConstructors = []struct {
	name string
	new  func(lib.Action, ...lib.InputOption) lib.InputConverter
	json func(lib.Action, []byte) (lib.InputConverter, error)
	want lib.InputConverter
}{
	{
		"maxmind", NewGeoLite2CountryMMDBIn, NewGeoLite2CountryMMDBInFromBytes,
		&geoLite2CountryMMDBIn{
			Type: TypeGeoLite2CountryMMDBIn, Action: lib.ActionAdd, Description: DescGeoLite2CountryMMDBIn,
			URI: filepath.Join("geolite2", "GeoLite2-Country.mmdb"), Want: map[string]bool{},
		},
	},
	{
		"dbip", NewDBIPCountryMMDBIn, NewDBIPCountryMMDBInFromBytes,
		&geoLite2CountryMMDBIn{
			Type: TypeDBIPCountryMMDBIn, Action: lib.ActionAdd, Description: DescDBIPCountryMMDBIn,
			URI: filepath.Join("db-ip", "dbip-country-lite.mmdb"), Want: map[string]bool{},
		},
	},
	{
		"ipinfo", NewIPInfoCountryMMDBIn, NewIPInfoCountryMMDBInFromBytes,
		&geoLite2CountryMMDBIn{
			Type: TypeIPInfoCountryMMDBIn, Action: lib.ActionAdd, Description: DescIPInfoCountryMMDBIn,
			URI: filepath.Join("ipinfo", "country.mmdb"), Want: map[string]bool{},
		},
	},
	{
		"country-csv", NewGeoLite2CountryCSVIn, NewGeoLite2CountryCSVInFromBytes,
		&geoLite2CountryCSVIn{
			Type: TypeGeoLite2CountryCSVIn, Action: lib.ActionAdd, Description: DescGeoLite2CountryCSVIn,
			CountryCodeFile: filepath.Join("geolite2", "GeoLite2-Country-Locations-en.csv"),
			IPv4File:        filepath.Join("geolite2", "GeoLite2-Country-Blocks-IPv4.csv"),
			IPv6File:        filepath.Join("geolite2", "GeoLite2-Country-Blocks-IPv6.csv"), Want: map[string]bool{},
		},
	},
	{
		"asn-csv", NewGeoLite2ASNCSVIn, NewGeoLite2ASNCSVInFromBytes,
		&geoLite2ASNCSVIn{
			Type: TypeGeoLite2ASNCSVIn, Action: lib.ActionAdd, Description: DescGeoLite2ASNCSVIn,
			IPv4File: filepath.Join("geolite2", "GeoLite2-ASN-Blocks-IPv4.csv"),
			IPv6File: filepath.Join("geolite2", "GeoLite2-ASN-Blocks-IPv6.csv"), Want: map[string][]string{},
		},
	},
}

var outputConstructors = []struct {
	name string
	new  func(lib.Action, ...lib.OutputOption) lib.OutputConverter
	json func(lib.Action, []byte) (lib.OutputConverter, error)
	want *geoLite2CountryMMDBOut
}{
	{
		"maxmind", NewGeoLite2CountryMMDBOut, NewGeoLite2CountryMMDBOutFromBytes,
		&geoLite2CountryMMDBOut{
			Type: TypeGeoLite2CountryMMDBOut, Action: lib.ActionOutput, Description: DescGeoLite2CountryMMDBOut,
			OutputName: "Country.mmdb", OutputDir: filepath.Join("output", "maxmind"),
		},
	},
	{
		"dbip", NewDBIPCountryMMDBOut, NewDBIPCountryMMDBOutFromBytes,
		&geoLite2CountryMMDBOut{
			Type: TypeDBIPCountryMMDBOut, Action: lib.ActionOutput, Description: DescDBIPCountryMMDBOut,
			OutputName: "Country.mmdb", OutputDir: filepath.Join("output", "db-ip"),
		},
	},
	{
		"ipinfo", NewIPInfoCountryMMDBOut, NewIPInfoCountryMMDBOutFromBytes,
		&geoLite2CountryMMDBOut{
			Type: TypeIPInfoCountryMMDBOut, Action: lib.ActionOutput, Description: DescIPInfoCountryMMDBOut,
			OutputName: "Country.mmdb", OutputDir: filepath.Join("output", "ipinfo"),
		},
	},
}

func assertEqual(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestConstructorDefaults(t *testing.T) {
	for _, tc := range inputConstructors {
		t.Run("input/"+tc.name, func(t *testing.T) {
			assertEqual(t, tc.new(lib.ActionAdd, nil), tc.want)
			for _, data := range []string{"", "{}", "null", `{"uri":"  ","country":"  ","ipv4":"  ","ipv6":"  ","onlyIPType":"  "}`} {
				got, err := tc.json(lib.ActionAdd, []byte(data))
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, got, tc.want)
			}
			assertEqual(t, tc.new(lib.ActionRemove).GetAction(), lib.ActionRemove)
		})
	}
	for _, tc := range outputConstructors {
		t.Run("output/"+tc.name, func(t *testing.T) {
			assertEqual(t, tc.new(lib.ActionOutput, nil), tc.want)
			for _, data := range []string{"", "{}", "null", `{"outputDir":"  ","outputName":"  ","sourceMMDBURI":"  ","onlyIPType":"  "}`} {
				got, err := tc.json(lib.ActionOutput, []byte(data))
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, got, tc.want)
			}
		})
	}
}

func TestMMDBInputOptionsAndJSON(t *testing.T) {
	for _, tc := range inputConstructors[:3] {
		t.Run(tc.name, func(t *testing.T) {
			opts := []lib.InputOption{
				WithURI("unused"), nil, WithURI(" https://example.test/country.mmdb "),
				WithInputWantedList([]string{"old"}), WithInputWantedList([]string{" us ", "", "CN", "us"}),
				WithInputOnlyIPType("invalid"), WithInputOnlyIPType(" IPv6 "),
			}
			got := tc.new(lib.ActionRemove, opts...).(*geoLite2CountryMMDBIn)
			assertEqual(t, got.URI, "https://example.test/country.mmdb")
			assertEqual(t, got.Want, map[string]bool{"US": true, "CN": true})
			assertEqual(t, got.OnlyIPType, lib.IPv6)
			assertEqual(t, got.GetAction(), lib.ActionRemove)
			fromJSON, err := tc.json(lib.ActionRemove, []byte(`{"uri":" https://example.test/country.mmdb ","wantedList":[" us ","","CN","us"],"onlyIPType":" IPv6 "}`))
			if err != nil {
				t.Fatal(err)
			}
			assertEqual(t, got, fromJSON)
			assertEqual(t, tc.new(lib.ActionAdd, WithURI("unused"), WithURI(" ")), tc.want)
		})
	}
}

func TestCSVOptionsAndJSON(t *testing.T) {
	for _, tc := range inputConstructors[3:] {
		t.Run(tc.name, func(t *testing.T) {
			for _, family := range []string{"ipv4", "ipv6", "both"} {
				t.Run(family, func(t *testing.T) {
					var ipv4, ipv6 string
					if family != "ipv6" {
						ipv4 = " https://example.test/v4.csv "
					}
					if family != "ipv4" {
						ipv6 = " ./v6.csv "
					}
					opts := []lib.InputOption{nil, WithIPv4File("unused"), WithIPv4File(ipv4), WithIPv6File(ipv6), WithInputOnlyIPType(" IPv4 ")}
					got := tc.new(lib.ActionRemove, opts...)
					fromJSON, err := tc.json(lib.ActionRemove, []byte(fmt.Sprintf(`{"ipv4":%q,"ipv6":%q,"onlyIPType":" IPv4 "}`, ipv4, ipv6)))
					if err != nil {
						t.Fatal(err)
					}
					assertEqual(t, got, fromJSON)
					switch g := got.(type) {
					case *geoLite2CountryCSVIn:
						assertEqual(t, g.IPv4File, strings.TrimSpace(ipv4))
						assertEqual(t, g.IPv6File, strings.TrimSpace(ipv6))
						assertEqual(t, g.CountryCodeFile, defaultGeoLite2CountryCodeFile)
						assertEqual(t, g.OnlyIPType, lib.IPv4)
					case *geoLite2ASNCSVIn:
						assertEqual(t, g.IPv4File, strings.TrimSpace(ipv4))
						assertEqual(t, g.IPv6File, strings.TrimSpace(ipv6))
						assertEqual(t, g.OnlyIPType, lib.IPv4)
					}
				})
			}
			assertEqual(t, tc.new(lib.ActionAdd, WithIPv4File("unused"), WithIPv4File(" "), WithIPv6File(" ")), tc.want)
		})
	}

	country := NewGeoLite2CountryCSVIn(lib.ActionAdd,
		WithCountryCodeFile("unused"), WithCountryCodeFile(" ./locations.csv "),
		WithInputWantedList([]string{" us ", "", "US", "cn"}),
	).(*geoLite2CountryCSVIn)
	assertEqual(t, country.CountryCodeFile, "./locations.csv")
	assertEqual(t, country.Want, map[string]bool{"US": true, "CN": true})
	fromJSON, err := NewGeoLite2CountryCSVInFromBytes(lib.ActionAdd, []byte(`{"country":" ./locations.csv ","wantedList":[" us ","","US","cn"]}`))
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, country, fromJSON)
}

func TestASNWantedListOptionsAndJSON(t *testing.T) {
	tests := []struct {
		name string
		list lib.WantedListExtended
		json string
		want map[string][]string
	}{
		{"slice", lib.WantedListExtended{TypeSlice: []string{" as123 ", "456", "", "AS"}},
			`{"wantedList":[" as123 ","456","","AS"]}`, map[string][]string{"123": {"AS123"}, "456": {"AS456"}}},
		{"map", lib.WantedListExtended{TypeMap: map[string][]string{" custom ": {" as123 ", "456", "", "AS"}, " ": {"789"}}},
			`{"wantedList":{" custom ":[" as123 ","456","","AS"]," ":["789"]}}`, map[string][]string{"123": {"CUSTOM"}, "456": {"CUSTOM"}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NewGeoLite2ASNCSVIn(lib.ActionAdd, WithInputASNWantedList(tc.list)).(*geoLite2ASNCSVIn)
			assertEqual(t, got.Want, tc.want)
			fromJSON, err := NewGeoLite2ASNCSVInFromBytes(lib.ActionAdd, []byte(tc.json))
			if err != nil {
				t.Fatal(err)
			}
			assertEqual(t, got, fromJSON)
		})
	}
	got := NewGeoLite2ASNCSVIn(lib.ActionAdd, WithInputASNWantedList(lib.WantedListExtended{
		TypeMap: map[string][]string{"CUSTOM": {"123", "456"}}, TypeSlice: []string{"AS123"},
	})).(*geoLite2ASNCSVIn)
	assertEqual(t, got.Want, map[string][]string{"123": {"AS123"}, "456": {"CUSTOM"}})
}

func TestOutputOptionsAndJSON(t *testing.T) {
	for _, tc := range outputConstructors {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.new(lib.ActionOutput,
				nil, WithOutputDir("unused"), WithOutputDir(" ./custom "),
				WithOutputName("unused"), WithOutputName(" Custom.mmdb "),
				WithOutputWantedList([]string{" us ", "CN", "us"}),
				WithOutputExcludedList([]string{" jp "}),
				WithOutputOverwriteList([]string{" cn ", "US"}),
				WithOutputOnlyIPType("invalid"), WithOutputOnlyIPType(" IPv6 "),
				WithOutputSourceMMDBURI(" https://example.test/source.mmdb "),
			).(*geoLite2CountryMMDBOut)
			assertEqual(t, got.OutputDir, "./custom")
			assertEqual(t, got.OutputName, "Custom.mmdb")
			assertEqual(t, got.Want, []string{" us ", "CN", "us"})
			assertEqual(t, got.Overwrite, []string{" cn ", "US"})
			assertEqual(t, got.Exclude, []string{" jp "})
			assertEqual(t, got.OnlyIPType, lib.IPv6)
			assertEqual(t, got.SourceMMDBURI, "https://example.test/source.mmdb")
			fromJSON, err := tc.json(lib.ActionOutput, []byte(`{"outputDir":" ./custom ","outputName":" Custom.mmdb ","wantedList":[" us ","CN","us"],"excludedList":[" jp "],"overwriteList":[" cn ","US"],"onlyIPType":" IPv6 ","sourceMMDBURI":" https://example.test/source.mmdb "}`))
			if err != nil {
				t.Fatal(err)
			}
			assertEqual(t, got, fromJSON)
			assertEqual(t, tc.new(lib.ActionOutput,
				WithOutputDir("unused"), WithOutputDir(" "),
				WithOutputName("unused"), WithOutputName(" "),
				WithOutputSourceMMDBURI("unused"), WithOutputSourceMMDBURI(" "),
			), tc.want)
		})
	}

	// Reusing blank options must retain each format's own defaults.
	dir := WithOutputDir("")
	uri := WithURI("")
	for i, tc := range outputConstructors {
		assertEqual(t, tc.new(lib.ActionOutput, dir), tc.want)
		assertEqual(t, inputConstructors[i].new(lib.ActionAdd, uri), inputConstructors[i].want)
	}
}

func TestMalformedJSON(t *testing.T) {
	for _, data := range []string{"{", "[]", "true", `{"onlyIPType":false}`, `{"wantedList":42}`, `{"wantedList":[1]}`} {
		for _, tc := range inputConstructors {
			t.Run("input/"+tc.name+"/"+data, func(t *testing.T) {
				got, err := tc.json(lib.ActionAdd, []byte(data))
				if err == nil || got != nil {
					t.Fatalf("got (%#v, %v), want nil converter and JSON error", got, err)
				}
			})
		}
		for _, tc := range outputConstructors {
			t.Run("output/"+tc.name+"/"+data, func(t *testing.T) {
				got, err := tc.json(lib.ActionOutput, []byte(data))
				if err == nil || got != nil {
					t.Fatalf("got (%#v, %v), want nil converter and JSON error", got, err)
				}
			})
		}
	}
}

type foreignConverter struct{}

func (foreignConverter) GetType() string                              { return "foreign" }
func (foreignConverter) GetAction() lib.Action                        { return lib.ActionAdd }
func (foreignConverter) GetDescription() string                       { return "foreign converter" }
func (foreignConverter) Input(c lib.Container) (lib.Container, error) { return c, nil }
func (foreignConverter) Output(lib.Container) error                   { return nil }

func TestConstructorValidation(t *testing.T) {
	type fatalCase struct {
		name    string
		message string
		run     func()
	}
	var cases []fatalCase
	for _, tc := range inputConstructors {
		for _, action := range []lib.Action{"", lib.ActionOutput, "unknown"} {
			cases = append(cases, fatalCase{"input/" + tc.name + "/action/" + string(action), "invalid action", func() { tc.new(action) }})
		}
		cases = append(cases,
			fatalCase{"input/" + tc.name + "/ip-type", "invalid onlyIPType", func() { tc.new(lib.ActionAdd, WithInputOnlyIPType("both")) }},
			fatalCase{"input/" + tc.name + "/json-ip-type", "invalid onlyIPType", func() { tc.json(lib.ActionAdd, []byte(`{"onlyIPType":"both"}`)) }},
			fatalCase{"input/" + tc.name + "/json-action", "invalid action", func() { tc.json(lib.ActionOutput, nil) }},
		)
	}
	for _, tc := range outputConstructors {
		for _, action := range []lib.Action{"", lib.ActionAdd, lib.ActionRemove, "unknown"} {
			cases = append(cases, fatalCase{"output/" + tc.name + "/action/" + string(action), "invalid action", func() { tc.new(action) }})
		}
		cases = append(cases,
			fatalCase{"output/" + tc.name + "/ip-type", "invalid onlyIPType", func() { tc.new(lib.ActionOutput, WithOutputOnlyIPType("both")) }},
			fatalCase{"output/" + tc.name + "/json-ip-type", "invalid onlyIPType", func() { tc.json(lib.ActionOutput, []byte(`{"onlyIPType":"both"}`)) }},
			fatalCase{"output/" + tc.name + "/json-action", "invalid action", func() { tc.json(lib.ActionAdd, nil) }},
		)
	}
	for name, option := range map[string]lib.InputOption{
		"WithURI": WithURI(""), "WithIPv4File": WithIPv4File(""), "WithIPv6File": WithIPv6File(""),
		"WithCountryCodeFile": WithCountryCodeFile(""), "WithInputWantedList": WithInputWantedList(nil),
		"WithInputOnlyIPType": WithInputOnlyIPType(""), "WithInputASNWantedList": WithInputASNWantedList(lib.WantedListExtended{}),
	} {
		cases = append(cases, fatalCase{"foreign/" + name, name + " does not support", func() {
			NewGeoLite2CountryMMDBIn(lib.ActionAdd, func(lib.InputConverter) { option(foreignConverter{}) })
		}})
	}
	for name, option := range map[string]lib.OutputOption{
		"WithOutputDir": WithOutputDir(""), "WithOutputName": WithOutputName(""),
		"WithOutputWantedList": WithOutputWantedList(nil), "WithOutputExcludedList": WithOutputExcludedList(nil),
		"WithOutputOverwriteList": WithOutputOverwriteList(nil), "WithOutputOnlyIPType": WithOutputOnlyIPType(""),
		"WithOutputSourceMMDBURI": WithOutputSourceMMDBURI(""),
	} {
		cases = append(cases, fatalCase{"foreign/" + name, name + " does not support", func() {
			NewGeoLite2CountryMMDBOut(lib.ActionOutput, func(lib.OutputConverter) { option(foreignConverter{}) })
		}})
	}
	cases = append(cases,
		fatalCase{"mismatch/uri", "WithURI does not support", func() { NewGeoLite2CountryCSVIn(lib.ActionAdd, WithURI("file")) }},
		fatalCase{"mismatch/ipv4", "WithIPv4File does not support", func() { NewGeoLite2CountryMMDBIn(lib.ActionAdd, WithIPv4File("file")) }},
		fatalCase{"mismatch/ipv6", "WithIPv6File does not support", func() { NewGeoLite2CountryMMDBIn(lib.ActionAdd, WithIPv6File("file")) }},
		fatalCase{"mismatch/country", "WithCountryCodeFile does not support", func() { NewGeoLite2ASNCSVIn(lib.ActionAdd, WithCountryCodeFile("file")) }},
		fatalCase{"mismatch/country-wanted", "WithInputWantedList does not support", func() { NewGeoLite2ASNCSVIn(lib.ActionAdd, WithInputWantedList(nil)) }},
		fatalCase{"mismatch/asn-wanted", "WithInputASNWantedList does not support", func() { NewGeoLite2CountryCSVIn(lib.ActionAdd, WithInputASNWantedList(lib.WantedListExtended{})) }},
	)

	if name := os.Getenv("GEOIP_MAXMIND_FATAL_CASE"); name != "" {
		for _, tc := range cases {
			if tc.name == name {
				tc.run()
				return
			}
		}
		t.Fatalf("unknown fatal case %q", name)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestConstructorValidation$")
			cmd.Env = append(os.Environ(), "GEOIP_MAXMIND_FATAL_CASE="+tc.name)
			output, err := cmd.CombinedOutput()
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 1 || !strings.Contains(string(output), tc.message) {
				t.Fatalf("expected log.Fatal with %q, got error %v and output %s", tc.message, err, output)
			}
		})
	}
}
