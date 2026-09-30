package plaintext

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func assertPrefixes(t *testing.T, container lib.Container, name string, want []string) {
	t.Helper()
	entry, found := container.GetEntry(name)
	if !found {
		t.Fatalf("entry %q not found", name)
	}
	got, err := entry.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("entry %q: got %v, want %v", name, got, want)
	}
}

func TestInlineTextInput(t *testing.T) {
	cidrs := []string{" 192.0.2.1 ", "2001:db8::/32"}
	parsed, err := NewTextInFromBytes(lib.ActionAdd, []byte(`{
		"name":" cn ", "ipOrCIDR":[" 192.0.2.1 ","2001:db8::/32"]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	direct := NewTextIn(lib.ActionAdd, WithNameAndIPOrCIDR(" cn ", cidrs), nil)
	if !reflect.DeepEqual(direct, parsed) {
		t.Fatalf("direct %#v differs from JSON %#v", direct, parsed)
	}
	for _, input := range []lib.InputConverter{direct, parsed} {
		container, err := input.Input(lib.NewContainer())
		if err != nil {
			t.Fatal(err)
		}
		assertPrefixes(t, container, "CN", []string{"192.0.2.1/32", "2001:db8::/32"})
		remove := NewTextIn(lib.ActionRemove,
			WithNameAndIPOrCIDR("cn", cidrs),
			WithInputOnlyIPType(lib.IPv6),
		)
		container, err = remove.Input(container)
		if err != nil {
			t.Fatal(err)
		}
		assertPrefixes(t, container, "CN", []string{"192.0.2.1/32"})
	}
}

func TestTextFileAndInlineInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cn.txt")
	if err := os.WriteFile(path, []byte(" IP-CIDR,192.0.2.0/24,no-resolve # comment\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cidrs := []string{"2001:db8::/32"}
	parsed, err := NewTextInFromBytes(lib.ActionAdd, configBytes(t, map[string]any{
		"name": " cn ", "uri": path, "ipOrCIDR": cidrs,
		"removePrefixesInLine": []string{"ip-cidr,"}, "removeSuffixesInLine": []string{",no-resolve"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []lib.InputConverter{
		parsed,
		NewTextIn(lib.ActionAdd,
			WithNameAndURI("cn", path), WithNameAndIPOrCIDR("cn", cidrs),
			WithRemovePrefixesInLine([]string{"ip-cidr,"}), WithRemoveSuffixesInLine([]string{",no-resolve"}),
		),
		NewTextIn(lib.ActionAdd,
			WithNameAndIPOrCIDR("cn", cidrs), WithNameAndURI("cn", path),
			WithRemovePrefixesInLine([]string{"ip-cidr,"}), WithRemoveSuffixesInLine([]string{",no-resolve"}),
		),
	} {
		container, err := input.Input(lib.NewContainer())
		if err != nil {
			t.Fatal(err)
		}
		assertPrefixes(t, container, "CN", []string{"192.0.2.0/24", "2001:db8::/32"})
	}
}

func TestInputFileFormats(t *testing.T) {
	sources := map[string]string{
		TypeTextIn:                  "# comment\n192.0.2.0/24 // comment\n2001:db8::/32\n",
		TypeJSONIn:                  `{"prefixes":["192.0.2.0/24",["2001:db8::/32"]]}`,
		TypeClashRuleSetIPCIDRIn:    "payload:\n  - '192.0.2.0/24'\n  - '2001:db8::/32'\n",
		TypeClashRuleSetClassicalIn: "payload:\n  - IP-CIDR,192.0.2.0/24,no-resolve\n  - IP-CIDR6,2001:db8::/32\n  - DOMAIN,example.com\n",
		TypeSurgeRuleSetIn:          "# comment\nIP-CIDR,192.0.2.0/24,no-resolve\nIP-CIDR6,2001:db8::/32\nDOMAIN,example.com\n",
	}
	for _, format := range inputFormats {
		t.Run(format.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "cn.txt")
			if err := os.WriteFile(path, []byte(sources[format.name]), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "us.txt"), []byte("not a valid IP"), 0644); err != nil {
				t.Fatal(err)
			}
			parsed, err := format.fromBytes(lib.ActionAdd, configBytes(t, map[string]any{
				"name": " cn ", "uri": path, "jsonPath": []string{" prefixes "},
			}))
			if err != nil {
				t.Fatal(err)
			}
			for _, input := range []lib.InputConverter{
				parsed,
				format.new(lib.ActionAdd, WithNameAndURI(" cn ", path), WithJSONPath([]string{" prefixes "})),
				format.new(lib.ActionAdd, WithInputDir(dir), WithInputWantedList([]string{" cn ", ""}), WithJSONPath([]string{"prefixes"})),
			} {
				container, err := input.Input(lib.NewContainer())
				if err != nil {
					t.Fatal(err)
				}
				if container.Len() != 1 {
					t.Fatalf("got %d entries, want 1", container.Len())
				}
				assertPrefixes(t, container, "CN", []string{"192.0.2.0/24", "2001:db8::/32"})
			}
			input := format.new(lib.ActionAdd, WithNameAndURI("cn", path), WithJSONPath([]string{"prefixes"}), WithInputOnlyIPType(lib.IPv4))
			container, err := input.Input(lib.NewContainer())
			if err != nil {
				t.Fatal(err)
			}
			assertPrefixes(t, container, "CN", []string{"192.0.2.0/24"})
		})
	}
}

func TestJSONPathSemantics(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		path   string
	}{
		{"root-array", `["192.0.2.0/24","2001:db8::/32"]`, "@this"},
		{"root-array-trimmed", `["192.0.2.0/24","2001:db8::/32"]`, " @this "},
		{"empty-key", `{"":["192.0.2.0/24","2001:db8::/32"]}`, ""},
		{"empty-key-trimmed", `{"":["192.0.2.0/24","2001:db8::/32"]}`, " "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "cn.json")
			if err := os.WriteFile(path, []byte(tc.source), 0644); err != nil {
				t.Fatal(err)
			}
			parsed, err := NewJSONInFromBytes(lib.ActionAdd, configBytes(t, map[string]any{
				"name": "cn", "uri": path, "jsonPath": []string{tc.path},
			}))
			if err != nil {
				t.Fatal(err)
			}
			for _, input := range []lib.InputConverter{
				parsed,
				NewJSONIn(lib.ActionAdd, WithNameAndURI("cn", path), WithJSONPath([]string{tc.path})),
			} {
				container, err := input.Input(lib.NewContainer())
				if err != nil {
					t.Fatal(err)
				}
				assertPrefixes(t, container, "CN", []string{"192.0.2.0/24", "2001:db8::/32"})
			}
		})
	}
}

func TestJSONSingleSourceWantedList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cn.json")
	if err := os.WriteFile(path, []byte(`{"prefixes":["192.0.2.0/24","2001:db8::/32"]}`), 0644); err != nil {
		t.Fatal(err)
	}
	parsed, err := NewJSONInFromBytes(lib.ActionAdd, configBytes(t, map[string]any{
		"name": " cn ", "uri": path, "jsonPath": []string{"prefixes"}, "wantedList": []string{" CN ", "", "cn"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	direct := NewJSONIn(lib.ActionAdd,
		WithNameAndURI(" cn ", path), WithJSONPath([]string{"prefixes"}),
		WithInputWantedList([]string{" CN ", "", "cn"}),
	)
	if !reflect.DeepEqual(direct, parsed) {
		t.Fatalf("direct %#v differs from JSON %#v", direct, parsed)
	}
	for _, input := range []lib.InputConverter{direct, parsed} {
		container, err := input.Input(lib.NewContainer())
		if err != nil {
			t.Fatal(err)
		}
		assertPrefixes(t, container, "CN", []string{"192.0.2.0/24", "2001:db8::/32"})
	}
}

func TestOutputFormatsWithDefaults(t *testing.T) {
	expected := map[string]string{
		TypeTextOut:                  "192.0.2.0/24\n2001:db8::/32\n",
		TypeClashRuleSetIPCIDROut:    "payload:\n  - '192.0.2.0/24'\n  - '2001:db8::/32'\n",
		TypeClashRuleSetClassicalOut: "payload:\n  - IP-CIDR,192.0.2.0/24\n  - IP-CIDR6,2001:db8::/32\n",
		TypeSurgeRuleSetOut:          "IP-CIDR,192.0.2.0/24\nIP-CIDR6,2001:db8::/32\n",
	}
	for _, format := range outputFormats {
		t.Run(format.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			container, err := NewTextIn(lib.ActionAdd,
				WithNameAndIPOrCIDR("CN", []string{"192.0.2.0/24", "2001:db8::/32"}),
			).Input(lib.NewContainer())
			if err != nil {
				t.Fatal(err)
			}
			if err := format.new(lib.ActionOutput).Output(container); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(format.defaultDir, "cn.txt"))
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != expected[format.name] {
				t.Fatalf("got %q, want %q", data, expected[format.name])
			}
		})
	}
}

func TestOutputFiltersAndLineOptions(t *testing.T) {
	container := lib.NewContainer()
	for _, name := range []string{"CN", "US", "JP"} {
		var err error
		container, err = NewTextIn(lib.ActionAdd,
			WithNameAndIPOrCIDR(name, []string{"192.0.2.0/24", "2001:db8::/32"}),
		).Input(container)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, format := range outputFormats {
		t.Run(format.name, func(t *testing.T) {
			dir := t.TempDir()
			output := format.new(lib.ActionOutput,
				WithOutputDir(dir), WithOutputExtension(".conf"),
				WithOutputWantedList([]string{"us", " cn "}), WithOutputExcludedList([]string{" US "}),
				WithOutputOnlyIPType(lib.IPv6), WithAddPrefixInLine(" prefix "), WithAddSuffixInLine(" suffix "),
			)
			if err := output.Output(container); err != nil {
				t.Fatal(err)
			}
			files, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(files) != 1 || files[0].Name() != "cn.conf" {
				t.Fatalf("expected only cn.conf, got %v", files)
			}
			want := map[string]string{
				TypeTextOut:                  " prefix 2001:db8::/32 suffix \n",
				TypeClashRuleSetIPCIDROut:    "payload:\n  - '2001:db8::/32'\n",
				TypeClashRuleSetClassicalOut: "payload:\n  - IP-CIDR6,2001:db8::/32\n",
				TypeSurgeRuleSetOut:          "IP-CIDR6,2001:db8::/32 suffix \n",
			}
			data, err := os.ReadFile(filepath.Join(dir, "cn.conf"))
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != want[format.name] {
				t.Fatalf("got %q, want %q", data, want[format.name])
			}
			filtered := format.new(lib.ActionOutput,
				WithOutputWantedList([]string{"cn"}), WithOutputExcludedList([]string{" CN "}),
			).(*textOut).filterAndSortList(container)
			if len(filtered) != 0 {
				t.Fatalf("all wanted entries were excluded, got %v", filtered)
			}
		})
	}
}
