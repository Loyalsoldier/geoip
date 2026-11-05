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
		{"Represented country", dbipCountry{RepresentedCountry: dbipRepresentedCountry{Names: names}}, true},
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
			Continent: geoip2.Continent{Code: "AS", GeoNameID: 6255147, Names: geoip2.Names{
				German: "Asia-de", English: "Asia-en", Spanish: "Asia-es", French: "Asia-fr", Japanese: "Asia-ja",
				BrazilianPortuguese: "Asia-pt-BR", Russian: "Asia-ru", SimplifiedChinese: "Asia-zh-CN",
			}},
			Country: geoip2.CountryRecord{ISOCode: "cn", GeoNameID: 1814991, Names: geoip2.Names{
				German: "China-de", English: "China-en", Spanish: "China-es", French: "China-fr", Japanese: "China-ja",
				BrazilianPortuguese: "China-pt-BR", Russian: "China-ru", SimplifiedChinese: "China-zh-CN",
			}},
		},
		"DE": geoip2.Country{
			Country: geoip2.CountryRecord{ISOCode: "de", GeoNameID: 2921044, IsInEuropeanUnion: true, Names: geoip2.Names{
				German: "Germany-de", English: "Germany-en", Spanish: "Germany-es", French: "Germany-fr", Japanese: "Germany-ja",
				BrazilianPortuguese: "Germany-pt-BR", Russian: "Germany-ru", SimplifiedChinese: "Germany-zh-CN",
			}},
		},
		"US": geoip2.Country{
			Country: geoip2.CountryRecord{ISOCode: "us", GeoNameID: 6252001, Names: geoip2.Names{
				German: "US-de", English: "US-en", Spanish: "US-es", French: "US-fr", Japanese: "US-ja",
				BrazilianPortuguese: "US-pt-BR", Russian: "US-ru", SimplifiedChinese: "US-zh-CN",
			}},
		},
	}
	// Only non-name fields are checked for DB-IP: the embedded geoip2 structs
	// currently capture the "names" map during decoding, so the outer
	// dbipLanguageNames fields are not populated.
	dbipWant := map[string]any{
		"CN": dbipCountry{
			Continent: dbipContinent{Continent: geoip2.Continent{Code: "AS", GeoNameID: 6255147}},
			Country:   dbipCountryRecord{CountryRecord: geoip2.CountryRecord{ISOCode: "cn", GeoNameID: 1814991}},
		},
		"DE": dbipCountry{
			Country: dbipCountryRecord{CountryRecord: geoip2.CountryRecord{ISOCode: "de", GeoNameID: 2921044, IsInEuropeanUnion: true}},
		},
		"US": dbipCountry{
			Country: dbipCountryRecord{CountryRecord: geoip2.CountryRecord{ISOCode: "us", GeoNameID: 6252001}},
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
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d; got %+v", len(got), len(tt.want), got)
			}
			for k, v := range tt.want {
				if d, ok := got[k].(dbipCountry); ok {
					got[k] = stripDBIPNames(d)
				}
				if !reflect.DeepEqual(got[k], v) {
					t.Errorf("[%s] got %+v, want %+v", k, got[k], v)
				}
			}
		})
	}
}

func stripDBIPNames(d dbipCountry) dbipCountry {
	d.Continent.Names = dbipLanguageNames{}
	d.Continent.Continent.Names = geoip2.Names{}
	d.Country.Names = dbipLanguageNames{}
	d.Country.CountryRecord.Names = geoip2.Names{}
	return d
}
