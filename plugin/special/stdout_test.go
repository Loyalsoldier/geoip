package special

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestStdoutConstants(t *testing.T) {
	if TypeStdout != "stdout" {
		t.Errorf("TypeStdout should be 'stdout', got: %s", TypeStdout)
	}
	if DescStdout != "Convert data to plaintext CIDR format and output to standard output" {
		t.Errorf("DescStdout should be correct description, got: %s", DescStdout)
	}
}

func TestNewStdout(t *testing.T) {
	tests := []struct {
		name         string
		action       lib.Action
		data         string
		expectError  bool
		expectWant   []string
		expectExclude []string
		expectIPType lib.IPType
	}{
		{
			name:         "Valid empty config",
			action:       lib.ActionOutput,
			data:         `{}`,
			expectError:  false,
			expectWant:   nil,
			expectExclude: nil,
			expectIPType: "",
		},
		{
			name:         "Valid config with want list",
			action:       lib.ActionOutput,
			data:         `{"wantedList": ["entry1", "entry2"]}`,
			expectError:  false,
			expectWant:   []string{"entry1", "entry2"},
			expectExclude: nil,
			expectIPType: "",
		},
		{
			name:         "Valid config with exclude list",
			action:       lib.ActionOutput,
			data:         `{"excludedList": ["exclude1", "exclude2"]}`,
			expectError:  false,
			expectWant:   nil,
			expectExclude: []string{"exclude1", "exclude2"},
			expectIPType: "",
		},
		{
			name:         "Valid config with IPv4 only",
			action:       lib.ActionOutput,
			data:         `{"onlyIPType": "ipv4"}`,
			expectError:  false,
			expectWant:   nil,
			expectExclude: nil,
			expectIPType: lib.IPv4,
		},
		{
			name:         "Valid config with IPv6 only",
			action:       lib.ActionOutput,
			data:         `{"onlyIPType": "ipv6"}`,
			expectError:  false,
			expectWant:   nil,
			expectExclude: nil,
			expectIPType: lib.IPv6,
		},
		{
			name:         "Complete config",
			action:       lib.ActionOutput,
			data:         `{"wantedList": ["want1"], "excludedList": ["exclude1"], "onlyIPType": "ipv4"}`,
			expectError:  false,
			expectWant:   []string{"want1"},
			expectExclude: []string{"exclude1"},
			expectIPType: lib.IPv4,
		},
		{
			name:        "Invalid JSON",
			action:      lib.ActionOutput,
			data:        `{invalid json}`,
			expectError: true,
		},
		{
			name:         "Empty data",
			action:       lib.ActionOutput,
			data:         ``,
			expectError:  false,
			expectWant:   nil,
			expectExclude: nil,
			expectIPType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter, err := newStdout(tt.action, json.RawMessage(tt.data))

			if tt.expectError && err == nil {
				t.Errorf("newStdout() should return error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("newStdout() should not return error but got: %v", err)
			}

			if !tt.expectError {
				if converter == nil {
					t.Error("newStdout() should return non-nil converter")
				} else {
					stdout := converter.(*Stdout)
					if stdout.GetType() != TypeStdout {
						t.Errorf("GetType() = %s; want %s", stdout.GetType(), TypeStdout)
					}
					if stdout.GetAction() != tt.action {
						t.Errorf("GetAction() = %s; want %s", stdout.GetAction(), tt.action)
					}
					if stdout.GetDescription() != DescStdout {
						t.Errorf("GetDescription() = %s; want %s", stdout.GetDescription(), DescStdout)
					}
					if len(stdout.Want) != len(tt.expectWant) {
						t.Errorf("Want length = %d; want %d", len(stdout.Want), len(tt.expectWant))
					}
					if len(stdout.Exclude) != len(tt.expectExclude) {
						t.Errorf("Exclude length = %d; want %d", len(stdout.Exclude), len(tt.expectExclude))
					}
					if stdout.OnlyIPType != tt.expectIPType {
						t.Errorf("OnlyIPType = %s; want %s", stdout.OnlyIPType, tt.expectIPType)
					}
				}
			}
		})
	}
}

func TestStdoutStruct(t *testing.T) {
	stdout := &Stdout{
		Type:        "custom-stdout",
		Action:      lib.ActionOutput,
		Description: "custom description",
		Want:        []string{"want1", "want2"},
		Exclude:     []string{"exclude1"},
		OnlyIPType:  lib.IPv4,
	}

	if stdout.GetType() != "custom-stdout" {
		t.Errorf("GetType() = %s; want custom-stdout", stdout.GetType())
	}
	if stdout.GetAction() != lib.ActionOutput {
		t.Errorf("GetAction() = %s; want %s", stdout.GetAction(), lib.ActionOutput)
	}
	if stdout.GetDescription() != "custom description" {
		t.Errorf("GetDescription() = %s; want custom description", stdout.GetDescription())
	}
}

