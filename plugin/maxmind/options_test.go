package maxmind

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

var mmdbVariants = []struct {
	name       string
	inputDesc  string
	outputDesc string
	uri        string
	outputDir  string
	newInput   func(lib.Action, ...lib.InputOption) lib.InputConverter
	inputJSON  func(lib.Action, []byte) (lib.InputConverter, error)
	newOutput  func(lib.Action, ...lib.OutputOption) lib.OutputConverter
	outputJSON func(lib.Action, []byte) (lib.OutputConverter, error)
}{
	{TypeGeoLite2CountryMMDBIn, DescGeoLite2CountryMMDBIn, DescGeoLite2CountryMMDBOut, defaultGeoLite2CountryMMDBFile, defaultMaxmindOutputDir, NewGeoLite2CountryMMDBIn, NewGeoLite2CountryMMDBInFromBytes, NewGeoLite2CountryMMDBOut, NewGeoLite2CountryMMDBOutFromBytes},
	{TypeDBIPCountryMMDBIn, DescDBIPCountryMMDBIn, DescDBIPCountryMMDBOut, defaultDBIPCountryMMDBFile, defaultDBIPOutputDir, NewDBIPCountryMMDBIn, NewDBIPCountryMMDBInFromBytes, NewDBIPCountryMMDBOut, NewDBIPCountryMMDBOutFromBytes},
	{TypeIPInfoCountryMMDBIn, DescIPInfoCountryMMDBIn, DescIPInfoCountryMMDBOut, defaultIPInfoCountryMMDBFile, defaultIPInfoOutputDir, NewIPInfoCountryMMDBIn, NewIPInfoCountryMMDBInFromBytes, NewIPInfoCountryMMDBOut, NewIPInfoCountryMMDBOutFromBytes},
}

var csvVariants = []struct {
	name      string
	desc      string
	ipv4      string
	ipv6      string
	newInput  func(lib.Action, ...lib.InputOption) lib.InputConverter
	inputJSON func(lib.Action, []byte) (lib.InputConverter, error)
}{
	{TypeGeoLite2CountryCSVIn, DescGeoLite2CountryCSVIn, defaultGeoLite2CountryIPv4File, defaultGeoLite2CountryIPv6File, NewGeoLite2CountryCSVIn, NewGeoLite2CountryCSVInFromBytes},
	{TypeGeoLite2ASNCSVIn, DescGeoLite2ASNCSVIn, defaultGeoLite2ASNCSVIPv4File, defaultGeoLite2ASNCSVIPv6File, NewGeoLite2ASNCSVIn, NewGeoLite2ASNCSVInFromBytes},
}

func assertEqual(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestMMDBConstructorDefaults(t *testing.T) {
	for _, variant := range mmdbVariants {
		t.Run(variant.name, func(t *testing.T) {
			wantInput := &GeoLite2CountryMMDBIn{
				Type: variant.name, Action: lib.ActionAdd, Description: variant.inputDesc,
				URI: variant.uri, Want: map[string]bool{},
			}
			assertEqual(t, variant.newInput(lib.ActionAdd), wantInput)
			assertEqual(t, variant.newInput(lib.ActionAdd, nil, WithURI(""), WithInputWantedList(nil), WithInputOnlyIPType("")), wantInput)
			assertEqual(t, variant.newInput(lib.ActionAdd, WithURI("custom.mmdb"), WithURI(" \t ")), wantInput)
			for _, data := range []string{"", "{}", "null", `{"uri":"","wantedList":[],"onlyIPType":""}`, `{"uri":" \t "}`} {
				got, err := variant.inputJSON(lib.ActionAdd, []byte(data))
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, got, wantInput)
			}

			wantOutput := &GeoLite2CountryMMDBOut{
				Type: variant.name, Action: lib.ActionOutput, Description: variant.outputDesc,
				OutputName: defaultGeoLite2CountryMMDBOutputName, OutputDir: variant.outputDir,
			}
			assertEqual(t, variant.newOutput(lib.ActionOutput), wantOutput)
			assertEqual(t, variant.newOutput(lib.ActionOutput, nil, WithOutputName(""), WithOutputDir(""),
				WithOutputWantedList(nil), WithOutputExcludedList(nil), WithOutputOverwriteList(nil),
				WithSourceMMDBURI(""), WithOutputOnlyIPType("")), wantOutput)
			assertEqual(t, variant.newOutput(lib.ActionOutput, WithOutputName("custom.mmdb"), WithOutputName(" "),
				WithOutputDir("custom"), WithOutputDir(" \t ")), wantOutput)
			for _, data := range []string{"", "{}", "null", `{"outputDir":"","outputName":"","onlyIPType":""}`, `{"outputDir":" \t ","outputName":" "}`} {
				got, err := variant.outputJSON(lib.ActionOutput, []byte(data))
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, got, wantOutput)
			}
		})
	}
}

