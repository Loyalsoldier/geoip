package special

import (
	"bufio"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestNewStdin(t *testing.T) {
	tests := []struct {
		name       string
		action     lib.Action
		opts       []lib.InputOption
		wantErr    string
		wantName   string
		wantIPType lib.IPType
	}{
		{
			name:     "name is trimmed",
			action:   lib.ActionAdd,
			opts:     []lib.InputOption{WithStdinName("  temp \t")},
			wantName: "temp",
		},
		{
			name:       "remove action with onlyIPType ipv4",
			action:     lib.ActionRemove,
			opts:       []lib.InputOption{WithStdinName("cn"), WithStdinOnlyIPType(lib.IPv4)},
			wantName:   "cn",
			wantIPType: lib.IPv4,
		},
		{
			name:       "onlyIPType ipv6",
			action:     lib.ActionAdd,
			opts:       []lib.InputOption{WithStdinOnlyIPType(lib.IPv6), WithStdinName("cn")},
			wantName:   "cn",
			wantIPType: lib.IPv6,
		},
		{
			name:     "nil options are ignored",
			action:   lib.ActionAdd,
			opts:     []lib.InputOption{nil, WithStdinName("cn"), nil},
			wantName: "cn",
		},
		{
			name:     "unknown action is accepted by constructor",
			action:   lib.Action("unknown"),
			opts:     []lib.InputOption{WithStdinName("cn")},
			wantName: "cn",
		},
		{
			name:    "missing name",
			action:  lib.ActionAdd,
			opts:    []lib.InputOption{WithStdinOnlyIPType(lib.IPv4)},
			wantErr: "❌ [type stdin | action add] name must be specified",
		},
		{
			name:    "blank name",
			action:  lib.ActionRemove,
			opts:    []lib.InputOption{WithStdinName("   ")},
			wantErr: "❌ [type stdin | action remove] name must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewStdin(tt.action, tt.opts...)
			if tt.wantErr != "" {
				assertErrorString(t, err, tt.wantErr)
				if got != nil {
					t.Fatalf("NewStdin() = %v, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewStdin() error = %v", err)
			}

			s, ok := got.(*stdin)
			if !ok {
				t.Fatalf("NewStdin() returned %T, want *stdin", got)
			}
			if s.GetType() != TypeStdin {
				t.Errorf("GetType() = %q, want %q", s.GetType(), TypeStdin)
			}
			if s.GetAction() != tt.action {
				t.Errorf("GetAction() = %q, want %q", s.GetAction(), tt.action)
			}
			if s.GetDescription() != DescStdin {
				t.Errorf("GetDescription() = %q, want %q", s.GetDescription(), DescStdin)
			}
			if s.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", s.Name, tt.wantName)
			}
			if s.OnlyIPType != tt.wantIPType {
				t.Errorf("OnlyIPType = %q, want %q", s.OnlyIPType, tt.wantIPType)
			}
		})
	}
}

