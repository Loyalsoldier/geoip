package maxmind

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func testDirectory(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp(".", ".maxmind-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	})
	return dir
}

func writeFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func addTestEntry(t *testing.T, container lib.Container, name string, prefixes ...string) {
	t.Helper()
	entry := lib.NewEntry(name)
	for _, prefix := range prefixes {
		if err := entry.AddPrefix(prefix); err != nil {
			t.Fatal(err)
		}
	}
	if err := container.Add(entry); err != nil {
		t.Fatal(err)
	}
}

func assertPrefixes(t *testing.T, container lib.Container, name string, want []string) {
	t.Helper()
	entry, found := container.GetEntry(name)
	if !found {
		t.Fatalf("entry %s not found", name)
	}
	got, err := entry.MarshalText()
	if len(want) == 0 {
		if err == nil || err.Error() != fmt.Sprintf("entry %s has no prefix", name) {
			t.Fatalf("expected an empty entry, got prefixes %v and error %v", got, err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, got, want)
}

func TestOutputListOrdering(t *testing.T) {
	container := lib.NewContainer()
	for _, name := range []string{"US", "CN", "JP", "CUSTOM"} {
		addTestEntry(t, container, name, "192.0.2.0/24")
	}
	tests := []struct {
		name string
		opts []lib.OutputOption
		want []string
	}{
		{"default", nil, []string{"CN", "CUSTOM", "JP", "US"}},
		{"overwrite", []lib.OutputOption{WithOutputOverwriteList([]string{" us ", "cn", "US"})}, []string{"CUSTOM", "JP", "US", "CN", "US"}},
		{"wanted-priority", []lib.OutputOption{
			WithOutputWantedList([]string{" us ", "", "cn", "US"}), WithOutputOverwriteList([]string{"CN", "US"}),
		}, []string{"US", "CN", "US"}},
		{"exclude-wanted", []lib.OutputOption{
			WithOutputWantedList([]string{"US", "CN"}), WithOutputExcludedList([]string{" us "}),
		}, []string{"CN"}},
		{"exclude-all-wanted", []lib.OutputOption{
			WithOutputWantedList([]string{"US"}), WithOutputExcludedList([]string{" us "}),
		}, []string{}},
		{"exclude-overwrite", []lib.OutputOption{
			WithOutputOverwriteList([]string{"US", "CN"}), WithOutputExcludedList([]string{" us "}),
		}, []string{"CUSTOM", "JP", "CN"}},
		{"blank-wanted", []lib.OutputOption{WithOutputWantedList([]string{" "})}, []string{"CN", "CUSTOM", "JP", "US"}},
	}
	for _, constructor := range outputConstructors {
		for _, tc := range tests {
			t.Run(constructor.name+"/"+tc.name, func(t *testing.T) {
				got := constructor.new(lib.ActionOutput, tc.opts...).(*geoLite2CountryMMDBOut)
				assertEqual(t, got.filterAndSortList(container), tc.want)
			})
		}
	}
}

func TestMMDBConversionWithOptions(t *testing.T) {
	for i, tc := range outputConstructors {
		t.Run(tc.name, func(t *testing.T) {
			dir := testDirectory(t)
			container := lib.NewContainer()
			for _, name := range []string{"CN", "CUSTOM"} {
				addTestEntry(t, container, name, "192.0.2.0/24", "2001:db8::/32")
			}

			out := tc.new(lib.ActionOutput, WithOutputDir(dir), WithOutputOverwriteList([]string{" cn "}))
			if err := out.Output(container); err != nil {
				t.Fatal(err)
			}
			sourcePath := filepath.Join(dir, "Country.mmdb")
			in := inputConstructors[i].new(lib.ActionAdd, WithURI(sourcePath), WithInputWantedList([]string{" cn "}))
			result, err := in.Input(lib.NewContainer())
			if err != nil {
				t.Fatal(err)
			}
			assertEqual(t, result.Len(), 1)
			assertPrefixes(t, result, "CN", []string{"192.0.2.0/24", "2001:db8::/32"})

			out = tc.new(lib.ActionOutput,
				WithOutputDir(dir), WithOutputName("IPv4.mmdb"), WithOutputOnlyIPType(" IPv4 "),
				WithOutputWantedList([]string{"CN"}), WithOutputSourceMMDBURI(sourcePath),
			)
			if err := out.Output(container); err != nil {
				t.Fatal(err)
			}
			in = inputConstructors[i].new(lib.ActionAdd, WithURI(filepath.Join(dir, "IPv4.mmdb")))
			result, err = in.Input(lib.NewContainer())
			if err != nil {
				t.Fatal(err)
			}
			assertPrefixes(t, result, "CN", []string{"192.0.2.0/24"})

			in = inputConstructors[i].new(lib.ActionAdd, WithURI(sourcePath), WithInputOnlyIPType(" IPv6 "))
			result, err = in.Input(lib.NewContainer())
			if err != nil {
				t.Fatal(err)
			}
			assertPrefixes(t, result, "CN", []string{"2001:db8::/32"})
			in = inputConstructors[i].new(lib.ActionRemove, WithURI(sourcePath), WithInputOnlyIPType(lib.IPv6))
			result, err = in.Input(result)
			if err != nil {
				t.Fatal(err)
			}
			assertPrefixes(t, result, "CN", nil)
		})
	}
}

func TestCSVConversionWithOptions(t *testing.T) {
	dir := testDirectory(t)
	countryFile := writeFixture(t, dir, "locations.csv", "geoname_id,locale_code,continent_code,continent_name,country_iso_code\n1,en,NA,North America,US\n2,en,AS,Asia,CN\n")
	countryIPv4 := writeFixture(t, dir, "country-v4.csv", "network,geoname_id,registered_country_geoname_id,represented_country_geoname_id\n192.0.2.0/24,0,1,0\n198.51.100.0/24,2,0,0\n")
	countryIPv6 := writeFixture(t, dir, "country-v6.csv", "network,geoname_id,registered_country_geoname_id,represented_country_geoname_id\n2001:db8::/32,1,0,0\n")
	asnIPv4 := writeFixture(t, dir, "asn-v4.csv", "network,autonomous_system_number,autonomous_system_organization\n192.0.2.0/24,123,Example\n198.51.100.0/24,456,Other\n")
	asnIPv6 := writeFixture(t, dir, "asn-v6.csv", "network,autonomous_system_number,autonomous_system_organization\n2001:db8::/32,123,Example\n")

	tests := []struct {
		name  string
		new   func(lib.Action, ...lib.InputOption) lib.InputConverter
		opts  []lib.InputOption
		entry string
	}{
		{"country", NewGeoLite2CountryCSVIn, []lib.InputOption{
			WithCountryCodeFile(countryFile), WithIPv4File(countryIPv4), WithInputWantedList([]string{" us "}),
		}, "US"},
		{"asn", NewGeoLite2ASNCSVIn, []lib.InputOption{
			WithIPv4File(asnIPv4), WithInputASNWantedList(lib.WantedListExtended{TypeMap: map[string][]string{" custom ": {" as123 "}}}),
		}, "CUSTOM"},
	}
	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.new(lib.ActionAdd, tc.opts...).Input(lib.NewContainer())
			if err != nil {
				t.Fatal(err)
			}
			assertEqual(t, result.Len(), 1)
			assertPrefixes(t, result, tc.entry, []string{"192.0.2.0/24"})

			ipv6 := []string{countryIPv6, asnIPv6}[i]
			opts := append(tc.opts, WithIPv6File(ipv6), WithInputOnlyIPType(" IPv6 "))
			result, err = tc.new(lib.ActionAdd, opts...).Input(result)
			if err != nil {
				t.Fatal(err)
			}
			assertPrefixes(t, result, tc.entry, []string{"192.0.2.0/24", "2001:db8::/32"})
			result, err = tc.new(lib.ActionRemove, opts...).Input(result)
			if err != nil {
				t.Fatal(err)
			}
			assertPrefixes(t, result, tc.entry, []string{"192.0.2.0/24"})
		})
	}
}
