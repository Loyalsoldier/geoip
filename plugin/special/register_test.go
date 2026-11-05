package special

import (
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestRegisteredConfigCreators(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{"cutter", `{"input":[{"type":"cutter","action":"remove","args":{"wantedList":["cn"]}}]}`},
		{"private", `{"input":[{"type":"private","action":"add"}]}`},
		{"stdin", `{"input":[{"type":"stdin","action":"add","args":{"name":"cn"}}]}`},
		{"test", `{"input":[{"type":"test","action":"add"}]}`},
		{"lookup", `{"output":[{"type":"lookup","args":{"search":"1.1.1.1"}}]}`},
		{"stdout", `{"output":[{"type":"stdout"}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance, err := lib.NewInstance()
			if err != nil {
				t.Fatal(err)
			}
			if err := instance.InitConfigFromBytes([]byte(tt.config)); err != nil {
				t.Fatalf("InitConfigFromBytes failed: %v", err)
			}
		})
	}
}
