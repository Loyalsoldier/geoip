package maxmind

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/oschwald/maxminddb-golang/v2"
)

var (
	geoLiteLanguages = []string{"de", "en", "es", "fr", "ja", "pt-BR", "ru", "zh-CN"}
	dbipLanguages    = []string{"de", "en", "es", "fr", "ja", "pt-BR", "ru", "zh-CN", "fa", "ko"}
)

// Read the actual file with an independent reader, not the plugin's input path.
func openOutput(t *testing.T, g *geolite2_country_mmdb_out) *maxminddb.Reader {
	t.Helper()
	db, err := maxminddb.Open(filepath.Join(g.OutputDir, g.OutputName))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Verify(); err != nil {
		t.Fatalf("invalid output database: %v", err)
	}
	return db
}

func assertDatabaseRecords(t *testing.T, db *maxminddb.Reader, want map[string]mmdbtype.Map) {
	t.Helper()
	var gotCIDRs []string
	for network := range db.Networks() {
		var record mmdbtype.Unmarshaler
		if err := network.Decode(&record); err != nil {
			t.Fatal(err)
		}
		cidr := network.Prefix().String()
		gotCIDRs = append(gotCIDRs, cidr)
		if got := record.Result(); !reflect.DeepEqual(got, want[cidr]) {
			t.Errorf("record for %s = %v, want %v", cidr, got, want[cidr])
		}
	}
	assertCIDRs(t, db.Metadata.DatabaseType, gotCIDRs, slices.Collect(maps.Keys(want)))
}

func TestMMDBOutput(t *testing.T) {
	c := containerWith(t, map[string][]string{
		"DE": {"192.0.2.0/24", "2001:db8::/32"}, "JP": {"192.0.2.0/25"}, "ZZ": {"198.51.100.0/24"},
	})
	for _, tt := range []struct {
		typ, databaseType, description string
		recordSize                     uint
		languages                      []string
	}{
		{TypeGeoLite2CountryMMDBOut, "GeoLite2-Country", "Customized GeoLite2 Country database", 24, geoLiteLanguages},
		{TypeDBIPCountryMMDBOut, "DBIP-Country-Lite", "Customized DB-IP Country Lite database", 24, dbipLanguages},
		{TypeIPInfoCountryMMDBOut, "IPInfo-Lite", "Customized IPInfo Lite database", 32, []string{"en"}},
	} {
		t.Run(tt.typ, func(t *testing.T) {
			g := geolite2_country_mmdb_out{
				Type: tt.typ, OutputDir: filepath.Join(t.TempDir(), "output"), OutputName: "result.mmdb",
				Want: []string{" de ", "missing", "jp", "ZZ"}, Exclude: []string{"zz"}, Overwrite: []string{"DE"},
			}
			if err := g.Output(c); err != nil {
				t.Fatal(err)
			}
			db := openOutput(t, &g)
			m := db.Metadata
			if m.DatabaseType != tt.databaseType || m.Description["en"] != tt.description || m.RecordSize != tt.recordSize || !slices.Equal(m.Languages, tt.languages) {
				t.Errorf("unexpected metadata: %+v", m)
			}
			de, jp := mmdbtype.Map{"country": countryCode("DE")}, mmdbtype.Map{"country": countryCode("JP")}
			if tt.typ == TypeIPInfoCountryMMDBOut {
				de, jp = mmdbtype.Map{"country_code": mmdbtype.String("DE")}, mmdbtype.Map{"country_code": mmdbtype.String("JP")}
			}
			assertDatabaseRecords(t, db, map[string]mmdbtype.Map{
				"192.0.2.0/25": jp, "192.0.2.128/25": de, "2001:db8::/32": de,
			})
		})
	}
}

