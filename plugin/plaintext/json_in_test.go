package plaintext

import (
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestJSONInConstructors(t *testing.T) {
	t.Parallel()

	testInputConstructors(t, TypeJSONIn, DescJSONIn,
		NewJSONIn, NewJSONInFromBytes,
		[]lib.InputOption{WithNameAndURI("cn", "./cn.json"), WithJSONPath([]string{"prefixes.#.ipv4Prefix"})},
		`{"name":"cn","uri":"./cn.json","jsonPath":["prefixes.#.ipv4Prefix"]}`,
	)
}

func TestJSONIn_ConstructorValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    string
		wantErr string
	}{
		{
			name:    "missing jsonPath",
			args:    `{"name":"cn","uri":"./cn.json"}`,
			wantErr: "❌ [type json | action add] missing jsonPath",
		},
		{
			name:    "empty jsonPath",
			args:    `{"name":"cn","uri":"./cn.json","jsonPath":[]}`,
			wantErr: "❌ [type json | action add] missing jsonPath",
		},
		{
			name:    "ipOrCIDR is not supported",
			args:    `{"name":"cn","ipOrCIDR":["1.1.1.1"],"jsonPath":["ip"]}`,
			wantErr: "❌ [type json | action add] ipOrCIDR is invalid for this input format",
		},
		{
			name:    "missing uri",
			args:    `{"name":"cn","jsonPath":["ip"]}`,
			wantErr: "❌ [type json | action add] missing uri or ipOrCIDR",
		},
		{
			name:    "inputDir with name",
			args:    `{"inputDir":"./json","name":"cn","jsonPath":["ip"]}`,
			wantErr: "❌ [type json | action add] inputDir is not allowed to be used with name or uri or ipOrCIDR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewJSONInFromBytes(lib.ActionAdd, []byte(tt.args))
			assertErrContains(t, err, tt.wantErr)
			if conv != nil {
				t.Errorf("expected nil converter, got %#v", conv)
			}
		})
	}
}

