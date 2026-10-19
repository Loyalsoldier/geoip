package main

import (
	"io"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestMain(m *testing.M) {
	// Silence log output (convert / text output log progress to stderr).
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

// resetFlags restores every flag of cmd to its default value and clears the
// "Changed" state, so that cobra's required / mutually-exclusive flag checks
// are not polluted by previous Execute calls on the shared rootCmd.
func resetFlags(cmd *cobra.Command) {
	reset := func(f *pflag.Flag) {
		if sv, ok := f.Value.(pflag.SliceValue); ok {
			_ = sv.Replace([]string{})
		} else {
			_ = f.Value.Set(f.DefValue)
		}
		f.Changed = false
	}
	cmd.Flags().VisitAll(reset)
	cmd.PersistentFlags().VisitAll(reset)
	for _, sub := range cmd.Commands() {
		resetFlags(sub)
	}
}

// runCmd executes rootCmd with the given args after resetting flag state.
func runCmd(t *testing.T, args ...string) error {
	t.Helper()
	resetFlags(rootCmd)
	rootCmd.SetArgs(args)
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})
	return rootCmd.Execute()
}

// captureStdout swaps os.Stdout with a pipe while fn runs and returns
// everything written to it. os.Stdout is always restored.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	restored := false
	restore := func() {
		if !restored {
			os.Stdout = orig
			restored = true
		}
	}
	t.Cleanup(restore)

	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	fn()

	w.Close()
	restore()
	out := <-done
	r.Close()
	return out
}

// withStdin swaps os.Stdin with a pipe fed with input for the rest of the
// test. os.Stdin is restored on cleanup.
func withStdin(t *testing.T, input string) {
	t.Helper()
	orig := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	go func() {
		_, _ = io.WriteString(w, input)
		w.Close()
	}()
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = orig
		r.Close()
	})
}

// fieldString reads an exported string-kind field from a (pointer to a)
// struct via reflection.
func fieldString(t *testing.T, v any, name string) string {
	t.Helper()
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	f := rv.FieldByName(name)
	if !f.IsValid() {
		t.Fatalf("field %s not found on %T", name, v)
	}
	return f.String()
}

func fieldStrings(t *testing.T, v any, name string) []string {
	t.Helper()
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	f := rv.FieldByName(name)
	if !f.IsValid() {
		t.Fatalf("field %s not found on %T", name, v)
	}
	out := make([]string, 0, f.Len())
	for i := 0; i < f.Len(); i++ {
		out = append(out, f.Index(i).String())
	}
	return out
}

func TestRootCmdSubcommands(t *testing.T) {
	tests := []struct {
		name    string
		aliases []string
	}{
		{"convert", []string{"conv"}},
		{"list", []string{"l", "ls"}},
		{"lookup", []string{"find"}},
		{"merge", []string{"m"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, n := range append([]string{tt.name}, tt.aliases...) {
				cmd, _, err := rootCmd.Find([]string{n})
				if err != nil {
					t.Fatalf("Find(%q) error: %v", n, err)
				}
				if cmd.Name() != tt.name {
					t.Errorf("Find(%q) = %q, want %q", n, cmd.Name(), tt.name)
				}
			}
		})
	}
	if !rootCmd.CompletionOptions.HiddenDefaultCmd {
		t.Error("completion command should be hidden")
	}
}

func TestRootCmdErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"unknown command", []string{"nope"}, "unknown command"},
		{"unknown flag", []string{"list", "--nope"}, "unknown flag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runCmd(t, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func listExpected(t *testing.T) string {
	t.Helper()
	return captureStdout(t, func() {
		lib.ListInputConverter()
		lib.ListOutputConverter()
	})
}

func TestListCmd(t *testing.T) {
	want := listExpected(t)
	mustContain := []string{
		"All available input formats:\n",
		"All available output formats:\n",
		"  - text (Convert plaintext IP & CIDR to other formats)\n",
		"  - stdin (Accept plaintext IP & CIDR from standard input, separated by newline)\n",
		"  - stdout (Convert data to plaintext CIDR format and output to standard output)\n",
		"  - lookup (Lookup specified IP or CIDR from various formats of data)\n",
	}

	for _, alias := range []string{"list", "l", "ls"} {
		t.Run(alias, func(t *testing.T) {
			var err error
			got := captureStdout(t, func() { err = runCmd(t, alias) })
			if err != nil {
				t.Fatalf("Execute error: %v", err)
			}
			if got != want {
				t.Fatalf("output mismatch\ngot:\n%s\nwant:\n%s", got, want)
			}
			for _, s := range mustContain {
				if !strings.Contains(got, s) {
					t.Errorf("output missing %q", s)
				}
			}
			in := strings.Index(got, "All available input formats:")
			out := strings.Index(got, "All available output formats:")
			if in < 0 || out < 0 || in > out {
				t.Errorf("input section should precede output section (in=%d out=%d)", in, out)
			}
		})
	}
}

// TestMainFunc runs main() with arguments that succeed, so log.Fatal is not hit.
func TestMainFunc(t *testing.T) {
	want := listExpected(t)
	resetFlags(rootCmd)
	rootCmd.SetArgs([]string{"list"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	got := captureStdout(t, main)
	if got != want {
		t.Fatalf("main() output mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}
