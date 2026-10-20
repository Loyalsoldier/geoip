package maxmind

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/oschwald/geoip2-golang/v2"
)

func TestDBIPModelsHasData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data interface{ HasData() bool }
		want bool
	}{
		{name: "language names/zero", data: dbipLanguageNames{}, want: false},
		{name: "language names/geoip2 names", data: dbipLanguageNames{Names: geoip2.Names{English: "Asia"}}, want: true},
		{name: "language names/persian", data: dbipLanguageNames{Persian: "آسیا"}, want: true},
		{name: "language names/korean", data: dbipLanguageNames{Korean: "아시아"}, want: true},

		{name: "continent/zero", data: dbipContinent{}, want: false},
		{name: "continent/code", data: dbipContinent{Continent: geoip2.Continent{Code: "AS"}}, want: true},
		{name: "continent/names", data: dbipContinent{Names: dbipLanguageNames{Korean: "아시아"}}, want: true},

		{name: "country record/zero", data: dbipCountryRecord{}, want: false},
		{name: "country record/iso code", data: dbipCountryRecord{CountryRecord: geoip2.CountryRecord{ISOCode: "CN"}}, want: true},
		{name: "country record/names", data: dbipCountryRecord{Names: dbipLanguageNames{Persian: "چین"}}, want: true},

		{name: "represented country/zero", data: dbipRepresentedCountry{}, want: false},
		{name: "represented country/type", data: dbipRepresentedCountry{RepresentedCountry: geoip2.RepresentedCountry{Type: "military"}}, want: true},
		{name: "represented country/names", data: dbipRepresentedCountry{Names: dbipLanguageNames{Names: geoip2.Names{German: "Italien"}}}, want: true},

		{name: "country/zero", data: dbipCountry{}, want: false},
		{name: "country/only traits network", data: dbipCountry{Traits: geoip2.CountryTraits{Network: mustParsePrefix(t, "1.0.0.0/24")}}, want: false},
		{name: "country/traits", data: dbipCountry{Traits: geoip2.CountryTraits{IsAnycast: true}}, want: true},
		{name: "country/continent", data: dbipCountry{Continent: dbipContinent{Continent: geoip2.Continent{GeoNameID: 1}}}, want: true},
		{name: "country/represented country", data: dbipCountry{RepresentedCountry: dbipRepresentedCountry{RepresentedCountry: geoip2.RepresentedCountry{ISOCode: "IT"}}}, want: true},
		{name: "country/country", data: dbipCountry{Country: dbipCountryRecord{CountryRecord: geoip2.CountryRecord{IsInEuropeanUnion: true}}}, want: true},
		{name: "country/registered country", data: dbipCountry{RegisteredCountry: dbipCountryRecord{Names: dbipLanguageNames{Korean: "미국"}}}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.data.HasData(); got != tt.want {
				t.Errorf("HasData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDefaultMMDBOutputDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		iType string
		want  string
	}{
		{name: "GeoLite2 country", iType: TypeGeoLite2CountryMMDBOut, want: filepath.Join("output", "maxmind")},
		{name: "DB-IP country", iType: TypeDBIPCountryMMDBOut, want: filepath.Join("output", "db-ip")},
		{name: "IPInfo country", iType: TypeIPInfoCountryMMDBOut, want: filepath.Join("output", "ipinfo")},
		{name: "unknown type", iType: "unknown", want: filepath.Join("output", "maxmind")},
		{name: "empty type", iType: "", want: filepath.Join("output", "maxmind")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := getDefaultMMDBOutputDir(tt.iType); got != tt.want {
				t.Errorf("getDefaultMMDBOutputDir(%q) = %q, want %q", tt.iType, got, tt.want)
			}
		})
	}
}

func TestNewGeoLite2CountryMMDBOutInternal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		iType  string
		iDesc  string
		action lib.Action
		opts   []lib.OutputOption
		want   *geolite2_country_mmdb_out
	}{
		{
			name:   "GeoLite2 defaults",
			iType:  TypeGeoLite2CountryMMDBOut,
			iDesc:  DescGeoLite2CountryMMDBOut,
			action: lib.ActionOutput,
			want: &geolite2_country_mmdb_out{
				Type:        TypeGeoLite2CountryMMDBOut,
				Action:      lib.ActionOutput,
				Description: DescGeoLite2CountryMMDBOut,
				OutputName:  defaultGeoLite2CountryMMDBOutputName,
				OutputDir:   defaultMaxmindOutputDir,
			},
		},
		{
			name:   "DB-IP defaults",
			iType:  TypeDBIPCountryMMDBOut,
			iDesc:  DescDBIPCountryMMDBOut,
			action: lib.ActionOutput,
			want: &geolite2_country_mmdb_out{
				Type:        TypeDBIPCountryMMDBOut,
				Action:      lib.ActionOutput,
				Description: DescDBIPCountryMMDBOut,
				OutputName:  defaultGeoLite2CountryMMDBOutputName,
				OutputDir:   defaultDBIPOutputDir,
			},
		},
		{
			name:   "IPInfo defaults",
			iType:  TypeIPInfoCountryMMDBOut,
			iDesc:  DescIPInfoCountryMMDBOut,
			action: lib.ActionOutput,
			want: &geolite2_country_mmdb_out{
				Type:        TypeIPInfoCountryMMDBOut,
				Action:      lib.ActionOutput,
				Description: DescIPInfoCountryMMDBOut,
				OutputName:  defaultGeoLite2CountryMMDBOutputName,
				OutputDir:   defaultIPInfoOutputDir,
			},
		},
		{
			name:   "unknown type defaults",
			iType:  "unknown",
			iDesc:  "desc",
			action: lib.ActionOutput,
			want: &geolite2_country_mmdb_out{
				Type:        "unknown",
				Action:      lib.ActionOutput,
				Description: "desc",
				OutputName:  defaultGeoLite2CountryMMDBOutputName,
				OutputDir:   defaultMaxmindOutputDir,
			},
		},
		{
			name:   "nil options are ignored and others are applied",
			iType:  TypeGeoLite2CountryMMDBOut,
			iDesc:  DescGeoLite2CountryMMDBOut,
			action: lib.ActionOutput,
			opts: []lib.OutputOption{
				nil,
				WithOutputName(" geo.mmdb "),
				WithOutputDir(" ./out "),
				nil,
				WithOutputWantedList([]string{"cn", " us "}),
				WithOverwriteList([]string{"jp"}),
				WithOutputExcludedList([]string{"private"}),
				WithOutputOnlyIPType(lib.IPv6),
				WithSourceMMDBURI("  ./source.mmdb "),
			},
			want: &geolite2_country_mmdb_out{
				Type:          TypeGeoLite2CountryMMDBOut,
				Action:        lib.ActionOutput,
				Description:   DescGeoLite2CountryMMDBOut,
				OutputName:    "geo.mmdb",
				OutputDir:     "./out",
				Want:          []string{"cn", " us "},
				Overwrite:     []string{"jp"},
				Exclude:       []string{"private"},
				OnlyIPType:    lib.IPv6,
				SourceMMDBURI: "./source.mmdb",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := newGeoLite2CountryMMDBOut(tt.iType, tt.iDesc, tt.action, tt.opts...)
			assertError(t, err, false, "", nil)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("newGeoLite2CountryMMDBOut() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestNewGeoLite2CountryMMDBOutFromBytesInternal(t *testing.T) {
	t.Parallel()

	types := []struct {
		iType      string
		iDesc      string
		defaultDir string
	}{
		{TypeGeoLite2CountryMMDBOut, DescGeoLite2CountryMMDBOut, defaultMaxmindOutputDir},
		{TypeDBIPCountryMMDBOut, DescDBIPCountryMMDBOut, defaultDBIPOutputDir},
		{TypeIPInfoCountryMMDBOut, DescIPInfoCountryMMDBOut, defaultIPInfoOutputDir},
	}

	tests := []struct {
		name    string
		data    []byte
		want    geolite2_country_mmdb_out // OutputDir empty means the default dir of the type
		wantErr bool
	}{
		{
			name: "nil data",
			data: nil,
			want: geolite2_country_mmdb_out{OutputName: defaultGeoLite2CountryMMDBOutputName},
		},
		{
			name: "empty JSON object",
			data: []byte(`{}`),
			want: geolite2_country_mmdb_out{OutputName: defaultGeoLite2CountryMMDBOutputName},
		},
		{
			name: "blank name and dir fall back to defaults",
			data: []byte(`{"outputName": "  ", "outputDir": "  ", "sourceMMDBURI": "   "}`),
			want: geolite2_country_mmdb_out{OutputName: defaultGeoLite2CountryMMDBOutputName},
		},
		{
			name: "all args",
			data: []byte(`{
				"outputName": " custom.mmdb ",
				"outputDir": " ./custom ",
				"wantedList": ["cn", "us"],
				"overwriteList": ["jp"],
				"excludedList": ["private"],
				"onlyIPType": "ipv4",
				"sourceMMDBURI": " https://example.com/source.mmdb "
			}`),
			want: geolite2_country_mmdb_out{
				OutputName:    "custom.mmdb",
				OutputDir:     "./custom",
				Want:          []string{"cn", "us"},
				Overwrite:     []string{"jp"},
				Exclude:       []string{"private"},
				OnlyIPType:    lib.IPv4,
				SourceMMDBURI: "https://example.com/source.mmdb",
			},
		},
		{
			name:    "invalid JSON",
			data:    []byte(`{"outputName"`),
			wantErr: true,
		},
		{
			name:    "invalid list type",
			data:    []byte(`{"wantedList": {"cn": true}}`),
			wantErr: true,
		},
	}

	for _, typ := range types {
		for _, tt := range tests {
			t.Run(typ.iType+"/"+tt.name, func(t *testing.T) {
				t.Parallel()

				got, err := newGeoLite2CountryMMDBOutFromBytes(typ.iType, typ.iDesc, lib.ActionOutput, tt.data)
				assertError(t, err, tt.wantErr, "[type "+typ.iType+" | action output]", nil)
				if tt.wantErr {
					if got != nil {
						t.Fatalf("expected nil converter on error, got %#v", got)
					}
					return
				}

				want := tt.want
				want.Type = typ.iType
				want.Action = lib.ActionOutput
				want.Description = typ.iDesc
				if want.OutputDir == "" {
					want.OutputDir = typ.defaultDir
				}
				if !reflect.DeepEqual(got, &want) {
					t.Fatalf("newGeoLite2CountryMMDBOutFromBytes() = %#v, want %#v", got, &want)
				}
			})
		}
	}
}

func TestOutputOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		start geolite2_country_mmdb_out
		opt   lib.OutputOption
		want  geolite2_country_mmdb_out
	}{
		{
			name:  "WithOutputName sets trimmed name",
			start: geolite2_country_mmdb_out{OutputName: "old"},
			opt:   WithOutputName("  new.mmdb\t"),
			want:  geolite2_country_mmdb_out{OutputName: "new.mmdb"},
		},
		{
			name:  "WithOutputName with empty name sets default",
			start: geolite2_country_mmdb_out{OutputName: "old"},
			opt:   WithOutputName("   "),
			want:  geolite2_country_mmdb_out{OutputName: defaultGeoLite2CountryMMDBOutputName},
		},
		{
			name:  "WithOutputDir sets trimmed dir",
			start: geolite2_country_mmdb_out{Type: TypeIPInfoCountryMMDBOut, OutputDir: "old"},
			opt:   WithOutputDir(" ./dir "),
			want:  geolite2_country_mmdb_out{Type: TypeIPInfoCountryMMDBOut, OutputDir: "./dir"},
		},
		{
			name:  "WithOutputDir with empty dir sets GeoLite2 default",
			start: geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, OutputDir: "old"},
			opt:   WithOutputDir(""),
			want:  geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, OutputDir: defaultMaxmindOutputDir},
		},
		{
			name:  "WithOutputDir with empty dir sets DB-IP default",
			start: geolite2_country_mmdb_out{Type: TypeDBIPCountryMMDBOut, OutputDir: "old"},
			opt:   WithOutputDir("  "),
			want:  geolite2_country_mmdb_out{Type: TypeDBIPCountryMMDBOut, OutputDir: defaultDBIPOutputDir},
		},
		{
			name:  "WithOutputDir with empty dir sets IPInfo default",
			start: geolite2_country_mmdb_out{Type: TypeIPInfoCountryMMDBOut, OutputDir: "old"},
			opt:   WithOutputDir(""),
			want:  geolite2_country_mmdb_out{Type: TypeIPInfoCountryMMDBOut, OutputDir: defaultIPInfoOutputDir},
		},
		{
			name:  "WithOutputWantedList keeps raw values",
			start: geolite2_country_mmdb_out{Want: []string{"old"}},
			opt:   WithOutputWantedList([]string{" cn ", ""}),
			want:  geolite2_country_mmdb_out{Want: []string{" cn ", ""}},
		},
		{
			name:  "WithOutputWantedList with nil list",
			start: geolite2_country_mmdb_out{Want: []string{"old"}},
			opt:   WithOutputWantedList(nil),
			want:  geolite2_country_mmdb_out{},
		},
		{
			name:  "WithOverwriteList keeps raw values",
			start: geolite2_country_mmdb_out{},
			opt:   WithOverwriteList([]string{"jp", " us"}),
			want:  geolite2_country_mmdb_out{Overwrite: []string{"jp", " us"}},
		},
		{
			name:  "WithOutputExcludedList keeps raw values",
			start: geolite2_country_mmdb_out{},
			opt:   WithOutputExcludedList([]string{"private "}),
			want:  geolite2_country_mmdb_out{Exclude: []string{"private "}},
		},
		{
			name:  "WithOutputOnlyIPType",
			start: geolite2_country_mmdb_out{OnlyIPType: lib.IPv4},
			opt:   WithOutputOnlyIPType(lib.IPv6),
			want:  geolite2_country_mmdb_out{OnlyIPType: lib.IPv6},
		},
		{
			name:  "WithSourceMMDBURI trims URI",
			start: geolite2_country_mmdb_out{},
			opt:   WithSourceMMDBURI(" ./source.mmdb\n"),
			want:  geolite2_country_mmdb_out{SourceMMDBURI: "./source.mmdb"},
		},
		{
			name:  "WithSourceMMDBURI with blank URI",
			start: geolite2_country_mmdb_out{SourceMMDBURI: "old"},
			opt:   WithSourceMMDBURI("   "),
			want:  geolite2_country_mmdb_out{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.start
			tt.opt(&got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestGeoLite2CountryMMDBOut_GetExtraInfo_GeoLite2(t *testing.T) {
	t.Parallel()

	content := buildMMDB(t, geoLite2StyleRecords()...)
	want := map[string]any{
		"CN": geoip2.Country{
			Continent: geoip2.Continent{
				Code:      "AS",
				GeoNameID: 6255147,
				Names:     geoip2.Names{English: "Asia", SimplifiedChinese: "亚洲"},
			},
			Country: geoip2.CountryRecord{
				ISOCode:   " cn ",
				GeoNameID: 1814991,
				Names:     geoip2.Names{English: "China", German: "China"},
			},
		},
		"US": geoip2.Country{
			Continent: geoip2.Continent{
				Code:      "NA",
				GeoNameID: 6255149,
				Names:     geoip2.Names{English: "North America"},
			},
			Country: geoip2.CountryRecord{
				ISOCode:   "US",
				GeoNameID: 6252001,
				Names:     geoip2.Names{English: "United States"},
			},
		},
		"IT": geoip2.Country{
			Continent: geoip2.Continent{
				Code:      "EU",
				GeoNameID: 6255148,
				Names:     geoip2.Names{English: "Europe"},
			},
			Country: geoip2.CountryRecord{
				ISOCode:           " it ",
				GeoNameID:         3175395,
				IsInEuropeanUnion: true,
				Names:             geoip2.Names{English: "Italy"},
			},
		},
		"DE": geoip2.Country{
			Country: geoip2.CountryRecord{
				ISOCode:           "DE",
				GeoNameID:         2921044,
				IsInEuropeanUnion: true,
				Names:             geoip2.Names{English: "Germany"},
			},
		},
	}

	for name, uri := range sourceURIs(t, content) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: uri}
			got, err := g.GetExtraInfo()
			assertError(t, err, false, "", nil)
			if len(got) != len(want) {
				t.Fatalf("GetExtraInfo() returned %d items %v, want %d items %v", len(got), sortedKeys(got), len(want), sortedKeys(want))
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("GetExtraInfo() = %#v, want %#v", got, want)
			}
		})
	}
}

