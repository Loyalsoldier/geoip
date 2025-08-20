package plaintext

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestTextOutConstants(t *testing.T) {
	if TypeTextOut != "text" {
		t.Errorf("TypeTextOut should be 'text', got: %s", TypeTextOut)
	}
	if DescTextOut != "Convert data to plaintext CIDR format" {
		t.Errorf("DescTextOut should be correct description, got: %s", DescTextOut)
	}
}

func TestNewTextOut(t *testing.T) {
	tests := []struct {
		name         string
		iType        string
		iDesc        string
		action       lib.Action
		data         string
		expectError  bool
		expectDir    string
		expectExt    string
		expectIPType lib.IPType
	}{
		{
			name:         "Valid empty config",
			iType:        TypeTextOut,
			iDesc:        DescTextOut,
			action:       lib.ActionOutput,
			data:         `{}`,
			expectError:  false,
			expectDir:    defaultOutputDirForTextOut,
			expectExt:    ".txt",
			expectIPType: "",
		},
		{
			name:         "Custom output directory",
			iType:        TypeTextOut,
			iDesc:        DescTextOut,
			action:       lib.ActionOutput,
			data:         `{"outputDir": "/custom/dir"}`,
			expectError:  false,
			expectDir:    "/custom/dir",
			expectExt:    ".txt",
			expectIPType: "",
		},
		{
			name:         "Custom output extension",
			iType:        TypeTextOut,
			iDesc:        DescTextOut,
			action:       lib.ActionOutput,
			data:         `{"outputExtension": ".dat"}`,
			expectError:  false,
			expectDir:    defaultOutputDirForTextOut,
			expectExt:    ".dat",
			expectIPType: "",
		},
		{
			name:         "IPv4 only",
			iType:        TypeTextOut,
			iDesc:        DescTextOut,
			action:       lib.ActionOutput,
			data:         `{"onlyIPType": "ipv4"}`,
			expectError:  false,
			expectDir:    defaultOutputDirForTextOut,
			expectExt:    ".txt",
			expectIPType: lib.IPv4,
		},
		{
			name:         "IPv6 only",
			iType:        TypeTextOut,
			iDesc:        DescTextOut,
			action:       lib.ActionOutput,
			data:         `{"onlyIPType": "ipv6"}`,
			expectError:  false,
			expectDir:    defaultOutputDirForTextOut,
			expectExt:    ".txt",
			expectIPType: lib.IPv6,
		},
		{
			name:         "With prefixes and suffixes",
			iType:        TypeTextOut,
			iDesc:        DescTextOut,
			action:       lib.ActionOutput,
			data:         `{"addPrefixInLine": "PREFIX:", "addSuffixInLine": ":SUFFIX"}`,
			expectError:  false,
			expectDir:    defaultOutputDirForTextOut,
			expectExt:    ".txt",
			expectIPType: "",
		},
		{
			name:        "Invalid JSON",
			iType:       TypeTextOut,
			iDesc:       DescTextOut,
			action:      lib.ActionOutput,
			data:        `{invalid json}`,
			expectError: true,
		},
		{
			name:         "Empty data",
			iType:        TypeTextOut,
			iDesc:        DescTextOut,
			action:       lib.ActionOutput,
			data:         ``,
			expectError:  false,
			expectDir:    defaultOutputDirForTextOut,
			expectExt:    ".txt",
			expectIPType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter, err := newTextOut(tt.iType, tt.iDesc, tt.action, json.RawMessage(tt.data))

			if tt.expectError && err == nil {
				t.Errorf("newTextOut() should return error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("newTextOut() should not return error but got: %v", err)
			}

			if !tt.expectError {
				if converter == nil {
					t.Error("newTextOut() should return non-nil converter")
				} else {
					textOut := converter.(*TextOut)
					if textOut.GetType() != tt.iType {
						t.Errorf("GetType() = %s; want %s", textOut.GetType(), tt.iType)
					}
					if textOut.GetAction() != tt.action {
						t.Errorf("GetAction() = %s; want %s", textOut.GetAction(), tt.action)
					}
					if textOut.GetDescription() != tt.iDesc {
						t.Errorf("GetDescription() = %s; want %s", textOut.GetDescription(), tt.iDesc)
					}
					if textOut.OutputDir != tt.expectDir {
						t.Errorf("OutputDir = %s; want %s", textOut.OutputDir, tt.expectDir)
					}
					if textOut.OutputExt != tt.expectExt {
						t.Errorf("OutputExt = %s; want %s", textOut.OutputExt, tt.expectExt)
					}
					if textOut.OnlyIPType != tt.expectIPType {
						t.Errorf("OnlyIPType = %s; want %s", textOut.OnlyIPType, tt.expectIPType)
					}
				}
			}
		})
	}
}

func TestTextOutStruct(t *testing.T) {
	textOut := &TextOut{
		Type:            "custom-text",
		Action:          lib.ActionOutput,
		Description:     "custom description",
		OutputDir:       "/custom/dir",
		OutputExt:       ".custom",
		Want:            []string{"want1", "want2"},
		Exclude:         []string{"exclude1"},
		OnlyIPType:      lib.IPv4,
		AddPrefixInLine: "PREFIX:",
		AddSuffixInLine: ":SUFFIX",
	}

	if textOut.GetType() != "custom-text" {
		t.Errorf("GetType() = %s; want custom-text", textOut.GetType())
	}
	if textOut.GetAction() != lib.ActionOutput {
		t.Errorf("GetAction() = %s; want %s", textOut.GetAction(), lib.ActionOutput)
	}
	if textOut.GetDescription() != "custom description" {
		t.Errorf("GetDescription() = %s; want custom description", textOut.GetDescription())
	}
}

