package maxmind

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/oschwald/geoip2-golang/v2"
	"github.com/oschwald/maxminddb-golang/v2"
)

func TestNewGeoLite2CountryMMDBOut(t *testing.T) {
	t.Parallel()

	testMMDBOutConstructors(t, findMMDBOutputType(t, TypeGeoLite2CountryMMDBOut))
}

// TestMMDBOut_Config checks that every MMDB output type can be created from
// a config file and run through lib.Instance.
func TestMMDBOut_Config(t *testing.T) {
	t.Parallel()

	for _, typ := range mmdbOutputTypes() {
		t.Run(typ.iType+"/valid args", func(t *testing.T) {
			t.Parallel()

			outputDir := t.TempDir()
			instance, err := newInstanceFromConfig(t, nil, []map[string]any{{
				"type": typ.iType,
				"args": map[string]any{"outputDir": outputDir, "outputName": "config.mmdb", "wantedList": []string{"cn"}},
			}})
			assertError(t, err, false, "", nil)

			container := newContainer(t, map[string][]string{
				"CN": {"1.0.0.0/24", "2402:4e00::/32"},
				"US": {"2.0.0.0/16"},
			})
			assertError(t, instance.RunOutput(container), false, "", nil)

			networks, metadata := readMMDBNetworks(t, filepath.Join(outputDir, "config.mmdb"))
			if metadata.DatabaseType != typ.dbType {
				t.Fatalf("DatabaseType = %q, want %q", metadata.DatabaseType, typ.dbType)
			}
			assertMMDBNetworks(t, networks, map[string]map[string]any{
				"1.0.0.0/24":     typ.plainRecord("CN"),
				"2402:4e00::/32": typ.plainRecord("CN"),
			})
		})

		t.Run(typ.iType+"/invalid args", func(t *testing.T) {
			t.Parallel()

			_, err := newInstanceFromConfig(t, nil, []map[string]any{{
				"type": typ.iType,
				"args": []string{"invalid"},
			}})
			assertError(t, err, true, "[type "+typ.iType+" | action output]", nil)
		})
	}
}

// TestMMDBOut_OutputWithoutSource checks the database written by every MMDB
// output type when no source MMDB is specified, and reads it back with the
// corresponding input type.
func TestMMDBOut_OutputWithoutSource(t *testing.T) {
	t.Parallel()

	lists := map[string][]string{
		"CN": {"1.0.0.0/23", "2402:4e00::/32"},
		"US": {"2.0.0.0/16", "2400:cb00::/32"},
		"DE": {"2001:db8::/32"},
		"IT": {"3.0.0.0/16"},
	}

	for _, typ := range mmdbOutputTypes() {
		t.Run(typ.iType, func(t *testing.T) {
			t.Parallel()

			outputDir := filepath.Join(t.TempDir(), "nested", "dir")
			converter, err := typ.newOut(lib.ActionOutput, WithOutputDir(outputDir), WithOutputName("out.mmdb"))
			assertError(t, err, false, "", nil)

			assertError(t, converter.Output(newContainer(t, lists)), false, "", nil)

			outputFile := filepath.Join(outputDir, "out.mmdb")
			networks, metadata := readMMDBNetworks(t, outputFile)

			if metadata.DatabaseType != typ.dbType {
				t.Errorf("DatabaseType = %q, want %q", metadata.DatabaseType, typ.dbType)
			}
			if want := map[string]string{"en": typ.dbDesc}; !reflect.DeepEqual(metadata.Description, want) {
				t.Errorf("Description = %v, want %v", metadata.Description, want)
			}
			if !slices.Equal(metadata.Languages, typ.languages) {
				t.Errorf("Languages = %v, want %v", metadata.Languages, typ.languages)
			}
			if metadata.RecordSize != typ.recordSize {
				t.Errorf("RecordSize = %d, want %d", metadata.RecordSize, typ.recordSize)
			}
			if metadata.IPVersion != 6 {
				t.Errorf("IPVersion = %d, want 6", metadata.IPVersion)
			}

			assertMMDBNetworks(t, networks, map[string]map[string]any{
				"1.0.0.0/23":     typ.plainRecord("CN"),
				"2402:4e00::/32": typ.plainRecord("CN"),
				"2.0.0.0/16":     typ.plainRecord("US"),
				"2400:cb00::/32": typ.plainRecord("US"),
				"2001:db8::/32":  typ.plainRecord("DE"),
				"3.0.0.0/16":     typ.plainRecord("IT"),
			})

			// Round trip: the written database must be readable by the
			// corresponding input type and produce the same lists.
			input, err := typ.newIn(lib.ActionAdd, WithURI(outputFile))
			assertError(t, err, false, "", nil)
			container, err := input.Input(lib.NewContainer())
			assertError(t, err, false, "", nil)
			assertContainer(t, container, lists)
		})
	}
}