func TestNewStdinFromBytes(t *testing.T) {
	tests := []struct {
		name          string
		action        lib.Action
		data          string
		wantErr       string
		wantJSONError bool
		wantName      string
		wantIPType    lib.IPType
	}{
		{
			name:       "valid JSON with all fields",
			action:     lib.ActionAdd,
			data:       `{"name": " cn ", "onlyIPType": "ipv4"}`,
			wantName:   "cn",
			wantIPType: lib.IPv4,
		},
		{
			name:     "valid JSON without onlyIPType",
			action:   lib.ActionRemove,
			data:     `{"name": "us"}`,
			wantName: "us",
		},
		{
			name:    "empty data",
			action:  lib.ActionAdd,
			data:    "",
			wantErr: "❌ [type stdin | action add] name must be specified",
		},
		{
			name:    "missing name",
			action:  lib.ActionAdd,
			data:    `{"onlyIPType": "ipv6"}`,
			wantErr: "❌ [type stdin | action add] name must be specified",
		},
		{
			name:          "invalid JSON",
			action:        lib.ActionAdd,
			data:          `{"name": "cn"`,
			wantJSONError: true,
		},
		{
			name:          "wrong JSON type",
			action:        lib.ActionAdd,
			data:          `{"name": 1}`,
			wantJSONError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewStdinFromBytes(tt.action, []byte(tt.data))
			switch {
			case tt.wantJSONError:
				if err == nil {
					t.Fatalf("NewStdinFromBytes() error = nil, want JSON error")
				}
				if got != nil {
					t.Fatalf("NewStdinFromBytes() = %v, want nil", got)
				}
				var syntaxErr *json.SyntaxError
				var typeErr *json.UnmarshalTypeError
				if !errors.As(err, &syntaxErr) && !errors.As(err, &typeErr) {
					t.Fatalf("error = %v, want wrapped JSON error", err)
				}
				wantPrefix := "❌ [type stdin | action " + string(tt.action) + "] "
				if !strings.HasPrefix(err.Error(), wantPrefix) {
					t.Fatalf("error = %q, want prefix %q", err.Error(), wantPrefix)
				}
				return
			case tt.wantErr != "":
				assertErrorString(t, err, tt.wantErr)
				return
			}

			if err != nil {
				t.Fatalf("NewStdinFromBytes() error = %v", err)
			}
			s := got.(*stdin)
			if s.Type != TypeStdin || s.Action != tt.action || s.Description != DescStdin {
				t.Errorf("got Type=%q Action=%q Description=%q", s.Type, s.Action, s.Description)
			}
			if s.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", s.Name, tt.wantName)
			}
			if s.OnlyIPType != tt.wantIPType {
				t.Errorf("OnlyIPType = %q, want %q", s.OnlyIPType, tt.wantIPType)
			}
		})
	}
}

