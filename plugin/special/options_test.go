package special

import (
	"io"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestSpecialInputConstructors(t *testing.T) {
	cases := []struct {
		name   string
		desc   string
		action lib.Action
		new    func(lib.Action, ...lib.InputOption) lib.InputConverter
		parse  func(lib.Action, []byte) (lib.InputConverter, error)
		opts   []lib.InputOption
		args   string
	}{
		{TypeStdin, DescStdin, lib.ActionAdd, NewStdin, NewStdinFromBytes,
			[]lib.InputOption{nil, WithName(" cn "), WithInputOnlyIPType(" IPv6 ")},
			`{"name":" cn ","onlyIPType":" IPv6 "}`},
		{TypePrivate, DescPrivate, lib.ActionRemove, NewPrivate, NewPrivateFromBytes,
			[]lib.InputOption{nil, WithInputOnlyIPType(" IPV4 ")}, `{"onlyIPType":" IPV4 "}`},
		{TypeCutter, DescCutter, lib.ActionRemove, NewCutter, NewCutterFromBytes,
			[]lib.InputOption{nil, WithInputWantedList([]string{" cn ", "", "US", "cn"}), WithInputOnlyIPType(" IPv6 ")},
			`{"wantedList":[" cn ","","US","cn"],"onlyIPType":" IPv6 "}`},
		{typeTest, descTest, lib.ActionAdd, NewTest, NewTestFromBytes, []lib.InputOption{nil}, `{}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.new(tc.action, tc.opts...)
			if got.GetType() != tc.name || got.GetAction() != tc.action || got.GetDescription() != tc.desc {
				t.Fatalf("unexpected metadata: %+v", got)
			}
			fromJSON, err := tc.parse(tc.action, []byte(tc.args))
			if err != nil || !reflect.DeepEqual(got, fromJSON) {
				t.Fatalf("JSON/options mismatch: %+v, %v", fromJSON, err)
			}
			switch got := got.(type) {
			case *stdin:
				if got.Name != "cn" || got.OnlyIPType != lib.IPv6 {
					t.Fatalf("stdin options not normalized: %+v", got)
				}
			case *private:
				if got.OnlyIPType != lib.IPv4 {
					t.Fatalf("private options not normalized: %+v", got)
				}
			case *cutter:
				if got.OnlyIPType != lib.IPv6 || !reflect.DeepEqual(got.Want, map[string]bool{"CN": true, "US": true}) {
					t.Fatalf("cutter options not normalized: %+v", got)
				}
			}
		})
	}
	if got := NewStdin(lib.ActionRemove, nil, WithName("cn")).(*stdin); got.OnlyIPType != "" {
		t.Fatalf("unexpected stdin defaults: %+v", got)
	}
	if got := NewCutter(lib.ActionRemove, nil, WithInputWantedList([]string{"cn"})).(*cutter); got.OnlyIPType != "" {
		t.Fatalf("unexpected cutter defaults: %+v", got)
	}
	for _, data := range [][]byte{nil, {}, []byte(`{}`), []byte(`null`)} {
		got, err := NewPrivateFromBytes(lib.ActionAdd, data)
		if err != nil || !reflect.DeepEqual(got, NewPrivate(lib.ActionAdd, nil)) {
			t.Fatalf("private defaults for %q: %+v, %v", data, got, err)
		}
		got, err = NewTestFromBytes(lib.ActionRemove, data)
		if err != nil || !reflect.DeepEqual(got, NewTest(lib.ActionRemove, nil)) {
			t.Fatalf("test defaults for %q: %+v, %v", data, got, err)
		}
	}
	called := false
	NewTest(lib.ActionAdd, nil, func(lib.InputConverter) { called = true })
	if !called {
		t.Fatal("test converter did not apply options")
	}
}

func TestSpecialOutputConstructors(t *testing.T) {
	out := NewStdout(lib.ActionOutput, nil).(*stdout)
	if out.GetType() != TypeStdout || out.GetAction() != lib.ActionOutput || out.GetDescription() != DescStdout ||
		out.OnlyIPType != "" || len(out.Want) != 0 || len(out.Exclude) != 0 {
		t.Fatalf("unexpected stdout defaults: %+v", out)
	}
	for _, data := range [][]byte{nil, {}, []byte(`{}`), []byte(`null`)} {
		got, err := NewStdoutFromBytes(lib.ActionOutput, data)
		if err != nil || !reflect.DeepEqual(out, got) {
			t.Fatalf("stdout defaults for %q: %+v, %v", data, got, err)
		}
	}
	out = NewStdout(lib.ActionOutput,
		WithOutputWantedList([]string{" us ", "CN", "", "cn"}), WithOutputExcludedList([]string{" Us "}),
		WithOutputOnlyIPType(" IPV4 "),
	).(*stdout)
	outJSON, err := NewStdoutFromBytes(lib.ActionOutput, []byte(`{"wantedList":[" us ","CN","","cn"],"excludedList":[" Us "],"onlyIPType":" IPV4 "}`))
	if err != nil || !reflect.DeepEqual(out, outJSON) || out.OnlyIPType != lib.IPv4 {
		t.Fatalf("stdout JSON/options mismatch: %+v, %v", outJSON, err)
	}
	if got := out.filterAndSortList(lib.NewContainer()); !slices.Equal(got, []string{"CN", "CN"}) {
		t.Fatalf("wanted/excluded list normalization or duplicate order changed: %v", got)
	}

	lookup := NewLookup(lib.ActionOutput, nil, WithSearch(" 192.0.2.0/24 ")).(*lookup)
	if lookup.GetType() != TypeLookup || lookup.GetAction() != lib.ActionOutput || lookup.GetDescription() != DescLookup ||
		lookup.Search != "192.0.2.0/24" || len(lookup.SearchList) != 0 {
		t.Fatalf("unexpected lookup defaults: %+v", lookup)
	}
	fromJSON, err := NewLookupFromBytes(lib.ActionOutput, []byte(`{"search":" 192.0.2.0/24 "}`))
	if err != nil || !reflect.DeepEqual(lookup, fromJSON) {
		t.Fatalf("lookup JSON/options mismatch: %+v, %v", fromJSON, err)
	}
	WithSearchList([]string{" cn ", "US"})(lookup)
	fromJSON, err = NewLookupFromBytes(lib.ActionOutput, []byte(`{"search":" 192.0.2.0/24 ","searchList":[" cn ","US"]}`))
	if err != nil || !reflect.DeepEqual(lookup, fromJSON) {
		t.Fatalf("lookup searchList mismatch: %+v, %v", fromJSON, err)
	}
}

func TestSpecialMalformedJSON(t *testing.T) {
	for _, parse := range []func(lib.Action, []byte) (lib.InputConverter, error){
		NewStdinFromBytes, NewPrivateFromBytes, NewCutterFromBytes, NewTestFromBytes,
	} {
		for _, data := range []string{`{`, `[]`, `"invalid"`} {
			if got, err := parse(lib.ActionAdd, []byte(data)); err == nil || got != nil {
				t.Errorf("input accepted malformed JSON %q", data)
			}
		}
	}
	for _, parse := range []func(lib.Action, []byte) (lib.OutputConverter, error){
		NewStdoutFromBytes, NewLookupFromBytes,
	} {
		for _, data := range []string{`{`, `[]`, `"invalid"`} {
			if got, err := parse(lib.ActionOutput, []byte(data)); err == nil || got != nil {
				t.Errorf("output accepted malformed JSON %q", data)
			}
		}
	}
}

func TestSpecialConstructorValidation(t *testing.T) {
	cases := []struct {
		name string
		want string
		run  func()
	}{
		{"stdin-missing", "missing name", func() { NewStdin(lib.ActionAdd) }},
		{"stdin-blank", "missing name", func() { NewStdin(lib.ActionAdd, WithName(" \t")) }},
		{"stdin-action", "add or remove", func() { NewStdin(lib.ActionOutput, WithName("cn")) }},
		{"stdin-iptype", "invalid onlyIPType", func() { NewStdin(lib.ActionAdd, WithName("cn"), WithInputOnlyIPType("ipv5")) }},
		{"private-action", "add or remove", func() { NewPrivate(lib.ActionOutput) }},
		{"private-iptype", "invalid onlyIPType", func() { NewPrivate(lib.ActionAdd, WithInputOnlyIPType("ipv5")) }},
		{"cutter-action", "remove action", func() { NewCutter(lib.ActionAdd, WithInputWantedList([]string{"cn"})) }},
		{"cutter-output-action", "remove action", func() { NewCutter(lib.ActionOutput, WithInputWantedList([]string{"cn"})) }},
		{"cutter-missing", "wantedList", func() { NewCutter(lib.ActionRemove) }},
		{"cutter-empty", "wantedList", func() { NewCutter(lib.ActionRemove, WithInputWantedList(nil)) }},
		{"cutter-blank", "wantedList", func() { NewCutter(lib.ActionRemove, WithInputWantedList([]string{" ", "\t"})) }},
		{"cutter-iptype", "invalid onlyIPType", func() { NewCutter(lib.ActionRemove, WithInputWantedList([]string{"cn"}), WithInputOnlyIPType("ipv5")) }},
		{"test-action", "add or remove", func() { NewTest(lib.ActionOutput) }},
		{"test-empty-action", "add or remove", func() { NewTest("") }},
		{"stdout-action", "output action", func() { NewStdout(lib.ActionAdd) }},
		{"stdout-iptype", "invalid onlyIPType", func() { NewStdout(lib.ActionOutput, WithOutputOnlyIPType("ipv5")) }},
		{"lookup-action", "output action", func() { NewLookup(lib.ActionAdd, WithSearch("192.0.2.1")) }},
		{"lookup-missing", "search target", func() { NewLookup(lib.ActionOutput) }},
		{"lookup-blank", "search target", func() { NewLookup(lib.ActionOutput, WithSearch(" \t ")) }},
		{"lookup-ip", "invalid IP or CIDR", func() { NewLookup(lib.ActionOutput, WithSearch("999.0.0.1")) }},
		{"lookup-cidr", "invalid IP or CIDR", func() { NewLookup(lib.ActionOutput, WithSearch("192.0.2.1/33")) }},
		{"lookup-ipv6-cidr", "invalid IP or CIDR", func() { NewLookup(lib.ActionOutput, WithSearch("2001:db8::/129")) }},
		{"unsupported-wanted-private", "not supported", func() { NewPrivate(lib.ActionAdd, WithInputWantedList([]string{"cn"})) }},
		{"unsupported-wanted-stdin", "not supported", func() { NewStdin(lib.ActionAdd, WithName("cn"), WithInputWantedList(nil)) }},
		{"unsupported-wanted-test", "not supported", func() { NewTest(lib.ActionAdd, WithInputWantedList(nil)) }},
		{"unsupported-iptype", "not supported", func() { NewTest(lib.ActionAdd, WithInputOnlyIPType(lib.IPv4)) }},
		{"json-stdin-missing", "missing name", func() { NewStdinFromBytes(lib.ActionAdd, nil) }},
		{"json-private-iptype", "invalid onlyIPType", func() { NewPrivateFromBytes(lib.ActionAdd, []byte(`{"onlyIPType":"ipv5"}`)) }},
		{"json-cutter-blank", "wantedList", func() { NewCutterFromBytes(lib.ActionRemove, []byte(`{"wantedList":[" "]}`)) }},
		{"json-test-action", "add or remove", func() { NewTestFromBytes(lib.ActionOutput, nil) }},
		{"json-stdout-iptype", "invalid onlyIPType", func() { NewStdoutFromBytes(lib.ActionOutput, []byte(`{"onlyIPType":"ipv5"}`)) }},
		{"json-lookup-missing", "search target", func() { NewLookupFromBytes(lib.ActionOutput, nil) }},
		{"json-lookup-invalid", "invalid IP or CIDR", func() { NewLookupFromBytes(lib.ActionOutput, []byte(`{"search":"invalid"}`)) }},
	}
	if name := os.Getenv("GEOIP_SPECIAL_INVALID_CASE"); name != "" {
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
			cmd := exec.Command(os.Args[0], "-test.run=^TestSpecialConstructorValidation$")
			cmd.Env = append(os.Environ(), "GEOIP_SPECIAL_INVALID_CASE="+tc.name)
			output, err := cmd.CombinedOutput()
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 1 || !strings.Contains(string(output), tc.want) {
				t.Fatalf("expected fatal %q, got %v: %s", tc.want, err, output)
			}
		})
	}
}

func TestPrivateCutterAndTestBehavior(t *testing.T) {
	container := lib.NewContainer()
	if _, err := NewPrivate(lib.ActionAdd).Input(container); err != nil {
		t.Fatal(err)
	}
	assertSpecialLookup(t, container, "10.1.2.3", "private", true)
	assertSpecialLookup(t, container, "::1", "private", true)
	if _, err := NewCutter(lib.ActionRemove, WithInputWantedList([]string{" private "}), WithInputOnlyIPType(" IPV4 ")).Input(container); err != nil {
		t.Fatal(err)
	}
	assertSpecialLookup(t, container, "10.1.2.3", "private", false)
	assertSpecialLookup(t, container, "::1", "private", true)
	if _, err := NewPrivate(lib.ActionRemove, WithInputOnlyIPType(" IPV6 ")).Input(container); err != nil {
		t.Fatal(err)
	}
	assertSpecialLookup(t, container, "::1", "private", false)

	if _, err := NewTest(lib.ActionAdd).Input(container); err != nil {
		t.Fatal(err)
	}
	assertSpecialLookup(t, container, "127.0.0.1", "test", true)
	if _, err := NewTest(lib.ActionRemove).Input(container); err != nil {
		t.Fatal(err)
	}
	assertSpecialLookup(t, container, "127.0.0.1", "test", false)
	if _, err := NewCutter(lib.ActionRemove, WithInputWantedList([]string{" test "})).Input(container); err != nil {
		t.Fatal(err)
	}
	if _, found := container.GetEntry("test"); found {
		t.Fatal("cutter did not remove the selected entry")
	}
	if _, found := container.GetEntry("private"); !found {
		t.Fatal("cutter removed an unselected entry")
	}
}

func TestStdinStdoutAndLookupBehavior(t *testing.T) {
	container := lib.NewContainer()
	input := "# comment\n192.0.2.0/24 # IPv4\n2001:db8::/32 // IPv6\ninvalid\n\n"
	readSpecialStdin(t, input, NewStdin(lib.ActionAdd, WithName(" cn ")), container)
	readSpecialStdin(t, "192.0.2.0/25\n", NewStdin(lib.ActionRemove, WithName("cn"), WithInputOnlyIPType(" IPv4 ")), container)
	assertSpecialLookup(t, container, "192.0.2.1", "cn", false)
	assertSpecialLookup(t, container, "192.0.2.128/25", "cn", true)
	assertSpecialLookup(t, container, "2001:db8::1", "cn", true)
	readSpecialStdin(t, input, NewStdin(lib.ActionAdd, WithName(" us "), WithInputOnlyIPType(" IPV4 ")), container)
	assertSpecialLookup(t, container, "2001:db8::1", "us", false)
	out := NewStdout(lib.ActionOutput,
		WithOutputWantedList([]string{" us ", "cn", "CN"}), WithOutputExcludedList([]string{"US"}),
		WithOutputOnlyIPType(" IPv4 "),
	)
	if got := captureSpecialOutput(t, func() error { return out.Output(container) }); got != "192.0.2.128/25\n192.0.2.128/25\n" {
		t.Fatalf("unexpected stdout output: %q", got)
	}
	if got := NewStdout(lib.ActionOutput).(*stdout).filterAndSortList(container); !slices.Equal(got, []string{"CN", "US"}) {
		t.Fatalf("stdout did not sort all lists: %v", got)
	}
	filtered := NewStdout(lib.ActionOutput, WithOutputWantedList([]string{"CN"}), WithOutputExcludedList([]string{" cn "})).(*stdout)
	if got := filtered.filterAndSortList(container); len(got) != 0 {
		t.Fatalf("excluded wanted lists must not fall back to all lists: %v", got)
	}
	for _, tc := range []struct {
		search string
		lists  []string
		want   string
	}{
		{" 192.0.2.128 ", nil, "cn,us\n"},
		{"192.0.2.128/25", []string{" cn ", ""}, "cn\n"},
		{"2001:db8::1", nil, "cn\n"},
		{"2001:db8::1", []string{" us "}, "false\n"},
		{"203.0.113.1", nil, "false\n"},
	} {
		lookup := NewLookup(lib.ActionOutput, WithSearch(tc.search), WithSearchList(tc.lists))
		if got := captureSpecialOutput(t, func() error { return lookup.Output(container) }); got != tc.want {
			t.Fatalf("lookup %s: got %q, want %q", tc.search, got, tc.want)
		}
	}
}

func assertSpecialLookup(t *testing.T, container lib.Container, search, list string, want bool) {
	t.Helper()
	_, found, err := container.Lookup(search, list)
	if err != nil || found != want {
		t.Fatalf("lookup %s in %s: found %t, want %t, err %v", search, list, found, want, err)
	}
}

func readSpecialStdin(t *testing.T, input string, converter lib.InputConverter, container lib.Container) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	if _, err := io.WriteString(w, input); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	original := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = original }()
	if _, err := converter.Input(container); err != nil {
		t.Fatal(err)
	}
}

func captureSpecialOutput(t *testing.T, output func() error) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	original := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = original }()
	if err := output(); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
