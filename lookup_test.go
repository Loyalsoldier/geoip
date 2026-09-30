package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/Loyalsoldier/geoip/plugin/special"
)

func TestLookupConstructors(t *testing.T) {
	for format := range supportedInputFormats {
		t.Run(format, func(t *testing.T) {
			input := getInputForLookup(format, "true", "source", "")
			if input.GetAction() != lib.ActionAdd || input.GetDescription() == "" {
				t.Fatalf("invalid converter metadata: %#v", input)
			}
		})
	}
	for _, format := range []string{"text", "clashRuleSet", "clashRuleSetClassical", "surgeRuleSet", "mihomoMRS", "singboxSRS"} {
		t.Run(format+" directory", func(t *testing.T) {
			input := getInputForLookup(format, "true", "", t.TempDir())
			if input.GetAction() != lib.ActionAdd {
				t.Fatalf("invalid action: %s", input.GetAction())
			}
		})
	}
	out := getOutputForLookup("192.0.2.1", "test").(*special.Lookup)
	if out.Search != "192.0.2.1" || len(out.SearchList) != 1 || out.SearchList[0] != "test" {
		t.Fatalf("lookup options lost: %#v", out)
	}
}

func TestLookupTextDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("192.0.2.0/24\n"), 0600); err != nil {
		t.Fatal(err)
	}
	container := lib.NewContainer()
	input := getInputForLookup("text", "true", "", dir)
	if _, err := input.Input(container); err != nil {
		t.Fatal(err)
	}
	if _, found, err := container.Lookup("192.0.2.1", "test"); err != nil || !found {
		t.Fatalf("directory lookup: found=%v, err=%v", found, err)
	}
}

func TestMergeConstructors(t *testing.T) {
	input := getInputForMerge().(*special.Stdin)
	if input.Name != "temp" || input.GetAction() != lib.ActionAdd {
		t.Fatalf("merge input: %#v", input)
	}
	for _, ipType := range []string{"", "ipv4", "ipv6"} {
		output := getOutputForMerge(ipType).(*special.Stdout)
		if output.OnlyIPType != lib.IPType(ipType) || output.GetAction() != lib.ActionOutput {
			t.Fatalf("merge output: %#v", output)
		}
	}
}

func TestRegisteredConfigCreators(t *testing.T) {
	inputs := map[string]string{
		"clashRuleSet":              `{"name":"test","uri":"source"}`,
		"clashRuleSetClassical":     `{"name":"test","uri":"source"}`,
		"cutter":                    `{"wantedList":["test"]}`,
		"dbipCountryMMDB":           `{}`,
		"ipinfoCountryMMDB":         `{}`,
		"json":                      `{"name":"test","uri":"source","jsonPath":["prefixes"]}`,
		"maxmindGeoLite2ASNCSV":     `{}`,
		"maxmindGeoLite2CountryCSV": `{}`,
		"maxmindMMDB":               `{}`,
		"mihomoMRS":                 `{"inputDir":"rules"}`,
		"private":                   `{}`,
		"singboxSRS":                `{"inputDir":"rules"}`,
		"stdin":                     `{"name":"test"}`,
		"surgeRuleSet":              `{"name":"test","uri":"source"}`,
		"text":                      `{"name":"test","ipOrCIDR":["192.0.2.0/24"]}`,
		"v2rayGeoIPDat":             `{"uri":"source"}`,
		"test":                      `{}`,
	}
	for format, args := range inputs {
		t.Run("input "+format, func(t *testing.T) {
			action := "add"
			if format == "cutter" {
				action = "remove"
			}
			instance, err := lib.NewInstance()
			if err != nil {
				t.Fatal(err)
			}
			config, err := json.Marshal(map[string]any{"input": []any{
				map[string]any{"type": format, "action": action, "args": json.RawMessage(args)},
			}})
			if err != nil {
				t.Fatal(err)
			}
			if err := instance.InitConfigFromBytes(config); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, format := range []string{"clashRuleSet", "clashRuleSetClassical", "dbipCountryMMDB", "ipinfoCountryMMDB", "lookup", "maxmindMMDB", "mihomoMRS", "singboxSRS", "stdout", "surgeRuleSet", "text", "v2rayGeoIPDat"} {
		t.Run("output "+format, func(t *testing.T) {
			instance, err := lib.NewInstance()
			if err != nil {
				t.Fatal(err)
			}
			args := "{}"
			if format == "lookup" {
				args = `{"search":"192.0.2.1"}`
			}
			config, err := json.Marshal(map[string]any{"output": []any{
				map[string]any{"type": format, "args": json.RawMessage(args)},
			}})
			if err != nil {
				t.Fatal(err)
			}
			if err := instance.InitConfigFromBytes(config); err != nil {
				t.Fatal(err)
			}
		})
	}
}