// TestStdinInput swaps os.Stdin, so it must not run in parallel.
func TestStdinInput(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		action      lib.Action
		onlyIPType  lib.IPType
		entries     []testEntry
		wantErr     string
		wantErrIs   error
		wantLen     int
		wantEntries []testEntry
	}{
		{
			name: "blank lines and comments are skipped",
			input: "\n   \n\t\n# comment\n// comment\n/* comment */\n" +
				"1.1.1.1 # trailing comment\n2.2.2.2 // trailing comment\n3.3.3.3 /* trailing comment */\n  4.4.4.4  \n",
			action:      lib.ActionAdd,
			wantLen:     1,
			wantEntries: []testEntry{{name: "temp", cidrs: []string{"1.1.1.1/32", "2.2.2.2/32", "3.3.3.3/32", "4.4.4.4/32"}}},
		},
		{
			name:        "invalid lines are skipped",
			input:       "not-an-ip\n1.1.1.1\n300.1.1.1\n1.1.1.0/33\nfoo/bar\n2001:db8::1\n2001:db8::/129\n:::\n",
			action:      lib.ActionAdd,
			wantLen:     1,
			wantEntries: []testEntry{{name: "temp", cidrs: []string{"1.1.1.1/32", "2001:db8::1/128"}}},
		},
		{
			name:        "IPv4-mapped IPv6 addresses are converted and malformed ones skipped",
			input:       "::ffff:192.168.1.1\n::ffff:192.168.0.0/120\n::ffff:192.168.0.0/95\n::ffff:1.2.3.4/129\n",
			action:      lib.ActionAdd,
			wantLen:     1,
			wantEntries: []testEntry{{name: "temp", cidrs: []string{"192.168.0.0/24", "192.168.1.1/32"}}},
		},
		{
			name:        "IPv4 CIDRs are normalized and merged",
			input:       "192.168.0.3/24\n192.168.1.5/24\n",
			action:      lib.ActionAdd,
			wantLen:     1,
			wantEntries: []testEntry{{name: "temp", cidrs: []string{"192.168.0.0/23"}}},
		},
		{
			name:        "IPv6 CIDRs are merged",
			input:       "2001:db8::/33\n2001:db8:8000::/33\n",
			action:      lib.ActionAdd,
			wantLen:     1,
			wantEntries: []testEntry{{name: "temp", cidrs: []string{"2001:db8::/32"}}},
		},
		{
			name:        "duplicates and contained prefixes collapse",
			input:       "10.0.0.0/8\n10.1.2.3\n10.0.0.0/8\n2001:db8::/32\n2001:db8::1\n",
			action:      lib.ActionAdd,
			wantLen:     1,
			wantEntries: []testEntry{{name: "temp", cidrs: []string{"10.0.0.0/8", "2001:db8::/32"}}},
		},
		{
			name:        "output is sorted with IPv4 before IPv6",
			input:       "2001:db8::1\n10.0.0.1\n1.0.0.0/24\n",
			action:      lib.ActionAdd,
			wantLen:     1,
			wantEntries: []testEntry{{name: "temp", cidrs: []string{"1.0.0.0/24", "10.0.0.1/32", "2001:db8::1/128"}}},
		},
		{
			name:        "CRLF line endings and missing trailing newline",
			input:       "1.1.1.1\r\n2.2.2.2\r\n2001:db8::1",
			action:      lib.ActionAdd,
			wantLen:     1,
			wantEntries: []testEntry{{name: "temp", cidrs: []string{"1.1.1.1/32", "2.2.2.2/32", "2001:db8::1/128"}}},
		},
		{
			name:        "empty input adds an empty entry",
			input:       "",
			action:      lib.ActionAdd,
			wantLen:     1,
			wantEntries: []testEntry{{name: "temp", cidrs: nil}},
		},
		{
			name:        "add only IPv4",
			input:       "1.1.1.1\n2001:db8::1\n",
			action:      lib.ActionAdd,
			onlyIPType:  lib.IPv4,
			wantLen:     1,
			wantEntries: []testEntry{{name: "temp", cidrs: []string{"1.1.1.1/32"}}},
		},
		{
			name:        "add only IPv6",
			input:       "1.1.1.1\n2001:db8::1\n",
			action:      lib.ActionAdd,
			onlyIPType:  lib.IPv6,
			wantLen:     1,
			wantEntries: []testEntry{{name: "temp", cidrs: []string{"2001:db8::1/128"}}},
		},
		{
			name:    "add merges into existing entry and keeps other entries",
			input:   "1.1.0.0/24\n2001:db8:8000::/33\n",
			action:  lib.ActionAdd,
			entries: []testEntry{{name: "temp", cidrs: []string{"1.1.1.0/24", "2001:db8::/33"}}, {name: "cn", cidrs: []string{"1.0.1.0/24"}}},
			wantLen: 2,
			wantEntries: []testEntry{
				{name: "temp", cidrs: []string{"1.1.0.0/23", "2001:db8::/32"}},
				{name: "cn", cidrs: []string{"1.0.1.0/24"}},
			},
		},
		{
			name:        "remove prefixes from existing entry",
			input:       "10.0.0.0/9\n2001:db8::/33\n",
			action:      lib.ActionRemove,
			entries:     []testEntry{{name: "temp", cidrs: []string{"10.0.0.0/8", "2001:db8::/32"}}},
			wantLen:     1,
			wantEntries: []testEntry{{name: "temp", cidrs: []string{"10.128.0.0/9", "2001:db8:8000::/33"}}},
		},
		{
			name:        "remove only IPv4",
			input:       "10.0.0.0/9\n2001:db8::/33\n",
			action:      lib.ActionRemove,
			onlyIPType:  lib.IPv4,
			entries:     []testEntry{{name: "temp", cidrs: []string{"10.0.0.0/8", "2001:db8::/32"}}},
			wantLen:     1,
			wantEntries: []testEntry{{name: "temp", cidrs: []string{"10.128.0.0/9", "2001:db8::/32"}}},
		},
		{
			name:        "remove only IPv6",
			input:       "10.0.0.0/9\n2001:db8::/33\n",
			action:      lib.ActionRemove,
			onlyIPType:  lib.IPv6,
			entries:     []testEntry{{name: "temp", cidrs: []string{"10.0.0.0/8", "2001:db8::/32"}}},
			wantLen:     1,
			wantEntries: []testEntry{{name: "temp", cidrs: []string{"10.0.0.0/8", "2001:db8:8000::/33"}}},
		},
		{
			name:    "remove when entry is missing",
			input:   "10.0.0.0/9\n",
			action:  lib.ActionRemove,
			entries: []testEntry{{name: "cn", cidrs: []string{"10.0.0.0/8"}}},
			wantErr: "❌ [type stdin | action remove] entry TEMP not found",
		},
		{
			name:      "unknown action",
			input:     "1.1.1.1\n",
			action:    lib.Action("unknown"),
			wantErr:   "❌ [type stdin | action unknown] unknown action",
			wantErrIs: lib.ErrUnknownAction,
		},
		{
			name:      "line longer than the scanner buffer",
			input:     "1.1.1.1\n" + strings.Repeat("a", bufio.MaxScanTokenSize+1) + "\n2.2.2.2\n",
			action:    lib.ActionAdd,
			wantErr:   "❌ [type stdin | action add] " + bufio.ErrTooLong.Error(),
			wantErrIs: bufio.ErrTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setStdin(t, tt.input)
			container := newTestContainer(t, tt.entries...)

			s, err := NewStdin(tt.action, WithStdinName("temp"), WithStdinOnlyIPType(tt.onlyIPType))
			if err != nil {
				t.Fatalf("NewStdin() error = %v", err)
			}

			got, err := s.Input(container)
			if tt.wantErr != "" {
				assertErrorString(t, err, tt.wantErr)
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("errors.Is(%v, %v) = false, want true", err, tt.wantErrIs)
				}
				if got != nil {
					t.Fatalf("Input() container = %v, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Input() error = %v", err)
			}
			if got != container {
				t.Fatalf("Input() returned a different container")
			}
			if got.Len() != tt.wantLen {
				t.Fatalf("container.Len() = %d, want %d", got.Len(), tt.wantLen)
			}
			for _, want := range tt.wantEntries {
				assertEntryCIDRs(t, got, want.name, want.cidrs)
			}
		})
	}
}

