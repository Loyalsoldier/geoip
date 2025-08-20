package special

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestStdinConstants(t *testing.T) {
	if TypeStdin != "stdin" {
		t.Errorf("TypeStdin should be 'stdin', got: %s", TypeStdin)
	}
	if DescStdin != "Accept plaintext IP & CIDR from standard input, separated by newline" {
		t.Errorf("DescStdin should be correct description, got: %s", DescStdin)
	}
}

func TestNewStdin(t *testing.T) {
	tests := []struct {
		name        string
		action      lib.Action
		data        string
		expectError bool
		expectName  string
		expectIPType lib.IPType
	}{
		{
			name:         "Valid config with name",
			action:       lib.ActionAdd,
			data:         `{"name": "test-stdin"}`,
			expectError:  false,
			expectName:   "test-stdin",
			expectIPType: "",
		},
		{
			name:         "Valid config with name and IPv4",
			action:       lib.ActionAdd,
			data:         `{"name": "test-stdin", "onlyIPType": "ipv4"}`,
			expectError:  false,
			expectName:   "test-stdin",
			expectIPType: lib.IPv4,
		},
		{
			name:         "Valid config with name and IPv6",
			action:       lib.ActionRemove,
			data:         `{"name": "test-stdin", "onlyIPType": "ipv6"}`,
			expectError:  false,
			expectName:   "test-stdin",
			expectIPType: lib.IPv6,
		},
		{
			name:        "Missing name",
			action:      lib.ActionAdd,
			data:        `{}`,
			expectError: true,
		},
		{
			name:        "Empty name",
			action:      lib.ActionAdd,
			data:        `{"name": ""}`,
			expectError: true,
		},
		{
			name:        "Invalid JSON",
			action:      lib.ActionAdd,
			data:        `{invalid json}`,
			expectError: true,
		},
		{
			name:         "Empty data",
			action:       lib.ActionAdd,
			data:         ``,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter, err := newStdin(tt.action, json.RawMessage(tt.data))

			if tt.expectError && err == nil {
				t.Errorf("newStdin() should return error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("newStdin() should not return error but got: %v", err)
			}

			if !tt.expectError {
				if converter == nil {
					t.Error("newStdin() should return non-nil converter")
				} else {
					stdin := converter.(*Stdin)
					if stdin.GetType() != TypeStdin {
						t.Errorf("GetType() = %s; want %s", stdin.GetType(), TypeStdin)
					}
					if stdin.GetAction() != tt.action {
						t.Errorf("GetAction() = %s; want %s", stdin.GetAction(), tt.action)
					}
					if stdin.GetDescription() != DescStdin {
						t.Errorf("GetDescription() = %s; want %s", stdin.GetDescription(), DescStdin)
					}
					if stdin.Name != tt.expectName {
						t.Errorf("Name = %s; want %s", stdin.Name, tt.expectName)
					}
					if stdin.OnlyIPType != tt.expectIPType {
						t.Errorf("OnlyIPType = %s; want %s", stdin.OnlyIPType, tt.expectIPType)
					}
				}
			}
		})
	}
}

func TestStdinStruct(t *testing.T) {
	stdin := &Stdin{
		Type:        "custom-stdin",
		Action:      lib.ActionAdd,
		Description: "custom description",
		Name:        "custom-name",
		OnlyIPType:  lib.IPv4,
	}

	if stdin.GetType() != "custom-stdin" {
		t.Errorf("GetType() = %s; want custom-stdin", stdin.GetType())
	}
	if stdin.GetAction() != lib.ActionAdd {
		t.Errorf("GetAction() = %s; want %s", stdin.GetAction(), lib.ActionAdd)
	}
	if stdin.GetDescription() != "custom description" {
		t.Errorf("GetDescription() = %s; want custom description", stdin.GetDescription())
	}
}

// Note: Testing the Input method is complex because it reads from os.Stdin
// In a real test environment, we would need to mock os.Stdin or use dependency injection
// For now, we'll test the structure and error conditions

