package plaintext

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

var inputFormats = []struct {
	name        string
	description string
	new         func(lib.Action, ...lib.InputOption) lib.InputConverter
	fromBytes   func(lib.Action, []byte) (lib.InputConverter, error)
}{
	{TypeTextIn, DescTextIn, NewTextIn, NewTextInFromBytes},
	{TypeJSONIn, DescJSONIn, NewJSONIn, NewJSONInFromBytes},
	{TypeClashRuleSetIPCIDRIn, DescClashRuleSetIPCIDRIn, NewClashRuleSetIPCIDRIn, NewClashRuleSetIPCIDRInFromBytes},
	{TypeClashRuleSetClassicalIn, DescClashRuleSetClassicalIn, NewClashRuleSetClassicalIn, NewClashRuleSetClassicalInFromBytes},
	{TypeSurgeRuleSetIn, DescSurgeRuleSetIn, NewSurgeRuleSetIn, NewSurgeRuleSetInFromBytes},
}

var outputFormats = []struct {
	name        string
	description string
	defaultDir  string
	new         func(lib.Action, ...lib.OutputOption) lib.OutputConverter
	fromBytes   func(lib.Action, []byte) (lib.OutputConverter, error)
}{
	{TypeTextOut, DescTextOut, defaultOutputDirForTextOut, NewTextOut, NewTextOutFromBytes},
	{TypeClashRuleSetIPCIDROut, DescClashRuleSetIPCIDROut, defaultOutputDirForClashRuleSetIPCIDROut, NewClashRuleSetIPCIDROut, NewClashRuleSetIPCIDROutFromBytes},
	{TypeClashRuleSetClassicalOut, DescClashRuleSetClassicalOut, defaultOutputDirForClashRuleSetClassicalOut, NewClashRuleSetClassicalOut, NewClashRuleSetClassicalOutFromBytes},
	{TypeSurgeRuleSetOut, DescSurgeRuleSetOut, defaultOutputDirForSurgeRuleSetOut, NewSurgeRuleSetOut, NewSurgeRuleSetOutFromBytes},
}

func configBytes(t *testing.T, config any) []byte {
	t.Helper()
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestInputConstructorDefaults(t *testing.T) {
	for _, format := range inputFormats {
		t.Run(format.name, func(t *testing.T) {
			var paths []string
			if format.name == TypeJSONIn {
				paths = []string{"prefixes"}
			}
			opts := []lib.InputOption{WithInputDir(" ./input "), WithJSONPath(paths)}
			want := &textIn{
				Type: format.name, Action: lib.ActionAdd, Description: format.description,
				InputDir: "./input", Want: map[string]bool{}, JSONPath: paths,
			}
			for _, got := range []lib.InputConverter{
				format.new(lib.ActionAdd, opts...),
				format.new(lib.ActionAdd, append(opts, nil)...),
			} {
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("got %#v, want %#v", got, want)
				}
				if got.GetType() != format.name || got.GetDescription() != format.description || got.GetAction() != lib.ActionAdd {
					t.Fatalf("incorrect converter metadata: %#v", got)
				}
			}
			got, err := format.fromBytes(lib.ActionAdd, configBytes(t, map[string]any{
				"inputDir": " ./input ", "jsonPath": paths,
			}))
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("JSON defaults: got %#v, error %v, want %#v", got, err, want)
			}
		})
	}
}

