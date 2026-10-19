package maxmind

import (
	"errors"
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
)

func names(en, de, zh string) mmdbtype.Map {
	return mmdbtype.Map{"en": mmdbtype.String(en), "de": mmdbtype.String(de), "zh-CN": mmdbtype.String(zh)}
}

func dbipNames(en, fa, ko string) mmdbtype.Map {
	return mmdbtype.Map{"en": mmdbtype.String(en), "fa": mmdbtype.String(fa), "ko": mmdbtype.String(ko)}
}

// extraInfoFixture covers country / registered / represented / no-code
// records plus a duplicated country to verify "first wins".
func extraInfoFixture(nameFn func(a, b, c string) mmdbtype.Map) []mmdbRecord {
	return []mmdbRecord{
		{"1.0.0.0/24", mmdbtype.Map{
			"continent": mmdbtype.Map{"code": mmdbtype.String("AS"), "geoname_id": mmdbtype.Uint32(6255147), "names": nameFn("Asia", "Asien", "亚洲")},
			"country":   mmdbtype.Map{"iso_code": mmdbtype.String("cn"), "geoname_id": mmdbtype.Uint32(1814991), "names": nameFn("China", "China", "中国")},
		}},
		{"4.4.4.0/24", mmdbtype.Map{"continent": mmdbtype.Map{"code": mmdbtype.String("NA")}}},
		{"8.8.8.0/24", mmdbtype.Map{
			"registered_country": mmdbtype.Map{"iso_code": mmdbtype.String("US"), "geoname_id": mmdbtype.Uint32(6252001), "names": nameFn("United States", "USA", "美国")},
		}},
		{"9.9.9.0/24", mmdbtype.Map{
			"continent":           mmdbtype.Map{"code": mmdbtype.String("EU"), "geoname_id": mmdbtype.Uint32(6255148), "names": nameFn("Europe", "Europa", "欧洲")},
			"represented_country": mmdbtype.Map{"iso_code": mmdbtype.String("DE"), "geoname_id": mmdbtype.Uint32(2921044), "is_in_european_union": mmdbtype.Bool(true), "names": nameFn("Germany", "Deutschland", "德国")},
		}},
		{"2001:4860::/32", mmdbtype.Map{
			"country": mmdbtype.Map{"iso_code": mmdbtype.String("CN"), "geoname_id": mmdbtype.Uint32(1), "names": nameFn("Other", "Other", "Other")},
		}},
	}
}

func ipinfoExtraFixture() []mmdbRecord {
	rec := func(cc, country, cont, contCode string) mmdbtype.Map {
		return mmdbtype.Map{
			"asn": mmdbtype.String("AS13335"), "as_name": mmdbtype.String("Cloudflare"), "as_domain": mmdbtype.String("cloudflare.com"),
			"country_code": mmdbtype.String(cc), "country": mmdbtype.String(country),
			"continent": mmdbtype.String(cont), "continent_code": mmdbtype.String(contCode),
		}
	}
	return []mmdbRecord{
		{"1.0.0.0/24", rec("cn", "China", "Asia", "AS")},
		{"8.8.8.0/24", rec("US", "United States", "North America", "NA")},
		{"2001:4860::/32", rec("CN", "Other", "Other", "OT")},
	}
}

