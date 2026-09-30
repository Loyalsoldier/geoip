package special

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestInputOptions(t *testing.T) {
	tests := []struct {
		name      string
		action    lib.Action
		new       func(lib.Action, ...lib.InputOption) lib.InputConverter
		fromBytes func(lib.Action, []byte) (lib.InputConverter, error)
		required  []lib.InputOption
		opts      []lib.InputOption
		config    string
		want      lib.InputConverter
	}{
		{
			name: TypeStdin, action: lib.ActionAdd, new: NewStdin, fromBytes: NewStdinFromBytes,
			required: []lib.InputOption{WithName("example")},
			opts:     []lib.InputOption{WithName(" example "), WithInputOnlyIPType(lib.IPv6)},
			config:   `{"name":" example ","onlyIPType":"ipv6"}`,
			want:     &stdin{Type: TypeStdin, Action: lib.ActionAdd, Description: DescStdin, Name: "example", OnlyIPType: lib.IPv6},
		},
		{
			name: TypeCutter, action: lib.ActionRemove, new: NewCutter, fromBytes: NewCutterFromBytes,
			required: []lib.InputOption{WithInputWantedList([]string{"example"})},
			opts:     []lib.InputOption{WithInputWantedList([]string{" example ", "", "EXAMPLE"}), WithInputOnlyIPType(lib.IPv4)},
			config:   `{"wantedList":[" example ","","EXAMPLE"],"onlyIPType":"ipv4"}`,
			want:     &cutter{Type: TypeCutter, Action: lib.ActionRemove, Description: DescCutter, Want: map[string]bool{"EXAMPLE": true}, OnlyIPType: lib.IPv4},
		},
		{
			name: TypePrivate, action: lib.ActionRemove, new: NewPrivate, fromBytes: NewPrivateFromBytes,
			opts:   []lib.InputOption{WithInputOnlyIPType(lib.IPv6)},
			config: `{"onlyIPType":"ipv6"}`,
			want:   &private{Type: TypePrivate, Action: lib.ActionRemove, Description: DescPrivate, OnlyIPType: lib.IPv6},
		},
		{
			name: typeTest, action: lib.ActionAdd, new: NewTest, fromBytes: NewTestFromBytes,
			config: `{}`,
			want:   &test{Type: typeTest, Action: lib.ActionAdd, Description: descTest},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			direct := tt.new(tt.action, append(tt.opts, nil)...)
			configured, err := tt.fromBytes(tt.action, []byte(tt.config))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(direct, tt.want) || !reflect.DeepEqual(configured, tt.want) {
				t.Fatalf("direct = %#v, configured = %#v, want %#v", direct, configured, tt.want)
			}

			defaults := tt.new(tt.action, tt.required...)
			withNil := tt.new(tt.action, append(tt.required, nil)...)
			if !reflect.DeepEqual(defaults, withNil) {
				t.Fatalf("nil option changed defaults: %#v != %#v", defaults, withNil)
			}
			switch c := defaults.(type) {
			case *stdin:
				if c.OnlyIPType != "" {
					t.Fatalf("default onlyIPType = %q", c.OnlyIPType)
				}
			case *cutter:
				if c.OnlyIPType != "" {
					t.Fatalf("default onlyIPType = %q", c.OnlyIPType)
				}
			case *private:
				if c.OnlyIPType != "" {
					t.Fatalf("default onlyIPType = %q", c.OnlyIPType)
				}
			}
			if len(tt.required) == 0 {
				for _, data := range [][]byte{nil, []byte("null"), []byte("{}")} {
					configured, err := tt.fromBytes(tt.action, data)
					if err != nil || !reflect.DeepEqual(defaults, configured) {
						t.Fatalf("default config %q = %#v, %v; want %#v", data, configured, err, defaults)
					}
				}
			}
		})
	}
}

