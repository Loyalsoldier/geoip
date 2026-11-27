package maxmind

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/oschwald/geoip2-golang/v2"
	"github.com/oschwald/maxminddb-golang/v2"
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
		name     string
		newFn    func(lib.Action, ...lib.OutputOption) (lib.OutputConverter, error)
		opts     []lib.OutputOption
		wantType string
		wantDesc string
		wantName string
		wantDir  string
		wantSrc  string
	}{
		{"maxmind default", NewGeoLite2CountryMMDBOut, nil, TypeGeoLite2CountryMMDBOut, DescGeoLite2CountryMMDBOut, defaultGeoLite2CountryMMDBOutputName, defaultMaxmindOutputDir, ""},
		{"dbip blank options", NewDBIPCountryMMDBOut, []lib.OutputOption{nil, WithOutputName(" "), WithOutputDir(" ")}, TypeDBIPCountryMMDBOut, DescDBIPCountryMMDBOut, defaultGeoLite2CountryMMDBOutputName, defaultDBIPOutputDir, ""},
		{"ipinfo custom", NewIPInfoCountryMMDBOut, []lib.OutputOption{WithOutputName(" a.mmdb "), WithOutputDir(" out "), WithSourceMMDBURI(" src ")}, TypeIPInfoCountryMMDBOut, DescIPInfoCountryMMDBOut, "a.mmdb", "out", "src"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := tt.newFn(lib.ActionOutput, tt.opts...)
			assertErr(t, err, false)
			g := conv.(*geolite2_country_mmdb_out)
			if g.GetType() != tt.wantType || g.GetDescription() != tt.wantDesc || g.GetAction() != lib.ActionOutput {
				t.Errorf("unexpected meta %+v", g)
			}
			if g.OutputName != tt.wantName || g.OutputDir != tt.wantDir || g.SourceMMDBURI != tt.wantSrc {
				t.Errorf("got %+v", g)
			}
		})
	}
}

func TestNewMMDBOutFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		newFn   func(lib.Action, []byte) (lib.OutputConverter, error)
		data    string
		want    geolite2_country_mmdb_out
		wantErr bool
	}{
		{"maxmind empty", NewGeoLite2CountryMMDBOutFromBytes, "", geolite2_country_mmdb_out{OutputName: defaultGeoLite2CountryMMDBOutputName, OutputDir: defaultMaxmindOutputDir}, false},
		{
			"dbip full", NewDBIPCountryMMDBOutFromBytes,
			`{"outputName":"x.mmdb","outputDir":"d","wantedList":["cn"],"overwriteList":["us"],"excludedList":["jp"],"onlyIPType":"ipv4","sourceMMDBURI":"s"}`,
			geolite2_country_mmdb_out{OutputName: "x.mmdb", OutputDir: "d", Want: []string{"cn"}, Overwrite: []string{"us"}, Exclude: []string{"jp"}, OnlyIPType: lib.IPv4, SourceMMDBURI: "s"},
			false,
		},
		{"ipinfo empty", NewIPInfoCountryMMDBOutFromBytes, "{}", geolite2_country_mmdb_out{OutputName: defaultGeoLite2CountryMMDBOutputName, OutputDir: defaultIPInfoOutputDir}, false},
		{"invalid", NewGeoLite2CountryMMDBOutFromBytes, "{", geolite2_country_mmdb_out{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := tt.newFn(lib.ActionOutput, []byte(tt.data))
			assertErr(t, err, tt.wantErr)
			if tt.wantErr {
				return
			}
			g := conv.(*geolite2_country_mmdb_out)
			if g.OutputName != tt.want.OutputName || g.OutputDir != tt.want.OutputDir || g.OnlyIPType != tt.want.OnlyIPType || g.SourceMMDBURI != tt.want.SourceMMDBURI ||
				!slices.Equal(g.Want, tt.want.Want) || !slices.Equal(g.Overwrite, tt.want.Overwrite) || !slices.Equal(g.Exclude, tt.want.Exclude) {
				t.Errorf("got %+v, want %+v", g, tt.want)
			}
		})
	}
}

