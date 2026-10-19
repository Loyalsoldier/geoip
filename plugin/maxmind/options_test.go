package maxmind

import (
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestOutputDirOptionReuse(t *testing.T) {
	for _, dir := range []string{"", " \t", " custom-output "} {
		t.Run(dir, func(t *testing.T) {
			opt := WithOutputDir(dir)
			for _, tc := range []struct {
				name string
				new  func(lib.Action, ...lib.OutputOption) (lib.OutputConverter, error)
				want string
			}{
				{TypeGeoLite2CountryMMDBOut, NewGeoLite2CountryMMDBOut, defaultMaxmindOutputDir},
				{TypeDBIPCountryMMDBOut, NewDBIPCountryMMDBOut, defaultDBIPOutputDir},
				{TypeIPInfoCountryMMDBOut, NewIPInfoCountryMMDBOut, defaultIPInfoOutputDir},
			} {
				t.Run(tc.name, func(t *testing.T) {
					out, err := tc.new(lib.ActionOutput, opt)
					if err != nil {
						t.Fatal(err)
					}
					want := tc.want
					if strings.TrimSpace(dir) != "" {
						want = strings.TrimSpace(dir)
					}
					if got := out.(*geolite2_country_mmdb_out).OutputDir; got != want {
						t.Fatalf("outputDir = %q, want %q", got, want)
					}
				})
			}
		})
	}
}

func TestURIOptionReuse(t *testing.T) {
	for _, uri := range []string{"", " \t", " custom.mmdb "} {
		t.Run(uri, func(t *testing.T) {
			opt := WithURI(uri)
			for _, tc := range []struct {
				name string
				new  func(lib.Action, ...lib.InputOption) (lib.InputConverter, error)
				want string
			}{
				{TypeGeoLite2CountryMMDBIn, NewGeoLite2CountryMMDBIn, defaultGeoLite2CountryMMDBFile},
				{TypeDBIPCountryMMDBIn, NewDBIPCountryMMDBIn, defaultDBIPCountryMMDBFile},
				{TypeIPInfoCountryMMDBIn, NewIPInfoCountryMMDBIn, defaultIPInfoCountryMMDBFile},
			} {
				t.Run(tc.name, func(t *testing.T) {
					in, err := tc.new(lib.ActionAdd, opt)
					if err != nil {
						t.Fatal(err)
					}
					want := tc.want
					if strings.TrimSpace(uri) != "" {
						want = strings.TrimSpace(uri)
					}
					if got := in.(*geolite2_country_mmdb_in).URI; got != want {
						t.Fatalf("uri = %q, want %q", got, want)
					}
				})
			}
		})
	}
}
