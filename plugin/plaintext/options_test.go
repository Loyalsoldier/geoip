package plaintext

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

type inputFormat struct {
	name        string
	description string
	new         func(lib.Action, ...lib.InputOption) lib.InputConverter
	fromBytes   func(lib.Action, []byte) (lib.InputConverter, error)
	content     string
}

var inputFormats = []inputFormat{
	{TypeTextIn, DescTextIn, NewTextIn, NewTextInFromBytes, "10.0.0.0/24\n2001:db8::/32\n"},
	{TypeJSONIn, DescJSONIn, NewJSONIn, NewJSONInFromBytes, `{"prefixes":["10.0.0.0/24","2001:db8::/32"]}`},
	{TypeClashRuleSetClassicalIn, DescClashRuleSetClassicalIn, NewClashRuleSetClassicalIn, NewClashRuleSetClassicalInFromBytes, "payload:\n  - IP-CIDR,10.0.0.0/24,no-resolve\n  - IP-CIDR6,2001:db8::/32\n  - DOMAIN,example.com\n"},
	{TypeClashRuleSetIPCIDRIn, DescClashRuleSetIPCIDRIn, NewClashRuleSetIPCIDRIn, NewClashRuleSetIPCIDRInFromBytes, "payload:\n  - '10.0.0.0/24'\n  - '2001:db8::/32'\n"},
	{TypeSurgeRuleSetIn, DescSurgeRuleSetIn, NewSurgeRuleSetIn, NewSurgeRuleSetInFromBytes, "# comment\nIP-CIDR,10.0.0.0/24,no-resolve\nIP-CIDR6,2001:db8::/32\nDOMAIN,example.com\n"},
}

type outputFormat struct {
	name        string
	description string
	new         func(lib.Action, ...lib.OutputOption) lib.OutputConverter
	fromBytes   func(lib.Action, []byte) (lib.OutputConverter, error)
	defaultDir  string
	content     string
	ipv6Content string
}

var outputFormats = []outputFormat{
	{TypeTextOut, DescTextOut, NewTextOut, NewTextOutFromBytes, defaultOutputDirForTextOut, "10.0.0.0/24\n2001:db8::/32\n", "2001:db8::/32\n"},
	{TypeClashRuleSetClassicalOut, DescClashRuleSetClassicalOut, NewClashRuleSetClassicalOut, NewClashRuleSetClassicalOutFromBytes, defaultOutputDirForClashRuleSetClassicalOut, "payload:\n  - IP-CIDR,10.0.0.0/24\n  - IP-CIDR6,2001:db8::/32\n", "payload:\n  - IP-CIDR6,2001:db8::/32\n"},
	{TypeClashRuleSetIPCIDROut, DescClashRuleSetIPCIDROut, NewClashRuleSetIPCIDROut, NewClashRuleSetIPCIDROutFromBytes, defaultOutputDirForClashRuleSetIPCIDROut, "payload:\n  - '10.0.0.0/24'\n  - '2001:db8::/32'\n", "payload:\n  - '2001:db8::/32'\n"},
	{TypeSurgeRuleSetOut, DescSurgeRuleSetOut, NewSurgeRuleSetOut, NewSurgeRuleSetOutFromBytes, defaultOutputDirForSurgeRuleSetOut, "IP-CIDR,10.0.0.0/24\nIP-CIDR6,2001:db8::/32\n", "IP-CIDR6,2001:db8::/32\n"},
}

