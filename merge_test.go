package main

import (
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/Loyalsoldier/geoip/plugin/special"
)

func TestGetInputForMerge(t *testing.T) {
	result := getInputForMerge()
	
	// Verify it returns a Stdin converter
	stdin, ok := result.(*special.Stdin)
	if !ok {
		t.Errorf("Expected *special.Stdin, got %T", result)
		return
	}

	// Verify the type
	if stdin.GetType() != special.TypeStdin {
		t.Errorf("Expected type %s, got %s", special.TypeStdin, stdin.GetType())
	}

	// Verify the action
	if stdin.GetAction() != lib.ActionAdd {
		t.Errorf("Expected action %s, got %s", lib.ActionAdd, stdin.GetAction())
	}

	// Verify the description
	if stdin.GetDescription() != special.DescStdin {
		t.Errorf("Expected description %s, got %s", special.DescStdin, stdin.GetDescription())
	}
}

func TestGetOutputForMerge(t *testing.T) {
	tests := []struct {
		name           string
		otype          string
		expectedIPType string
	}{
		{"IPv4 only", "ipv4", "ipv4"},
		{"IPv6 only", "ipv6", "ipv6"},
		{"Both IP types", "", ""}, // Default case
		{"Empty string", "", ""},
		{"Invalid type", "invalid", ""}, // Should default
		{"Mixed case IPv4", "IPv4", ""}, // Should default since case doesn't match
		{"Mixed case IPv6", "IPv6", ""}, // Should default since case doesn't match
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getOutputForMerge(tt.otype)
			
			// Verify it returns a Stdout converter
			stdout, ok := result.(*special.Stdout)
			if !ok {
				t.Errorf("Expected *special.Stdout, got %T", result)
				return
			}

			// Verify the type
			if stdout.GetType() != special.TypeStdout {
				t.Errorf("Expected type %s, got %s", special.TypeStdout, stdout.GetType())
			}

			// Verify the action
			if stdout.GetAction() != lib.ActionOutput {
				t.Errorf("Expected action %s, got %s", lib.ActionOutput, stdout.GetAction())
			}

			// Verify the description
			if stdout.GetDescription() != special.DescStdout {
				t.Errorf("Expected description %s, got %s", special.DescStdout, stdout.GetDescription())
			}

			// For specific IP type cases, we can't easily test the OnlyIPType field
			// without more detailed knowledge of the Stdout struct internals
			// The important thing is that the function returns the right type and doesn't panic
		})
	}
}

func TestMergeCmd(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "command configuration",
			test: func(t *testing.T) {
				if mergeCmd.Use != "merge" {
					t.Errorf("Expected Use to be 'merge', got '%s'", mergeCmd.Use)
				}
				
				expectedAliases := []string{"m"}
				if len(mergeCmd.Aliases) != len(expectedAliases) {
					t.Errorf("Expected %d aliases, got %d", len(expectedAliases), len(mergeCmd.Aliases))
				}
				
				for i, alias := range expectedAliases {
					if i >= len(mergeCmd.Aliases) || mergeCmd.Aliases[i] != alias {
						t.Errorf("Expected alias '%s' at index %d", alias, i)
					}
				}
				
				expectedShort := "Merge plaintext IP & CIDR from standard input, then print to standard output"
				if mergeCmd.Short != expectedShort {
					t.Errorf("Expected Short to be '%s', got '%s'", expectedShort, mergeCmd.Short)
				}
			},
		},
		{
			name: "command flags",
			test: func(t *testing.T) {
				// Check onlyiptype flag exists as persistent flag
				onlyiptypeFlag := mergeCmd.PersistentFlags().Lookup("onlyiptype")
				if onlyiptypeFlag == nil {
					t.Error("Expected onlyiptype flag to exist")
				}
				
				// Check that the flag has correct shorthand
				if onlyiptypeFlag.Shorthand != "t" {
					t.Errorf("Expected shorthand 't', got '%s'", onlyiptypeFlag.Shorthand)
				}
				
				// Check default value
				if onlyiptypeFlag.DefValue != "" {
					t.Errorf("Expected empty default value, got '%s'", onlyiptypeFlag.DefValue)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestMergeCmdExecution(t *testing.T) {
	// Test that the merge command can be executed with valid flags
	// We'll test this by checking that the Run function is set and doesn't panic immediately
	
	if mergeCmd.Run == nil {
		t.Error("Expected merge command to have a Run function")
		return
	}

	// We can't easily test the full execution without mocking stdin/stdout and the lib package
	// But we can test that the command structure is correct and functions are callable
	
	// Test that helper functions can be called without panicking
	t.Run("getInputForMerge doesn't panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("getInputForMerge panicked: %v", r)
			}
		}()
		
		result := getInputForMerge()
		if result == nil {
			t.Error("getInputForMerge returned nil")
		}
	})

	t.Run("getOutputForMerge doesn't panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("getOutputForMerge panicked: %v", r)
			}
		}()
		
		testCases := []string{"", "ipv4", "ipv6", "invalid"}
		for _, otype := range testCases {
			result := getOutputForMerge(otype)
			if result == nil {
				t.Errorf("getOutputForMerge(%s) returned nil", otype)
			}
		}
	})
}