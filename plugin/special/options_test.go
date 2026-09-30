package special

import (
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestInputOptions(t *testing.T) {
	tests := []struct {
		name   string
		action lib.Action
		data   string
		parse  func(lib.Action, []byte) (lib.InputConverter, error)
		direct lib.InputConverter
	}{
		{"stdin", lib.ActionAdd, `{"name":" test ","onlyIPType":"ipv4"}`, NewStdinFromBytes,
			NewStdin(lib.ActionAdd, nil, WithName(" test "), WithInputOnlyIPType(lib.IPv4))},
		{"private", lib.ActionRemove, `{"onlyIPType":"ipv6"}`, NewPrivateFromBytes,
			NewPrivate(lib.ActionRemove, nil, WithInputOnlyIPType(lib.IPv6))},
		{"private defaults", lib.ActionAdd, "", NewPrivateFromBytes, NewPrivate(lib.ActionAdd)},
		{"cutter", lib.ActionRemove, `{"wantedList":[" cn ","","CN"],"onlyIPType":"ipv4"}`, NewCutterFromBytes,
			NewCutter(lib.ActionRemove, nil, WithInputWantedList([]string{" cn ", "", "CN"}), WithInputOnlyIPType(lib.IPv4))},
		{"test", lib.ActionAdd, "", NewTestFromBytes, NewTest(lib.ActionAdd, nil)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.parse(tt.action, []byte(tt.data))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.direct) {
				t.Fatalf("JSON = %#v, options = %#v", got, tt.direct)
			}
			if _, err := tt.parse(tt.action, []byte("{")); err == nil {
				t.Fatal("malformed JSON was accepted")
			}
		})
	}
	c := NewCutter(lib.ActionRemove, WithInputWantedList([]string{" cn ", "", "CN"})).(*Cutter)
	if !reflect.DeepEqual(c.Want, map[string]bool{"CN": true}) {
		t.Fatalf("wanted list was not normalized: %v", c.Want)
	}
}

func TestOutputOptions(t *testing.T) {
	tests := []struct {
		name   string
		data   string
		parse  func(lib.Action, []byte) (lib.OutputConverter, error)
		direct lib.OutputConverter
	}{
		{"stdout defaults", "", NewStdoutFromBytes, NewStdout(lib.ActionOutput, nil)},
		{"stdout", `{"wantedList":["cn"],"excludedList":["private"],"onlyIPType":"ipv4"}`, NewStdoutFromBytes,
			NewStdout(lib.ActionOutput, WithOutputWantedList([]string{"cn"}), WithOutputExcludedList([]string{"private"}), WithOutputOnlyIPType(lib.IPv4))},
		{"lookup", `{"search":" 192.0.2.1 ","searchList":["cn"]}`, NewLookupFromBytes,
			NewLookup(lib.ActionOutput, nil, WithSearch(" 192.0.2.1 "), WithSearchList([]string{"cn"}))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.parse(lib.ActionOutput, []byte(tt.data))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.direct) {
				t.Fatalf("JSON = %#v, options = %#v", got, tt.direct)
			}
			if _, err := tt.parse(lib.ActionOutput, []byte("{")); err == nil {
				t.Fatal("malformed JSON was accepted")
			}
		})
	}
}

func TestInvalidOptions(t *testing.T) {
	tests := []struct {
		name string
		want string
		run  func()
	}{
		{"stdin missing name", "missing name", func() { NewStdin(lib.ActionAdd) }},
		{"stdin blank name", "missing name", func() { NewStdin(lib.ActionAdd, WithName(" \t")) }},
		{"stdin invalid action", "invalid input action", func() { NewStdin(lib.ActionOutput, WithName("test")) }},
		{"private invalid action", "invalid input action", func() { NewPrivate("") }},
		{"private invalid ip", "invalid onlyIPType", func() { NewPrivate(lib.ActionAdd, WithInputOnlyIPType("ip")) }},
		{"cutter invalid action", "only supports", func() { NewCutter(lib.ActionAdd, WithInputWantedList([]string{"cn"})) }},
		{"cutter missing list", "wantedList", func() { NewCutter(lib.ActionRemove) }},
		{"cutter blank list", "wantedList", func() { NewCutter(lib.ActionRemove, WithInputWantedList([]string{" "})) }},
		{"lookup missing search", "search target", func() { NewLookup(lib.ActionOutput) }},
		{"lookup invalid ip", "invalid IP or CIDR", func() { NewLookup(lib.ActionOutput, WithSearch("not-an-ip")) }},
		{"lookup invalid cidr", "invalid IP or CIDR", func() { NewLookup(lib.ActionOutput, WithSearch("192.0.2.0/99")) }},
		{"lookup invalid action", "invalid output action", func() { NewLookup(lib.ActionAdd, WithSearch("::1")) }},
		{"stdout invalid action", "invalid output action", func() { NewStdout(lib.ActionAdd) }},
		{"stdout invalid ip", "invalid onlyIPType", func() { NewStdout(lib.ActionOutput, WithOutputOnlyIPType("ip")) }},
		{"test invalid action", "invalid input action", func() { NewTest(lib.ActionOutput) }},
		{"JSON validation", "missing name", func() { NewStdinFromBytes(lib.ActionAdd, []byte(`{"name":" "}`)) }},
	}
	if name := os.Getenv("GEOIP_SPECIAL_INVALID_CASE"); name != "" {
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
			cmd := exec.Command(os.Args[0], "-test.run=^TestInvalidOptions$")
			cmd.Env = append(os.Environ(), "GEOIP_SPECIAL_INVALID_CASE="+tt.name)
			output, err := cmd.CombinedOutput()
			if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 || !strings.Contains(string(output), tt.want) {
				t.Fatalf("expected constructor rejection %q, got %v: %s", tt.want, err, output)
			}
		})
	}
}

func TestPrivateAndCutterConversion(t *testing.T) {
	container := lib.NewContainer()
	if _, err := NewPrivate(lib.ActionAdd, WithInputOnlyIPType(lib.IPv4)).Input(container); err != nil {
		t.Fatal(err)
	}
	if _, found, err := container.Lookup("192.168.1.1"); err != nil || !found {
		t.Fatalf("private IPv4 lookup: found=%v, err=%v", found, err)
	}
	if _, found, err := container.Lookup("fc00::1"); err != nil || found {
		t.Fatalf("filtered IPv6 lookup: found=%v, err=%v", found, err)
	}
	if _, err := NewCutter(lib.ActionRemove, WithInputWantedList([]string{"private"})).Input(container); err != nil {
		t.Fatal(err)
	}
	if container.Len() != 0 {
		t.Fatal("cutter did not remove private entry")
	}
}
