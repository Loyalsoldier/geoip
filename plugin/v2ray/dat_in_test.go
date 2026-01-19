package v2ray

import (
	"encoding/json"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestGeoIPDatIn_NewGeoIPDatIn(t *testing.T) {
	tests := []struct {
		name         string
		action       lib.Action
		data         json.RawMessage
		expectType   string
		expectIPType lib.IPType
		expectErr    bool
	}{
		{
			name:         "Valid action with URI",
			action:       lib.ActionAdd,
			data:         json.RawMessage(`{"uri": "https://example.com/geoip.dat"}`),
			expectType:   TypeGeoIPDatIn,
			expectIPType: "",
			expectErr:    false,
		},
		{
			name:         "Valid action with wanted list",
			action:       lib.ActionAdd,
			data:         json.RawMessage(`{"uri": "test.dat", "wantedList": ["CN", "US"]}`),
			expectType:   TypeGeoIPDatIn,
			expectIPType: "",
			expectErr:    false,
		},
		{
			name:         "Valid action with IPv4 only",
			action:       lib.ActionAdd,
			data:         json.RawMessage(`{"uri": "test.dat", "onlyIPType": "ipv4"}`),
			expectType:   TypeGeoIPDatIn,
			expectIPType: lib.IPv4,
			expectErr:    false,
		},
		{
			name:      "Missing URI",
			action:    lib.ActionAdd,
			data:      json.RawMessage(`{}`),
			expectErr: true,
		},
		{
			name:      "Empty URI",
			action:    lib.ActionAdd,
			data:      json.RawMessage(`{"uri": ""}`),
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
			converter, err := newGeoIPDatIn(tt.action, tt.data)
			if (err != nil) != tt.expectErr {
				t.Errorf("newGeoIPDatIn() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr {
				datIn := converter.(*GeoIPDatIn)
				if datIn.GetType() != tt.expectType {
					t.Errorf("GetType() = %v, expect %v", datIn.GetType(), tt.expectType)
				}
				if datIn.GetAction() != tt.action {
					t.Errorf("GetAction() = %v, expect %v", datIn.GetAction(), tt.action)
				}
				if datIn.OnlyIPType != tt.expectIPType {
					t.Errorf("OnlyIPType = %v, expect %v", datIn.OnlyIPType, tt.expectIPType)
				}
			}
		})
	}
}

func TestGeoIPDatIn_GetType(t *testing.T) {
	datIn := &GeoIPDatIn{Type: TypeGeoIPDatIn}
	result := datIn.GetType()
	if result != TypeGeoIPDatIn {
		t.Errorf("GetType() = %v, expect %v", result, TypeGeoIPDatIn)
	}
}

func TestGeoIPDatIn_GetAction(t *testing.T) {
	action := lib.ActionAdd
	datIn := &GeoIPDatIn{Action: action}
	result := datIn.GetAction()
	if result != action {
		t.Errorf("GetAction() = %v, expect %v", result, action)
	}
}

func TestGeoIPDatIn_GetDescription(t *testing.T) {
	datIn := &GeoIPDatIn{Description: DescGeoIPDatIn}
	result := datIn.GetDescription()
	if result != DescGeoIPDatIn {
		t.Errorf("GetDescription() = %v, expect %v", result, DescGeoIPDatIn)
	}
}

func TestGeoIPDatIn_Input(t *testing.T) {
	tests := []struct {
		name      string
		datIn     *GeoIPDatIn
		expectErr bool
	}{
		{
			name: "Non-existent file",
			datIn: &GeoIPDatIn{
				Type:   TypeGeoIPDatIn,
				Action: lib.ActionAdd,
				URI:    "/nonexistent/file.dat",
			},
			expectErr: true,
		},
		{
			name: "Invalid action",
			datIn: &GeoIPDatIn{
				Type:   TypeGeoIPDatIn,
				Action: lib.Action("invalid"),
				URI:    "test.dat",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := lib.NewContainer()
			_, err := tt.datIn.Input(container)

			if (err != nil) != tt.expectErr {
				t.Errorf("Input() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestGeoIPDatIn_Constants(t *testing.T) {
	if TypeGeoIPDatIn != "v2rayGeoIPDat" {
		t.Errorf("TypeGeoIPDatIn = %v, expect %v", TypeGeoIPDatIn, "v2rayGeoIPDat")
	}
	if DescGeoIPDatIn != "Convert V2Ray GeoIP dat to other formats" {
		t.Errorf("DescGeoIPDatIn = %v, expect correct description", DescGeoIPDatIn)
	}
}