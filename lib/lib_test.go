package lib

import (
	"errors"
	"testing"
)

func TestConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  any
		want any
	}{
		{name: "ActionAdd", got: ActionAdd, want: Action("add")},
		{name: "ActionRemove", got: ActionRemove, want: Action("remove")},
		{name: "ActionOutput", got: ActionOutput, want: Action("output")},
		{name: "IPv4", got: IPv4, want: IPType("ipv4")},
		{name: "IPv6", got: IPv6, want: IPType("ipv6")},
		{name: "CaseRemovePrefix", got: CaseRemovePrefix, want: CaseRemove(0)},
		{name: "CaseRemoveEntry", got: CaseRemoveEntry, want: CaseRemove(1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.got != tt.want {
				t.Fatalf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestActionsRegistry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		action Action
		want   bool
	}{
		{action: ActionAdd, want: true},
		{action: ActionRemove, want: true},
		{action: ActionOutput, want: true},
		{action: "", want: false},
		{action: "ADD", want: false},
		{action: "unknown", want: false},
	}

	if len(ActionsRegistry) != 3 {
		t.Fatalf("len(ActionsRegistry) = %d, want 3", len(ActionsRegistry))
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			t.Parallel()
			if got := ActionsRegistry[tt.action]; got != tt.want {
				t.Fatalf("ActionsRegistry[%q] = %v, want %v", tt.action, got, tt.want)
			}
		})
	}
}

func TestIgnoreIPOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opt  IgnoreIPOption
		want IPType
	}{
		{name: "IgnoreIPv4", opt: IgnoreIPv4, want: IPv4},
		{name: "IgnoreIPv6", opt: IgnoreIPv6, want: IPv6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.opt(); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err  error
		want string
	}{
		{err: ErrDuplicatedConverter, want: "duplicated converter"},
		{err: ErrUnknownAction, want: "unknown action"},
		{err: ErrNotSupportedFormat, want: "not supported format"},
		{err: ErrInvalidIPType, want: "invalid IP type"},
		{err: ErrInvalidIP, want: "invalid IP address"},
		{err: ErrInvalidIPLength, want: "invalid IP address length"},
		{err: ErrInvalidIPNet, want: "invalid IPNet address"},
		{err: ErrInvalidCIDR, want: "invalid CIDR"},
		{err: ErrInvalidPrefix, want: "invalid prefix"},
		{err: ErrInvalidPrefixType, want: "invalid prefix type"},
		{err: ErrCommentLine, want: "comment line"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if tt.err.Error() != tt.want {
				t.Fatalf("Error() = %q, want %q", tt.err.Error(), tt.want)
			}
			for _, other := range tests {
				if other.err != tt.err && errors.Is(tt.err, other.err) {
					t.Fatalf("%v should not match %v", tt.err, other.err)
				}
			}
		})
	}
}
