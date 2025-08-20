package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestRootCmd(t *testing.T) {
	if rootCmd == nil {
		t.Error("rootCmd should not be nil")
	}
	if rootCmd.Use != "geoip" {
		t.Errorf("rootCmd.Use should be 'geoip', got: %s", rootCmd.Use)
	}
	if rootCmd.Short == "" {
		t.Error("rootCmd.Short should not be empty")
	}
}

func TestRootCmdHelp(t *testing.T) {
	// Test that the root command can display help without errors
	output := captureOutput(func() {
		args := []string{"--help"}
		rootCmd.SetArgs(args)
		rootCmd.Execute()
	})

	if !strings.Contains(output, "geoip") {
		t.Error("Help output should contain 'geoip'")
	}
	if !strings.Contains(output, "Available Commands") {
		t.Error("Help output should contain 'Available Commands'")
	}
}

func TestListCmd(t *testing.T) {
	if listCmd == nil {
		t.Error("listCmd should not be nil")
	}
	if listCmd.Use != "list" {
		t.Errorf("listCmd.Use should be 'list', got: %s", listCmd.Use)
	}
	
	// Check aliases
	expectedAliases := []string{"l", "ls"}
	if len(listCmd.Aliases) != len(expectedAliases) {
		t.Errorf("listCmd should have %d aliases, got %d", len(expectedAliases), len(listCmd.Aliases))
	}
	for i, alias := range expectedAliases {
		if i < len(listCmd.Aliases) && listCmd.Aliases[i] != alias {
			t.Errorf("listCmd.Aliases[%d] should be '%s', got '%s'", i, alias, listCmd.Aliases[i])
		}
	}
}

func TestListCmdExecution(t *testing.T) {
	// Test that the list command executes without errors
	output := captureOutput(func() {
		args := []string{"list"}
		rootCmd.SetArgs(args)
		rootCmd.Execute()
	})

	if !strings.Contains(output, "All available input formats") {
		t.Error("List output should contain 'All available input formats'")
	}
	if !strings.Contains(output, "All available output formats") {
		t.Error("List output should contain 'All available output formats'")
	}
}

func TestListCmdWithAlias(t *testing.T) {
	// Test that the list command works with alias
	output := captureOutput(func() {
		args := []string{"l"}
		rootCmd.SetArgs(args)
		rootCmd.Execute()
	})

	if !strings.Contains(output, "All available input formats") {
		t.Error("List output with alias should contain 'All available input formats'")
	}
}

func TestConvertCmd(t *testing.T) {
	if convertCmd == nil {
		t.Error("convertCmd should not be nil")
	}
	if convertCmd.Use != "convert" {
		t.Errorf("convertCmd.Use should be 'convert', got: %s", convertCmd.Use)
	}
	
	// Check aliases
	expectedAliases := []string{"conv"}
	if len(convertCmd.Aliases) != len(expectedAliases) {
		t.Errorf("convertCmd should have %d aliases, got %d", len(expectedAliases), len(convertCmd.Aliases))
	}
	for i, alias := range expectedAliases {
		if i < len(convertCmd.Aliases) && convertCmd.Aliases[i] != alias {
			t.Errorf("convertCmd.Aliases[%d] should be '%s', got '%s'", i, alias, convertCmd.Aliases[i])
		}
	}
}

func TestConvertCmdHelp(t *testing.T) {
	// Test that the convert command can display help
	output := captureOutput(func() {
		args := []string{"convert", "--help"}
		rootCmd.SetArgs(args)
		rootCmd.Execute()
	})

	if !strings.Contains(output, "convert") {
		t.Error("Convert help output should contain 'convert'")
	}
	if !strings.Contains(output, "config") {
		t.Error("Convert help output should contain 'config' flag")
	}
}

func TestConvertCmdWithInvalidConfig(t *testing.T) {
	// Test convert command with non-existent config file
	// This should fail gracefully
	
	// Capture stderr for this test since log.Fatal writes to stderr
	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	// This will likely call os.Exit, so we can't easily test it
	// Just verify the command structure is correct
	
	w.Close()
	os.Stderr = oldStderr
	
	// Just test that the command has the right structure
	flag := convertCmd.PersistentFlags().Lookup("config")
	if flag == nil {
		t.Error("convertCmd should have 'config' flag")
	}
	if flag.DefValue != "config.json" {
		t.Errorf("config flag default should be 'config.json', got: %s", flag.DefValue)
	}
}

func TestConvertCmdWithValidConfig(t *testing.T) {
	// Create a minimal valid config file for testing
	tempDir := os.TempDir()
	configFile := filepath.Join(tempDir, "test_config.json")
	
	configContent := `{
		"input": [],
		"output": []
	}`
	
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	defer os.Remove(configFile)

	// This test is tricky because the convert command calls log.Fatal on errors
	// and there's no easy way to capture that without changing the command structure
	// So we'll just test that the flag parsing works correctly
	
	args := []string{"convert", "--config", configFile}
	convertCmd.SetArgs(args[1:]) // Remove "convert" since we're calling the command directly
	
	// Parse flags to ensure they work
	err = convertCmd.ParseFlags(args[1:])
	if err != nil {
		t.Errorf("Flag parsing should not fail: %v", err)
	}
	
	configFlag, err := convertCmd.Flags().GetString("config")
	if err != nil {
		t.Errorf("Getting config flag should not fail: %v", err)
	}
	if configFlag != configFile {
		t.Errorf("Config flag should be %s, got %s", configFile, configFlag)
	}
}

func TestCommandsAreAddedToRoot(t *testing.T) {
	// Test that all expected commands are added to the root command
	commands := rootCmd.Commands()
	
	expectedCommands := []string{"list", "convert"}
	foundCommands := make(map[string]bool)
	
	for _, cmd := range commands {
		foundCommands[cmd.Use] = true
	}
	
	for _, expected := range expectedCommands {
		if !foundCommands[expected] {
			t.Errorf("Root command should have '%s' subcommand", expected)
		}
	}
}

func TestCommandShortDescriptions(t *testing.T) {
	tests := []struct {
		cmd         *cobra.Command
		name        string
		shouldContain string
	}{
		{
			cmd:           listCmd,
			name:          "list",
			shouldContain: "List all available",
		},
		{
			cmd:           convertCmd,
			name:          "convert", 
			shouldContain: "Convert geoip data",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cmd.Short == "" {
				t.Errorf("%s command should have a short description", tt.name)
			}
			if !strings.Contains(tt.cmd.Short, tt.shouldContain) {
				t.Errorf("%s command short description should contain '%s', got: %s", 
					tt.name, tt.shouldContain, tt.cmd.Short)
			}
		})
	}
}

func TestRootCmdCompletionOptions(t *testing.T) {
	if !rootCmd.CompletionOptions.HiddenDefaultCmd {
		t.Error("Root command should have HiddenDefaultCmd set to true")
	}
}

func TestMainFunction(t *testing.T) {
	// We can't easily test the main function since it calls rootCmd.Execute()
	// and may call os.Exit, but we can verify the structure exists
	
	// This is more of a compilation test - if this test runs, 
	// it means the main function compiles correctly
	if rootCmd == nil {
		t.Error("main() function depends on rootCmd being initialized")
	}
}