func TestInputConstructorParity(t *testing.T) {
	for _, format := range inputFormats {
		for _, action := range []lib.Action{lib.ActionAdd, lib.ActionRemove} {
			t.Run(format.name+"/"+string(action), func(t *testing.T) {
				got := format.new(action,
					nil,
					WithNameAndURI("unused", "unused"),
					WithNameAndURI(" cn ", " data.txt "),
					WithInputWantedList([]string{" cn ", "", "US", "cn"}),
					WithInputOnlyIPType(lib.IPv6),
					WithJSONPath([]string{" prefixes "}),
					WithRemovePrefixesInLine([]string{" IP-CIDR, "}),
					WithRemoveSuffixesInLine([]string{",no-resolve "}),
					nil,
				)
				adapted, err := format.fromBytes(action, []byte(`{
					"name":" cn ", "uri":" data.txt ",
					"wantedList":[" cn ","","US","cn"], "onlyIPType":"ipv6",
					"jsonPath":[" prefixes "],
					"removePrefixesInLine":[" IP-CIDR, "], "removeSuffixesInLine":[",no-resolve "]
				}`))
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, adapted) {
					t.Fatalf("options = %#v; JSON = %#v", got, adapted)
				}
				if got.GetType() != format.name || got.GetDescription() != format.description || got.GetAction() != action {
					t.Fatalf("unexpected metadata: %#v", got)
				}
				input := got.(*TextIn)
				if input.Name != "cn" || input.URI != "data.txt" || input.OnlyIPType != lib.IPv6 {
					t.Fatalf("unexpected source: %#v", input)
				}
				if !reflect.DeepEqual(input.Want, map[string]bool{"CN": true, "US": true}) {
					t.Fatalf("wanted list = %#v", input.Want)
				}
			})
		}
		t.Run(format.name+"/directory", func(t *testing.T) {
			got := format.new(lib.ActionAdd, nil, WithInputDir(" input "), WithJSONPath([]string{"prefixes"}))
			adapted, err := format.fromBytes(lib.ActionAdd, []byte(`{"inputDir":" input ","jsonPath":["prefixes"]}`))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, adapted) || got.(*TextIn).InputDir != "input" {
				t.Fatalf("options = %#v; JSON = %#v", got, adapted)
			}
		})
	}
}

func TestTextInlineConstructorParity(t *testing.T) {
	for _, uri := range []string{"", " data.txt "} {
		t.Run(uri, func(t *testing.T) {
			opts := []lib.InputOption{WithNameAndIPOrCIDR(" cn ", []string{"1.1.1.1", "2001:db8::/32"})}
			if uri != "" {
				opts = []lib.InputOption{WithNameAndURI(" cn ", uri), WithIPOrCIDR([]string{"1.1.1.1", "2001:db8::/32"})}
			}
			got := NewTextIn(lib.ActionAdd, opts...)
			data := marshalConfig(t, map[string]any{"name": " cn ", "uri": uri, "ipOrCIDR": []string{"1.1.1.1", "2001:db8::/32"}})
			adapted, err := NewTextInFromBytes(lib.ActionAdd, data)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, adapted) {
				t.Fatalf("options = %#v; JSON = %#v", got, adapted)
			}
		})
	}
}

func TestOutputConstructorParity(t *testing.T) {
	for _, format := range outputFormats {
		t.Run(format.name, func(t *testing.T) {
			got := format.new(lib.ActionOutput,
				nil,
				WithOutputDir(" unused "),
				WithOutputDir(" custom "),
				WithOutputExtension(" .rules "),
				WithOutputWantedList([]string{" cn ", "US", ""}),
				WithOutputExcludedList([]string{" us "}),
				WithOutputOnlyIPType(lib.IPv4),
				WithAddPrefixInLine(" prefix "),
				WithAddSuffixInLine(" suffix "),
				nil,
			)
			adapted, err := format.fromBytes(lib.ActionOutput, []byte(`{
				"outputDir":" custom ", "outputExtension":" .rules ",
				"wantedList":[" cn ","US",""], "excludedList":[" us "], "onlyIPType":"ipv4",
				"addPrefixInLine":" prefix ", "addSuffixInLine":" suffix "
			}`))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, adapted) {
				t.Fatalf("options = %#v; JSON = %#v", got, adapted)
			}
			if got.GetType() != format.name || got.GetDescription() != format.description || got.GetAction() != lib.ActionOutput {
				t.Fatalf("unexpected metadata: %#v", got)
			}
			output := got.(*TextOut)
			if output.OutputDir != "custom" || output.OutputExt != ".rules" || output.AddPrefixInLine != " prefix " || output.AddSuffixInLine != " suffix " {
				t.Fatalf("unexpected output options: %#v", output)
			}
		})
	}
}

