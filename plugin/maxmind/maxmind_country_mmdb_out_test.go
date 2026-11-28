package maxmind

import (
	"bytes"
	"errors"
	"io/fs"
	"maps"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/oschwald/geoip2-golang/v2"
	"github.com/oschwald/maxminddb-golang/v2"
	"go4.org/netipx"
)

// isoCodeRecord returns the data that GeoLite2 and DB-IP databases hold for
// the networks of a list without extra info.
func isoCodeRecord(list string) mmdbtype.Map {
	return mmdbtype.Map{"country": mmdbtype.Map{"iso_code": mmdbtype.String(list)}}
}

// countryCodeRecord returns the data that IPInfo databases hold for the
// networks of a list without extra info.
func countryCodeRecord(list string) mmdbtype.Map {
	return mmdbtype.Map{"country_code": mmdbtype.String(list)}
}

// ipinfoListRecord returns the data that IPInfo databases hold for the
// networks of a list that is enriched with a place. It tells no AS, as the
// networks of a list don't share one.
func ipinfoListRecord(place mmdbtype.Map) mmdbtype.Map {
	data := maps.Clone(place)
	data["asn"] = mmdbtype.String("")
	data["as_name"] = mmdbtype.String("")
	data["as_domain"] = mmdbtype.String("")
	return data
}

// geoLite2Names returns the counterpart of localizedNames in the languages of
// GeoLite2 databases, in the form of extra info.
func geoLite2Names(place string) geoip2.Names {
	return geoip2.Names{
		German:              place + " (de)",
		English:             place + " (en)",
		Spanish:             place + " (es)",
		French:              place + " (fr)",
		Japanese:            place + " (ja)",
		BrazilianPortuguese: place + " (pt-BR)",
		Russian:             place + " (ru)",
		SimplifiedChinese:   place + " (zh-CN)",
	}
}

// dbipNames returns the counterpart of localizedNames in the languages of
// DB-IP databases, in the form of extra info.
func dbipNames(place string) dbipLanguageNames {
	return dbipLanguageNames{
		German:              place + " (de)",
		English:             place + " (en)",
		Spanish:             place + " (es)",
		Persian:             place + " (fa)",
		French:              place + " (fr)",
		Japanese:            place + " (ja)",
		Korean:              place + " (ko)",
		BrazilianPortuguese: place + " (pt-BR)",
		Russian:             place + " (ru)",
		SimplifiedChinese:   place + " (zh-CN)",
	}
}

