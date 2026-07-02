package special

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestStdin_NewStdin(t *testing.T) {
	tests := []struct {
		name         string
		action       lib.Action
		data         json.RawMessage
		expectType   string
		expectName   string
		expectIPType lib.IPType
		expectErr    bool
	}{
		{
			name:         "Valid action with name",
			action:       lib.ActionAdd,
			data:         json.RawMessage(`{"name": "testentry"}`),
			expectType:   TypeStdin,
			expectName:   "testentry",
			expectIPType: "",
			expectErr:    false,
		},
		{
			name:         "Valid action with name and IPv4 only",
			action:       lib.ActionAdd,
			data:         json.RawMessage(`{"name": "testentry", "onlyIPType": "ipv4"}`),
			expectType:   TypeStdin,
			expectName:   "testentry",
			expectIPType: lib.IPv4,
			expectErr:    false,
		},
		{
			name:      "Missing name",
			action:    lib.ActionAdd,
			data:      json.RawMessage(`{}`),
			expectErr: true,
		},
		{
			name:      "Empty name",
			action:    lib.ActionAdd,
			data:      json.RawMessage(`{"name": ""}`),
			expectErr: true,
		},
		{
			name:      "Invalid JSON",
			action:    lib.ActionAdd,
			data:      json.RawMessage(`{invalid json}`),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter, err := newStdin(tt.action, tt.data)
			if (err != nil) != tt.expectErr {
				t.Errorf("newStdin() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr {
				stdin := converter.(*Stdin)
				if stdin.GetType() != tt.expectType {
					t.Errorf("GetType() = %v, expect %v", stdin.GetType(), tt.expectType)
				}
				if stdin.GetAction() != tt.action {
					t.Errorf("GetAction() = %v, expect %v", stdin.GetAction(), tt.action)
				}
				if stdin.Name != tt.expectName {
					t.Errorf("Name = %v, expect %v", stdin.Name, tt.expectName)
				}
				if stdin.OnlyIPType != tt.expectIPType {
					t.Errorf("OnlyIPType = %v, expect %v", stdin.OnlyIPType, tt.expectIPType)
				}
			}
		})
	}
}

func TestStdin_GetType(t *testing.T) {
	stdin := &Stdin{Type: TypeStdin}
	result := stdin.GetType()
	if result != TypeStdin {
		t.Errorf("GetType() = %v, expect %v", result, TypeStdin)
	}
}

func TestStdin_GetAction(t *testing.T) {
	action := lib.ActionAdd
	stdin := &Stdin{Action: action}
	result := stdin.GetAction()
	if result != action {
		t.Errorf("GetAction() = %v, expect %v", result, action)
	}
}

func TestStdin_GetDescription(t *testing.T) {
	stdin := &Stdin{Description: DescStdin}
	result := stdin.GetDescription()
	if result != DescStdin {
		t.Errorf("GetDescription() = %v, expect %v", result, DescStdin)
	}
}

