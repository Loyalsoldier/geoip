package main

import (
	"testing"
)

func TestConvertCmd(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "command configuration",
			test: func(t *testing.T) {
				if convertCmd.Use != "convert" {
					t.Errorf("Expected Use to be 'convert', got '%s'", convertCmd.Use)
				}
				
				expectedAliases := []string{"conv"}
				if len(convertCmd.Aliases) != len(expectedAliases) {
					t.Errorf("Expected %d aliases, got %d", len(expectedAliases), len(convertCmd.Aliases))
				}
				
				for i, alias := range expectedAliases {
					if i >= len(convertCmd.Aliases) || convertCmd.Aliases[i] != alias {
						t.Errorf("Expected alias '%s' at index %d", alias, i)
					}
				}
				
				expectedShort := "Convert geoip data from one format to another by using config file"
				if convertCmd.Short != expectedShort {
					t.Errorf("Expected Short to be '%s', got '%s'", expectedShort, convertCmd.Short)
				}
			},
		},
		{
			name: "command flags",
			test: func(t *testing.T) {
				// Check config flag exists as persistent flag
				configFlag := convertCmd.PersistentFlags().Lookup("config")
				if configFlag == nil {
					t.Error("Expected config flag to exist")
				}
				
				// Check that the flag has correct shorthand
				if configFlag.Shorthand != "c" {
					t.Errorf("Expected shorthand 'c', got '%s'", configFlag.Shorthand)
				}
				
				// Check default value
				expectedDefault := "config.json"
				if configFlag.DefValue != expectedDefault {
					t.Errorf("Expected default value '%s', got '%s'", expectedDefault, configFlag.DefValue)
				}
			},
		},
		{
			name: "command has run function",
			test: func(t *testing.T) {
				if convertCmd.Run == nil {
					t.Error("Expected convert command to have a Run function")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestConvertCmdExecution(t *testing.T) {
	// Test that the convert command structure is correct
	// We can't easily test the full execution without mocking the lib package and file system
	// But we can test that the command is properly configured
	
	// Test that the command is added to root command
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "convert" {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Convert command should be added to root command")
	}
}

func TestConvertInit(t *testing.T) {
	// Test that the init function properly sets up the command
	// The init function should have:
	// 1. Added the command to rootCmd
	// 2. Set up the config flag
	
	// Check that convertCmd is not nil (should be initialized)
	if convertCmd == nil {
		t.Error("convertCmd should be initialized")
	}
	
	// Check that the command has the config flag with persistent flag
	configFlag := convertCmd.PersistentFlags().Lookup("config")
	if configFlag == nil {
		t.Error("Expected config flag to be a persistent flag")
	}
}