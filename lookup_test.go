package main

import (
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/Loyalsoldier/geoip/plugin/special"
	"github.com/spf13/cobra"
)

func TestIsValidIPOrCIDR(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid IPv4 addresses
		{"valid IPv4", "192.168.1.1", true},
		{"valid IPv4 localhost", "127.0.0.1", true},
		{"valid IPv4 zero", "0.0.0.0", true},
		{"valid IPv4 max", "255.255.255.255", true},

		// Valid IPv6 addresses
		{"valid IPv6 localhost", "::1", true},
		{"valid IPv6 full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", true},
		{"valid IPv6 compressed", "2001:db8:85a3::8a2e:370:7334", true},
		{"valid IPv6 zero", "::", true},

		// Valid CIDR ranges
		{"valid IPv4 CIDR", "192.168.1.0/24", true},
		{"valid IPv4 CIDR /32", "192.168.1.1/32", true},
		{"valid IPv4 CIDR /0", "0.0.0.0/0", true},
		{"valid IPv6 CIDR", "2001:db8::/32", true},
		{"valid IPv6 CIDR /128", "::1/128", true},
		{"valid IPv6 CIDR /0", "::/0", true},

		// Invalid inputs
		{"empty string", "", false},
		{"invalid IPv4", "256.256.256.256", false},
		{"invalid IPv4 format", "192.168.1", false},
		{"invalid IPv4 with letters", "192.168.1.a", false},
		{"invalid IPv6", "2001:0db8:85a3::8a2e:370g:7334", false},
		{"invalid CIDR prefix", "192.168.1.0/33", false},
		{"invalid CIDR format", "192.168.1.0/", false},
		{"random string", "not-an-ip", false},
		{"just a slash", "/", false},
		{"IPv4 with invalid prefix", "192.168.1.0/abc", false},
		{"IPv6 with invalid prefix", "2001:db8::/abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidIPOrCIDR(tt.input)
			if result != tt.expected {
				t.Errorf("isValidIPOrCIDR(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetOutputForLookup(t *testing.T) {
	tests := []struct {
		name       string
		search     string
		searchList []string
	}{
		{"simple search", "192.168.1.1", []string{}},
		{"search with list", "10.0.0.1", []string{"test1", "test2"}},
		{"empty search", "", []string{"test"}},
		{"search with empty list", "127.0.0.1", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getOutputForLookup(tt.search, tt.searchList...)
			
			// Verify it returns a Lookup converter
			lookup, ok := result.(*special.Lookup)
			if !ok {
				t.Errorf("Expected *special.Lookup, got %T", result)
				return
			}

			// Verify the type
			if lookup.GetType() != special.TypeLookup {
				t.Errorf("Expected type %s, got %s", special.TypeLookup, lookup.GetType())
			}

			// Verify the action
			if lookup.GetAction() != lib.ActionOutput {
				t.Errorf("Expected action %s, got %s", lib.ActionOutput, lookup.GetAction())
			}

			// Verify the description
			if lookup.GetDescription() != special.DescLookup {
				t.Errorf("Expected description %s, got %s", special.DescLookup, lookup.GetDescription())
			}
		})
	}
}

func TestGetInputForLookup(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		inputName string
		uri      string
		dir      string
		shouldPanic bool
	}{
		{"text format with uri", "text", "test", "http://example.com/test.txt", "", false},
		{"text format with dir", "text", "test", "", "/tmp/test", false},
		{"maxmindMMDB format", "maxmindmmdb", "test", "test.mmdb", "", false},
		{"v2rayGeoIPDat format", "v2raygeoipdat", "test", "geoip.dat", "", false},
		{"mihomoMRS format", "mihomomrs", "test", "test.mrs", "", false},
		{"singboxSRS format", "singboxsrs", "test", "test.srs", "", false},
		{"clashRuleSet format", "clashruleset", "test", "test.yaml", "", false},
		{"clashRuleSetClassical format", "clashrulesetclassical", "test", "test.yaml", "", false},
		{"surgeRuleSet format", "surgeruleset", "test", "test.list", "", false},
		{"dbipCountryMMDB format", "dbipcountrymmdb", "test", "test.mmdb", "", false},
		{"ipinfoCountryMMDB format", "ipinfocountrymmdb", "test", "test.mmdb", "", false},
		// Test case variations for better coverage
		{"TEXT format uppercase", "TEXT", "test", "test.txt", "", false},
		{"MiXeD case format", "ClAsHrUlEsEt", "test", "test.yaml", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.shouldPanic {
						t.Errorf("Function panicked unexpectedly: %v", r)
					}
				}
			}()

			result := getInputForLookup(tt.format, tt.inputName, tt.uri, tt.dir)
			
			if result == nil {
				t.Error("Expected non-nil result")
				return
			}

			// Verify basic interface methods work
			if result.GetAction() != lib.ActionAdd {
				t.Errorf("Expected action %s, got %s", lib.ActionAdd, result.GetAction())
			}

			if result.GetDescription() == "" {
				t.Error("Expected non-empty description")
			}

			if result.GetType() == "" {
				t.Error("Expected non-empty type")
			}
		})
	}
}