func TestTextOutMarshalBytes(t *testing.T) {
	tests := []struct {
		name       string
		onlyIPType lib.IPType
		prefixes   []string
		expectError bool
		checkContent string
	}{
		{
			name:         "All IP types",
			onlyIPType:   "",
			prefixes:     []string{"192.168.1.0/24", "2001:db8::/32"},
			expectError:  false,
			checkContent: "192.168.1.0/24",
		},
		{
			name:         "IPv4 only",
			onlyIPType:   lib.IPv4,
			prefixes:     []string{"192.168.1.0/24", "2001:db8::/32"},
			expectError:  false,
			checkContent: "192.168.1.0/24",
		},
		{
			name:         "IPv6 only",
			onlyIPType:   lib.IPv6,
			prefixes:     []string{"192.168.1.0/24", "2001:db8::/32"},
			expectError:  false,
			checkContent: "2001:db8::/32",
		},
		{
			name:        "Empty entry",
			onlyIPType:  "",
			prefixes:    []string{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := lib.NewEntry("test")
			
			// Add prefixes to entry
			for _, prefix := range tt.prefixes {
				err := entry.AddPrefix(prefix)
				if err != nil {
					t.Fatalf("AddPrefix failed: %v", err)
				}
			}

			textOut := &TextOut{
				Type:            TypeTextOut,
				OnlyIPType:      tt.onlyIPType,
				AddPrefixInLine: "",
				AddSuffixInLine: "",
			}

			result, err := textOut.marshalBytes(entry)

			if tt.expectError && err == nil {
				t.Errorf("marshalBytes() should return error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("marshalBytes() should not return error but got: %v", err)
			}

			if !tt.expectError {
				if len(result) == 0 {
					t.Error("marshalBytes() should return non-empty result")
				}

				if tt.checkContent != "" && !strings.Contains(string(result), tt.checkContent) {
					t.Errorf("marshalBytes() should contain %s", tt.checkContent)
				}
			}
		})
	}
}

func TestTextOutMarshalBytesWithPrefixSuffix(t *testing.T) {
	entry := lib.NewEntry("test")
	err := entry.AddPrefix("192.168.1.0/24")
	if err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}

	textOut := &TextOut{
		Type:            TypeTextOut,
		AddPrefixInLine: "PREFIX:",
		AddSuffixInLine: ":SUFFIX",
	}

	result, err := textOut.marshalBytes(entry)
	if err != nil {
		t.Errorf("marshalBytes() should not return error: %v", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "PREFIX:192.168.1.0/24:SUFFIX") {
		t.Errorf("marshalBytes() should contain prefixed and suffixed CIDR, got: %s", resultStr)
	}
}

func TestTextOutOutput(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := os.TempDir()
	testOutputDir := filepath.Join(tempDir, "test-output")
	
	// Clean up after test
	defer os.RemoveAll(testOutputDir)

	// Create a container with test data
	container := lib.NewContainer()
	entry := lib.NewEntry("test-entry")
	err := entry.AddPrefix("192.168.1.0/24")
	if err != nil {
		t.Fatalf("AddPrefix failed: %v", err)
	}
	err = container.Add(entry)
	if err != nil {
		t.Fatalf("Add entry failed: %v", err)
	}

	textOut := &TextOut{
		Type:        TypeTextOut,
		Action:      lib.ActionOutput,
		Description: DescTextOut,
		OutputDir:   testOutputDir,
		OutputExt:   ".txt",
		Want:        []string{"test-entry"},
	}

	err = textOut.Output(container)
	if err != nil {
		t.Errorf("Output() should not return error: %v", err)
	}

	// Check that output file was created
	expectedFile := filepath.Join(testOutputDir, "test-entry.txt")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Output file should be created at %s", expectedFile)
	} else if err != nil {
		t.Errorf("Error checking output file: %v", err)
	} else {
		// Check file content
		content, err := os.ReadFile(expectedFile)
		if err != nil {
			t.Errorf("Error reading output file: %v", err)
		} else {
			contentStr := string(content)
			if !strings.Contains(contentStr, "192.168.1.0/24") {
				t.Errorf("Output file should contain CIDR, got: %s", contentStr)
			}
		}
	}
}

func TestTextOutOutput_EmptyContainer(t *testing.T) {
	container := lib.NewContainer()

	textOut := &TextOut{
		Type:        TypeTextOut,
		Action:      lib.ActionOutput,
		Description: DescTextOut,
		OutputDir:   os.TempDir(),
		OutputExt:   ".txt",
	}

	err := textOut.Output(container)
	if err != nil {
		t.Errorf("Output() with empty container should not return error: %v", err)
	}
}

func TestDefaultOutputDirectories(t *testing.T) {
	tests := []struct {
		name        string
		defaultDir  string
		expectPath  string
	}{
		{
			name:        "TextOut default",
			defaultDir:  defaultOutputDirForTextOut,
			expectPath:  "output/text",
		},
		{
			name:        "ClashRuleSetClassical default",
			defaultDir:  defaultOutputDirForClashRuleSetClassicalOut,
			expectPath:  "output/clash/classical",
		},
		{
			name:        "ClashRuleSetIPCIDR default",
			defaultDir:  defaultOutputDirForClashRuleSetIPCIDROut,
			expectPath:  "output/clash/ipcidr",
		},
		{
			name:        "SurgeRuleSet default",
			defaultDir:  defaultOutputDirForSurgeRuleSetOut,
			expectPath:  "output/surge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.defaultDir, tt.expectPath) {
				t.Errorf("Default directory %s should contain %s", tt.defaultDir, tt.expectPath)
			}
		})
	}
}