func TestOutputDefaults(t *testing.T) {
	for _, format := range outputFormats {
		t.Run(format.name, func(t *testing.T) {
			want := format.new(lib.ActionOutput)
			if output := want.(*TextOut); output.OutputDir != format.defaultDir || output.OutputExt != ".txt" {
				t.Fatalf("unexpected defaults: %#v", output)
			}
			for _, opts := range [][]lib.OutputOption{
				{nil},
				{WithOutputDir(""), WithOutputExtension("")},
				{WithOutputDir(" \t "), WithOutputExtension("\n")},
				{WithOutputDir("custom"), WithOutputDir(""), WithOutputExtension(".rules"), WithOutputExtension("")},
			} {
				if got := format.new(lib.ActionOutput, opts...); !reflect.DeepEqual(got, want) {
					t.Errorf("options = %#v; defaults = %#v", got, want)
				}
			}
			for _, data := range [][]byte{nil, {}, []byte(`{}`), []byte(`null`), []byte(`{"outputDir":"","outputExtension":""}`), []byte(`{"outputDir":" \t ","outputExtension":" "}`)} {
				got, err := format.fromBytes(lib.ActionOutput, data)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("JSON %s = %#v; defaults = %#v", data, got, want)
				}
			}
		})
	}
}

func TestMalformedJSON(t *testing.T) {
	for _, format := range inputFormats {
		t.Run(format.name+"/input", func(t *testing.T) {
			for _, data := range []string{`{`, `[]`, `{"name":3}`, `{"wantedList":"cn"}`, `{"ipOrCIDR":1}`, `{"jsonPath":"prefixes"}`} {
				if got, err := format.fromBytes(lib.ActionAdd, []byte(data)); err == nil || got != nil {
					t.Errorf("JSON %s: got %v, error %v", data, got, err)
				}
			}
		})
	}
	for _, format := range outputFormats {
		t.Run(format.name+"/output", func(t *testing.T) {
			for _, data := range []string{`{`, `[]`, `{"outputDir":3}`, `{"wantedList":"cn"}`, `{"excludedList":1}`} {
				if got, err := format.fromBytes(lib.ActionOutput, []byte(data)); err == nil || got != nil {
					t.Errorf("JSON %s: got %v, error %v", data, got, err)
				}
			}
		})
	}
}