func TestGeoLite2CountryMMDBOut_filterAndSortList(t *testing.T) {
	container := newTestContainer(t, map[string][]string{
		"US":      {"8.8.8.0/24"},
		"CN":      {"1.0.1.0/24"},
		"PRIVATE": {"10.0.0.0/8"},
		"JP":      {"1.0.16.0/20"},
	})

	tests := []struct {
		name      string
		wanted    []string
		overwrite []string
		excluded  []string
		want      []string
	}{
		{name: "all lists in alphabetical order", want: []string{"CN", "JP", "PRIVATE", "US"}},
		{name: "excluded lists", excluded: []string{" us ", "", "Private", "missing"}, want: []string{"CN", "JP"}},
		{name: "all lists are excluded", excluded: []string{"cn", "jp", "private", "us"}, want: []string{}},

		{name: "wanted lists keep their order", wanted: []string{"us", " cn ", "", "Jp"}, want: []string{"US", "CN", "JP"}},
		{name: "wanted lists may not exist", wanted: []string{"missing", "cn"}, want: []string{"MISSING", "CN"}},
		{name: "wanted lists but the excluded ones", wanted: []string{"us", "cn", "jp"}, excluded: []string{"cn"}, want: []string{"US", "JP"}},
		{name: "all wanted lists are excluded", wanted: []string{"cn", "us"}, excluded: []string{"US", "CN"}, want: []string{}},
		{name: "blank wanted lists want all lists", wanted: []string{"", "  "}, want: []string{"CN", "JP", "PRIVATE", "US"}},

		{name: "overwrite lists come last in their order", overwrite: []string{"us", " cn ", ""}, want: []string{"JP", "PRIVATE", "US", "CN"}},
		{name: "overwrite lists may not exist", overwrite: []string{"missing", "cn"}, want: []string{"JP", "PRIVATE", "US", "MISSING", "CN"}},
		{name: "overwrite lists but the excluded ones", overwrite: []string{"us", "cn"}, excluded: []string{"us", "jp"}, want: []string{"PRIVATE", "CN"}},
		{name: "wanted lists take precedence over overwrite lists", wanted: []string{"cn", "us"}, overwrite: []string{"cn"}, want: []string{"CN", "US"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_country_mmdb_out{Want: tt.wanted, Overwrite: tt.overwrite, Exclude: tt.excluded}

			got := g.filterAndSortList(container)
			if len(got) != len(tt.want) || !slices.Equal(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGeoLite2CountryMMDBOut_marshalData(t *testing.T) {
	const source = "./source.mmdb"

	// Extra info of each format, as GetExtraInfo provides it.
	geoLite2Country := geoip2.CountryRecord{ISOCode: "DE", GeoNameID: 2921044, IsInEuropeanUnion: true, Names: geoLite2Names("Germany")}
	geoLite2Continent := geoip2.Continent{Code: "EU", GeoNameID: 6255148, Names: geoLite2Names("Europe")}
	dbipCountryInfo := dbipCountryRecord{ISOCode: "DE", GeoNameID: 2921044, IsInEuropeanUnion: true, Names: dbipNames("Germany")}
	dbipContinentInfo := dbipContinent{Code: "EU", GeoNameID: 6255148, Names: dbipNames("Europe")}
	ipinfoInfo := ipInfoLite{Continent: "Europe", ContinentCode: "EU", Country: "Germany", CountryCode: "DE"}

	// The data that the extra info above is expected to be turned into.
	wantGeoLite2Country := countryData("DE", 2921044, true, localizedNames("Germany", geoLite2Languages))
	wantGeoLite2Continent := continentData("EU", 6255148, localizedNames("Europe", geoLite2Languages))
	wantDBIPCountry := countryData("DE", 2921044, true, localizedNames("Germany", dbipLanguages))
	wantDBIPContinent := continentData("EU", 6255148, localizedNames("Europe", dbipLanguages))
	wantIPInfo := ipinfoListRecord(ipinfoPlace("EU", "Europe", "DE", "Germany"))

	germany := []string{"5.1.0.0/16", "2a00::/22"}

	tests := []struct {
		name string
		out  geolite2_country_mmdb_out
		// cidrs are the networks of the list named DE, which default to germany.
		cidrs     []string
		extraInfo map[string]any
		// wantNetworks default to germany.
		wantNetworks []string
		// want is the data expected for each of the networks.
		want    mmdbtype.DataType
		wantErr *errCheck
	}{
		{
			name: "geolite2 without source",
			out:  geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut},
			want: isoCodeRecord("DE"),
		},
		{
			name: "dbip without source",
			out:  geolite2_country_mmdb_out{Type: TypeDBIPCountryMMDBOut},
			want: isoCodeRecord("DE"),
		},
		{
			name: "ipinfo without source",
			out:  geolite2_country_mmdb_out{Type: TypeIPInfoCountryMMDBOut},
			want: countryCodeRecord("DE"),
		},
		{
			name:      "extra info is ignored without source",
			out:       geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: "  "},
			extraInfo: map[string]any{"DE": geoip2.Country{Continent: geoLite2Continent, Country: geoLite2Country}},
			want:      isoCodeRecord("DE"),
		},
		{
			name:    "unknown format without source",
			out:     geolite2_country_mmdb_out{Type: "unknown"},
			wantErr: errIs(lib.ErrNotSupportedFormat),
		},

		{
			name:      "geolite2 with continent and country",
			out:       geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: source},
			extraInfo: map[string]any{"DE": geoip2.Country{Continent: geoLite2Continent, Country: geoLite2Country}},
			want:      mmdbtype.Map{"continent": wantGeoLite2Continent, "country": wantGeoLite2Country},
		},
		{
			name:      "geolite2 with country only",
			out:       geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: source},
			extraInfo: map[string]any{"DE": geoip2.Country{Country: geoLite2Country}},
			want:      mmdbtype.Map{"country": wantGeoLite2Country},
		},
		{
			name: "geolite2 country code is the name of the list",
			out:  geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: source},
			extraInfo: map[string]any{"DE": geoip2.Country{Country: geoip2.CountryRecord{
				ISOCode: " de ", GeoNameID: 2921044, IsInEuropeanUnion: true, Names: geoLite2Names("Germany"),
			}}},
			want: mmdbtype.Map{"country": wantGeoLite2Country},
		},
		{
			name:      "geolite2 without extra info of the list",
			out:       geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: source},
			extraInfo: map[string]any{"FR": geoip2.Country{Continent: geoLite2Continent, Country: geoLite2Country}},
			want:      isoCodeRecord("DE"),
		},
		{
			name: "geolite2 without any extra info",
			out:  geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: source},
			want: isoCodeRecord("DE"),
		},
		{
			name:      "geolite2 with extra info of another format",
			out:       geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: source},
			extraInfo: map[string]any{"DE": dbipCountry{Continent: dbipContinentInfo, Country: dbipCountryInfo}},
			want:      isoCodeRecord("DE"),
		},

		{
			name:      "dbip with continent and country",
			out:       geolite2_country_mmdb_out{Type: TypeDBIPCountryMMDBOut, SourceMMDBURI: source},
			extraInfo: map[string]any{"DE": dbipCountry{Continent: dbipContinentInfo, Country: dbipCountryInfo}},
			want:      mmdbtype.Map{"continent": wantDBIPContinent, "country": wantDBIPCountry},
		},
		{
			name:      "dbip with country only",
			out:       geolite2_country_mmdb_out{Type: TypeDBIPCountryMMDBOut, SourceMMDBURI: source},
			extraInfo: map[string]any{"DE": dbipCountry{Country: dbipCountryInfo}},
			want:      mmdbtype.Map{"country": wantDBIPCountry},
		},
		{
			name:      "dbip without extra info of the list",
			out:       geolite2_country_mmdb_out{Type: TypeDBIPCountryMMDBOut, SourceMMDBURI: source},
			extraInfo: map[string]any{"FR": dbipCountry{Continent: dbipContinentInfo, Country: dbipCountryInfo}},
			want:      isoCodeRecord("DE"),
		},
		{
			name:      "dbip with extra info of another format",
			out:       geolite2_country_mmdb_out{Type: TypeDBIPCountryMMDBOut, SourceMMDBURI: source},
			extraInfo: map[string]any{"DE": geoip2.Country{Continent: geoLite2Continent, Country: geoLite2Country}},
			want:      isoCodeRecord("DE"),
		},

		{
			name:      "ipinfo with place",
			out:       geolite2_country_mmdb_out{Type: TypeIPInfoCountryMMDBOut, SourceMMDBURI: source},
			extraInfo: map[string]any{"DE": ipinfoInfo},
			want:      wantIPInfo,
		},
		{
			name: "ipinfo country code is the name of the list",
			out:  geolite2_country_mmdb_out{Type: TypeIPInfoCountryMMDBOut, SourceMMDBURI: source},
			extraInfo: map[string]any{"DE": ipInfoLite{
				Continent: "Europe", ContinentCode: "EU", Country: "Germany", CountryCode: " de ",
			}},
			want: wantIPInfo,
		},
		{
			name:      "ipinfo without extra info of the list",
			out:       geolite2_country_mmdb_out{Type: TypeIPInfoCountryMMDBOut, SourceMMDBURI: source},
			extraInfo: map[string]any{"FR": ipinfoInfo},
			want:      countryCodeRecord("DE"),
		},
		{
			name:      "ipinfo with extra info of another format",
			out:       geolite2_country_mmdb_out{Type: TypeIPInfoCountryMMDBOut, SourceMMDBURI: source},
			extraInfo: map[string]any{"DE": geoip2.Country{Continent: geoLite2Continent, Country: geoLite2Country}},
			want:      countryCodeRecord("DE"),
		},
		{
			name:      "unknown format with source",
			out:       geolite2_country_mmdb_out{Type: "unknown", SourceMMDBURI: source},
			extraInfo: map[string]any{"DE": ipinfoInfo},
			wantErr:   errIs(lib.ErrNotSupportedFormat),
		},

		{
			name:         "ipv4 only",
			out:          geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, OnlyIPType: lib.IPv4},
			wantNetworks: []string{"5.1.0.0/16"},
			want:         isoCodeRecord("DE"),
		},
		{
			name:         "ipv6 only",
			out:          geolite2_country_mmdb_out{Type: TypeIPInfoCountryMMDBOut, OnlyIPType: lib.IPv6},
			wantNetworks: []string{"2a00::/22"},
			want:         countryCodeRecord("DE"),
		},
		{
			name:         "reserved networks",
			out:          geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut},
			cidrs:        []string{"10.0.0.0/8", "192.168.0.0/16", "fc00::/7"},
			wantNetworks: []string{"10.0.0.0/8", "192.168.0.0/16", "fc00::/7"},
			want:         isoCodeRecord("DE"),
		},
		{
			name:    "list without networks of the only ip type",
			out:     geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, OnlyIPType: lib.IPv6},
			cidrs:   []string{"5.1.0.0/16"},
			wantErr: errContains("entry DE has no prefix"),
		},
		{
			name:    "list without networks",
			out:     geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut},
			cidrs:   []string{},
			wantErr: errContains("entry DE has no prefix"),
		},
		{
			// 6to4 addresses are aliases of the IPv4 addresses they embed.
			name:    "network aliased to ipv4",
			out:     geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut},
			cidrs:   []string{"5.1.0.0/16", "2002:501::/32"},
			wantErr: errAs[*mmdbwriter.AliasedNetworkError](),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.out.Action = lib.ActionOutput
			cidrs := tt.cidrs
			if cidrs == nil {
				cidrs = germany
			}
			wantNetworks := tt.wantNetworks
			if wantNetworks == nil {
				wantNetworks = germany
			}
			writer := newMMDBWriter(t, 6)

			err := tt.out.marshalData(writer, newTestEntry(t, "de", cidrs...), tt.extraInfo)
			if assertErr(t, err, tt.wantErr) {
				return
			}

			want := make([]mmdbRecord, 0, len(wantNetworks))
			for _, network := range wantNetworks {
				want = append(want, mmdbRecord{network, tt.want})
			}
			_, got := readMMDB(t, writeMMDB(t, writer))
			assertMMDBRecords(t, got, want)
		})
	}
}

