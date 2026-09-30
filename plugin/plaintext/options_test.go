package plaintext

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
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
	{TypeClashRuleSetClassicalIn, DescClashRuleSetClassicalIn, NewClashRuleSetClassicalIn, NewClashRuleSetClassicalInFromBytes},
	{TypeClashRuleSetIPCIDRIn, DescClashRuleSetIPCIDRIn, NewClashRuleSetIPCIDRIn, NewClashRuleSetIPCIDRInFromBytes},
	{TypeSurgeRuleSetIn, DescSurgeRuleSetIn, NewSurgeRuleSetIn, NewSurgeRuleSetInFromBytes},
}

var outputFormats = []struct {
	name        string
	description string
	defaultDir  string
	new         func(lib.Action, ...lib.OutputOption) lib.OutputConverter
	fromBytes   func(lib.Action, []byte) (lib.OutputConverter, error)
}{
	{TypeTextOut, DescTextOut, filepath.Join("output", "text"), NewTextOut, NewTextOutFromBytes},
	{TypeClashRuleSetClassicalOut, DescClashRuleSetClassicalOut, filepath.Join("output", "clash", "classical"), NewClashRuleSetClassicalOut, NewClashRuleSetClassicalOutFromBytes},
	{TypeClashRuleSetIPCIDROut, DescClashRuleSetIPCIDROut, filepath.Join("output", "clash", "ipcidr"), NewClashRuleSetIPCIDROut, NewClashRuleSetIPCIDROutFromBytes},
	{TypeSurgeRuleSetOut, DescSurgeRuleSetOut, filepath.Join("output", "surge"), NewSurgeRuleSetOut, NewSurgeRuleSetOutFromBytes},
}

