package maxmind

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

// TestDefaultPaths verifies the default files and directories that the
// documentation of the formats promises.
func TestDefaultPaths(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"geolite2 country mmdb", defaultGeoLite2CountryMMDBFile, filepath.Join("geolite2", "GeoLite2-Country.mmdb")},
		{"dbip country mmdb", defaultDBIPCountryMMDBFile, filepath.Join("db-ip", "dbip-country-lite.mmdb")},
		{"ipinfo country mmdb", defaultIPInfoCountryMMDBFile, filepath.Join("ipinfo", "country.mmdb")},
		{"geolite2 country codes", defaultGeoLite2CountryCodeFile, filepath.Join("geolite2", "GeoLite2-Country-Locations-en.csv")},
		{"geolite2 country ipv4 blocks", defaultGeoLite2CountryIPv4File, filepath.Join("geolite2", "GeoLite2-Country-Blocks-IPv4.csv")},
		{"geolite2 country ipv6 blocks", defaultGeoLite2CountryIPv6File, filepath.Join("geolite2", "GeoLite2-Country-Blocks-IPv6.csv")},
		{"geolite2 asn ipv4 blocks", defaultGeoLite2ASNCSVIPv4File, filepath.Join("geolite2", "GeoLite2-ASN-Blocks-IPv4.csv")},
		{"geolite2 asn ipv6 blocks", defaultGeoLite2ASNCSVIPv6File, filepath.Join("geolite2", "GeoLite2-ASN-Blocks-IPv6.csv")},
		{"output name", defaultGeoLite2CountryMMDBOutputName, "Country.mmdb"},
		{"maxmind output directory", defaultMaxmindOutputDir, filepath.Join("output", "maxmind")},
		{"dbip output directory", defaultDBIPOutputDir, filepath.Join("output", "db-ip")},
		{"ipinfo output directory", defaultIPInfoOutputDir, filepath.Join("output", "ipinfo")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestMMDBInputConstructors(t *testing.T) {
	tests := []struct {
		name       string
		typ        string
		desc       string
		defaultURI string
		newFn      func(lib.Action, ...lib.InputOption) (lib.InputConverter, error)
		fromBytes  func(lib.Action, []byte) (lib.InputConverter, error)
	}{
		{"geolite2", TypeGeoLite2CountryMMDBIn, DescGeoLite2CountryMMDBIn, defaultGeoLite2CountryMMDBFile, NewGeoLite2CountryMMDBIn, NewGeoLite2CountryMMDBInFromBytes},
		{"dbip", TypeDBIPCountryMMDBIn, DescDBIPCountryMMDBIn, defaultDBIPCountryMMDBFile, NewDBIPCountryMMDBIn, NewDBIPCountryMMDBInFromBytes},
		{"ipinfo", TypeIPInfoCountryMMDBIn, DescIPInfoCountryMMDBIn, defaultIPInfoCountryMMDBFile, NewIPInfoCountryMMDBIn, NewIPInfoCountryMMDBInFromBytes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaults := &geolite2_country_mmdb_in{Type: tt.typ, Action: lib.ActionAdd, Description: tt.desc, URI: tt.defaultURI}
			customized := &geolite2_country_mmdb_in{
				Type: tt.typ, Action: lib.ActionAdd, Description: tt.desc,
				URI: "./custom.mmdb", Want: map[string]bool{"CN": true}, OnlyIPType: lib.IPv4,
			}

			constructors := []struct {
				name string
				new  func() (lib.InputConverter, error)
				want *geolite2_country_mmdb_in
			}{
				{"options", func() (lib.InputConverter, error) { return tt.newFn(lib.ActionAdd) }, defaults},
				{
					"custom options",
					func() (lib.InputConverter, error) {
						return tt.newFn(lib.ActionAdd, WithURI("./custom.mmdb"), WithInputWantedList([]string{"cn"}), WithInputOnlyIPType(lib.IPv4))
					},
					customized,
				},
				{"bytes", func() (lib.InputConverter, error) { return tt.fromBytes(lib.ActionAdd, nil) }, defaults},
				{
					"custom bytes",
					func() (lib.InputConverter, error) {
						return tt.fromBytes(lib.ActionAdd, []byte(`{"uri": "./custom.mmdb", "wantedList": ["cn"], "onlyIPType": "ipv4"}`))
					},
					customized,
				},
			}

			for _, constructor := range constructors {
				t.Run(constructor.name, func(t *testing.T) {
					got, err := constructor.new()
					assertErr(t, err, nil)
					assertMMDBIn(t, got.(*geolite2_country_mmdb_in), constructor.want)

					if got.GetType() != tt.typ || got.GetAction() != lib.ActionAdd || got.GetDescription() != tt.desc {
						t.Fatalf("unexpected type %q, action %q or description %q", got.GetType(), got.GetAction(), got.GetDescription())
					}
				})
			}
		})
	}
}