func TestGeoLite2CountryMMDBOut_writeFile(t *testing.T) {
	const filename = "Country.mmdb"

	// newWriter returns a writer that holds a single record.
	newWriter := func(t *testing.T, opts mmdbwriter.Options) *mmdbwriter.Tree {
		t.Helper()
		writer, err := mmdbwriter.New(opts)
		if err != nil {
			t.Fatalf("mmdbwriter.New(): %v", err)
		}
		if err := writer.Insert(netipx.PrefixIPNet(netip.MustParsePrefix("1.0.1.0/24")), isoCodeRecord("CN")); err != nil {
			t.Fatalf("Insert(): %v", err)
		}
		return writer
	}
	validOptions := mmdbwriter.Options{DatabaseType: "Test", Description: map[string]string{"en": "Test database"}}
	// The writer accepts a record size that it can't write.
	invalidOptions := mmdbwriter.Options{RecordSize: 16}

	tests := []struct {
		name string
		// outputDir returns the output directory, which is located in dir.
		outputDir func(t *testing.T, dir string) string
		options   mmdbwriter.Options
		// wantErr returns the error expected for the given output directory.
		wantErr func(outputDir string) *errCheck
	}{
		{
			name:      "existing directory",
			outputDir: func(t *testing.T, dir string) string { return dir },
			options:   validOptions,
		},
		{
			name:      "missing directories are created",
			outputDir: func(t *testing.T, dir string) string { return filepath.Join(dir, "output", "maxmind") },
			options:   validOptions,
		},
		{
			name: "existing file is replaced",
			outputDir: func(t *testing.T, dir string) string {
				writeTestFile(t, dir, filename, bytes.Repeat([]byte("stale"), 1<<16))
				return dir
			},
			options: validOptions,
		},
		{
			name: "output directory is a file",
			outputDir: func(t *testing.T, dir string) string {
				return writeTestFile(t, dir, "file", nil)
			},
			options: validOptions,
			wantErr: func(outputDir string) *errCheck { return errPath("mkdir", outputDir) },
		},
		{
			name: "output file is a directory",
			outputDir: func(t *testing.T, dir string) string {
				if err := os.Mkdir(filepath.Join(dir, filename), 0755); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			options: validOptions,
			wantErr: func(outputDir string) *errCheck { return errPath("open", filepath.Join(outputDir, filename)) },
		},
		{
			name:      "database can't be written",
			outputDir: func(t *testing.T, dir string) string { return dir },
			options:   invalidOptions,
			wantErr:   func(string) *errCheck { return errAny() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputDir := tt.outputDir(t, t.TempDir())
			g := &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, Action: lib.ActionOutput, OutputDir: outputDir}
			writer := newWriter(t, tt.options)

			err := g.writeFile(filename, writer)
			if tt.wantErr != nil {
				assertErr(t, err, tt.wantErr(outputDir))
				return
			}
			assertErr(t, err, nil)

			got, err := os.ReadFile(filepath.Join(outputDir, filename))
			if err != nil {
				t.Fatal(err)
			}
			// Databases are written reproducibly, so that the file is to hold
			// exactly what the writer writes again, without anything stale.
			if want := writeMMDB(t, writer); !bytes.Equal(got, want) {
				t.Fatalf("the file holds %d bytes that differ from the %d bytes of the database", len(got), len(want))
			}
			_, records := readMMDB(t, got)
			assertMMDBRecords(t, records, []mmdbRecord{{"1.0.1.0/24", isoCodeRecord("CN")}})
		})
	}
}

