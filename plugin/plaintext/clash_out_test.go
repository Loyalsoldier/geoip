package plaintext

import (
	"path/filepath"
	"testing"
)

func TestClashRuleSetClassicalOutConstructors(t *testing.T) {
	t.Parallel()

	testOutputConstructors(t, TypeClashRuleSetClassicalOut, DescClashRuleSetClassicalOut,
		filepath.Join("output", "clash", "classical"),
		NewClashRuleSetClassicalOut, NewClashRuleSetClassicalOutFromBytes)
}

func TestClashRuleSetIPCIDROutConstructors(t *testing.T) {
	t.Parallel()

	testOutputConstructors(t, TypeClashRuleSetIPCIDROut, DescClashRuleSetIPCIDROut,
		filepath.Join("output", "clash", "ipcidr"),
		NewClashRuleSetIPCIDROut, NewClashRuleSetIPCIDROutFromBytes)
}

func TestClashRuleSetOut_Output(t *testing.T) {
	t.Parallel()

	lists := map[string][]string{
		"CN":     {"192.168.0.3/24", "192.168.1.5/24", "2001:db8::/33", "2001:db8:8000::/33", "::ffff:1.2.3.0/120"},
		"US":     {"8.8.8.8", "::ffff:8.8.4.4"},
		"V6ONLY": {"2001:db8:1::/48", "2001:DB8::ABCD"},
	}

	tests := []struct {
		name  string
		iType string
		args  map[string]any
		lists map[string][]string
		want  map[string]string
	}{
		{
			name:  "ipcidr all lists",
			iType: TypeClashRuleSetIPCIDROut,
			lists: lists,
			want: map[string]string{
				"cn.txt":     "payload:\n  - '1.2.3.0/24'\n  - '192.168.0.0/23'\n  - '2001:db8::/32'\n",
				"us.txt":     "payload:\n  - '8.8.4.4/32'\n  - '8.8.8.8/32'\n",
				"v6only.txt": "payload:\n  - '2001:db8::abcd/128'\n  - '2001:db8:1::/48'\n",
			},
		},
		{
			name:  "ipcidr ignores line prefix and suffix",
			iType: TypeClashRuleSetIPCIDROut,
			args: map[string]any{
				"outputExtension": ".yaml",
				"wantedList":      []string{"CN"},
				"addPrefixInLine": "IP-CIDR,",
				"addSuffixInLine": ",no-resolve",
			},
			lists: lists,
			want: map[string]string{
				"cn.yaml": "payload:\n  - '1.2.3.0/24'\n  - '192.168.0.0/23'\n  - '2001:db8::/32'\n",
			},
		},
		{
			name:  "ipcidr only IPv4 with excluded list",
			iType: TypeClashRuleSetIPCIDROut,
			args:  map[string]any{"onlyIPType": "ipv4", "excludedList": []string{"v6only"}},
			lists: lists,
			want: map[string]string{
				"cn.txt": "payload:\n  - '1.2.3.0/24'\n  - '192.168.0.0/23'\n",
				"us.txt": "payload:\n  - '8.8.4.4/32'\n  - '8.8.8.8/32'\n",
			},
		},
		{
			name:  "classical all lists",
			iType: TypeClashRuleSetClassicalOut,
			lists: lists,
			want: map[string]string{
				"cn.txt":     "payload:\n  - IP-CIDR,1.2.3.0/24\n  - IP-CIDR,192.168.0.0/23\n  - IP-CIDR6,2001:db8::/32\n",
				"us.txt":     "payload:\n  - IP-CIDR,8.8.4.4/32\n  - IP-CIDR,8.8.8.8/32\n",
				"v6only.txt": "payload:\n  - IP-CIDR6,2001:db8::abcd/128\n  - IP-CIDR6,2001:db8:1::/48\n",
			},
		},
		{
			name:  "classical ignores line prefix and suffix",
			iType: TypeClashRuleSetClassicalOut,
			args: map[string]any{
				"outputExtension": ".yml",
				"wantedList":      []string{"us", "missing"},
				"addPrefixInLine": "HOST,",
				"addSuffixInLine": ",no-resolve",
			},
			lists: lists,
			want: map[string]string{
				"us.yml": "payload:\n  - IP-CIDR,8.8.4.4/32\n  - IP-CIDR,8.8.8.8/32\n",
			},
		},
		{
			name:  "classical only IPv6 with wanted list",
			iType: TypeClashRuleSetClassicalOut,
			args:  map[string]any{"onlyIPType": "ipv6", "wantedList": []string{"cn", "v6only"}},
			lists: lists,
			want: map[string]string{
				"cn.txt":     "payload:\n  - IP-CIDR6,2001:db8::/32\n",
				"v6only.txt": "payload:\n  - IP-CIDR6,2001:db8::abcd/128\n  - IP-CIDR6,2001:db8:1::/48\n",
			},
		},
		{
			name:  "classical with everything excluded writes nothing",
			iType: TypeClashRuleSetClassicalOut,
			args:  map[string]any{"excludedList": []string{"cn", "us", "v6only"}},
			lists: lists,
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			outputDir, err := runOutput(t, tt.iType, tt.args, tt.lists)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertOutputFiles(t, outputDir, tt.want)
		})
	}
}

