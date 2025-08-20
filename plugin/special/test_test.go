package special

import (
	"encoding/json"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestTestConstants(t *testing.T) {
	if entryNameTest != "test" {
		t.Errorf("entryNameTest should be 'test', got: %s", entryNameTest)
	}
	if typeTest != "test" {
		t.Errorf("typeTest should be 'test', got: %s", typeTest)
	}
	if descTest != "Convert specific CIDR to other formats (for test only)" {
		t.Errorf("descTest should be correct description, got: %s", descTest)
	}
}

func TestTestCIDRs(t *testing.T) {
	if len(testCIDRs) == 0 {
		t.Error("testCIDRs should not be empty")
	}
	
	expectedCIDR := "127.0.0.0/8"
	found := false
	for _, cidr := range testCIDRs {
		if cidr == expectedCIDR {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("testCIDRs should contain %s", expectedCIDR)
	}
}

func TestNewTest(t *testing.T) {
	tests := []struct {
		name   string
		action lib.Action
		data   json.RawMessage
	}{
		{
			name:   "ActionAdd",
			action: lib.ActionAdd,
			data:   json.RawMessage(`{}`),
		},
		{
			name:   "ActionRemove",
			action: lib.ActionRemove,
			data:   json.RawMessage(`{}`),
		},
		{
			name:   "ActionOutput",
			action: lib.ActionOutput,
			data:   json.RawMessage(`{}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter, err := newTest(tt.action, tt.data)
			if err != nil {
				t.Errorf("newTest() should not return error: %v", err)
			}
			if converter == nil {
				t.Error("newTest() should return non-nil converter")
			}

			// Test interface methods
			if converter.GetType() != typeTest {
				t.Errorf("GetType() = %s; want %s", converter.GetType(), typeTest)
			}
			if converter.GetAction() != tt.action {
				t.Errorf("GetAction() = %s; want %s", converter.GetAction(), tt.action)
			}
			if converter.GetDescription() != descTest {
				t.Errorf("GetDescription() = %s; want %s", converter.GetDescription(), descTest)
			}
		})
	}
}

func TestTestStruct(t *testing.T) {
	testConverter := &test{
		Type:        "custom-type",
		Action:      lib.ActionAdd,
		Description: "custom description",
	}

	if testConverter.GetType() != "custom-type" {
		t.Errorf("GetType() = %s; want custom-type", testConverter.GetType())
	}
	if testConverter.GetAction() != lib.ActionAdd {
		t.Errorf("GetAction() = %s; want %s", testConverter.GetAction(), lib.ActionAdd)
	}
	if testConverter.GetDescription() != "custom description" {
		t.Errorf("GetDescription() = %s; want custom description", testConverter.GetDescription())
	}
}

func TestTestInput_Add(t *testing.T) {
	converter := &test{
		Type:        typeTest,
		Action:      lib.ActionAdd,
		Description: descTest,
	}

	container := lib.NewContainer()
	initialLen := container.Len()

	result, err := converter.Input(container)
	if err != nil {
		t.Errorf("Input() should not return error: %v", err)
	}
	if result == nil {
		t.Error("Input() should return non-nil container")
	}

	// Check that entry was added
	if result.Len() != initialLen+1 {
		t.Errorf("Container should have %d entries after add, got %d", initialLen+1, result.Len())
	}

	// Check that the test entry exists
	entry, found := result.GetEntry(entryNameTest)
	if !found {
		t.Errorf("Container should contain entry with name %s", entryNameTest)
	}
	if entry.GetName() != "TEST" { // Entry names are converted to uppercase
		t.Errorf("Entry name should be TEST, got %s", entry.GetName())
	}
}

func TestTestInput_Remove(t *testing.T) {
	// First add an entry to remove
	container := lib.NewContainer()
	entry := lib.NewEntry(entryNameTest)
	for _, cidr := range testCIDRs {
		err := entry.AddPrefix(cidr)
		if err != nil {
			t.Fatalf("AddPrefix failed: %v", err)
		}
	}
	err := container.Add(entry)
	if err != nil {
		t.Fatalf("Add entry failed: %v", err)
	}

	converter := &test{
		Type:        typeTest,
		Action:      lib.ActionRemove,
		Description: descTest,
	}

	result, err := converter.Input(container)
	if err != nil {
		t.Errorf("Input() with remove action should not return error: %v", err)
	}
	if result == nil {
		t.Error("Input() should return non-nil container")
	}

	// The container should still exist but might have modified contents
	// The exact behavior depends on the Remove implementation
}

func TestTestInput_UnknownAction(t *testing.T) {
	converter := &test{
		Type:        typeTest,
		Action:      lib.Action("unknown"),
		Description: descTest,
	}

	container := lib.NewContainer()

	_, err := converter.Input(container)
	if err != lib.ErrUnknownAction {
		t.Errorf("Input() with unknown action should return ErrUnknownAction, got: %v", err)
	}
}

func TestTestInput_InvalidCIDR(t *testing.T) {
	// Temporarily modify testCIDRs to include an invalid CIDR
	originalCIDRs := testCIDRs
	testCIDRs = []string{"invalid-cidr"}
	defer func() {
		testCIDRs = originalCIDRs
	}()

	converter := &test{
		Type:        typeTest,
		Action:      lib.ActionAdd,
		Description: descTest,
	}

	container := lib.NewContainer()

	_, err := converter.Input(container)
	if err == nil {
		t.Error("Input() with invalid CIDR should return error")
	}
}

func TestTestInput_Integration(t *testing.T) {
	// Test the full workflow: create converter, add to container, verify results
	converter, err := newTest(lib.ActionAdd, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("newTest() failed: %v", err)
	}

	container := lib.NewContainer()
	
	// Add test entry
	result, err := converter.Input(container)
	if err != nil {
		t.Fatalf("Input() failed: %v", err)
	}

	// Verify the entry was added correctly
	entry, found := result.GetEntry(entryNameTest)
	if !found {
		t.Fatal("Test entry should be found in container")
	}

	// Verify the entry contains the expected CIDR
	cidrs, err := entry.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() failed: %v", err)
	}

	if len(cidrs) == 0 {
		t.Error("Entry should contain at least one CIDR")
	}

	// Check that the test CIDR is included
	found = false
	for _, cidr := range cidrs {
		if cidr == testCIDRs[0] {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Entry should contain test CIDR %s", testCIDRs[0])
	}
}

func TestTestInput_MultipleActions(t *testing.T) {
	container := lib.NewContainer()

	// First add the entry
	addConverter := &test{
		Type:        typeTest,
		Action:      lib.ActionAdd,
		Description: descTest,
	}

	result, err := addConverter.Input(container)
	if err != nil {
		t.Fatalf("Add action failed: %v", err)
	}

	if result.Len() != 1 {
		t.Errorf("Container should have 1 entry after add, got %d", result.Len())
	}

	// Then try to remove it
	removeConverter := &test{
		Type:        typeTest,
		Action:      lib.ActionRemove,
		Description: descTest,
	}

	result, err = removeConverter.Input(result)
	if err != nil {
		t.Errorf("Remove action should not return error: %v", err)
	}

	// The exact behavior after remove depends on the implementation
	// Just verify it doesn't crash
	if result == nil {
		t.Error("Remove action should return non-nil container")
	}
}