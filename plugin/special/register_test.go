package special

import (
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestRegisteredConfigCreators(t *testing.T) {
	const config = `{
		"input": [
			{"type":"private","action":"add","args":{"onlyIPType":"ipv4"}},
			{"type":"test","action":"add"},
			{"type":"stdin","action":"add","args":{"name":"custom"}},
			{"type":"cutter","action":"remove","args":{"wantedList":["custom"],"onlyIPType":"ipv4"}}
		],
		"output": [
			{"type":"lookup","args":{"search":"10.0.0.1"}},
			{"type":"lookup","args":{"search":"fc00::1"}},
			{"type":"stdout","args":{"wantedList":["custom","test"]}}
		]
	}`
	instance, err := lib.NewInstance()
	if err != nil {
		t.Fatal(err)
	}
	if err := instance.InitConfigFromBytes([]byte(config)); err != nil {
		t.Fatal(err)
	}
	withStdin(t, "192.0.2.1\n2001:db8::1\n")
	out := captureStdout(t, func() { err = instance.Run() })
	if err != nil {
		t.Fatal(err)
	}
	const want = "private\nfalse\n2001:db8::1/128\n127.0.0.0/8\n"
	if out != want {
		t.Fatalf("output = %q, want %q", out, want)
	}
}

func TestRegisteredConverterListings(t *testing.T) {
	tests := []struct {
		name    string
		list    func()
		formats map[string]string
	}{
		{"input", lib.ListInputConverter, map[string]string{
			"cutter": DescCutter, "private": DescPrivate, "stdin": DescStdin, "test": descTest,
		}},
		{"output", lib.ListOutputConverter, map[string]string{
			"lookup": DescLookup, "stdout": DescStdout,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := captureStdout(t, tt.list)
			for name, desc := range tt.formats {
				if !strings.Contains(out, "  - "+name+" ("+desc+")\n") {
					t.Errorf("missing %s format %q (%s) in listing:\n%s", tt.name, name, desc, out)
				}
			}
		})
	}
}
