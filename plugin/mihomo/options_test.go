package mihomo

import (
	"bytes"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestMRSConstructors(t *testing.T) {
	in := NewMRSIn(lib.ActionAdd, nil,
		WithNameAndURI(" cn ", " cn.mrs "),
		WithInputWantedList([]string{" cn ", "", "US", "cn"}),
		WithInputOnlyIPType(" IPv6 "),
	).(*mrs_in)
	if in.GetType() != TypeMRSIn || in.GetAction() != lib.ActionAdd || in.GetDescription() != DescMRSIn {
		t.Fatalf("unexpected input metadata: %+v", in)
	}
	if in.Name != "cn" || in.URI != "cn.mrs" || in.OnlyIPType != lib.IPv6 ||
		!reflect.DeepEqual(in.Want, map[string]bool{"CN": true, "US": true}) {
		t.Fatalf("input options not normalized: %+v", in)
	}
	fromJSON, err := NewMRSInFromBytes(lib.ActionAdd, []byte(`{"name":" cn ","uri":" cn.mrs ","wantedList":[" cn ","","US","cn"],"onlyIPType":" IPv6 "}`))
	if err != nil || !reflect.DeepEqual(in, fromJSON) {
		t.Fatalf("input JSON mismatch: got %+v, err %v", fromJSON, err)
	}

	dir := NewMRSIn(lib.ActionRemove, WithInputDir(" ./rules ")).(*mrs_in)
	if dir.InputDir != "./rules" || dir.Name != "" || dir.URI != "" || dir.OnlyIPType != "" || len(dir.Want) != 0 {
		t.Fatalf("unexpected input directory defaults: %+v", dir)
	}
	dirJSON, err := NewMRSInFromBytes(lib.ActionRemove, []byte(`{"inputDir":" ./rules "}`))
	if err != nil || dirJSON.(*mrs_in).InputDir != dir.InputDir {
		t.Fatalf("input directory JSON mismatch: %+v, %v", dirJSON, err)
	}

	out := NewMRSOut(lib.ActionOutput, nil, WithOutputDir(" ")).(*mrs_out)
	if out.GetType() != TypeMRSOut || out.GetAction() != lib.ActionOutput || out.GetDescription() != DescMRSOut ||
		out.OutputDir != defaultOutputDir || out.OnlyIPType != "" || len(out.Want) != 0 || len(out.Exclude) != 0 {
		t.Fatalf("unexpected output defaults: %+v", out)
	}
	for _, data := range [][]byte{nil, {}, []byte(`{}`), []byte(`null`)} {
		got, err := NewMRSOutFromBytes(lib.ActionOutput, data)
		if err != nil || !reflect.DeepEqual(out, got) {
			t.Fatalf("output defaults for %q: %+v, %v", data, got, err)
		}
	}
	out = NewMRSOut(lib.ActionOutput,
		WithOutputDir(" ./mrs "), WithOutputWantedList([]string{" us ", "CN", "", "cn"}),
		WithOutputExcludedList([]string{" Us "}), WithOutputOnlyIPType(" IPV4 "),
	).(*mrs_out)
	outJSON, err := NewMRSOutFromBytes(lib.ActionOutput, []byte(`{"outputDir":" ./mrs ","wantedList":[" us ","CN","","cn"],"excludedList":[" Us "],"onlyIPType":" IPV4 "}`))
	if err != nil || !reflect.DeepEqual(out, outJSON) || out.OutputDir != "./mrs" || out.OnlyIPType != lib.IPv4 {
		t.Fatalf("output JSON/options mismatch: %+v, %v", outJSON, err)
	}
	if got := out.filterAndSortList(lib.NewContainer()); !slices.Equal(got, []string{"CN", "CN"}) {
		t.Fatalf("wanted/excluded list normalization or duplicate order changed: %v", got)
	}
}

func TestMRSMalformedJSON(t *testing.T) {
	for _, data := range []string{`{`, `[]`, `{"onlyIPType":7}`} {
		if got, err := NewMRSInFromBytes(lib.ActionAdd, []byte(data)); err == nil || got != nil {
			t.Errorf("input accepted malformed JSON %q", data)
		}
		if got, err := NewMRSOutFromBytes(lib.ActionOutput, []byte(data)); err == nil || got != nil {
			t.Errorf("output accepted malformed JSON %q", data)
		}
	}
}

func TestMRSConstructorValidation(t *testing.T) {
	cases := []struct {
		name string
		want string
		run  func()
	}{
		{"missing", "missing", func() { NewMRSIn(lib.ActionAdd) }},
		{"blank", "missing", func() { NewMRSIn(lib.ActionAdd, WithNameAndURI(" ", "\t"), WithInputDir(" ")) }},
		{"name-only", "together", func() { NewMRSIn(lib.ActionAdd, WithNameAndURI("cn", "")) }},
		{"uri-only", "together", func() { NewMRSIn(lib.ActionAdd, WithNameAndURI("", "cn.mrs")) }},
		{"name-and-dir", "together", func() { NewMRSIn(lib.ActionAdd, WithNameAndURI("cn", ""), WithInputDir("dir")) }},
		{"uri-and-dir", "together", func() { NewMRSIn(lib.ActionAdd, WithNameAndURI("", "cn.mrs"), WithInputDir("dir")) }},
		{"pair-and-dir", "not allowed", func() { NewMRSIn(lib.ActionAdd, WithNameAndURI("cn", "cn.mrs"), WithInputDir("dir")) }},
		{"input-action", "add or remove", func() { NewMRSIn(lib.ActionOutput, WithInputDir("dir")) }},
		{"input-empty-action", "add or remove", func() { NewMRSIn("", WithInputDir("dir")) }},
		{"input-iptype", "invalid onlyIPType", func() { NewMRSIn(lib.ActionAdd, WithInputDir("dir"), WithInputOnlyIPType("ipv5")) }},
		{"output-action", "output action", func() { NewMRSOut(lib.ActionAdd) }},
		{"output-iptype", "invalid onlyIPType", func() { NewMRSOut(lib.ActionOutput, WithOutputOnlyIPType("ipv5")) }},
		{"json-missing", "missing", func() { NewMRSInFromBytes(lib.ActionAdd, nil) }},
		{"json-name-only", "together", func() { NewMRSInFromBytes(lib.ActionAdd, []byte(`{"name":"cn"}`)) }},
		{"json-conflict", "not allowed", func() { NewMRSInFromBytes(lib.ActionAdd, []byte(`{"name":"cn","uri":"cn.mrs","inputDir":"dir"}`)) }},
		{"json-input-iptype", "invalid onlyIPType", func() { NewMRSInFromBytes(lib.ActionAdd, []byte(`{"inputDir":"dir","onlyIPType":"ipv5"}`)) }},
		{"json-output-iptype", "invalid onlyIPType", func() { NewMRSOutFromBytes(lib.ActionOutput, []byte(`{"onlyIPType":"ipv5"}`)) }},
	}
	if name := os.Getenv("GEOIP_MRS_INVALID_CASE"); name != "" {
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
			cmd := exec.Command(os.Args[0], "-test.run=^TestMRSConstructorValidation$")
			cmd.Env = append(os.Environ(), "GEOIP_MRS_INVALID_CASE="+tc.name)
			output, err := cmd.CombinedOutput()
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 1 || !strings.Contains(string(output), tc.want) {
				t.Fatalf("expected fatal %q, got %v: %s", tc.want, err, output)
			}
		})
	}
}

