package main

import (
	"testing"
)

func TestInitImports(t *testing.T) {
	// The init.go file contains only side-effect imports
	// We can't directly test import statements, but we can test that the imports
	// had their intended effect by checking if the plugins are registered
	
	// Since the imports are side-effect imports (with _), they should have
	// executed their init() functions which register the plugins
	
	// We can't directly test this without access to the plugin registry,
	// but we can test that the file structure is correct and doesn't cause compilation errors
	
	// The mere fact that this test compiles and runs means the imports are valid
	t.Log("All plugin imports are valid and don't cause compilation errors")
	
	// Test that this is a valid Go package
	// If the imports were invalid, the package wouldn't compile
	
	// List of expected plugins that should be imported
	expectedPlugins := []string{
		"maxmind",
		"mihomo", 
		"plaintext",
		"singbox",
		"special",
		"v2ray",
	}
	
	// We can't directly verify the imports without reflection or package introspection
	// But we can log what we expect to be imported
	for _, plugin := range expectedPlugins {
		t.Logf("Expected plugin import: github.com/Loyalsoldier/geoip/plugin/%s", plugin)
	}
}

func TestPackageStructure(t *testing.T) {
	// Test that the main package structure is correct
	// This test verifies that all the expected components exist
	
	// Check that key variables are defined
	if rootCmd == nil {
		t.Error("rootCmd should be defined")
	}
	
	if convertCmd == nil {
		t.Error("convertCmd should be defined") 
	}
	
	if listCmd == nil {
		t.Error("listCmd should be defined")
	}
	
	if lookupCmd == nil {
		t.Error("lookupCmd should be defined")
	}
	
	if mergeCmd == nil {
		t.Error("mergeCmd should be defined")
	}
	
	// Check that supportedInputFormats is defined and not empty
	if supportedInputFormats == nil {
		t.Error("supportedInputFormats should be defined")
	}
	
	if len(supportedInputFormats) == 0 {
		t.Error("supportedInputFormats should not be empty")
	}
}

func TestMainPackageInit(t *testing.T) {
	// Test that the init functions have been called and set up the commands correctly
	
	// All commands should be added to the root command
	commands := rootCmd.Commands()
	if len(commands) == 0 {
		t.Error("Root command should have subcommands after init")
	}
	
	expectedCommands := map[string]bool{
		"convert": false,
		"list":    false,
		"lookup":  false,
		"merge":   false,
	}
	
	for _, cmd := range commands {
		if _, exists := expectedCommands[cmd.Use]; exists {
			expectedCommands[cmd.Use] = true
		}
	}
	
	for cmdName, found := range expectedCommands {
		if !found {
			t.Errorf("Expected command '%s' not found in root command", cmdName)
		}
	}
}