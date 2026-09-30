package v2ray

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

func TestGeoIPDatInOptions(t *testing.T) {
	want := &GeoIPDatIn{
		Type: TypeGeoIPDatIn, Action: lib.ActionAdd, Description: DescGeoIPDatIn,
		URI: "geoip.dat", Want: map[string]bool{"CN": true, "US": true}, OnlyIPType: lib.IPv4,
	}
	got := NewGeoIPDatIn(lib.ActionAdd, nil, WithURI("old.dat"), WithURI(" geoip.dat "),
		WithInputWantedList([]string{" cn ", "", "CN", "us"}), WithInputOnlyIPType(lib.IPv4),
	)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("constructor = %#v, want %#v", got, want)
	}
	fromJSON, err := NewGeoIPDatInFromBytes(lib.ActionAdd, []byte(`{
		"uri":" geoip.dat ","wantedList":[" cn ","","CN","us"],"onlyIPType":"ipv4"
	}`))
	if err != nil || !reflect.DeepEqual(fromJSON, got) {
		t.Fatalf("JSON adapter = %#v, %v; want %#v", fromJSON, err, got)
	}

	for _, action := range []lib.Action{lib.ActionAdd, lib.ActionRemove} {
		for _, ipType := range []lib.IPType{"", lib.IPv4, lib.IPv6} {
			got := NewGeoIPDatIn(action, nil, WithURI("geoip.dat"), WithInputOnlyIPType(ipType)).(*GeoIPDatIn)
			if got.URI != "geoip.dat" || len(got.Want) != 0 || got.OnlyIPType != ipType ||
				got.GetType() != TypeGeoIPDatIn || got.GetAction() != action || got.GetDescription() != DescGeoIPDatIn {
				t.Fatalf("URI-only input = %#v", got)
			}
		}
	}
	fromJSON, err = NewGeoIPDatInFromBytes(lib.ActionRemove, []byte(`{"uri":"geoip.dat"}`))
	wantDefault := NewGeoIPDatIn(lib.ActionRemove, WithURI("geoip.dat"))
	if err != nil || !reflect.DeepEqual(fromJSON, wantDefault) {
		t.Fatalf("URI-only JSON adapter = %#v, %v; want %#v", fromJSON, err, wantDefault)
	}
}

func TestGeoIPDatOutOptionsAndDefaults(t *testing.T) {
	want := &GeoIPDatOut{
		Type: TypeGeoIPDatOut, Action: lib.ActionOutput, Description: DescGeoIPDatOut,
		OutputDir: filepath.Join("output", "dat"), OutputName: "geoip.dat",
	}
	for _, opts := range [][]lib.OutputOption{
		nil, {nil}, {WithOutputDir("old"), WithOutputDir(" "), WithOutputName("old.dat"), WithOutputName("\t")},
	} {
		got := NewGeoIPDatOut(lib.ActionOutput, opts...)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("default output = %#v, want %#v", got, want)
		}
	}
	for _, data := range [][]byte{nil, {}, []byte(`{}`), []byte(`null`), []byte(`{"outputDir":" ","outputName":" "}`)} {
		got, err := NewGeoIPDatOutFromBytes(lib.ActionOutput, data)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("default JSON output = %#v, %v; want %#v", got, err, want)
		}
	}

	got := NewGeoIPDatOut(lib.ActionOutput, nil, WithOutputDir(" custom "), WithOutputName(" custom.dat "),
		WithOutputWantedList([]string{" cn ", "us"}), WithOutputExcludedList([]string{"us"}),
		WithOneFilePerList(true), WithOutputOnlyIPType(lib.IPv6),
	).(*GeoIPDatOut)
	want = &GeoIPDatOut{
		Type: TypeGeoIPDatOut, Action: lib.ActionOutput, Description: DescGeoIPDatOut,
		OutputDir: "custom", OutputName: "custom.dat", Want: []string{" cn ", "us"}, Exclude: []string{"us"},
		OneFilePerList: true, OnlyIPType: lib.IPv6,
	}
	if !reflect.DeepEqual(got, want) || got.GetType() != TypeGeoIPDatOut || got.GetAction() != lib.ActionOutput || got.GetDescription() != DescGeoIPDatOut {
		t.Fatalf("output = %#v, want %#v", got, want)
	}
	fromJSON, err := NewGeoIPDatOutFromBytes(lib.ActionOutput, []byte(`{
		"outputDir":" custom ","outputName":" custom.dat ","wantedList":[" cn ","us"],
		"excludedList":["us"],"oneFilePerList":true,"onlyIPType":"ipv6"
	}`))
	if err != nil || !reflect.DeepEqual(fromJSON, got) {
		t.Fatalf("JSON adapter = %#v, %v; want %#v", fromJSON, err, got)
	}
	if got := NewGeoIPDatOut(lib.ActionOutput, WithOneFilePerList(true), WithOneFilePerList(false)).(*GeoIPDatOut); got.OneFilePerList {
		t.Fatal("later option did not disable oneFilePerList")
	}
}