// dbipSummary holds the fields of a dbipCountry extra info that are
// independent of localized names.
type dbipSummary struct {
	ContinentCode      string
	ContinentGeoNameID uint
	ISOCode            string
	GeoNameID          uint
	IsInEuropeanUnion  bool
}

func summarizeDBIP(t *testing.T, v any) dbipSummary {
	t.Helper()

	info, ok := v.(dbipCountry)
	if !ok {
		t.Fatalf("extra info has type %T, want dbipCountry", v)
	}
	if info.RegisteredCountry.HasData() || info.RepresentedCountry.HasData() || info.Traits.HasData() {
		t.Fatalf("extra info has unexpected data: %#v", info)
	}

	return dbipSummary{
		ContinentCode:      info.Continent.Code,
		ContinentGeoNameID: info.Continent.GeoNameID,
		ISOCode:            info.Country.ISOCode,
		GeoNameID:          info.Country.GeoNameID,
		IsInEuropeanUnion:  info.Country.IsInEuropeanUnion,
	}
}

func TestGeoLite2CountryMMDBOut_GetExtraInfo_DBIP(t *testing.T) {
	t.Parallel()

	content := buildMMDB(t, geoLite2StyleRecords()...)
	want := map[string]dbipSummary{
		"CN": {ContinentCode: "AS", ContinentGeoNameID: 6255147, ISOCode: " cn ", GeoNameID: 1814991},
		"US": {ContinentCode: "NA", ContinentGeoNameID: 6255149, ISOCode: "US", GeoNameID: 6252001},
		"IT": {ContinentCode: "EU", ContinentGeoNameID: 6255148, ISOCode: " it ", GeoNameID: 3175395, IsInEuropeanUnion: true},
		"DE": {ISOCode: "DE", GeoNameID: 2921044, IsInEuropeanUnion: true},
	}

	for name, uri := range sourceURIs(t, content) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := &geolite2_country_mmdb_out{Type: TypeDBIPCountryMMDBOut, SourceMMDBURI: uri}
			got, err := g.GetExtraInfo()
			assertError(t, err, false, "", nil)
			if len(got) != len(want) {
				t.Fatalf("GetExtraInfo() returned %d items %v, want %d items %v", len(got), sortedKeys(got), len(want), sortedKeys(want))
			}
			for code, wantSummary := range want {
				v, found := got[code]
				if !found {
					t.Fatalf("GetExtraInfo() has no info for %q", code)
				}
				if gotSummary := summarizeDBIP(t, v); gotSummary != wantSummary {
					t.Fatalf("GetExtraInfo()[%q] = %+v, want %+v", code, gotSummary, wantSummary)
				}
			}
		})
	}
}

