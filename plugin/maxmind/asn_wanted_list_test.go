package maxmind

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestASNWantedListOptions(t *testing.T) {
	single := map[string][]string{"64496": {"AS64496"}}
	multiple := map[string][]string{"64496": {"AS64496"}, "64497": {"AS64497"}}
	custom := map[string][]string{"64496": {"A", "B"}, "64497": {"B"}}
	tests := []struct {
		name string
		opt  lib.InputOption
		want map[string][]string
	}{
		{"number", WithGeoLite2ASNCSVWantedList(uint(64496)), single},
		{"prefixed string", WithGeoLite2ASNCSVWantedList(" aS64496 "), single},
		{"number array", WithGeoLite2ASNCSVWantedList([]uint{64497, 64496, 64497}), multiple},
		{"string array", WithGeoLite2ASNCSVWantedList([]string{"as64497", " 64496 ", "AS64496", " AS "}), multiple},
		{"named numbers", WithGeoLite2ASNCSVWantedList(map[string][]uint{
			" b ": {64496, 64497, 64497}, "a": {64496}, " ": {1},
		}), custom},
		{"named strings", WithGeoLite2ASNCSVWantedList(map[string][]string{
			" b ": {"AS64497", "64496"}, "a": {"as64496", "64496"}, " ": {"1"},
		}), custom},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in, err := NewGeoLite2ASNCSVIn(lib.ActionAdd, tt.opt)
			if err != nil {
				t.Fatal(err)
			}
			if got := in.(*geolite2_asn_csv_in).Want; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("wanted lists = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestASNWantedListJSON(t *testing.T) {
	tests := []struct {
		name, data string
		want       map[string][]string
	}{
		{"empty", " ", map[string][]string{}},
		{"numbers", "[64496,64497]", map[string][]string{"64496": {"AS64496"}, "64497": {"AS64497"}}},
		{"strings", `[" As64496 ","64496"]`, map[string][]string{"64496": {"AS64496"}}},
		{"named numbers", `{" example ":[64496]}`, map[string][]string{"64496": {"EXAMPLE"}}},
		{"named strings", `{"example":["as64496"]}`, map[string][]string{"64496": {"EXAMPLE"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Unmarshalling into an existing option must replace, not append to, the old selection.
			list := GeoLite2ASNCSVWantedList{TypeSliceNum: []uint{1}, TypeSliceStr: []string{"2"},
				TypeMapNum: map[string][]uint{"old": {3}}, TypeMapStr: map[string][]string{"old": {"4"}}}
			if err := list.UnmarshalJSON([]byte(tt.data)); err != nil {
				t.Fatal(err)
			}
			g := &geolite2_asn_csv_in{}
			WithGeoLite2ASNCSVWantedList(list)(g)
			if !reflect.DeepEqual(g.Want, tt.want) {
				t.Errorf("wanted lists = %v, want %v", g.Want, tt.want)
			}
		})
	}
	for _, invalid := range []string{`42`, `["AS64496",64497]`, `{"example":64496}`} {
		t.Run("invalid "+invalid, func(t *testing.T) {
			_, err := NewGeoLite2ASNCSVInFromBytes(lib.ActionAdd, []byte(`{"wantedList":`+invalid+`}`))
			if err == nil || !strings.Contains(err.Error(), "invalid JSON config for `wantedList`") {
				t.Errorf("got %v, want invalid wantedList error", err)
			}
		})
	}
}