func TestInvalidConstructors(t *testing.T) {
	type fatalCase struct {
		name    string
		message string
		run     func()
	}
	var cases []fatalCase
	for _, format := range inputFormats {
		message := "name and uri"
		if format.name == TypeJSONIn {
			message = "jsonPath"
		}
		cases = append(cases, fatalCase{format.name + "/no-options", message, func() { format.new(lib.ActionAdd) }})
		for _, data := range [][]byte{nil, []byte(`{}`), []byte(`null`)} {
			cases = append(cases, fatalCase{format.name + "/empty-json/" + string(data), message, func() {
				if _, err := format.fromBytes(lib.ActionAdd, data); err != nil {
					t.Fatalf("unexpected JSON error: %v", err)
				}
			}})
		}
		type inputCase struct {
			name    string
			action  lib.Action
			opts    []lib.InputOption
			args    map[string]any
			message string
		}
		invalid := []inputCase{
			{"no-source", lib.ActionAdd, nil, nil, "name and uri"},
			{"blank-name", lib.ActionAdd, []lib.InputOption{WithNameAndURI(" \t ", "input")}, map[string]any{"name": " \t ", "uri": "input"}, "name and uri"},
			{"blank-uri", lib.ActionAdd, []lib.InputOption{WithNameAndURI("cn", " \t ")}, map[string]any{"name": "cn", "uri": " \t "}, "name and uri"},
			{"blank-directory", lib.ActionAdd, []lib.InputOption{WithInputDir(" \t ")}, map[string]any{"inputDir": " \t "}, "name and uri"},
			{"mixed-directory", lib.ActionAdd, []lib.InputOption{WithInputDir("input"), WithNameAndURI("cn", "file")}, map[string]any{"inputDir": "input", "name": "cn", "uri": "file"}, "inputDir is not allowed"},
			{"directory-with-name", lib.ActionAdd, []lib.InputOption{WithInputDir("input"), WithNameAndURI("cn", "")}, map[string]any{"inputDir": "input", "name": "cn"}, "inputDir is not allowed"},
			{"directory-with-uri", lib.ActionAdd, []lib.InputOption{WithInputDir("input"), WithNameAndURI("", "file")}, map[string]any{"inputDir": "input", "uri": "file"}, "inputDir is not allowed"},
			{"ip-type", lib.ActionAdd, []lib.InputOption{WithInputDir("input"), WithInputOnlyIPType("invalid")}, map[string]any{"inputDir": "input", "onlyIPType": "invalid"}, "onlyIPType"},
		}
		for _, action := range []lib.Action{"", lib.ActionOutput, "invalid"} {
			invalid = append(invalid, inputCase{"action-" + string(action), action, []lib.InputOption{WithInputDir("input")}, map[string]any{"inputDir": "input"}, "action must be add or remove"})
		}
		if format.name == TypeTextIn {
			invalid = append(invalid,
				inputCase{"inline-without-name", lib.ActionAdd, []lib.InputOption{WithIPOrCIDR([]string{"1.1.1.1"})}, map[string]any{"ipOrCIDR": []string{"1.1.1.1"}}, "name is required"},
				inputCase{"blank-inline", lib.ActionAdd, []lib.InputOption{WithNameAndIPOrCIDR("cn", []string{" \t "})}, map[string]any{"name": "cn", "ipOrCIDR": []string{" \t "}}, "blank values"},
				inputCase{"empty-inline", lib.ActionAdd, []lib.InputOption{WithNameAndIPOrCIDR("cn", nil)}, map[string]any{"name": "cn", "ipOrCIDR": []string{}}, "name and uri"},
				inputCase{"directory-with-inline", lib.ActionAdd, []lib.InputOption{WithInputDir("input"), WithIPOrCIDR([]string{"1.1.1.1"})}, map[string]any{"inputDir": "input", "ipOrCIDR": []string{"1.1.1.1"}}, "inputDir is not allowed"},
			)
		} else {
			invalid = append(invalid, inputCase{"inline", lib.ActionAdd, []lib.InputOption{WithNameAndURI("cn", "file"), WithIPOrCIDR([]string{"1.1.1.1"})}, map[string]any{"name": "cn", "uri": "file", "ipOrCIDR": []string{"1.1.1.1"}}, "ipOrCIDR is invalid"})
		}
		for _, test := range invalid {
			args := test.args
			if args == nil {
				args = make(map[string]any)
			}
			opts := test.opts
			if format.name == TypeJSONIn {
				args["jsonPath"] = []string{"prefixes"}
				opts = append(opts, WithJSONPath([]string{"prefixes"}))
			}
			data := marshalConfig(t, args)
			cases = append(cases,
				fatalCase{format.name + "/" + test.name + "/options", test.message, func() { format.new(test.action, opts...) }},
				fatalCase{format.name + "/" + test.name + "/json", test.message, func() {
					if _, err := format.fromBytes(test.action, data); err != nil {
						t.Fatalf("unexpected JSON error: %v", err)
					}
				}},
			)
		}
	}
	for _, paths := range [][]string{nil, {}} {
		data := marshalConfig(t, map[string]any{"inputDir": "input", "jsonPath": paths})
		cases = append(cases,
			fatalCase{"json/paths/" + string(data) + "/options", "jsonPath", func() { NewJSONIn(lib.ActionAdd, WithInputDir("input"), WithJSONPath(paths)) }},
			fatalCase{"json/paths/" + string(data) + "/json", "jsonPath", func() {
				if _, err := NewJSONInFromBytes(lib.ActionAdd, data); err != nil {
					t.Fatalf("unexpected JSON error: %v", err)
				}
			}},
		)
	}
	for _, format := range outputFormats {
		for _, action := range []lib.Action{"", lib.ActionAdd, lib.ActionRemove, "invalid"} {
			cases = append(cases,
				fatalCase{format.name + "/output/action-" + string(action) + "/options", "action must be output", func() { format.new(action) }},
				fatalCase{format.name + "/output/action-" + string(action) + "/json", "action must be output", func() {
					if _, err := format.fromBytes(action, nil); err != nil {
						t.Fatalf("unexpected JSON error: %v", err)
					}
				}},
			)
		}
		cases = append(cases,
			fatalCase{format.name + "/output/ip-type/options", "onlyIPType", func() { format.new(lib.ActionOutput, WithOutputOnlyIPType("invalid")) }},
			fatalCase{format.name + "/output/ip-type/json", "onlyIPType", func() {
				if _, err := format.fromBytes(lib.ActionOutput, []byte(`{"onlyIPType":"invalid"}`)); err != nil {
					t.Fatalf("unexpected JSON error: %v", err)
				}
			}},
		)
	}

	const fatalCaseEnv = "GEOIP_PLAINTEXT_FATAL_CASE"
	if name := os.Getenv(fatalCaseEnv); name != "" {
		for _, test := range cases {
			if test.name == name {
				test.run()
				return
			}
		}
		t.Fatalf("unknown fatal case %q", name)
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestInvalidConstructors$")
			cmd.Env = append(os.Environ(), fatalCaseEnv+"="+test.name)
			output, err := cmd.CombinedOutput()
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 1 || !strings.Contains(string(output), test.message) {
				t.Fatalf("wanted fatal %q, got %v:\n%s", test.message, err, output)
			}
		})
	}
}