func marshalArgs(t *testing.T, args map[string]any) []byte {
	t.Helper()
	data, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestInputConstructors(t *testing.T) {
	for _, format := range inputFormats {
		t.Run(format.name, func(t *testing.T) {
			for _, action := range []lib.Action{lib.ActionAdd, lib.ActionRemove} {
				for _, directory := range []bool{false, true} {
					opts := []lib.InputOption{nil}
					args := make(map[string]any)
					if directory {
						opts = append(opts, WithInputDir(" inputs "))
						args["inputDir"] = " inputs "
					} else {
						opts = append(opts, WithNameAndURI(" example ", " source.txt "))
						args["name"], args["uri"] = " example ", " source.txt "
					}
					if format.name == TypeJSONIn {
						opts = append(opts, WithJSONPath([]string{"addresses"}))
						args["jsonPath"] = []string{"addresses"}
					}
					direct := format.new(action, opts...).(*textIn)
					adapted, err := format.fromBytes(action, marshalArgs(t, args))
					if err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(direct, adapted) {
						t.Fatalf("constructor/JSON mismatch: %#v != %#v", direct, adapted)
					}
					if direct.GetType() != format.name || direct.GetDescription() != format.description || direct.GetAction() != action {
						t.Fatalf("incorrect metadata: %#v", direct)
					}
					if direct.OnlyIPType != "" || len(direct.Want) != 0 {
						t.Fatalf("incorrect input defaults: %#v", direct)
					}
					if directory {
						if direct.InputDir != "inputs" || direct.Name != "" || direct.URI != "" {
							t.Fatalf("incorrect directory source: %#v", direct)
						}
					} else if direct.Name != "example" || direct.URI != "source.txt" || direct.InputDir != "" {
						t.Fatalf("incorrect named source: %#v", direct)
					}
				}
			}
		})
	}
}

func TestInputOptionJSONParity(t *testing.T) {
	for _, format := range inputFormats {
		t.Run(format.name, func(t *testing.T) {
			want := []string{" cn ", "", "CN", " us "}
			prefixes, suffixes := []string{" \tHOST, "}, []string{" ,no-resolve\t "}
			paths := []string{" addresses ", "ipv6_addresses"}
			opts := []lib.InputOption{
				WithNameAndURI("old", "old.txt"),
				nil,
				WithNameAndURI(" example ", " source.txt "),
				WithInputWantedList(want),
				WithInputOnlyIPType(" IPv6 "),
				WithJSONPath(paths),
				WithRemovePrefixesInLine(prefixes),
				WithRemoveSuffixesInLine(suffixes),
			}
			args := map[string]any{
				"name": " example ", "uri": " source.txt ",
				"wantedList": want, "onlyIPType": " IPv6 ", "jsonPath": paths,
				"removePrefixesInLine": prefixes, "removeSuffixesInLine": suffixes,
			}
			if format.name == TypeTextIn {
				opts = append(opts, WithIPOrCIDR([]string{"192.0.2.1"}))
				args["ipOrCIDR"] = []string{"192.0.2.1"}
			}
			direct := format.new(lib.ActionAdd, opts...).(*textIn)
			adapted, err := format.fromBytes(lib.ActionAdd, marshalArgs(t, args))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(direct, adapted) {
				t.Fatalf("constructor/JSON mismatch: %#v != %#v", direct, adapted)
			}
			if direct.Name != "example" || direct.URI != "source.txt" || direct.OnlyIPType != lib.IPv6 {
				t.Fatalf("options not normalized: %#v", direct)
			}
			if !reflect.DeepEqual(direct.Want, map[string]bool{"CN": true, "US": true}) {
				t.Fatalf("wantedList not normalized: %v", direct.Want)
			}
			if !reflect.DeepEqual(direct.JSONPath, paths) || !reflect.DeepEqual(direct.RemovePrefixesInLine, prefixes) || !reflect.DeepEqual(direct.RemoveSuffixesInLine, suffixes) {
				t.Fatalf("paths or affixes changed: %#v", direct)
			}
		})
	}
}

func TestOutputDefaults(t *testing.T) {
	for _, format := range outputFormats {
		t.Run(format.name, func(t *testing.T) {
			direct := format.new(lib.ActionOutput).(*textOut)
			if direct.GetType() != format.name || direct.GetDescription() != format.description || direct.GetAction() != lib.ActionOutput {
				t.Fatalf("incorrect metadata: %#v", direct)
			}
			if direct.OutputDir != format.defaultDir || direct.OutputExt != ".txt" || direct.OnlyIPType != "" || len(direct.Want) != 0 || len(direct.Exclude) != 0 || direct.AddPrefixInLine != "" || direct.AddSuffixInLine != "" {
				t.Fatalf("incorrect output defaults: %#v", direct)
			}
			withEmptyOptions := format.new(lib.ActionOutput, nil, WithOutputDir(" \t"), WithOutputExtension(" \t"))
			if !reflect.DeepEqual(direct, withEmptyOptions) {
				t.Fatalf("empty options changed defaults: %#v", withEmptyOptions)
			}
			for _, data := range [][]byte{nil, {}, []byte(`{}`), []byte(`null`), []byte(`{"outputDir":"","outputExtension":""}`)} {
				adapted, err := format.fromBytes(lib.ActionOutput, data)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(direct, adapted) {
					t.Fatalf("JSON defaults differ for %q: %#v", data, adapted)
				}
			}
		})
	}
}

func TestOutputOptionJSONParity(t *testing.T) {
	for _, format := range outputFormats {
		t.Run(format.name, func(t *testing.T) {
			want, exclude := []string{" cn ", "US"}, []string{" us "}
			prefix, suffix := " \tHOST, ", " ,no-resolve\t "
			direct := format.new(lib.ActionOutput,
				WithOutputDir("old"),
				nil,
				WithOutputDir(" exports "),
				WithOutputExtension(" .conf "),
				WithOutputWantedList(want),
				WithOutputExcludedList(exclude),
				WithOutputOnlyIPType(" IPv4 "),
				WithAddPrefixInLine(prefix),
				WithAddSuffixInLine(suffix),
			).(*textOut)
			adapted, err := format.fromBytes(lib.ActionOutput, marshalArgs(t, map[string]any{
				"outputDir": " exports ", "outputExtension": " .conf ",
				"wantedList": want, "excludedList": exclude, "onlyIPType": " IPv4 ",
				"addPrefixInLine": prefix, "addSuffixInLine": suffix,
			}))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(direct, adapted) {
				t.Fatalf("constructor/JSON mismatch: %#v != %#v", direct, adapted)
			}
			if direct.OutputDir != "exports" || direct.OutputExt != ".conf" || direct.OnlyIPType != lib.IPv4 {
				t.Fatalf("options not normalized: %#v", direct)
			}
			if !reflect.DeepEqual(direct.Want, want) || !reflect.DeepEqual(direct.Exclude, exclude) || direct.AddPrefixInLine != prefix || direct.AddSuffixInLine != suffix {
				t.Fatalf("lists or affixes changed: %#v", direct)
			}
		})
	}
}

func TestMalformedJSON(t *testing.T) {
	for _, format := range inputFormats {
		t.Run("input/"+format.name, func(t *testing.T) {
			for _, data := range []string{`{`, `[]`, `"text"`, `42`, ` `, `{"name":false}`, `{"onlyIPType":[]}`, `{"jsonPath":"addresses"}`} {
				if converter, err := format.fromBytes(lib.ActionAdd, []byte(data)); err == nil || converter != nil {
					t.Fatalf("expected parse error and nil converter for %q, got %v, %v", data, converter, err)
				}
			}
		})
	}
	for _, format := range outputFormats {
		t.Run("output/"+format.name, func(t *testing.T) {
			for _, data := range []string{`{`, `[]`, `"text"`, `42`, ` `, `{"outputDir":false}`, `{"onlyIPType":[]}`, `{"wantedList":"cn"}`} {
				if converter, err := format.fromBytes(lib.ActionOutput, []byte(data)); err == nil || converter != nil {
					t.Fatalf("expected parse error and nil converter for %q, got %v, %v", data, converter, err)
				}
			}
		})
	}
}

func TestConstructorFatalValidation(t *testing.T) {
	type fatalCase struct {
		name    string
		message string
		run     func()
	}
	var cases []fatalCase
	inputCases := []struct {
		name    string
		action  lib.Action
		opts    []lib.InputOption
		args    map[string]any
		message string
	}{
		{"missing-source", lib.ActionAdd, nil, nil, "missing inputDir or name"},
		{"blank-source", lib.ActionAdd, []lib.InputOption{WithNameAndURI(" ", "\t")}, map[string]any{"name": " ", "uri": "\t"}, "missing inputDir or name"},
		{"missing-name", lib.ActionAdd, []lib.InputOption{WithNameAndURI("", "source")}, map[string]any{"uri": "source"}, "missing inputDir or name"},
		{"missing-uri", lib.ActionAdd, []lib.InputOption{WithNameAndURI("cn", "")}, map[string]any{"name": "cn"}, "name and uri"},
		{"directory-and-source", lib.ActionAdd, []lib.InputOption{WithInputDir("source"), WithNameAndURI("cn", "file")}, map[string]any{"inputDir": "source", "name": "cn", "uri": "file"}, "inputDir is not allowed"},
		{"directory-and-name", lib.ActionAdd, []lib.InputOption{WithInputDir("source"), WithNameAndURI("cn", "")}, map[string]any{"inputDir": "source", "name": "cn"}, "inputDir is not allowed"},
		{"directory-and-uri", lib.ActionAdd, []lib.InputOption{WithInputDir("source"), WithNameAndURI("", "file")}, map[string]any{"inputDir": "source", "uri": "file"}, "inputDir is not allowed"},
		{"invalid-ip-type", lib.ActionAdd, []lib.InputOption{WithInputDir("source"), WithInputOnlyIPType("ip4")}, map[string]any{"inputDir": "source", "onlyIPType": "ip4"}, "invalid onlyIPType"},
		{"output-action", lib.ActionOutput, []lib.InputOption{WithInputDir("source")}, map[string]any{"inputDir": "source"}, "only supports add or remove action"},
		{"empty-action", "", []lib.InputOption{WithInputDir("source")}, map[string]any{"inputDir": "source"}, "only supports add or remove action"},
		{"unknown-action", "other", []lib.InputOption{WithInputDir("source")}, map[string]any{"inputDir": "source"}, "only supports add or remove action"},
	}
	for _, format := range inputFormats {
		for _, tc := range inputCases {
			opts := append([]lib.InputOption{nil}, tc.opts...)
			args := make(map[string]any)
			for key, value := range tc.args {
				args[key] = value
			}
			if format.name == TypeJSONIn {
				opts = append(opts, WithJSONPath([]string{"addresses"}))
				args["jsonPath"] = []string{"addresses"}
			}
			data := marshalArgs(t, args)
			cases = append(cases,
				fatalCase{"input/" + format.name + "/" + tc.name, tc.message, func() { format.new(tc.action, opts...) }},
				fatalCase{"input-json/" + format.name + "/" + tc.name, tc.message, func() { _, _ = format.fromBytes(tc.action, data) }},
			)
		}
		if format.name != TypeTextIn {
			cases = append(cases,
				fatalCase{"input/" + format.name + "/inline", "ipOrCIDR is invalid", func() {
					format.new(lib.ActionAdd, WithNameAndURI("cn", "source"), WithIPOrCIDR([]string{"192.0.2.1"}), WithJSONPath([]string{"addresses"}))
				}},
				fatalCase{"input/" + format.name + "/named-inline", "ipOrCIDR is invalid", func() {
					format.new(lib.ActionAdd, WithNameAndIPOrCIDR("cn", []string{"192.0.2.1"}), WithJSONPath([]string{"addresses"}))
				}},
				fatalCase{"input-json/" + format.name + "/inline", "ipOrCIDR is invalid", func() {
					_, _ = format.fromBytes(lib.ActionAdd, []byte(`{"name":"cn","uri":"source","ipOrCIDR":["192.0.2.1"],"jsonPath":["addresses"]}`))
				}},
				fatalCase{"input-json/" + format.name + "/named-inline", "ipOrCIDR is invalid", func() {
					_, _ = format.fromBytes(lib.ActionAdd, []byte(`{"name":"cn","ipOrCIDR":["192.0.2.1"],"jsonPath":["addresses"]}`))
				}},
			)
		}
		cases = append(cases, fatalCase{"input-json/" + format.name + "/nil", "", func() {
			_, _ = format.fromBytes(lib.ActionAdd, nil)
		}})
	}
	for _, paths := range [][]string{nil, {}, {""}, {" \t "}, {"addresses", " "}} {
		cases = append(cases,
			fatalCase{"input/json/path-" + string(marshalArgs(t, map[string]any{"paths": paths})), "jsonPath", func() {
				NewJSONIn(lib.ActionAdd, WithInputDir("source"), WithJSONPath(paths))
			}},
			fatalCase{"input-json/json/path-" + string(marshalArgs(t, map[string]any{"paths": paths})), "jsonPath", func() {
				_, _ = NewJSONInFromBytes(lib.ActionAdd, marshalArgs(t, map[string]any{"inputDir": "source", "jsonPath": paths}))
			}},
		)
	}
	cases = append(cases,
		fatalCase{"input/text/unnamed-inline", "missing inputDir or name", func() {
			NewTextIn(lib.ActionAdd, WithNameAndIPOrCIDR(" ", []string{"192.0.2.1"}))
		}},
		fatalCase{"input/text/unpaired-inline", "missing inputDir or name", func() {
			NewTextIn(lib.ActionAdd, WithIPOrCIDR([]string{"192.0.2.1"}))
		}},
		fatalCase{"input/text/empty-inline", "name and ipOrCIDR", func() {
			NewTextIn(lib.ActionAdd, WithNameAndIPOrCIDR("cn", nil))
		}},
		fatalCase{"input/text/half-uri-pair-with-inline", "name and uri", func() {
			NewTextIn(lib.ActionAdd, WithNameAndURI("cn", ""), WithIPOrCIDR([]string{"192.0.2.1"}))
		}},
		fatalCase{"input/text/directory-inline", "inputDir is not allowed", func() {
			NewTextIn(lib.ActionAdd, WithInputDir("source"), WithIPOrCIDR([]string{"192.0.2.1"}))
		}},
		fatalCase{"input/text/directory-named-inline", "inputDir is not allowed", func() {
			NewTextIn(lib.ActionAdd, WithInputDir("source"), WithNameAndIPOrCIDR("cn", []string{"192.0.2.1"}))
		}},
		fatalCase{"input-json/text/unnamed-inline", "missing inputDir or name", func() {
			_, _ = NewTextInFromBytes(lib.ActionAdd, []byte(`{"ipOrCIDR":["192.0.2.1"]}`))
		}},
		fatalCase{"input-json/text/uri-unnamed-inline", "missing inputDir or name", func() {
			_, _ = NewTextInFromBytes(lib.ActionAdd, []byte(`{"uri":"source","ipOrCIDR":["192.0.2.1"]}`))
		}},
		fatalCase{"input-json/text/empty-inline", "name and uri", func() {
			_, _ = NewTextInFromBytes(lib.ActionAdd, []byte(`{"name":"cn","ipOrCIDR":[]}`))
		}},
		fatalCase{"input-json/text/directory-inline", "inputDir is not allowed", func() {
			_, _ = NewTextInFromBytes(lib.ActionAdd, []byte(`{"inputDir":"source","ipOrCIDR":["192.0.2.1"]}`))
		}},
		fatalCase{"input-json/text/directory-named-inline", "inputDir is not allowed", func() {
			_, _ = NewTextInFromBytes(lib.ActionAdd, []byte(`{"inputDir":"source","name":"cn","ipOrCIDR":["192.0.2.1"]}`))
		}},
	)
	for _, format := range outputFormats {
		for _, action := range []lib.Action{"", lib.ActionAdd, lib.ActionRemove, "other"} {
			cases = append(cases,
				fatalCase{"output/" + format.name + "/action-" + string(action), "only supports output action", func() {
					format.new(action)
				}},
				fatalCase{"output-json/" + format.name + "/action-" + string(action), "only supports output action", func() {
					_, _ = format.fromBytes(action, nil)
				}},
			)
		}
		cases = append(cases,
			fatalCase{"output/" + format.name + "/ip-type", "invalid onlyIPType", func() {
				format.new(lib.ActionOutput, WithOutputOnlyIPType("ip4"))
			}},
			fatalCase{"output-json/" + format.name + "/ip-type", "invalid onlyIPType", func() {
				_, _ = format.fromBytes(lib.ActionOutput, []byte(`{"onlyIPType":"ip4"}`))
			}},
		)
	}

	const childKey = "GEOIP_PLAINTEXT_FATAL_CASE"
	if childCase := os.Getenv(childKey); childCase != "" {
		for _, tc := range cases {
			if tc.name == childCase {
				tc.run()
				return
			}
		}
		t.Fatalf("unknown fatal test case %q", childCase)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestConstructorFatalValidation$")
			cmd.Env = append(os.Environ(), childKey+"="+tc.name)
			output, err := cmd.CombinedOutput()
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
				t.Fatalf("expected log.Fatal exit code 1, got %v\n%s", err, output)
			}
			if !strings.Contains(string(output), "❌ [type ") || !strings.Contains(string(output), tc.message) {
				t.Fatalf("expected constructor diagnostic %q, got %s", tc.message, output)
			}
		})
	}
}

