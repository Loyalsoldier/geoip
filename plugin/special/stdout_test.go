package special

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestNewStdout(t *testing.T) {
	tests := []struct {
		name        string
		action      lib.Action
		opts        []lib.OutputOption
		wantWant    []string
		wantExclude []string
		wantIPType  lib.IPType
	}{
		{name: "no options", action: lib.ActionOutput},
		{
			name:   "all options",
			action: lib.ActionOutput,
			opts: []lib.OutputOption{
				WithStdoutWantedList([]string{"cn", "us"}),
				WithStdoutExcludedList([]string{"jp"}),
				WithStdoutOnlyIPType(lib.IPv6),
			},
			wantWant:    []string{"cn", "us"},
			wantExclude: []string{"jp"},
			wantIPType:  lib.IPv6,
		},
		{
			name:       "nil options are ignored",
			action:     lib.ActionAdd,
			opts:       []lib.OutputOption{nil, WithStdoutOnlyIPType(lib.IPv4), nil},
			wantIPType: lib.IPv4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewStdout(tt.action, tt.opts...)
			if err != nil {
				t.Fatalf("NewStdout() error = %v", err)
			}
			s, ok := got.(*stdout)
			if !ok {
				t.Fatalf("NewStdout() returned %T, want *stdout", got)
			}
			if s.GetType() != TypeStdout {
				t.Errorf("GetType() = %q, want %q", s.GetType(), TypeStdout)
			}
			if s.GetAction() != tt.action {
				t.Errorf("GetAction() = %q, want %q", s.GetAction(), tt.action)
			}
			if s.GetDescription() != DescStdout {
				t.Errorf("GetDescription() = %q, want %q", s.GetDescription(), DescStdout)
			}
			assertStringSlice(t, "Want", s.Want, tt.wantWant)
			assertStringSlice(t, "Exclude", s.Exclude, tt.wantExclude)
			if s.OnlyIPType != tt.wantIPType {
				t.Errorf("OnlyIPType = %q, want %q", s.OnlyIPType, tt.wantIPType)
			}
		})
	}
}