func TestMMDBOutputConstructors(t *testing.T) {
	tests := []struct {
		name       string
		typ        string
		desc       string
		defaultDir string
		newFn      func(lib.Action, ...lib.OutputOption) (lib.OutputConverter, error)
		fromBytes  func(lib.Action, []byte) (lib.OutputConverter, error)
	}{
		{"geolite2", TypeGeoLite2CountryMMDBOut, DescGeoLite2CountryMMDBOut, defaultMaxmindOutputDir, NewGeoLite2CountryMMDBOut, NewGeoLite2CountryMMDBOutFromBytes},
		{"dbip", TypeDBIPCountryMMDBOut, DescDBIPCountryMMDBOut, defaultDBIPOutputDir, NewDBIPCountryMMDBOut, NewDBIPCountryMMDBOutFromBytes},
		{"ipinfo", TypeIPInfoCountryMMDBOut, DescIPInfoCountryMMDBOut, defaultIPInfoOutputDir, NewIPInfoCountryMMDBOut, NewIPInfoCountryMMDBOutFromBytes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaults := &geolite2_country_mmdb_out{
				Type: tt.typ, Action: lib.ActionOutput, Description: tt.desc,
				OutputName: defaultGeoLite2CountryMMDBOutputName, OutputDir: tt.defaultDir,
			}
			customized := &geolite2_country_mmdb_out{
				Type: tt.typ, Action: lib.ActionOutput, Description: tt.desc,
				OutputName: "custom.mmdb", OutputDir: "./dir", Want: []string{"cn"}, SourceMMDBURI: "./source.mmdb",
			}

			constructors := []struct {
				name string
				new  func() (lib.OutputConverter, error)
				want *geolite2_country_mmdb_out
			}{
				{"options", func() (lib.OutputConverter, error) { return tt.newFn(lib.ActionOutput) }, defaults},
				{
					"custom options",
					func() (lib.OutputConverter, error) {
						return tt.newFn(lib.ActionOutput, WithOutputName("custom.mmdb"), WithOutputDir("./dir"), WithOutputWantedList([]string{"cn"}), WithSourceMMDBURI("./source.mmdb"))
					},
					customized,
				},
				{"bytes", func() (lib.OutputConverter, error) { return tt.fromBytes(lib.ActionOutput, nil) }, defaults},
				{
					"custom bytes",
					func() (lib.OutputConverter, error) {
						return tt.fromBytes(lib.ActionOutput, []byte(`{"outputName": "custom.mmdb", "outputDir": "./dir", "wantedList": ["cn"], "sourceMMDBURI": "./source.mmdb"}`))
					},
					customized,
				},
			}

			for _, constructor := range constructors {
				t.Run(constructor.name, func(t *testing.T) {
					got, err := constructor.new()
					assertErr(t, err, nil)
					assertMMDBOut(t, got.(*geolite2_country_mmdb_out), constructor.want)

					if got.GetType() != tt.typ || got.GetAction() != lib.ActionOutput || got.GetDescription() != tt.desc {
						t.Fatalf("unexpected type %q, action %q or description %q", got.GetType(), got.GetAction(), got.GetDescription())
					}
				})
			}
		})
	}
}

