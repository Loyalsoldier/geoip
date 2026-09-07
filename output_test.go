package main

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

type outputSelectionContainer struct {
	lib.Container
	selected []string
}

func (c *outputSelectionContainer) GetEntry(name string) (*lib.Entry, bool) {
	c.selected = append(c.selected, name)
	return nil, false
}

func TestOutputListSelection(t *testing.T) {
	formats := []string{
		"text", "clashRuleSet", "clashRuleSetClassical", "surgeRuleSet",
		"stdout", "v2rayGeoIPDat", "mihomoMRS", "singboxSRS",
		"maxmindMMDB", "dbipCountryMMDB", "ipinfoCountryMMDB",
	}
	tests := []struct {
		name      string
		want      []string
		exclude   []string
		overwrite []string
		expected  []string
	}{
		{name: "all lists", expected: []string{"CN", "JP", "US"}},
		{name: "empty wanted list", want: []string{}, expected: []string{"CN", "JP", "US"}},
		{name: "blank wanted names", want: []string{"", " "}, expected: []string{"CN", "JP", "US"}},
		{name: "excluded list", exclude: []string{" cn "}, expected: []string{"JP", "US"}},
		{name: "explicit wanted list", want: []string{" cn "}, expected: []string{"CN"}},
		{name: "partially excluded", want: []string{"CN", " jp "}, exclude: []string{"cn"}, expected: []string{"JP"}},
		{name: "all wanted excluded", want: []string{" cn "}, exclude: []string{"CN"}},
		{name: "all wanted excluded with blanks", want: []string{"", " cn ", " "}, exclude: []string{"CN"}},
		{name: "all wanted excluded with overwrite", want: []string{"CN"}, exclude: []string{"cn"}, overwrite: []string{"US"}},
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
								"wantedList":    tt.want,
								"excludedList":  tt.exclude,
								"overwriteList": tt.overwrite,
								"outputDir":     t.TempDir(),
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
					if !slices.Equal(container.selected, tt.expected) {
						t.Errorf("selected lists = %v, want %v", container.selected, tt.expected)
					}
				})
			}
		})
	}
}