func TestOutputOptions(t *testing.T) {
	stdoutOptions := []lib.OutputOption{
		WithOutputWantedList([]string{" example ", "", "other"}),
		WithOutputExcludedList([]string{"other"}),
		WithOutputOnlyIPType(lib.IPv4),
		nil,
	}
	directStdout := NewStdout(lib.ActionOutput, stdoutOptions...)
	configuredStdout, err := NewStdoutFromBytes(lib.ActionOutput, []byte(`{"wantedList":[" example ","","other"],"excludedList":["other"],"onlyIPType":"ipv4"}`))
	if err != nil {
		t.Fatal(err)
	}
	wantStdout := &stdout{
		Type: TypeStdout, Action: lib.ActionOutput, Description: DescStdout,
		Want: []string{" example ", "", "other"}, Exclude: []string{"other"}, OnlyIPType: lib.IPv4,
	}
	if !reflect.DeepEqual(directStdout, wantStdout) || !reflect.DeepEqual(configuredStdout, wantStdout) {
		t.Fatalf("stdout direct = %#v, configured = %#v, want %#v", directStdout, configuredStdout, wantStdout)
	}

	directLookup := NewLookup(lib.ActionOutput, nil, WithSearch(" 127.0.0.1 "), WithSearchList([]string{" example "}))
	configuredLookup, err := NewLookupFromBytes(lib.ActionOutput, []byte(`{"search":" 127.0.0.1 ","searchList":[" example "]}`))
	if err != nil {
		t.Fatal(err)
	}
	wantLookup := &lookup{
		Type: TypeLookup, Action: lib.ActionOutput, Description: DescLookup,
		Search: "127.0.0.1", SearchList: []string{" example "},
	}
	if !reflect.DeepEqual(directLookup, wantLookup) || !reflect.DeepEqual(configuredLookup, wantLookup) {
		t.Fatalf("lookup direct = %#v, configured = %#v, want %#v", directLookup, configuredLookup, wantLookup)
	}

	defaultStdout := NewStdout(lib.ActionOutput).(*stdout)
	nilStdout := NewStdout(lib.ActionOutput, nil)
	if !reflect.DeepEqual(defaultStdout, nilStdout) || defaultStdout.OnlyIPType != "" || defaultStdout.Exclude != nil || defaultStdout.Want != nil {
		t.Fatalf("unexpected stdout defaults: %#v, %#v", defaultStdout, nilStdout)
	}
	defaultLookup := NewLookup(lib.ActionOutput, WithSearch("127.0.0.1")).(*lookup)
	nilLookup := NewLookup(lib.ActionOutput, nil, WithSearch("127.0.0.1"))
	if !reflect.DeepEqual(defaultLookup, nilLookup) || defaultLookup.SearchList != nil {
		t.Fatalf("unexpected lookup defaults: %#v, %#v", defaultLookup, nilLookup)
	}
}

func TestStdoutDefaults(t *testing.T) {
	container := lib.NewContainer()
	for name, prefixes := range map[string][]string{
		"zulu":  {"198.51.100.0/24"},
		"alpha": {"192.0.2.0/24", "2001:db8::/32"},
	} {
		entry := lib.NewEntry(name)
		for _, prefix := range prefixes {
			if err := entry.AddPrefix(prefix); err != nil {
				t.Fatal(err)
			}
		}
		if err := container.Add(entry); err != nil {
			t.Fatal(err)
		}
	}

	converters := map[string]lib.OutputConverter{
		"no options":     NewStdout(lib.ActionOutput),
		"nil option":     NewStdout(lib.ActionOutput, nil),
		"nil wanted":     NewStdout(lib.ActionOutput, WithOutputWantedList(nil)),
		"empty wanted":   NewStdout(lib.ActionOutput, WithOutputWantedList([]string{})),
		"blank wanted":   NewStdout(lib.ActionOutput, WithOutputWantedList([]string{"", " \t"})),
		"empty excluded": NewStdout(lib.ActionOutput, WithOutputExcludedList(nil)),
	}
	for _, data := range [][]byte{nil, []byte(""), []byte("null"), []byte("{}"), []byte(`{"wantedList":[]}`)} {
		configured, err := NewStdoutFromBytes(lib.ActionOutput, data)
		if err != nil {
			t.Fatal(err)
		}
		converters["config "+string(data)] = configured
	}
	for name, converter := range converters {
		t.Run(name, func(t *testing.T) {
			output := captureOutput(t, func() error { return converter.Output(container) })
			if want := "192.0.2.0/24\n2001:db8::/32\n198.51.100.0/24\n"; output != want {
				t.Fatalf("default stdout output = %q, want %q", output, want)
			}
		})
	}
}