func TestGeoLite2CountryMMDBOut_Output(t *testing.T) {
	t.Parallel()

	// List A overlaps with list B, so the order of writing matters.
	lists := map[string][]string{
		"A": {"1.0.0.0/23", "2001:db8::/32"},
		"B": {"1.0.0.0/24"},
		"C": {"3.0.0.0/16", "2400:cb00::/32"},
	}

	tests := []struct {
		name      string
		opts      []lib.OutputOption
		container map[string][]string // nil means lists
		want      map[string]string   // network -> list name; nil means no file is written
	}{
		{
			name: "all lists in sorted order",
			want: map[string]string{
				"1.0.0.0/24":     "B",
				"1.0.1.0/24":     "A",
				"2001:db8::/32":  "A",
				"3.0.0.0/16":     "C",
				"2400:cb00::/32": "C",
			},
		},
		{
			name: "overwrite list is written at last",
			opts: []lib.OutputOption{WithOverwriteList([]string{" a ", ""})},
			want: map[string]string{
				"1.0.0.0/23":     "A",
				"2001:db8::/32":  "A",
				"3.0.0.0/16":     "C",
				"2400:cb00::/32": "C",
			},
		},
		{
			name: "overwrite list keeps its order",
			opts: []lib.OutputOption{WithOverwriteList([]string{"A", "B"})},
			want: map[string]string{
				"1.0.0.0/24":     "B",
				"1.0.1.0/24":     "A",
				"2001:db8::/32":  "A",
				"3.0.0.0/16":     "C",
				"2400:cb00::/32": "C",
			},
		},
		{
			name: "excluded overwrite list is not written",
			opts: []lib.OutputOption{WithOverwriteList([]string{"a"}), WithOutputExcludedList([]string{" A "})},
			want: map[string]string{
				"1.0.0.0/24":     "B",
				"3.0.0.0/16":     "C",
				"2400:cb00::/32": "C",
			},
		},
		{
			name: "excluded list is not written",
			opts: []lib.OutputOption{WithOutputExcludedList([]string{"b", "", "not-exist"})},
			want: map[string]string{
				"1.0.0.0/23":     "A",
				"2001:db8::/32":  "A",
				"3.0.0.0/16":     "C",
				"2400:cb00::/32": "C",
			},
		},
		{
			name: "wanted list",
			opts: []lib.OutputOption{WithOutputWantedList([]string{" c "})},
			want: map[string]string{
				"3.0.0.0/16":     "C",
				"2400:cb00::/32": "C",
			},
		},
		{
			name: "wanted list keeps its order",
			opts: []lib.OutputOption{WithOutputWantedList([]string{"b", "a"})},
			want: map[string]string{
				"1.0.0.0/23":    "A",
				"2001:db8::/32": "A",
			},
		},
		{
			name: "wanted list has a higher priority than overwrite list",
			opts: []lib.OutputOption{WithOutputWantedList([]string{"a", "b"}), WithOverwriteList([]string{"a", "c"})},
			want: map[string]string{
				"1.0.0.0/24":    "B",
				"1.0.1.0/24":    "A",
				"2001:db8::/32": "A",
			},
		},
		{
			name: "excluded wanted list is not written",
			opts: []lib.OutputOption{WithOutputWantedList([]string{"a", "c"}), WithOutputExcludedList([]string{"c"})},
			want: map[string]string{
				"1.0.0.0/23":    "A",
				"2001:db8::/32": "A",
			},
		},
		{
			name: "wanted list not found in container is skipped",
			opts: []lib.OutputOption{WithOutputWantedList([]string{"not-exist", "c"})},
			want: map[string]string{
				"3.0.0.0/16":     "C",
				"2400:cb00::/32": "C",
			},
		},
		{
			name: "only IPv4",
			opts: []lib.OutputOption{WithOutputOnlyIPType(lib.IPv4)},
			want: map[string]string{
				"1.0.0.0/24": "B",
				"1.0.1.0/24": "A",
				"3.0.0.0/16": "C",
			},
		},
		{
			name: "only IPv6",
			opts: []lib.OutputOption{WithOutputOnlyIPType(lib.IPv6), WithOutputWantedList([]string{"A", "C"})},
			want: map[string]string{
				"2001:db8::/32":  "A",
				"2400:cb00::/32": "C",
			},
		},
		{
			name: "no file is written when wanted lists are not found",
			opts: []lib.OutputOption{WithOutputWantedList([]string{"not-exist"})},
		},
		{
			name: "no file is written when all wanted lists are excluded",
			opts: []lib.OutputOption{WithOutputWantedList([]string{"a"}), WithOutputExcludedList([]string{"A"})},
		},
		{
			name:      "no file is written for empty container",
			container: map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			outputDir := t.TempDir()
			opts := append([]lib.OutputOption{WithOutputDir(outputDir)}, tt.opts...)
			converter, err := NewGeoLite2CountryMMDBOut(lib.ActionOutput, opts...)
			assertError(t, err, false, "", nil)

			containerLists := tt.container
			if containerLists == nil {
				containerLists = lists
			}
			assertError(t, converter.Output(newContainer(t, containerLists)), false, "", nil)

			outputFile := filepath.Join(outputDir, defaultGeoLite2CountryMMDBOutputName)
			if tt.want == nil {
				assertFileNotExist(t, outputFile)
				return
			}

			want := make(map[string]map[string]any, len(tt.want))
			for network, name := range tt.want {
				want[network] = isoCodeRecord(name)
			}
			networks, _ := readMMDBNetworks(t, outputFile)
			assertMMDBNetworks(t, networks, want)
		})
	}
}

