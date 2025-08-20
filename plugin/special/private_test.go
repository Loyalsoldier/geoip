package special

import (
	"encoding/json"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestPrivateConstants(t *testing.T) {
	if entryNamePrivate != "private" {
		t.Errorf("entryNamePrivate should be 'private', got: %s", entryNamePrivate)
	}
	if TypePrivate != "private" {
		t.Errorf("TypePrivate should be 'private', got: %s", TypePrivate)
	}
	if DescPrivate != "Convert LAN and private network CIDR to other formats" {
		t.Errorf("DescPrivate should be correct description, got: %s", DescPrivate)
	}
}

func TestPrivateCIDRs(t *testing.T) {
	if len(privateCIDRs) == 0 {
		t.Error("privateCIDRs should not be empty")
	}

	// Check for some expected private CIDRs
	expectedCIDRs := []string{
		"10.0.0.0/8",
		"192.168.0.0/16",
		"172.16.0.0/12",
		"127.0.0.0/8",
		"fc00::/7",
		"::1/128",
	}

	for _, expected := range expectedCIDRs {
		found := false
		for _, cidr := range privateCIDRs {
			if cidr == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("privateCIDRs should contain %s", expected)
		}
	}
}

func TestNewPrivate(t *testing.T) {
	tests := []struct {
		name         string
		action       lib.Action
		data         string
		expectError  bool
		expectIPType lib.IPType
	}{
		{
			name:         "Valid empty config",
			action:       lib.ActionAdd,
			data:         `{}`,
			expectError:  false,
			expectIPType: "",
		},
		{
			name:         "Valid config with IPv4",
			action:       lib.ActionAdd,
			data:         `{"onlyIPType": "ipv4"}`,
			expectError:  false,
			expectIPType: lib.IPv4,
		},
		{
			name:         "Valid config with IPv6",
			action:       lib.ActionRemove,
			data:         `{"onlyIPType": "ipv6"}`,
			expectError:  false,
			expectIPType: lib.IPv6,
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
			expectError:  false,
			expectIPType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter, err := newPrivate(tt.action, json.RawMessage(tt.data))

			if tt.expectError && err == nil {
				t.Errorf("newPrivate() should return error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("newPrivate() should not return error but got: %v", err)
			}

			if !tt.expectError {
				if converter == nil {
					t.Error("newPrivate() should return non-nil converter")
				} else {
					private := converter.(*Private)
					if private.GetType() != TypePrivate {
						t.Errorf("GetType() = %s; want %s", private.GetType(), TypePrivate)
					}
					if private.GetAction() != tt.action {
						t.Errorf("GetAction() = %s; want %s", private.GetAction(), tt.action)
					}
					if private.GetDescription() != DescPrivate {
						t.Errorf("GetDescription() = %s; want %s", private.GetDescription(), DescPrivate)
					}
					if private.OnlyIPType != tt.expectIPType {
						t.Errorf("OnlyIPType = %s; want %s", private.OnlyIPType, tt.expectIPType)
					}
				}
			}
		})
	}
}

func TestPrivateInput_Add(t *testing.T) {
	converter, err := newPrivate(lib.ActionAdd, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("newPrivate() failed: %v", err)
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

	// Check that the private entry exists
	entry, found := result.GetEntry(entryNamePrivate)
	if !found {
		t.Errorf("Container should contain entry with name %s", entryNamePrivate)
	}
	if entry.GetName() != "PRIVATE" {
		t.Errorf("Entry name should be PRIVATE, got %s", entry.GetName())
	}

	// Verify the entry contains some private CIDRs
	cidrs, err := entry.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() failed: %v", err)
	}

	if len(cidrs) == 0 {
		t.Error("Entry should contain private CIDRs")
	}

	// Check that some expected private CIDRs are included
	expectedCIDRs := []string{"10.0.0.0/8", "192.168.0.0/16"}
	for _, expected := range expectedCIDRs {
		found := false
		for _, cidr := range cidrs {
			if cidr == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Entry should contain private CIDR %s", expected)
		}
	}
}

func TestPrivateInput_IPv4Only(t *testing.T) {
	converter, err := newPrivate(lib.ActionAdd, json.RawMessage(`{"onlyIPType": "ipv4"}`))
	if err != nil {
		t.Fatalf("newPrivate() failed: %v", err)
	}

	container := lib.NewContainer()
	result, err := converter.Input(container)
	if err != nil {
		t.Errorf("Input() should not return error: %v", err)
	}

	// Verify entry was added
	entry, found := result.GetEntry(entryNamePrivate)
	if !found {
		t.Error("Container should contain the private entry")
	}

	// Check that only IPv4 CIDRs are included
	cidrs, err := entry.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() failed: %v", err)
	}

	// Should contain IPv4 private ranges
	hasIPv4 := false
	for _, cidr := range cidrs {
		if cidr == "10.0.0.0/8" || cidr == "192.168.0.0/16" {
			hasIPv4 = true
			break
		}
	}
	if !hasIPv4 {
		t.Error("Entry should contain IPv4 private CIDRs")
	}
}

func TestPrivateInput_UnknownAction(t *testing.T) {
	private := &Private{
		Type:        TypePrivate,
		Action:      lib.Action("unknown"),
		Description: DescPrivate,
	}

	container := lib.NewContainer()
	_, err := private.Input(container)
	if err != lib.ErrUnknownAction {
		t.Errorf("Input() with unknown action should return ErrUnknownAction, got: %v", err)
	}
}