// TestStdinConfigCreator swaps os.Stdin, so it must not run in parallel.
func TestStdinConfigCreator(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		wantInitErr bool
		want        []string
	}{
		{
			name:   "valid config",
			config: `{"input": [{"type": "stdin", "action": "add", "args": {"name": "cn"}}]}`,
			want:   []string{"1.0.1.0/24", "2001:250::/35"},
		},
		{
			name:   "valid config with onlyIPType",
			config: `{"input": [{"type": "STDIN", "action": "add", "args": {"name": "cn", "onlyIPType": "ipv4"}}]}`,
			want:   []string{"1.0.1.0/24"},
		},
		{
			name:        "missing name",
			config:      `{"input": [{"type": "stdin", "action": "add"}]}`,
			wantInitErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setStdin(t, "1.0.1.0/24\n2001:250::/35\n")

			instance, err := lib.NewInstance()
			if err != nil {
				t.Fatalf("lib.NewInstance() error = %v", err)
			}
			err = instance.InitConfigFromBytes([]byte(tt.config))
			if tt.wantInitErr {
				if err == nil {
					t.Fatalf("InitConfigFromBytes() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("InitConfigFromBytes() error = %v", err)
			}

			container := lib.NewContainer()
			if err := instance.RunInput(container); err != nil {
				t.Fatalf("RunInput() error = %v", err)
			}
			assertEntryCIDRs(t, container, "cn", tt.want)
		})
	}
}

// TestStdinInputContainerError swaps os.Stdin, so it must not run in parallel.
func TestStdinInputContainerError(t *testing.T) {
	tests := []struct {
		name       string
		action     lib.Action
		failAdd    bool
		failRemove bool
		wantErr    string
	}{
		{name: "container add fails", action: lib.ActionAdd, failAdd: true, wantErr: "❌ [type stdin | action add] fake container failure"},
		{name: "container remove fails", action: lib.ActionRemove, failRemove: true, wantErr: "❌ [type stdin | action remove] fake container failure"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setStdin(t, "1.1.1.1\n")
			inner := newTestContainer(t, testEntry{name: "temp", cidrs: []string{"1.1.1.0/24"}})
			container := &failingContainer{Container: inner, failAdd: tt.failAdd, failRemove: tt.failRemove}

			s, err := NewStdin(tt.action, WithStdinName("temp"))
			if err != nil {
				t.Fatalf("NewStdin() error = %v", err)
			}

			got, err := s.Input(container)
			assertErrorString(t, err, tt.wantErr)
			if !errors.Is(err, errFakeContainer) {
				t.Errorf("errors.Is(%v, errFakeContainer) = false, want true", err)
			}
			if got != nil {
				t.Fatalf("Input() container = %v, want nil", got)
			}
			assertEntryCIDRs(t, inner, "temp", []string{"1.1.1.0/24"})
		})
	}
}
