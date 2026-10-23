package main

import (
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/Loyalsoldier/geoip/plugin/special"
)

const mergeTestInput = `# comment line
192.168.0.3/24
192.168.1.5/24

2001:db8::/33
2001:db8:8000::/33 // trailing comment
invalid-line
10.0.0.1
::ffff:172.16.0.0/108
::ffff:1.2.3.4/95
`

func TestMergeCmd(t *testing.T) {
	const (
		wantIPv4 = "10.0.0.1/32\n172.16.0.0/12\n192.168.0.0/23\n"
		wantIPv6 = "2001:db8::/32\n"
	)

	tests := []struct {
		name    string
		args    []string
		input   string
		wantOut string
	}{
		{name: "merge both IP types", args: []string{"merge"}, input: mergeTestInput, wantOut: wantIPv4 + wantIPv6},
		{name: "alias m", args: []string{"m"}, input: mergeTestInput, wantOut: wantIPv4 + wantIPv6},
		{name: "only IPv4 with short flag", args: []string{"merge", "-t", "ipv4"}, input: mergeTestInput, wantOut: wantIPv4},
		{name: "only IPv6 with long flag", args: []string{"merge", "--onlyiptype", "ipv6"}, input: mergeTestInput, wantOut: wantIPv6},
		{name: "onlyiptype is trimmed and case insensitive", args: []string{"m", "--onlyiptype= IPv4 "}, input: mergeTestInput, wantOut: wantIPv4},
		{name: "explicit empty onlyiptype", args: []string{"merge", "-t", ""}, input: mergeTestInput, wantOut: wantIPv4 + wantIPv6},
		{name: "only IPv6 with IPv4-only input prints nothing", args: []string{"merge", "-t", "ipv6"}, input: "1.1.1.1\n", wantOut: ""},
		{name: "empty input prints nothing", args: []string{"merge"}, input: "", wantOut: ""},
		{name: "only comments and invalid lines print nothing", args: []string{"merge"}, input: "# a\n// b\n/* c */\nfoo\n", wantOut: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setStdin(t, tt.input)

			stdout, cobraOutput, err := executeCommand(t, tt.args...)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if cobraOutput != "" {
				t.Errorf("cobra output = %q, want empty", cobraOutput)
			}
			assertLines(t, stdout, tt.wantOut)
		})
	}
}

// TestMergeCmdFlagIsolation makes sure a flag set in one execution does not
// leak into the next one.
func TestMergeCmdFlagIsolation(t *testing.T) {
	setStdin(t, "1.1.1.1\n2001:db8::1\n")
	if stdout, _, err := executeCommand(t, "merge", "-t", "ipv4"); err != nil || stdout != "1.1.1.1/32\n" {
		t.Fatalf("first run: stdout = %q, err = %v", stdout, err)
	}

	setStdin(t, "1.1.1.1\n2001:db8::1\n")
	stdout, _, err := executeCommand(t, "merge")
	if err != nil {
		t.Fatalf("second run: Execute() error = %v", err)
	}
	assertLines(t, stdout, "1.1.1.1/32\n2001:db8::1/128\n")
}

func TestGetInputForMerge(t *testing.T) {
	input := getInputForMerge()
	if input == nil {
		t.Fatalf("getInputForMerge() = nil")
	}
	if input.GetType() != special.TypeStdin {
		t.Errorf("GetType() = %q, want %q", input.GetType(), special.TypeStdin)
	}
	if input.GetAction() != lib.ActionAdd {
		t.Errorf("GetAction() = %q, want %q", input.GetAction(), lib.ActionAdd)
	}
	if input.GetDescription() != special.DescStdin {
		t.Errorf("GetDescription() = %q, want %q", input.GetDescription(), special.DescStdin)
	}

	// The input reads from standard input into an entry named "temp".
	setStdin(t, "192.168.0.3/24\n192.168.1.5/24\n")
	container, err := input.Input(lib.NewContainer())
	if err != nil {
		t.Fatalf("Input() error = %v", err)
	}
	if container.Len() != 1 {
		t.Fatalf("container.Len() = %d, want 1", container.Len())
	}
	entry, found := container.GetEntry("temp")
	if !found {
		t.Fatalf("entry %q not found", "temp")
	}
	got, err := entry.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error = %v", err)
	}
	want := []string{"192.168.0.0/23"}
	if len(got) != len(want) {
		t.Fatalf("CIDRs = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("CIDRs[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestGetOutputForMerge(t *testing.T) {
	tests := []struct {
		name    string
		otype   string
		wantOut string
	}{
		{name: "empty means both IP types", otype: "", wantOut: "1.0.1.0/24\n2001:250::/35\n3.0.0.0/8\n"},
		{name: "ipv4", otype: "ipv4", wantOut: "1.0.1.0/24\n3.0.0.0/8\n"},
		{name: "ipv6", otype: "ipv6", wantOut: "2001:250::/35\n"},
		{name: "unknown value means both IP types", otype: "ipv5", wantOut: "1.0.1.0/24\n2001:250::/35\n3.0.0.0/8\n"},
		{name: "value is not normalized here", otype: "IPv4", wantOut: "1.0.1.0/24\n2001:250::/35\n3.0.0.0/8\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := getOutputForMerge(tt.otype)
			if output == nil {
				t.Fatalf("getOutputForMerge(%q) = nil", tt.otype)
			}
			if output.GetType() != special.TypeStdout {
				t.Errorf("GetType() = %q, want %q", output.GetType(), special.TypeStdout)
			}
			if output.GetAction() != lib.ActionOutput {
				t.Errorf("GetAction() = %q, want %q", output.GetAction(), lib.ActionOutput)
			}

			container := lib.NewContainer()
			for name, cidrs := range map[string][]string{
				"cn": {"1.0.1.0/24", "2001:250::/35"},
				"us": {"3.0.0.0/8"},
			} {
				entry := lib.NewEntry(name)
				for _, cidr := range cidrs {
					if err := entry.AddPrefix(cidr); err != nil {
						t.Fatalf("AddPrefix(%q) error = %v", cidr, err)
					}
				}
				if err := container.Add(entry); err != nil {
					t.Fatalf("container.Add() error = %v", err)
				}
			}

			var outErr error
			stdout := captureStdout(t, func() {
				outErr = output.Output(container)
			})
			if outErr != nil {
				t.Fatalf("Output() error = %v", outErr)
			}
			assertLines(t, stdout, tt.wantOut)
		})
	}
}

func TestMergeCmdFatal(t *testing.T) {
	if args, ok := subprocessArgs(t); ok {
		rootCmd.SetArgs(args)
		_ = rootCmd.Execute()
		return
	}

	tests := []struct {
		name    string
		args    []string
		stdin   string
		wantMsg string
	}{
		{name: "invalid onlyiptype", args: []string{"merge", "-t", "ipv5"}, stdin: "1.1.1.1\n", wantMsg: "invalid argument onlyiptype: ipv5"},
		{name: "invalid onlyiptype is normalized in message", args: []string{"merge", "--onlyiptype", " IPV5 "}, stdin: "1.1.1.1\n", wantMsg: "invalid argument onlyiptype: ipv5"},
		{name: "line too long on stdin", args: []string{"merge"}, stdin: strings.Repeat("1", 70*1024) + "\n", wantMsg: "token too long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stderr := runSubprocess(t, "TestMergeCmdFatal", tt.args, tt.stdin)
			assertFatal(t, code, stderr, tt.wantMsg)
		})
	}
}