func TestStdoutFilterAndSortList(t *testing.T) {
	// Create a container with test entries
	container := lib.NewContainer()
	
	// Add test entries
	entries := []string{"entry1", "entry2", "entry3", "excluded"}
	for _, name := range entries {
		entry := lib.NewEntry(name)
		err := entry.AddPrefix("127.0.0.1")
		if err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
		err = container.Add(entry)
		if err != nil {
			t.Fatalf("Add entry failed: %v", err)
		}
	}

	tests := []struct {
		name           string
		want           []string
		exclude        []string
		expectedLength int
		shouldContain  []string
		shouldNotContain []string
	}{
		{
			name:           "No filters",
			want:           nil,
			exclude:        nil,
			expectedLength: 4,
			shouldContain:  []string{"ENTRY1", "ENTRY2", "ENTRY3", "EXCLUDED"},
			shouldNotContain: nil,
		},
		{
			name:           "Want list only",
			want:           []string{"entry1", "entry3"},
			exclude:        nil,
			expectedLength: 2,
			shouldContain:  []string{"ENTRY1", "ENTRY3"},
			shouldNotContain: []string{"ENTRY2", "EXCLUDED"},
		},
		{
			name:           "Exclude list only",
			want:           nil,
			exclude:        []string{"excluded"},
			expectedLength: 3,
			shouldContain:  []string{"ENTRY1", "ENTRY2", "ENTRY3"},
			shouldNotContain: []string{"EXCLUDED"},
		},
		{
			name:           "Want and exclude lists",
			want:           []string{"entry1", "entry2", "excluded"},
			exclude:        []string{"excluded"},
			expectedLength: 2,
			shouldContain:  []string{"ENTRY1", "ENTRY2"},
			shouldNotContain: []string{"ENTRY3", "EXCLUDED"},
		},
		{
			name:           "Empty want list",
			want:           []string{},
			exclude:        nil,
			expectedLength: 4,
			shouldContain:  []string{"ENTRY1", "ENTRY2", "ENTRY3", "EXCLUDED"},
			shouldNotContain: nil,
		},
		{
			name:           "Want list with spaces and case variations",
			want:           []string{"  ENTRY1  ", "entry2", "Entry3"},
			exclude:        nil,
			expectedLength: 3,
			shouldContain:  []string{"ENTRY1", "ENTRY2", "ENTRY3"},
			shouldNotContain: []string{"EXCLUDED"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout := &Stdout{
				Want:    tt.want,
				Exclude: tt.exclude,
			}

			result := stdout.filterAndSortList(container)

			if len(result) != tt.expectedLength {
				t.Errorf("filterAndSortList() length = %d; want %d", len(result), tt.expectedLength)
			}

			// Check that expected items are present
			for _, expected := range tt.shouldContain {
				found := false
				for _, item := range result {
					if item == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("filterAndSortList() should contain %s", expected)
				}
			}

			// Check that excluded items are not present
			for _, excluded := range tt.shouldNotContain {
				for _, item := range result {
					if item == excluded {
						t.Errorf("filterAndSortList() should not contain %s", excluded)
					}
				}
			}

			// Check that result is sorted
			for i := 1; i < len(result); i++ {
				if result[i-1] > result[i] {
					t.Errorf("filterAndSortList() result should be sorted, but %s > %s", result[i-1], result[i])
				}
			}
		})
	}
}