func TestInputOptionsMatchJSON(t *testing.T) {
	for _, format := range inputFormats {
		t.Run(format.name, func(t *testing.T) {
			opts := []lib.InputOption{
				nil,
				WithNameAndURI(" cn ", " ./cn.txt "),
				WithInputWantedList([]string{" cn ", "", "US", "cn"}),
				WithInputOnlyIPType(lib.IPv6),
				WithJSONPath([]string{" prefixes "}),
				WithRemovePrefixesInLine([]string{" IP-CIDR,"}),
				WithRemoveSuffixesInLine([]string{",no-resolve "}),
			}
			data := map[string]any{
				"name": " cn ", "uri": " ./cn.txt ",
				"wantedList": []string{" cn ", "", "US", "cn"}, "onlyIPType": "ipv6",
				"jsonPath": []string{" prefixes "}, "removePrefixesInLine": []string{" IP-CIDR,"},
				"removeSuffixesInLine": []string{",no-resolve "},
			}
			if format.name == TypeTextIn {
				cidrs := []string{" 192.0.2.0/24 ", "2001:db8::/32"}
				opts = append(opts, WithNameAndIPOrCIDR(" cn ", cidrs))
				data["ipOrCIDR"] = cidrs
			}
			direct := format.new(lib.ActionRemove, opts...)
			parsed, err := format.fromBytes(lib.ActionRemove, configBytes(t, data))
			if err != nil || !reflect.DeepEqual(direct, parsed) {
				t.Fatalf("direct %#v differs from JSON %#v, error %v", direct, parsed, err)
			}
			got := direct.(*textIn)
			if got.Name != "cn" || got.URI != "./cn.txt" || !reflect.DeepEqual(got.Want, map[string]bool{"CN": true, "US": true}) {
				t.Fatalf("source/list normalization failed: %#v", got)
			}
		})
	}
}

func TestOutputConstructorDefaults(t *testing.T) {
	for _, format := range outputFormats {
		t.Run(format.name, func(t *testing.T) {
			want := &textOut{
				Type: format.name, Action: lib.ActionOutput, Description: format.description,
				OutputDir: format.defaultDir, OutputExt: ".txt",
			}
			for _, got := range []lib.OutputConverter{
				format.new(lib.ActionOutput),
				format.new(lib.ActionOutput, nil),
				format.new(lib.ActionOutput, WithOutputDir(" "), WithOutputExtension(" ")),
			} {
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("got %#v, want %#v", got, want)
				}
				if got.GetType() != format.name || got.GetDescription() != format.description || got.GetAction() != lib.ActionOutput {
					t.Fatalf("incorrect converter metadata: %#v", got)
				}
			}
			for _, data := range [][]byte{nil, {}, []byte("{}"), []byte("null")} {
				got, err := format.fromBytes(lib.ActionOutput, data)
				if err != nil || !reflect.DeepEqual(got, want) {
					t.Fatalf("JSON %q: got %#v, error %v, want %#v", data, got, err, want)
				}
			}
		})
	}
}

func TestOutputOptionsMatchJSON(t *testing.T) {
	for _, format := range outputFormats {
		t.Run(format.name, func(t *testing.T) {
			direct := format.new(lib.ActionOutput,
				nil,
				WithOutputDir(" ./custom "),
				WithOutputExtension(" .conf "),
				WithOutputWantedList([]string{" cn ", "us"}),
				WithOutputExcludedList([]string{" Us "}),
				WithOutputOnlyIPType(lib.IPv4),
				WithAddPrefixInLine("  IP-CIDR,"),
				WithAddSuffixInLine(",no-resolve  "),
			)
			parsed, err := format.fromBytes(lib.ActionOutput, []byte(`{
				"outputDir": " ./custom ", "outputExtension": " .conf ",
				"wantedList": [" cn ", "us"], "excludedList": [" Us "],
				"onlyIPType": "ipv4", "addPrefixInLine": "  IP-CIDR,",
				"addSuffixInLine": ",no-resolve  "
			}`))
			if err != nil || !reflect.DeepEqual(direct, parsed) {
				t.Fatalf("direct %#v differs from JSON %#v, error %v", direct, parsed, err)
			}
			got := direct.(*textOut)
			if got.OutputDir != "./custom" || got.OutputExt != ".conf" || got.AddPrefixInLine != "  IP-CIDR," || got.AddSuffixInLine != ",no-resolve  " {
				t.Fatalf("incorrect option normalization: %#v", got)
			}
		})
	}
}

