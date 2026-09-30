package mihomo

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestMRSInOptions(t *testing.T) {
	want := &MRSIn{
		Type: TypeMRSIn, Action: lib.ActionAdd, Description: DescMRSIn,
		Name: "cn", URI: "cn.mrs", Want: map[string]bool{"CN": true, "US": true}, OnlyIPType: lib.IPv4,
	}
	got := NewMRSIn(lib.ActionAdd, nil,
		WithNameAndURI("old", "old.mrs"), WithNameAndURI(" cn ", " cn.mrs "),
		WithInputWantedList([]string{" cn ", "", "CN", "us"}), WithInputOnlyIPType(lib.IPv4),
	)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("constructor = %#v, want %#v", got, want)
	}
	fromJSON, err := NewMRSInFromBytes(lib.ActionAdd, []byte(`{
		"name":" cn ","uri":" cn.mrs ","wantedList":[" cn ","","CN","us"],"onlyIPType":"ipv4"
	}`))
	if err != nil || !reflect.DeepEqual(fromJSON, got) {
		t.Fatalf("JSON adapter = %#v, %v; want %#v", fromJSON, err, got)
	}

	for _, action := range []lib.Action{lib.ActionAdd, lib.ActionRemove} {
		for _, ipType := range []lib.IPType{"", lib.IPv4, lib.IPv6} {
			got := NewMRSIn(action, nil, WithInputDir(" rules "), WithInputOnlyIPType(ipType)).(*MRSIn)
			if got.InputDir != "rules" || got.Name != "" || got.URI != "" || len(got.Want) != 0 ||
				got.OnlyIPType != ipType || got.GetType() != TypeMRSIn || got.GetAction() != action || got.GetDescription() != DescMRSIn {
				t.Fatalf("directory input = %#v", got)
			}
		}
	}
	fromJSON, err = NewMRSInFromBytes(lib.ActionRemove, []byte(`{"inputDir":" rules "}`))
	wantDir := NewMRSIn(lib.ActionRemove, WithInputDir("rules"))
	if err != nil || !reflect.DeepEqual(fromJSON, wantDir) {
		t.Fatalf("directory JSON adapter = %#v, %v; want %#v", fromJSON, err, wantDir)
	}
}

func TestMRSOutOptionsAndDefaults(t *testing.T) {
	want := &MRSOut{
		Type: TypeMRSOut, Action: lib.ActionOutput, Description: DescMRSOut,
		OutputDir: filepath.Join("output", "mrs"),
	}
	for _, opts := range [][]lib.OutputOption{nil, {nil}, {WithOutputDir("old"), WithOutputDir(" \t ")}} {
		got := NewMRSOut(lib.ActionOutput, opts...)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("default output = %#v, want %#v", got, want)
		}
	}
	for _, data := range [][]byte{nil, {}, []byte(`{}`), []byte(`null`), []byte(`{"outputDir":" "}`)} {
		got, err := NewMRSOutFromBytes(lib.ActionOutput, data)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("default JSON output = %#v, %v; want %#v", got, err, want)
		}
	}

	got := NewMRSOut(lib.ActionOutput, nil, WithOutputDir(" custom "),
		WithOutputWantedList([]string{" cn ", "us"}), WithOutputExcludedList([]string{"us"}),
		WithOutputOnlyIPType(lib.IPv6),
	).(*MRSOut)
	want = &MRSOut{
		Type: TypeMRSOut, Action: lib.ActionOutput, Description: DescMRSOut,
		OutputDir: "custom", Want: []string{" cn ", "us"}, Exclude: []string{"us"}, OnlyIPType: lib.IPv6,
	}
	if !reflect.DeepEqual(got, want) || got.GetType() != TypeMRSOut || got.GetAction() != lib.ActionOutput || got.GetDescription() != DescMRSOut {
		t.Fatalf("output = %#v, want %#v", got, want)
	}
	fromJSON, err := NewMRSOutFromBytes(lib.ActionOutput, []byte(`{
		"outputDir":" custom ","wantedList":[" cn ","us"],"excludedList":["us"],"onlyIPType":"ipv6"
	}`))
	if err != nil || !reflect.DeepEqual(fromJSON, got) {
		t.Fatalf("JSON adapter = %#v, %v; want %#v", fromJSON, err, got)
	}
}

