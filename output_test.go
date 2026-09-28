package main

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

type outputSelectionContainer struct {
	lib.Container
	requested []string
}

func (c *outputSelectionContainer) GetEntry(name string) (*lib.Entry, bool) {
	c.requested = append(c.requested, name)
	return nil, false
}

func TestOutputListSelection(t *testing.T) {
	formats := []string{
		"text", "clashRuleSet", "clashRuleSetClassical", "surgeRuleSet",
		"mihomoMRS", "singboxSRS", "v2rayGeoIPDat", "stdout",
		"maxmindMMDB", "dbipCountryMMDB", "ipinfoCountryMMDB",
	}
	tests := []struct {
		name      string
		want      []string
		exclude   []string
		overwrite []string
		expected  []string
		mmdbOrder []string
	}{
		{name: "all", expected: []string{"CN", "JP", "US"}},
		{name: "empty wanted list", want: []string{}, expected: []string{"CN", "JP", "US"}},
		{name: "blank wanted names", want: []string{"", " \t "}, expected: []string{"CN", "JP", "US"}},
		{name: "exclude only", exclude: []string{" cn "}, expected: []string{"JP", "US"}},
		{
			name: "wanted names and order", want: []string{" us ", "", "cn"},
			expected: []string{"CN", "US"}, mmdbOrder: []string{"US", "CN"},
		},
		{name: "all wanted excluded", want: []string{"cn"}, exclude: []string{" CN "}},
		{name: "multiple wanted excluded", want: []string{"cn", " jp "}, exclude: []string{"CN", "JP"}},
		{name: "missing wanted excluded", want: []string{"missing"}, exclude: []string{"MISSING"}},
		{
			name: "partially excluded", want: []string{"cn", "jp", "us"},
			exclude: []string{"JP"}, expected: []string{"CN", "US"},
		},
		{
			name: "overwrite order", overwrite: []string{"cn", "jp"},
			expected: []string{"CN", "JP", "US"}, mmdbOrder: []string{"US", "CN", "JP"},
		},
		{
			name: "wanted order overrides overwrite", want: []string{"us", "cn"}, overwrite: []string{"jp"},
			expected: []string{"CN", "US"}, mmdbOrder: []string{"US", "CN"},
		},
		{
			name: "excluded wanted does not fall back to overwrite",
			want: []string{"cn"}, exclude: []string{"cn"}, overwrite: []string{"us"},
		},
	}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					container := &outputSelectionContainer{Container: lib.NewContainer()}
					for _, name := range []string{"US", "CN", "JP"} {
						if err := container.Add(lib.NewEntry(name)); err != nil {
							t.Fatal(err)
						}
					}
					config, err := json.Marshal(map[string]any{
						"output": []any{map[string]any{
							"type": format,
							"args": map[string]any{
								"wantedList": tt.want, "excludedList": tt.exclude,
								"overwriteList": tt.overwrite, "outputDir": t.TempDir(),
							},
						}},
					})
					if err != nil {
						t.Fatal(err)
					}
					instance, err := lib.NewInstance()
					if err != nil {
						t.Fatal(err)
					}
					if err := instance.InitConfigFromBytes(config); err != nil {
						t.Fatal(err)
					}
					if err := instance.RunOutput(container); err != nil {
						t.Fatal(err)
					}
					expected := tt.expected
					if tt.mmdbOrder != nil && (format == "maxmindMMDB" || format == "dbipCountryMMDB" || format == "ipinfoCountryMMDB") {
						expected = tt.mmdbOrder
					}
					if !slices.Equal(container.requested, expected) {
						t.Fatalf("selected %v, want %v", container.requested, expected)
					}
				})
			}
		})
	}
}