func TestJSONIn_Input(t *testing.T) {
	t.Parallel()

	const data = `{
  "syncToken": "1700000000",
  "prefixes": [
    {"ipv4Prefix": "192.168.0.3/24", "service": "A"},
    {"ipv4Prefix": " 192.168.1.5/24 ", "service": "B"},
    {"ipv6Prefix": "2001:db8::/33", "service": "A"},
    {"ipv6Prefix": "2001:db8:8000::/33", "service": "B"},
    {"ipv6Prefix": "::ffff:1.2.3.0/120", "service": "C"}
  ],
  "nested": {"list": [["10.0.0.1", ["::ffff:5.6.7.8"]], "2001:DB8::ABCD"]},
  "single": "8.8.8.8",
  "empty": "  ",
  "number": 42,
  "object": {"ip": "1.1.1.1"},
  "badMapped": "::ffff:1.2.3.0/95",
  "badIP": "not-an-ip",
  "arrayWithNull": ["9.9.9.9", null]
}`

	tests := []struct {
		name      string
		content   string
		args      map[string]any
		action    lib.Action
		initial   map[string][]string
		want      map[string][]string
		wantErr   string
		wantErrIs error
	}{
		{
			name:    "add with multiple paths",
			content: data,
			args:    map[string]any{"jsonPath": []string{"prefixes.#.ipv4Prefix", " prefixes.#.ipv6Prefix "}},
			action:  lib.ActionAdd,
			initial: map[string][]string{"US": {"8.8.8.8"}},
			want: map[string][]string{
				"CN": {"1.2.3.0/24", "192.168.0.0/23", "2001:db8::/32"},
				"US": {"8.8.8.8/32"},
			},
		},
		{
			name:    "add with query and nested arrays",
			content: data,
			args:    map[string]any{"jsonPath": []string{`prefixes.#(service=="B")#.ipv4Prefix`, "nested.list", "single"}},
			action:  lib.ActionAdd,
			want:    map[string][]string{"CN": {"5.6.7.8/32", "8.8.8.8/32", "10.0.0.1/32", "192.168.1.0/24", "2001:db8::abcd/128"}},
		},
		{
			name:    "add only IPv6",
			content: data,
			args:    map[string]any{"jsonPath": []string{"prefixes.#.ipv4Prefix", "prefixes.#.ipv6Prefix"}, "onlyIPType": "ipv6"},
			action:  lib.ActionAdd,
			want:    map[string][]string{"CN": {"2001:db8::/32"}},
		},
		{
			name:    "remove",
			content: data,
			args:    map[string]any{"jsonPath": []string{"prefixes.#.ipv4Prefix"}},
			action:  lib.ActionRemove,
			initial: map[string][]string{"CN": {"192.168.0.0/16", "2001:db8::/32"}},
			want: map[string][]string{"CN": {
				"192.168.2.0/23", "192.168.4.0/22", "192.168.8.0/21", "192.168.16.0/20",
				"192.168.32.0/19", "192.168.64.0/18", "192.168.128.0/17", "2001:db8::/32",
			}},
		},
		{
			name:    "remove only IPv4",
			content: data,
			args:    map[string]any{"jsonPath": []string{"prefixes.#.ipv6Prefix", "single"}, "onlyIPType": "ipv4"},
			action:  lib.ActionRemove,
			initial: map[string][]string{"CN": {"8.8.8.8/30", "1.2.3.0/23", "2001:db8::/32"}},
			want:    map[string][]string{"CN": {"1.2.2.0/24", "8.8.8.9/32", "8.8.8.10/31", "2001:db8::/32"}},
		},
		{
			name:    "wantedList not containing the list",
			content: data,
			args:    map[string]any{"jsonPath": []string{"single"}, "wantedList": []string{"us"}},
			action:  lib.ActionAdd,
			wantErr: "no entry is generated",
		},
		{
			name:    "invalid JSON",
			content: `{"single": "8.8.8.8"`,
			args:    map[string]any{"jsonPath": []string{"single"}},
			action:  lib.ActionAdd,
			wantErr: "invalid JSON data",
		},
		{
			name:    "missing path",
			content: data,
			args:    map[string]any{"jsonPath": []string{"single", "missing.path"}},
			action:  lib.ActionAdd,
			wantErr: "failed to process JSON: invalid IP address or CIDR (value not exist)",
		},
		{
			name:    "empty string",
			content: data,
			args:    map[string]any{"jsonPath": []string{"empty"}},
			action:  lib.ActionAdd,
			wantErr: "failed to process JSON: empty string",
		},
		{
			name:    "number value",
			content: data,
			args:    map[string]any{"jsonPath": []string{"number"}},
			action:  lib.ActionAdd,
			wantErr: "failed to process JSON: invalid IP address or CIDR, please check",
		},
		{
			name:    "object value",
			content: data,
			args:    map[string]any{"jsonPath": []string{"object"}},
			action:  lib.ActionAdd,
			wantErr: "failed to process JSON: invalid IP address or CIDR, please check",
		},
		{
			name:    "null in array",
			content: data,
			args:    map[string]any{"jsonPath": []string{"arrayWithNull"}},
			action:  lib.ActionAdd,
			wantErr: "failed to process JSON: invalid IP address or CIDR, please check",
		},
		{
			name:    "malformed IPv4-mapped CIDR",
			content: data,
			args:    map[string]any{"jsonPath": []string{"badMapped"}},
			action:  lib.ActionAdd,
			wantErr: "failed to process JSON: " + lib.ErrInvalidCIDR.Error(),
		},
		{
			name:    "invalid IP",
			content: data,
			args:    map[string]any{"jsonPath": []string{"badIP"}},
			action:  lib.ActionAdd,
			wantErr: "failed to process JSON: " + lib.ErrInvalidIP.Error(),
		},
		{
			name:      "unknown action",
			content:   data,
			args:      map[string]any{"jsonPath": []string{"single"}},
			action:    lib.Action("unknown"),
			wantErrIs: lib.ErrUnknownAction,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for _, source := range allInputSources {
				t.Run(string(source), func(t *testing.T) {
					t.Parallel()

					args := withInputSource(t, source, ".json", tt.content, tt.args)
					got, err := runInput(t, TypeJSONIn, tt.action, args, tt.initial)
					assertInputResult(t, TypeJSONIn, tt.action, got, err, tt.want, tt.wantErr, tt.wantErrIs)
				})
			}
		})
	}
}

func TestJSONIn_Config(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, dir, "cn.json", `{"ips":["1.1.1.0/24","2001:db8::/32"]}`)
	writeTestFile(t, dir, "us.json", `{"ips":["8.8.8.8"]}`)

	instance, err := newInstanceFromConfig(t, map[string]any{
		"input": []map[string]any{
			{"type": "json", "action": "add", "args": map[string]any{"inputDir": dir, "jsonPath": []string{"ips"}, "wantedList": []string{"cn"}}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	container := lib.NewContainer()
	if err := instance.RunInput(container); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContainer(t, container, map[string][]string{"CN": {"1.1.1.0/24", "2001:db8::/32"}})
}

func TestJSONIn_ConfigInvalid(t *testing.T) {
	t.Parallel()

	_, err := newInstanceFromConfig(t, map[string]any{
		"input": []map[string]any{
			{"type": "json", "action": "add", "args": map[string]any{"name": "cn", "uri": "./cn.json"}},
		},
	})
	assertErrContains(t, err, "❌ [type json | action add] missing jsonPath")
}