func TestConstructorValidation(t *testing.T) {
	tests := []struct {
		name    string
		run     func()
		message string
	}{
		{"stdin missing name", func() { NewStdin(lib.ActionAdd) }, "missing name"},
		{"stdin nil option", func() { NewStdin(lib.ActionAdd, nil) }, "missing name"},
		{"stdin blank name", func() { NewStdin(lib.ActionAdd, WithName(" \t")) }, "missing name"},
		{"cutter missing list", func() { NewCutter(lib.ActionRemove) }, "wantedList"},
		{"cutter nil option", func() { NewCutter(lib.ActionRemove, nil) }, "wantedList"},
		{"cutter blank list", func() { NewCutter(lib.ActionRemove, WithInputWantedList([]string{"", " \t"})) }, "wantedList"},
		{"lookup missing search", func() { NewLookup(lib.ActionOutput) }, "search target"},
		{"lookup nil option", func() { NewLookup(lib.ActionOutput, nil) }, "search target"},
		{"lookup blank search", func() { NewLookup(lib.ActionOutput, WithSearch(" \t")) }, "search target"},
		{"stdin invalid action", func() { NewStdin(lib.ActionOutput, WithName("example")) }, "action"},
		{"private invalid action", func() { NewPrivate(lib.ActionOutput) }, "action"},
		{"test invalid action", func() { NewTest(lib.ActionOutput) }, "action"},
		{"cutter invalid action", func() { NewCutter(lib.ActionAdd, WithInputWantedList([]string{"example"})) }, "action"},
		{"stdout invalid action", func() { NewStdout(lib.ActionAdd, WithOutputWantedList([]string{"example"})) }, "action"},
		{"lookup invalid action", func() { NewLookup(lib.ActionRemove, WithSearch("127.0.0.1")) }, "action"},
		{"stdin invalid IP type", func() { NewStdin(lib.ActionAdd, WithName("example"), WithInputOnlyIPType("invalid")) }, "onlyIPType"},
		{"cutter invalid IP type", func() {
			NewCutter(lib.ActionRemove, WithInputWantedList([]string{"example"}), WithInputOnlyIPType("invalid"))
		}, "onlyIPType"},
		{"private invalid IP type", func() { NewPrivate(lib.ActionAdd, WithInputOnlyIPType("invalid")) }, "onlyIPType"},
		{"stdout invalid IP type", func() {
			NewStdout(lib.ActionOutput, WithOutputWantedList([]string{"example"}), WithOutputOnlyIPType("invalid"))
		}, "onlyIPType"},
		{"stdin empty config", func() { NewStdinFromBytes(lib.ActionAdd, nil) }, "missing name"},
		{"cutter empty config", func() { NewCutterFromBytes(lib.ActionRemove, []byte("{}")) }, "wantedList"},
		{"lookup empty config", func() { NewLookupFromBytes(lib.ActionOutput, nil) }, "search target"},
		{"private invalid config", func() { NewPrivateFromBytes(lib.ActionAdd, []byte(`{"onlyIPType":"invalid"}`)) }, "onlyIPType"},
		{"test invalid config action", func() { NewTestFromBytes("", nil) }, "action"},
	}

	if name := os.Getenv("GEOIP_SPECIAL_FATAL_CASE"); name != "" {
		for _, tt := range tests {
			if tt.name == name {
				tt.run()
				return
			}
		}
		t.Fatalf("unknown fatal case %q", name)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestConstructorValidation$")
			cmd.Env = append(os.Environ(), "GEOIP_SPECIAL_FATAL_CASE="+tt.name)
			output, err := cmd.CombinedOutput()
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
				t.Fatalf("constructor did not exit with log.Fatal: %v\n%s", err, output)
			}
			if !strings.Contains(string(output), tt.message) || !strings.Contains(string(output), "[type ") {
				t.Fatalf("missing validation diagnostic %q: %s", tt.message, output)
			}
		})
	}
}

func TestFromBytesRejectsMalformedJSON(t *testing.T) {
	inputs := []func(lib.Action, []byte) (lib.InputConverter, error){
		NewStdinFromBytes, NewCutterFromBytes, NewPrivateFromBytes, NewTestFromBytes,
	}
	outputs := []func(lib.Action, []byte) (lib.OutputConverter, error){
		NewStdoutFromBytes, NewLookupFromBytes,
	}
	for _, data := range [][]byte{[]byte("{"), []byte("[]"), []byte("true")} {
		for _, fromBytes := range inputs {
			if converter, err := fromBytes(lib.ActionAdd, data); err == nil || converter != nil {
				t.Fatalf("malformed input config %q returned %#v, %v", data, converter, err)
			}
		}
		for _, fromBytes := range outputs {
			if converter, err := fromBytes(lib.ActionOutput, data); err == nil || converter != nil {
				t.Fatalf("malformed output config %q returned %#v, %v", data, converter, err)
			}
		}
	}
}