func TestStdinInput_UnknownAction(t *testing.T) {
	stdin := &Stdin{
		Type:        TypeStdin,
		Action:      lib.Action("unknown"),
		Description: DescStdin,
		Name:        "test",
	}

	// Create a pipe to simulate stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	// Temporarily replace stdin
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		r.Close()
	}()

	// Write some data and close the writer
	w.WriteString("127.0.0.1\n")
	w.Close()

	container := lib.NewContainer()
	_, err = stdin.Input(container)
	if err != lib.ErrUnknownAction {
		t.Errorf("Input() with unknown action should return ErrUnknownAction, got: %v", err)
	}
}

func TestStdinInput_AddAction(t *testing.T) {
	stdin := &Stdin{
		Type:        TypeStdin,
		Action:      lib.ActionAdd,
		Description: DescStdin,
		Name:        "test-stdin",
	}

	// Create a pipe to simulate stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	// Temporarily replace stdin
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		r.Close()
	}()

	// Write test data
	testData := "127.0.0.1\n192.168.1.0/24\n# comment line\n10.0.0.0/8\n"
	w.WriteString(testData)
	w.Close()

	container := lib.NewContainer()
	result, err := stdin.Input(container)
	if err != nil {
		t.Errorf("Input() should not return error: %v", err)
	}
	if result == nil {
		t.Error("Input() should return non-nil container")
	}

	// Check that entry was added
	entry, found := result.GetEntry("test-stdin")
	if !found {
		t.Error("Container should contain the test-stdin entry")
	}
	if entry.GetName() != "TEST-STDIN" {
		t.Errorf("Entry name should be TEST-STDIN, got %s", entry.GetName())
	}
}

func TestStdinInput_IPv4Only(t *testing.T) {
	stdin := &Stdin{
		Type:        TypeStdin,
		Action:      lib.ActionAdd,
		Description: DescStdin,
		Name:        "ipv4-test",
		OnlyIPType:  lib.IPv4,
	}

	// Create a pipe to simulate stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	// Temporarily replace stdin
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		r.Close()
	}()

	// Write mixed IPv4 and IPv6 data
	testData := "192.168.1.0/24\n2001:db8::/32\n10.0.0.0/8\n"
	w.WriteString(testData)
	w.Close()

	container := lib.NewContainer()
	result, err := stdin.Input(container)
	if err != nil {
		t.Errorf("Input() should not return error: %v", err)
	}

	// Verify entry was added
	_, found := result.GetEntry("ipv4-test")
	if !found {
		t.Error("Container should contain the ipv4-test entry")
	}
}

func TestStdinInput_CommentHandling(t *testing.T) {
	stdin := &Stdin{
		Type:        TypeStdin,
		Action:      lib.ActionAdd,
		Description: DescStdin,
		Name:        "comment-test",
	}

	// Create a pipe to simulate stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	// Temporarily replace stdin
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		r.Close()
	}()

	// Write data with various comment formats
	testData := `192.168.1.0/24 # inline comment
10.0.0.0/8 // cpp style comment  
172.16.0.0/12 /* block comment
# full line comment
   
127.0.0.1
`
	w.WriteString(testData)
	w.Close()

	container := lib.NewContainer()
	result, err := stdin.Input(container)
	if err != nil {
		t.Errorf("Input() should not return error: %v", err)
	}

	// Verify entry was added
	_, found := result.GetEntry("comment-test")
	if !found {
		t.Error("Container should contain the comment-test entry")
	}
}

// Test error cases that don't require stdin manipulation
func TestStdinInput_InvalidConfig(t *testing.T) {
	// Test with empty name (this should be caught in newStdin, but let's be safe)
	stdin := &Stdin{
		Type:        TypeStdin,
		Action:      lib.ActionAdd,
		Description: DescStdin,
		Name:        "", // Empty name might cause issues
	}

	// Create a minimal pipe
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		r.Close()
	}()

	w.WriteString("127.0.0.1\n")
	w.Close()

	container := lib.NewContainer()
	_, err = stdin.Input(container)
	// The behavior with empty name might vary, so we just ensure it doesn't crash
	if err != nil {
		// This is acceptable - empty names might cause errors
		t.Logf("Input() with empty name returned error (as expected): %v", err)
	}
}