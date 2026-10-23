package plaintext

import (
	"path/filepath"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestClashRuleSetIPCIDRInConstructors(t *testing.T) {
	t.Parallel()

	testInputConstructors(t, TypeClashRuleSetIPCIDRIn, DescClashRuleSetIPCIDRIn,
		NewClashRuleSetIPCIDRIn, NewClashRuleSetIPCIDRInFromBytes,
		[]lib.InputOption{WithNameAndURI("cn", "./cn.yaml")},
		`{"name":"cn","uri":"./cn.yaml"}`,
	)
}

func TestClashRuleSetClassicalInConstructors(t *testing.T) {
	t.Parallel()

	testInputConstructors(t, TypeClashRuleSetClassicalIn, DescClashRuleSetClassicalIn,
		NewClashRuleSetClassicalIn, NewClashRuleSetClassicalInFromBytes,
		[]lib.InputOption{WithInputDir("./clash")},
		`{"inputDir":"./clash","wantedList":["cn"]}`,
	)
}

func TestClashRuleSetIn_RejectsIPOrCIDR(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		iType string
		newFn func(lib.Action, []byte) (lib.InputConverter, error)
	}{
		{name: "ipcidr", iType: TypeClashRuleSetIPCIDRIn, newFn: NewClashRuleSetIPCIDRInFromBytes},
		{name: "classical", iType: TypeClashRuleSetClassicalIn, newFn: NewClashRuleSetClassicalInFromBytes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := tt.newFn(lib.ActionAdd, []byte(`{"name":"cn","ipOrCIDR":["1.1.1.1"]}`))
			assertErrContains(t, err, "❌ [type "+tt.iType+" | action add] ipOrCIDR is invalid for this input format")
			if conv != nil {
				t.Errorf("expected nil converter, got %#v", conv)
			}
		})
	}
}

func TestClashRuleSetIPCIDRIn_Input(t *testing.T) {
	t.Parallel()

	const ruleSet = "# Clash ipcidr rule set\n" +
		"payload:\n" +
		"  - '192.168.0.3/24'\n" +
		"  - '192.168.1.5/24'\n" +
		"  - \"2001:db8::/33\"\n" +
		"  - 2001:db8:8000::/33\n" +
		"  - '::ffff:1.2.3.0/120'\n" +
		"  - '::ffff:5.6.7.8'\n" +
		"  - ''\n"

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
			name:    "add",
			content: ruleSet,
			action:  lib.ActionAdd,
			initial: map[string][]string{"US": {"8.8.8.8"}},
			want: map[string][]string{
				"CN": {"1.2.3.0/24", "5.6.7.8/32", "192.168.0.0/23", "2001:db8::/32"},
				"US": {"8.8.8.8/32"},
			},
		},
		{
			name:    "add only IPv6",
			content: ruleSet,
			args:    map[string]any{"onlyIPType": "ipv6"},
			action:  lib.ActionAdd,
			want:    map[string][]string{"CN": {"2001:db8::/32"}},
		},
		{
			name:    "remove only IPv4",
			content: "payload:\n  - '10.0.0.0/9'\n  - '2001:db8::/33'\n",
			args:    map[string]any{"onlyIPType": "ipv4"},
			action:  lib.ActionRemove,
			initial: map[string][]string{"CN": {"10.0.0.0/8", "2001:db8::/32"}},
			want:    map[string][]string{"CN": {"10.128.0.0/9", "2001:db8::/32"}},
		},
		{
			name:    "wantedList containing the list",
			content: ruleSet,
			args:    map[string]any{"wantedList": []string{"CN"}, "onlyIPType": "ipv4"},
			action:  lib.ActionAdd,
			want:    map[string][]string{"CN": {"1.2.3.0/24", "5.6.7.8/32", "192.168.0.0/23"}},
		},
		{
			name:    "wantedList not containing the list",
			content: ruleSet,
			args:    map[string]any{"wantedList": []string{"us"}},
			action:  lib.ActionAdd,
			wantErr: "no entry is generated",
		},
		{
			name:    "invalid YAML",
			content: "payload:\n\t- '1.1.1.1'\n",
			action:  lib.ActionAdd,
			wantErr: "yaml:",
		},
		{
			name:      "malformed IPv4-mapped CIDR",
			content:   "payload:\n  - '::ffff:1.2.3.0/95'\n",
			action:    lib.ActionAdd,
			wantErrIs: lib.ErrInvalidCIDR,
		},
		{
			name:      "unknown action",
			content:   ruleSet,
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

					args := withInputSource(t, source, ".yaml", tt.content, tt.args)
					got, err := runInput(t, TypeClashRuleSetIPCIDRIn, tt.action, args, tt.initial)
					assertInputResult(t, TypeClashRuleSetIPCIDRIn, tt.action, got, err, tt.want, tt.wantErr, tt.wantErrIs)
				})
			}
		})
	}
}

