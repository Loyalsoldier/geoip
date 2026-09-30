package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/Loyalsoldier/geoip/plugin/mihomo"
	"github.com/Loyalsoldier/geoip/plugin/singbox"
	"github.com/Loyalsoldier/geoip/plugin/special"
)

func TestLookupConstructors(t *testing.T) {
	for format := range supportedInputFormats {
		t.Run(format, func(t *testing.T) {
			input := getInputForLookup(format, "true", "input-file", "")
			if input.GetAction() != lib.ActionAdd || input.GetDescription() == "" {
				t.Fatalf("invalid lookup input metadata: %+v", input)
			}
		})
	}
	output := getOutputForLookup("192.0.2.1", "CN")
	if output.GetType() != special.TypeLookup || output.GetAction() != lib.ActionOutput {
		t.Fatalf("invalid lookup output metadata: %+v", output)
	}
}

func TestLookupDirectoryInputs(t *testing.T) {
	entry := lib.NewEntry("CN")
	if err := entry.AddPrefix("192.0.2.0/24"); err != nil {
		t.Fatal(err)
	}
	source := lib.NewContainer()
	if err := source.Add(entry); err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{
		"text", "clashRuleSet", "clashRuleSetClassical", "surgeRuleSet", "singboxSRS", "mihomoMRS",
	} {
		t.Run(format, func(t *testing.T) {
			dir := t.TempDir()
			switch format {
			case "singboxSRS":
				if err := singbox.NewSRSOut(lib.ActionOutput, singbox.WithOutputDir(dir)).Output(source); err != nil {
					t.Fatal(err)
				}
			case "mihomoMRS":
				if err := mihomo.NewMRSOut(lib.ActionOutput, mihomo.WithOutputDir(dir)).Output(source); err != nil {
					t.Fatal(err)
				}
			default:
				contents := "192.0.2.0/24\n"
				switch format {
				case "clashRuleSet":
					contents = "payload:\n  - '192.0.2.0/24'\n"
				case "clashRuleSetClassical":
					contents = "payload:\n  - IP-CIDR,192.0.2.0/24\n"
				case "surgeRuleSet":
					contents = "IP-CIDR,192.0.2.0/24\n"
				}
				if err := os.WriteFile(filepath.Join(dir, "cn.txt"), []byte(contents), 0600); err != nil {
					t.Fatal(err)
				}
			}

			input := getInputForLookup(format, "true", "", dir)
			result, err := input.Input(lib.NewContainer())
			if err != nil {
				t.Fatal(err)
			}
			_, found, err := result.Lookup("192.0.2.1", "CN")
			if err != nil || !found {
				t.Fatalf("directory lookup failed: found=%v, err=%v", found, err)
			}
		})
	}
}

func TestMergeConstructors(t *testing.T) {
	input := getInputForMerge()
	if input.GetType() != special.TypeStdin || input.GetAction() != lib.ActionAdd {
		t.Fatalf("invalid merge input metadata: %+v", input)
	}
	for _, ipType := range []string{"", "ipv4", "ipv6"} {
		output := getOutputForMerge(ipType)
		if output.GetType() != special.TypeStdout || output.GetAction() != lib.ActionOutput {
			t.Fatalf("invalid merge output metadata: %+v", output)
		}
	}
}