func TestNamedInlineTextCompatibility(t *testing.T) {
	for _, action := range []lib.Action{lib.ActionAdd, lib.ActionRemove} {
		direct := NewTextIn(action, nil, WithNameAndIPOrCIDR(" cn ", []string{"1.0.0.1", "1.0.0.1/24"}))
		adapted, err := NewTextInFromBytes(action, []byte(`{"name":" cn ","ipOrCIDR":["1.0.0.1","1.0.0.1/24"]}`))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(direct, adapted) {
			t.Fatalf("inline constructor/JSON mismatch: %#v != %#v", direct, adapted)
		}
		for _, input := range []lib.InputConverter{direct, adapted} {
			container := lib.NewContainer()
			if action == lib.ActionRemove {
				container, err = NewTextIn(lib.ActionAdd, WithNameAndIPOrCIDR("cn", []string{"1.0.0.0/23"})).Input(container)
				if err != nil {
					t.Fatal(err)
				}
			}
			container, err = input.Input(container)
			if err != nil {
				t.Fatal(err)
			}
			want := []string{"1.0.0.0/24"}
			if action == lib.ActionRemove {
				want = []string{"1.0.1.0/24"}
			}
			assertEntryCIDRs(t, container, "CN", want)
		}
	}
}

func assertEntryCIDRs(t *testing.T, container lib.Container, name string, want []string) {
	t.Helper()
	entry, found := container.GetEntry(name)
	if !found {
		t.Fatalf("missing entry %s", name)
	}
	got, err := entry.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s CIDRs: got %v, want %v", name, got, want)
	}
}