func TestGeoLite2CountryMMDBOut_OutputWithSource(t *testing.T) {
	t.Parallel()

	sourceFile := writeTempFile(t, "source.mmdb", buildMMDB(t, geoLite2StyleRecords()...))

	cn := map[string]any{
		"continent": map[string]any{
			"code":       "AS",
			"geoname_id": uint64(6255147),
			"names":      namesRecord(geoLite2Languages, map[string]string{"en": "Asia", "zh-CN": "亚洲"}),
		},
		"country": map[string]any{
			"iso_code":             "CN",
			"geoname_id":           uint64(1814991),
			"is_in_european_union": false,
			"names":                namesRecord(geoLite2Languages, map[string]string{"en": "China", "de": "China"}),
		},
	}
	us := map[string]any{
		"continent": map[string]any{
			"code":       "NA",
			"geoname_id": uint64(6255149),
			"names":      namesRecord(geoLite2Languages, map[string]string{"en": "North America"}),
		},
		"country": map[string]any{
			"iso_code":             "US",
			"geoname_id":           uint64(6252001),
			"is_in_european_union": false,
			"names":                namesRecord(geoLite2Languages, map[string]string{"en": "United States"}),
		},
	}
	it := map[string]any{
		"continent": map[string]any{
			"code":       "EU",
			"geoname_id": uint64(6255148),
			"names":      namesRecord(geoLite2Languages, map[string]string{"en": "Europe"}),
		},
		"country": map[string]any{
			"iso_code":             "IT",
			"geoname_id":           uint64(3175395),
			"is_in_european_union": true,
			"names":                namesRecord(geoLite2Languages, map[string]string{"en": "Italy"}),
		},
	}
	de := map[string]any{
		"country": map[string]any{
			"iso_code":             "DE",
			"geoname_id":           uint64(2921044),
			"is_in_european_union": true,
			"names":                namesRecord(geoLite2Languages, map[string]string{"en": "Germany"}),
		},
	}

	outputDir := t.TempDir()
	converter, err := NewGeoLite2CountryMMDBOut(lib.ActionOutput,
		WithOutputDir(outputDir),
		WithSourceMMDBURI(sourceFile),
	)
	assertError(t, err, false, "", nil)

	container := newContainer(t, map[string][]string{
		"CN": {"1.0.0.0/24", "2402:4e00::/32"},
		"US": {"2.0.0.0/16"},
		"IT": {"3.0.0.0/16"},
		"DE": {"2001:db8::/32"},
		"XX": {"5.0.0.0/16"}, // no extra info in source MMDB
	})
	assertError(t, converter.Output(container), false, "", nil)

	networks, _ := readMMDBNetworks(t, filepath.Join(outputDir, defaultGeoLite2CountryMMDBOutputName))
	assertMMDBNetworks(t, networks, map[string]map[string]any{
		"1.0.0.0/24":     cn,
		"2402:4e00::/32": cn,
		"2.0.0.0/16":     us,
		"3.0.0.0/16":     it,
		"2001:db8::/32":  de,
		"5.0.0.0/16":     isoCodeRecord("XX"),
	})
}

