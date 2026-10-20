package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

const (
	// envSubprocess marks a re-executed test binary that is expected to hit
	// a log.Fatal path.
	envSubprocess = "GEOIP_TEST_FATAL"
	// envSubprocessArgs holds the JSON-encoded arguments for the subprocess.
	envSubprocessArgs = "GEOIP_TEST_FATAL_ARGS"
)

// commandFlags lists the flags defined by every command so that they can be
// reset between executions; cobra keeps flag values and their "changed"
// state across rootCmd.Execute calls in the same process.
var commandFlags = map[*cobra.Command][]string{
	rootCmd:    {"help"},
	listCmd:    {"help"},
	convertCmd: {"help", "config"},
	mergeCmd:   {"help", "onlyiptype"},
	lookupCmd:  {"help", "format", "uri", "dir", "searchlist"},
}

// resetFlags restores every known flag to its default value and clears its
// changed state.
func resetFlags(t *testing.T) {
	t.Helper()

	for cmd, names := range commandFlags {
		for _, name := range names {
			f := cmd.Flags().Lookup(name)
			if f == nil {
				f = cmd.PersistentFlags().Lookup(name)
			}
			if f == nil {
				continue // e.g. the help flag before the first execution
			}

			var err error
			if sv, ok := f.Value.(interface{ Replace([]string) error }); ok {
				err = sv.Replace([]string{})
			} else {
				err = f.Value.Set(f.DefValue)
			}
			if err != nil {
				t.Fatalf("reset flag %q of command %q: %v", name, cmd.Name(), err)
			}
			f.Changed = false
		}
	}
}

// captureStdout runs fn while os.Stdout is redirected to a pipe and returns
// everything written to it. os.Stdout is always restored.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}

	done := make(chan string, 1)
	go func() {
		var b strings.Builder
		_, _ = io.Copy(&b, r)
		_ = r.Close()
		done <- b.String()
	}()

	orig := os.Stdout
	os.Stdout = w
	func() {
		defer func() {
			os.Stdout = orig
			_ = w.Close()
		}()
		fn()
	}()

	return <-done
}

// captureLog redirects the standard logger to a buffer for the rest of the
// test and returns the buffer.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(orig) })

	return &buf
}

// setStdin replaces os.Stdin with a file holding content for the rest of the
// test. The original os.Stdin is restored on cleanup.
func setStdin(t *testing.T, content string) {
	t.Helper()

	path := writeFile(t, t.TempDir(), "stdin.txt", content)
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("os.Open() error = %v", err)
	}

	orig := os.Stdin
	os.Stdin = f
	t.Cleanup(func() {
		os.Stdin = orig
		_ = f.Close()
	})
}

// writeFile writes content to dir/name and returns the full path.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}

	return path
}

// executeCommand runs rootCmd in-process with args. It returns what was
// printed to os.Stdout, what cobra printed itself (usage and errors) and the
// error returned by Execute. Flags are reset before and after the run.
func executeCommand(t *testing.T, args ...string) (stdout, cobraOutput string, err error) {
	t.Helper()

	resetFlags(t)
	t.Cleanup(func() { resetFlags(t) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	if args == nil {
		args = []string{}
	}
	rootCmd.SetArgs(args)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs([]string{})
	})

	stdout = captureStdout(t, func() {
		err = rootCmd.Execute()
	})

	return stdout, buf.String(), err
}

// subprocessArgs reports whether the current process is a re-executed test
// binary started by runSubprocess, and if so returns the arguments for it.
func subprocessArgs(t *testing.T) ([]string, bool) {
	t.Helper()

	if os.Getenv(envSubprocess) != "1" {
		return nil, false
	}

	var args []string
	if err := json.Unmarshal([]byte(os.Getenv(envSubprocessArgs)), &args); err != nil {
		t.Fatalf("decode %s: %v", envSubprocessArgs, err)
	}

	return args, true
}

// runSubprocess re-executes the test binary, running only testName with args
// passed through the environment and stdin as standard input. It returns the
// exit code and the standard error output of the subprocess.
func runSubprocess(t *testing.T, testName string, args []string, stdin string) (int, string) {
	t.Helper()

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^"+testName+"$", "-test.count=1")
	cmd.Env = append(os.Environ(), envSubprocess+"=1", envSubprocessArgs+"="+string(data))
	cmd.Stdin = strings.NewReader(stdin)
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err == nil {
		return 0, stderr.String()
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("running subprocess: %v", err)
	}

	return exitErr.ExitCode(), stderr.String()
}

