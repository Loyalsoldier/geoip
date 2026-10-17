package maxmind

import (
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestMMDBOutputDirOptionReuse(t *testing.T) {
	formats := []struct {
		newConverter func(lib.Action, ...lib.OutputOption) lib.OutputConverter
		defaultDir   string
	}{
		{NewGeoLite2CountryMMDBOut, defaultMaxmindOutputDir},
		{NewDBIPCountryMMDBOut, defaultDBIPOutputDir},
		{NewIPInfoCountryMMDBOut, defaultIPInfoOutputDir},
	}
	for _, dir := range []string{"", " \t ", " custom-output "} {
		opt := WithMMDBOutputDir(dir)
		for _, format := range formats {
			converter := format.newConverter(lib.ActionOutput, opt).(*geoLite2CountryMMDBOut)
			want := strings.TrimSpace(dir)
			if want == "" {
				want = format.defaultDir
			}
			if converter.OutputDir != want {
				t.Errorf("%s with directory %q: got %q, want %q", converter.Type, dir, converter.OutputDir, want)
			}
		}
	}
}
