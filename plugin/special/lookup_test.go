package special

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestLookup_NewLookup(t *testing.T) {
	tests := []struct {
		name           string
		action         lib.Action
		data           json.RawMessage
		expectType     string
		expectSearch   string
		expectSearchList []string
		expectErr      bool
	}{
		{
			name:           "Valid lookup with IP",
			action:         lib.ActionOutput,
			data:           json.RawMessage(`{"search": "192.168.1.1"}`),
			expectType:     TypeLookup,
			expectSearch:   "192.168.1.1",
			expectSearchList: nil,
			expectErr:      false,
		},
		{
			name:           "Valid lookup with CIDR",
			action:         lib.ActionOutput,
			data:           json.RawMessage(`{"search": "192.168.1.0/24"}`),
			expectType:     TypeLookup,
			expectSearch:   "192.168.1.0/24",
			expectSearchList: nil,
			expectErr:      false,
		},
		{
			name:           "Valid lookup with search list",
			action:         lib.ActionOutput,
			data:           json.RawMessage(`{"search": "192.168.1.1", "searchList": ["list1", "list2"]}`),
			expectType:     TypeLookup,
			expectSearch:   "192.168.1.1",
			expectSearchList: []string{"list1", "list2"},
			expectErr:      false,
		},
		{
			name:      "Missing search",
			action:    lib.ActionOutput,
			data:      json.RawMessage(`{}`),
			expectErr: true,
		},
		{
			name:      "Empty search",
			action:    lib.ActionOutput,
			data:      json.RawMessage(`{"search": ""}`),
			expectErr: true,
		},
		{
			name:      "Whitespace only search",
			action:    lib.ActionOutput,
			data:      json.RawMessage(`{"search": "   "}`),
			expectErr: true,
		},
		{
			name:      "Invalid JSON",
			action:    lib.ActionOutput,
			data:      json.RawMessage(`{invalid json}`),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter, err := newLookup(tt.action, tt.data)
			if (err != nil) != tt.expectErr {
				t.Errorf("newLookup() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr {
				lookup := converter.(*Lookup)
				if lookup.GetType() != tt.expectType {
					t.Errorf("GetType() = %v, expect %v", lookup.GetType(), tt.expectType)
				}
				if lookup.GetAction() != tt.action {
					t.Errorf("GetAction() = %v, expect %v", lookup.GetAction(), tt.action)
				}
				if lookup.Search != tt.expectSearch {
					t.Errorf("Search = %v, expect %v", lookup.Search, tt.expectSearch)
				}
				if len(lookup.SearchList) != len(tt.expectSearchList) {
					t.Errorf("SearchList length = %v, expect %v", len(lookup.SearchList), len(tt.expectSearchList))
				}
				for i, item := range tt.expectSearchList {
					if i < len(lookup.SearchList) && lookup.SearchList[i] != item {
						t.Errorf("SearchList[%d] = %v, expect %v", i, lookup.SearchList[i], item)
					}
				}
			}
		})
	}
}

func TestLookup_GetType(t *testing.T) {
	lookup := &Lookup{Type: TypeLookup}
	result := lookup.GetType()
	if result != TypeLookup {
		t.Errorf("GetType() = %v, expect %v", result, TypeLookup)
	}
}

func TestLookup_GetAction(t *testing.T) {
	action := lib.ActionOutput
	lookup := &Lookup{Action: action}
	result := lookup.GetAction()
	if result != action {
		t.Errorf("GetAction() = %v, expect %v", result, action)
	}
}

func TestLookup_GetDescription(t *testing.T) {
	lookup := &Lookup{Description: DescLookup}
	result := lookup.GetDescription()
	if result != DescLookup {
		t.Errorf("GetDescription() = %v, expect %v", result, DescLookup)
	}
}