func TestGeoIPDatJSONErrorsAndRegistration(t *testing.T) {
	for _, data := range []string{`{`, `[]`, `{"onlyIPType":123}`, `{"wantedList":"cn"}`} {
		if got, err := NewGeoIPDatInFromBytes(lib.ActionAdd, []byte(data)); got != nil || err == nil {
			t.Errorf("input accepted malformed JSON %q: %#v, %v", data, got, err)
		}
		if got, err := NewGeoIPDatOutFromBytes(lib.ActionOutput, []byte(data)); got != nil || err == nil {
			t.Errorf("output accepted malformed JSON %q: %#v, %v", data, got, err)
		}
	}
	if got, err := NewGeoIPDatOutFromBytes(lib.ActionOutput, []byte(`{"oneFilePerList":"true"}`)); got != nil || err == nil {
		t.Errorf("output accepted non-boolean oneFilePerList: %#v, %v", got, err)
	}
	instance, err := lib.NewInstance()
	if err != nil {
		t.Fatal(err)
	}
	if err := instance.InitConfigFromBytes([]byte(`{
		"input":[{"type":"v2rayGeoIPDat","action":"add","args":{"uri":"geoip.dat"}}],
		"output":[{"type":"v2rayGeoIPDat","action":"output"}]
	}`)); err != nil {
		t.Fatalf("registered JSON adapters: %v", err)
	}
}

func TestGeoIPDatInvalidOptions(t *testing.T) {
	cases := []struct {
		name   string
		create func()
		want   string
	}{
		{"missing-uri", func() { NewGeoIPDatIn(lib.ActionAdd) }, "uri must be specified"},
		{"nil-option", func() { NewGeoIPDatIn(lib.ActionAdd, nil) }, "uri must be specified"},
		{"empty-uri", func() { NewGeoIPDatIn(lib.ActionAdd, WithURI(" \t ")) }, "uri must be specified"},
		{"empty-input-action", func() { NewGeoIPDatIn("", WithURI("geoip.dat")) }, "invalid input action"},
		{"output-input-action", func() { NewGeoIPDatIn(lib.ActionOutput, WithURI("geoip.dat")) }, "invalid input action"},
		{"unknown-input-action", func() { NewGeoIPDatIn("invalid", WithURI("geoip.dat")) }, "invalid input action"},
		{"input-ip-type", func() { NewGeoIPDatIn(lib.ActionAdd, WithURI("geoip.dat"), WithInputOnlyIPType("IPv4")) }, "invalid onlyIPType"},
		{"empty-output-action", func() { NewGeoIPDatOut("") }, "invalid output action"},
		{"add-output-action", func() { NewGeoIPDatOut(lib.ActionAdd) }, "invalid output action"},
		{"remove-output-action", func() { NewGeoIPDatOut(lib.ActionRemove) }, "invalid output action"},
		{"unknown-output-action", func() { NewGeoIPDatOut("invalid") }, "invalid output action"},
		{"output-ip-type", func() { NewGeoIPDatOut(lib.ActionOutput, WithOutputOnlyIPType("ipv5")) }, "invalid onlyIPType"},
		{"json-missing-uri", func() { _, _ = NewGeoIPDatInFromBytes(lib.ActionAdd, nil) }, "uri must be specified"},
		{"json-empty-uri", func() { _, _ = NewGeoIPDatInFromBytes(lib.ActionAdd, []byte(`{}`)) }, "uri must be specified"},
		{"json-null-uri", func() { _, _ = NewGeoIPDatInFromBytes(lib.ActionAdd, []byte(`null`)) }, "uri must be specified"},
		{"json-blank-uri", func() { _, _ = NewGeoIPDatInFromBytes(lib.ActionAdd, []byte(`{"uri":" "}`)) }, "uri must be specified"},
		{"json-input-ip-type", func() { _, _ = NewGeoIPDatInFromBytes(lib.ActionAdd, []byte(`{"uri":"geoip.dat","onlyIPType":"bad"}`)) }, "invalid onlyIPType"},
		{"json-output-ip-type", func() { _, _ = NewGeoIPDatOutFromBytes(lib.ActionOutput, []byte(`{"onlyIPType":"bad"}`)) }, "invalid onlyIPType"},
	}
	if name := os.Getenv("GEOIP_DAT_FATAL_CASE"); name != "" {
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
			cmd := exec.Command(os.Args[0], "-test.run=^TestGeoIPDatInvalidOptions$")
			cmd.Env = append(os.Environ(), "GEOIP_DAT_FATAL_CASE="+tc.name)
			output, err := cmd.CombinedOutput()
			exitErr, ok := err.(*exec.ExitError)
			if !ok || exitErr.ExitCode() != 1 || !strings.Contains(string(output), tc.want) ||
				!strings.Contains(string(output), "[type v2rayGeoIPDat | action ") {
				t.Fatalf("expected log.Fatal containing %q; got %v, %s", tc.want, err, output)
			}
		})
	}
}

