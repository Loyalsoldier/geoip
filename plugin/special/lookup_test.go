package special

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestNewLookup(t *testing.T) {
	tests := []struct {
		name           string
		action         lib.Action
		opts           []lib.OutputOption
		wantErr        string
		wantSearch     string
		wantSearchList []string
	}{
		{
			name:   "search and search list",
			action: lib.ActionOutput,
			opts: []lib.OutputOption{
				WithLookupSearch("1.1.1.1"),
				WithLookupSearchList([]string{"cn", "us"}),
			},
			wantSearch:     "1.1.1.1",
			wantSearchList: []string{"cn", "us"},
		},
		{
			name:       "search is trimmed",
			action:     lib.ActionOutput,
			opts:       []lib.OutputOption{WithLookupSearch("  1.0.0.0/24 \t")},
			wantSearch: "1.0.0.0/24",
		},
		{
			name:       "nil options are ignored",
			action:     lib.ActionOutput,
			opts:       []lib.OutputOption{nil, WithLookupSearch("::1"), nil},
			wantSearch: "::1",
		},
		{
			name:       "any action is accepted",
			action:     lib.ActionAdd,
			opts:       []lib.OutputOption{WithLookupSearch("::1")},
			wantSearch: "::1",
		},
		{
			name:    "missing search",
			action:  lib.ActionOutput,
			opts:    []lib.OutputOption{WithLookupSearchList([]string{"cn"})},
			wantErr: "❌ [type lookup | action output] search must be specified",
		},
		{
			name:    "blank search",
			action:  lib.ActionOutput,
			opts:    []lib.OutputOption{WithLookupSearch("   ")},
			wantErr: "❌ [type lookup | action output] search must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewLookup(tt.action, tt.opts...)
			if tt.wantErr != "" {
				assertErrorString(t, err, tt.wantErr)
				if got != nil {
					t.Fatalf("NewLookup() = %v, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewLookup() error = %v", err)
			}

			l, ok := got.(*lookup)
			if !ok {
				t.Fatalf("NewLookup() returned %T, want *lookup", got)
			}
			if l.GetType() != TypeLookup {
				t.Errorf("GetType() = %q, want %q", l.GetType(), TypeLookup)
			}
			if l.GetAction() != tt.action {
				t.Errorf("GetAction() = %q, want %q", l.GetAction(), tt.action)
			}
			if l.GetDescription() != DescLookup {
				t.Errorf("GetDescription() = %q, want %q", l.GetDescription(), DescLookup)
			}
			if l.Search != tt.wantSearch {
				t.Errorf("Search = %q, want %q", l.Search, tt.wantSearch)
			}
			assertStringSlice(t, "SearchList", l.SearchList, tt.wantSearchList)
		})
	}
}

func TestNewLookupFromBytes(t *testing.T) {
	tests := []struct {
		name           string
		data           string
		wantErr        string
		wantJSONError  bool
		wantSearch     string
		wantSearchList []string
	}{
		{
			name:           "valid JSON with all fields",
			data:           `{"search": " 1.1.1.1 ", "searchList": ["cn", "us"]}`,
			wantSearch:     "1.1.1.1",
			wantSearchList: []string{"cn", "us"},
		},
		{
			name:       "valid JSON without search list",
			data:       `{"search": "2001:db8::/32"}`,
			wantSearch: "2001:db8::/32",
		},
		{
			name:    "empty data",
			data:    "",
			wantErr: "❌ [type lookup | action output] search must be specified",
		},
		{
			name:    "missing search",
			data:    `{"searchList": ["cn"]}`,
			wantErr: "❌ [type lookup | action output] search must be specified",
		},
		{
			name:          "invalid JSON",
			data:          `{"search": `,
			wantJSONError: true,
		},
		{
			name:          "wrong JSON type",
			data:          `{"search": ["1.1.1.1"]}`,
			wantJSONError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewLookupFromBytes(lib.ActionOutput, []byte(tt.data))
			switch {
			case tt.wantJSONError:
				if err == nil {
					t.Fatalf("NewLookupFromBytes() error = nil, want JSON error")
				}
				if got != nil {
					t.Fatalf("NewLookupFromBytes() = %v, want nil", got)
				}
				var syntaxErr *json.SyntaxError
				var typeErr *json.UnmarshalTypeError
				if !errors.As(err, &syntaxErr) && !errors.As(err, &typeErr) {
					t.Fatalf("error = %v, want wrapped JSON error", err)
				}
				if !strings.HasPrefix(err.Error(), "❌ [type lookup | action output] ") {
					t.Fatalf("error = %q, want lookup prefix", err.Error())
				}
				return
			case tt.wantErr != "":
				assertErrorString(t, err, tt.wantErr)
				return
			}

			if err != nil {
				t.Fatalf("NewLookupFromBytes() error = %v", err)
			}
			l := got.(*lookup)
			if l.Type != TypeLookup || l.Action != lib.ActionOutput || l.Description != DescLookup {
				t.Errorf("got Type=%q Action=%q Description=%q", l.Type, l.Action, l.Description)
			}
			if l.Search != tt.wantSearch {
				t.Errorf("Search = %q, want %q", l.Search, tt.wantSearch)
			}
			assertStringSlice(t, "SearchList", l.SearchList, tt.wantSearchList)
		})
	}
}