func TestHasData(t *testing.T) {
	tests := []struct {
		name string
		d    dbipCountry
		want bool
	}{
		{"zero", dbipCountry{}, false},
		{"continent", dbipCountry{Continent: dbipContinent{Names: dbipLanguageNames{Persian: "x"}}}, true},
		{"represented", dbipCountry{RepresentedCountry: dbipRepresentedCountry{Type: "military"}}, true},
		{"country", dbipCountry{Country: dbipCountryRecord{ISOCode: "CN"}}, true},
		{"registered", dbipCountry{RegisteredCountry: dbipCountryRecord{ISOCode: "CN"}}, true},
		{"traits", dbipCountry{Traits: geoip2.CountryTraits{IsAnycast: true}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.HasData(); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func newTestContainer(t *testing.T, data map[string][]string) lib.Container {
	t.Helper()
	c := lib.NewContainer()
	for name, cidrs := range data {
		e := lib.NewEntry(name)
		for _, cidr := range cidrs {
			if err := e.AddPrefix(cidr); err != nil {
				t.Fatal(err)
			}
		}
		if err := c.Add(e); err != nil {
			t.Fatal(err)
		}
	}
	return c
}

func TestFilterAndSortList(t *testing.T) {
	c := newTestContainer(t, map[string][]string{
		"CN": {"1.0.0.0/24"}, "US": {"2.0.0.0/24"}, "JP": {"3.0.0.0/24"}, "KR": {"4.0.0.0/24"},
	})
	tests := []struct {
		name                     string
		want, overwrite, exclude []string
		expect                   []string
	}{
		{"all sorted", nil, nil, nil, []string{"CN", "JP", "KR", "US"}},
		{"wanted keeps order", []string{" us ", "", "cn", "jp"}, []string{"kr"}, []string{"JP"}, []string{"US", "CN"}},
		{"wanted all excluded", []string{"cn"}, nil, []string{"cn"}, []string{}},
		{"overwrite last", nil, []string{"cn", " ", "jp"}, []string{"us", "", "jp"}, []string{"KR", "CN"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_country_mmdb_out{Want: tt.want, Overwrite: tt.overwrite, Exclude: tt.exclude}
			got := g.filterAndSortList(c)
			if len(got) != len(tt.expect) || !slices.Equal(got, tt.expect) {
				t.Errorf("got %v, want %v", got, tt.expect)
			}
		})
	}
}

// readMMDB returns map[countryCode][]cidr of an output mmdb file.
func readMMDB(t *testing.T, file, iType string) map[string][]string {
	t.Helper()
	db, err := maxminddb.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	result := make(map[string][]string)
	for network := range db.Networks() {
		var code string
		if iType == TypeIPInfoCountryMMDBOut {
			var r ipInfoLite
			if err := network.Decode(&r); err != nil {
				t.Fatal(err)
			}
			code = r.CountryCode
		} else {
			var r geoip2.Country
			if err := network.Decode(&r); err != nil {
				t.Fatal(err)
			}
			code = r.Country.ISOCode
		}
		result[code] = append(result[code], network.Prefix().String())
	}
	return result
}

func maxmindSourceFixture(t *testing.T) []byte {
	names := func(en string) mmdbtype.Map {
		return mmdbtype.Map{
			"en": mmdbtype.String(en), "zh-CN": mmdbtype.String(en + "-zh"),
			"fa": mmdbtype.String(en + "-fa"), "ko": mmdbtype.String(en + "-ko"),
		}
	}
	return buildMMDB(t,
		mmdbRecord{"1.0.0.0/24", mmdbtype.Map{
			"continent": mmdbtype.Map{"code": mmdbtype.String("AS"), "geoname_id": mmdbtype.Uint32(1), "names": names("Asia")},
			"country":   mmdbtype.Map{"iso_code": mmdbtype.String("cn"), "geoname_id": mmdbtype.Uint32(2), "names": names("China")},
		}},
		mmdbRecord{"1.0.1.0/24", countryRec("country", "CN")},
		mmdbRecord{"2.0.0.0/24", mmdbtype.Map{"registered_country": mmdbtype.Map{"iso_code": mmdbtype.String("US"), "names": names("United States"), "is_in_european_union": mmdbtype.Bool(false)}}},
		mmdbRecord{"2.0.1.0/24", countryRec("registered_country", "US")},
		mmdbRecord{"3.0.0.0/24", mmdbtype.Map{"represented_country": mmdbtype.Map{"iso_code": mmdbtype.String("DE"), "names": names("Germany"), "is_in_european_union": mmdbtype.Bool(true)}}},
		mmdbRecord{"3.0.1.0/24", countryRec("represented_country", "DE")},
		mmdbRecord{"4.0.0.0/24", mmdbtype.Map{"continent": mmdbtype.Map{"code": mmdbtype.String("EU")}}},
	)
}

func ipinfoSourceFixture(t *testing.T) []byte {
	return buildMMDB(t,
		mmdbRecord{"1.0.0.0/24", mmdbtype.Map{
			"asn": mmdbtype.String("AS1"), "as_name": mmdbtype.String("n"), "as_domain": mmdbtype.String("d"),
			"continent": mmdbtype.String("Asia"), "continent_code": mmdbtype.String("AS"),
			"country": mmdbtype.String("China"), "country_code": mmdbtype.String("cn"),
		}},
		mmdbRecord{"1.0.1.0/24", mmdbtype.Map{"country_code": mmdbtype.String("CN"), "country": mmdbtype.String("Other")}},
		mmdbRecord{"2.0.0.0/24", mmdbtype.Map{"country_code": mmdbtype.String(" "), "asn": mmdbtype.String("AS2")}},
	)
}

func TestGetExtraInfo(t *testing.T) {
	dir := t.TempDir()
	maxmindSrc := writeFile(t, dir, "maxmind.mmdb", maxmindSourceFixture(t))
	ipinfoSrc := writeFile(t, dir, "ipinfo.mmdb", ipinfoSourceFixture(t))
	asnOnly := writeFile(t, dir, "asn.mmdb", buildMMDB(t, mmdbRecord{"1.0.0.0/24", mmdbtype.Map{"asn": mmdbtype.String("AS1")}}))
	emptySrc := writeFile(t, dir, "empty.mmdb", buildMMDB(t))
	garbage := writeFile(t, dir, "garbage.mmdb", []byte("garbage"))
	badCountry := writeFile(t, dir, "badcountry.mmdb", badCountryMMDB(t))
	badIPInfo := writeFile(t, dir, "badipinfo.mmdb", badIPInfoMMDB(t))
	badMetadata := writeFile(t, dir, "metadata.mmdb", unverifiedMMDB(t))
	srv := newServer(t, map[string][]byte{"/maxmind.mmdb": maxmindSourceFixture(t)})

	tests := []struct {
		name     string
		iType    string
		src      string
		wantErr  bool
		wantNil  bool
		wantKeys []string
	}{
		{name: "no source", iType: TypeGeoLite2CountryMMDBOut, src: "  ", wantNil: true},
		{name: "maxmind", iType: TypeGeoLite2CountryMMDBOut, src: maxmindSrc, wantKeys: []string{"CN", "DE", "US"}},
		{name: "maxmind remote", iType: TypeGeoLite2CountryMMDBOut, src: srv.URL + "/maxmind.mmdb", wantKeys: []string{"CN", "DE", "US"}},
		{name: "dbip", iType: TypeDBIPCountryMMDBOut, src: maxmindSrc, wantKeys: []string{"CN", "DE", "US"}},
		{name: "ipinfo", iType: TypeIPInfoCountryMMDBOut, src: ipinfoSrc, wantKeys: []string{"CN"}},
		{name: "ipinfo without countries", iType: TypeIPInfoCountryMMDBOut, src: asnOnly, wantErr: true},
		{name: "missing file", iType: TypeGeoLite2CountryMMDBOut, src: dir + "/none", wantErr: true},
		{name: "remote 404", iType: TypeGeoLite2CountryMMDBOut, src: srv.URL + "/none", wantErr: true},
		{name: "garbage", iType: TypeGeoLite2CountryMMDBOut, src: garbage, wantErr: true},
		{name: "invalid metadata", iType: TypeGeoLite2CountryMMDBOut, src: badMetadata, wantErr: true},
		{name: "empty", iType: TypeGeoLite2CountryMMDBOut, src: emptySrc, wantErr: true},
		{name: "unsupported type", iType: "unknown", src: maxmindSrc, wantErr: true},
		{name: "maxmind decode error", iType: TypeGeoLite2CountryMMDBOut, src: badCountry, wantErr: true},
		{name: "dbip decode error", iType: TypeDBIPCountryMMDBOut, src: badCountry, wantErr: true},
		{name: "ipinfo decode error", iType: TypeIPInfoCountryMMDBOut, src: badIPInfo, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &geolite2_country_mmdb_out{Type: tt.iType, SourceMMDBURI: tt.src}
			info, err := g.GetExtraInfo()
			assertErr(t, err, tt.wantErr)
			if tt.wantErr {
				return
			}
			if tt.wantNil {
				if info != nil {
					t.Errorf("expected nil, got %v", info)
				}
				return
			}
			keys := make([]string, 0, len(info))
			for k := range info {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			if len(keys) != len(tt.wantKeys) || !slices.Equal(keys, tt.wantKeys) {
				t.Errorf("keys = %v, want %v", keys, tt.wantKeys)
			}
		})
	}

	t.Run("maxmind content", func(t *testing.T) {
		info, err := (&geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: maxmindSrc}).GetExtraInfo()
		assertErr(t, err, false)
		cn := info["CN"].(geoip2.Country)
		if cn.Continent.Code != "AS" || cn.Country.Names.English != "China" || cn.Country.GeoNameID != 2 {
			t.Errorf("CN = %+v", cn)
		}
		de := info["DE"].(geoip2.Country)
		if de.Country.ISOCode != "DE" || !de.Country.IsInEuropeanUnion || de.Country.Names.English != "Germany" {
			t.Errorf("DE = %+v", de)
		}
	})

	t.Run("dbip content", func(t *testing.T) {
		info, err := (&geolite2_country_mmdb_out{Type: TypeDBIPCountryMMDBOut, SourceMMDBURI: maxmindSrc}).GetExtraInfo()
		assertErr(t, err, false)
		cn := info["CN"].(dbipCountry)
		if cn.Continent.Code != "AS" || cn.Country.ISOCode != "cn" || cn.Country.GeoNameID != 2 {
			t.Errorf("CN = %+v", cn)
		}
		de := info["DE"].(dbipCountry)
		if de.Country.ISOCode != "DE" || !de.Country.IsInEuropeanUnion {
			t.Errorf("DE = %+v", de)
		}
	})

	t.Run("ipinfo content", func(t *testing.T) {
		info, err := (&geolite2_country_mmdb_out{Type: TypeIPInfoCountryMMDBOut, SourceMMDBURI: ipinfoSrc}).GetExtraInfo()
		assertErr(t, err, false)
		want := ipInfoLite{Continent: "Asia", ContinentCode: "AS", Country: "China", CountryCode: "cn"}
		if got := info["CN"].(ipInfoLite); got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})
}

func TestMMDBOut_Output(t *testing.T) {
	srcDir := t.TempDir()
	maxmindSrc := writeFile(t, srcDir, "maxmind.mmdb", maxmindSourceFixture(t))
	ipinfoSrc := writeFile(t, srcDir, "ipinfo.mmdb", ipinfoSourceFixture(t))
	fileAsDir := writeFile(t, srcDir, "file", []byte("x"))

	defaultData := map[string][]string{
		"CN": {"1.0.0.0/24", "2001:db8::/32"},
		"US": {"2.0.0.0/24"},
		"DE": {"3.0.0.0/24"},
		"JP": {"5.0.0.0/24"},
	}
	overlap := map[string][]string{"CN": {"1.0.0.0/24"}, "US": {"1.0.0.0/24", "2.0.0.0/24"}}

	tests := []struct {
		name       string
		iType      string
		src        string
		want       []string
		overwrite  []string
		only       lib.IPType
		outputDir  string // if empty a temp dir is used
		outputName string
		wantErr    bool
		wantFile   bool
		data       map[string][]string
		expect     map[string][]string
	}{
		{
			name: "maxmind no source", iType: TypeGeoLite2CountryMMDBOut, wantFile: true,
			expect: map[string][]string{"CN": {"1.0.0.0/24", "2001:db8::/32"}, "US": {"2.0.0.0/24"}, "DE": {"3.0.0.0/24"}, "JP": {"5.0.0.0/24"}},
		},
		{
			name: "dbip no source ipv4", iType: TypeDBIPCountryMMDBOut, only: lib.IPv4, want: []string{"CN"}, wantFile: true,
			expect: map[string][]string{"CN": {"1.0.0.0/24"}},
		},
		{
			name: "ipinfo no source", iType: TypeIPInfoCountryMMDBOut, want: []string{"US", "CN"}, wantFile: true,
			expect: map[string][]string{"CN": {"1.0.0.0/24", "2001:db8::/32"}, "US": {"2.0.0.0/24"}},
		},
		{
			name: "maxmind with source", iType: TypeGeoLite2CountryMMDBOut, src: maxmindSrc, wantFile: true,
			expect: map[string][]string{"CN": {"1.0.0.0/24", "2001:db8::/32"}, "US": {"2.0.0.0/24"}, "DE": {"3.0.0.0/24"}, "JP": {"5.0.0.0/24"}},
		},
		{
			name: "dbip with source", iType: TypeDBIPCountryMMDBOut, src: maxmindSrc, wantFile: true,
			expect: map[string][]string{"CN": {"1.0.0.0/24", "2001:db8::/32"}, "US": {"2.0.0.0/24"}, "DE": {"3.0.0.0/24"}, "JP": {"5.0.0.0/24"}},
		},
		{
			name: "ipinfo with source", iType: TypeIPInfoCountryMMDBOut, src: ipinfoSrc, want: []string{"CN", "JP"}, wantFile: true,
			expect: map[string][]string{"CN": {"1.0.0.0/24", "2001:db8::/32"}, "JP": {"5.0.0.0/24"}},
		},
		{
			name: "overwrite wins shared prefix", iType: TypeGeoLite2CountryMMDBOut, data: overlap, overwrite: []string{"cn"}, wantFile: true,
			expect: map[string][]string{"CN": {"1.0.0.0/24"}, "US": {"2.0.0.0/24"}},
		},
		{
			name: "wanted order overrides overwrite", iType: TypeGeoLite2CountryMMDBOut, data: overlap, overwrite: []string{"cn"}, want: []string{"CN", "US"}, wantFile: true,
			expect: map[string][]string{"US": {"1.0.0.0/24", "2.0.0.0/24"}},
		},
		{name: "insert into aliased network", iType: TypeGeoLite2CountryMMDBOut, data: map[string][]string{"CN": {"2002:100::/24"}}, wantErr: true},
		{name: "wanted not found", iType: TypeGeoLite2CountryMMDBOut, want: []string{"XX"}, wantFile: false},
		{name: "unsupported type no source", iType: "unknown", wantErr: true},
		{name: "unsupported type with source", iType: "unknown", src: maxmindSrc, wantErr: true},
		{name: "bad source", iType: TypeGeoLite2CountryMMDBOut, src: srcDir + "/none", wantErr: true},
		{name: "marshal error", iType: TypeGeoLite2CountryMMDBOut, want: []string{"US"}, only: lib.IPv6, wantErr: true},
		{name: "mkdir error", iType: TypeGeoLite2CountryMMDBOut, outputDir: filepath.Join(fileAsDir, "sub"), wantErr: true},
		{name: "open file error", iType: TypeGeoLite2CountryMMDBOut, outputName: ".", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outDir := tt.outputDir
			if outDir == "" {
				outDir = t.TempDir()
			}
			name := tt.outputName
			if name == "" {
				name = "out.mmdb"
			}
			g := &geolite2_country_mmdb_out{
				Type: tt.iType, Action: lib.ActionOutput, OutputDir: outDir, OutputName: name,
				Want: tt.want, Overwrite: tt.overwrite, OnlyIPType: tt.only, SourceMMDBURI: tt.src,
			}
			data := tt.data
			if data == nil {
				data = defaultData
			}
			err := g.Output(newTestContainer(t, data))
			assertErr(t, err, tt.wantErr)
			if tt.wantErr {
				return
			}
			file := filepath.Join(outDir, name)
			if _, statErr := os.Stat(file); (statErr == nil) != tt.wantFile {
				t.Fatalf("file exists = %v, want %v", statErr == nil, tt.wantFile)
			}
			if !tt.wantFile {
				return
			}
			got := readMMDB(t, file, tt.iType)
			if len(got) != len(tt.expect) {
				t.Fatalf("got %v, want %v", got, tt.expect)
			}
			for code, cidrs := range tt.expect {
				assertCIDRs(t, got[code], cidrs)
			}
		})
	}
}

func TestMMDBOut_OutputExtraInfoContent(t *testing.T) {
	srcDir := t.TempDir()
	maxmindSrc := writeFile(t, srcDir, "maxmind.mmdb", maxmindSourceFixture(t))
	ipinfoSrc := writeFile(t, srcDir, "ipinfo.mmdb", ipinfoSourceFixture(t))
	c := newTestContainer(t, map[string][]string{"CN": {"1.0.0.0/24"}, "US": {"2.0.0.0/24"}, "DE": {"3.0.0.0/24"}})

	t.Run("maxmind", func(t *testing.T) {
		dir := t.TempDir()
		g := &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, OutputDir: dir, OutputName: "o.mmdb", SourceMMDBURI: maxmindSrc}
		assertErr(t, g.Output(c), false)
		db, err := maxminddb.Open(filepath.Join(dir, "o.mmdb"))
		assertErr(t, err, false)
		defer db.Close()
		var r geoip2.Country
		assertErr(t, db.Lookup(mustAddr("1.0.0.1")).Decode(&r), false)
		if r.Continent.Code != "AS" || r.Country.ISOCode != "CN" || r.Country.Names.SimplifiedChinese != "China-zh" {
			t.Errorf("CN = %+v", r)
		}
		var d geoip2.Country
		assertErr(t, db.Lookup(mustAddr("3.0.0.1")).Decode(&d), false)
		if d.Continent.Code != "" || d.Country.ISOCode != "DE" || !d.Country.IsInEuropeanUnion {
			t.Errorf("DE = %+v", d)
		}
	})

	t.Run("dbip", func(t *testing.T) {
		dir := t.TempDir()
		g := &geolite2_country_mmdb_out{Type: TypeDBIPCountryMMDBOut, OutputDir: dir, OutputName: "o.mmdb", SourceMMDBURI: maxmindSrc}
		assertErr(t, g.Output(c), false)
		db, err := maxminddb.Open(filepath.Join(dir, "o.mmdb"))
		assertErr(t, err, false)
		defer db.Close()
		checkNames := func(t *testing.T, result maxminddb.Result, field, english string) {
			t.Helper()
			var names map[string]string
			assertErr(t, result.DecodePath(&names, field, "names"), false)
			for lang, want := range map[string]string{"en": english, "zh-CN": english + "-zh", "fa": english + "-fa", "ko": english + "-ko"} {
				if got := names[lang]; got != want {
					t.Errorf("%s.names[%q] = %q, want %q", field, lang, got, want)
				}
			}
		}
		for _, tt := range []struct {
			sourceField, ip, code, name string
			geonameID                   uint
			inEU                        bool
		}{
			{"country", "1.0.0.1", "CN", "China", 2, false},
			{"registered_country", "2.0.0.1", "US", "United States", 0, false},
			{"represented_country", "3.0.0.1", "DE", "Germany", 0, true},
		} {
			t.Run(tt.sourceField, func(t *testing.T) {
				result := db.Lookup(mustAddr(tt.ip))
				var r geoip2.Country
				assertErr(t, result.Decode(&r), false)
				if r.Country.ISOCode != tt.code || r.Country.GeoNameID != tt.geonameID || r.Country.IsInEuropeanUnion != tt.inEU {
					t.Errorf("country = %+v", r.Country)
				}
				checkNames(t, result, "country", tt.name)
				if tt.code == "CN" {
					if r.Continent.Code != "AS" || r.Continent.GeoNameID != 1 {
						t.Errorf("continent = %+v", r.Continent)
					}
					checkNames(t, result, "continent", "Asia")
				}
			})
		}
	})

	t.Run("ipinfo", func(t *testing.T) {
		dir := t.TempDir()
		g := &geolite2_country_mmdb_out{Type: TypeIPInfoCountryMMDBOut, OutputDir: dir, OutputName: "o.mmdb", SourceMMDBURI: ipinfoSrc}
		assertErr(t, g.Output(c), false)
		db, err := maxminddb.Open(filepath.Join(dir, "o.mmdb"))
		assertErr(t, err, false)
		defer db.Close()
		var r ipInfoLite
		assertErr(t, db.Lookup(mustAddr("1.0.0.1")).Decode(&r), false)
		want := ipInfoLite{Continent: "Asia", ContinentCode: "AS", Country: "China", CountryCode: "CN"}
		if r != want {
			t.Errorf("got %+v, want %+v", r, want)
		}
	})
}

func TestDBIPLanguageNamesHasData(t *testing.T) {
	tests := []struct {
		name string
		n    dbipLanguageNames
		want bool
	}{
		{"zero", dbipLanguageNames{}, false},
		{"persian", dbipLanguageNames{Persian: "x"}, true},
		{"english", dbipLanguageNames{dbipBaseNames: dbipBaseNames{English: "x"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.n.HasData(); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarshalData_UnsupportedTypeWithSource(t *testing.T) {
	g := &geolite2_country_mmdb_out{Type: "unknown", SourceMMDBURI: "src"}
	e := lib.NewEntry("CN")
	if err := e.AddPrefix("1.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	if err := g.marshalData(nil, e, nil); err != lib.ErrNotSupportedFormat {
		t.Errorf("err = %v, want %v", err, lib.ErrNotSupportedFormat)
	}
}