func TestMMDBConstructorJSONParity(t *testing.T) {
	for _, variant := range mmdbVariants {
		for _, ipType := range []lib.IPType{"", lib.IPv4, lib.IPv6} {
			t.Run(variant.name+"/"+string(ipType), func(t *testing.T) {
				for _, action := range []lib.Action{lib.ActionAdd, lib.ActionRemove} {
					got := variant.newInput(action, WithURI(" custom.mmdb "), WithInputWantedList([]string{" cn ", "US", "", "cn"}), WithInputOnlyIPType(ipType))
					fromJSON, err := variant.inputJSON(action, []byte(fmt.Sprintf(`{"uri":" custom.mmdb ","wantedList":[" cn ","US","","cn"],"onlyIPType":%q}`, ipType)))
					if err != nil {
						t.Fatal(err)
					}
					assertEqual(t, got, fromJSON)
					assertEqual(t, got, &GeoLite2CountryMMDBIn{
						Type: variant.name, Action: action, Description: variant.inputDesc,
						URI: "custom.mmdb", Want: map[string]bool{"CN": true, "US": true}, OnlyIPType: ipType,
					})
				}

				want := &GeoLite2CountryMMDBOut{
					Type: variant.name, Action: lib.ActionOutput, Description: variant.outputDesc,
					OutputName: "custom.mmdb", OutputDir: "custom",
					Want: []string{"us", " cn "}, Overwrite: []string{"jp", "us"},
					Exclude: []string{" jp "}, OnlyIPType: ipType, SourceMMDBURI: "source.mmdb",
				}
				got := variant.newOutput(lib.ActionOutput, WithOutputName(" custom.mmdb "), WithOutputDir(" custom "),
					WithOutputWantedList(want.Want), WithOutputOverwriteList(want.Overwrite),
					WithOutputExcludedList(want.Exclude), WithOutputOnlyIPType(ipType), WithSourceMMDBURI(" source.mmdb "))
				fromJSON, err := variant.outputJSON(lib.ActionOutput, []byte(fmt.Sprintf(`{
					"outputName":" custom.mmdb ","outputDir":" custom ","wantedList":["us"," cn "],
					"overwriteList":["jp","us"],"excludedList":[" jp "],"onlyIPType":%q,"sourceMMDBURI":" source.mmdb "
				}`, ipType)))
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, got, fromJSON)
				assertEqual(t, got, want)
			})
		}
	}
}

func csvFiles(g lib.InputConverter) (string, string) {
	switch g := g.(type) {
	case *GeoLite2CountryCSVIn:
		return g.IPv4File, g.IPv6File
	case *GeoLite2ASNCSVIn:
		return g.IPv4File, g.IPv6File
	default:
		panic("not a CSV input")
	}
}

func TestCSVConstructorDefaults(t *testing.T) {
	for _, variant := range csvVariants {
		t.Run(variant.name, func(t *testing.T) {
			want := variant.newInput(lib.ActionAdd)
			assertEqual(t, want.GetType(), variant.name)
			assertEqual(t, want.GetAction(), lib.ActionAdd)
			assertEqual(t, want.GetDescription(), variant.desc)
			ipv4, ipv6 := csvFiles(want)
			assertEqual(t, ipv4, variant.ipv4)
			assertEqual(t, ipv6, variant.ipv6)
			if country, ok := want.(*GeoLite2CountryCSVIn); ok {
				assertEqual(t, country.CountryCodeFile, defaultGeoLite2CountryCodeFile)
				assertEqual(t, variant.newInput(lib.ActionAdd, WithCountryCodeFile(" \t ")), want)
			}
			assertEqual(t, variant.newInput(lib.ActionAdd, nil, WithIPv4File(""), WithIPv6File(""),
				WithInputWantedList(nil), WithInputOnlyIPType("")), want)
			assertEqual(t, variant.newInput(lib.ActionAdd, WithIPv4File("old.csv"), WithIPv4File(" "),
				WithIPv6File("\t")), want)
			for _, data := range []string{"", "{}", "null", `{"country":"","ipv4":"","ipv6":"","wantedList":[],"onlyIPType":""}`, `{"country":" ","ipv4":"\t","ipv6":" "}`} {
				got, err := variant.inputJSON(lib.ActionAdd, []byte(data))
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, got, want)
			}
		})
	}
}

