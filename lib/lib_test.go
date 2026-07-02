package lib

import (
	"testing"
)

func TestConstants(t *testing.T) {
	// Test Action constants
	if ActionAdd != "add" {
		t.Errorf("ActionAdd should be 'add', got: %s", ActionAdd)
	}
	if ActionRemove != "remove" {
		t.Errorf("ActionRemove should be 'remove', got: %s", ActionRemove)
	}
	if ActionOutput != "output" {
		t.Errorf("ActionOutput should be 'output', got: %s", ActionOutput)
	}

	// Test IPType constants
	if IPv4 != "ipv4" {
		t.Errorf("IPv4 should be 'ipv4', got: %s", IPv4)
	}
	if IPv6 != "ipv6" {
		t.Errorf("IPv6 should be 'ipv6', got: %s", IPv6)
	}

	// Test CaseRemove constants
	if CaseRemovePrefix != 0 {
		t.Errorf("CaseRemovePrefix should be 0, got: %d", CaseRemovePrefix)
	}
	if CaseRemoveEntry != 1 {
		t.Errorf("CaseRemoveEntry should be 1, got: %d", CaseRemoveEntry)
	}
}

func TestActionsRegistry(t *testing.T) {
	// Test that all defined actions are in the registry
	expectedActions := []Action{ActionAdd, ActionRemove, ActionOutput}
	
	for _, action := range expectedActions {
		if !ActionsRegistry[action] {
			t.Errorf("Action %s should be registered in ActionsRegistry", action)
		}
	}

	// Test that the registry contains exactly the expected number of actions
	if len(ActionsRegistry) != len(expectedActions) {
		t.Errorf("ActionsRegistry should contain exactly %d actions, got: %d", len(expectedActions), len(ActionsRegistry))
	}

	// Test that only valid actions are in the registry
	for action, valid := range ActionsRegistry {
		if !valid {
			t.Errorf("Action %s should have value true in ActionsRegistry", action)
		}
		
		// Check that it's one of our expected actions
		found := false
		for _, expectedAction := range expectedActions {
			if action == expectedAction {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Unexpected action %s found in ActionsRegistry", action)
		}
	}
}

func TestIgnoreIPOptions(t *testing.T) {
	// Test IgnoreIPv4 function
	if IgnoreIPv4() != IPv4 {
		t.Errorf("IgnoreIPv4() should return IPv4, got: %s", IgnoreIPv4())
	}

	// Test IgnoreIPv6 function
	if IgnoreIPv6() != IPv6 {
		t.Errorf("IgnoreIPv6() should return IPv6, got: %s", IgnoreIPv6())
	}
}

func TestActionString(t *testing.T) {
	// Test Action type string conversion
	action := ActionAdd
	if string(action) != "add" {
		t.Errorf("Action string conversion failed. Expected: add, Got: %s", string(action))
	}
}

func TestIPTypeString(t *testing.T) {
	// Test IPType string conversion
	ipType := IPv4
	if string(ipType) != "ipv4" {
		t.Errorf("IPType string conversion failed. Expected: ipv4, Got: %s", string(ipType))
	}
}

func TestCaseRemoveInt(t *testing.T) {
	// Test CaseRemove int conversion
	caseRemove := CaseRemovePrefix
	if int(caseRemove) != 0 {
		t.Errorf("CaseRemove int conversion failed. Expected: 0, Got: %d", int(caseRemove))
	}
}

// Test interface definitions by creating mock implementations
type mockTyper struct {
	typeValue string
}

func (m *mockTyper) GetType() string {
	return m.typeValue
}

type mockActioner struct {
	actionValue Action
}

func (m *mockActioner) GetAction() Action {
	return m.actionValue
}

type mockDescriptioner struct {
	descValue string
}

func (m *mockDescriptioner) GetDescription() string {
	return m.descValue
}

type mockInputConverter struct {
	*mockTyper
	*mockActioner
	*mockDescriptioner
}

func (m *mockInputConverter) Input(container Container) (Container, error) {
	return container, nil
}

type mockOutputConverter struct {
	*mockTyper
	*mockActioner
	*mockDescriptioner
}

func (m *mockOutputConverter) Output(container Container) error {
	return nil
}

func TestInterfaces(t *testing.T) {
	// Test Typer interface
	typer := &mockTyper{typeValue: "test"}
	if typer.GetType() != "test" {
		t.Errorf("Typer interface implementation failed")
	}

	// Test Actioner interface
	actioner := &mockActioner{actionValue: ActionAdd}
	if actioner.GetAction() != ActionAdd {
		t.Errorf("Actioner interface implementation failed")
	}

	// Test Descriptioner interface
	descriptioner := &mockDescriptioner{descValue: "test description"}
	if descriptioner.GetDescription() != "test description" {
		t.Errorf("Descriptioner interface implementation failed")
	}

	// Test InputConverter interface
	inputConverter := &mockInputConverter{
		mockTyper:        &mockTyper{typeValue: "input"},
		mockActioner:     &mockActioner{actionValue: ActionAdd},
		mockDescriptioner: &mockDescriptioner{descValue: "input desc"},
	}

	if inputConverter.GetType() != "input" {
		t.Errorf("InputConverter GetType failed")
	}
	if inputConverter.GetAction() != ActionAdd {
		t.Errorf("InputConverter GetAction failed")
	}
	if inputConverter.GetDescription() != "input desc" {
		t.Errorf("InputConverter GetDescription failed")
	}

	container := NewContainer()
	result, err := inputConverter.Input(container)
	if err != nil {
		t.Errorf("InputConverter Input failed: %v", err)
	}
	if result != container {
		t.Errorf("InputConverter Input should return the same container")
	}

	// Test OutputConverter interface
	outputConverter := &mockOutputConverter{
		mockTyper:        &mockTyper{typeValue: "output"},
		mockActioner:     &mockActioner{actionValue: ActionOutput},
		mockDescriptioner: &mockDescriptioner{descValue: "output desc"},
	}

	if outputConverter.GetType() != "output" {
		t.Errorf("OutputConverter GetType failed")
	}
	if outputConverter.GetAction() != ActionOutput {
		t.Errorf("OutputConverter GetAction failed")
	}
	if outputConverter.GetDescription() != "output desc" {
		t.Errorf("OutputConverter GetDescription failed")
	}

	err = outputConverter.Output(container)
	if err != nil {
		t.Errorf("OutputConverter Output failed: %v", err)
	}
}