func TestInputConversionAndRegistration(t *testing.T) {
	for _, format := range inputFormats {
		t.Run(format.name, func(t *testing.T) {
			dir := testDataDir(t)
			for _, name := range []string{"cn", "us"} {
				writeTestFile(t, filepath.Join(dir, name+".txt"), format.content)
			}
			opts := []lib.InputOption{
				WithInputDir(dir),
				WithInputWantedList([]string{" cn ", ""}),
				WithJSONPath([]string{"prefixes"}),
			}
			want := []string{"10.0.0.0/24", "2001:db8::/32"}
			got, err := format.new(lib.ActionAdd, opts...).Input(lib.NewContainer())
			if err != nil {
				t.Fatal(err)
			}
			assertPrefixes(t, got, "CN", want)
			if got.Len() != 1 {
				t.Fatalf("wanted one entry, got %d", got.Len())
			}
			data := marshalConfig(t, map[string]any{"input": []any{map[string]any{
				"type": format.name, "action": "add",
				"args": map[string]any{"inputDir": dir, "wantedList": []string{" cn ", ""}, "jsonPath": []string{"prefixes"}},
			}}})
			instance, err := lib.NewInstance()
			if err != nil {
				t.Fatal(err)
			}
			if err := instance.InitConfigFromBytes(data); err != nil {
				t.Fatal(err)
			}
			container := lib.NewContainer()
			if err := instance.RunInput(container); err != nil {
				t.Fatal(err)
			}
			assertPrefixes(t, container, "CN", want)
			if container.Len() != 1 {
				t.Fatalf("registered input wanted one entry, got %d", container.Len())
			}
		})
	}
}

func TestJSONInputEmptyKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.json")
	writeTestFile(t, path, `{"":["192.0.2.0/24"],"prefixes":["2001:db8::/32"]}`)
	for _, test := range []struct {
		name  string
		paths []string
		want  []string
	}{
		{"empty", []string{""}, []string{"192.0.2.0/24"}},
		{"trimmed", []string{" \t "}, []string{"192.0.2.0/24"}},
		{"mixed", []string{"prefixes", ""}, []string{"192.0.2.0/24", "2001:db8::/32"}},
	} {
		for _, source := range []string{"options", "json"} {
			t.Run(test.name+"/"+source, func(t *testing.T) {
				var input lib.InputConverter
				if source == "json" {
					data := marshalConfig(t, map[string]any{"name": "test", "uri": path, "jsonPath": test.paths})
					var err error
					input, err = NewJSONInFromBytes(lib.ActionAdd, data)
					if err != nil {
						t.Fatal(err)
					}
				} else {
					input = NewJSONIn(lib.ActionAdd, WithNameAndURI("test", path), WithJSONPath(test.paths))
				}
				container, err := input.Input(lib.NewContainer())
				if err != nil {
					t.Fatal(err)
				}
				assertPrefixes(t, container, "test", test.want)
			})
		}
	}
}

func TestTextSourcesAndLineOptions(t *testing.T) {
	dir := testDataDir(t)
	path := filepath.Join(dir, "cn.txt")
	writeTestFile(t, path, "# comment\nIP-CIDR,10.0.0.0/24,no-resolve // comment\n")
	for _, useJSON := range []bool{false, true} {
		var input lib.InputConverter
		if useJSON {
			data := marshalConfig(t, map[string]any{
				"name": "cn", "uri": path, "ipOrCIDR": []string{"2001:db8::/32"},
				"removePrefixesInLine": []string{"IP-CIDR,"}, "removeSuffixesInLine": []string{",no-resolve"},
			})
			var err error
			input, err = NewTextInFromBytes(lib.ActionAdd, data)
			if err != nil {
				t.Fatal(err)
			}
		} else {
			input = NewTextIn(lib.ActionAdd, WithNameAndURI("cn", path), WithIPOrCIDR([]string{"2001:db8::/32"}),
				WithRemovePrefixesInLine([]string{"IP-CIDR,"}), WithRemoveSuffixesInLine([]string{",no-resolve"}))
		}
		container, err := input.Input(lib.NewContainer())
		if err != nil {
			t.Fatal(err)
		}
		assertPrefixes(t, container, "CN", []string{"10.0.0.0/24", "2001:db8::/32"})
		remove := NewTextIn(lib.ActionRemove,
			WithNameAndIPOrCIDR("cn", []string{"10.0.0.0/24", "2001:db8::/32"}),
			WithInputOnlyIPType(lib.IPv4))
		if _, err := remove.Input(container); err != nil {
			t.Fatal(err)
		}
		assertPrefixes(t, container, "CN", []string{"2001:db8::/32"})
	}

	inline := NewTextIn(lib.ActionAdd, WithNameAndIPOrCIDR("cn", []string{"1.1.1.1", "2001:db8::1"}),
		WithInputWantedList([]string{"other"}), WithInputOnlyIPType(lib.IPv6))
	adapted, err := NewTextInFromBytes(lib.ActionAdd, []byte(`{
		"name":"cn", "ipOrCIDR":["1.1.1.1","2001:db8::1"], "wantedList":["other"], "onlyIPType":"ipv6"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []lib.InputConverter{inline, adapted} {
		container, err := input.Input(lib.NewContainer())
		if err != nil {
			t.Fatal(err)
		}
		assertPrefixes(t, container, "CN", []string{"2001:db8::1/128"})
	}
}

func TestOutputConversionAndRegistration(t *testing.T) {
	for _, format := range outputFormats {
		t.Run(format.name, func(t *testing.T) {
			container, err := NewTextIn(lib.ActionAdd, WithNameAndIPOrCIDR("cn", []string{"10.0.0.0/24", "2001:db8::/32"})).Input(lib.NewContainer())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := NewTextIn(lib.ActionAdd, WithNameAndIPOrCIDR("us", []string{"1.1.1.1"})).Input(container); err != nil {
				t.Fatal(err)
			}
			entry, _ := container.GetEntry("cn")
			data, err := format.new(lib.ActionOutput).(*TextOut).marshalBytes(entry)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != format.content {
				t.Fatalf("output = %q; want %q", data, format.content)
			}
			for _, useJSON := range []bool{false, true} {
				dir := testDataDir(t)
				if useJSON {
					config := marshalConfig(t, map[string]any{"output": []any{map[string]any{
						"type": format.name, "action": "output",
						"args": map[string]any{"outputDir": dir, "outputExtension": ".rules", "wantedList": []string{"US", " cn ", ""}, "excludedList": []string{" us "}, "onlyIPType": "ipv6"},
					}}})
					instance, err := lib.NewInstance()
					if err != nil {
						t.Fatal(err)
					}
					if err := instance.InitConfigFromBytes(config); err != nil {
						t.Fatal(err)
					}
					if err := instance.RunOutput(container); err != nil {
						t.Fatal(err)
					}
				} else {
					output := format.new(lib.ActionOutput, WithOutputDir(dir), WithOutputExtension(".rules"),
						WithOutputWantedList([]string{"US", " cn ", ""}), WithOutputExcludedList([]string{" us "}), WithOutputOnlyIPType(lib.IPv6))
					if err := output.Output(container); err != nil {
						t.Fatal(err)
					}
				}
				data, err := os.ReadFile(filepath.Join(dir, "cn.rules"))
				if err != nil {
					t.Fatal(err)
				}
				if string(data) != format.ipv6Content {
					t.Fatalf("output = %q; want %q", data, format.ipv6Content)
				}
				files, err := os.ReadDir(dir)
				if err != nil {
					t.Fatal(err)
				}
				if len(files) != 1 {
					t.Fatalf("wanted one output file, got %d", len(files))
				}
			}
			filtered := format.new(lib.ActionOutput, WithOutputWantedList([]string{"cn"}), WithOutputExcludedList([]string{"CN"})).(*TextOut)
			if list := filtered.filterAndSortList(container); len(list) != 0 {
				t.Fatalf("fully excluded wanted list = %v", list)
			}
		})
	}
}

func TestOutputLineOptions(t *testing.T) {
	entry := lib.NewEntry("cn")
	if err := entry.AddPrefix("10.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		output lib.OutputConverter
		want   string
	}{
		{NewTextOut(lib.ActionOutput, WithAddPrefixInLine(" IP-CIDR, "), WithAddSuffixInLine(" ,no-resolve ")), " IP-CIDR, 10.0.0.0/24 ,no-resolve \n"},
		{NewSurgeRuleSetOut(lib.ActionOutput, WithAddSuffixInLine(",no-resolve")), "IP-CIDR,10.0.0.0/24,no-resolve\n"},
	} {
		got, err := test.output.(*TextOut).marshalBytes(entry)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != test.want {
			t.Fatalf("output = %q; want %q", got, test.want)
		}
	}
}

func marshalConfig(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func testDataDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp(".", ".plaintext-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	})
	return dir
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func assertPrefixes(t *testing.T, container lib.Container, name string, want []string) {
	t.Helper()
	entry, found := container.GetEntry(name)
	if !found {
		t.Fatalf("entry %s not found", name)
	}
	got, err := entry.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s prefixes = %v; want %v", name, got, want)
	}
}
