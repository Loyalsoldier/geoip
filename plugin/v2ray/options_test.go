package v2ray

import (
	"bytes"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"google.golang.org/protobuf/proto"
)

func TestGeoIPDatConstructors(t *testing.T) {
	in := NewGeoIPDatIn(lib.ActionRemove, nil, WithURI(" ./geoip.dat "),
		WithInputWantedList([]string{" cn ", "", "US", "cn"}), WithInputOnlyIPType(" IPv6 "),
	).(*geoip_dat_in)
	if in.GetType() != TypeGeoIPDatIn || in.GetAction() != lib.ActionRemove || in.GetDescription() != DescGeoIPDatIn {
		t.Fatalf("unexpected input metadata: %+v", in)
	}
	if in.URI != "./geoip.dat" || in.OnlyIPType != lib.IPv6 ||
		!reflect.DeepEqual(in.Want, map[string]bool{"CN": true, "US": true}) {
		t.Fatalf("input options not normalized: %+v", in)
	}
	inJSON, err := NewGeoIPDatInFromBytes(lib.ActionRemove, []byte(`{"uri":" ./geoip.dat ","wantedList":[" cn ","","US","cn"],"onlyIPType":" IPv6 "}`))
	if err != nil || !reflect.DeepEqual(in, inJSON) {
		t.Fatalf("input JSON mismatch: %+v, %v", inJSON, err)
	}
	minimal := NewGeoIPDatIn(lib.ActionAdd, WithURI("geoip.dat")).(*geoip_dat_in)
	if minimal.OnlyIPType != "" || len(minimal.Want) != 0 {
		t.Fatalf("unexpected input defaults: %+v", minimal)
	}
	minimalJSON, err := NewGeoIPDatInFromBytes(lib.ActionAdd, []byte(`{"uri":"geoip.dat"}`))
	if err != nil || minimalJSON.(*geoip_dat_in).URI != minimal.URI {
		t.Fatalf("URI must be sufficient without a name: %+v, %v", minimalJSON, err)
	}

	out := NewGeoIPDatOut(lib.ActionOutput, nil, WithOutputName(" "), WithOutputDir("\t")).(*geoip_dat_out)
	if out.GetType() != TypeGeoIPDatOut || out.GetAction() != lib.ActionOutput || out.GetDescription() != DescGeoIPDatOut ||
		out.OutputName != defaultOutputName || out.OutputDir != defaultOutputDir || out.OneFilePerList ||
		out.OnlyIPType != "" || len(out.Want) != 0 || len(out.Exclude) != 0 {
		t.Fatalf("unexpected output defaults: %+v", out)
	}
	for _, data := range [][]byte{nil, {}, []byte(`{}`), []byte(`null`)} {
		got, err := NewGeoIPDatOutFromBytes(lib.ActionOutput, data)
		if err != nil || !reflect.DeepEqual(out, got) {
			t.Fatalf("output defaults for %q: %+v, %v", data, got, err)
		}
	}
	out = NewGeoIPDatOut(lib.ActionOutput,
		WithOutputDir(" ./dat "), WithOutputName(" custom.dat "),
		WithOutputWantedList([]string{" us ", "CN", "", "cn"}), WithOutputExcludedList([]string{" Us "}),
		WithOneFilePerList(true), WithOutputOnlyIPType(" IPV4 "),
	).(*geoip_dat_out)
	outJSON, err := NewGeoIPDatOutFromBytes(lib.ActionOutput, []byte(`{"outputDir":" ./dat ","outputName":" custom.dat ","wantedList":[" us ","CN","","cn"],"excludedList":[" Us "],"oneFilePerList":true,"onlyIPType":" IPV4 "}`))
	if err != nil || !reflect.DeepEqual(out, outJSON) ||
		out.OutputDir != "./dat" || out.OutputName != "custom.dat" || !out.OneFilePerList || out.OnlyIPType != lib.IPv4 {
		t.Fatalf("output JSON/options mismatch: %+v, %v", outJSON, err)
	}
	if got := out.filterAndSortList(lib.NewContainer()); !slices.Equal(got, []string{"CN", "CN"}) {
		t.Fatalf("wanted/excluded list normalization or duplicate order changed: %v", got)
	}
}

func TestGeoIPDatMalformedJSON(t *testing.T) {
	for _, data := range []string{`{`, `[]`, `{"onlyIPType":7}`} {
		if got, err := NewGeoIPDatInFromBytes(lib.ActionAdd, []byte(data)); err == nil || got != nil {
			t.Errorf("input accepted malformed JSON %q", data)
		}
		if got, err := NewGeoIPDatOutFromBytes(lib.ActionOutput, []byte(data)); err == nil || got != nil {
			t.Errorf("output accepted malformed JSON %q", data)
		}
	}
	if got, err := NewGeoIPDatOutFromBytes(lib.ActionOutput, []byte(`{"oneFilePerList":"true"}`)); err == nil || got != nil {
		t.Error("accepted invalid oneFilePerList type")
	}
}

