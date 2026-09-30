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
	in := NewSRSIn(lib.ActionAdd, nil,
		WithNameAndURI(" cn ", " cn.srs "),
		WithInputWantedList([]string{" cn ", "", "CN"}),
		WithInputOnlyIPType(lib.IPv4),
	).(*srs_in)
	if in.Name != "cn" || in.URI != "cn.srs" || !reflect.DeepEqual(in.Want, map[string]bool{"CN": true}) {
		t.Fatalf("input options not normalized: %#v", in)
	}
	parsedIn, err := NewSRSInFromBytes(lib.ActionAdd, []byte(`{"name":" cn ","uri":" cn.srs ","wantedList":[" cn ","","CN"],"onlyIPType":"ipv4"}`))
	if err != nil || !reflect.DeepEqual(parsedIn, in) {
		t.Fatalf("JSON input = %#v, %v; options = %#v", parsedIn, err, in)
	}
	dir := NewSRSIn(lib.ActionRemove, WithInputDir(" rules ")).(*srs_in)
	if dir.InputDir != "rules" || dir.Name != "" || dir.URI != "" {
		t.Fatalf("directory input = %#v", dir)
	}
	for _, data := range [][]byte{nil, []byte("{}"), []byte(`{"outputDir":" "}`)} {
		parsed, err := NewSRSOutFromBytes(lib.ActionOutput, data)
		if err != nil || !reflect.DeepEqual(parsed, NewSRSOut(lib.ActionOutput, nil)) {
			t.Fatalf("output defaults disagree: %#v, %v", parsed, err)
		}
		if parsed.(*srs_out).OutputDir != defaultOutputDir {
			t.Fatalf("wrong default output directory: %#v", parsed)
		}
	}
	out := NewSRSOut(lib.ActionOutput,
		WithOutputDir(" rules "),
		WithOutputWantedList([]string{"cn"}),
		WithOutputExcludedList([]string{"private"}),
		WithOutputOnlyIPType(lib.IPv6),
	)
	parsedOut, err := NewSRSOutFromBytes(lib.ActionOutput, []byte(`{"outputDir":" rules ","wantedList":["cn"],"excludedList":["private"],"onlyIPType":"ipv6"}`))
	if err != nil || !reflect.DeepEqual(parsedOut, out) {
		t.Fatalf("JSON output = %#v, %v; options = %#v", parsedOut, err, out)
	}
	if _, err := NewSRSInFromBytes(lib.ActionAdd, []byte("{")); err == nil {
		t.Fatal("malformed input JSON accepted")
	}
	if _, err := NewSRSOutFromBytes(lib.ActionOutput, []byte("{")); err == nil {
		t.Fatal("malformed output JSON accepted")
	}
}

func TestInvalidSRSOptions(t *testing.T) {
	tests := []struct {
		name string
		want string
		run  func()
	}{
		{"missing source", "missing name or uri or inputDir", func() { NewSRSIn(lib.ActionAdd) }},
		{"blank source", "missing name or uri or inputDir", func() { NewSRSIn(lib.ActionAdd, WithNameAndURI(" ", " "), WithInputDir(" ")) }},
		{"name only", "specified together", func() { NewSRSIn(lib.ActionAdd, WithNameAndURI("cn", "")) }},
		{"uri only", "specified together", func() { NewSRSIn(lib.ActionAdd, WithNameAndURI("", "cn.srs")) }},
		{"dir with half pair", "specified together", func() { NewSRSIn(lib.ActionAdd, WithNameAndURI("cn", ""), WithInputDir("rules")) }},
		{"dir with pair", "not allowed", func() { NewSRSIn(lib.ActionAdd, WithNameAndURI("cn", "cn.srs"), WithInputDir("rules")) }},
		{"input action", "invalid input action", func() { NewSRSIn(lib.ActionOutput, WithInputDir("rules")) }},
		{"input ip", "invalid onlyIPType", func() { NewSRSIn(lib.ActionAdd, WithInputDir("rules"), WithInputOnlyIPType("ip")) }},
		{"output action", "invalid output action", func() { NewSRSOut(lib.ActionAdd) }},
		{"output ip", "invalid onlyIPType", func() { NewSRSOut(lib.ActionOutput, WithOutputOnlyIPType("ip")) }},
		{"JSON validation", "specified together", func() { NewSRSInFromBytes(lib.ActionAdd, []byte(`{"name":"cn"}`)) }},
	}
	if name := os.Getenv("GEOIP_SRS_INVALID_CASE"); name != "" {
		for _, tt := range tests {
			if tt.name == name {
				tt.run()
				return
			}
		}
		t.Fatal("unknown case")
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestInvalidSRSOptions$")
			cmd.Env = append(os.Environ(), "GEOIP_SRS_INVALID_CASE="+tt.name)
			output, err := cmd.CombinedOutput()
			if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 || !strings.Contains(string(output), tt.want) {
				t.Fatalf("expected constructor rejection %q, got %v: %s", tt.want, err, output)
			}
		})
	}
}

func TestSRSRoundTrip(t *testing.T) {
	dir := t.TempDir()
	container := lib.NewContainer()
	entry := lib.NewEntry("test")
	for _, prefix := range []string{"192.0.2.0/24", "2001:db8::/32"} {
		if err := entry.AddPrefix(prefix); err != nil {
			t.Fatal(err)
		}
	}
	if err := container.Add(entry); err != nil {
		t.Fatal(err)
	}
	if err := NewSRSOut(lib.ActionOutput, WithOutputDir(dir)).Output(container); err != nil {
		t.Fatal(err)
	}
	for _, input := range []lib.InputConverter{
		NewSRSIn(lib.ActionAdd, WithInputDir(dir)),
		NewSRSIn(lib.ActionAdd, WithNameAndURI("test", filepath.Join(dir, "test.srs"))),
	} {
		got := lib.NewContainer()
		if _, err := input.Input(got); err != nil {
			t.Fatal(err)
		}
		for _, ip := range []string{"192.0.2.1", "2001:db8::1"} {
			if _, found, err := got.Lookup(ip, "test"); err != nil || !found {
				t.Fatalf("roundtrip lookup %s: found=%v, err=%v", ip, found, err)
			}
		}
	}
}