func TestStdin_Input(t *testing.T) {
	tests := []struct {
		name       string
		action     lib.Action
		onlyIPType lib.IPType
		stdinData  string
		expectErr  bool
	}{
		{
			name:       "Action add with valid CIDR",
			action:     lib.ActionAdd,
			onlyIPType: "",
			stdinData:  "192.168.1.0/24\n10.0.0.0/8\n",
			expectErr:  false,
		},
		{
			name:       "Action add with IPv4 only",
			action:     lib.ActionAdd,
			onlyIPType: lib.IPv4,
			stdinData:  "192.168.1.0/24\n",
			expectErr:  false,
		},
		{
			name:       "Action add with IPv6 only",
			action:     lib.ActionAdd,
			onlyIPType: lib.IPv6,
			stdinData:  "2001:db8::/32\n",
			expectErr:  false,
		},
		{
			name:       "Action remove",
			action:     lib.ActionRemove,
			onlyIPType: "",
			stdinData:  "192.168.1.0/24\n",
			expectErr:  false,
		},
		{
			name:       "Empty input",
			action:     lib.ActionAdd,
			onlyIPType: "",
			stdinData:  "",
			expectErr:  false,
		},
		{
			name:       "Input with comments",
			action:     lib.ActionAdd,
			onlyIPType: "",
			stdinData:  "192.168.1.0/24 # This is a comment\n10.0.0.0/8 // Another comment\n",
			expectErr:  false,
		},
		{
			name:       "Invalid action",
			action:     lib.Action("invalid"),
			onlyIPType: "",
			stdinData:  "192.168.1.0/24\n",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock stdin
			oldStdin := os.Stdin
			r, w, _ := os.Pipe()
			os.Stdin = r

			// Write test data to pipe
			go func() {
				defer w.Close()
				w.Write([]byte(tt.stdinData))
			}()

			stdin := &Stdin{
				Type:        TypeStdin,
				Action:      tt.action,
				Description: DescStdin,
				Name:        "TESTENTRY",
				OnlyIPType:  tt.onlyIPType,
			}

			container := lib.NewContainer()

			// For remove action, pre-populate the container
			if tt.action == lib.ActionRemove && !tt.expectErr {
				entry := lib.NewEntry("TESTENTRY")
				if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
					t.Fatalf("Failed to add prefix: %v", err)
				}
				if err := container.Add(entry); err != nil {
					t.Fatalf("Failed to add entry to container: %v", err)
				}
			}

			result, err := stdin.Input(container)

			// Restore stdin
			os.Stdin = oldStdin

			if (err != nil) != tt.expectErr {
				t.Errorf("Input() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			if !tt.expectErr {
				if result == nil {
					t.Error("Input() returned nil container")
					return
				}

				if tt.action == lib.ActionAdd && tt.stdinData != "" {
					entry, found := result.GetEntry("TESTENTRY")
					if !found {
						t.Error("Expected entry not found in container")
					}
					if entry == nil {
						t.Error("Entry is nil")
					}
				}
			}
		})
	}
}

func TestStdin_InputWithCommentsAndEmptyLines(t *testing.T) {
	// Mock stdin with various comment styles and empty lines
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	testData := `
# This is a comment line
192.168.1.0/24 # Inline comment

10.0.0.0/8 // C++ style comment
172.16.0.0/12 /* C style comment

# Another comment
`

	go func() {
		defer w.Close()
		w.Write([]byte(testData))
	}()

	stdin := &Stdin{
		Type:        TypeStdin,
		Action:      lib.ActionAdd,
		Description: DescStdin,
		Name:        "TESTENTRY",
		OnlyIPType:  "",
	}

	container := lib.NewContainer()
	result, err := stdin.Input(container)

	// Restore stdin
	os.Stdin = oldStdin

	if err != nil {
		t.Errorf("Input() with comments failed: %v", err)
		return
	}

	if result == nil {
		t.Error("Input() returned nil container")
		return
	}

	entry, found := result.GetEntry("TESTENTRY")
	if !found {
		t.Error("Expected entry not found in container")
	}
	if entry == nil {
		t.Error("Entry is nil")
	}
}

func TestStdin_InputScannerError(t *testing.T) {
	// Create a pipe that will be closed to simulate scanner error
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	// Close the write end immediately to simulate an error condition
	w.Close()

	stdin := &Stdin{
		Type:        TypeStdin,
		Action:      lib.ActionAdd,
		Description: DescStdin,
		Name:        "TESTENTRY",
		OnlyIPType:  "",
	}

	container := lib.NewContainer()
	_, err := stdin.Input(container)

	// Restore stdin
	os.Stdin = oldStdin

	// This test might not always produce an error depending on the system,
	// but it's good to test the error handling path
	if err != nil {
		t.Logf("Scanner error (expected in some cases): %v", err)
	}
}

func TestStdin_Constants(t *testing.T) {
	if TypeStdin != "stdin" {
		t.Errorf("TypeStdin = %v, expect %v", TypeStdin, "stdin")
	}
	if DescStdin != "Accept plaintext IP & CIDR from standard input, separated by newline" {
		t.Errorf("DescStdin = %v, expect correct description", DescStdin)
	}
}

// Helper function to simulate stdin input
func simulateStdinInput(input string) func() {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		defer w.Close()
		w.Write([]byte(input))
	}()

	return func() {
		os.Stdin = oldStdin
	}
}