func TestCSVConstructorJSONParity(t *testing.T) {
	for _, variant := range csvVariants {
		for _, files := range []struct{ ipv4, ipv6 string }{{"v4.csv", ""}, {"", "v6.csv"}, {"v4.csv", "v6.csv"}} {
			for _, ipType := range []lib.IPType{"", lib.IPv4, lib.IPv6} {
				t.Run(variant.name+"/"+files.ipv4+"/"+files.ipv6+"/"+string(ipType), func(t *testing.T) {
					for _, action := range []lib.Action{lib.ActionAdd, lib.ActionRemove} {
						opts := []lib.InputOption{WithIPv4File(" " + files.ipv4 + " "), WithIPv6File(" " + files.ipv6 + " "),
							WithInputWantedList([]string{" as123 ", "456", "", "AS123"}), WithInputOnlyIPType(ipType)}
						if variant.name == TypeGeoLite2CountryCSVIn {
							opts = append(opts, WithCountryCodeFile(" country.csv "))
						}
						got := variant.newInput(action, opts...)
						fromJSON, err := variant.inputJSON(action, []byte(fmt.Sprintf(`{
							"country":" country.csv ","ipv4":%q,"ipv6":%q,"wantedList":[" as123 ","456","","AS123"],"onlyIPType":%q
						}`, " "+files.ipv4+" ", " "+files.ipv6+" ", ipType)))
						if err != nil {
							t.Fatal(err)
						}
						assertEqual(t, got, fromJSON)
						ipv4, ipv6 := csvFiles(got)
						assertEqual(t, ipv4, files.ipv4)
						assertEqual(t, ipv6, files.ipv6)
						switch g := got.(type) {
						case *GeoLite2CountryCSVIn:
							assertEqual(t, g.CountryCodeFile, "country.csv")
							assertEqual(t, g.Want, map[string]bool{"AS123": true, "456": true})
							assertEqual(t, g.OnlyIPType, ipType)
						case *GeoLite2ASNCSVIn:
							assertEqual(t, g.Want, map[string][]string{"123": {"AS123"}, "456": {"AS456"}})
							assertEqual(t, g.OnlyIPType, ipType)
						}
					}
				})
			}
		}
	}
}

