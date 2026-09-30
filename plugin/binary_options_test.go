package plugin_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/Loyalsoldier/geoip/plugin/mihomo"
	"github.com/Loyalsoldier/geoip/plugin/singbox"
	"github.com/Loyalsoldier/geoip/plugin/v2ray"
)

func TestBinaryOptionsFromBytes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		direct lib.InputConverter
		parse  func(lib.Action, []byte) (lib.InputConverter, error)
		data   string
	}{
		{
			"mrs",
			mihomo.NewMRSIn(lib.ActionAdd, nil, mihomo.WithNameAndURI(" test ", " test.mrs "),
				mihomo.WithInputWantedList([]string{" cn ", "", "us"}), mihomo.WithInputOnlyIPType(lib.IPv4)),
			mihomo.NewMRSInFromBytes,
			`{"name":" test ","uri":" test.mrs ","wantedList":[" cn ","","us"],"onlyIPType":"ipv4"}`,
		},
		{
			"srs",
			singbox.NewSRSIn(lib.ActionAdd, nil, singbox.WithInputDir(" rules "),
				singbox.WithInputWantedList([]string{" cn ", "", "us"}), singbox.WithInputOnlyIPType(lib.IPv6)),
			singbox.NewSRSInFromBytes,
			`{"inputDir":" rules ","wantedList":[" cn ","","us"],"onlyIPType":"ipv6"}`,
		},
		{
			"dat",
			v2ray.NewGeoIPDatIn(lib.ActionAdd, nil, v2ray.WithURI(" test.dat "),
				v2ray.WithInputWantedList([]string{" cn ", "", "us"}), v2ray.WithInputOnlyIPType(lib.IPv4)),
			v2ray.NewGeoIPDatInFromBytes,
			`{"uri":" test.dat ","wantedList":[" cn ","","us"],"onlyIPType":"ipv4"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.parse(lib.ActionAdd, []byte(tc.data))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.direct) {
				t.Fatalf("JSON and functional options differ: %#v != %#v", got, tc.direct)
			}
			if _, err := tc.parse(lib.ActionAdd, []byte(`{"uri":`)); err == nil {
				t.Fatal("malformed JSON was accepted")
			}
		})
	}

	for _, tc := range []struct {
		name   string
		direct lib.OutputConverter
		parse  func(lib.Action, []byte) (lib.OutputConverter, error)
	}{
		{"mrs", mihomo.NewMRSOut(lib.ActionOutput, nil, mihomo.WithOutputDir(" custom "),
			mihomo.WithOutputWantedList([]string{"test"}), mihomo.WithOutputExcludedList([]string{"other"}),
			mihomo.WithOutputOnlyIPType(lib.IPv6)), mihomo.NewMRSOutFromBytes},
		{"srs", singbox.NewSRSOut(lib.ActionOutput, nil, singbox.WithOutputDir(" custom "),
			singbox.WithOutputWantedList([]string{"test"}), singbox.WithOutputExcludedList([]string{"other"}),
			singbox.WithOutputOnlyIPType(lib.IPv6)), singbox.NewSRSOutFromBytes},
		{"dat", v2ray.NewGeoIPDatOut(lib.ActionOutput, nil, v2ray.WithOutputDir(" custom "),
			v2ray.WithOutputName(" custom.dat "), v2ray.WithOneFilePerList(true),
			v2ray.WithOutputWantedList([]string{"test"}), v2ray.WithOutputExcludedList([]string{"other"}),
			v2ray.WithOutputOnlyIPType(lib.IPv6)), v2ray.NewGeoIPDatOutFromBytes},
	} {
		t.Run(tc.name+"-output", func(t *testing.T) {
			got, err := tc.parse(lib.ActionOutput, []byte(`{"outputDir":" custom ","outputName":" custom.dat ",
				"wantedList":["test"],"excludedList":["other"],"onlyIPType":"ipv6","oneFilePerList":true}`))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.direct) {
				t.Fatalf("JSON and functional options differ: %#v != %#v", got, tc.direct)
			}
			if _, err := tc.parse(lib.ActionOutput, []byte(`{"outputDir":`)); err == nil {
				t.Fatal("malformed JSON was accepted")
			}
		})
	}
}

func TestBinaryDefaultOutputRoundTrip(t *testing.T) {
	want := []string{"192.0.2.0/24", "2001:db8::/32"}
	for _, tc := range []struct {
		name   string
		output func(lib.Action, ...lib.OutputOption) lib.OutputConverter
		input  func(lib.Action, string) lib.InputConverter
		file   string
	}{
		{"mrs", mihomo.NewMRSOut, func(action lib.Action, path string) lib.InputConverter {
			return mihomo.NewMRSIn(action, mihomo.WithNameAndURI("TEST", path))
		}, filepath.Join("output", "mrs", "test.mrs")},
		{"srs", singbox.NewSRSOut, func(action lib.Action, path string) lib.InputConverter {
			return singbox.NewSRSIn(action, singbox.WithNameAndURI("TEST", path))
		}, filepath.Join("output", "srs", "test.srs")},
		{"dat", v2ray.NewGeoIPDatOut, func(action lib.Action, path string) lib.InputConverter {
			return v2ray.NewGeoIPDatIn(action, v2ray.WithURI(path))
		}, filepath.Join("output", "dat", "geoip.dat")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			source := lib.NewContainer()
			entry := lib.NewEntry("TEST")
			for _, prefix := range want {
				if err := entry.AddPrefix(prefix); err != nil {
					t.Fatal(err)
				}
			}
			if err := source.Add(entry); err != nil {
				t.Fatal(err)
			}
			if err := tc.output(lib.ActionOutput, nil).Output(source); err != nil {
				t.Fatal(err)
			}
			result, err := tc.input(lib.ActionAdd, tc.file).Input(lib.NewContainer())
			if err != nil {
				t.Fatal(err)
			}
			got, ok := result.GetEntry("TEST")
			if !ok {
				t.Fatal("round trip lost the entry")
			}
			prefixes, err := got.MarshalText()
			if err != nil || !reflect.DeepEqual(prefixes, want) {
				t.Fatalf("round trip prefixes: %v, error: %v", prefixes, err)
			}
			if _, err := tc.input(lib.ActionRemove, tc.file).Input(result); err != nil {
				t.Fatal(err)
			}
			for _, prefix := range want {
				if _, found, err := result.Lookup(prefix); err != nil || found {
					t.Fatalf("removed prefix %s still found: %v, error: %v", prefix, found, err)
				}
			}
		})
	}
}

func TestBinaryInvalidOptions(t *testing.T) {
	cases := []struct {
		name string
		run  func()
		want string
	}{
		{"mrs-empty", func() { mihomo.NewMRSIn(lib.ActionAdd) }, "missing name"},
		{"srs-empty", func() { singbox.NewSRSIn(lib.ActionAdd) }, "missing name"},
		{"dat-empty", func() { v2ray.NewGeoIPDatIn(lib.ActionAdd) }, "uri must be specified"},
		{"mrs-name", func() {
			mihomo.NewMRSIn(lib.ActionAdd, mihomo.WithNameAndURI("test", ""), mihomo.WithInputDir("rules"))
		}, "must be specified together"},
		{"srs-uri", func() {
			singbox.NewSRSIn(lib.ActionAdd, singbox.WithNameAndURI("", "test.srs"), singbox.WithInputDir("rules"))
		}, "must be specified together"},
		{"mrs-mixed", func() {
			mihomo.NewMRSIn(lib.ActionAdd, mihomo.WithNameAndURI("test", "test.mrs"), mihomo.WithInputDir("rules"))
		}, "inputDir cannot"},
		{"srs-mixed", func() {
			singbox.NewSRSIn(lib.ActionAdd, singbox.WithNameAndURI("test", "test.srs"), singbox.WithInputDir("rules"))
		}, "inputDir cannot"},
		{"dat-blank", func() { v2ray.NewGeoIPDatIn(lib.ActionAdd, v2ray.WithURI(" \t")) }, "uri must"},
		{"mrs-json", func() {
			_, _ = mihomo.NewMRSInFromBytes(lib.ActionAdd, []byte(`{"name":"test"}`))
		}, "must be specified together"},
		{"srs-json", func() {
			_, _ = singbox.NewSRSInFromBytes(lib.ActionAdd, []byte(`{"inputDir":" "}`))
		}, "missing name"},
		{"dat-json", func() {
			_, _ = v2ray.NewGeoIPDatInFromBytes(lib.ActionAdd, []byte(`{}`))
		}, "uri must"},
		{"mrs-action", func() {
			mihomo.NewMRSIn(lib.ActionOutput, mihomo.WithInputDir("rules"))
		}, "invalid input action"},
		{"srs-action", func() { singbox.NewSRSOut(lib.ActionAdd) }, "invalid output action"},
		{"dat-action", func() { v2ray.NewGeoIPDatOut(lib.ActionRemove) }, "invalid output action"},
		{"mrs-ip", func() {
			mihomo.NewMRSOut(lib.ActionOutput, mihomo.WithOutputOnlyIPType("invalid"))
		}, "invalid onlyIPType"},
		{"srs-ip", func() {
			singbox.NewSRSIn(lib.ActionAdd, singbox.WithInputDir("rules"), singbox.WithInputOnlyIPType("invalid"))
		}, "invalid onlyIPType"},
		{"dat-ip", func() {
			v2ray.NewGeoIPDatIn(lib.ActionAdd, v2ray.WithURI("test.dat"), v2ray.WithInputOnlyIPType("invalid"))
		}, "invalid onlyIPType"},
	}
	if name := os.Getenv("GEOIP_BINARY_INVALID_OPTION"); name != "" {
		for _, tc := range cases {
			if tc.name == name {
				tc.run()
				return
			}
		}
		t.Fatalf("unknown case %q", name)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestBinaryInvalidOptions$")
			cmd.Env = append(os.Environ(), "GEOIP_BINARY_INVALID_OPTION="+tc.name)
			output, err := cmd.CombinedOutput()
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 1 || !strings.Contains(string(output), tc.want) {
				t.Fatalf("expected constructor failure containing %q; error: %v, output: %s", tc.want, err, output)
			}
		})
	}
}