func TestLookupOutput(t *testing.T) {
	entries := []testEntry{
		{name: "cn", cidrs: []string{"1.0.1.0/24", "2001:250::/35"}},
		{name: "us", cidrs: []string{"1.0.0.0/16", "2600::/16"}},
		{name: "jp", cidrs: []string{"1.0.16.0/20"}},
		{name: "v6only", cidrs: []string{"2001:db8::/32"}},
	}

	tests := []struct {
		name       string
		entries    []testEntry
		search     string
		searchList []string
		wantOut    string
		wantErr    string
		wantErrIs  error
	}{
		{name: "IP in multiple lists is sorted and lowercased", entries: entries, search: "1.0.1.1", wantOut: "cn,us\n"},
		{name: "IP in a single list", entries: entries, search: "1.0.2.1", wantOut: "us\n"},
		{name: "IP in overlapping lists", entries: entries, search: "1.0.16.1", wantOut: "jp,us\n"},
		{name: "IP not found", entries: entries, search: "8.8.8.8", wantOut: "false\n"},
		{name: "IPv6 found", entries: entries, search: "2001:250::1", wantOut: "cn\n"},
		{name: "IPv6 not found", entries: entries, search: "2001:db9::1", wantOut: "false\n"},
		{name: "IPv4-mapped IPv6 address", entries: entries, search: "::ffff:1.0.1.1", wantOut: "cn,us\n"},
		{name: "CIDR in multiple lists", entries: entries, search: "1.0.1.0/24", wantOut: "cn,us\n"},
		{name: "CIDR only fully contained lists match", entries: entries, search: "1.0.0.0/23", wantOut: "us\n"},
		{name: "CIDR not found", entries: entries, search: "1.0.0.0/15", wantOut: "false\n"},
		{name: "IPv6 CIDR found", entries: entries, search: "2001:db8::/48", wantOut: "v6only\n"},
		{name: "IPv6 CIDR not found", entries: entries, search: "2001:db9::/48", wantOut: "false\n"},
		{name: "IPv4-mapped IPv6 CIDR", entries: entries, search: "::ffff:1.0.1.0/120", wantOut: "cn,us\n"},
		{name: "search list filters results", entries: entries, search: "1.0.1.1", searchList: []string{"cn"}, wantOut: "cn\n"},
		{name: "search list is case and space insensitive", entries: entries, search: "1.0.1.1", searchList: []string{" jp ", "US"}, wantOut: "us\n"},
		{name: "search list without match", entries: entries, search: "1.0.2.1", searchList: []string{"cn", "jp"}, wantOut: "false\n"},
		{name: "blank search list searches all", entries: entries, search: "1.0.1.1", searchList: []string{"", "  "}, wantOut: "cn,us\n"},
		{name: "empty container", search: "1.1.1.1", wantOut: "false\n"},
		{name: "invalid IP", entries: entries, search: "not-an-ip", wantErr: "❌ [type lookup | action output] invalid IP or CIDR"},
		{name: "invalid IPv4 octet", entries: entries, search: "1.2.3.256", wantErr: "❌ [type lookup | action output] invalid IP or CIDR"},
		{name: "invalid CIDR prefix length", entries: entries, search: "1.0.1.0/33", wantErr: "❌ [type lookup | action output] invalid IP or CIDR"},
		{name: "invalid CIDR address", entries: entries, search: "abc/24", wantErr: "❌ [type lookup | action output] invalid IP or CIDR"},
		{
			name:      "container lookup error for IPv4-mapped CIDR shorter than 96 bits",
			entries:   entries,
			search:    "::ffff:1.0.1.0/95",
			wantErr:   "❌ [type lookup | action output] invalid CIDR",
			wantErrIs: lib.ErrInvalidCIDR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := newTestContainer(t, tt.entries...)

			l, err := NewLookup(lib.ActionOutput, WithLookupSearch(tt.search), WithLookupSearchList(tt.searchList))
			if err != nil {
				t.Fatalf("NewLookup() error = %v", err)
			}

			var outputErr error
			out := captureStdout(t, func() {
				outputErr = l.Output(container)
			})

			if tt.wantErr != "" {
				assertErrorString(t, outputErr, tt.wantErr)
				if tt.wantErrIs != nil && !errors.Is(outputErr, tt.wantErrIs) {
					t.Errorf("errors.Is(%v, %v) = false, want true", outputErr, tt.wantErrIs)
				}
				if out != "" {
					t.Errorf("stdout = %q, want empty on error", out)
				}
				return
			}
			if outputErr != nil {
				t.Fatalf("Output() error = %v", outputErr)
			}
			if out != tt.wantOut {
				t.Fatalf("stdout = %q, want %q", out, tt.wantOut)
			}
		})
	}
}

