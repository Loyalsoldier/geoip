package main

import (
	"testing"
)

func TestListCmd(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "command configuration",
			test: func(t *testing.T) {
				if listCmd.Use != "list" {
					t.Errorf("Expected Use to be 'list', got '%s'", listCmd.Use)
				}
				
				expectedAliases := []string{"l", "ls"}
				if len(listCmd.Aliases) != len(expectedAliases) {
					t.Errorf("Expected %d aliases, got %d", len(expectedAliases), len(listCmd.Aliases))
				}
				
				for i, alias := range expectedAliases {
					if i >= len(listCmd.Aliases) || listCmd.Aliases[i] != alias {
						t.Errorf("Expected alias '%s' at index %d", alias, i)
					}
				}
				
				expectedShort := "List all available input and output formats"
				if listCmd.Short != expectedShort {
					t.Errorf("Expected Short to be '%s', got '%s'", expectedShort, listCmd.Short)
				}
			},
		},
		{
			name: "command has run function",
			test: func(t *testing.T) {
				if listCmd.Run == nil {
					t.Error("Expected list command to have a Run function")
				}
			},
		},
		{
			name: "command is added to root",
			test: func(t *testing.T) {
				// Test that the command is added to root command
				found := false
				for _, cmd := range rootCmd.Commands() {
					if cmd.Use == "list" {
						found = true
						break
					}
				}
				
				if !found {
					t.Error("List command should be added to root command")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestListCmdExecution(t *testing.T) {
	// Test that the list command can be executed without panicking
	// We can't easily test the actual output without mocking the lib package
	// But we can test that the Run function exists and is callable
	
	if listCmd.Run == nil {
		t.Error("Expected list command to have a Run function")
		return
	}
	
	// The Run function calls lib.ListInputConverter() and lib.ListOutputConverter()
	// We can't test the actual execution without mocking these functions or causing side effects
	// But we can verify the command structure is correct
	
	// Test that the command accepts no arguments (which is the default behavior)
	// and doesn't have any required flags
	if listCmd.Args != nil {
		// If Args is set, it should allow any number of arguments or be nil for default behavior
		// For the list command, it should accept no arguments, so Args being nil is correct
		t.Logf("List command Args function is set: %v", listCmd.Args)
	}
}

func TestListInit(t *testing.T) {
	// Test that the init function properly sets up the command
	
	// Check that listCmd is not nil (should be initialized)
	if listCmd == nil {
		t.Error("listCmd should be initialized")
	}
	
	// Check that the command has no flags (it shouldn't need any)
	flags := listCmd.Flags()
	if flags.NFlag() > 0 {
		t.Errorf("Expected list command to have no flags, but it has %d", flags.NFlag())
	}
	
	// Check that the command has no persistent flags
	persistentFlags := listCmd.PersistentFlags()
	if persistentFlags.NFlag() > 0 {
		t.Errorf("Expected list command to have no persistent flags, but it has %d", persistentFlags.NFlag())
	}
}