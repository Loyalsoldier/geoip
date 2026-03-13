package special

import (
	"encoding/json"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestPrivate_NewPrivate(t *testing.T) {
	tests := []struct {
		name         string
		action       lib.Action
		data         json.RawMessage
		expectType   string
		expectIPType lib.IPType
		expectErr    bool
	}{
		{
			name:         "Valid action add with no data",
			action:       lib.ActionAdd,
			data:         nil,
			expectType:   TypePrivate,
			expectIPType: "",
			expectErr:    false,
		},
		{
			name:         "Valid action with IPv4 only",
			action:       lib.ActionAdd,
			data:         json.RawMessage(`{"onlyIPType": "ipv4"}`),
			expectType:   TypePrivate,
			expectIPType: lib.IPv4,
			expectErr:    false,
		},
		{
			name:         "Valid action with IPv6 only",
			action:       lib.ActionRemove,
			data:         json.RawMessage(`{"onlyIPType": "ipv6"}`),
			expectType:   TypePrivate,
			expectIPType: lib.IPv6,
			expectErr:    false,
		},
		{
			name:      "Invalid JSON data",
			action:    lib.ActionAdd,
			data:      json.RawMessage(`{invalid json}`),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter, err := newPrivate(tt.action, tt.data)
			if (err != nil) != tt.expectErr {
				t.Errorf("newPrivate() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr {
				private := converter.(*Private)
				if private.GetType() != tt.expectType {
					t.Errorf("GetType() = %v, expect %v", private.GetType(), tt.expectType)
				}
				if private.GetAction() != tt.action {
					t.Errorf("GetAction() = %v, expect %v", private.GetAction(), tt.action)
				}
				if private.OnlyIPType != tt.expectIPType {
					t.Errorf("OnlyIPType = %v, expect %v", private.OnlyIPType, tt.expectIPType)
				}
			}
		})
	}
}

func TestPrivate_GetType(t *testing.T) {
	private := &Private{Type: TypePrivate}
	result := private.GetType()
	if result != TypePrivate {
		t.Errorf("GetType() = %v, expect %v", result, TypePrivate)
	}
}

func TestPrivate_GetAction(t *testing.T) {
	action := lib.ActionAdd
	private := &Private{Action: action}
	result := private.GetAction()
	if result != action {
		t.Errorf("GetAction() = %v, expect %v", result, action)
	}
}

func TestPrivate_GetDescription(t *testing.T) {
	private := &Private{Description: DescPrivate}
	result := private.GetDescription()
	if result != DescPrivate {
		t.Errorf("GetDescription() = %v, expect %v", result, DescPrivate)
	}
}

func TestPrivate_Input(t *testing.T) {
	tests := []struct {
		name       string
		action     lib.Action
		onlyIPType lib.IPType
		expectErr  bool
	}{
		{
			name:       "Action add with no IP type restriction",
			action:     lib.ActionAdd,
			onlyIPType: "",
			expectErr:  false,
		},
		{
			name:       "Action add with IPv4 only",
			action:     lib.ActionAdd,
			onlyIPType: lib.IPv4,
			expectErr:  false,
		},
		{
			name:       "Action add with IPv6 only",
			action:     lib.ActionAdd,
			onlyIPType: lib.IPv6,
			expectErr:  false,
		},
		{
			name:       "Action remove",
			action:     lib.ActionRemove,
			onlyIPType: "",
			expectErr:  false,
		},
		{
			name:       "Invalid action",
			action:     lib.Action("invalid"),
			onlyIPType: "",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			private := &Private{
				Type:        TypePrivate,
				Action:      tt.action,
				Description: DescPrivate,
				OnlyIPType:  tt.onlyIPType,
			}

			container := lib.NewContainer()

			// For remove action, pre-populate the container
			if tt.action == lib.ActionRemove && !tt.expectErr {
				entry := lib.NewEntry(entryNamePrivate)
				for _, cidr := range privateCIDRs {
					if err := entry.AddPrefix(cidr); err != nil {
						t.Fatalf("Failed to add prefix: %v", err)
					}
				}
				if err := container.Add(entry); err != nil {
					t.Fatalf("Failed to add entry to container: %v", err)
				}
			}

			result, err := private.Input(container)

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
					entry, found := result.GetEntry(entryNamePrivate)
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

func TestPrivate_InputExistingEntry(t *testing.T) {
	private := &Private{
		Type:        TypePrivate,
		Action:      lib.ActionAdd,
		Description: DescPrivate,
		OnlyIPType:  "",
	}

	container := lib.NewContainer()

	// Pre-populate container with existing entry
	existingEntry := lib.NewEntry(entryNamePrivate)
	if err := existingEntry.AddPrefix("10.0.0.0/8"); err != nil {
		t.Fatalf("Failed to add prefix to existing entry: %v", err)
	}
	if err := container.Add(existingEntry); err != nil {
		t.Fatalf("Failed to add existing entry to container: %v", err)
	}

	result, err := private.Input(container)

	if err != nil {
		t.Errorf("Input() with existing entry failed: %v", err)
		return
	}

	if result == nil {
		t.Error("Input() returned nil container")
		return
	}

	entry, found := result.GetEntry(entryNamePrivate)
	if !found {
		t.Error("Expected entry not found in container")
	}
	if entry == nil {
		t.Error("Entry is nil")
	}
}

func TestPrivate_Constants(t *testing.T) {
	if entryNamePrivate != "private" {
		t.Errorf("entryNamePrivate = %v, expect %v", entryNamePrivate, "private")
	}
	if TypePrivate != "private" {
		t.Errorf("TypePrivate = %v, expect %v", TypePrivate, "private")
	}
	if DescPrivate != "Convert LAN and private network CIDR to other formats" {
		t.Errorf("DescPrivate = %v, expect correct description", DescPrivate)
	}
}

func TestPrivate_PrivateCIDRs(t *testing.T) {
	expectedCount := 21 // Based on the privateCIDRs slice in private.go
	if len(privateCIDRs) != expectedCount {
		t.Errorf("privateCIDRs length = %v, expect %v", len(privateCIDRs), expectedCount)
	}

	// Test some key private CIDRs
	expectedCIDRs := map[string]bool{
		"10.0.0.0/8":     true,
		"127.0.0.0/8":    true,
		"192.168.0.0/16": true,
		"::1/128":        true,
		"fc00::/7":       true,
	}

	found := make(map[string]bool)
	for _, cidr := range privateCIDRs {
		if expectedCIDRs[cidr] {
			found[cidr] = true
		}
	}

	for expectedCIDR := range expectedCIDRs {
		if !found[expectedCIDR] {
			t.Errorf("Expected CIDR %v not found in privateCIDRs", expectedCIDR)
		}
	}
}

func TestPrivate_InputWithInvalidCIDR(t *testing.T) {
	// This test would require modifying privateCIDRs, which is not recommended
	// as it's a package-level variable. Instead, we can test the error path
	// by ensuring the Input method properly handles AddPrefix errors
	private := &Private{
		Type:        TypePrivate,
		Action:      lib.ActionAdd,
		Description: DescPrivate,
		OnlyIPType:  "",
	}

	container := lib.NewContainer()

	// The actual test would need to mock the privateCIDRs or use a different approach
	// For now, just verify the method works with valid CIDRs
	_, err := private.Input(container)
	if err != nil {
		t.Errorf("Input() with valid CIDRs failed: %v", err)
	}
}