func TestMMDBListOrder(t *testing.T) {
	c := containerWith(t, map[string][]string{"JP": nil, "DE": nil, "CA": nil, "US": nil})
	for _, tt := range []struct {
		name                       string
		wanted, overwrite, exclude []string
		want                       []string
	}{
		{"alphabetical by default", nil, nil, nil, []string{"CA", "DE", "JP", "US"}},
		{"overwrite last in specified order", []string{" "}, []string{" us", " ", "CA ", "de"}, []string{" de "}, []string{"JP", "US", "CA"}},
		{"wanted before overwrite", []string{" JP ", "ca", "de"}, []string{"ca", "jp"}, []string{"DE"}, []string{"JP", "CA"}},
		{"all wanted excluded", []string{"DE"}, []string{"JP"}, []string{"De"}, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := geolite2_country_mmdb_out{Want: tt.wanted, Overwrite: tt.overwrite, Exclude: tt.exclude}
			if got := g.filterAndSortList(c); !slices.Equal(got, tt.want) {
				t.Errorf("order = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMMDBOutputNoMatchingEntries(t *testing.T) {
	g := geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, OutputDir: t.TempDir(), OutputName: "result.mmdb", Want: []string{"missing"}}
	if err := g.Output(containerWith(t, map[string][]string{"DE": {"192.0.2.0/24"}})); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(g.OutputDir, g.OutputName)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected no output file, stat returned %v", err)
	}
}

func TestMMDBOutputErrors(t *testing.T) {
	c := containerWith(t, map[string][]string{"DE": {"192.0.2.0/24"}})
	t.Run("source unavailable", func(t *testing.T) {
		dir := t.TempDir()
		g := geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: filepath.Join(dir, "missing"), OutputDir: dir, OutputName: "out.mmdb"}
		if err := g.Output(c); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("got %v, want file not found", err)
		}
	})
	t.Run("no prefixes in requested family", func(t *testing.T) {
		g := geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, OnlyIPType: lib.IPv6, OutputDir: t.TempDir(), OutputName: "out.mmdb"}
		if err := g.Output(c); err == nil || !strings.Contains(err.Error(), "has no prefix") {
			t.Errorf("got %v, want no prefix error", err)
		}
	})
	t.Run("unsupported format", func(t *testing.T) {
		g := geolite2_country_mmdb_out{Type: "unknown", OutputDir: t.TempDir(), OutputName: "out.mmdb"}
		if err := g.Output(c); !errors.Is(err, lib.ErrNotSupportedFormat) {
			t.Errorf("got %v, want ErrNotSupportedFormat", err)
		}
	})
	t.Run("cannot create output directory", func(t *testing.T) {
		file := writeFixture(t, "not-a-directory", nil)
		g := geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, OutputDir: filepath.Join(file, "subdir"), OutputName: "out.mmdb"}
		if err := g.Output(c); !errors.As(err, new(*os.PathError)) {
			t.Errorf("got %v, want path error", err)
		}
	})
	t.Run("cannot open output file", func(t *testing.T) {
		dir := t.TempDir()
		g := geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, OutputDir: filepath.Dir(dir), OutputName: filepath.Base(dir)}
		if err := g.Output(c); !errors.As(err, new(*os.PathError)) {
			t.Errorf("got %v, want path error", err)
		}
	})
	t.Run("writer cannot serialize tree", func(t *testing.T) {
		tree, err := mmdbwriter.New(mmdbwriter.Options{RecordSize: 16}) // not a supported on-disk record size
		if err != nil {
			t.Fatal(err)
		}
		g := geolite2_country_mmdb_out{OutputDir: t.TempDir()}
		if err := g.writeFile("unsupported.mmdb", tree); err == nil || !strings.Contains(err.Error(), "record size") {
			t.Errorf("got %v, want serialization error", err)
		}
	})
	t.Run("IPv6 prefix inside an IPv4 alias", func(t *testing.T) {
		g := geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, OutputDir: t.TempDir(), OutputName: "out.mmdb"}
		c := containerWith(t, map[string][]string{"DE": {"2002:abcd::/32"}})
		if err := g.Output(c); !errors.As(err, new(*mmdbwriter.AliasedNetworkError)) {
			t.Errorf("got %v, want aliased network error", err)
		}
	})
}