func TestGeoIPDatRoundTrip(t *testing.T) {
	dir := fmt.Sprintf(".dat-roundtrip-%d", os.Getpid())
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

	for _, tc := range []struct {
		name           string
		oneFilePerList bool
		ipType         lib.IPType
		files          []string
		want           []string
	}{
		{"combined", false, "", []string{"custom.dat"}, []string{"192.0.2.0/24", "2001:db8::/32"}},
		{"ipv4", false, lib.IPv4, []string{"custom.dat"}, []string{"192.0.2.0/24"}},
		{"ipv6", false, lib.IPv6, []string{"custom.dat"}, []string{"2001:db8::/32"}},
		{"per-list", true, "", []string{"cn.dat", "us.dat"}, []string{"192.0.2.0/24", "2001:db8::/32"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outputDir := filepath.Join(dir, tc.name)
			out := NewGeoIPDatOut(lib.ActionOutput, WithOutputDir(outputDir), WithOutputName("custom.dat"),
				WithOutputWantedList([]string{" cn ", "us", "jp"}), WithOutputExcludedList([]string{" JP "}),
				WithOneFilePerList(tc.oneFilePerList), WithOutputOnlyIPType(tc.ipType),
			)
			if err := out.Output(source); err != nil {
				t.Fatal(err)
			}
			files, err := os.ReadDir(outputDir)
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, file := range files {
				names = append(names, file.Name())
			}
			if !reflect.DeepEqual(names, tc.files) {
				t.Fatalf("output files = %v, want %v", names, tc.files)
			}
			result := lib.NewContainer()
			for _, name := range names {
				in := NewGeoIPDatIn(lib.ActionAdd, WithURI(filepath.Join(outputDir, name)))
				if _, err := in.Input(result); err != nil {
					t.Fatal(err)
				}
			}
			if result.Len() != 2 {
				t.Fatalf("expected 2 entries, got %d", result.Len())
			}
			assertDatPrefixes(t, result, "CN", tc.want)
			assertDatPrefixes(t, result, "US", tc.want)
		})
	}

	file := filepath.Join(dir, "combined", "custom.dat")
	input := NewGeoIPDatIn(lib.ActionAdd, WithURI(file), WithInputWantedList([]string{" cn "}), WithInputOnlyIPType(lib.IPv6))
	ipv6Only, err := input.Input(lib.NewContainer())
	if err != nil {
		t.Fatal(err)
	}
	assertDatPrefixes(t, ipv6Only, "CN", []string{"2001:db8::/32"})
	if ipv6Only.Len() != 1 {
		t.Fatalf("wantedList produced %d entries", ipv6Only.Len())
	}

	config, err := json.Marshal(map[string]any{
		"input": []any{map[string]any{
			"type": TypeGeoIPDatIn, "action": lib.ActionRemove,
			"args": map[string]any{"uri": file, "wantedList": []string{"cn"}, "onlyIPType": lib.IPv6},
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
	assertDatPrefixes(t, source, "CN", []string{"192.0.2.0/24"})
	assertDatPrefixes(t, source, "US", []string{"192.0.2.0/24", "2001:db8::/32"})
}

func assertDatPrefixes(t *testing.T, container lib.Container, name string, want []string) {
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
