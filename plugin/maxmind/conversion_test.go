package maxmind

import (
	"bytes"
	"encoding/json"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/oschwald/maxminddb-golang/v2"
)

func testDirectory(t *testing.T) string {
	t.Helper()
	dir := ".maxmind-test-" + strings.ReplaceAll(t.Name(), "/", "-") + "-" + strconv.Itoa(os.Getpid())
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	})
	return dir
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func containerPrefixes(t *testing.T, container lib.Container) map[string][]string {
	t.Helper()
	result := make(map[string][]string)
	for entry := range container.Loop() {
		prefixes, err := entry.MarshalText()
		if err != nil {
			t.Fatal(err)
		}
		slices.Sort(prefixes)
		result[entry.GetName()] = prefixes
	}
	return result
}

func TestCountryCSVResolution(t *testing.T) {
	dir := testDirectory(t)
	country, blocks := filepath.Join(dir, "country.csv"), filepath.Join(dir, "blocks.csv")
	writeTestFile(t, country, "geoname_id,locale_code,continent_code,continent_name,country_iso_code\n1,en,AS,Asia,cn\n2,en,NA,America,US\n3,en,AS,Asia,JP\n10,en,AS,Asia,\n")
	writeTestFile(t, blocks, "network,geoname_id,registered_country_geoname_id,represented_country_geoname_id\n1.0.0.0/24,1,2,3\n1.0.1.0/24,10,2,3\n1.0.2.0/24,999,888,3\n1.0.3.0/24,999,888,777\n2001:db8::/32,1,,\n")
	for _, tc := range []struct {
		name string
		want []string
		ip   lib.IPType
		data map[string][]string
	}{
		{"all", nil, "", map[string][]string{"CN": {"1.0.0.0/24", "2001:db8::/32"}, "US": {"1.0.1.0/24"}, "JP": {"1.0.2.0/24"}}},
		{"filtered", []string{" cn ", "us"}, lib.IPv4, map[string][]string{"CN": {"1.0.0.0/24"}, "US": {"1.0.1.0/24"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			direct := NewGeoLite2CountryCSVIn(lib.ActionAdd, WithCountryCodeFile(country), WithIPv4File(blocks),
				WithInputWantedList(tc.want), WithInputOnlyIPType(tc.ip))
			data, err := json.Marshal(map[string]any{"country": country, "ipv4": blocks, "wantedList": tc.want, "onlyIPType": tc.ip})
			if err != nil {
				t.Fatal(err)
			}
			adapter, err := NewGeoLite2CountryCSVInFromBytes(lib.ActionAdd, data)
			if err != nil {
				t.Fatal(err)
			}
			for _, converter := range []lib.InputConverter{direct, adapter} {
				container, err := converter.Input(lib.NewContainer())
				if err != nil {
					t.Fatal(err)
				}
				if got := containerPrefixes(t, container); !reflect.DeepEqual(got, tc.data) {
					t.Fatalf("country resolution = %#v; want %#v", got, tc.data)
				}
			}
		})
	}
}

func TestASNCSVFormsConversion(t *testing.T) {
	dir := testDirectory(t)
	blocks := filepath.Join(dir, "asn.csv")
	writeTestFile(t, blocks, "network,autonomous_system_number,autonomous_system_organization\n1.0.0.0/24,123,First\n1.0.1.0/24,456,Second\n2001:db8::/32,123,First\n")
	for _, tc := range []struct {
		name string
		want any
		ip   lib.IPType
		data map[string][]string
	}{
		{"all", nil, "", map[string][]string{"AS123": {"1.0.0.0/24", "2001:db8::/32"}, "AS456": {"1.0.1.0/24"}}},
		{"array", []string{" as123 "}, lib.IPv4, map[string][]string{"AS123": {"1.0.0.0/24"}}},
		{"map", map[string][]string{" service ": {"AS123"}, "shared": {"123"}}, lib.IPv6,
			map[string][]string{"SERVICE": {"2001:db8::/32"}, "SHARED": {"2001:db8::/32"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			direct := NewGeoLite2ASNCSVIn(lib.ActionAdd, WithIPv4File(blocks), WithInputWantedList(tc.want), WithInputOnlyIPType(tc.ip))
			data, err := json.Marshal(map[string]any{"ipv4": blocks, "wantedList": tc.want, "onlyIPType": tc.ip})
			if err != nil {
				t.Fatal(err)
			}
			adapter, err := NewGeoLite2ASNCSVInFromBytes(lib.ActionAdd, data)
			if err != nil {
				t.Fatal(err)
			}
			for _, converter := range []lib.InputConverter{direct, adapter} {
				container, err := converter.Input(lib.NewContainer())
				if err != nil {
					t.Fatal(err)
				}
				if got := containerPrefixes(t, container); !reflect.DeepEqual(got, tc.data) {
					t.Fatalf("ASN conversion = %#v; want %#v", got, tc.data)
				}
			}
		})
	}
}

func TestMMDBCountryResolution(t *testing.T) {
	for _, tc := range inputConstructors[:3] {
		t.Run(tc.name, func(t *testing.T) {
			writer, err := mmdbwriter.New(mmdbwriter.Options{IncludeReservedNetworks: true})
			if err != nil {
				t.Fatal(err)
			}
			records := []mmdbtype.Map{
				{"country": mmdbtype.Map{"iso_code": mmdbtype.String(" cn ")}, "registered_country": mmdbtype.Map{"iso_code": mmdbtype.String("US")}},
				{"registered_country": mmdbtype.Map{"iso_code": mmdbtype.String(" us ")}, "represented_country": mmdbtype.Map{"iso_code": mmdbtype.String("JP")}},
				{"represented_country": mmdbtype.Map{"iso_code": mmdbtype.String(" jp ")}},
			}
			if tc.name == TypeIPInfoCountryMMDBIn {
				records = []mmdbtype.Map{
					{"country_code": mmdbtype.String(" cn ")},
					{"country_code": mmdbtype.String(" us ")},
					{"country_code": mmdbtype.String(" jp ")},
				}
			}
			for i, record := range records {
				_, prefix, err := net.ParseCIDR("1.0." + strconv.Itoa(i) + ".0/24")
				if err != nil {
					t.Fatal(err)
				}
				if err := writer.Insert(prefix, record); err != nil {
					t.Fatal(err)
				}
			}
			var content bytes.Buffer
			if _, err := writer.WriteTo(&content); err != nil {
				t.Fatal(err)
			}
			for _, wanted := range [][]string{nil, {" cn ", "jp"}} {
				converter := tc.new(lib.ActionAdd, WithInputWantedList(wanted)).(*geoLite2CountryMMDBIn)
				entries := make(map[string]*lib.Entry)
				if err := converter.generateEntries(content.Bytes(), entries); err != nil {
					t.Fatal(err)
				}
				container := lib.NewContainer()
				for _, entry := range entries {
					if err := container.Add(entry); err != nil {
						t.Fatal(err)
					}
				}
				want := map[string][]string{"CN": {"1.0.0.0/24"}, "US": {"1.0.1.0/24"}, "JP": {"1.0.2.0/24"}}
				if wanted != nil {
					delete(want, "US")
				}
				if got := containerPrefixes(t, container); !reflect.DeepEqual(got, want) {
					t.Fatalf("MMDB resolution = %#v; want %#v", got, want)
				}
			}
		})
	}
}

func TestMMDBOutputPriority(t *testing.T) {
	container := lib.NewContainer()
	for _, name := range []string{"CN", "US", "JP"} {
		entry := lib.NewEntry(name)
		if err := entry.AddPrefix("1.0.0.0/24"); err != nil {
			t.Fatal(err)
		}
		if err := container.Add(entry); err != nil {
			t.Fatal(err)
		}
	}
	for _, converter := range outputConstructors {
		for _, tc := range []struct {
			name      string
			wanted    []string
			overwrite []string
			excluded  []string
			order     []string
		}{
			{"wanted-priority", []string{" us ", "cn"}, []string{"us"}, nil, []string{"US", "CN"}},
			{"overwrite-fallback", nil, []string{" us ", "cn"}, nil, []string{"JP", "US", "CN"}},
			{"blank-wanted-fallback", []string{" "}, []string{"cn"}, nil, []string{"JP", "US", "CN"}},
			{"exclude-overwrite", nil, []string{"us", "cn"}, []string{" cn "}, []string{"JP", "US"}},
			{"exclude-wanted", []string{"us", "cn"}, []string{"cn"}, []string{"cn"}, []string{"US"}},
			{"all-wanted-excluded", []string{"cn"}, []string{"us"}, []string{"cn"}, []string{}},
		} {
			t.Run(converter.name+"/"+tc.name, func(t *testing.T) {
				dir := testDirectory(t)
				g := converter.new(lib.ActionOutput, WithOutputDir(dir), WithOutputWantedList(tc.wanted),
					WithOutputOverwriteList(tc.overwrite), WithOutputExcludedList(tc.excluded)).(*geoLite2CountryMMDBOut)
				if got := g.filterAndSortList(container); !slices.Equal(got, tc.order) {
					t.Fatalf("write order = %v; want %v", got, tc.order)
				}
				if err := g.Output(container); err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(dir, "Country.mmdb")
				if len(tc.order) == 0 {
					if _, err := os.Stat(path); !os.IsNotExist(err) {
						t.Fatalf("excluded wanted list produced output: %v", err)
					}
					return
				}
				db, err := maxminddb.Open(path)
				if err != nil {
					t.Fatal(err)
				}
				defer db.Close()
				var record struct {
					Country struct {
						ISOCode string `maxminddb:"iso_code"`
					} `maxminddb:"country"`
					CountryCode string `maxminddb:"country_code"`
				}
				if err := db.Lookup(netip.MustParseAddr("1.0.0.1")).Decode(&record); err != nil {
					t.Fatal(err)
				}
				got := record.Country.ISOCode
				if converter.name == TypeIPInfoCountryMMDBOut {
					got = record.CountryCode
				}
				if want := tc.order[len(tc.order)-1]; got != want {
					t.Fatalf("overlapping prefix country = %q; want %q", got, want)
				}
			})
		}
	}
}