func TestASNWantedListExtended(t *testing.T) {
	lists := lib.WantedListExtended{TypeMap: map[string][]string{
		" cloud ": {" AS123 ", "456", "", "as"},
		" other ": {"as123"},
		" ":       {"789"},
	}}
	got := NewGeoLite2ASNCSVIn(lib.ActionAdd, WithInputWantedListExtended(lists)).(*GeoLite2ASNCSVIn)
	fromJSON, err := NewGeoLite2ASNCSVInFromBytes(lib.ActionAdd, []byte(`{
		"wantedList":{" cloud ":[" AS123 ","456","","as"]," other ":["as123"]," ":["789"]}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	// Multiple lists may share an ASN; map iteration does not define their order.
	slices.Sort(got.Want["123"])
	slices.Sort(fromJSON.(*GeoLite2ASNCSVIn).Want["123"])
	assertEqual(t, got, fromJSON)
	assertEqual(t, got.Want, map[string][]string{"123": {"CLOUD", "OTHER"}, "456": {"CLOUD"}})

	lists.TypeSlice = []string{" AS123 ", "789"}
	got = NewGeoLite2ASNCSVIn(lib.ActionAdd, WithInputWantedListExtended(lists)).(*GeoLite2ASNCSVIn)
	assertEqual(t, got.Want, map[string][]string{"123": {"AS123"}, "456": {"CLOUD"}, "789": {"AS789"}})
	assertEqual(t, NewGeoLite2ASNCSVIn(lib.ActionAdd, WithInputWantedListExtended(lib.WantedListExtended{})),
		NewGeoLite2ASNCSVIn(lib.ActionAdd))
}

func TestFromBytesRejectsMalformedJSON(t *testing.T) {
	inputs := []func(lib.Action, []byte) (lib.InputConverter, error){NewGeoLite2CountryCSVInFromBytes, NewGeoLite2ASNCSVInFromBytes}
	for _, variant := range mmdbVariants {
		inputs = append(inputs, variant.inputJSON)
		for _, data := range []string{`{`, `[]`, `{"outputName":1}`, `{"onlyIPType":1}`, `{"wantedList":[1]}`, `{"excludedList":false}`, `{"overwriteList":{}}`, `{"sourceMMDBURI":[]}`} {
			got, err := variant.outputJSON(lib.ActionOutput, []byte(data))
			if err == nil || got != nil {
				t.Fatalf("%s accepted invalid output JSON %s: %#v, %v", variant.name, data, got, err)
			}
		}
	}
	for _, inputJSON := range inputs {
		for _, data := range []string{`{`, `[]`, `{"onlyIPType":1}`, `{"wantedList":[1]}`, `{"wantedList":{"cloud":[123]}}`, `{"wantedList":true}`} {
			got, err := inputJSON(lib.ActionAdd, []byte(data))
			if err == nil || got != nil {
				t.Fatalf("accepted invalid input JSON %s: %#v, %v", data, got, err)
			}
		}
	}
}

func TestConstructorsRejectInvalidOptions(t *testing.T) {
	type fatalCase struct {
		name string
		want string
		run  func()
	}
	var cases []fatalCase
	addInputCases := func(name string, newInput func(lib.Action, ...lib.InputOption) lib.InputConverter, inputJSON func(lib.Action, []byte) (lib.InputConverter, error)) {
		for _, action := range []lib.Action{"", lib.ActionOutput, "invalid"} {
			cases = append(cases,
				fatalCase{name + "/input/action/" + string(action), "invalid action", func() { newInput(action) }},
				fatalCase{name + "/input/json/action/" + string(action), "invalid action", func() { inputJSON(action, nil) }})
		}
		for _, ipType := range []lib.IPType{"IPv4", "invalid", " "} {
			cases = append(cases,
				fatalCase{name + "/input/ip/" + string(ipType), "invalid onlyIPType", func() {
					newInput(lib.ActionAdd, WithInputOnlyIPType(ipType))
				}},
				fatalCase{name + "/input/json/ip/" + string(ipType), "invalid onlyIPType", func() {
					inputJSON(lib.ActionAdd, []byte(fmt.Sprintf(`{"onlyIPType":%q}`, ipType)))
				}})
		}
	}
	for _, variant := range csvVariants {
		addInputCases(variant.name, variant.newInput, variant.inputJSON)
	}
	for _, variant := range mmdbVariants {
		addInputCases(variant.name, variant.newInput, variant.inputJSON)
		for _, action := range []lib.Action{"", lib.ActionAdd, lib.ActionRemove, "invalid"} {
			cases = append(cases,
				fatalCase{variant.name + "/output/action/" + string(action), "invalid action", func() { variant.newOutput(action) }},
				fatalCase{variant.name + "/output/json/action/" + string(action), "invalid action", func() { variant.outputJSON(action, nil) }})
		}
		for _, ipType := range []lib.IPType{"IPv6", "invalid", " "} {
			cases = append(cases,
				fatalCase{variant.name + "/output/ip/" + string(ipType), "invalid onlyIPType", func() {
					variant.newOutput(lib.ActionOutput, WithOutputOnlyIPType(ipType))
				}},
				fatalCase{variant.name + "/output/json/ip/" + string(ipType), "invalid onlyIPType", func() {
					variant.outputJSON(lib.ActionOutput, []byte(fmt.Sprintf(`{"onlyIPType":%q}`, ipType)))
				}})
		}
	}

	if name := os.Getenv("GEOIP_MAXMIND_FATAL_CASE"); name != "" {
		for _, tc := range cases {
			if tc.name == name {
				tc.run()
				t.Fatal("constructor unexpectedly returned")
			}
		}
		t.Fatalf("unknown subprocess case %q", name)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestConstructorsRejectInvalidOptions$")
			cmd.Env = append(os.Environ(), "GEOIP_MAXMIND_FATAL_CASE="+tc.name)
			output, err := cmd.CombinedOutput()
			exitErr, ok := err.(*exec.ExitError)
			if !ok || exitErr.ExitCode() != 1 || !strings.Contains(string(output), tc.want) {
				t.Fatalf("expected log.Fatal containing %q, got %v: %s", tc.want, err, output)
			}
		})
	}
}