func TestNewStdoutFromBytes(t *testing.T) {
	tests := []struct {
		name          string
		data          string
		wantJSONError bool
		wantWant      []string
		wantExclude   []string
		wantIPType    lib.IPType
	}{
		{name: "empty data", data: ""},
		{name: "empty JSON object", data: `{}`},
		{
			name:        "all fields",
			data:        `{"wantedList": ["cn", "us"], "excludedList": ["us"], "onlyIPType": "ipv4"}`,
			wantWant:    []string{"cn", "us"},
			wantExclude: []string{"us"},
			wantIPType:  lib.IPv4,
		},
		{name: "invalid JSON", data: `{"wantedList": ["cn"`, wantJSONError: true},
		{name: "wrong JSON type", data: `{"excludedList": "cn"}`, wantJSONError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewStdoutFromBytes(lib.ActionOutput, []byte(tt.data))
			if tt.wantJSONError {
				if err == nil {
					t.Fatalf("NewStdoutFromBytes() error = nil, want JSON error")
				}
				if got != nil {
					t.Fatalf("NewStdoutFromBytes() = %v, want nil", got)
				}
				var syntaxErr *json.SyntaxError
				var typeErr *json.UnmarshalTypeError
				if !errors.As(err, &syntaxErr) && !errors.As(err, &typeErr) {
					t.Fatalf("error = %v, want wrapped JSON error", err)
				}
				if !strings.HasPrefix(err.Error(), "❌ [type stdout | action output] ") {
					t.Fatalf("error = %q, want stdout prefix", err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("NewStdoutFromBytes() error = %v", err)
			}
			s := got.(*stdout)
			if s.Type != TypeStdout || s.Action != lib.ActionOutput || s.Description != DescStdout {
				t.Errorf("got Type=%q Action=%q Description=%q", s.Type, s.Action, s.Description)
			}
			assertStringSlice(t, "Want", s.Want, tt.wantWant)
			assertStringSlice(t, "Exclude", s.Exclude, tt.wantExclude)
			if s.OnlyIPType != tt.wantIPType {
				t.Errorf("OnlyIPType = %q, want %q", s.OnlyIPType, tt.wantIPType)
			}
		})
	}
}

// stdoutTestEntries is shared by the stdout tests. EMPTY holds no prefix at
// all, JP is IPv4 only and V6 is IPv6 only.
var stdoutTestEntries = []testEntry{
	{name: "us", cidrs: []string{"3.0.0.0/8", "2600::/16"}},
	{name: "cn", cidrs: []string{"1.0.1.0/24", "1.0.2.0/23", "2001:250::/35"}},
	{name: "jp", cidrs: []string{"1.0.16.0/20"}},
	{name: "v6", cidrs: []string{"2001:db8::/32"}},
	{name: "empty"},
}

func TestStdoutFilterAndSortList(t *testing.T) {
	tests := []struct {
		name    string
		want    []string
		exclude []string
		result  []string
	}{
		{name: "no lists returns all sorted names", result: []string{"CN", "EMPTY", "JP", "US", "V6"}},
		{name: "excluded list", exclude: []string{" cn ", "Jp", ""}, result: []string{"EMPTY", "US", "V6"}},
		{name: "wanted list is normalized and sorted", want: []string{" v6", "us", "CN", "", "xx"}, result: []string{"CN", "US", "V6", "XX"}},
		{name: "wanted and excluded lists", want: []string{"us", "cn"}, exclude: []string{"CN"}, result: []string{"US"}},
		{name: "wanted list fully excluded", want: []string{"cn", "us"}, exclude: []string{"US", " cn "}, result: []string{}},
		{name: "blank wanted list means no wanted list", want: []string{"", "  "}, exclude: []string{"empty"}, result: []string{"CN", "JP", "US", "V6"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := newTestContainer(t, stdoutTestEntries...)
			s := &stdout{Want: tt.want, Exclude: tt.exclude}

			got := s.filterAndSortList(container)
			if got == nil {
				t.Fatalf("filterAndSortList() = nil, want non-nil slice")
			}
			assertStringSlice(t, "filterAndSortList()", got, tt.result)
		})
	}
}

// TestStdoutOutput swaps os.Stdout, so it must not run in parallel.
func TestStdoutOutput(t *testing.T) {
	tests := []struct {
		name       string
		entries    []testEntry
		want       []string
		exclude    []string
		onlyIPType lib.IPType
		wantOut    string
	}{
		{
			name:    "all entries sorted by name, empty entry skipped",
			entries: stdoutTestEntries,
			wantOut: "1.0.1.0/24\n1.0.2.0/23\n2001:250::/35\n" +
				"1.0.16.0/20\n" +
				"3.0.0.0/8\n2600::/16\n" +
				"2001:db8::/32\n",
		},
		{
			name:       "only IPv4 skips IPv6-only entries",
			entries:    stdoutTestEntries,
			onlyIPType: lib.IPv4,
			wantOut:    "1.0.1.0/24\n1.0.2.0/23\n1.0.16.0/20\n3.0.0.0/8\n",
		},
		{
			name:       "only IPv6 skips IPv4-only entries",
			entries:    stdoutTestEntries,
			onlyIPType: lib.IPv6,
			wantOut:    "2001:250::/35\n2600::/16\n2001:db8::/32\n",
		},
		{
			name:    "wanted list in sorted order",
			entries: stdoutTestEntries,
			want:    []string{"us", " cn "},
			wantOut: "1.0.1.0/24\n1.0.2.0/23\n2001:250::/35\n3.0.0.0/8\n2600::/16\n",
		},
		{
			name:    "wanted entry not found is skipped",
			entries: stdoutTestEntries,
			want:    []string{"xx", "jp"},
			wantOut: "1.0.16.0/20\n",
		},
		{
			name:    "wanted and excluded lists",
			entries: stdoutTestEntries,
			want:    []string{"us", "cn"},
			exclude: []string{"cn"},
			wantOut: "3.0.0.0/8\n2600::/16\n",
		},
		{
			name:    "wanted list fully excluded prints nothing",
			entries: stdoutTestEntries,
			want:    []string{"cn"},
			exclude: []string{" CN "},
			wantOut: "",
		},
		{
			name:    "excluded list only",
			entries: stdoutTestEntries,
			exclude: []string{"cn", "us", "v6"},
			wantOut: "1.0.16.0/20\n",
		},
		{
			name:       "wanted entry without CIDRs for selected IP type",
			entries:    stdoutTestEntries,
			want:       []string{"v6", "empty"},
			onlyIPType: lib.IPv4,
			wantOut:    "",
		},
		{
			name:    "empty container",
			wantOut: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := newTestContainer(t, tt.entries...)

			s, err := NewStdout(lib.ActionOutput,
				WithStdoutWantedList(tt.want),
				WithStdoutExcludedList(tt.exclude),
				WithStdoutOnlyIPType(tt.onlyIPType),
			)
			if err != nil {
				t.Fatalf("NewStdout() error = %v", err)
			}

			var outputErr error
			out := captureStdout(t, func() {
				outputErr = s.Output(container)
			})
			if outputErr != nil {
				t.Fatalf("Output() error = %v", outputErr)
			}

			wantLines := strings.Split(strings.TrimSuffix(tt.wantOut, "\n"), "\n")
			gotLines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
			assertStringSlice(t, "stdout lines", gotLines, wantLines)
			if out != tt.wantOut {
				t.Fatalf("stdout = %q, want %q", out, tt.wantOut)
			}
		})
	}
}

