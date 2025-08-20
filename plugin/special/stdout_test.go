package special

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestStdout_NewStdout(t *testing.T) {
	tests := []struct {
		name         string
		action       lib.Action
		data         json.RawMessage
		expectType   string
		expectIPType lib.IPType
		expectWant   []string
		expectErr    bool
	}{
		{
			name:         "Valid action with no data",
			action:       lib.ActionOutput,
			data:         nil,
			expectType:   TypeStdout,
			expectIPType: "",
			expectWant:   nil,
			expectErr:    false,
		},
		{
			name:         "Valid action with wanted list",
			action:       lib.ActionOutput,
			data:         json.RawMessage(`{"wantedList": ["test1", "test2"], "onlyIPType": "ipv4"}`),
			expectType:   TypeStdout,
			expectIPType: lib.IPv4,
			expectWant:   []string{"test1", "test2"},
			expectErr:    false,
		},
		{
			name:         "Valid action with excluded list",
			action:       lib.ActionOutput,
			data:         json.RawMessage(`{"excludedList": ["exclude1"], "onlyIPType": "ipv6"}`),
			expectType:   TypeStdout,
			expectIPType: lib.IPv6,
			expectErr:    false,
		},
		{
			name:      "Invalid JSON data",
			action:    lib.ActionOutput,
			data:      json.RawMessage(`{invalid json}`),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter, err := newStdout(tt.action, tt.data)
			if (err != nil) != tt.expectErr {
				t.Errorf("newStdout() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr {
				stdout := converter.(*Stdout)
				if stdout.GetType() != tt.expectType {
					t.Errorf("GetType() = %v, expect %v", stdout.GetType(), tt.expectType)
				}
				if stdout.GetAction() != tt.action {
					t.Errorf("GetAction() = %v, expect %v", stdout.GetAction(), tt.action)
				}
				if stdout.OnlyIPType != tt.expectIPType {
					t.Errorf("OnlyIPType = %v, expect %v", stdout.OnlyIPType, tt.expectIPType)
				}
				if tt.expectWant != nil {
					if len(stdout.Want) != len(tt.expectWant) {
						t.Errorf("Want length = %v, expect %v", len(stdout.Want), len(tt.expectWant))
					}
					for i, want := range tt.expectWant {
						if i < len(stdout.Want) && stdout.Want[i] != want {
							t.Errorf("Want[%d] = %v, expect %v", i, stdout.Want[i], want)
						}
					}
				}
			}
		})
	}
}

func TestStdout_GetType(t *testing.T) {
	stdout := &Stdout{Type: TypeStdout}
	result := stdout.GetType()
	if result != TypeStdout {
		t.Errorf("GetType() = %v, expect %v", result, TypeStdout)
	}
}

func TestStdout_GetAction(t *testing.T) {
	action := lib.ActionOutput
	stdout := &Stdout{Action: action}
	result := stdout.GetAction()
	if result != action {
		t.Errorf("GetAction() = %v, expect %v", result, action)
	}
}

func TestStdout_GetDescription(t *testing.T) {
	stdout := &Stdout{Description: DescStdout}
	result := stdout.GetDescription()
	if result != DescStdout {
		t.Errorf("GetDescription() = %v, expect %v", result, DescStdout)
	}
}

func TestStdout_Output(t *testing.T) {
	// Create a container with test entries
	container := lib.NewContainer()
	
	// Add a test entry
	entry := lib.NewEntry("TEST1")
	if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("Failed to add prefix: %v", err)
	}
	if err := container.Add(entry); err != nil {
		t.Fatalf("Failed to add entry to container: %v", err)
	}

	tests := []struct {
		name       string
		stdout     *Stdout
		expectErr  bool
	}{
		{
			name: "Output all entries",
			stdout: &Stdout{
				Type:        TypeStdout,
				Action:      lib.ActionOutput,
				Description: DescStdout,
			},
			expectErr: false,
		},
		{
			name: "Output with wanted list",
			stdout: &Stdout{
				Type:        TypeStdout,
				Action:      lib.ActionOutput,
				Description: DescStdout,
				Want:        []string{"TEST1"},
			},
			expectErr: false,
		},
		{
			name: "Output with excluded list",
			stdout: &Stdout{
				Type:        TypeStdout,
				Action:      lib.ActionOutput,
				Description: DescStdout,
				Exclude:     []string{"TEST1"},
			},
			expectErr: false,
		},
		{
			name: "Output with IPv4 only",
			stdout: &Stdout{
				Type:        TypeStdout,
				Action:      lib.ActionOutput,
				Description: DescStdout,
				OnlyIPType:  lib.IPv4,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := tt.stdout.Output(container)

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			// Read captured output
			var buf bytes.Buffer
			io.Copy(&buf, r)
			r.Close()

			if (err != nil) != tt.expectErr {
				t.Errorf("Output() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			if !tt.expectErr {
				output := buf.String()
				// For most cases, we expect some output unless excluded
				if tt.stdout.Exclude != nil && len(tt.stdout.Exclude) > 0 {
					// If TEST1 is excluded, output should be empty
					if len(output) > 0 {
						t.Logf("Output (excluded): %s", output)
					}
				} else if tt.stdout.Want != nil && len(tt.stdout.Want) > 0 {
					// If TEST1 is wanted, we should have output
					if len(output) == 0 {
						t.Error("Expected output for wanted entry")
					}
				}
			}
		})
	}
}

