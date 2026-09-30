package maxmind

import (
	"encoding/json"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

var inputConstructors = []struct {
	name        string
	description string
	new         func(lib.Action, ...lib.InputOption) lib.InputConverter
	fromBytes   func(lib.Action, []byte) (lib.InputConverter, error)
}{
	{TypeGeoLite2CountryMMDBIn, DescGeoLite2CountryMMDBIn, NewGeoLite2CountryMMDBIn, NewGeoLite2CountryMMDBInFromBytes},
	{TypeDBIPCountryMMDBIn, DescDBIPCountryMMDBIn, NewDBIPCountryMMDBIn, NewDBIPCountryMMDBInFromBytes},
	{TypeIPInfoCountryMMDBIn, DescIPInfoCountryMMDBIn, NewIPInfoCountryMMDBIn, NewIPInfoCountryMMDBInFromBytes},
	{TypeGeoLite2CountryCSVIn, DescGeoLite2CountryCSVIn, NewGeoLite2CountryCSVIn, NewGeoLite2CountryCSVInFromBytes},
	{TypeGeoLite2ASNCSVIn, DescGeoLite2ASNCSVIn, NewGeoLite2ASNCSVIn, NewGeoLite2ASNCSVInFromBytes},
}

var outputConstructors = []struct {
	name        string
	description string
	dir         string
	new         func(lib.Action, ...lib.OutputOption) lib.OutputConverter
	fromBytes   func(lib.Action, []byte) (lib.OutputConverter, error)
}{
	{TypeGeoLite2CountryMMDBOut, DescGeoLite2CountryMMDBOut, "output/maxmind", NewGeoLite2CountryMMDBOut, NewGeoLite2CountryMMDBOutFromBytes},
	{TypeDBIPCountryMMDBOut, DescDBIPCountryMMDBOut, "output/db-ip", NewDBIPCountryMMDBOut, NewDBIPCountryMMDBOutFromBytes},
	{TypeIPInfoCountryMMDBOut, DescIPInfoCountryMMDBOut, "output/ipinfo", NewIPInfoCountryMMDBOut, NewIPInfoCountryMMDBOutFromBytes},
}

func TestConstructorDefaults(t *testing.T) {
	for _, tc := range inputConstructors {
		t.Run("input/"+tc.name, func(t *testing.T) {
			want := tc.new(lib.ActionAdd)
			if want.GetType() != tc.name || want.GetDescription() != tc.description || want.GetAction() != lib.ActionAdd {
				t.Fatalf("unexpected metadata: %#v", want)
			}
			if got := tc.new(lib.ActionAdd, nil); !reflect.DeepEqual(got, want) {
				t.Fatalf("nil option changed defaults: %#v", got)
			}
			for _, data := range []string{"", "{}", "null"} {
				got, err := tc.fromBytes(lib.ActionAdd, []byte(data))
				if err != nil || !reflect.DeepEqual(got, want) {
					t.Fatalf("JSON %q defaults = %#v, %v; want %#v", data, got, err, want)
				}
			}
			switch g := want.(type) {
			case *geoLite2CountryMMDBIn:
				paths := map[string]string{
					TypeGeoLite2CountryMMDBIn: "geolite2/GeoLite2-Country.mmdb",
					TypeDBIPCountryMMDBIn:     "db-ip/dbip-country-lite.mmdb",
					TypeIPInfoCountryMMDBIn:   "ipinfo/country.mmdb",
				}
				if g.URI != paths[tc.name] || len(g.Want) != 0 || g.OnlyIPType != "" {
					t.Fatalf("unexpected MMDB defaults: %#v", g)
				}
				if got := tc.new(lib.ActionAdd, WithURI(" "), WithInputWantedList(nil)); !reflect.DeepEqual(got, want) {
					t.Fatalf("empty options changed defaults: %#v", got)
				}
			case *geoLite2CountryCSVIn:
				if g.CountryCodeFile != "geolite2/GeoLite2-Country-Locations-en.csv" ||
					g.IPv4File != "geolite2/GeoLite2-Country-Blocks-IPv4.csv" ||
					g.IPv6File != "geolite2/GeoLite2-Country-Blocks-IPv6.csv" ||
					len(g.Want) != 0 || g.OnlyIPType != "" {
					t.Fatalf("unexpected country CSV defaults: %#v", g)
				}
				if got := tc.new(lib.ActionAdd, WithCountryCodeFile(" "), WithIPv4File(""), WithIPv6File(" "), WithInputWantedList(nil)); !reflect.DeepEqual(got, want) {
					t.Fatalf("empty options changed defaults: %#v", got)
				}
			case *geoLite2ASNCSVIn:
				if g.IPv4File != "geolite2/GeoLite2-ASN-Blocks-IPv4.csv" ||
					g.IPv6File != "geolite2/GeoLite2-ASN-Blocks-IPv6.csv" ||
					len(g.Want) != 0 || g.OnlyIPType != "" {
					t.Fatalf("unexpected ASN CSV defaults: %#v", g)
				}
				if got := tc.new(lib.ActionAdd, WithIPv4File(" "), WithIPv6File(""), WithInputWantedList(nil)); !reflect.DeepEqual(got, want) {
					t.Fatalf("empty options changed defaults: %#v", got)
				}
			}
		})
	}
	for _, tc := range outputConstructors {
		t.Run("output/"+tc.name, func(t *testing.T) {
			want := tc.new(lib.ActionOutput).(*geoLite2CountryMMDBOut)
			if want.GetType() != tc.name || want.GetDescription() != tc.description || want.GetAction() != lib.ActionOutput ||
				want.OutputDir != tc.dir || want.OutputName != "Country.mmdb" || want.OnlyIPType != "" || want.SourceMMDBURI != "" ||
				len(want.Want) != 0 || len(want.Exclude) != 0 || len(want.Overwrite) != 0 {
				t.Fatalf("unexpected output defaults: %#v", want)
			}
			for _, opts := range [][]lib.OutputOption{nil, {nil}, {WithOutputName(" "), WithOutputDir(" ")}} {
				if got := tc.new(lib.ActionOutput, opts...); !reflect.DeepEqual(got, want) {
					t.Fatalf("empty options changed defaults: %#v", got)
				}
			}
			for _, data := range []string{"", "{}", "null"} {
				got, err := tc.fromBytes(lib.ActionOutput, []byte(data))
				if err != nil || !reflect.DeepEqual(got, want) {
					t.Fatalf("JSON %q defaults = %#v, %v; want %#v", data, got, err, want)
				}
			}
		})
	}
}

func TestInputJSONOptionsEquivalence(t *testing.T) {
	for _, tc := range inputConstructors {
		for _, action := range []lib.Action{lib.ActionAdd, lib.ActionRemove} {
			for _, ipType := range []lib.IPType{"", lib.IPv4, lib.IPv6} {
				t.Run(tc.name+"/"+string(action)+"/"+string(ipType), func(t *testing.T) {
					lists := []string{" cn ", "US", "", "cn"}
					if tc.name == TypeGeoLite2ASNCSVIn {
						lists = []string{" as123 ", "456", "", "AS123"}
					}
					opts := []lib.InputOption{nil, WithInputWantedList(lists), WithInputOnlyIPType(ipType)}
					args := map[string]any{"wantedList": lists, "onlyIPType": ipType}
					switch tc.name {
					case TypeGeoLite2CountryCSVIn:
						opts = append(opts, WithCountryCodeFile(" locations.csv "), WithIPv4File(" v4.csv "))
						args["country"], args["ipv4"] = " locations.csv ", " v4.csv "
					case TypeGeoLite2ASNCSVIn:
						opts = append(opts, WithIPv6File(" v6.csv "))
						args["ipv6"] = " v6.csv "
					default:
						opts = append(opts, WithURI(" country.mmdb "))
						args["uri"] = " country.mmdb "
					}
					data, err := json.Marshal(args)
					if err != nil {
						t.Fatal(err)
					}
					want := tc.new(action, opts...)
					got, err := tc.fromBytes(action, data)
					if err != nil || !reflect.DeepEqual(got, want) {
						t.Fatalf("JSON = %#v, %v; direct = %#v", got, err, want)
					}
					switch g := got.(type) {
					case *geoLite2CountryMMDBIn:
						if g.URI != "country.mmdb" || !reflect.DeepEqual(g.Want, map[string]bool{"CN": true, "US": true}) {
							t.Fatalf("unexpected MMDB options: %#v", g)
						}
					case *geoLite2CountryCSVIn:
						if g.CountryCodeFile != "locations.csv" || g.IPv4File != "v4.csv" || g.IPv6File != "" ||
							!reflect.DeepEqual(g.Want, map[string]bool{"CN": true, "US": true}) {
							t.Fatalf("unexpected country CSV options: %#v", g)
						}
					case *geoLite2ASNCSVIn:
						if g.IPv4File != "" || g.IPv6File != "v6.csv" ||
							!reflect.DeepEqual(g.Want, map[string][]string{"123": {"AS123"}, "456": {"AS456"}}) {
							t.Fatalf("unexpected ASN CSV options: %#v", g)
						}
					}
				})
			}
		}
	}
}

func TestCSVSingleFileOptions(t *testing.T) {
	for _, tc := range inputConstructors[3:] {
		for _, family := range []string{"ipv4", "ipv6"} {
			t.Run(tc.name+"/"+family, func(t *testing.T) {
				opt := WithIPv4File("blocks.csv")
				if family == "ipv6" {
					opt = WithIPv6File("blocks.csv")
				}
				got := tc.new(lib.ActionAdd, opt)
				fromJSON, err := tc.fromBytes(lib.ActionAdd, []byte(`{"`+family+`":"blocks.csv"}`))
				if err != nil || !reflect.DeepEqual(got, fromJSON) {
					t.Fatalf("JSON = %#v, %v; direct = %#v", fromJSON, err, got)
				}
				var v4, v6 string
				switch g := got.(type) {
				case *geoLite2CountryCSVIn:
					v4, v6 = g.IPv4File, g.IPv6File
				case *geoLite2ASNCSVIn:
					v4, v6 = g.IPv4File, g.IPv6File
				}
				if family == "ipv4" && (v4 != "blocks.csv" || v6 != "") ||
					family == "ipv6" && (v4 != "" || v6 != "blocks.csv") {
					t.Fatalf("single-family option defaulted another file: %q, %q", v4, v6)
				}
			})
		}
	}
}

func TestOutputJSONOptionsEquivalence(t *testing.T) {
	for _, tc := range outputConstructors {
		for _, ipType := range []lib.IPType{"", lib.IPv4, lib.IPv6} {
			t.Run(tc.name+"/"+string(ipType), func(t *testing.T) {
				want := tc.new(lib.ActionOutput, nil,
					WithOutputName(" custom.mmdb "), WithOutputDir(" custom "),
					WithOutputWantedList([]string{"us", "cn"}), WithOutputOverwriteList([]string{"cn", "us"}),
					WithOutputExcludedList([]string{"private"}), WithOutputOnlyIPType(ipType), WithSourceMMDBURI(" source.mmdb "))
				data := `{"outputName":" custom.mmdb ","outputDir":" custom ","wantedList":["us","cn"],"overwriteList":["cn","us"],"excludedList":["private"],"sourceMMDBURI":" source.mmdb ","onlyIPType":"` + string(ipType) + `"}`
				got, err := tc.fromBytes(lib.ActionOutput, []byte(data))
				if err != nil || !reflect.DeepEqual(got, want) {
					t.Fatalf("JSON = %#v, %v; direct = %#v", got, err, want)
				}
				g := got.(*geoLite2CountryMMDBOut)
				if g.OutputName != "custom.mmdb" || g.OutputDir != "custom" || g.SourceMMDBURI != "source.mmdb" ||
					g.OnlyIPType != ipType || !slices.Equal(g.Want, []string{"us", "cn"}) ||
					!slices.Equal(g.Overwrite, []string{"cn", "us"}) || !slices.Equal(g.Exclude, []string{"private"}) {
					t.Fatalf("unexpected output options: %#v", g)
				}
			})
		}
	}
}

func TestMalformedJSON(t *testing.T) {
	for _, data := range []string{`{`, `[]`, `{"onlyIPType":1}`} {
		for _, tc := range inputConstructors {
			if got, err := tc.fromBytes(lib.ActionAdd, []byte(data)); err == nil || got != nil {
				t.Errorf("%s accepted malformed input %q: %#v, %v", tc.name, data, got, err)
			}
		}
		for _, tc := range outputConstructors {
			if got, err := tc.fromBytes(lib.ActionOutput, []byte(data)); err == nil || got != nil {
				t.Errorf("%s accepted malformed output %q: %#v, %v", tc.name, data, got, err)
			}
		}
	}
}

func TestASNWantedListForms(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
		list any
		want map[string][]string
	}{
		{"array", `[" as123 ","456","","AS"]`, []string{" as123 ", "456", "", "AS"}, map[string][]string{"123": {"AS123"}, "456": {"AS456"}}},
		{"map", `{" cloud ":["as123","456"],"other":["AS123"]," ":["789"],"empty":["","AS"]}`,
			map[string][]string{" cloud ": {"as123", "456"}, "other": {"AS123"}, " ": {"789"}, "empty": {"", "AS"}},
			map[string][]string{"123": {"CLOUD", "OTHER"}, "456": {"CLOUD"}}},
		{"extended-array", `["AS123"]`, lib.WantedListExtended{TypeSlice: []string{"AS123"}}, map[string][]string{"123": {"AS123"}}},
		{"extended-map", `{"cloud":["AS123"]}`, lib.WantedListExtended{TypeMap: map[string][]string{"cloud": {"AS123"}}}, map[string][]string{"123": {"CLOUD"}}},
		{"nil", `null`, nil, map[string][]string{}},
		{"nil-array", `null`, []string(nil), map[string][]string{}},
		{"nil-map", `null`, map[string][]string(nil), map[string][]string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			direct := NewGeoLite2ASNCSVIn(lib.ActionAdd, WithInputWantedList(tc.list)).(*geoLite2ASNCSVIn)
			got, err := NewGeoLite2ASNCSVInFromBytes(lib.ActionAdd, []byte(`{"wantedList":`+tc.data+`}`))
			if err != nil {
				t.Fatal(err)
			}
			fromJSON := got.(*geoLite2ASNCSVIn)
			for _, want := range direct.Want {
				slices.Sort(want)
			}
			for _, want := range fromJSON.Want {
				slices.Sort(want)
			}
			if !reflect.DeepEqual(direct.Want, tc.want) || !reflect.DeepEqual(direct, fromJSON) {
				t.Fatalf("direct = %#v; JSON = %#v; want %#v", direct, fromJSON, tc.want)
			}
		})
	}
}