// assertFatal asserts that a subprocess exited with status 1 (log.Fatal) and
// that its standard error contains wantMsg.
func assertFatal(t *testing.T, code int, stderr, wantMsg string) {
	t.Helper()

	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stderr, wantMsg) {
		t.Fatalf("stderr = %q, want it to contain %q", stderr, wantMsg)
	}
}

// assertLines compares the length first and then every line of got and want.
func assertLines(t *testing.T, got, want string) {
	t.Helper()

	gotLines := strings.Split(got, "\n")
	wantLines := strings.Split(want, "\n")
	if len(gotLines) != len(wantLines) {
		t.Fatalf("line count = %d, want %d\ngot:\n%s\nwant:\n%s", len(gotLines), len(wantLines), got, want)
	}
	for i := range wantLines {
		if gotLines[i] != wantLines[i] {
			t.Errorf("line %d = %q, want %q", i, gotLines[i], wantLines[i])
		}
	}
}

func TestRootCmd(t *testing.T) {
	if rootCmd.Use != "geoip" {
		t.Errorf("rootCmd.Use = %q, want %q", rootCmd.Use, "geoip")
	}
	if rootCmd.Short == "" {
		t.Errorf("rootCmd.Short is empty")
	}
	if !rootCmd.CompletionOptions.HiddenDefaultCmd {
		t.Errorf("rootCmd.CompletionOptions.HiddenDefaultCmd = false, want true")
	}

	tests := []struct {
		name    string
		cmd     *cobra.Command
		aliases []string
	}{
		{name: "convert", cmd: convertCmd, aliases: []string{"conv"}},
		{name: "list", cmd: listCmd, aliases: []string{"l", "ls"}},
		{name: "lookup", cmd: lookupCmd, aliases: []string{"find"}},
		{name: "merge", cmd: mergeCmd, aliases: []string{"m"}},
	}

	// cobra lazily adds its default "help" and "completion" commands on the
	// first Execute, so ignore them to keep this test order independent.
	var registered []string
	for _, c := range rootCmd.Commands() {
		switch c.Name() {
		case "help":
			continue
		case "completion":
			if !c.Hidden {
				t.Errorf("default completion command should be hidden")
			}
			continue
		}
		registered = append(registered, c.Name())
	}
	slices.Sort(registered)
	wantRegistered := []string{"convert", "list", "lookup", "merge"}
	if len(registered) != len(wantRegistered) {
		t.Fatalf("registered commands = %q, want %q", registered, wantRegistered)
	}
	for i := range wantRegistered {
		if registered[i] != wantRegistered[i] {
			t.Errorf("registered command [%d] = %q, want %q", i, registered[i], wantRegistered[i])
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cmd.Name() != tt.name {
				t.Errorf("Name() = %q, want %q", tt.cmd.Name(), tt.name)
			}
			if tt.cmd.Parent() != rootCmd {
				t.Errorf("command %q is not attached to rootCmd", tt.name)
			}
			if tt.cmd.Short == "" {
				t.Errorf("command %q has an empty Short description", tt.name)
			}
			if tt.cmd.Run == nil {
				t.Errorf("command %q has no Run function", tt.name)
			}
			if len(tt.cmd.Aliases) != len(tt.aliases) {
				t.Fatalf("Aliases = %q, want %q", tt.cmd.Aliases, tt.aliases)
			}
			for i := range tt.aliases {
				if tt.cmd.Aliases[i] != tt.aliases[i] {
					t.Errorf("Aliases[%d] = %q, want %q", i, tt.cmd.Aliases[i], tt.aliases[i])
				}
			}

			for _, alias := range append([]string{tt.name}, tt.aliases...) {
				found, _, err := rootCmd.Find([]string{alias})
				if err != nil {
					t.Fatalf("rootCmd.Find(%q) error = %v", alias, err)
				}
				if found != tt.cmd {
					t.Errorf("rootCmd.Find(%q) = %q, want %q", alias, found.Name(), tt.name)
				}
			}
		})
	}
}

