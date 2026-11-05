package maxmind

import (
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestRegisteredConfigCreators(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{"GeoLite2 MMDB in", `{"input":[{"type":"` + TypeGeoLite2CountryMMDBIn + `","action":"add"}]}`},
		{"DBIP MMDB in", `{"input":[{"type":"` + TypeDBIPCountryMMDBIn + `","action":"add"}]}`},
		{"IPInfo MMDB in", `{"input":[{"type":"` + TypeIPInfoCountryMMDBIn + `","action":"add"}]}`},
		{"GeoLite2 ASN CSV in", `{"input":[{"type":"` + TypeGeoLite2ASNCSVIn + `","action":"add"}]}`},
		{"GeoLite2 country CSV in", `{"input":[{"type":"` + TypeGeoLite2CountryCSVIn + `","action":"add"}]}`},
		{"GeoLite2 MMDB out", `{"output":[{"type":"` + TypeGeoLite2CountryMMDBOut + `"}]}`},
		{"DBIP MMDB out", `{"output":[{"type":"` + TypeDBIPCountryMMDBOut + `"}]}`},
		{"IPInfo MMDB out", `{"output":[{"type":"` + TypeIPInfoCountryMMDBOut + `"}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance, err := lib.NewInstance()
			if err != nil {
				t.Fatalf("NewInstance() error = %v", err)
			}
			if err := instance.InitConfigFromBytes([]byte(tt.config)); err != nil {
				t.Errorf("InitConfigFromBytes() error = %v", err)
			}
		})
	}
}