func TestMRSInMemoryRoundTrip(t *testing.T) {
	entry := lib.NewEntry("cn")
	for _, prefix := range []string{"192.0.2.0/24", "2001:db8::/32"} {
		if err := entry.AddPrefix(prefix); err != nil {
			t.Fatal(err)
		}
	}
	ranges, err := entry.MarshalIPRange()
	if err != nil {
		t.Fatal(err)
	}
	var data bytes.Buffer
	out := NewMRSOut(lib.ActionOutput).(*mrs_out)
	if err := out.convertToMrs(ranges, &data); err != nil {
		t.Fatal(err)
	}
	in := NewMRSIn(lib.ActionAdd, WithNameAndURI("cn", "cn.mrs"), WithInputWantedList([]string{" cn "})).(*mrs_in)
	entries := make(map[string]*lib.Entry)
	if err := in.generateEntries("cn", &data, entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries["CN"] == nil {
		t.Fatalf("unexpected entries: %v", entries)
	}
	got, err := entries["CN"].MarshalText()
	if err != nil || !slices.Equal(got, []string{"192.0.2.0/24", "2001:db8::/32"}) {
		t.Fatalf("CIDR round trip: %v, %v", got, err)
	}
	if err := in.generateEntries("us", strings.NewReader("not MRS"), entries); err != nil || len(entries) != 1 {
		t.Fatalf("wanted filter must run before parsing: %v", err)
	}
	container := lib.NewContainer()
	for _, name := range []string{"US", "CN", "JP"} {
		if err := container.Add(lib.NewEntry(name)); err != nil {
			t.Fatal(err)
		}
	}
	out = NewMRSOut(lib.ActionOutput, WithOutputExcludedList([]string{" us "})).(*mrs_out)
	if got := out.filterAndSortList(container); !slices.Equal(got, []string{"CN", "JP"}) {
		t.Fatalf("unexpected sorted output: %v", got)
	}
	out = NewMRSOut(lib.ActionOutput, WithOutputWantedList([]string{"CN"}), WithOutputExcludedList([]string{" cn "})).(*mrs_out)
	if got := out.filterAndSortList(container); len(got) != 0 {
		t.Fatalf("excluded wanted lists must not fall back to all lists: %v", got)
	}
}