func TestClashRuleSetOut_Output_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		iType   string
		args    map[string]any
		lists   map[string][]string
		wantErr string
	}{
		{
			name:    "ipcidr list without IPv6 prefix",
			iType:   TypeClashRuleSetIPCIDROut,
			args:    map[string]any{"onlyIPType": "ipv6"},
			lists:   map[string][]string{"CN": {"2001:db8::/32"}, "US": {"8.8.8.8"}},
			wantErr: "❌ [type clashRuleSet | action output] entry US has no prefix",
		},
		{
			name:    "classical empty list",
			iType:   TypeClashRuleSetClassicalOut,
			lists:   map[string][]string{"EMPTY": nil},
			wantErr: "❌ [type clashRuleSetClassical | action output] entry EMPTY has no prefix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := runOutput(t, tt.iType, tt.args, tt.lists)
			assertErrContains(t, err, tt.wantErr)
		})
	}
}

func TestClashRuleSetOut_Config(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	classicalDir := filepath.Join(dir, "classical")
	ipcidrDir := filepath.Join(dir, "ipcidr")

	instance, err := newInstanceFromConfig(t, map[string]any{
		"output": []map[string]any{
			{"type": "clashRuleSetClassical", "action": "output", "args": map[string]any{"outputDir": classicalDir, "outputExtension": ".yaml"}},
			{"type": "clashRuleSet", "action": "output", "args": map[string]any{"outputDir": ipcidrDir, "onlyIPType": "ipv4"}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	container := newTestContainer(t, map[string][]string{"CN": {"1.1.1.1/24", "2001:db8::/32"}})
	if err := instance.RunOutput(container); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertOutputFiles(t, classicalDir, map[string]string{
		"cn.yaml": "payload:\n  - IP-CIDR,1.1.1.0/24\n  - IP-CIDR6,2001:db8::/32\n",
	})
	assertOutputFiles(t, ipcidrDir, map[string]string{
		"cn.txt": "payload:\n  - '1.1.1.0/24'\n",
	})
}

func TestClashRuleSetOut_ConfigInvalid(t *testing.T) {
	t.Parallel()

	for _, iType := range []string{TypeClashRuleSetClassicalOut, TypeClashRuleSetIPCIDROut} {
		t.Run(iType, func(t *testing.T) {
			t.Parallel()

			_, err := newInstanceFromConfig(t, map[string]any{
				"output": []map[string]any{
					{"type": iType, "action": "output", "args": map[string]any{"wantedList": "cn"}},
				},
			})
			assertErrContains(t, err, "❌ [type "+iType+" | action output]")
			assertErrContains(t, err, "cannot unmarshal")
		})
	}
}