func TestGeoLite2CountryMMDBOut_OutputErrors(t *testing.T) {
	t.Parallel()

	regularFile := writeTempFile(t, "regular-file", []byte("content"))

	tests := []struct {
		name        string
		iType       string
		opts        func(t *testing.T) []lib.OutputOption
		lists       map[string][]string
		errContains string
		target      error
	}{
		{
			name:  "source MMDB not found",
			iType: TypeGeoLite2CountryMMDBOut,
			opts: func(t *testing.T) []lib.OutputOption {
				return []lib.OutputOption{WithSourceMMDBURI(filepath.Join(t.TempDir(), "missing.mmdb"))}
			},
			lists:       map[string][]string{"CN": {"1.0.0.0/24"}},
			errContains: "[type " + TypeGeoLite2CountryMMDBOut + " | action output]",
			target:      os.ErrNotExist,
		},
		{
			name:  "list has no prefix of the only IP type",
			iType: TypeDBIPCountryMMDBOut,
			opts: func(t *testing.T) []lib.OutputOption {
				return []lib.OutputOption{WithOutputOnlyIPType(lib.IPv6)}
			},
			lists:       map[string][]string{"CN": {"1.0.0.0/24"}},
			errContains: "entry CN has no prefix",
		},
		{
			name:        "network can not be inserted into aliased network",
			iType:       TypeIPInfoCountryMMDBOut,
			lists:       map[string][]string{"CN": {"2002:100::/24"}},
			errContains: "aliased network",
		},
		{
			name:  "output dir can not be created",
			iType: TypeGeoLite2CountryMMDBOut,
			opts: func(t *testing.T) []lib.OutputOption {
				return []lib.OutputOption{WithOutputDir(filepath.Join(regularFile, "sub"))}
			},
			lists:       map[string][]string{"CN": {"1.0.0.0/24"}},
			errContains: "[type " + TypeGeoLite2CountryMMDBOut + " | action output]",
		},
		{
			name:        "unsupported type",
			iType:       "unknown",
			lists:       map[string][]string{"CN": {"1.0.0.0/24"}},
			errContains: "[type unknown | action output]",
			target:      lib.ErrNotSupportedFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			outputDir := t.TempDir()
			opts := []lib.OutputOption{WithOutputDir(outputDir)}
			if tt.opts != nil {
				opts = append(opts, tt.opts(t)...)
			}
			converter, err := newGeoLite2CountryMMDBOut(tt.iType, "desc", lib.ActionOutput, opts...)
			assertError(t, err, false, "", nil)

			err = converter.Output(newContainer(t, tt.lists))
			assertError(t, err, true, tt.errContains, tt.target)
			assertFileNotExist(t, filepath.Join(outputDir, defaultGeoLite2CountryMMDBOutputName))
		})
	}

	t.Run("unsupported type with empty container", func(t *testing.T) {
		t.Parallel()

		outputDir := t.TempDir()
		g := &geolite2_country_mmdb_out{Type: "unknown", Action: lib.ActionOutput, OutputDir: outputDir, OutputName: "out.mmdb"}
		assertError(t, g.Output(lib.NewContainer()), false, "", nil)
		assertFileNotExist(t, filepath.Join(outputDir, "out.mmdb"))
	})
}

