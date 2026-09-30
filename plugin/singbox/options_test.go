package singbox

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestSRSOptions(t *testing.T) {
	input := NewSRSIn(lib.ActionAdd, nil,
		WithNameAndURI(" cn ", " cn.srs "),
		WithInputWantedList([]string{" cn ", "", "CN"}),
		WithInputOnlyIPType(" IPv4 "),
	).(*srs_in)
	if input.Name != "cn" || input.URI != "cn.srs" || input.OnlyIPType != lib.IPv4 ||
		!reflect.DeepEqual(input.Want, map[string]bool{"CN": true}) {
		t.Fatalf("unexpected input options: %+v", input)
	}
	fromJSON, err := NewSRSInFromBytes(lib.ActionAdd, []byte(`{
		"name":" cn ", "uri":" cn.srs ", "wantedList":[" cn ","","CN"], "onlyIPType":" IPv4 "
	}`))
	if err != nil || !reflect.DeepEqual(input, fromJSON) {
		t.Fatalf("JSON input mismatch: %+v, %v", fromJSON, err)
	}
	dir := NewSRSIn(lib.ActionRemove, WithInputDir(" rules ")).(*srs_in)
	if dir.InputDir != "rules" || dir.Name != "" || dir.URI != "" {
		t.Fatalf("unexpected directory input: %+v", dir)
	}

	for _, opts := range [][]lib.OutputOption{nil, {nil}, {WithOutputDir(" ")}} {
		output := NewSRSOut(lib.ActionOutput, opts...).(*srs_out)
		if output.OutputDir != defaultOutputDir {
			t.Fatalf("missing default output directory: %+v", output)
		}
	}
	output := NewSRSOut(lib.ActionOutput,
		WithOutputDir(" rules "),
		WithOutputWantedList([]string{"cn", "us"}),
		WithOutputExcludedList([]string{"us"}),
		WithOutputOnlyIPType(" IPv6 "),
	).(*srs_out)
	fromJSONOut, err := NewSRSOutFromBytes(lib.ActionOutput, []byte(`{
		"outputDir":" rules ", "wantedList":["cn","us"], "excludedList":["us"], "onlyIPType":" IPv6 "
	}`))
	if err != nil || !reflect.DeepEqual(output, fromJSONOut) {
		t.Fatalf("JSON output mismatch: %+v, %v", fromJSONOut, err)
	}
	for _, data := range [][]byte{nil, []byte(`{}`), []byte(`null`)} {
		fromJSONOut, err := NewSRSOutFromBytes(lib.ActionOutput, data)
		if err != nil || !reflect.DeepEqual(NewSRSOut(lib.ActionOutput), fromJSONOut) {
			t.Fatalf("default JSON output mismatch: %+v, %v", fromJSONOut, err)
		}
	}
	if _, err := NewSRSInFromBytes(lib.ActionAdd, []byte(`{`)); err == nil {
		t.Fatal("expected malformed input JSON error")
	}
	if _, err := NewSRSOutFromBytes(lib.ActionOutput, []byte(`{`)); err == nil {
		t.Fatal("expected malformed output JSON error")
	}
}

func TestSRSInvalidOptions(t *testing.T) {
	tests := []struct {
		name    string
		message string
		run     func()
	}{
		{"missing", "missing name or uri or inputDir", func() { NewSRSIn(lib.ActionAdd) }},
		{"blank", "missing name or uri or inputDir", func() {
			NewSRSIn(lib.ActionAdd, WithNameAndURI(" ", " "), WithInputDir(" "))
		}},
		{"name", "name and uri must be specified together", func() {
			NewSRSIn(lib.ActionAdd, WithNameAndURI("cn", ""))
		}},
		{"uri", "name and uri must be specified together", func() {
			NewSRSIn(lib.ActionAdd, WithNameAndURI("", "cn.srs"))
		}},
		{"name-dir", "name and uri must be specified together", func() {
			NewSRSIn(lib.ActionAdd, WithNameAndURI("cn", ""), WithInputDir("rules"))
		}},
		{"file-dir", "inputDir is not allowed", func() {
			NewSRSIn(lib.ActionAdd, WithNameAndURI("cn", "cn.srs"), WithInputDir("rules"))
		}},
		{"input-action", "only supports add or remove", func() {
			NewSRSIn(lib.ActionOutput, WithInputDir("rules"))
		}},
		{"input-ip", "invalid onlyIPType", func() {
			NewSRSIn(lib.ActionAdd, WithInputDir("rules"), WithInputOnlyIPType("ipv5"))
		}},
		{"output-action", "only supports output", func() { NewSRSOut(lib.ActionAdd) }},
		{"output-ip", "invalid onlyIPType", func() {
			NewSRSOut(lib.ActionOutput, WithOutputOnlyIPType("ipv5"))
		}},
		{"json-missing", "missing name or uri or inputDir", func() {
			NewSRSInFromBytes(lib.ActionAdd, []byte(`{}`))
		}},
		{"json-pair", "name and uri must be specified together", func() {
			NewSRSInFromBytes(lib.ActionAdd, []byte(`{"name":"cn","inputDir":"rules"}`))
		}},
	}
	if name := os.Getenv("GEOIP_SRS_INVALID_OPTION"); name != "" {
		for _, tt := range tests {
			if tt.name == name {
				tt.run()
				return
			}
		}
		t.Fatalf("unknown case %q", name)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestSRSInvalidOptions$")
			cmd.Env = append(os.Environ(), "GEOIP_SRS_INVALID_OPTION="+tt.name)
			output, err := cmd.CombinedOutput()
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 1 || !strings.Contains(string(output), tt.message) {
				t.Fatalf("expected exit 1 with %q, got %v: %s", tt.message, err, output)
			}
		})
	}
}

func TestSRSRoundTrip(t *testing.T) {
	container := lib.NewContainer()
	entry := lib.NewEntry("CN")
	for _, prefix := range []string{"192.0.2.0/24", "2001:db8::/32"} {
		if err := entry.AddPrefix(prefix); err != nil {
			t.Fatal(err)
		}
	}
	if err := container.Add(entry); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := NewSRSOut(lib.ActionOutput, WithOutputDir(dir)).Output(container); err != nil {
		t.Fatal(err)
	}
	for _, opts := range [][]lib.InputOption{
		{WithNameAndURI("CN", filepath.Join(dir, "cn.srs"))},
		{WithInputDir(dir), WithInputWantedList([]string{" cn "})},
	} {
		result, err := NewSRSIn(lib.ActionAdd, opts...).Input(lib.NewContainer())
		if err != nil {
			t.Fatal(err)
		}
		for _, ip := range []string{"192.0.2.1", "2001:db8::1"} {
			_, found, err := result.Lookup(ip, "CN")
			if err != nil || !found {
				t.Fatalf("round-trip lost %s: %v", ip, err)
			}
		}
	}
}