func localTestDir(t *testing.T) string {
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

func TestLocalTextConversions(t *testing.T) {
	dir := localTestDir(t)
	source := filepath.Join(dir, "source.txt")
	if err := os.WriteFile(source, []byte("# comment\nHOST,192.0.2.0/24,no-resolve\nHOST,2001:db8::/32,no-resolve\n"), 0600); err != nil {
		t.Fatal(err)
	}
	direct := NewTextIn(lib.ActionAdd,
		WithNameAndURI(" sample ", source),
		WithIPOrCIDR([]string{"198.51.100.1"}),
		WithRemovePrefixesInLine([]string{"HOST,"}),
		WithRemoveSuffixesInLine([]string{",no-resolve"}),
		WithInputOnlyIPType(" IPv4 "),
	)
	adapted, err := NewTextInFromBytes(lib.ActionAdd, marshalArgs(t, map[string]any{
		"name": " sample ", "uri": source, "ipOrCIDR": []string{"198.51.100.1"},
		"removePrefixesInLine": []string{"HOST,"}, "removeSuffixesInLine": []string{",no-resolve"},
		"onlyIPType": " IPv4 ",
	}))
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []lib.InputConverter{direct, adapted} {
		container, err := input.Input(lib.NewContainer())
		if err != nil {
			t.Fatal(err)
		}
		assertEntryCIDRs(t, container, "SAMPLE", []string{"192.0.2.0/24", "198.51.100.1/32"})
		prefix, suffix := " \tHOST, ", " ; \t"
		output := NewTextOut(lib.ActionOutput,
			WithOutputDir(filepath.Join(dir, "output")),
			WithOutputExtension(".list"),
			WithOutputWantedList([]string{" sample ", "missing"}),
			WithOutputExcludedList([]string{"missing"}),
			WithAddPrefixInLine(prefix),
			WithAddSuffixInLine(suffix),
		)
		if err := output.Output(container); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(dir, "output", "sample.list"))
		if err != nil {
			t.Fatal(err)
		}
		want := prefix + "192.0.2.0/24" + suffix + "\n" + prefix + "198.51.100.1/32" + suffix + "\n"
		if string(data) != want {
			t.Fatalf("text output: got %q, want %q", data, want)
		}
	}
}