// sourcePlaces are the continents and countries that the source MMDB of
// TestGeoLite2CountryMMDBOut_Output tells about, in the format shared by
// GeoLite2 and DB-IP country databases.
type sourcePlaces struct {
	asia, europe        mmdbtype.Map
	china, germany, usa mmdbtype.Map
}

// newSourcePlaces returns places whose localized names are given by names.
func newSourcePlaces(names func(place string) mmdbtype.Map) sourcePlaces {
	return sourcePlaces{
		asia:    continentData("AS", 6255147, names("Asia")),
		europe:  continentData("EU", 6255148, names("Europe")),
		china:   countryData("CN", 1814991, false, names("China")),
		germany: countryData("DE", 2921044, true, names("Germany")),
		usa:     countryData("US", 6252001, false, names("United States")),
	}
}

// source returns the records of a source MMDB.
func (p sourcePlaces) source() []mmdbRecord {
	return []mmdbRecord{
		{"1.0.1.0/24", mmdbtype.Map{"continent": p.asia, "country": p.china, "registered_country": p.usa}},
		// Only the first network of a country provides extra info.
		{"1.0.2.0/23", mmdbtype.Map{"continent": p.europe, "country": countryData("CN", 1, true, nil)}},
		// The registered country is used in the absence of a country.
		{"5.1.0.0/16", mmdbtype.Map{"continent": p.europe, "registered_country": p.germany, "represented_country": representedCountryData(p.usa)}},
		// The represented country is the last resort.
		{"6.0.0.0/8", mmdbtype.Map{"represented_country": representedCountryData(p.usa)}},
		// A network without any country provides no extra info.
		{"7.0.0.0/8", mmdbtype.Map{"continent": p.europe}},
	}
}

