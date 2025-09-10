package special

import (
	"encoding/json"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestCutter_NewCutter(t *testing.T) {
	tests := []struct {
		name         string
		action       lib.Action
		data         json.RawMessage
		expectType   string
		expectIPType lib.IPType
		expectWant   map[string]bool
		expectErr    bool
	}{
		{
			name:         "Valid remove action with wanted list",
			action:       lib.ActionRemove,
			data:         json.RawMessage(`{"wantedList": ["test1", "test2"], "onlyIPType": "ipv4"}`),
			expectType:   TypeCutter,
			expectIPType: lib.IPv4,
			expectWant:   map[string]bool{"TEST1": true, "TEST2": true},
			expectErr:    false,
		},
		{
			name:         "Valid remove action with IPv6",
			action:       lib.ActionRemove,
			data:         json.RawMessage(`{"wantedList": ["test"], "onlyIPType": "ipv6"}`),
			expectType:   TypeCutter,
			expectIPType: lib.IPv6,
			expectWant:   map[string]bool{"TEST": true},
			expectErr:    false,
		},
		{
			name:      "Invalid action",
			action:    lib.ActionAdd,
			data:      json.RawMessage(`{"wantedList": ["test"]}`),
			expectErr: true,
		},
		{
			name:      "Empty wanted list",
			action:    lib.ActionRemove,
			data:      json.RawMessage(`{"wantedList": []}`),
			expectErr: true,
		},
		{
			name:      "Missing wanted list",
			action:    lib.ActionRemove,
			data:      json.RawMessage(`{}`),
			expectErr: true,
		},
		{
			name:      "Invalid JSON",
			action:    lib.ActionRemove,
			data:      json.RawMessage(`{invalid json}`),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter, err := newCutter(tt.action, tt.data)
			if (err != nil) != tt.expectErr {
				t.Errorf("newCutter() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr {
				cutter := converter.(*Cutter)
				if cutter.GetType() != tt.expectType {
					t.Errorf("GetType() = %v, expect %v", cutter.GetType(), tt.expectType)
				}
				if cutter.GetAction() != tt.action {
					t.Errorf("GetAction() = %v, expect %v", cutter.GetAction(), tt.action)
				}
				if cutter.OnlyIPType != tt.expectIPType {
					t.Errorf("OnlyIPType = %v, expect %v", cutter.OnlyIPType, tt.expectIPType)
				}
				if len(cutter.Want) != len(tt.expectWant) {
					t.Errorf("Want length = %v, expect %v", len(cutter.Want), len(tt.expectWant))
				}
				for k, v := range tt.expectWant {
					if cutter.Want[k] != v {
						t.Errorf("Want[%s] = %v, expect %v", k, cutter.Want[k], v)
					}
				}
			}
		})
	}
}

func TestCutter_GetType(t *testing.T) {
	cutter := &Cutter{Type: TypeCutter}
	result := cutter.GetType()
	if result != TypeCutter {
		t.Errorf("GetType() = %v, expect %v", result, TypeCutter)
	}
}

func TestCutter_GetAction(t *testing.T) {
	action := lib.ActionRemove
	cutter := &Cutter{Action: action}
	result := cutter.GetAction()
	if result != action {
		t.Errorf("GetAction() = %v, expect %v", result, action)
	}
}

func TestCutter_GetDescription(t *testing.T) {
	cutter := &Cutter{Description: DescCutter}
	result := cutter.GetDescription()
	if result != DescCutter {
		t.Errorf("GetDescription() = %v, expect %v", result, DescCutter)
	}
}

func TestCutter_Input(t *testing.T) {
	tests := []struct {
		name       string
		want       map[string]bool
		onlyIPType lib.IPType
		expectErr  bool
	}{
		{
			name:       "Remove specific entries",
			want:       map[string]bool{"TEST1": true},
			onlyIPType: "",
			expectErr:  false,
		},
		{
			name:       "Remove with IPv4 only",
			want:       map[string]bool{"TEST1": true},
			onlyIPType: lib.IPv4,
			expectErr:  false,
		},
		{
			name:       "Remove with IPv6 only",
			want:       map[string]bool{"TEST1": true},
			onlyIPType: lib.IPv6,
			expectErr:  false,
		},
		{
			name:       "Remove all entries when want list matches all",
			want:       map[string]bool{"TEST1": true, "TEST2": true},
			onlyIPType: "",
			expectErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cutter := &Cutter{
				Type:        TypeCutter,
				Action:      lib.ActionRemove,
				Description: DescCutter,
				Want:        tt.want,
				OnlyIPType:  tt.onlyIPType,
			}

			// Create container with test entries
			container := lib.NewContainer()
			entry1 := lib.NewEntry("TEST1")
			entry2 := lib.NewEntry("TEST2")

			if err := entry1.AddPrefix("192.168.1.0/24"); err != nil {
				t.Fatalf("Failed to add prefix to entry1: %v", err)
			}
			if err := entry2.AddPrefix("192.168.2.0/24"); err != nil {
				t.Fatalf("Failed to add prefix to entry2: %v", err)
			}

			if err := container.Add(entry1); err != nil {
				t.Fatalf("Failed to add entry1: %v", err)
			}
			if err := container.Add(entry2); err != nil {
				t.Fatalf("Failed to add entry2: %v", err)
			}

			originalCount := container.Len()

			result, err := cutter.Input(container)

			if (err != nil) != tt.expectErr {
				t.Errorf("Input() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			if !tt.expectErr {
				if result == nil {
					t.Error("Input() returned nil container")
					return
				}

				// Check that appropriate entries were removed
				for wantedEntry := range tt.want {
					_, found := result.GetEntry(wantedEntry)
					if found && tt.onlyIPType == "" {
						// If onlyIPType is empty, the entire entry should be removed
						t.Errorf("Entry %s should have been removed but still exists", wantedEntry)
					}
				}

				// If specific IP types are targeted, the entry might still exist but be modified
				if tt.onlyIPType != "" {
					// The entries should still exist but have the specified IP type removed
					for wantedEntry := range tt.want {
						entry, found := result.GetEntry(wantedEntry)
						if !found {
							continue // Entry completely removed, which is also valid
						}
						if entry == nil {
							t.Errorf("Entry %s is nil", wantedEntry)
						}
					}
				}

				t.Logf("Original count: %d, Final count: %d", originalCount, result.Len())
			}
		})
	}
}

func TestCutter_InputEmptyContainer(t *testing.T) {
	cutter := &Cutter{
		Type:        TypeCutter,
		Action:      lib.ActionRemove,
		Description: DescCutter,
		Want:        map[string]bool{"TEST": true},
		OnlyIPType:  "",
	}

	container := lib.NewContainer()
	result, err := cutter.Input(container)

	if err != nil {
		t.Errorf("Input() with empty container failed: %v", err)
		return
	}

	if result == nil {
		t.Error("Input() returned nil container")
		return
	}

	if result.Len() != 0 {
		t.Errorf("Empty container should remain empty, got %d entries", result.Len())
	}
}

func TestCutter_InputNoMatchingEntries(t *testing.T) {
	cutter := &Cutter{
		Type:        TypeCutter,
		Action:      lib.ActionRemove,
		Description: DescCutter,
		Want:        map[string]bool{"NONEXISTENT": true},
		OnlyIPType:  "",
	}

	// Create container with test entries
	container := lib.NewContainer()
	entry := lib.NewEntry("TEST")
	if err := entry.AddPrefix("192.168.1.0/24"); err != nil {
		t.Fatalf("Failed to add prefix: %v", err)
	}
	if err := container.Add(entry); err != nil {
		t.Fatalf("Failed to add entry: %v", err)
	}

	originalCount := container.Len()

	result, err := cutter.Input(container)

	if err != nil {
		t.Errorf("Input() with no matching entries failed: %v", err)
		return
	}

	if result == nil {
		t.Error("Input() returned nil container")
		return
	}

	if result.Len() != originalCount {
		t.Errorf("Container length changed from %d to %d when no entries should be removed", originalCount, result.Len())
	}
}

func TestCutter_Constants(t *testing.T) {
	if TypeCutter != "cutter" {
		t.Errorf("TypeCutter = %v, expect %v", TypeCutter, "cutter")
	}
	if DescCutter != "Remove data from previous steps" {
		t.Errorf("DescCutter = %v, expect correct description", DescCutter)
	}
}