func TestGeoLite2CountryMMDBOut_filterAndSortList(t *testing.T) {
	t.Parallel()

	container := newContainer(t, map[string][]string{
		"CN": {"1.0.0.0/24"},
		"US": {"2.0.0.0/24"},
		"JP": {"3.0.0.0/24"},
		"DE": {"4.0.0.0/24"},
	})

	tests := []struct {
		name      string
		want      []string
		overwrite []string
		exclude   []string
		expected  []string
	}{
		{
			name:     "all lists sorted",
			expected: []string{"CN", "DE", "JP", "US"},
		},
		{
			name:     "excluded lists are removed",
			exclude:  []string{" cn ", "", "not-exist", "jp"},
			expected: []string{"DE", "US"},
		},
		{
			name:     "wanted lists keep order and are normalized",
			want:     []string{" us ", "", "cn", "not-exist"},
			expected: []string{"US", "CN", "NOT-EXIST"},
		},
		{
			name:     "excluded wanted lists are removed",
			want:     []string{"us", "cn"},
			exclude:  []string{"US"},
			expected: []string{"CN"},
		},
		{
			name:     "all wanted lists are excluded",
			want:     []string{"us", "cn"},
			exclude:  []string{"US", "CN"},
			expected: []string{},
		},
		{
			name:     "blank wanted lists are ignored",
			want:     []string{"", "   "},
			expected: []string{"CN", "DE", "JP", "US"},
		},
		{
			name:      "wanted lists have a higher priority than overwrite lists",
			want:      []string{"jp"},
			overwrite: []string{"cn"},
			expected:  []string{"JP"},
		},
		{
			name:      "overwrite lists are placed at last in order",
			overwrite: []string{" us ", "", "cn", "not-exist"},
			expected:  []string{"DE", "JP", "US", "CN", "NOT-EXIST"},
		},
		{
			name:      "excluded overwrite lists are removed",
			overwrite: []string{"us", "cn"},
			exclude:   []string{"cn", "de"},
			expected:  []string{"JP", "US"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := &geolite2_country_mmdb_out{Want: tt.want, Overwrite: tt.overwrite, Exclude: tt.exclude}
			got := g.filterAndSortList(container)
			if got == nil {
				t.Fatal("filterAndSortList() returned nil")
			}
			if len(got) != len(tt.expected) || !slices.Equal(got, tt.expected) {
				t.Fatalf("filterAndSortList() = %v (len %d), want %v (len %d)", got, len(got), tt.expected, len(tt.expected))
			}
		})
	}
}