// enrichedLists are the lists to output along with the extra info of a source MMDB.
var enrichedLists = map[string][]string{
	"CN":      {"1.0.1.0/24", "2001:250::/35"},
	"DE":      {"5.1.0.0/16"},
	"US":      {"8.8.8.0/24"},
	"PRIVATE": {"10.0.0.0/8"},
}

// enriched returns the records expected for enrichedLists, given the extra
// info of the records of source.
func (p sourcePlaces) enriched() []mmdbRecord {
	return []mmdbRecord{
		{"1.0.1.0/24", mmdbtype.Map{"continent": p.asia, "country": p.china}},
		{"5.1.0.0/16", mmdbtype.Map{"continent": p.europe, "country": p.germany}},
		// The network of the represented country tells no continent.
		{"8.8.8.0/24", mmdbtype.Map{"country": p.usa}},
		// The source tells nothing about this list.
		{"10.0.0.0/8", isoCodeRecord("PRIVATE")},
		{"2001:250::/35", mmdbtype.Map{"continent": p.asia, "country": p.china}},
	}
}

func TestGeoLite2CountryMMDBOut_Output(t *testing.T) {
	const (
		geoLite2 = TypeGeoLite2CountryMMDBOut
		dbip     = TypeDBIPCountryMMDBOut
		ipinfo   = TypeIPInfoCountryMMDBOut
	)

	// The lists to output by default.
	lists := map[string][]string{
		"CN":      {"1.0.1.0/24", "1.0.2.0/23", "2001:250::/35"},
		"US":      {"8.8.8.0/24", "2001:4860::/32"},
		"PRIVATE": {"10.0.0.0/8", "fc00::/7"},
	}
	// allRecords returns the records of all the lists above, given the data
	// of the networks of a list.
	allRecords := func(data func(list string) mmdbtype.Map) []mmdbRecord {
		return []mmdbRecord{
			{"1.0.1.0/24", data("CN")},
			{"1.0.2.0/23", data("CN")},
			{"8.8.8.0/24", data("US")},
			{"10.0.0.0/8", data("PRIVATE")},
			{"2001:250::/35", data("CN")},
			{"2001:4860::/32", data("US")},
			{"fc00::/7", data("PRIVATE")},
		}
	}
	countryRecords := []mmdbRecord{
		{"1.0.1.0/24", isoCodeRecord("CN")},
		{"1.0.2.0/23", isoCodeRecord("CN")},
		{"8.8.8.0/24", isoCodeRecord("US")},
		{"2001:250::/35", isoCodeRecord("CN")},
		{"2001:4860::/32", isoCodeRecord("US")},
	}

	// Lists whose networks overlap: the list that is written last gets them.
	overlappingLists := map[string][]string{"CN": {"1.1.0.0/22"}, "CLOUDFLARE": {"1.1.1.0/24"}}
	cloudflareOverChina := []mmdbRecord{
		{"1.1.0.0/24", isoCodeRecord("CN")},
		{"1.1.1.0/24", isoCodeRecord("CLOUDFLARE")},
		{"1.1.2.0/23", isoCodeRecord("CN")},
	}
	chinaOverCloudflare := []mmdbRecord{{"1.1.0.0/22", isoCodeRecord("CN")}}

	// Localized names of the places of source MMDBs and of generated MMDBs.
	localized := func(languages []string) func(string) mmdbtype.Map {
		return func(place string) mmdbtype.Map { return localizedNames(place, languages) }
	}
	blank := func(languages []string) func(string) mmdbtype.Map {
		return func(string) mmdbtype.Map { return blankNames(languages) }
	}
	unnamed := func(string) mmdbtype.Map { return nil }

	ipinfoChina := ipinfoPlace("AS", "Asia", "CN", "China")
	ipinfoGermany := ipinfoPlace("EU", "Europe", "DE", "Germany")
	ipinfoSource := []mmdbRecord{
		{"1.0.1.0/24", ipinfoData(ipinfoChina, "AS4134")},
		// Only the first network of a country provides extra info.
		{"1.0.2.0/23", ipinfoData(ipinfoPlace("EU", "Europe", "CN", "Not China"), "AS4837")},
		{"5.1.0.0/16", ipinfoData(ipinfoGermany, "AS3320")},
		// Country codes are trimmed and upper-cased.
		{"6.0.0.0/8", ipinfoData(ipinfoPlace("NA", "North America", " us ", "United States"), "AS668")},
	}
	ipinfoEnriched := []mmdbRecord{
		{"1.0.1.0/24", ipinfoListRecord(ipinfoChina)},
		{"5.1.0.0/16", ipinfoListRecord(ipinfoGermany)},
		{"8.8.8.0/24", ipinfoListRecord(ipinfoPlace("NA", "North America", "US", "United States"))},
		// The source tells nothing about this list.
		{"10.0.0.0/8", countryCodeRecord("PRIVATE")},
		{"2001:250::/35", ipinfoListRecord(ipinfoChina)},
	}

	dir := t.TempDir()
	occupiedFile := writeTestFile(t, dir, "file", nil)
	occupiedName := "directory.mmdb"
	if err := os.Mkdir(filepath.Join(dir, occupiedName), 0755); err != nil {
		t.Fatal(err)
	}
	missingFile := filepath.Join(dir, "missing.mmdb")

	tests := []struct {
		name string
		// out is the output to run. Its output directory defaults to a new one
		// and its output name to Country.mmdb.
		out geolite2_country_mmdb_out
		// source holds the records of the source MMDB, if there is to be one.
		source []mmdbRecord
		// lists are the lists to output, which default to the ones above.
		lists map[string][]string
		// want holds the records of the generated MMDB, if there is to be one.
		want    []mmdbRecord
		wantErr *errCheck
	}{
		{name: "geolite2", out: geolite2_country_mmdb_out{Type: geoLite2}, want: allRecords(isoCodeRecord)},
		{name: "dbip", out: geolite2_country_mmdb_out{Type: dbip}, want: allRecords(isoCodeRecord)},
		{name: "ipinfo", out: geolite2_country_mmdb_out{Type: ipinfo}, want: allRecords(countryCodeRecord)},
		{name: "blank source", out: geolite2_country_mmdb_out{Type: geoLite2, SourceMMDBURI: "  "}, want: allRecords(isoCodeRecord)},

		{
			name: "wanted lists",
			out:  geolite2_country_mmdb_out{Type: geoLite2, Want: []string{"us", "missing", " cn "}},
			want: countryRecords,
		},
		{
			name: "excluded lists",
			out:  geolite2_country_mmdb_out{Type: geoLite2, Exclude: []string{"private"}},
			want: countryRecords,
		},
		{
			name: "ipv4 only",
			out:  geolite2_country_mmdb_out{Type: geoLite2, OnlyIPType: lib.IPv4},
			want: []mmdbRecord{
				{"1.0.1.0/24", isoCodeRecord("CN")},
				{"1.0.2.0/23", isoCodeRecord("CN")},
				{"8.8.8.0/24", isoCodeRecord("US")},
				{"10.0.0.0/8", isoCodeRecord("PRIVATE")},
			},
		},
		{
			name: "ipv6 only",
			out:  geolite2_country_mmdb_out{Type: ipinfo, OnlyIPType: lib.IPv6},
			want: []mmdbRecord{
				{"2001:250::/35", countryCodeRecord("CN")},
				{"2001:4860::/32", countryCodeRecord("US")},
				{"fc00::/7", countryCodeRecord("PRIVATE")},
			},
		},
		{
			name: "no list to output",
			out:  geolite2_country_mmdb_out{Type: geoLite2, Want: []string{"missing"}},
		},
		{
			name:  "no list at all",
			out:   geolite2_country_mmdb_out{Type: geoLite2},
			lists: map[string][]string{},
		},

		{
			name:  "overlapping networks go to the last list in alphabetical order",
			out:   geolite2_country_mmdb_out{Type: geoLite2},
			lists: overlappingLists,
			want:  chinaOverCloudflare,
		},
		{
			name:  "overlapping networks go to the overwrite list",
			out:   geolite2_country_mmdb_out{Type: geoLite2, Overwrite: []string{"cloudflare"}},
			lists: overlappingLists,
			want:  cloudflareOverChina,
		},
		{
			name:  "overlapping networks go to the last wanted list",
			out:   geolite2_country_mmdb_out{Type: geoLite2, Want: []string{"cn", "cloudflare"}},
			lists: overlappingLists,
			want:  cloudflareOverChina,
		},
		{
			name:  "wanted lists take precedence over overwrite lists",
			out:   geolite2_country_mmdb_out{Type: geoLite2, Want: []string{"cloudflare", "cn"}, Overwrite: []string{"cloudflare"}},
			lists: overlappingLists,
			want:  chinaOverCloudflare,
		},

		{
			name:   "geolite2 enriched with localized names",
			out:    geolite2_country_mmdb_out{Type: geoLite2},
			source: newSourcePlaces(localized(geoLite2Languages)).source(),
			lists:  enrichedLists,
			want:   newSourcePlaces(localized(geoLite2Languages)).enriched(),
		},
		{
			name:   "geolite2 enriched without localized names",
			out:    geolite2_country_mmdb_out{Type: geoLite2},
			source: newSourcePlaces(unnamed).source(),
			lists:  enrichedLists,
			want:   newSourcePlaces(blank(geoLite2Languages)).enriched(),
		},
		{
			name:   "dbip enriched with localized names",
			out:    geolite2_country_mmdb_out{Type: dbip},
			source: newSourcePlaces(localized(dbipLanguages)).source(),
			lists:  enrichedLists,
			want:   newSourcePlaces(localized(dbipLanguages)).enriched(),
		},
		{
			name:   "dbip enriched without localized names",
			out:    geolite2_country_mmdb_out{Type: dbip},
			source: newSourcePlaces(unnamed).source(),
			lists:  enrichedLists,
			want:   newSourcePlaces(blank(dbipLanguages)).enriched(),
		},
		{
			name:   "ipinfo enriched",
			out:    geolite2_country_mmdb_out{Type: ipinfo},
			source: ipinfoSource,
			lists:  enrichedLists,
			want:   ipinfoEnriched,
		},
		{
			name:   "enriched wanted lists",
			out:    geolite2_country_mmdb_out{Type: ipinfo, Want: []string{"private", "de"}, OnlyIPType: lib.IPv4},
			source: ipinfoSource,
			lists:  enrichedLists,
			want: []mmdbRecord{
				{"5.1.0.0/16", ipinfoListRecord(ipinfoGermany)},
				{"10.0.0.0/8", countryCodeRecord("PRIVATE")},
			},
		},

		{
			name:    "missing source",
			out:     geolite2_country_mmdb_out{Type: geoLite2, SourceMMDBURI: missingFile},
			wantErr: errIs(os.ErrNotExist),
		},
		{
			name:    "source of another format",
			out:     geolite2_country_mmdb_out{Type: geoLite2},
			source:  ipinfoSource,
			wantErr: errAs[maxminddb.UnmarshalTypeError](),
		},
		{
			name:    "list without networks of the only ip type",
			out:     geolite2_country_mmdb_out{Type: geoLite2, OnlyIPType: lib.IPv4},
			lists:   map[string][]string{"CN": {"1.0.1.0/24"}, "IPV6": {"2001:db8::/32"}},
			wantErr: errContains("entry IPV6 has no prefix"),
		},
		{
			// 6to4 addresses are aliases of the IPv4 addresses they embed.
			name:    "network aliased to ipv4",
			out:     geolite2_country_mmdb_out{Type: geoLite2},
			lists:   map[string][]string{"6TO4": {"2002::/16"}},
			wantErr: errAs[*mmdbwriter.AliasedNetworkError](),
		},
		{
			name:    "unknown format",
			out:     geolite2_country_mmdb_out{Type: "unknown"},
			wantErr: errIs(lib.ErrNotSupportedFormat),
		},
		{
			name:    "output directory is a file",
			out:     geolite2_country_mmdb_out{Type: geoLite2, OutputDir: occupiedFile},
			wantErr: errPath("mkdir", occupiedFile),
		},
		{
			name:    "output file is a directory",
			out:     geolite2_country_mmdb_out{Type: geoLite2, OutputDir: dir, OutputName: occupiedName},
			wantErr: errPath("open", filepath.Join(dir, occupiedName)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.out
			g.Action = lib.ActionOutput
			if g.OutputDir == "" {
				g.OutputDir = t.TempDir()
			}
			if g.OutputName == "" {
				g.OutputName = defaultGeoLite2CountryMMDBOutputName
			}
			if tt.source != nil {
				g.SourceMMDBURI = writeTestFile(t, t.TempDir(), "source.mmdb", buildMMDB(t, tt.source...))
			}
			if tt.lists == nil {
				tt.lists = lists
			}

			err := g.Output(newTestContainer(t, tt.lists))
			if assertErr(t, err, tt.wantErr) {
				assertConverterErr(t, err, g.Type, g.Action)
				return
			}

			file := filepath.Join(g.OutputDir, g.OutputName)
			if tt.want == nil {
				if _, err := os.Stat(file); !errors.Is(err, fs.ErrNotExist) {
					t.Fatalf("no file is expected at %s, but Stat() returns error %v", file, err)
				}
				return
			}

			content, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			_, got := readMMDB(t, content)
			assertMMDBRecords(t, got, tt.want)
		})
	}
}