func TestStdout_FilterAndSortList(t *testing.T) {
	container := lib.NewContainer()

	// Add test entries
	entry1 := lib.NewEntry("TEST1")
	entry2 := lib.NewEntry("TEST2")
	entry3 := lib.NewEntry("EXCLUDE")

	if err := entry1.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("Failed to add prefix to entry1: %v", err)
	}
	if err := entry2.AddPrefix("192.168.2.0/24"); err != nil {
		t.Fatalf("Failed to add prefix to entry2: %v", err)
	}
	if err := entry3.AddPrefix("192.168.3.0/24"); err != nil {
		t.Fatalf("Failed to add prefix to entry3: %v", err)
	}

	if err := container.Add(entry1); err != nil {
		t.Fatalf("Failed to add entry1: %v", err)
	}
	if err := container.Add(entry2); err != nil {
		t.Fatalf("Failed to add entry2: %v", err)
	}
	if err := container.Add(entry3); err != nil {
		t.Fatalf("Failed to add entry3: %v", err)
	}

	tests := []struct {
		name     string
		stdout   *Stdout
		expected []string
	}{
		{
			name: "No filters",
			stdout: &Stdout{
				Want:    nil,
				Exclude: nil,
			},
			expected: []string{"EXCLUDE", "TEST1", "TEST2"}, // Sorted
		},
		{
			name: "With wanted list",
			stdout: &Stdout{
				Want:    []string{"TEST1", "TEST2"},
				Exclude: nil,
			},
			expected: []string{"TEST1", "TEST2"},
		},
		{
			name: "With excluded list",
			stdout: &Stdout{
				Want:    nil,
				Exclude: []string{"EXCLUDE"},
			},
			expected: []string{"TEST1", "TEST2"},
		},
		{
			name: "With both wanted and excluded",
			stdout: &Stdout{
				Want:    []string{"TEST1", "TEST2", "EXCLUDE"},
				Exclude: []string{"EXCLUDE"},
			},
			expected: []string{"TEST1", "TEST2"},
		},
		{
			name: "Empty wanted list",
			stdout: &Stdout{
				Want:    []string{},
				Exclude: []string{"TEST1"},
			},
			expected: []string{"EXCLUDE", "TEST2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.stdout.filterAndSortList(container)
			if len(result) != len(tt.expected) {
				t.Errorf("filterAndSortList() length = %v, expect %v", len(result), len(tt.expected))
				t.Errorf("Got: %v", result)
				t.Errorf("Expected: %v", tt.expected)
				return
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("filterAndSortList()[%d] = %v, expect %v", i, result[i], expected)
				}
			}
		})
	}
}

func TestStdout_GenerateCIDRList(t *testing.T) {
	tests := []struct {
		name       string
		onlyIPType lib.IPType
		prefix     string
		expectErr  bool
	}{
		{
			name:       "All IP types",
			onlyIPType: "",
			prefix:     "192.168.1.0/24",
			expectErr:  false,
		},
		{
			name:       "IPv4 only",
			onlyIPType: lib.IPv4,
			prefix:     "192.168.1.0/24",
			expectErr:  false,
		},
		{
			name:       "IPv6 only with IPv6 prefix",
			onlyIPType: lib.IPv6,
			prefix:     "2001:db8::/32",
			expectErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout := &Stdout{
				OnlyIPType: tt.onlyIPType,
			}

			entry := lib.NewEntry("TEST")
			if err := entry.AddPrefix(tt.prefix); err != nil {
				t.Fatalf("Failed to add prefix: %v", err)
			}

			result, err := stdout.generateCIDRList(entry)
			if (err != nil) != tt.expectErr {
				t.Errorf("generateCIDRList() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			if !tt.expectErr {
				if len(result) == 0 {
					t.Error("generateCIDRList() returned empty list")
				}
			}
		})
	}
}

func TestStdout_GenerateCIDRListEmpty(t *testing.T) {
	stdout := &Stdout{}
	entry := lib.NewEntry("EMPTY")
	
	_, err := stdout.generateCIDRList(entry)
	if err == nil {
		t.Error("generateCIDRList() should return error for empty entry")
	}
}

func TestStdout_Constants(t *testing.T) {
	if TypeStdout != "stdout" {
		t.Errorf("TypeStdout = %v, expect %v", TypeStdout, "stdout")
	}
	if DescStdout != "Convert data to plaintext CIDR format and output to standard output" {
		t.Errorf("DescStdout = %v, expect correct description", DescStdout)
	}
}