func TestGeoIPDatConstructorValidation(t *testing.T) {
	cases := []struct {
		name string
		want string
		run  func()
	}{
		{"missing-uri", "missing uri", func() { NewGeoIPDatIn(lib.ActionAdd) }},
		{"blank-uri", "missing uri", func() { NewGeoIPDatIn(lib.ActionAdd, WithURI(" \t ")) }},
		{"input-action", "add or remove", func() { NewGeoIPDatIn(lib.ActionOutput, WithURI("geoip.dat")) }},
		{"input-empty-action", "add or remove", func() { NewGeoIPDatIn("", WithURI("geoip.dat")) }},
		{"input-iptype", "invalid onlyIPType", func() { NewGeoIPDatIn(lib.ActionAdd, WithURI("geoip.dat"), WithInputOnlyIPType("ip")) }},
		{"output-action", "output action", func() { NewGeoIPDatOut(lib.ActionRemove) }},
		{"output-empty-action", "output action", func() { NewGeoIPDatOut("") }},
		{"output-iptype", "invalid onlyIPType", func() { NewGeoIPDatOut(lib.ActionOutput, WithOutputOnlyIPType("ipv5")) }},
		{"json-missing", "missing uri", func() { NewGeoIPDatInFromBytes(lib.ActionAdd, nil) }},
		{"json-blank", "missing uri", func() { NewGeoIPDatInFromBytes(lib.ActionAdd, []byte(`{"uri":" "}`)) }},
		{"json-input-iptype", "invalid onlyIPType", func() { NewGeoIPDatInFromBytes(lib.ActionAdd, []byte(`{"uri":"geoip.dat","onlyIPType":"ipv5"}`)) }},
		{"json-output-iptype", "invalid onlyIPType", func() { NewGeoIPDatOutFromBytes(lib.ActionOutput, []byte(`{"onlyIPType":"ipv5"}`)) }},
	}
	if name := os.Getenv("GEOIP_DAT_INVALID_CASE"); name != "" {
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
			cmd := exec.Command(os.Args[0], "-test.run=^TestGeoIPDatConstructorValidation$")
			cmd.Env = append(os.Environ(), "GEOIP_DAT_INVALID_CASE="+tc.name)
			output, err := cmd.CombinedOutput()
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 1 || !strings.Contains(string(output), tc.want) {
				t.Fatalf("expected fatal %q, got %v: %s", tc.want, err, output)
			}
		})
	}
}

func TestGeoIPDatInMemoryRoundTrip(t *testing.T) {
	entry := lib.NewEntry("cn")
	for _, prefix := range []string{"192.0.2.0/24", "2001:db8::/32"} {
		if err := entry.AddPrefix(prefix); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		ipType lib.IPType
		want   []string
	}{
		{"", []string{"192.0.2.0/24", "2001:db8::/32"}},
		{lib.IPv4, []string{"192.0.2.0/24"}},
		{lib.IPv6, []string{"2001:db8::/32"}},
	} {
		t.Run(string(tc.ipType), func(t *testing.T) {
			out := NewGeoIPDatOut(lib.ActionOutput, WithOutputOnlyIPType(tc.ipType)).(*geoip_dat_out)
			geoIP, err := out.generateGeoIP(entry)
			if err != nil {
				t.Fatal(err)
			}
			list := &GeoIPList{Entry: []*GeoIP{{CountryCode: "US"}, geoIP}}
			out.sort(list)
			if list.Entry[0].CountryCode != "CN" || list.Entry[1].CountryCode != "US" {
				t.Fatal("country codes are not sorted")
			}
			data, err := proto.Marshal(list)
			if err != nil {
				t.Fatal(err)
			}
			in := NewGeoIPDatIn(lib.ActionAdd, WithURI("geoip.dat"), WithInputWantedList([]string{" cn "})).(*geoip_dat_in)
			entries := make(map[string]*lib.Entry)
			if err := in.generateEntries(bytes.NewReader(data), entries); err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || entries["CN"] == nil {
				t.Fatalf("wanted list filtering failed: %v", entries)
			}
			got, err := entries["CN"].MarshalText()
			if err != nil || !slices.Equal(got, tc.want) {
				t.Fatalf("CIDR round trip: %v, %v", got, err)
			}
		})
	}
	container := lib.NewContainer()
	for _, name := range []string{"US", "CN", "JP"} {
		if err := container.Add(lib.NewEntry(name)); err != nil {
			t.Fatal(err)
		}
	}
	out := NewGeoIPDatOut(lib.ActionOutput, WithOutputExcludedList([]string{" us "})).(*geoip_dat_out)
	if got := out.filterAndSortList(container); !slices.Equal(got, []string{"CN", "JP"}) {
		t.Fatalf("unexpected sorted output: %v", got)
	}
	out = NewGeoIPDatOut(lib.ActionOutput, WithOutputWantedList([]string{"CN"}), WithOutputExcludedList([]string{" cn "})).(*geoip_dat_out)
	if got := out.filterAndSortList(container); len(got) != 0 {
		t.Fatalf("excluded wanted lists must not fall back to all lists: %v", got)
	}
}
