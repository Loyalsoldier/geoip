package special

import (
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestRegisteredConfigCreators(t *testing.T) {
	const config = `{
		"input": [
			{"type":"private","action":"add"},
			{"type":"test","action":"add"},
			{"type":"stdin","action":"add","args":{"name":"custom"}},
			{"type":"cutter","action":"remove","args":{"wantedList":["custom"],"onlyIPType":"ipv4"}}
		],
		"output": [
			{"type":"lookup","args":{"search":"10.0.0.1"}},
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
	const want = "private\n2001:db8::1/128\n127.0.0.0/8\n"
	if out != want {
		t.Fatalf("output = %q, want %q", out, want)
	}
}
