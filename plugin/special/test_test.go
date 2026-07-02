package special

import (
	"encoding/json"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestTestPlugin_NewTest(t *testing.T) {
	tests := []struct {
		name       string
		action     lib.Action
		data       json.RawMessage
		expectType string
		expectErr  bool
	}{
		{
			name:       "Valid action add",
			action:     lib.ActionAdd,
			data:       json.RawMessage(`{}`),
			expectType: typeTest,
			expectErr:  false,
		},
		{
			name:       "Valid action remove",
			action:     lib.ActionRemove,
			data:       json.RawMessage(`{}`),
			expectType: typeTest,
			expectErr:  false,
		},
		{
			name:       "Empty data",
			action:     lib.ActionAdd,
			data:       nil,
			expectType: typeTest,
			expectErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter, err := newTest(tt.action, tt.data)
			if (err != nil) != tt.expectErr {
				t.Errorf("newTest() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr {
				if converter.GetType() != tt.expectType {
					t.Errorf("GetType() = %v, expect %v", converter.GetType(), tt.expectType)
				}
				if converter.GetAction() != tt.action {
					t.Errorf("GetAction() = %v, expect %v", converter.GetAction(), tt.action)
				}
				if converter.GetDescription() != descTest {
					t.Errorf("GetDescription() = %v, expect %v", converter.GetDescription(), descTest)
				}
			}
		})
	}
}

func TestTestPlugin_GetType(t *testing.T) {
	testPlugin := &test{Type: typeTest}
	result := testPlugin.GetType()
	if result != typeTest {
		t.Errorf("GetType() = %v, expect %v", result, typeTest)
	}
}

func TestTestPlugin_GetAction(t *testing.T) {
	action := lib.ActionAdd
	testPlugin := &test{Action: action}
	result := testPlugin.GetAction()
	if result != action {
		t.Errorf("GetAction() = %v, expect %v", result, action)
	}
}

func TestTestPlugin_GetDescription(t *testing.T) {
	testPlugin := &test{Description: descTest}
	result := testPlugin.GetDescription()
	if result != descTest {
		t.Errorf("GetDescription() = %v, expect %v", result, descTest)
	}
}

func TestTestPlugin_Input(t *testing.T) {
	tests := []struct {
		name      string
		action    lib.Action
		expectErr bool
	}{
		{
			name:      "Action add",
			action:    lib.ActionAdd,
			expectErr: false,
		},
		{
			name:      "Invalid action",
			action:    lib.Action("invalid"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPlugin := &test{
				Type:        typeTest,
				Action:      tt.action,
				Description: descTest,
			}

			container := lib.NewContainer()
			result, err := testPlugin.Input(container)

			if (err != nil) != tt.expectErr {
				t.Errorf("Input() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			if !tt.expectErr {
				if result == nil {
					t.Error("Input() returned nil container")
					return
				}

				if tt.action == lib.ActionAdd {
					entry, found := result.GetEntry(entryNameTest)
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

func TestTestPlugin_InputRemove(t *testing.T) {
	container := lib.NewContainer()
	
	// Pre-populate the container with the test entry
	entry := lib.NewEntry(entryNameTest)
	for _, cidr := range testCIDRs {
		if err := entry.AddPrefix(cidr); err != nil {
			t.Fatalf("Failed to add prefix: %v", err)
		}
	}
	if err := container.Add(entry); err != nil {
		t.Fatalf("Failed to add entry to container: %v", err)
	}

	// Now test the remove action
	testPlugin := &test{
		Type:        typeTest,
		Action:      lib.ActionRemove,
		Description: descTest,
	}
	
	result, err := testPlugin.Input(container)
	if err != nil {
		t.Errorf("Remove action failed: %v", err)
		return
	}
	
	if result == nil {
		t.Error("Input() returned nil container")
	}
}

func TestTestPlugin_InputWithInvalidCIDR(t *testing.T) {
	// Mock testCIDRs with invalid CIDR to test error handling
	originalCIDRs := testCIDRs
	testCIDRs = []string{"invalid-cidr"}
	defer func() { testCIDRs = originalCIDRs }()

	testPlugin := &test{
		Type:        typeTest,
		Action:      lib.ActionAdd,
		Description: descTest,
	}

	container := lib.NewContainer()
	_, err := testPlugin.Input(container)

	if err == nil {
		t.Error("Expected error for invalid CIDR, got nil")
	}
}

func TestTestPlugin_Constants(t *testing.T) {
	if entryNameTest != "test" {
		t.Errorf("entryNameTest = %v, expect %v", entryNameTest, "test")
	}
	if typeTest != "test" {
		t.Errorf("typeTest = %v, expect %v", typeTest, "test")
	}
	if descTest != "Convert specific CIDR to other formats (for test only)" {
		t.Errorf("descTest = %v, expect correct description", descTest)
	}
}

func TestTestPlugin_TestCIDRs(t *testing.T) {
	expectedCIDRs := []string{"127.0.0.0/8"}
	if len(testCIDRs) != len(expectedCIDRs) {
		t.Errorf("testCIDRs length = %v, expect %v", len(testCIDRs), len(expectedCIDRs))
	}
	for i, cidr := range testCIDRs {
		if cidr != expectedCIDRs[i] {
			t.Errorf("testCIDRs[%d] = %v, expect %v", i, cidr, expectedCIDRs[i])
		}
	}
}