func TestMRSJSONErrorsAndRegistration(t *testing.T) {
	for _, data := range []string{`{`, `[]`, `{"onlyIPType":123}`, `{"wantedList":"cn"}`} {
		if got, err := NewMRSInFromBytes(lib.ActionAdd, []byte(data)); got != nil || err == nil {
			t.Errorf("input accepted malformed JSON %q: %#v, %v", data, got, err)
		}
		if got, err := NewMRSOutFromBytes(lib.ActionOutput, []byte(data)); got != nil || err == nil {
			t.Errorf("output accepted malformed JSON %q: %#v, %v", data, got, err)
		}
	}
	instance, err := lib.NewInstance()
	if err != nil {
		t.Fatal(err)
	}
	if err := instance.InitConfigFromBytes([]byte(`{
		"input":[{"type":"mihomoMRS","action":"add","args":{"name":"cn","uri":"cn.mrs"}}],
		"output":[{"type":"mihomoMRS","action":"output"}]
	}`)); err != nil {
		t.Fatalf("registered JSON adapters: %v", err)
	}
}

func TestMRSInvalidOptions(t *testing.T) {
	cases := []struct {
		name   string
		create func()
		want   string
	}{
		{"missing-source", func() { NewMRSIn(lib.ActionAdd) }, "missing inputDir"},
		{"nil-option", func() { NewMRSIn(lib.ActionAdd, nil) }, "missing inputDir"},
		{"empty-source", func() { NewMRSIn(lib.ActionAdd, WithNameAndURI(" ", "\t"), WithInputDir(" ")) }, "missing inputDir"},
		{"name-only", func() { NewMRSIn(lib.ActionAdd, WithNameAndURI("cn", "")) }, "specified together"},
		{"uri-only", func() { NewMRSIn(lib.ActionAdd, WithNameAndURI("", "cn.mrs")) }, "specified together"},
		{"mixed-source", func() { NewMRSIn(lib.ActionAdd, WithNameAndURI("cn", "cn.mrs"), WithInputDir("rules")) }, "cannot be used"},
		{"directory-name-only", func() { NewMRSIn(lib.ActionAdd, WithNameAndURI("cn", ""), WithInputDir("rules")) }, "specified together"},
		{"directory-uri-only", func() { NewMRSIn(lib.ActionAdd, WithNameAndURI("", "cn.mrs"), WithInputDir("rules")) }, "specified together"},
		{"empty-input-action", func() { NewMRSIn("", WithInputDir("rules")) }, "invalid input action"},
		{"output-input-action", func() { NewMRSIn(lib.ActionOutput, WithInputDir("rules")) }, "invalid input action"},
		{"unknown-input-action", func() { NewMRSIn("invalid", WithInputDir("rules")) }, "invalid input action"},
		{"input-ip-type", func() { NewMRSIn(lib.ActionAdd, WithInputDir("rules"), WithInputOnlyIPType("IPv4")) }, "invalid onlyIPType"},
		{"empty-output-action", func() { NewMRSOut("") }, "invalid output action"},
		{"add-output-action", func() { NewMRSOut(lib.ActionAdd) }, "invalid output action"},
		{"remove-output-action", func() { NewMRSOut(lib.ActionRemove) }, "invalid output action"},
		{"unknown-output-action", func() { NewMRSOut("invalid") }, "invalid output action"},
		{"output-ip-type", func() { NewMRSOut(lib.ActionOutput, WithOutputOnlyIPType("ipv5")) }, "invalid onlyIPType"},
		{"json-missing-source", func() { _, _ = NewMRSInFromBytes(lib.ActionAdd, nil) }, "missing inputDir"},
		{"json-empty-source", func() { _, _ = NewMRSInFromBytes(lib.ActionAdd, []byte(`{}`)) }, "missing inputDir"},
		{"json-null-source", func() { _, _ = NewMRSInFromBytes(lib.ActionAdd, []byte(`null`)) }, "missing inputDir"},
		{"json-half-pair", func() { _, _ = NewMRSInFromBytes(lib.ActionAdd, []byte(`{"name":"cn","uri":" "}`)) }, "specified together"},
		{"json-mixed-source", func() {
			_, _ = NewMRSInFromBytes(lib.ActionAdd, []byte(`{"name":"cn","uri":"cn.mrs","inputDir":"rules"}`))
		}, "cannot be used"},
		{"json-input-ip-type", func() { _, _ = NewMRSInFromBytes(lib.ActionAdd, []byte(`{"inputDir":"rules","onlyIPType":"bad"}`)) }, "invalid onlyIPType"},
		{"json-output-ip-type", func() { _, _ = NewMRSOutFromBytes(lib.ActionOutput, []byte(`{"onlyIPType":"bad"}`)) }, "invalid onlyIPType"},
	}
	if name := os.Getenv("GEOIP_MRS_FATAL_CASE"); name != "" {
		for _, tc := range cases {
			if tc.name == name {
				tc.create()
				return
			}
		}
		t.Fatalf("unknown child case %q", name)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestMRSInvalidOptions$")
			cmd.Env = append(os.Environ(), "GEOIP_MRS_FATAL_CASE="+tc.name)
			output, err := cmd.CombinedOutput()
			exitErr, ok := err.(*exec.ExitError)
			if !ok || exitErr.ExitCode() != 1 || !strings.Contains(string(output), tc.want) ||
				!strings.Contains(string(output), "[type mihomoMRS | action ") {
				t.Fatalf("expected log.Fatal containing %q; got %v, %s", tc.want, err, output)
			}
		})
	}
}