// TestRegisteredConverters converts the default files of each input format
// into the default file of an output format, as a configuration file that
// names no file does.
func TestRegisteredConverters(t *testing.T) {
	countryMMDB := buildMMDB(t, countryMMDBRecords(geoLite2Languages)...)
	ipinfoMMDB := buildMMDB(t, ipinfoMMDBRecords()...)

	// countryRecords returns the records expected for countryLists, given the
	// data of the networks of a list.
	countryRecords := func(data func(list string) mmdbtype.Map) []mmdbRecord {
		return []mmdbRecord{
			{"1.0.1.0/24", data("CN")},
			{"1.0.2.0/23", data("CN")},
			{"5.1.0.0/16", data("DE")},
			{"6.0.0.0/8", data("US")},
			{"8.8.8.0/24", data("US")},
			{"2001:250::/35", data("CN")},
			{"2001:4860::/32", data("US")},
		}
	}

	tests := []struct {
		name  string
		input string
		// args are the arguments of the input, if any.
		args   string
		output string
		// files are the default files of the input, by path.
		files map[string][]byte
		// outputDir is the default directory of the output.
		outputDir string
		want      []mmdbRecord
	}{
		{
			name:   "geolite2 country csv to geolite2 mmdb",
			input:  TypeGeoLite2CountryCSVIn,
			output: TypeGeoLite2CountryMMDBOut,
			files: map[string][]byte{
				defaultGeoLite2CountryCodeFile: []byte(countryCodeCSV),
				defaultGeoLite2CountryIPv4File: []byte(countryIPv4CSV),
				defaultGeoLite2CountryIPv6File: []byte(countryIPv6CSV),
			},
			outputDir: defaultMaxmindOutputDir,
			want:      countryRecords(isoCodeRecord),
		},
		{
			name:   "geolite2 asn csv to geolite2 mmdb",
			input:  TypeGeoLite2ASNCSVIn,
			output: TypeGeoLite2CountryMMDBOut,
			files: map[string][]byte{
				defaultGeoLite2ASNCSVIPv4File: []byte(asnIPv4CSV),
				defaultGeoLite2ASNCSVIPv6File: []byte(asnIPv6CSV),
			},
			outputDir: defaultMaxmindOutputDir,
			want: []mmdbRecord{
				{"1.0.0.0/24", isoCodeRecord("AS13335")},
				{"1.0.4.0/22", isoCodeRecord("AS38803")},
				{"1.1.1.0/24", isoCodeRecord("AS13335")},
				{"8.8.8.0/24", isoCodeRecord("AS15169")},
				{"2001:4860::/32", isoCodeRecord("AS15169")},
				{"2606:4700::/32", isoCodeRecord("AS13335")},
			},
		},
		{
			// AS38803 has no IPv6 network, so that there is no list of it.
			name:   "ipv6 of geolite2 asn csv to geolite2 mmdb",
			input:  TypeGeoLite2ASNCSVIn,
			args:   `{"onlyIPType": "ipv6"}`,
			output: TypeGeoLite2CountryMMDBOut,
			files: map[string][]byte{
				defaultGeoLite2ASNCSVIPv4File: []byte(asnIPv4CSV),
				defaultGeoLite2ASNCSVIPv6File: []byte(asnIPv6CSV),
			},
			outputDir: defaultMaxmindOutputDir,
			want: []mmdbRecord{
				{"2001:4860::/32", isoCodeRecord("AS15169")},
				{"2606:4700::/32", isoCodeRecord("AS13335")},
			},
		},
		{
			name:      "geolite2 mmdb to geolite2 mmdb",
			input:     TypeGeoLite2CountryMMDBIn,
			output:    TypeGeoLite2CountryMMDBOut,
			files:     map[string][]byte{defaultGeoLite2CountryMMDBFile: countryMMDB},
			outputDir: defaultMaxmindOutputDir,
			want:      countryRecords(isoCodeRecord),
		},
		{
			name:      "dbip mmdb to dbip mmdb",
			input:     TypeDBIPCountryMMDBIn,
			output:    TypeDBIPCountryMMDBOut,
			files:     map[string][]byte{defaultDBIPCountryMMDBFile: countryMMDB},
			outputDir: defaultDBIPOutputDir,
			want:      countryRecords(isoCodeRecord),
		},
		{
			name:      "ipinfo mmdb to ipinfo mmdb",
			input:     TypeIPInfoCountryMMDBIn,
			output:    TypeIPInfoCountryMMDBOut,
			files:     map[string][]byte{defaultIPInfoCountryMMDBFile: ipinfoMMDB},
			outputDir: defaultIPInfoOutputDir,
			want:      countryRecords(countryCodeRecord),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The default files are relative to the working directory.
			t.Chdir(t.TempDir())
			for path, content := range tt.files {
				writeTestFile(t, ".", path, content)
			}

			instance, err := lib.NewInstance()
			if err != nil {
				t.Fatal(err)
			}
			args := tt.args
			if args == "" {
				args = "{}"
			}
			config := fmt.Sprintf(`{"input": [{"type": %q, "action": "add", "args": %s}], "output": [{"type": %q}]}`, tt.input, args, tt.output)
			if err := instance.InitConfigFromBytes([]byte(config)); err != nil {
				t.Fatalf("InitConfigFromBytes(): %v", err)
			}
			if err := instance.Run(); err != nil {
				t.Fatalf("Run(): %v", err)
			}

			content, err := os.ReadFile(filepath.Join(tt.outputDir, defaultGeoLite2CountryMMDBOutputName))
			if err != nil {
				t.Fatal(err)
			}
			_, got := readMMDB(t, content)
			assertMMDBRecords(t, got, tt.want)
		})
	}
}