func TestClashRuleSetClassicalIn_Input(t *testing.T) {
	t.Parallel()

	const ruleSet = "# Clash classical rule set\n" +
		"payload:\n" +
		"  - DOMAIN-SUFFIX,example.com\n" +
		"  - IP-CIDR,192.168.0.3/24\n" +
		"  - IP-CIDR,192.168.1.5/24,no-resolve\n" +
		"  - IP-CIDR6,2001:db8::/33\n" +
		"  - ip-cidr6,2001:db8:8000::/33,no-resolve\n" +
		"  - IP-CIDR6,::ffff:1.2.3.0/120\n" +
		"  - IP-ASN,13335\n" +
		"  - IP-CIDR,\n" +
		"  - MATCH,DIRECT\n"

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
			name:    "add",
			content: ruleSet,
			action:  lib.ActionAdd,
			initial: map[string][]string{"CN": {"10.0.0.0/8"}},
			want:    map[string][]string{"CN": {"1.2.3.0/24", "10.0.0.0/8", "192.168.0.0/23", "2001:db8::/32"}},
		},
		{
			name:    "add only IPv4",
			content: ruleSet,
			args:    map[string]any{"onlyIPType": "ipv4"},
			action:  lib.ActionAdd,
			want:    map[string][]string{"CN": {"1.2.3.0/24", "192.168.0.0/23"}},
		},
		{
			name:    "remove only IPv6",
			content: "payload:\n  - IP-CIDR,10.0.0.0/9\n  - IP-CIDR6,2001:db8::/33,no-resolve\n",
			args:    map[string]any{"onlyIPType": "ipv6"},
			action:  lib.ActionRemove,
			initial: map[string][]string{"CN": {"10.0.0.0/8", "2001:db8::/32"}},
			want:    map[string][]string{"CN": {"10.0.0.0/8", "2001:db8:8000::/33"}},
		},
		{
			name:    "rule set without IP rules creates an empty list",
			content: "payload:\n  - DOMAIN,example.com\n",
			action:  lib.ActionAdd,
			want:    map[string][]string{"CN": nil},
		},
		{
			name:    "wantedList not containing the list",
			content: ruleSet,
			args:    map[string]any{"wantedList": []string{"jp"}},
			action:  lib.ActionAdd,
			wantErr: "no entry is generated",
		},
		{
			name:    "invalid YAML",
			content: "payload: [IP-CIDR,1.1.1.1\n",
			action:  lib.ActionAdd,
			wantErr: "yaml:",
		},
		{
			name:      "malformed IPv4-mapped CIDR",
			content:   "payload:\n  - IP-CIDR6,::ffff:1.2.3.4/129\n",
			action:    lib.ActionAdd,
			wantErrIs: lib.ErrInvalidCIDR,
		},
		{
			name:    "remove from missing list",
			content: ruleSet,
			action:  lib.ActionRemove,
			wantErr: "entry CN not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for _, source := range allInputSources {
				t.Run(string(source), func(t *testing.T) {
					t.Parallel()

					args := withInputSource(t, source, ".yaml", tt.content, tt.args)
					got, err := runInput(t, TypeClashRuleSetClassicalIn, tt.action, args, tt.initial)
					assertInputResult(t, TypeClashRuleSetClassicalIn, tt.action, got, err, tt.want, tt.wantErr, tt.wantErrIs)
				})
			}
		})
	}
}

func TestClashRuleSetIn_Config(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ipcidr := writeTestFile(t, dir, "ipcidr.yaml", "payload:\n  - '1.1.1.0/24'\n")
	classical := writeTestFile(t, dir, "classical.yaml", "payload:\n  - IP-CIDR6,2001:db8::/32,no-resolve\n")

	instance, err := newInstanceFromConfig(t, map[string]any{
		"input": []map[string]any{
			{"type": "clashRuleSet", "action": "add", "args": map[string]any{"name": "cn", "uri": ipcidr}},
			{"type": "clashRuleSetClassical", "action": "add", "args": map[string]any{"name": "cn", "uri": classical}},
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

func TestClashRuleSetIn_ConfigInvalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		iType   string
		args    map[string]any
		wantErr string
	}{
		{
			iType:   TypeClashRuleSetIPCIDRIn,
			args:    map[string]any{"name": "cn", "ipOrCIDR": []string{"1.1.1.1"}},
			wantErr: "❌ [type clashRuleSet | action add] ipOrCIDR is invalid for this input format",
		},
		{
			iType:   TypeClashRuleSetClassicalIn,
			args:    map[string]any{"inputDir": filepath.Join("clash", "yaml"), "uri": "./cn.yaml"},
			wantErr: "❌ [type clashRuleSetClassical | action add] inputDir is not allowed to be used with name or uri or ipOrCIDR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.iType, func(t *testing.T) {
			t.Parallel()

			_, err := newInstanceFromConfig(t, map[string]any{
				"input": []map[string]any{{"type": tt.iType, "action": "add", "args": tt.args}},
			})
			assertErrContains(t, err, tt.wantErr)
		})
	}
}
