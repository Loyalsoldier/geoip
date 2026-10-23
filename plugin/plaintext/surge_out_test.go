package plaintext

import (
	"path/filepath"
	"testing"
)

func TestSurgeRuleSetOutConstructors(t *testing.T) {
	t.Parallel()

	testOutputConstructors(t, TypeSurgeRuleSetOut, DescSurgeRuleSetOut,
		filepath.Join("output", "surge"),
		NewSurgeRuleSetOut, NewSurgeRuleSetOutFromBytes)
}

func TestSurgeRuleSetOut_Output(t *testing.T) {
	t.Parallel()

	lists := map[string][]string{
		"CN":      {"192.168.0.3/24", "192.168.1.5/24", "2001:db8::/33", "2001:db8:8000::/33", "::ffff:1.2.3.0/120"},
		"US":      {"8.8.8.8", "::ffff:8.8.4.4"},
		"V6ONLY":  {"2001:db8:1::/48"},
		"PRIVATE": {"10.0.0.0/8", "fc00::/7"},
	}

	tests := []struct {
		name  string
		args  map[string]any
		lists map[string][]string
		want  map[string]string
	}{
		{
			name:  "all lists",
			lists: lists,
			want: map[string]string{
				"cn.txt":      "IP-CIDR,1.2.3.0/24\nIP-CIDR,192.168.0.0/23\nIP-CIDR6,2001:db8::/32\n",
				"us.txt":      "IP-CIDR,8.8.4.4/32\nIP-CIDR,8.8.8.8/32\n",
				"v6only.txt":  "IP-CIDR6,2001:db8:1::/48\n",
				"private.txt": "IP-CIDR,10.0.0.0/8\nIP-CIDR6,fc00::/7\n",
			},
		},
		{
			name: "suffix is added but prefix is ignored",
			args: map[string]any{
				"outputExtension": ".list",
				"wantedList":      []string{"cn", "private"},
				"addPrefixInLine": "IGNORED,",
				"addSuffixInLine": ",no-resolve",
			},
			lists: lists,
			want: map[string]string{
				"cn.list":      "IP-CIDR,1.2.3.0/24,no-resolve\nIP-CIDR,192.168.0.0/23,no-resolve\nIP-CIDR6,2001:db8::/32,no-resolve\n",
				"private.list": "IP-CIDR,10.0.0.0/8,no-resolve\nIP-CIDR6,fc00::/7,no-resolve\n",
			},
		},
		{
			name:  "only IPv4 with excluded list",
			args:  map[string]any{"onlyIPType": "ipv4", "excludedList": []string{"V6ONLY", "us"}},
			lists: lists,
			want: map[string]string{
				"cn.txt":      "IP-CIDR,1.2.3.0/24\nIP-CIDR,192.168.0.0/23\n",
				"private.txt": "IP-CIDR,10.0.0.0/8\n",
			},
		},
		{
			name:  "only IPv6 with wanted list",
			args:  map[string]any{"onlyIPType": "ipv6", "wantedList": []string{"v6only", "private"}, "outputExtension": ".conf"},
			lists: lists,
			want: map[string]string{
				"v6only.conf":  "IP-CIDR6,2001:db8:1::/48\n",
				"private.conf": "IP-CIDR6,fc00::/7\n",
			},
		},
		{
			name:  "wanted list with only missing lists writes nothing",
			args:  map[string]any{"wantedList": []string{"missing"}},
			lists: lists,
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			outputDir, err := runOutput(t, TypeSurgeRuleSetOut, tt.args, tt.lists)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertOutputFiles(t, outputDir, tt.want)
		})
	}
}

func TestSurgeRuleSetOut_Output_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    map[string]any
		lists   map[string][]string
		wantErr string
	}{
		{
			name:    "list without IPv4 prefix",
			args:    map[string]any{"onlyIPType": "ipv4"},
			lists:   map[string][]string{"V6ONLY": {"2001:db8::/32"}},
			wantErr: "❌ [type surgeRuleSet | action output] entry V6ONLY has no prefix",
		},
		{
			name:    "empty list",
			lists:   map[string][]string{"CN": {"1.1.1.1"}, "EMPTY": nil},
			wantErr: "❌ [type surgeRuleSet | action output] entry EMPTY has no prefix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := runOutput(t, TypeSurgeRuleSetOut, tt.args, tt.lists)
			assertErrContains(t, err, tt.wantErr)
		})
	}
}

func TestSurgeRuleSetOut_Config(t *testing.T) {
	t.Parallel()

	outputDir := filepath.Join(t.TempDir(), "surge")
	instance, err := newInstanceFromConfig(t, map[string]any{
		"output": []map[string]any{
			{
				"type":   "surgeRuleSet",
				"action": "output",
				"args": map[string]any{
					"outputDir":       outputDir,
					"outputExtension": ".conf",
					"wantedList":      []string{"cn"},
					"addSuffixInLine": ",no-resolve",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	container := newTestContainer(t, map[string][]string{
		"CN": {"1.1.1.1/24", "2001:db8::/32"},
		"US": {"8.8.8.8"},
	})
	if err := instance.RunOutput(container); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertOutputFiles(t, outputDir, map[string]string{
		"cn.conf": "IP-CIDR,1.1.1.0/24,no-resolve\nIP-CIDR6,2001:db8::/32,no-resolve\n",
	})
}

func TestSurgeRuleSetOut_ConfigInvalid(t *testing.T) {
	t.Parallel()

	_, err := newInstanceFromConfig(t, map[string]any{
		"output": []map[string]any{
			{"type": "surgeRuleSet", "action": "output", "args": map[string]any{"outputDir": 1}},
		},
	})
	assertErrContains(t, err, "❌ [type surgeRuleSet | action output]")
	assertErrContains(t, err, "cannot unmarshal")
}