func TestMalformedJSON(t *testing.T) {
	for _, data := range []string{"{", " ", "[]", `{"onlyIPType":42}`} {
		for _, format := range inputFormats {
			if got, err := format.fromBytes(lib.ActionAdd, []byte(data)); err == nil || got != nil {
				t.Errorf("%s input accepted malformed JSON %q: %#v, %v", format.name, data, got, err)
			}
		}
		for _, format := range outputFormats {
			if got, err := format.fromBytes(lib.ActionOutput, []byte(data)); err == nil || got != nil {
				t.Errorf("%s output accepted malformed JSON %q: %#v, %v", format.name, data, got, err)
			}
		}
	}
}

func TestConstructorValidation(t *testing.T) {
	type fatalCase struct {
		name string
		run  func()
		want string
	}
	var cases []fatalCase
	for _, format := range inputFormats {
		cases = append(cases,
			fatalCase{format.name + "/no-options", func() { format.new(lib.ActionAdd) }, "[type " + format.name},
			fatalCase{format.name + "/nil-option", func() { format.new(lib.ActionAdd, nil) }, "[type " + format.name},
		)
		for _, data := range []string{"", "null", "{}"} {
			cases = append(cases, fatalCase{
				format.name + "/missing-source-json-" + data,
				func() { _, _ = format.fromBytes(lib.ActionAdd, []byte(data)) },
				"[type " + format.name,
			})
		}
		for _, invalid := range []struct {
			name   string
			action lib.Action
			opts   []lib.InputOption
			json   string
			want   string
		}{
			{"name-only", lib.ActionAdd, []lib.InputOption{WithNameAndURI("cn", "")}, `{"name":"cn"}`, "missing uri"},
			{"uri-only", lib.ActionAdd, []lib.InputOption{WithNameAndURI("", "cn.txt")}, `{"uri":"cn.txt"}`, "missing inputDir or name"},
			{"blank-name", lib.ActionAdd, []lib.InputOption{WithNameAndURI(" ", "cn.txt")}, `{"name":" ","uri":"cn.txt"}`, "missing inputDir or name"},
			{"blank-uri", lib.ActionAdd, []lib.InputOption{WithNameAndURI("cn", " ")}, `{"name":"cn","uri":" "}`, "missing uri"},
			{"mixed-sources", lib.ActionAdd, []lib.InputOption{WithInputDir("./input"), WithNameAndURI("cn", "cn.txt")}, `{"inputDir":"./input","name":"cn","uri":"cn.txt"}`, "inputDir is not allowed"},
			{"directory-and-name", lib.ActionAdd, []lib.InputOption{WithInputDir("./input"), WithNameAndURI("cn", "")}, `{"inputDir":"./input","name":"cn"}`, "inputDir is not allowed"},
			{"directory-and-uri", lib.ActionAdd, []lib.InputOption{WithInputDir("./input"), WithNameAndURI("", "cn.txt")}, `{"inputDir":"./input","uri":"cn.txt"}`, "inputDir is not allowed"},
			{"invalid-ip-type", lib.ActionAdd, []lib.InputOption{WithInputDir("./input"), WithInputOnlyIPType("ipv5")}, `{"inputDir":"./input","onlyIPType":"ipv5"}`, "invalid onlyIPType"},
			{"output-action", lib.ActionOutput, []lib.InputOption{WithInputDir("./input")}, `{"inputDir":"./input"}`, "invalid input action"},
			{"empty-action", "", []lib.InputOption{WithInputDir("./input")}, `{"inputDir":"./input"}`, "invalid input action"},
		} {
			cases = append(cases, fatalCase{
				format.name + "/" + invalid.name,
				func() { format.new(invalid.action, append(invalid.opts, WithJSONPath([]string{"prefixes"}))...) },
				invalid.want,
			})
			cases = append(cases, fatalCase{
				format.name + "/" + invalid.name + "-json",
				func() {
					data := strings.TrimSuffix(invalid.json, "}") + `,"jsonPath":["prefixes"]}`
					if _, err := format.fromBytes(invalid.action, []byte(data)); err != nil {
						t.Fatalf("unexpected JSON error: %v", err)
					}
				},
				invalid.want,
			})
		}
		if format.name != TypeTextIn {
			cases = append(cases,
				fatalCase{
					format.name + "/inline-unsupported",
					func() { format.new(lib.ActionAdd, WithNameAndIPOrCIDR("cn", []string{"192.0.2.0/24"})) },
					"ipOrCIDR is invalid",
				},
				fatalCase{
					format.name + "/inline-unsupported-json",
					func() { _, _ = format.fromBytes(lib.ActionAdd, []byte(`{"name":"cn","ipOrCIDR":["192.0.2.0/24"]}`)) },
					"ipOrCIDR is invalid",
				},
			)
		}
	}
	cases = append(cases,
		fatalCase{"json/missing-path", func() { NewJSONIn(lib.ActionAdd, WithNameAndURI("cn", "cn.json")) }, "missing jsonPath"},
		fatalCase{"json/missing-path-json", func() { _, _ = NewJSONInFromBytes(lib.ActionAdd, []byte(`{"name":"cn","uri":"cn.json"}`)) }, "missing jsonPath"},
		fatalCase{"json/empty-path-list", func() {
			NewJSONIn(lib.ActionAdd, WithInputDir("./input"), WithJSONPath([]string{}))
		}, "missing jsonPath"},
		fatalCase{"json/empty-path-list-json", func() {
			_, _ = NewJSONInFromBytes(lib.ActionAdd, []byte(`{"inputDir":"./input","jsonPath":[]}`))
		}, "missing jsonPath"},
		fatalCase{"text/inline-missing-name", func() { NewTextIn(lib.ActionAdd, WithNameAndIPOrCIDR("", []string{"192.0.2.1"})) }, "missing inputDir or name"},
		fatalCase{"text/inline-missing-name-json", func() {
			_, _ = NewTextInFromBytes(lib.ActionAdd, []byte(`{"ipOrCIDR":["192.0.2.1"]}`))
		}, "missing inputDir or name"},
		fatalCase{"text/inline-with-directory", func() {
			NewTextIn(lib.ActionAdd, WithNameAndIPOrCIDR("cn", []string{"192.0.2.1"}), WithInputDir("./input"))
		}, "inputDir is not allowed"},
		fatalCase{"text/inline-with-directory-json", func() {
			_, _ = NewTextInFromBytes(lib.ActionAdd, []byte(`{"name":"cn","ipOrCIDR":["192.0.2.1"],"inputDir":"./input"}`))
		}, "inputDir is not allowed"},
	)
	for _, format := range outputFormats {
		for _, action := range []lib.Action{"", lib.ActionAdd, lib.ActionRemove, "unknown"} {
			cases = append(cases, fatalCase{
				format.name + "/output-action-" + string(action),
				func() { format.new(action) },
				"invalid output action",
			})
		}
		cases = append(cases,
			fatalCase{format.name + "/output-ip-type", func() { format.new(lib.ActionOutput, WithOutputOnlyIPType("ipv5")) }, "invalid onlyIPType"},
			fatalCase{format.name + "/output-ip-type-json", func() {
				_, _ = format.fromBytes(lib.ActionOutput, []byte(`{"onlyIPType":"ipv5"}`))
			}, "invalid onlyIPType"},
		)
	}

	const env = "GEOIP_PLAINTEXT_FATAL_CASE"
	if name := os.Getenv(env); name != "" {
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
			cmd.Env = append(os.Environ(), env+"="+tc.name)
			output, err := cmd.CombinedOutput()
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
				t.Fatalf("expected log.Fatal exit code 1, got %v; output: %s", err, output)
			}
			if !strings.Contains(string(output), tc.want) {
				t.Fatalf("expected %q in fatal output: %s", tc.want, output)
			}
		})
	}
}