func TestCommandFlags(t *testing.T) {
	tests := []struct {
		name       string
		cmd        *cobra.Command
		flag       string
		shorthand  string
		defValue   string
		persistent bool
	}{
		{name: "convert config", cmd: convertCmd, flag: "config", shorthand: "c", defValue: "config.json", persistent: true},
		{name: "merge onlyiptype", cmd: mergeCmd, flag: "onlyiptype", shorthand: "t", defValue: "", persistent: true},
		{name: "lookup format", cmd: lookupCmd, flag: "format", shorthand: "f", defValue: ""},
		{name: "lookup uri", cmd: lookupCmd, flag: "uri", shorthand: "u", defValue: ""},
		{name: "lookup dir", cmd: lookupCmd, flag: "dir", shorthand: "d", defValue: ""},
		{name: "lookup searchlist", cmd: lookupCmd, flag: "searchlist", shorthand: "l", defValue: "[]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := tt.cmd.Flags()
			if tt.persistent {
				flags = tt.cmd.PersistentFlags()
			}
			f := flags.Lookup(tt.flag)
			if f == nil {
				t.Fatalf("flag %q not found", tt.flag)
			}
			if f.Shorthand != tt.shorthand {
				t.Errorf("Shorthand = %q, want %q", f.Shorthand, tt.shorthand)
			}
			if f.DefValue != tt.defValue {
				t.Errorf("DefValue = %q, want %q", f.DefValue, tt.defValue)
			}
			if f.Usage == "" {
				t.Errorf("flag %q has an empty usage", tt.flag)
			}
		})
	}

	t.Run("lookup format is required", func(t *testing.T) {
		got := lookupCmd.Flags().Lookup("format").Annotations[cobra.BashCompOneRequiredFlag]
		if len(got) != 1 || got[0] != "true" {
			t.Fatalf("format annotation %q = %q, want [\"true\"]", cobra.BashCompOneRequiredFlag, got)
		}
	})

	t.Run("lookup dir is marked as directory", func(t *testing.T) {
		if _, ok := lookupCmd.Flags().Lookup("dir").Annotations[cobra.BashCompSubdirsInDir]; !ok {
			t.Fatalf("dir flag has no %q annotation", cobra.BashCompSubdirsInDir)
		}
	})
}

func TestRootCmdHelp(t *testing.T) {
	stdout, cobraOutput, err := executeCommand(t)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}

	for _, want := range []string{"Usage:", "geoip [command]", "convert", "list", "lookup", "merge"} {
		if !strings.Contains(cobraOutput, want) {
			t.Errorf("help output does not contain %q:\n%s", want, cobraOutput)
		}
	}
	// The default completion command is hidden.
	if strings.Contains(cobraOutput, "completion") {
		t.Errorf("help output should not list the hidden completion command:\n%s", cobraOutput)
	}
}

func TestRootCmdUnknownCommand(t *testing.T) {
	stdout, _, err := executeCommand(t, "bogus")
	if err == nil {
		t.Fatalf("Execute() error = nil, want error")
	}
	want := `unknown command "bogus" for "geoip"`
	if !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("Execute() error = %q, want prefix %q", err.Error(), want)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
}

// TestMainFunc runs main() in-process for a successful command. The
// log.Fatal branch of main is covered by TestMainFatal in a subprocess.
func TestMainFunc(t *testing.T) {
	resetFlags(t)
	rootCmd.SetArgs([]string{"list"})
	t.Cleanup(func() {
		rootCmd.SetArgs([]string{})
		resetFlags(t)
	})

	out := captureStdout(t, main)
	if !strings.HasPrefix(out, "All available input formats:\n") {
		t.Fatalf("stdout = %q, want list output", out)
	}
}

func TestMainFatal(t *testing.T) {
	if args, ok := subprocessArgs(t); ok {
		rootCmd.SetArgs(args)
		main()
		return
	}

	tests := []struct {
		name    string
		args    []string
		wantMsg string
	}{
		{name: "unknown command", args: []string{"bogus"}, wantMsg: `unknown command "bogus" for "geoip"`},
		{name: "unknown flag", args: []string{"list", "--bogus"}, wantMsg: "unknown flag: --bogus"},
		{name: "lookup missing required flag", args: []string{"lookup", "-u", "x"}, wantMsg: `required flag(s) "format" not set`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stderr := runSubprocess(t, "TestMainFatal", tt.args, "")
			assertFatal(t, code, stderr, tt.wantMsg)
		})
	}

	t.Run("successful command exits with status 0", func(t *testing.T) {
		code, stderr := runSubprocess(t, "TestMainFatal", []string{"list"}, "")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
		}
	})
}