func TestGeoLite2CountryMMDBOut_Output_metadata(t *testing.T) {
	tests := []struct {
		name         string
		typ          string
		databaseType string
		description  string
		languages    []string
		recordSize   uint
	}{
		{"geolite2", TypeGeoLite2CountryMMDBOut, "GeoLite2-Country", "Customized GeoLite2 Country database", geoLite2Languages, 24},
		{"dbip", TypeDBIPCountryMMDBOut, "DBIP-Country-Lite", "Customized DB-IP Country Lite database", dbipLanguages, 24},
		{"ipinfo", TypeIPInfoCountryMMDBOut, "IPInfo-Lite", "Customized IPInfo Lite database", []string{"en"}, 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_country_mmdb_out{
				Type:       tt.typ,
				Action:     lib.ActionOutput,
				OutputName: defaultGeoLite2CountryMMDBOutputName,
				OutputDir:  t.TempDir(),
			}
			assertErr(t, g.Output(newTestContainer(t, map[string][]string{"CN": {"1.0.1.0/24"}})), nil)

			content, err := os.ReadFile(filepath.Join(g.OutputDir, g.OutputName))
			if err != nil {
				t.Fatal(err)
			}
			got, _ := readMMDB(t, content)

			if got.DatabaseType != tt.databaseType {
				t.Errorf("database type = %q, want %q", got.DatabaseType, tt.databaseType)
			}
			if want := map[string]string{"en": tt.description}; !maps.Equal(got.Description, want) {
				t.Errorf("description = %v, want %v", got.Description, want)
			}
			if !slices.Equal(got.Languages, tt.languages) {
				t.Errorf("languages = %v, want %v", got.Languages, tt.languages)
			}
			if got.RecordSize != tt.recordSize {
				t.Errorf("record size = %d, want %d", got.RecordSize, tt.recordSize)
			}
			// A database of IP version 6 holds both IPv4 and IPv6 networks.
			if got.IPVersion != 6 {
				t.Errorf("IP version = %d, want 6", got.IPVersion)
			}
		})
	}
}