func TestStdoutGenerateCIDRList(t *testing.T) {
	container := newTestContainer(t, stdoutTestEntries...)

	tests := []struct {
		name       string
		entry      string
		onlyIPType lib.IPType
		want       []string
		wantErr    bool
	}{
		{name: "both IP types", entry: "cn", want: []string{"1.0.1.0/24", "1.0.2.0/23", "2001:250::/35"}},
		{name: "only IPv4", entry: "cn", onlyIPType: lib.IPv4, want: []string{"1.0.1.0/24", "1.0.2.0/23"}},
		{name: "only IPv6", entry: "cn", onlyIPType: lib.IPv6, want: []string{"2001:250::/35"}},
		{name: "IPv6-only entry with only IPv4", entry: "v6", onlyIPType: lib.IPv4, wantErr: true},
		{name: "empty entry", entry: "empty", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, found := container.GetEntry(tt.entry)
			if !found {
				t.Fatalf("entry %q not found", tt.entry)
			}
			s := &stdout{OnlyIPType: tt.onlyIPType}

			got, err := s.generateCIDRList(entry)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("generateCIDRList() = %q, want error", got)
				}
				if got != nil {
					t.Fatalf("generateCIDRList() = %q, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("generateCIDRList() error = %v", err)
			}
			assertStringSlice(t, "generateCIDRList()", got, tt.want)
		})
	}
}

// TestStdoutConfigCreator swaps os.Stdout, so it must not run in parallel.
func TestStdoutConfigCreator(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		wantInitErr bool
		wantOut     string
	}{
		{
			name:    "no args",
			config:  `{"output": [{"type": "stdout"}]}`,
			wantOut: "1.0.1.0/24\n2001:250::/35\n3.0.0.0/8\n",
		},
		{
			name:    "with args",
			config:  `{"output": [{"type": "STDOUT", "action": "output", "args": {"wantedList": ["cn", "us"], "excludedList": ["us"], "onlyIPType": "ipv6"}}]}`,
			wantOut: "2001:250::/35\n",
		},
		{
			name:        "invalid args",
			config:      `{"output": [{"type": "stdout", "args": {"wantedList": 1}}]}`,
			wantInitErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			container := newTestContainer(t,
				testEntry{name: "cn", cidrs: []string{"1.0.1.0/24", "2001:250::/35"}},
				testEntry{name: "us", cidrs: []string{"3.0.0.0/8"}},
			)
			out := captureStdout(t, func() {
				if err := instance.RunOutput(container); err != nil {
					t.Errorf("RunOutput() error = %v", err)
				}
			})
			if out != tt.wantOut {
				t.Fatalf("stdout = %q, want %q", out, tt.wantOut)
			}
		})
	}
}
