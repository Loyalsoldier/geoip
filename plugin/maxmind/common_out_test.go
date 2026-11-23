package maxmind

import (
	"reflect"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/oschwald/geoip2-golang/v2"
)

func TestGetDefaultMMDBOutputDir(t *testing.T) {
	tests := []struct {
		iType string
		want  string
	}{
		{TypeGeoLite2CountryMMDBOut, defaultMaxmindOutputDir},
		{TypeDBIPCountryMMDBOut, defaultDBIPOutputDir},
		{TypeIPInfoCountryMMDBOut, defaultIPInfoOutputDir},
		{"unknown", defaultMaxmindOutputDir},
	}
	for _, tt := range tests {
		t.Run(tt.iType, func(t *testing.T) {
			if got := getDefaultMMDBOutputDir(tt.iType); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewMMDBOut(t *testing.T) {
	tests := []struct {
		name string
		fn   func(lib.Action, ...lib.OutputOption) (lib.OutputConverter, error)
		opts []lib.OutputOption
		want *geolite2_country_mmdb_out
	}{
		{
			name: "GeoLite2 defaults",
			fn:   NewGeoLite2CountryMMDBOut,
			opts: []lib.OutputOption{nil},
			want: &geolite2_country_mmdb_out{
				Type: TypeGeoLite2CountryMMDBOut, Action: lib.ActionOutput, Description: DescGeoLite2CountryMMDBOut,
				OutputName: defaultGeoLite2CountryMMDBOutputName, OutputDir: defaultMaxmindOutputDir,
			},
		},
		{
			name: "DBIP with options",
			fn:   NewDBIPCountryMMDBOut,
			opts: []lib.OutputOption{
				WithOutputName(" a.mmdb "),
				WithOutputDir(" out "),
				WithOutputWantedList([]string{"CN"}),
				WithOverwriteList([]string{"US"}),
				WithOutputExcludedList([]string{"JP"}),
				WithOutputOnlyIPType(lib.IPv4),
				WithSourceMMDBURI(" src.mmdb "),
			},
			want: &geolite2_country_mmdb_out{
				Type: TypeDBIPCountryMMDBOut, Action: lib.ActionOutput, Description: DescDBIPCountryMMDBOut,
				OutputName: "a.mmdb", OutputDir: "out", Want: []string{"CN"}, Overwrite: []string{"US"},
				Exclude: []string{"JP"}, OnlyIPType: lib.IPv4, SourceMMDBURI: "src.mmdb",
			},
		},
		{
			name: "IPInfo with empty name and dir",
			fn:   NewIPInfoCountryMMDBOut,
			opts: []lib.OutputOption{WithOutputName("  "), WithOutputDir("  ")},
			want: &geolite2_country_mmdb_out{
				Type: TypeIPInfoCountryMMDBOut, Action: lib.ActionOutput, Description: DescIPInfoCountryMMDBOut,
				OutputName: defaultGeoLite2CountryMMDBOutputName, OutputDir: defaultIPInfoOutputDir,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fn(lib.ActionOutput, tt.opts...)
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
			if got.GetType() != tt.want.Type || got.GetAction() != tt.want.Action || got.GetDescription() != tt.want.Description {
				t.Errorf("getter mismatch")
			}
		})
	}
}

func TestNewMMDBOutFromBytes(t *testing.T) {
	ctors := []struct {
		name  string
		fn    func(lib.Action, []byte) (lib.OutputConverter, error)
		iType string
		desc  string
		dir   string
	}{
		{"GeoLite2", NewGeoLite2CountryMMDBOutFromBytes, TypeGeoLite2CountryMMDBOut, DescGeoLite2CountryMMDBOut, defaultMaxmindOutputDir},
		{"DBIP", NewDBIPCountryMMDBOutFromBytes, TypeDBIPCountryMMDBOut, DescDBIPCountryMMDBOut, defaultDBIPOutputDir},
		{"IPInfo", NewIPInfoCountryMMDBOutFromBytes, TypeIPInfoCountryMMDBOut, DescIPInfoCountryMMDBOut, defaultIPInfoOutputDir},
	}
	for _, c := range ctors {
		tests := []struct {
			name    string
			data    []byte
			want    *geolite2_country_mmdb_out
			wantErr bool
		}{
			{
				name: "Empty data",
				want: &geolite2_country_mmdb_out{
					Type: c.iType, Action: lib.ActionOutput, Description: c.desc,
					OutputName: defaultGeoLite2CountryMMDBOutputName, OutputDir: c.dir,
				},
			},
			{
				name: "Full data",
				data: []byte(`{"outputName":"x.mmdb","outputDir":"d","wantedList":["cn"],"overwriteList":["us"],"excludedList":["jp"],"onlyIPType":"ipv6","sourceMMDBURI":"s.mmdb"}`),
				want: &geolite2_country_mmdb_out{
					Type: c.iType, Action: lib.ActionOutput, Description: c.desc,
					OutputName: "x.mmdb", OutputDir: "d", Want: []string{"cn"}, Overwrite: []string{"us"},
					Exclude: []string{"jp"}, OnlyIPType: lib.IPv6, SourceMMDBURI: "s.mmdb",
				},
			},
			{name: "Invalid JSON", data: []byte(`{`), wantErr: true},
		}
		for _, tt := range tests {
			t.Run(c.name+"/"+tt.name, func(t *testing.T) {
				got, err := c.fn(lib.ActionOutput, tt.data)
				if (err != nil) != tt.wantErr {
					t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
				}
				if tt.wantErr {
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("got %+v, want %+v", got, tt.want)
				}
			})
		}
	}
}

func TestDBIPHasData(t *testing.T) {
	names := dbipLanguageNames{Korean: "x"}
	tests := []struct {
		name string
		d    dbipCountry
		want bool
	}{
		{"Zero", dbipCountry{}, false},
		{"Continent", dbipCountry{Continent: dbipContinent{Names: names}}, true},
		{"Represented country", dbipCountry{RepresentedCountry: dbipRepresentedCountry{Type: "military"}}, true},
		{"Country", dbipCountry{Country: dbipCountryRecord{Names: names}}, true},
		{"Registered country", dbipCountry{RegisteredCountry: dbipCountryRecord{Names: names}}, true},
		{"Traits", dbipCountry{Traits: geoip2.CountryTraits{IsAnycast: true}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.HasData(); got != tt.want {
				t.Errorf("HasData() = %v, want %v", got, tt.want)
			}
		})
	}
	if (dbipLanguageNames{}).HasData() || !names.HasData() {
		t.Errorf("dbipLanguageNames.HasData() mismatch")
	}
}

func geoNames(prefix string) mmdbtype.Map {
	m := mmdbtype.Map{}
	for _, lang := range []string{"de", "en", "es", "fa", "fr", "ja", "ko", "pt-BR", "ru", "zh-CN"} {
		m[mmdbtype.String(lang)] = mmdbtype.String(prefix + "-" + lang)
	}
	return m
}

// countrySourceRecords is used for both GeoLite2 and DB-IP source MMDBs.
func countrySourceRecords() []mmdbRecord {
	continent := mmdbtype.Map{"code": mmdbtype.String("AS"), "geoname_id": mmdbtype.Uint32(6255147), "names": geoNames("Asia")}
	return []mmdbRecord{
		{"1.0.1.0/24", mmdbtype.Map{
			"continent": continent,
			"country":   mmdbtype.Map{"iso_code": mmdbtype.String("cn"), "geoname_id": mmdbtype.Uint32(1814991), "names": geoNames("China")},
		}},
		// Duplicated country, must not overwrite the first one
		{"1.0.2.0/24", mmdbtype.Map{
			"country": mmdbtype.Map{"iso_code": mmdbtype.String("CN"), "geoname_id": mmdbtype.Uint32(1)},
		}},
		{"1.0.3.0/24", mmdbtype.Map{
			"registered_country": mmdbtype.Map{"iso_code": mmdbtype.String("de"), "geoname_id": mmdbtype.Uint32(2921044), "is_in_european_union": mmdbtype.Bool(true), "names": geoNames("Germany")},
		}},
		{"1.0.4.0/24", mmdbtype.Map{
			"registered_country": mmdbtype.Map{"iso_code": mmdbtype.String("DE")},
		}},
		{"1.0.5.0/24", mmdbtype.Map{
			"represented_country": mmdbtype.Map{"iso_code": mmdbtype.String("us"), "geoname_id": mmdbtype.Uint32(6252001), "names": geoNames("US")},
		}},
		{"1.0.6.0/24", mmdbtype.Map{
			"represented_country": mmdbtype.Map{"iso_code": mmdbtype.String("US")},
		}},
		{"1.0.7.0/24", mmdbtype.Map{"traits": mmdbtype.Map{"is_anycast": mmdbtype.Bool(true)}}},
	}
}

func ipinfoSourceRecords() []mmdbRecord {
	return []mmdbRecord{
		{"1.0.1.0/24", mmdbtype.Map{
			"asn": mmdbtype.String("AS1"), "as_name": mmdbtype.String("n"), "as_domain": mmdbtype.String("d"),
			"continent": mmdbtype.String("Asia"), "continent_code": mmdbtype.String("AS"),
			"country": mmdbtype.String("China"), "country_code": mmdbtype.String("cn"),
		}},
		{"1.0.2.0/24", mmdbtype.Map{"country": mmdbtype.String("Other"), "country_code": mmdbtype.String("CN")}},
	}
}

func wantGeoNames(prefix string) geoip2.Names {
	return geoip2.Names{
		German: prefix + "-de", English: prefix + "-en", Spanish: prefix + "-es", French: prefix + "-fr",
		Japanese: prefix + "-ja", BrazilianPortuguese: prefix + "-pt-BR", Russian: prefix + "-ru", SimplifiedChinese: prefix + "-zh-CN",
	}
}

func wantDBIPNames(prefix string) dbipLanguageNames {
	return dbipLanguageNames{dbipStandardNames: dbipStandardNames(wantGeoNames(prefix)), Persian: prefix + "-fa", Korean: prefix + "-ko"}
}

func TestGetExtraInfo(t *testing.T) {
	countrySrc := buildMMDB(t, "GeoLite2-Country", 24, countrySourceRecords())
	ipinfoSrc := buildMMDB(t, "IPInfo-Lite", 32, ipinfoSourceRecords())
	countryFile := writeTempFile(t, "country.mmdb", countrySrc)
	ipinfoFile := writeTempFile(t, "ipinfo.mmdb", ipinfoSrc)
	emptyFile := writeTempFile(t, "empty.mmdb", buildMMDB(t, "GeoLite2-Country", 24, []mmdbRecord{
		{"1.0.7.0/24", mmdbtype.Map{"traits": mmdbtype.Map{"is_anycast": mmdbtype.Bool(true)}}},
	}))
	badCountryFile := writeTempFile(t, "badcountry.mmdb", buildMMDB(t, "GeoLite2-Country", 24, []mmdbRecord{
		{"1.0.1.0/24", mmdbtype.Map{"country": mmdbtype.String("x")}},
	}))
	badIPInfoFile := writeTempFile(t, "badipinfo.mmdb", buildMMDB(t, "IPInfo-Lite", 32, []mmdbRecord{
		{"1.0.1.0/24", mmdbtype.Map{"country_code": mmdbtype.Map{}}},
	}))
	garbageFile := writeTempFile(t, "garbage.mmdb", []byte("garbage"))
	srv := newFileServer(t, map[string][]byte{"/country.mmdb": countrySrc})

	geoliteWant := map[string]any{
		"CN": geoip2.Country{
			Continent: geoip2.Continent{Code: "AS", GeoNameID: 6255147, Names: wantGeoNames("Asia")},
			Country:   geoip2.CountryRecord{ISOCode: "cn", GeoNameID: 1814991, Names: wantGeoNames("China")},
		},
		"DE": geoip2.Country{
			Country: geoip2.CountryRecord{ISOCode: "de", GeoNameID: 2921044, IsInEuropeanUnion: true, Names: wantGeoNames("Germany")},
		},
		"US": geoip2.Country{
			Country: geoip2.CountryRecord{ISOCode: "us", GeoNameID: 6252001, Names: wantGeoNames("US")},
		},
	}
	dbipWant := map[string]any{
		"CN": dbipCountry{
			Continent: dbipContinent{Code: "AS", GeoNameID: 6255147, Names: wantDBIPNames("Asia")},
			Country:   dbipCountryRecord{ISOCode: "cn", GeoNameID: 1814991, Names: wantDBIPNames("China")},
		},
		"DE": dbipCountry{
			Country: dbipCountryRecord{ISOCode: "de", GeoNameID: 2921044, IsInEuropeanUnion: true, Names: wantDBIPNames("Germany")},
		},
		"US": dbipCountry{
			Country: dbipCountryRecord{ISOCode: "us", GeoNameID: 6252001, Names: wantDBIPNames("US")},
		},
	}
	ipinfoWant := map[string]any{
		"CN": ipInfoLite{Continent: "Asia", ContinentCode: "AS", Country: "China", CountryCode: "cn"},
	}

	tests := []struct {
		name    string
		g       *geolite2_country_mmdb_out
		want    map[string]any
		wantErr bool
	}{
		{name: "Empty URI", g: &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: "  "}},
		{name: "GeoLite2", g: &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: countryFile}, want: geoliteWant},
		{name: "GeoLite2 remote", g: &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: srv.URL + "/country.mmdb"}, want: geoliteWant},
		{name: "DBIP", g: &geolite2_country_mmdb_out{Type: TypeDBIPCountryMMDBOut, SourceMMDBURI: countryFile}, want: dbipWant},
		{name: "IPInfo", g: &geolite2_country_mmdb_out{Type: TypeIPInfoCountryMMDBOut, SourceMMDBURI: ipinfoFile}, want: ipinfoWant},
		{name: "Unsupported type", g: &geolite2_country_mmdb_out{Type: "unknown", SourceMMDBURI: countryFile}, wantErr: true},
		{name: "No extra info", g: &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: emptyFile}, wantErr: true},
		{name: "File not found", g: &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: "/non/existent"}, wantErr: true},
		{name: "Remote 404", g: &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: srv.URL + "/missing"}, wantErr: true},
		{name: "Invalid MMDB", g: &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: garbageFile}, wantErr: true},
		{name: "GeoLite2 decode error", g: &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: badCountryFile}, wantErr: true},
		{name: "DBIP decode error", g: &geolite2_country_mmdb_out{Type: TypeDBIPCountryMMDBOut, SourceMMDBURI: badCountryFile}, wantErr: true},
		{name: "IPInfo decode error", g: &geolite2_country_mmdb_out{Type: TypeIPInfoCountryMMDBOut, SourceMMDBURI: badIPInfoFile}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.g.GetExtraInfo()
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetExtraInfo() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