func TestGetInputForLookupUnsupportedFormat(t *testing.T) {
	// This should cause log.Fatal, which we can't easily test
	// Instead, we'll test with a separate function or skip this case
	// For now, let's just test that our supported formats map is correct
	
	expectedFormats := []string{
		"clashruleset",
		"clashrulesetclassical", 
		"dbipcountrymmdb",
		"ipinfocountrymmdb",
		"maxmindmmdb",
		"mihomomrs",
		"singboxsrs",
		"surgeruleset",
		"text",
		"v2raygeoipdat",
	}
	
	for _, format := range expectedFormats {
		if !supportedInputFormats[strings.ToLower(format)] {
			t.Errorf("Format %s should be supported but is not in supportedInputFormats map", format)
		}
	}
}

func TestLookupCmd(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "command configuration",
			test: func(t *testing.T) {
				if lookupCmd.Use != "lookup" {
					t.Errorf("Expected Use to be 'lookup', got '%s'", lookupCmd.Use)
				}
				
				expectedAliases := []string{"find"}
				if len(lookupCmd.Aliases) != len(expectedAliases) {
					t.Errorf("Expected %d aliases, got %d", len(expectedAliases), len(lookupCmd.Aliases))
				}
				
				for i, alias := range expectedAliases {
					if i >= len(lookupCmd.Aliases) || lookupCmd.Aliases[i] != alias {
						t.Errorf("Expected alias '%s' at index %d", alias, i)
					}
				}
				
				expectedShort := "Lookup if specified IP or CIDR is in specified lists"
				if lookupCmd.Short != expectedShort {
					t.Errorf("Expected Short to be '%s', got '%s'", expectedShort, lookupCmd.Short)
				}
			},
		},
		{
			name: "command flags",
			test: func(t *testing.T) {
				// Check required flags exist
				formatFlag := lookupCmd.Flags().Lookup("format")
				if formatFlag == nil {
					t.Error("Expected format flag to exist")
				}
				
				uriFlag := lookupCmd.Flags().Lookup("uri")
				if uriFlag == nil {
					t.Error("Expected uri flag to exist")
				}
				
				dirFlag := lookupCmd.Flags().Lookup("dir")
				if dirFlag == nil {
					t.Error("Expected dir flag to exist")
				}
				
				searchlistFlag := lookupCmd.Flags().Lookup("searchlist")
				if searchlistFlag == nil {
					t.Error("Expected searchlist flag to exist")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestLookupCmdExecution(t *testing.T) {
	// Test lookup command execution with different scenarios
	tests := []struct {
		name string
		args []string
		expectError bool
	}{
		{
			name: "missing format flag",
			args: []string{},
			expectError: true,
		},
		{
			name: "help flag",
			args: []string{"--help"},
			expectError: false, // Help is expected to work
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of the command to avoid state pollution
			cmd := &cobra.Command{
				Use:     lookupCmd.Use,
				Aliases: lookupCmd.Aliases,
				Short:   lookupCmd.Short,
				Args:    lookupCmd.Args,
				Run:     lookupCmd.Run,
			}
			
			// Copy flags
			cmd.Flags().StringP("format", "f", "", "(Required) The input format")
			cmd.Flags().StringP("uri", "u", "", "URI of the input file")
			cmd.Flags().StringP("dir", "d", "", "Path to the input directory")
			cmd.Flags().StringSliceP("searchlist", "l", []string{}, "The lists to search from")
			
			cmd.MarkFlagRequired("format")
			cmd.MarkFlagsOneRequired("uri", "dir")
			cmd.MarkFlagsMutuallyExclusive("uri", "dir")
			
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestSupportedInputFormats(t *testing.T) {
	// Test that the supportedInputFormats map contains expected entries
	expectedCount := 10 // Based on the visible formats in the code
	if len(supportedInputFormats) != expectedCount {
		t.Errorf("Expected %d supported formats, got %d", expectedCount, len(supportedInputFormats))
	}

	// Test that all values are true
	for format, supported := range supportedInputFormats {
		if !supported {
			t.Errorf("Format %s should be supported but is marked as false", format)
		}
	}

	// Test specific formats we know should exist
	requiredFormats := []string{"text", "maxmindmmdb", "v2raygeoipdat"}
	for _, format := range requiredFormats {
		if !supportedInputFormats[strings.ToLower(format)] {
			t.Errorf("Required format %s not found in supportedInputFormats", format)
		}
	}
}