func TestLookupOutputDoesNotModifyContainer(t *testing.T) {
	container := newTestContainer(t,
		testEntry{name: "b", cidrs: []string{"1.0.0.0/8"}},
		testEntry{name: "a", cidrs: []string{"1.0.0.0/8"}},
	)

	l, err := NewLookup(lib.ActionOutput, WithLookupSearch("1.1.1.1"))
	if err != nil {
		t.Fatalf("NewLookup() error = %v", err)
	}
	out := captureStdout(t, func() {
		if err := l.Output(container); err != nil {
			t.Errorf("Output() error = %v", err)
		}
	})
	if out != "a,b\n" {
		t.Fatalf("stdout = %q, want %q", out, "a,b\n")
	}

	if container.Len() != 2 {
		t.Fatalf("container.Len() = %d, want 2", container.Len())
	}
	names := make([]string, 0, container.Len())
	for entry := range container.Loop() {
		names = append(names, entry.GetName())
	}
	slices.Sort(names)
	assertStringSlice(t, "entry names", names, []string{"A", "B"})
	assertEntryCIDRs(t, container, "a", []string{"1.0.0.0/8"})
	assertEntryCIDRs(t, container, "b", []string{"1.0.0.0/8"})
}

func TestLookupConfigCreator(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		wantInitErr bool
		wantOut     string
	}{
		{
			name:    "valid config",
			config:  `{"output": [{"type": "lookup", "args": {"search": "1.0.1.1", "searchList": ["cn"]}}]}`,
			wantOut: "cn\n",
		},
		{
			name:    "valid config without search list",
			config:  `{"output": [{"type": "LOOKUP", "action": "output", "args": {"search": "8.8.8.8"}}]}`,
			wantOut: "false\n",
		},
		{
			name:        "missing search",
			config:      `{"output": [{"type": "lookup", "args": {}}]}`,
			wantInitErr: true,
		},
		{
			name:        "invalid args",
			config:      `{"output": [{"type": "lookup", "args": {"search": 1}}]}`,
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
				testEntry{name: "cn", cidrs: []string{"1.0.1.0/24"}},
				testEntry{name: "us", cidrs: []string{"1.0.0.0/16"}},
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