func TestLocalDirectoryAndFormatConversions(t *testing.T) {
	dir := localTestDir(t)
	for _, name := range []string{"cn", "us"} {
		if err := os.WriteFile(filepath.Join(dir, name+".txt"), []byte("192.0.2.0/24\n2001:db8::/32\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	container, err := NewTextIn(lib.ActionAdd, WithInputDir(dir), WithInputWantedList([]string{" cn "}), WithInputOnlyIPType(" IPv6 ")).Input(lib.NewContainer())
	if err != nil {
		t.Fatal(err)
	}
	if container.Len() != 1 {
		t.Fatalf("directory selection produced %d entries", container.Len())
	}
	assertEntryCIDRs(t, container, "CN", []string{"2001:db8::/32"})

	for _, format := range outputFormats {
		t.Run(format.name, func(t *testing.T) {
			outputDir := filepath.Join(dir, format.name)
			output, err := format.fromBytes(lib.ActionOutput, marshalArgs(t, map[string]any{"outputDir": outputDir}))
			if err != nil {
				t.Fatal(err)
			}
			if err := output.Output(container); err != nil {
				t.Fatal(err)
			}
			for _, inputFormat := range inputFormats {
				if inputFormat.name != format.name {
					continue
				}
				input, err := inputFormat.fromBytes(lib.ActionAdd, marshalArgs(t, map[string]any{"name": "cn", "uri": filepath.Join(outputDir, "cn.txt")}))
				if err != nil {
					t.Fatal(err)
				}
				roundTrip, err := input.Input(lib.NewContainer())
				if err != nil {
					t.Fatal(err)
				}
				assertEntryCIDRs(t, roundTrip, "CN", []string{"2001:db8::/32"})
			}
		})
	}
	jsonPath := filepath.Join(dir, "source.json")
	if err := os.WriteFile(jsonPath, []byte(`{"addresses":["192.0.2.0/24"],"ipv6_addresses":["2001:db8::/32"]}`), 0600); err != nil {
		t.Fatal(err)
	}
	jsonInput := NewJSONIn(lib.ActionAdd, WithNameAndURI("json", jsonPath), WithJSONPath([]string{"addresses", "ipv6_addresses"}))
	jsonContainer, err := jsonInput.Input(lib.NewContainer())
	if err != nil {
		t.Fatal(err)
	}
	assertEntryCIDRs(t, jsonContainer, "JSON", []string{"192.0.2.0/24", "2001:db8::/32"})
}