func TestGeoLite2CountryMMDBOut_marshalData_GeoLite2(t *testing.T) {
	t.Parallel()

	names := geoip2.Names{
		German:              "de-name",
		English:             "en-name",
		Spanish:             "es-name",
		French:              "fr-name",
		Japanese:            "ja-name",
		BrazilianPortuguese: "pt-BR-name",
		Russian:             "ru-name",
		SimplifiedChinese:   "zh-CN-name",
	}
	wantNames := map[string]any{
		"de":    "de-name",
		"en":    "en-name",
		"es":    "es-name",
		"fr":    "fr-name",
		"ja":    "ja-name",
		"pt-BR": "pt-BR-name",
		"ru":    "ru-name",
		"zh-CN": "zh-CN-name",
	}

	extraInfo := map[string]any{
		"CN": geoip2.Country{
			Continent: geoip2.Continent{Code: "AS", GeoNameID: 1, Names: names},
			Country:   geoip2.CountryRecord{ISOCode: "should-be-ignored", GeoNameID: 2, IsInEuropeanUnion: true, Names: names},
		},
		"DE": geoip2.Country{
			Country: geoip2.CountryRecord{GeoNameID: 3, Names: names},
		},
		"BAD": "not a geoip2.Country",
	}

	tests := []struct {
		name  string
		entry *lib.Entry
		want  map[string]map[string]any
	}{
		{
			name:  "extra info with continent",
			entry: newEntry(t, "CN", "1.0.0.0/24", "2402:4e00::/32"),
			want: map[string]map[string]any{
				"1.0.0.0/24": {
					"continent": map[string]any{"code": "AS", "geoname_id": uint64(1), "names": wantNames},
					"country":   map[string]any{"iso_code": "CN", "geoname_id": uint64(2), "is_in_european_union": true, "names": wantNames},
				},
				"2402:4e00::/32": {
					"continent": map[string]any{"code": "AS", "geoname_id": uint64(1), "names": wantNames},
					"country":   map[string]any{"iso_code": "CN", "geoname_id": uint64(2), "is_in_european_union": true, "names": wantNames},
				},
			},
		},
		{
			name:  "extra info without continent",
			entry: newEntry(t, "DE", "2001:db8::/32"),
			want: map[string]map[string]any{
				"2001:db8::/32": {
					"country": map[string]any{"iso_code": "DE", "geoname_id": uint64(3), "is_in_european_union": false, "names": wantNames},
				},
			},
		},
		{
			name:  "extra info not found",
			entry: newEntry(t, "XX", "5.0.0.0/16"),
			want:  map[string]map[string]any{"5.0.0.0/16": isoCodeRecord("XX")},
		},
		{
			name:  "extra info of wrong type",
			entry: newEntry(t, "BAD", "6.0.0.0/16"),
			want:  map[string]map[string]any{"6.0.0.0/16": isoCodeRecord("BAD")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, SourceMMDBURI: "source.mmdb"}
			assertMMDBNetworks(t, marshalAndRead(t, g, tt.entry, extraInfo), tt.want)
		})
	}
}

func TestGeoLite2CountryMMDBOut_marshalData_OnlyIPType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		onlyIPType lib.IPType
		want       []string
	}{
		{name: "both IP types", onlyIPType: "", want: []string{"1.0.0.0/24", "2402:4e00::/32"}},
		{name: "only IPv4", onlyIPType: lib.IPv4, want: []string{"1.0.0.0/24"}},
		{name: "only IPv6", onlyIPType: lib.IPv6, want: []string{"2402:4e00::/32"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, OnlyIPType: tt.onlyIPType}
			networks := marshalAndRead(t, g, newEntry(t, "CN", "1.0.0.0/24", "2402:4e00::/32"), nil)

			want := make(map[string]map[string]any, len(tt.want))
			for _, network := range tt.want {
				want[network] = isoCodeRecord("CN")
			}
			assertMMDBNetworks(t, networks, want)
		})
	}
}

func TestGeoLite2CountryMMDBOut_marshalData_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		g           *geolite2_country_mmdb_out
		entry       *lib.Entry
		errContains string
		target      error
	}{
		{
			name:   "unsupported type without source",
			g:      &geolite2_country_mmdb_out{Type: "unknown"},
			entry:  newEntry(t, "CN", "1.0.0.0/24"),
			target: lib.ErrNotSupportedFormat,
		},
		{
			name:   "unsupported type with source",
			g:      &geolite2_country_mmdb_out{Type: "unknown", SourceMMDBURI: "source.mmdb"},
			entry:  newEntry(t, "CN", "1.0.0.0/24"),
			target: lib.ErrNotSupportedFormat,
		},
		{
			name:        "entry without prefix",
			g:           &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut},
			entry:       lib.NewEntry("CN"),
			errContains: "entry CN has no prefix",
		},
		{
			name:        "entry without prefix of the only IP type",
			g:           &geolite2_country_mmdb_out{Type: TypeIPInfoCountryMMDBOut, OnlyIPType: lib.IPv4},
			entry:       newEntry(t, "CN", "2402:4e00::/32"),
			errContains: "entry CN has no prefix",
		},
		{
			name:        "insert into aliased network",
			g:           &geolite2_country_mmdb_out{Type: TypeDBIPCountryMMDBOut},
			entry:       newEntry(t, "CN", "2002::/24"),
			errContains: "aliased network",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			writer, err := mmdbwriter.New(mmdbwriter.Options{IncludeReservedNetworks: true})
			if err != nil {
				t.Fatalf("mmdbwriter.New() error = %v", err)
			}

			err = tt.g.marshalData(writer, tt.entry, map[string]any{})
			assertError(t, err, true, tt.errContains, tt.target)
		})
	}
}