func TestConstructorValidation(t *testing.T) {
	type fatalCase struct {
		name  string
		field string
		run   func()
	}
	var cases []fatalCase
	for _, tc := range inputConstructors {
		for _, action := range []lib.Action{"", lib.ActionOutput, "invalid"} {
			cases = append(cases,
				fatalCase{"input/" + tc.name + "/action/" + string(action), "action", func() { tc.new(action) }},
				fatalCase{"input/" + tc.name + "/json-action/" + string(action), "action", func() { tc.fromBytes(action, []byte(`{}`)) }})
		}
		for _, ipType := range []lib.IPType{"IPv4", " ipv6 ", "invalid"} {
			cases = append(cases,
				fatalCase{"input/" + tc.name + "/ip/" + string(ipType), "onlyIPType", func() { tc.new(lib.ActionAdd, WithInputOnlyIPType(ipType)) }},
				fatalCase{"input/" + tc.name + "/json-ip/" + string(ipType), "onlyIPType", func() {
					tc.fromBytes(lib.ActionAdd, []byte(`{"onlyIPType":"`+string(ipType)+`"}`))
				}})
		}
		cases = append(cases, fatalCase{"input/" + tc.name + "/wanted", "wantedList", func() {
			tc.new(lib.ActionAdd, WithInputWantedList(123))
		}})
	}
	for _, tc := range outputConstructors {
		for _, action := range []lib.Action{"", lib.ActionAdd, lib.ActionRemove, "invalid"} {
			cases = append(cases,
				fatalCase{"output/" + tc.name + "/action/" + string(action), "action", func() { tc.new(action) }},
				fatalCase{"output/" + tc.name + "/json-action/" + string(action), "action", func() { tc.fromBytes(action, []byte(`{}`)) }})
		}
		cases = append(cases,
			fatalCase{"output/" + tc.name + "/ip", "onlyIPType", func() { tc.new(lib.ActionOutput, WithOutputOnlyIPType("invalid")) }},
			fatalCase{"output/" + tc.name + "/json-ip", "onlyIPType", func() { tc.fromBytes(lib.ActionOutput, []byte(`{"onlyIPType":"invalid"}`)) }})
	}
	for _, tc := range []struct {
		name string
		data string
		list any
	}{
		{"number", `123`, 123},
		{"string", `"AS123"`, "AS123"},
		{"boolean", `true`, true},
		{"mixed-array", `["AS123",123]`, []any{"AS123", 123}},
		{"null-array-item", `[null]`, []any{nil}},
		{"map-string", `{"cloud":"AS123"}`, map[string]string{"cloud": "AS123"}},
		{"map-mixed-array", `{"cloud":["AS123",123]}`, map[string]any{"cloud": []any{"AS123", 123}}},
		{"map-object", `{"cloud":{"asn":"AS123"}}`, map[string]any{"cloud": map[string]string{"asn": "AS123"}}},
	} {
		cases = append(cases,
			fatalCase{"asn/" + tc.name, "wantedList", func() { NewGeoLite2ASNCSVIn(lib.ActionAdd, WithInputWantedList(tc.list)) }},
			fatalCase{"asn/json-" + tc.name, "wantedList", func() {
				NewGeoLite2ASNCSVInFromBytes(lib.ActionAdd, []byte(`{"wantedList":`+tc.data+`}`))
			}})
	}
	if child := os.Getenv("MAXMIND_FATAL_CASE"); child != "" {
		for _, tc := range cases {
			if tc.name == child {
				tc.run()
				t.Fatal("invalid constructor returned instead of exiting")
			}
		}
		t.Fatalf("unknown subprocess case %q", child)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestConstructorValidation$")
			cmd.Env = append(os.Environ(), "MAXMIND_FATAL_CASE="+tc.name)
			output, err := cmd.CombinedOutput()
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 1 || !strings.Contains(string(output), "invalid "+tc.field) {
				t.Fatalf("expected fatal %s: %v\n%s", tc.field, err, output)
			}
		})
	}
}