// TestRegisteredConverterArguments verifies that the arguments of a
// configuration file reach the converters of the formats.
func TestRegisteredConverterArguments(t *testing.T) {
	tests := []struct {
		name string
		// config is a converter of a configuration file.
		config  string
		wantErr bool
	}{
		{"geolite2 mmdb input", `"input": [{"type": "maxmindMMDB", "action": "add", "args": {"uri": "./custom.mmdb"}}]`, false},
		{"geolite2 mmdb input with invalid arguments", `"input": [{"type": "maxmindMMDB", "action": "add", "args": {"uri": 1}}]`, true},
		{"dbip mmdb input", `"input": [{"type": "dbipCountryMMDB", "action": "remove", "args": {"wantedList": ["cn"]}}]`, false},
		{"dbip mmdb input with invalid arguments", `"input": [{"type": "dbipCountryMMDB", "action": "remove", "args": {"wantedList": "cn"}}]`, true},
		{"ipinfo mmdb input", `"input": [{"type": "ipinfoCountryMMDB", "action": "add", "args": {"onlyIPType": "ipv4"}}]`, false},
		{"ipinfo mmdb input with invalid arguments", `"input": [{"type": "ipinfoCountryMMDB", "action": "add", "args": {"onlyIPType": 4}}]`, true},
		{"geolite2 country csv input", `"input": [{"type": "maxmindGeoLite2CountryCSV", "action": "add", "args": {"country": "./country.csv"}}]`, false},
		{"geolite2 country csv input with invalid arguments", `"input": [{"type": "maxmindGeoLite2CountryCSV", "action": "add", "args": {"country": 1}}]`, true},
		{"geolite2 asn csv input", `"input": [{"type": "maxmindGeoLite2ASNCSV", "action": "add", "args": {"wantedList": {"cloudflare": ["AS13335"]}}}]`, false},
		{"geolite2 asn csv input with invalid arguments", `"input": [{"type": "maxmindGeoLite2ASNCSV", "action": "add", "args": {"wantedList": "AS13335"}}]`, true},
		{"geolite2 mmdb output", `"output": [{"type": "maxmindMMDB", "action": "output", "args": {"outputName": "custom.mmdb"}}]`, false},
		{"geolite2 mmdb output with invalid arguments", `"output": [{"type": "maxmindMMDB", "action": "output", "args": {"outputName": 1}}]`, true},
		{"dbip mmdb output", `"output": [{"type": "dbipCountryMMDB", "args": {"outputDir": "./dir"}}]`, false},
		{"dbip mmdb output with invalid arguments", `"output": [{"type": "dbipCountryMMDB", "args": {"outputDir": 1}}]`, true},
		{"ipinfo mmdb output", `"output": [{"type": "ipinfoCountryMMDB", "args": {"sourceMMDBURI": "./source.mmdb"}}]`, false},
		{"ipinfo mmdb output with invalid arguments", `"output": [{"type": "ipinfoCountryMMDB", "args": {"sourceMMDBURI": 1}}]`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance, err := lib.NewInstance()
			if err != nil {
				t.Fatal(err)
			}

			err = instance.InitConfigFromBytes([]byte("{" + tt.config + "}"))
			if (err != nil) != tt.wantErr {
				t.Fatalf("InitConfigFromBytes() error = %v, want an error: %v", err, tt.wantErr)
			}
		})
	}
}

func TestRegisteredConverterListings(t *testing.T) {
	tests := []struct {
		name    string
		list    func()
		formats map[string]string
	}{
		{"input", lib.ListInputConverter, map[string]string{
			TypeGeoLite2CountryMMDBIn: DescGeoLite2CountryMMDBIn,
			TypeDBIPCountryMMDBIn:     DescDBIPCountryMMDBIn,
			TypeIPInfoCountryMMDBIn:   DescIPInfoCountryMMDBIn,
			TypeGeoLite2CountryCSVIn:  DescGeoLite2CountryCSVIn,
			TypeGeoLite2ASNCSVIn:      DescGeoLite2ASNCSVIn,
		}},
		{"output", lib.ListOutputConverter, map[string]string{
			TypeGeoLite2CountryMMDBOut: DescGeoLite2CountryMMDBOut,
			TypeDBIPCountryMMDBOut:     DescDBIPCountryMMDBOut,
			TypeIPInfoCountryMMDBOut:   DescIPInfoCountryMMDBOut,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := captureStdout(t, tt.list)
			for name, desc := range tt.formats {
				if !strings.Contains(out, "  - "+name+" ("+desc+")\n") {
					t.Errorf("missing %s format %q (%s) in listing:\n%s", tt.name, name, desc, out)
				}
			}
		})
	}
}
