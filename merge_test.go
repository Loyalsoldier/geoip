package main

import (
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/Loyalsoldier/geoip/plugin/special"
)

const mergeInput = `1.0.0.0/24
1.0.1.0/24
# a comment line
10.0.0.1 // trailing comment

not-an-ip
2001:db8::/33
2001:db8:8000::/33
1.0.0.128/25
`

func TestMergeCmd(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		input string
		want  []string
	}{
		{
			name:  "all types",
			args:  []string{"merge"},
			input: mergeInput,
			want:  []string{"1.0.0.0/23", "10.0.0.1/32", "2001:db8::/32"},
		},
		{
			name:  "alias m",
			args:  []string{"m"},
			input: mergeInput,
			want:  []string{"1.0.0.0/23", "10.0.0.1/32", "2001:db8::/32"},
		},
		{
			name:  "only ipv4",
			args:  []string{"merge", "-t", "ipv4"},
			input: mergeInput,
			want:  []string{"1.0.0.0/23", "10.0.0.1/32"},
		},
		{
			name:  "only ipv6 long flag, mixed case and spaces",
			args:  []string{"merge", "--onlyiptype", "  IPv6 "},
			input: mergeInput,
			want:  []string{"2001:db8::/32"},
		},
		{
			name:  "only ipv6 but no ipv6 input prints nothing",
			args:  []string{"merge", "-t", "ipv6"},
			input: "192.168.0.0/24\n192.168.1.0/24\n",
			want:  []string{},
		},
		{
			name:  "ipv4-mapped ipv6 normalized to ipv4",
			args:  []string{"merge"},
			input: "::ffff:192.168.0.1\n192.168.0.0/32\n",
			want:  []string{"192.168.0.0/31"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withStdin(t, tt.input)
			var err error
			out := captureStdout(t, func() { err = runCmd(t, tt.args...) })
			if err != nil {
				t.Fatalf("Execute error: %v", err)
			}
			got := strings.Fields(out)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d lines %v, want %d %v", len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("line %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
			wantRaw := ""
			for _, s := range tt.want {
				wantRaw += s + "\n"
			}
			if out != wantRaw {
				t.Errorf("raw output = %q, want %q", out, wantRaw)
			}
		})
	}
	// NOTE: invalid -t values (e.g. "ipv5") hit log.Fatal and are not tested.
}

func TestGetInputForMerge(t *testing.T) {
	in := getInputForMerge()
	if in == nil {
		t.Fatal("nil input")
	}
	if got := in.GetType(); got != special.TypeStdin {
		t.Errorf("type = %q, want %q", got, special.TypeStdin)
	}
	if got := in.GetAction(); got != lib.ActionAdd {
		t.Errorf("action = %q, want %q", got, lib.ActionAdd)
	}
	if got := fieldString(t, in, "Name"); got != "temp" {
		t.Errorf("name = %q, want temp", got)
	}
}

func TestGetOutputForMerge(t *testing.T) {
	tests := []struct {
		otype string
		want  lib.IPType
	}{
		{"", ""},
		{"ipv4", lib.IPv4},
		{"ipv6", lib.IPv6},
		{"other", ""},
		{"IPV4", ""}, // caller lowercases; function itself is case-sensitive
	}
	for _, tt := range tests {
		t.Run("otype="+tt.otype, func(t *testing.T) {
			out := getOutputForMerge(tt.otype)
			if out == nil {
				t.Fatal("nil output")
			}
			if got := out.GetType(); got != special.TypeStdout {
				t.Errorf("type = %q, want %q", got, special.TypeStdout)
			}
			if got := out.GetAction(); got != lib.ActionOutput {
				t.Errorf("action = %q, want %q", got, lib.ActionOutput)
			}
			if got := fieldString(t, out, "OnlyIPType"); got != string(tt.want) {
				t.Errorf("OnlyIPType = %q, want %q", got, tt.want)
			}
		})
	}
}