func TestGetDefaultMMDBOutputDir(t *testing.T) {
	tests := []struct {
		typ  string
		want string
	}{
		{TypeGeoLite2CountryMMDBOut, defaultMaxmindOutputDir},
		{TypeDBIPCountryMMDBOut, defaultDBIPOutputDir},
		{TypeIPInfoCountryMMDBOut, defaultIPInfoOutputDir},
		{"other", defaultMaxmindOutputDir},
	}
	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			if got := getDefaultMMDBOutputDir(tt.typ); got != tt.want {
				t.Errorf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestNewMMDBOut(t *testing.T) {
	type ctor func(lib.Action, ...lib.OutputOption) (lib.OutputConverter, error)
	tests := []struct {
		name     string
		ctor     ctor
		opts     []lib.OutputOption
		wantType string
		wantDesc string
		wantName string
		wantDir  string
		wantSrc  string
	}{
		{"geolite2 default", NewGeoLite2CountryMMDBOut, nil, TypeGeoLite2CountryMMDBOut, DescGeoLite2CountryMMDBOut, "Country.mmdb", defaultMaxmindOutputDir, ""},
		{"dbip default", NewDBIPCountryMMDBOut, []lib.OutputOption{nil}, TypeDBIPCountryMMDBOut, DescDBIPCountryMMDBOut, "Country.mmdb", defaultDBIPOutputDir, ""},
		{"ipinfo default", NewIPInfoCountryMMDBOut, nil, TypeIPInfoCountryMMDBOut, DescIPInfoCountryMMDBOut, "Country.mmdb", defaultIPInfoOutputDir, ""},
		{"custom", NewGeoLite2CountryMMDBOut,
			[]lib.OutputOption{WithOutputName(" a.mmdb "), WithOutputDir(" /tmp/out "), WithSourceMMDBURI(" src.mmdb ")},
			TypeGeoLite2CountryMMDBOut, DescGeoLite2CountryMMDBOut, "a.mmdb", "/tmp/out", "src.mmdb"},
		{"blank name/dir fall back", NewIPInfoCountryMMDBOut,
			[]lib.OutputOption{WithOutputName("  "), WithOutputDir("  ")},
			TypeIPInfoCountryMMDBOut, DescIPInfoCountryMMDBOut, "Country.mmdb", defaultIPInfoOutputDir, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := tt.ctor(lib.ActionOutput, tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			g := conv.(*geolite2_country_mmdb_out)
			if g.GetType() != tt.wantType || g.GetDescription() != tt.wantDesc || g.GetAction() != lib.ActionOutput {
				t.Errorf("metadata: %+v", g)
			}
			if g.OutputName != tt.wantName || g.OutputDir != tt.wantDir || g.SourceMMDBURI != tt.wantSrc {
				t.Errorf("got %+v", g)
			}
		})
	}
}

func TestNewMMDBOutFromBytes(t *testing.T) {
	type ctor func(lib.Action, []byte) (lib.OutputConverter, error)
	tests := []struct {
		name    string
		ctor    ctor
		data    string
		wantErr bool
		check   func(t *testing.T, g *geolite2_country_mmdb_out)
	}{
		{"geolite2 empty", NewGeoLite2CountryMMDBOutFromBytes, "", false, func(t *testing.T, g *geolite2_country_mmdb_out) {
			if g.Type != TypeGeoLite2CountryMMDBOut || g.OutputName != "Country.mmdb" || g.OutputDir != defaultMaxmindOutputDir {
				t.Errorf("got %+v", g)
			}
		}},
		{"dbip full", NewDBIPCountryMMDBOutFromBytes,
			`{"outputName":"x.mmdb","outputDir":"d","wantedList":["a"],"overwriteList":["b"],"excludedList":["c"],"onlyIPType":"ipv4","sourceMMDBURI":"s"}`,
			false, func(t *testing.T, g *geolite2_country_mmdb_out) {
				if g.Type != TypeDBIPCountryMMDBOut || g.OutputName != "x.mmdb" || g.OutputDir != "d" || g.OnlyIPType != lib.IPv4 || g.SourceMMDBURI != "s" ||
					!slices.Equal(g.Want, []string{"a"}) || !slices.Equal(g.Overwrite, []string{"b"}) || !slices.Equal(g.Exclude, []string{"c"}) {
					t.Errorf("got %+v", g)
				}
			}},
		{"ipinfo empty object", NewIPInfoCountryMMDBOutFromBytes, `{}`, false, func(t *testing.T, g *geolite2_country_mmdb_out) {
			if g.Type != TypeIPInfoCountryMMDBOut || g.OutputDir != defaultIPInfoOutputDir {
				t.Errorf("got %+v", g)
			}
		}},
		{"geolite2 invalid", NewGeoLite2CountryMMDBOutFromBytes, `{`, true, nil},
		{"dbip invalid", NewDBIPCountryMMDBOutFromBytes, `[]`, true, nil},
		{"ipinfo invalid", NewIPInfoCountryMMDBOutFromBytes, `{"outputName":1}`, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := tt.ctor(lib.ActionOutput, []byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, conv.(*geolite2_country_mmdb_out))
		})
	}
}

func TestMMDBOutRegistered(t *testing.T) {
	tests := []struct {
		name string
		typ  string
	}{
		{"geolite2", TypeGeoLite2CountryMMDBOut},
		{"dbip", TypeDBIPCountryMMDBOut},
		{"ipinfo", TypeIPInfoCountryMMDBOut},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst, _ := lib.NewInstance()
			if err := inst.InitConfigFromBytes([]byte(`{"input":[],"output":[{"type":"` + tt.typ + `","action":"output","args":{"outputName":"a"}}]}`)); err != nil {
				t.Fatal(err)
			}
			inst2, _ := lib.NewInstance()
			if err := inst2.InitConfigFromBytes([]byte(`{"input":[],"output":[{"type":"` + tt.typ + `","action":"output","args":"bad"}]}`)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestFilterAndSortList(t *testing.T) {
	c := containerWith(t, map[string][]string{
		"CN": {"1.0.0.0/24"}, "US": {"8.8.8.0/24"}, "JP": {"9.9.9.0/24"}, "DE": {"5.5.5.0/24"},
	})
	tests := []struct {
		name      string
		want      []string
		overwrite []string
		exclude   []string
		expect    []string
	}{
		{"all sorted", nil, nil, nil, []string{"CN", "DE", "JP", "US"}},
		{"exclude", nil, nil, []string{" de ", ""}, []string{"CN", "JP", "US"}},
		{"overwrite last in given order", nil, []string{"us", "CN", " "}, nil, []string{"DE", "JP", "US", "CN"}},
		{"overwrite excluded", nil, []string{"US"}, []string{"us"}, []string{"CN", "DE", "JP"}},
		{"wanted keeps order and ignores overwrite", []string{" us", "cn", "", "XX"}, []string{"JP"}, nil, []string{"US", "CN", "XX"}},
		{"wanted minus excluded", []string{"US", "CN"}, nil, []string{"cn"}, []string{"US"}},
		{"wanted all excluded", []string{"US"}, nil, []string{"US"}, []string{}},
		{"wanted only blanks behaves as no wanted list", []string{" ", ""}, nil, nil, []string{"CN", "DE", "JP", "US"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_country_mmdb_out{Want: tt.want, Overwrite: tt.overwrite, Exclude: tt.exclude}
			got := g.filterAndSortList(c)
			if len(got) != len(tt.expect) || !slices.Equal(got, tt.expect) {
				t.Errorf("got %v (%d), want %v (%d)", got, len(got), tt.expect, len(tt.expect))
			}
		})
	}
}

func TestDBIPHasData(t *testing.T) {
	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{"names zero", dbipLanguageNames{}.HasData(), false},
		{"names persian", dbipLanguageNames{Persian: "x"}.HasData(), true},
		{"names english", dbipLanguageNames{English: "x"}.HasData(), true},
		{"continent zero", dbipContinent{}.HasData(), false},
		{"continent names", dbipContinent{Names: dbipLanguageNames{Korean: "x"}}.HasData(), true},
		{"continent code", dbipContinent{Code: "AS"}.HasData(), true},
		{"country record zero", dbipCountryRecord{}.HasData(), false},
		{"country record iso", dbipCountryRecord{ISOCode: "CN"}.HasData(), true},
		{"represented zero", dbipRepresentedCountry{}.HasData(), false},
		{"represented iso", dbipRepresentedCountry{ISOCode: "DE"}.HasData(), true},
		{"country zero", dbipCountry{}.HasData(), false},
		{"country via continent", dbipCountry{Continent: dbipContinent{Code: "EU"}}.HasData(), true},
		{"country via represented", dbipCountry{RepresentedCountry: dbipRepresentedCountry{Names: dbipLanguageNames{Persian: "x"}}}.HasData(), true},
		{"country via country", dbipCountry{Country: dbipCountryRecord{Names: dbipLanguageNames{Korean: "x"}}}.HasData(), true},
		{"country via registered", dbipCountry{RegisteredCountry: dbipCountryRecord{ISOCode: "US"}}.HasData(), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %v want %v", tt.got, tt.want)
			}
		})
	}
}

func TestGetExtraInfoGeoLite2AndDBIP(t *testing.T) {
	geoBytes := buildMMDB(t, "GeoLite2-Country", extraInfoFixture(names))
	geoPath := writeFile(t, "geo.mmdb", geoBytes)
	dbipPath := writeMMDB(t, "DBIP-Country-Lite", extraInfoFixture(dbipNames))
	srv := serveFiles(t, map[string][]byte{"/geo.mmdb": geoBytes})

	type rec struct {
		contCode, contEn       string
		contID                 uint64
		iso, en                string
		id                     uint64
		eu                     bool
		extra1, extra2, extra3 string // de/zh-CN for geolite2, fa/ko for dbip
	}
	tests := []struct {
		name   string
		typ    string
		uri    string
		expect map[string]rec
	}{
		{"geolite2 local", TypeGeoLite2CountryMMDBOut, geoPath, map[string]rec{
			"CN": {"AS", "Asia", 6255147, "cn", "China", 1814991, false, "China", "中国", ""},
			"US": {"", "", 0, "US", "United States", 6252001, false, "USA", "美国", ""},
			"DE": {"EU", "Europe", 6255148, "DE", "Germany", 2921044, true, "Deutschland", "德国", ""},
		}},
		{"geolite2 remote", TypeGeoLite2CountryMMDBOut, srv.URL + "/geo.mmdb", map[string]rec{
			"CN": {"AS", "Asia", 6255147, "cn", "China", 1814991, false, "China", "中国", ""},
			"US": {"", "", 0, "US", "United States", 6252001, false, "USA", "美国", ""},
			"DE": {"EU", "Europe", 6255148, "DE", "Germany", 2921044, true, "Deutschland", "德国", ""},
		}},
		{"dbip local", TypeDBIPCountryMMDBOut, dbipPath, map[string]rec{
			"CN": {"AS", "Asia", 6255147, "cn", "China", 1814991, false, "China", "中国", ""},
			"US": {"", "", 0, "US", "United States", 6252001, false, "USA", "美国", ""},
			"DE": {"EU", "Europe", 6255148, "DE", "Germany", 2921044, true, "Deutschland", "德国", ""},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_country_mmdb_out{Type: tt.typ, SourceMMDBURI: tt.uri}
			info, err := g.GetExtraInfo()
			if err != nil {
				t.Fatal(err)
			}
			if len(info) != len(tt.expect) {
				t.Fatalf("got %d keys %v, want %d", len(info), info, len(tt.expect))
			}
			for cc, want := range tt.expect {
				var got rec
				switch v := info[cc].(type) {
				case geoip2.Country:
					got = rec{v.Continent.Code, v.Continent.Names.English, uint64(v.Continent.GeoNameID), v.Country.ISOCode, v.Country.Names.English,
						uint64(v.Country.GeoNameID), v.Country.IsInEuropeanUnion, v.Country.Names.German, v.Country.Names.SimplifiedChinese, ""}
				case dbipCountry:
					// dbip fixture uses fa/ko instead of de/zh-CN
					got = rec{v.Continent.Code, v.Continent.Names.English, uint64(v.Continent.GeoNameID), v.Country.ISOCode, v.Country.Names.English,
						uint64(v.Country.GeoNameID), v.Country.IsInEuropeanUnion, v.Country.Names.Persian, v.Country.Names.Korean, ""}
				default:
					t.Fatalf("%s: unexpected type %T", cc, v)
				}
				if got != want {
					t.Errorf("%s: got %+v, want %+v", cc, got, want)
				}
			}
		})
	}
}

func TestGetExtraInfoIPInfo(t *testing.T) {
	path := writeMMDB(t, "ipinfo", ipinfoExtraFixture())
	tests := []struct {
		name   string
		expect map[string]ipInfoLite
	}{
		{"first record wins and AS fields are cleared", map[string]ipInfoLite{
			"CN": {Continent: "Asia", ContinentCode: "AS", Country: "China", CountryCode: "cn"},
			"US": {Continent: "North America", ContinentCode: "NA", Country: "United States", CountryCode: "US"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_country_mmdb_out{Type: TypeIPInfoCountryMMDBOut, SourceMMDBURI: path}
			info, err := g.GetExtraInfo()
			if err != nil {
				t.Fatal(err)
			}
			if len(info) != len(tt.expect) {
				t.Fatalf("got %v", info)
			}
			for k, v := range tt.expect {
				if info[k] != v {
					t.Errorf("%s: got %+v want %+v", k, info[k], v)
				}
			}
		})
	}
}

func TestGetExtraInfoErrors(t *testing.T) {
	notMMDB := writeFile(t, "x.mmdb", []byte("garbage"))
	noInfo := writeMMDB(t, "x", []mmdbRecord{{"4.4.4.0/24", mmdbtype.Map{"continent": mmdbtype.Map{"code": mmdbtype.String("NA")}}}})
	badCountry := writeMMDB(t, "x", []mmdbRecord{{"1.0.0.0/24", mmdbtype.Map{"country": mmdbtype.String("bad")}}})
	badIPInfo := writeMMDB(t, "x", []mmdbRecord{{"1.0.0.0/24", mmdbtype.Map{"country_code": mmdbtype.Uint32(1)}}})
	good := writeMMDB(t, "x", extraInfoFixture(names))
	srv := serveFiles(t, nil)

	tests := []struct {
		name    string
		typ     string
		uri     string
		wantErr string
		wantNil bool
	}{
		{"empty uri returns nil", TypeGeoLite2CountryMMDBOut, "   ", "", true},
		{"missing file", TypeGeoLite2CountryMMDBOut, "/nonexistent/x.mmdb", "no such file", false},
		{"remote 404", TypeGeoLite2CountryMMDBOut, srv.URL + "/x.mmdb", "404", false},
		{"not mmdb", TypeGeoLite2CountryMMDBOut, notMMDB, "", false},
		{"no extra info", TypeGeoLite2CountryMMDBOut, noInfo, "no extra info found", false},
		{"geolite2 decode error", TypeGeoLite2CountryMMDBOut, badCountry, "", false},
		{"dbip decode error", TypeDBIPCountryMMDBOut, badCountry, "", false},
		{"ipinfo decode error", TypeIPInfoCountryMMDBOut, badIPInfo, "", false},
		{"unsupported type", "other", good, lib.ErrNotSupportedFormat.Error(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_country_mmdb_out{Type: tt.typ, SourceMMDBURI: tt.uri}
			info, err := g.GetExtraInfo()
			if tt.wantNil {
				if err != nil || info != nil {
					t.Fatalf("got %v, %v", info, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error, got info %v", info)
			}
			if tt.wantErr != "" {
				assertErrContains(t, err, tt.wantErr)
			}
		})
	}
}

// readBack loads an output MMDB with the matching input converter.
func readBack(t *testing.T, inType, path string) map[string][]string {
	t.Helper()
	conv, err := newGeoLite2CountryMMDBIn(inType, "d", lib.ActionAdd, WithURI(path))
	if err != nil {
		t.Fatal(err)
	}
	c, err := conv.Input(lib.NewContainer())
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	return dumpContainer(t, c)
}

func TestMMDBOutOutput(t *testing.T) {
	data := map[string][]string{
		"CN": {"1.0.0.0/24", "2001:4860::/32"},
		"US": {"8.8.8.0/24", "2400:cb00::/32"},
		"JP": {"9.9.9.0/24"},
	}

	tests := []struct {
		name       string
		typ        string
		inType     string
		want       []string
		overwrite  []string
		exclude    []string
		only       lib.IPType
		dbType     string
		recordSize uint
		expect     map[string][]string
	}{
		{name: "geolite2 all", typ: TypeGeoLite2CountryMMDBOut, inType: TypeGeoLite2CountryMMDBIn, dbType: "GeoLite2-Country", recordSize: 24, expect: data},
		{name: "dbip all", typ: TypeDBIPCountryMMDBOut, inType: TypeDBIPCountryMMDBIn, dbType: "DBIP-Country-Lite", recordSize: 24, expect: data},
		{name: "ipinfo all", typ: TypeIPInfoCountryMMDBOut, inType: TypeIPInfoCountryMMDBIn, dbType: "IPInfo-Lite", recordSize: 32, expect: data},
		{name: "geolite2 wanted", typ: TypeGeoLite2CountryMMDBOut, inType: TypeGeoLite2CountryMMDBIn, want: []string{"jp"}, dbType: "GeoLite2-Country", recordSize: 24,
			expect: map[string][]string{"JP": {"9.9.9.0/24"}}},
		{name: "geolite2 exclude", typ: TypeGeoLite2CountryMMDBOut, inType: TypeGeoLite2CountryMMDBIn, exclude: []string{"US"}, dbType: "GeoLite2-Country", recordSize: 24,
			expect: map[string][]string{"CN": {"1.0.0.0/24", "2001:4860::/32"}, "JP": {"9.9.9.0/24"}}},
		{name: "geolite2 only ipv6", typ: TypeGeoLite2CountryMMDBOut, inType: TypeGeoLite2CountryMMDBIn, want: []string{"CN", "US"}, only: lib.IPv6, dbType: "GeoLite2-Country", recordSize: 24,
			expect: map[string][]string{"CN": {"2001:4860::/32"}, "US": {"2400:cb00::/32"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "nested", "out")
			g := &geolite2_country_mmdb_out{Type: tt.typ, Action: lib.ActionOutput, OutputName: "out.mmdb", OutputDir: dir,
				Want: tt.want, Overwrite: tt.overwrite, Exclude: tt.exclude, OnlyIPType: tt.only}
			if err := g.Output(containerWith(t, data)); err != nil {
				t.Fatalf("Output: %v", err)
			}
			path := filepath.Join(dir, "out.mmdb")
			db, err := maxminddb.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			if db.Metadata.DatabaseType != tt.dbType || uint(db.Metadata.RecordSize) != tt.recordSize {
				t.Errorf("metadata: type=%q size=%d", db.Metadata.DatabaseType, db.Metadata.RecordSize)
			}
			db.Close()

			got := readBack(t, tt.inType, path)
			if len(got) != len(tt.expect) {
				t.Fatalf("got %v, want %v", got, tt.expect)
			}
			for k, v := range tt.expect {
				if len(got[k]) != len(v) || !slices.Equal(got[k], v) {
					t.Errorf("%s: got %v want %v", k, got[k], v)
				}
			}
		})
	}
}

func TestMMDBOutOverwriteOrder(t *testing.T) {
	data := map[string][]string{"AA": {"1.0.0.0/16"}, "ZZ": {"1.0.0.0/24"}}
	tests := []struct {
		name      string
		overwrite []string
		expect    map[string][]string
	}{
		{"default sort: ZZ written last wins overlap", nil, map[string][]string{
			"AA": {"1.0.1.0/24", "1.0.2.0/23", "1.0.4.0/22", "1.0.8.0/21", "1.0.16.0/20", "1.0.32.0/19", "1.0.64.0/18", "1.0.128.0/17"},
			"ZZ": {"1.0.0.0/24"},
		}},
		{"overwrite AA: AA written last wins", []string{"AA"}, map[string][]string{"AA": {"1.0.0.0/16"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			g := &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, OutputName: "o.mmdb", OutputDir: dir, Overwrite: tt.overwrite}
			if err := g.Output(containerWith(t, data)); err != nil {
				t.Fatal(err)
			}
			got := readBack(t, TypeGeoLite2CountryMMDBIn, filepath.Join(dir, "o.mmdb"))
			if len(got) != len(tt.expect) {
				t.Fatalf("got %v want %v", got, tt.expect)
			}
			for k, v := range tt.expect {
				if !slices.Equal(got[k], v) {
					t.Errorf("%s: got %v want %v", k, got[k], v)
				}
			}
		})
	}
}

func TestMMDBOutOutputWithExtraInfo(t *testing.T) {
	geoSrc := writeMMDB(t, "GeoLite2-Country", extraInfoFixture(names))
	dbipSrc := writeMMDB(t, "DBIP-Country-Lite", extraInfoFixture(dbipNames))
	ipinfoSrc := writeMMDB(t, "ipinfo", ipinfoExtraFixture())

	data := map[string][]string{
		"CN": {"1.0.0.0/24"}, // has continent
		"US": {"8.8.8.0/24"}, // no continent
		"FR": {"5.5.5.0/24"}, // not in source
	}

	tests := []struct {
		name  string
		typ   string
		src   string
		check func(t *testing.T, db *maxminddb.Reader)
	}{
		{"geolite2", TypeGeoLite2CountryMMDBOut, geoSrc, func(t *testing.T, db *maxminddb.Reader) {
			cases := []struct {
				ip, iso, en, contCode, zh string
				id                        uint64
			}{
				{"1.0.0.1", "CN", "China", "AS", "中国", 1814991},
				{"8.8.8.8", "US", "United States", "", "美国", 6252001},
				{"5.5.5.5", "FR", "", "", "", 0},
			}
			for _, c := range cases {
				var r geoip2.Country
				if err := db.Lookup(netip.MustParseAddr(c.ip)).Decode(&r); err != nil {
					t.Fatal(err)
				}
				if r.Country.ISOCode != c.iso || r.Country.Names.English != c.en || r.Continent.Code != c.contCode ||
					r.Country.Names.SimplifiedChinese != c.zh || uint64(r.Country.GeoNameID) != c.id {
					t.Errorf("%s: got %+v", c.ip, r)
				}
			}
		}},
		{"dbip", TypeDBIPCountryMMDBOut, dbipSrc, func(t *testing.T, db *maxminddb.Reader) {
			cases := []struct {
				ip, iso, en, contCode, fa, ko string
			}{
				{"1.0.0.1", "CN", "China", "AS", "China", "中国"},
				{"8.8.8.8", "US", "United States", "", "USA", "美国"},
				{"5.5.5.5", "FR", "", "", "", ""},
			}
			for _, c := range cases {
				var r dbipCountry
				if err := db.Lookup(netip.MustParseAddr(c.ip)).Decode(&r); err != nil {
					t.Fatal(err)
				}
				if r.Country.ISOCode != c.iso || r.Country.Names.English != c.en || r.Continent.Code != c.contCode ||
					r.Country.Names.Persian != c.fa || r.Country.Names.Korean != c.ko {
					t.Errorf("%s: got %+v", c.ip, r)
				}
			}
		}},
		{"ipinfo", TypeIPInfoCountryMMDBOut, ipinfoSrc, func(t *testing.T, db *maxminddb.Reader) {
			cases := []struct {
				ip   string
				want ipInfoLite
			}{
				{"1.0.0.1", ipInfoLite{Continent: "Asia", ContinentCode: "AS", Country: "China", CountryCode: "CN"}},
				{"8.8.8.8", ipInfoLite{Continent: "North America", ContinentCode: "NA", Country: "United States", CountryCode: "US"}},
				{"5.5.5.5", ipInfoLite{CountryCode: "FR"}},
			}
			for _, c := range cases {
				var r ipInfoLite
				if err := db.Lookup(netip.MustParseAddr(c.ip)).Decode(&r); err != nil {
					t.Fatal(err)
				}
				if r != c.want {
					t.Errorf("%s: got %+v want %+v", c.ip, r, c.want)
				}
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			g := &geolite2_country_mmdb_out{Type: tt.typ, OutputName: "o.mmdb", OutputDir: dir, SourceMMDBURI: tt.src}
			if err := g.Output(containerWith(t, data)); err != nil {
				t.Fatal(err)
			}
			db, err := maxminddb.Open(filepath.Join(dir, "o.mmdb"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			tt.check(t, db)
		})
	}
}

func TestMMDBOutOutputErrorsAndNoop(t *testing.T) {
	data := map[string][]string{"CN": {"1.0.0.0/24"}, "V6": {"2001:4860::/32"}}
	fileAsDir := writeFile(t, "afile", []byte("x"))

	tests := []struct {
		name      string
		typ       string
		dir       string
		outName   string
		want      []string
		only      lib.IPType
		src       string
		wantErr   string
		wantNoOut bool
	}{
		{name: "extra info error", typ: TypeGeoLite2CountryMMDBOut, src: "/nonexistent/src.mmdb", wantErr: "no such file"},
		{name: "wanted entry not found writes nothing", typ: TypeGeoLite2CountryMMDBOut, want: []string{"XX"}, wantNoOut: true},
		{name: "marshal error: entry has no prefix of requested type", typ: TypeGeoLite2CountryMMDBOut, want: []string{"V6"}, only: lib.IPv4, wantErr: "has no prefix"},
		{name: "unsupported type", typ: "other", wantErr: lib.ErrNotSupportedFormat.Error()},
		{name: "mkdir fails", typ: TypeGeoLite2CountryMMDBOut, dir: filepath.Join(fileAsDir, "sub"), wantErr: "❌"},
		{name: "open file fails", typ: TypeGeoLite2CountryMMDBOut, outName: ".", wantErr: "❌"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.dir
			if dir == "" {
				dir = t.TempDir()
			}
			name := tt.outName
			if name == "" {
				name = "o.mmdb"
			}
			g := &geolite2_country_mmdb_out{Type: tt.typ, OutputName: name, OutputDir: dir, Want: tt.want, OnlyIPType: tt.only, SourceMMDBURI: tt.src}
			err := g.Output(containerWith(t, data))
			if tt.wantErr != "" {
				assertErrContains(t, err, tt.wantErr)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantNoOut {
				if _, err := os.Stat(filepath.Join(dir, name)); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("expected no output file, stat err=%v", err)
				}
			}
		})
	}
}

func TestMarshalDataUnsupportedWithSource(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		src  string
	}{
		{"no source", "other", ""},
		{"with source", "other", "something.mmdb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := mmdbwriter.New(mmdbwriter.Options{})
			if err != nil {
				t.Fatal(err)
			}
			e := lib.NewEntry("CN")
			if err := e.AddPrefix("1.0.0.0/24"); err != nil {
				t.Fatal(err)
			}
			g := &geolite2_country_mmdb_out{Type: tt.typ, SourceMMDBURI: tt.src}
			if err := g.marshalData(w, e, nil); !errors.Is(err, lib.ErrNotSupportedFormat) {
				t.Fatalf("got %v", err)
			}
		})
	}
}