func TestStdoutGenerateCIDRList(t *testing.T) {
	tests := []struct {
		name       string
		onlyIPType lib.IPType
		prefixes   []string
		expectError bool
		expectContains []string
	}{
		{
			name:       "All IP types",
			onlyIPType: "",
			prefixes:   []string{"192.168.1.0/24", "2001:db8::/32"},
			expectError: false,
			expectContains: []string{"192.168.1.0/24", "2001:db8::/32"},
		},
		{
			name:       "IPv4 only",
			onlyIPType: lib.IPv4,
			prefixes:   []string{"192.168.1.0/24", "2001:db8::/32"},
			expectError: false,
			expectContains: []string{"192.168.1.0/24"},
		},
		{
			name:       "IPv6 only",
			onlyIPType: lib.IPv6,
			prefixes:   []string{"192.168.1.0/24", "2001:db8::/32"},
			expectError: false,
			expectContains: []string{"2001:db8::/32"},
		},
		{
			name:       "Empty entry",
			onlyIPType: "",
			prefixes:   []string{},
			expectError: true,
			expectContains: nil,
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

			stdout := &Stdout{
				OnlyIPType: tt.onlyIPType,
			}

			result, err := stdout.generateCIDRList(entry)

			if tt.expectError && err == nil {
				t.Errorf("generateCIDRList() should return error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("generateCIDRList() should not return error but got: %v", err)
			}

			if !tt.expectError {
				if len(result) == 0 {
					t.Error("generateCIDRList() should return non-empty list")
				}

				// Check that expected CIDRs are present
				for _, expected := range tt.expectContains {
					found := false
					for _, cidr := range result {
						if cidr == expected {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("generateCIDRList() should contain %s", expected)
					}
				}
			}
		})
	}
}

func captureStdout(f func()) string {
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

func TestStdoutOutput(t *testing.T) {
	// Create a container with test entries
	container := lib.NewContainer()
	
	// Add test entries
	testData := map[string][]string{
		"entry1": {"192.168.1.0/24", "10.0.0.0/8"},
		"entry2": {"172.16.0.0/12"},
		"entry3": {"2001:db8::/32"},
	}

	for name, prefixes := range testData {
		entry := lib.NewEntry(name)
		for _, prefix := range prefixes {
			err := entry.AddPrefix(prefix)
			if err != nil {
				t.Fatalf("AddPrefix failed for %s: %v", name, err)
			}
		}
		err := container.Add(entry)
		if err != nil {
			t.Fatalf("Add entry failed for %s: %v", name, err)
		}
	}

	tests := []struct {
		name         string
		want         []string
		exclude      []string
		onlyIPType   lib.IPType
		expectOutput bool
		checkContains []string
	}{
		{
			name:         "Output all entries",
			want:         nil,
			exclude:      nil,
			onlyIPType:   "",
			expectOutput: true,
			checkContains: []string{"192.168.1.0/24", "172.16.0.0/12", "2001:db8::/32"},
		},
		{
			name:         "Output specific entries",
			want:         []string{"entry1"},
			exclude:      nil,
			onlyIPType:   "",
			expectOutput: true,
			checkContains: []string{"192.168.1.0/24", "10.0.0.0/8"},
		},
		{
			name:         "Exclude entries",
			want:         nil,
			exclude:      []string{"entry3"},
			onlyIPType:   "",
			expectOutput: true,
			checkContains: []string{"192.168.1.0/24", "172.16.0.0/12"},
		},
		{
			name:         "IPv4 only",
			want:         nil,
			exclude:      nil,
			onlyIPType:   lib.IPv4,
			expectOutput: true,
			checkContains: []string{"192.168.1.0/24", "172.16.0.0/12"},
		},
		{
			name:         "IPv6 only",
			want:         nil,
			exclude:      nil,
			onlyIPType:   lib.IPv6,
			expectOutput: true,
			checkContains: []string{"2001:db8::/32"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout := &Stdout{
				Type:        TypeStdout,
				Action:      lib.ActionOutput,
				Description: DescStdout,
				Want:        tt.want,
				Exclude:     tt.exclude,
				OnlyIPType:  tt.onlyIPType,
			}

			output := captureStdout(func() {
				err := stdout.Output(container)
				if err != nil {
					t.Errorf("Output() should not return error: %v", err)
				}
			})

			if tt.expectOutput {
				if output == "" {
					t.Error("Output() should write to stdout")
				}

				// Check that expected content is present
				for _, expected := range tt.checkContains {
					if !strings.Contains(output, expected) {
						t.Errorf("Output should contain %s", expected)
					}
				}

				// Check that output ends with newlines
				lines := strings.Split(strings.TrimSpace(output), "\n")
				for _, line := range lines {
					if line == "" {
						continue
					}
					// Each line should be a valid CIDR or IP
					if !strings.Contains(line, ".") && !strings.Contains(line, ":") {
						t.Errorf("Output line should be a valid IP/CIDR: %s", line)
					}
				}
			}
		})
	}
}

func TestStdoutOutput_EmptyContainer(t *testing.T) {
	container := lib.NewContainer()

	stdout := &Stdout{
		Type:        TypeStdout,
		Action:      lib.ActionOutput,
		Description: DescStdout,
	}

	output := captureStdout(func() {
		err := stdout.Output(container)
		if err != nil {
			t.Errorf("Output() should not return error for empty container: %v", err)
		}
	})

	if output != "" {
		t.Errorf("Output() with empty container should produce no output, got: %s", output)
	}
}