func TestGeoLite2CountryMMDBOut_GetExtraInfo_IPInfo(t *testing.T) {
	t.Parallel()

	content := buildMMDB(t, ipInfoStyleRecords()...)
	want := map[string]any{
		"CN": ipInfoLite{Continent: "Asia", ContinentCode: "AS", Country: "China", CountryCode: " cn "},
		"US": ipInfoLite{Continent: "North America", ContinentCode: "NA", Country: "United States", CountryCode: "us"},
		"DE": ipInfoLite{Continent: "Europe", ContinentCode: "EU", Country: "Germany", CountryCode: "DE"},
	}

	for name, uri := range sourceURIs(t, content) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := &geolite2_country_mmdb_out{Type: TypeIPInfoCountryMMDBOut, SourceMMDBURI: uri}
			got, err := g.GetExtraInfo()
			assertError(t, err, false, "", nil)
			if len(got) != len(want) {
				t.Fatalf("GetExtraInfo() returned %d items %v, want %d items %v", len(got), sortedKeys(got), len(want), sortedKeys(want))
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("GetExtraInfo() = %#v, want %#v", got, want)
			}
		})
	}
}

func TestGeoLite2CountryMMDBOut_GetExtraInfo_Errors(t *testing.T) {
	t.Parallel()

	validContent := buildMMDB(t, geoLite2StyleRecords()...)
	validFile := writeTempFile(t, "valid.mmdb", validContent)
	emptyFile := writeTempFile(t, "empty.mmdb", buildMMDB(t))
	invalidFile := writeTempFile(t, "invalid.mmdb", []byte("this is not a mmdb file"))
	corruptedFile := writeTempFile(t, "corrupted.mmdb", corruptMMDB(t, validContent))
	missingFile := filepath.Join(t.TempDir(), "missing.mmdb")
	notFoundURL := newTestServer(t, http.StatusNotFound, []byte("not found")).URL + "/source.mmdb"
	invalidURL := newTestServer(t, http.StatusOK, []byte("not a mmdb")).URL + "/source.mmdb"
	brokenURL := failingServerURL(t)

	types := []string{TypeGeoLite2CountryMMDBOut, TypeDBIPCountryMMDBOut, TypeIPInfoCountryMMDBOut}

	type testCase struct {
		name        string
		iType       string
		uri         string
		wantNil     bool // expect nil info and nil error
		errContains string
		target      error
	}

	var tests []testCase
	for _, iType := range types {
		decodeErrorFile := writeTempFile(t, "decode-error.mmdb", buildMMDB(t, decodeErrorRecords(iType)...))

		tests = append(tests,
			testCase{name: "empty URI", iType: iType, uri: "", wantNil: true},
			testCase{name: "blank URI", iType: iType, uri: " \t ", wantNil: true},
			testCase{name: "missing file", iType: iType, uri: missingFile, target: os.ErrNotExist},
			testCase{name: "remote not found", iType: iType, uri: notFoundURL, errContains: "404"},
			testCase{name: "remote connection aborted", iType: iType, uri: brokenURL},
			testCase{name: "invalid local mmdb", iType: iType, uri: invalidFile},
			testCase{name: "invalid remote mmdb", iType: iType, uri: invalidURL},
			testCase{name: "corrupted mmdb", iType: iType, uri: corruptedFile},
			testCase{name: "record decode error", iType: iType, uri: decodeErrorFile, errContains: "cannot unmarshal"},
			testCase{name: "no extra info", iType: iType, uri: emptyFile, errContains: "no extra info found in the source MMDB file: " + emptyFile},
		)
	}

	tests = append(tests,
		testCase{name: "unsupported type", iType: "unknown", uri: validFile, target: lib.ErrNotSupportedFormat},
		testCase{name: "unsupported type with empty URI", iType: "unknown", uri: "", wantNil: true},
	)

	for _, tt := range tests {
		t.Run(tt.iType+"/"+tt.name, func(t *testing.T) {
			t.Parallel()

			g := &geolite2_country_mmdb_out{Type: tt.iType, SourceMMDBURI: tt.uri}
			got, err := g.GetExtraInfo()
			if got != nil {
				t.Fatalf("GetExtraInfo() = %v, want nil", got)
			}
			assertError(t, err, !tt.wantNil, tt.errContains, tt.target)
		})
	}
}

func TestGeoLite2CountryMMDBOut_GetExtraInfo_NoISOCode(t *testing.T) {
	t.Parallel()

	// Records without any ISO code produce no extra info for GeoLite2 and DB-IP types.
	content := buildMMDB(t,
		mmdbRecord{"4.0.0.0/16", mMap{"continent": mMap{"code": mStr("EU")}}},
		mmdbRecord{"5.0.0.0/16", mMap{}},
	)
	file := writeTempFile(t, "no-iso.mmdb", content)

	for _, iType := range []string{TypeGeoLite2CountryMMDBOut, TypeDBIPCountryMMDBOut} {
		t.Run(iType, func(t *testing.T) {
			t.Parallel()

			g := &geolite2_country_mmdb_out{Type: iType, SourceMMDBURI: file}
			got, err := g.GetExtraInfo()
			if got != nil {
				t.Fatalf("GetExtraInfo() = %v, want nil", got)
			}
			assertError(t, err, true, "no extra info found", nil)
			if errors.Is(err, lib.ErrNotSupportedFormat) {
				t.Fatalf("unexpected error type: %v", err)
			}
		})
	}
}