func TestPrivateAndTestInput(t *testing.T) {
	container := lib.NewContainer()
	if _, err := NewPrivate(lib.ActionAdd, nil).Input(container); err != nil {
		t.Fatal(err)
	}
	for _, address := range []string{"127.0.0.1", "::1"} {
		if _, found, err := container.Lookup(address, entryNamePrivate); err != nil || !found {
			t.Fatalf("default private input missing %s: %v", address, err)
		}
	}
	if _, err := NewPrivate(lib.ActionRemove, WithInputOnlyIPType(lib.IPv4)).Input(container); err != nil {
		t.Fatal(err)
	}
	if _, found, err := container.Lookup("127.0.0.1", entryNamePrivate); err != nil || found {
		t.Fatalf("private IPv4 removal failed: found = %v, err = %v", found, err)
	}
	if _, found, err := container.Lookup("::1", entryNamePrivate); err != nil || !found {
		t.Fatalf("private IPv4 removal affected IPv6: %v", err)
	}

	if _, err := NewTest(lib.ActionAdd, nil).Input(container); err != nil {
		t.Fatal(err)
	}
	entry, found := container.GetEntry(entryNameTest)
	if !found {
		t.Fatal("test entry not created")
	}
	prefixes, err := entry.MarshalText()
	if err != nil || !reflect.DeepEqual(prefixes, testCIDRs) {
		t.Fatalf("test input prefixes = %v, %v; want %v", prefixes, err, testCIDRs)
	}
	if _, err := NewTest(lib.ActionRemove).Input(container); err != nil {
		t.Fatal(err)
	}
	if _, found, err := container.Lookup("127.0.0.1", entryNameTest); err != nil || found {
		t.Fatalf("test removal failed: found = %v, err = %v", found, err)
	}
}

func TestStdinAndCutterInput(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	original := os.Stdin
	os.Stdin = reader
	defer func() { os.Stdin = original }()

	if _, err := io.WriteString(writer, "\n# comment\ninvalid\n192.0.2.1 # host\n2001:db8::1 // host\n"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	container := lib.NewContainer()
	if _, err := NewStdin(lib.ActionAdd, WithName(" example ")).Input(container); err != nil {
		t.Fatal(err)
	}
	if _, err := NewTest(lib.ActionAdd).Input(container); err != nil {
		t.Fatal(err)
	}
	if _, err := NewCutter(lib.ActionRemove, WithInputWantedList([]string{" example "}), WithInputOnlyIPType(lib.IPv4)).Input(container); err != nil {
		t.Fatal(err)
	}
	for address, want := range map[string]bool{"192.0.2.1": false, "2001:db8::1": true} {
		if _, found, err := container.Lookup(address, "example"); err != nil || found != want {
			t.Fatalf("lookup %s: found = %v, err = %v; want %v", address, found, err, want)
		}
	}
	if _, err := NewCutter(lib.ActionRemove, WithInputWantedList([]string{"example"})).Input(container); err != nil {
		t.Fatal(err)
	}
	if _, found := container.GetEntry("example"); found || container.Len() != 1 {
		t.Fatalf("cutter removed incorrect entries: remaining = %d", container.Len())
	}
}

func TestRegisteredConfigAndOutputBehavior(t *testing.T) {
	instance, err := lib.NewInstance()
	if err != nil {
		t.Fatal(err)
	}
	err = instance.InitConfigFromBytes([]byte(`{
		"input":[
			{"type":"private","action":"add"},
			{"type":"test","action":"add"}
		],
		"output":[
			{"type":"stdout","args":{"wantedList":[" test ","private"],"excludedList":["PRIVATE"],"onlyIPType":"ipv4"}},
			{"type":"lookup","args":{"search":" 127.0.0.1 "}},
			{"type":"lookup","args":{"search":"127.0.0.0/8","searchList":[" test "]}},
			{"type":"lookup","args":{"search":"8.8.8.8"}}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	output := captureOutput(t, instance.Run)
	if want := "127.0.0.0/8\nprivate,test\ntest\nfalse\n"; output != want {
		t.Fatalf("output = %q, want %q", output, want)
	}

	container := lib.NewContainer()
	if _, err := NewTest(lib.ActionAdd).Input(container); err != nil {
		t.Fatal(err)
	}
	output = captureOutput(t, func() error {
		return NewStdout(lib.ActionOutput, WithOutputWantedList([]string{"test"}), WithOutputExcludedList([]string{"test"})).Output(container)
	})
	if output != "" {
		t.Fatalf("excluded stdout list produced output %q", output)
	}
	if err := NewLookup(lib.ActionOutput, WithSearch("invalid")).Output(container); err == nil {
		t.Fatal("lookup accepted an invalid IP")
	}
}

func captureOutput(t *testing.T, run func() error) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	original := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = original }()
	if err := run(); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}
