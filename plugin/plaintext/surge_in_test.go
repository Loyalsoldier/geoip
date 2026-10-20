package plaintext

import (
	"net/http"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestSurgeRuleSetInConstructors(t *testing.T) {
	t.Parallel()

	testInputConstructors(t, TypeSurgeRuleSetIn, DescSurgeRuleSetIn,
		NewSurgeRuleSetIn, NewSurgeRuleSetInFromBytes,
		[]lib.InputOption{WithNameAndURI("cn", "https://example.com/cn.list"), WithInputOnlyIPType(lib.IPv4)},
		`{"name":"cn","uri":"https://example.com/cn.list","onlyIPType":"ipv4"}`,
	)
}

func TestSurgeRuleSetIn_RejectsIPOrCIDR(t *testing.T) {
	t.Parallel()

	conv, err := NewSurgeRuleSetIn(lib.ActionAdd, WithNameAndURI("cn", ""), WithIPOrCIDR([]string{"1.1.1.1"}))
	assertErrContains(t, err, "❌ [type surgeRuleSet | action add] ipOrCIDR is invalid for this input format")
	if conv != nil {
		t.Errorf("expected nil converter, got %#v", conv)
	}
}

func TestSurgeRuleSetIn_Input(t *testing.T) {
	t.Parallel()

	const ruleSet = "# Surge rule set\n" +
		"\n" +
		"DOMAIN-SUFFIX,example.com\n" +
		"IP-CIDR,192.168.0.3/24 // first half\n" +
		"IP-CIDR,192.168.1.5/24,no-resolve # second half\n" +
		"  ip-cidr6,2001:db8::/33  \n" +
		"IP-CIDR6,2001:db8:8000::/33,no-resolve /* comment */\n" +
		"IP-CIDR6,::ffff:1.2.3.0/120\n" +
		"IP-CIDR6,::ffff:5.6.7.8,no-resolve\n" +
		"IP-CIDR,\n" +
		"IP-CIDR, ,no-resolve\n" +
		"IP-CIDR6 ,2001:db9::/32\n" + // ignored: not an "ip-cidr6," rule
		"// IP-CIDR,10.0.0.0/8\n" +
		"IP-ASN,13335,no-resolve\n" +
		"FINAL,DIRECT\n"

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
			initial: map[string][]string{"US": {"2001:db8::/32"}},
			want: map[string][]string{
				"CN": {"1.2.3.0/24", "5.6.7.8/32", "192.168.0.0/23", "2001:db8::/32"},
				"US": {"2001:db8::/32"},
			},
		},
		{
			name:    "add with uppercase prefix and mapped addresses",
			content: "IP-CIDR6,::FFFF:1.2.3.0/120\nIP-CIDR6,::ffff:5.6.7.8,no-resolve\nIP-CIDR6,2001:DB8::ABCD\n",
			action:  lib.ActionAdd,
			want:    map[string][]string{"CN": {"1.2.3.0/24", "5.6.7.8/32", "2001:db8::abcd/128"}},
		},
		{
			name:    "add only IPv4",
			content: ruleSet,
			args:    map[string]any{"onlyIPType": "ipv4"},
			action:  lib.ActionAdd,
			want:    map[string][]string{"CN": {"1.2.3.0/24", "5.6.7.8/32", "192.168.0.0/23"}},
		},
		{
			name:    "remove",
			content: "IP-CIDR,192.168.0.0/24\nIP-CIDR6,2001:db8:8000::/33\n",
			action:  lib.ActionRemove,
			initial: map[string][]string{"CN": {"192.168.0.0/23", "2001:db8::/32"}},
			want:    map[string][]string{"CN": {"192.168.1.0/24", "2001:db8::/33"}},
		},
		{
			name:    "remove only IPv6",
			content: "IP-CIDR,192.168.0.0/24\nIP-CIDR6,2001:db8:8000::/33\n",
			args:    map[string]any{"onlyIPType": "ipv6"},
			action:  lib.ActionRemove,
			initial: map[string][]string{"CN": {"192.168.0.0/23", "2001:db8::/32"}},
			want:    map[string][]string{"CN": {"192.168.0.0/23", "2001:db8::/33"}},
		},
		{
			name:    "wantedList not containing the list",
			content: ruleSet,
			args:    map[string]any{"wantedList": []string{"us", "jp"}},
			action:  lib.ActionAdd,
			wantErr: "no entry is generated",
		},
		{
			name:      "invalid CIDR",
			content:   "IP-CIDR,1.1.1.1/33\n",
			action:    lib.ActionAdd,
			wantErrIs: lib.ErrInvalidCIDR,
		},
		{
			name:      "invalid IP",
			content:   "IP-CIDR,not-an-ip,no-resolve\n",
			action:    lib.ActionAdd,
			wantErrIs: lib.ErrInvalidIP,
		},
		{
			name:      "malformed IPv4-mapped CIDR",
			content:   "IP-CIDR6,::ffff:1.2.3.0/95\n",
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

					args := withInputSource(t, source, ".list", tt.content, tt.args)
					got, err := runInput(t, TypeSurgeRuleSetIn, tt.action, args, tt.initial)
					assertInputResult(t, TypeSurgeRuleSetIn, tt.action, got, err, tt.want, tt.wantErr, tt.wantErrIs)
				})
			}
		})
	}
}

func TestSurgeRuleSetIn_Config(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.StatusOK, "IP-CIDR,1.1.1.0/24,no-resolve\nIP-CIDR6,2001:db8::/32\n")

	instance, err := newInstanceFromConfig(t, map[string]any{
		"input": []map[string]any{
			{"type": "surgeRuleSet", "action": "add", "args": map[string]any{"name": "cn", "uri": srv.URL + "/cn.list", "onlyIPType": "ipv6"}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	container := lib.NewContainer()
	if err := instance.RunInput(container); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContainer(t, container, map[string][]string{"CN": {"2001:db8::/32"}})
}

func TestSurgeRuleSetIn_ConfigInvalid(t *testing.T) {
	t.Parallel()

	_, err := newInstanceFromConfig(t, map[string]any{
		"input": []map[string]any{
			{"type": "surgeRuleSet", "action": "remove", "args": map[string]any{"uri": "./cn.list"}},
		},
	})
	assertErrContains(t, err, "❌ [type surgeRuleSet | action remove] missing inputDir or name")
}