func TestGeoLite2CountryMMDBOut_writeFile(t *testing.T) {
	t.Parallel()

	newWriter := func(t *testing.T) *mmdbwriter.Tree {
		t.Helper()

		writer, err := mmdbwriter.New(mmdbwriter.Options{
			DatabaseType: "Test-DB",
			Description:  map[string]string{"en": "Test database"},
		})
		if err != nil {
			t.Fatalf("mmdbwriter.New() error = %v", err)
		}
		if err := writer.Insert(mustParseIPNet(t, "1.0.0.0/24"), mmdbtype.String("CN")); err != nil {
			t.Fatalf("writer.Insert() error = %v", err)
		}
		return writer
	}

	t.Run("creates nested output dir and truncates existing file", func(t *testing.T) {
		t.Parallel()

		outputDir := filepath.Join(t.TempDir(), "a", "b")
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			t.Fatal(err)
		}
		outputFile := filepath.Join(outputDir, "out.mmdb")
		if err := os.WriteFile(outputFile, make([]byte, 1<<20), 0644); err != nil {
			t.Fatal(err)
		}

		g := &geolite2_country_mmdb_out{Type: TypeGeoLite2CountryMMDBOut, OutputDir: filepath.Join(outputDir, "c")}
		assertError(t, g.writeFile("out.mmdb", newWriter(t)), false, "", nil)

		g.OutputDir = outputDir
		assertError(t, g.writeFile("out.mmdb", newWriter(t)), false, "", nil)

		for _, file := range []string{filepath.Join(outputDir, "c", "out.mmdb"), outputFile} {
			content, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			db, err := maxminddb.OpenBytes(content)
			if err != nil {
				t.Fatalf("maxminddb.OpenBytes(%s) error = %v", file, err)
			}
			if err := db.Verify(); err != nil {
				t.Fatalf("db.Verify(%s) error = %v", file, err)
			}
			var got string
			if err := db.Lookup(mustParseAddr(t, "1.0.0.1")).Decode(&got); err != nil || got != "CN" {
				t.Fatalf("Lookup(1.0.0.1) = %q, %v; want %q", got, err, "CN")
			}
			db.Close()
		}
	})

	t.Run("output dir can not be created", func(t *testing.T) {
		t.Parallel()

		regularFile := writeTempFile(t, "regular-file", []byte("content"))
		g := &geolite2_country_mmdb_out{OutputDir: filepath.Join(regularFile, "sub")}
		assertError(t, g.writeFile("out.mmdb", newWriter(t)), true, "", nil)
	})

	t.Run("output file can not be opened", func(t *testing.T) {
		t.Parallel()

		outputDir := t.TempDir()
		// A directory with the same name as the output file.
		if err := os.Mkdir(filepath.Join(outputDir, "out.mmdb"), 0755); err != nil {
			t.Fatal(err)
		}
		g := &geolite2_country_mmdb_out{OutputDir: outputDir}
		assertError(t, g.writeFile("out.mmdb", newWriter(t)), true, "", nil)
	})

	t.Run("output file can not be written", func(t *testing.T) {
		t.Parallel()

		const devFull = "/dev/full"
		if runtime.GOOS != "linux" {
			t.Skipf("%s is only available on linux", devFull)
		}
		if _, err := os.Stat(devFull); err != nil {
			t.Skipf("%s is not available: %v", devFull, err)
		}

		outputDir := t.TempDir()
		if err := os.Symlink(devFull, filepath.Join(outputDir, "out.mmdb")); err != nil {
			t.Skipf("can not create symlink: %v", err)
		}
		g := &geolite2_country_mmdb_out{OutputDir: outputDir}
		assertError(t, g.writeFile("out.mmdb", newWriter(t)), true, "no space left on device", nil)
	})
}