func TestLookup_Output(t *testing.T) {
	tests := []struct {
		name         string
		search       string
		searchList   []string
		expectOutput string
		expectErr    bool
	}{
		{
			name:         "Valid IP found",
			search:       "192.168.1.1",
			searchList:   nil,
			expectOutput: "test\n",
			expectErr:    false,
		},
		{
			name:         "Valid CIDR found",
			search:       "192.168.1.0/24",
			searchList:   nil,
			expectOutput: "test\n",
			expectErr:    false,
		},
		{
			name:         "IP not found",
			search:       "10.0.0.1",
			searchList:   nil,
			expectOutput: "false\n",
			expectErr:    false,
		},
		{
			name:         "Valid IP with search list",
			search:       "192.168.1.1",
			searchList:   []string{"TEST"},
			expectOutput: "test\n",
			expectErr:    false,
		},
		{
			name:         "Valid IP with non-matching search list",
			search:       "192.168.1.1",
			searchList:   []string{"NONEXISTENT"},
			expectOutput: "false\n",
			expectErr:    false,
		},
		{
			name:      "Invalid IP",
			search:    "invalid-ip",
			expectErr: true,
		},
		{
			name:      "Invalid CIDR",
			search:    "192.168.1.0/99",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a container with test data
			container := lib.NewContainer()
			entry := lib.NewEntry("TEST")
			if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
				t.Fatalf("Failed to add prefix: %v", err)
			}
			if err := container.Add(entry); err != nil {
				t.Fatalf("Failed to add entry: %v", err)
			}

			lookup := &Lookup{
				Type:        TypeLookup,
				Action:      lib.ActionOutput,
				Description: DescLookup,
				Search:      tt.search,
				SearchList:  tt.searchList,
			}

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := lookup.Output(container)

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
				if output != tt.expectOutput {
					t.Errorf("Output() = %q, expect %q", output, tt.expectOutput)
				}
			}
		})
	}
}

func TestLookup_OutputMultipleMatches(t *testing.T) {
	// Create a container with multiple entries that match
	container := lib.NewContainer()
	
	entry1 := lib.NewEntry("TEST1")
	if err := entry1.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("Failed to add prefix to entry1: %v", err)
	}
	if err := container.Add(entry1); err != nil {
		t.Fatalf("Failed to add entry1: %v", err)
	}

	entry2 := lib.NewEntry("TEST2")
	if err := entry2.AddPrefix("192.168.0.0/16"); err != nil {
		t.Fatalf("Failed to add prefix to entry2: %v", err)
	}
	if err := container.Add(entry2); err != nil {
		t.Fatalf("Failed to add entry2: %v", err)
	}

	lookup := &Lookup{
		Type:        TypeLookup,
		Action:      lib.ActionOutput,
		Description: DescLookup,
		Search:      "192.168.1.1",
		SearchList:  nil,
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := lookup.Output(container)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	if err != nil {
		t.Errorf("Output() with multiple matches failed: %v", err)
		return
	}

	output := buf.String()
	// Should contain both entries, sorted alphabetically
	expected := "test1,test2\n"
	if output != expected {
		t.Errorf("Output() = %q, expect %q", output, expected)
	}
}

func TestLookup_OutputIPv6(t *testing.T) {
	// Create a container with IPv6 entry
	container := lib.NewContainer()
	entry := lib.NewEntry("IPV6TEST")
	if err := entry.AddPrefix("2001:db8::/32"); err != nil {
		t.Fatalf("Failed to add IPv6 prefix: %v", err)
	}
	if err := container.Add(entry); err != nil {
		t.Fatalf("Failed to add IPv6 entry: %v", err)
	}

	lookup := &Lookup{
		Type:        TypeLookup,
		Action:      lib.ActionOutput,
		Description: DescLookup,
		Search:      "2001:db8::1",
		SearchList:  nil,
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := lookup.Output(container)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	if err != nil {
		t.Errorf("Output() with IPv6 failed: %v", err)
		return
	}

	output := buf.String()
	expected := "ipv6test\n"
	if output != expected {
		t.Errorf("Output() = %q, expect %q", output, expected)
	}
}

func TestLookup_Constants(t *testing.T) {
	if TypeLookup != "lookup" {
		t.Errorf("TypeLookup = %v, expect %v", TypeLookup, "lookup")
	}
	if DescLookup != "Lookup specified IP or CIDR from various formats of data" {
		t.Errorf("DescLookup = %v, expect correct description", DescLookup)
	}
}