func TestMRSRoundTrip(t *testing.T) {
	dir := fmt.Sprintf(".mrs-roundtrip-%d", os.Getpid())
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	})
	source := lib.NewContainer()
	for _, name := range []string{"cn", "us", "jp"} {
		entry := lib.NewEntry(name)
		for _, prefix := range []string{"192.0.2.0/24", "2001:db8::/32"} {
			if err := entry.AddPrefix(prefix); err != nil {
				t.Fatal(err)
			}
		}
		if err := source.Add(entry); err != nil {
			t.Fatal(err)
		}
	}

	for _, ipType := range []lib.IPType{"", lib.IPv4, lib.IPv6} {
		name := string(ipType)
		if name == "" {
			name = "both"
		}
		t.Run(name, func(t *testing.T) {
			outputDir := filepath.Join(dir, name)
			out := NewMRSOut(lib.ActionOutput, WithOutputDir(outputDir),
				WithOutputWantedList([]string{" cn ", "us", "jp"}), WithOutputExcludedList([]string{" JP "}),
				WithOutputOnlyIPType(ipType),
			)
			if err := out.Output(source); err != nil {
				t.Fatal(err)
			}
			files, err := os.ReadDir(outputDir)
			if err != nil {
				t.Fatal(err)
			}
			if len(files) != 2 || files[0].Name() != "cn.mrs" || files[1].Name() != "us.mrs" {
				t.Fatalf("unexpected output files: %v", files)
			}

			want := []string{"192.0.2.0/24", "2001:db8::/32"}
			switch ipType {
			case lib.IPv4:
				want = want[:1]
			case lib.IPv6:
				want = want[1:]
			}
			in := NewMRSIn(lib.ActionAdd, WithInputDir(outputDir), WithInputWantedList([]string{" cn "}))
			result, err := in.Input(lib.NewContainer())
			if err != nil {
				t.Fatal(err)
			}
			assertMRSPrefixes(t, result, "CN", want)
			if result.Len() != 1 {
				t.Fatalf("wantedList produced %d entries", result.Len())
			}

			fileIn := NewMRSIn(lib.ActionAdd, WithNameAndURI("renamed", filepath.Join(outputDir, "cn.mrs")))
			result, err = fileIn.Input(lib.NewContainer())
			if err != nil {
				t.Fatal(err)
			}
			assertMRSPrefixes(t, result, "RENAMED", want)
		})
	}

	file := filepath.Join(dir, "both", "cn.mrs")
	input := NewMRSIn(lib.ActionAdd, WithNameAndURI("cn", file), WithInputOnlyIPType(lib.IPv6))
	ipv6Only, err := input.Input(lib.NewContainer())
	if err != nil {
		t.Fatal(err)
	}
	assertMRSPrefixes(t, ipv6Only, "CN", []string{"2001:db8::/32"})

	config, err := json.Marshal(map[string]any{
		"input": []any{map[string]any{
			"type": TypeMRSIn, "action": lib.ActionRemove,
			"args": map[string]any{"name": "cn", "uri": file, "onlyIPType": lib.IPv6},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	instance, err := lib.NewInstance()
	if err != nil {
		t.Fatal(err)
	}
	if err := instance.InitConfigFromBytes(config); err != nil {
		t.Fatal(err)
	}
	if err := instance.RunInput(source); err != nil {
		t.Fatal(err)
	}
	assertMRSPrefixes(t, source, "CN", []string{"192.0.2.0/24"})
}

func assertMRSPrefixes(t *testing.T, container lib.Container, name string, want []string) {
	t.Helper()
	entry, found := container.GetEntry(name)
	if !found {
		t.Fatalf("entry %q not found", name)
	}
	got, err := entry.MarshalText()
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("prefixes = %v, %v; want %v", got, err, want)
	}
}
