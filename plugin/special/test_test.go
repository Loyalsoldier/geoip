package special

import (
	"errors"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestNewTest(t *testing.T) {
	for _, action := range []lib.Action{lib.ActionAdd, lib.ActionRemove} {
		t.Run(string(action), func(t *testing.T) {
			got, err := newTestFromBytes(action, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got.GetType() != typeTest || got.GetAction() != action || got.GetDescription() != descTest {
				t.Fatalf("got %+v", got)
			}
			if _, err := newTest(action, nil, func(lib.InputConverter) {}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestTestInput(t *testing.T) {
	tests := []struct {
		name    string
		action  lib.Action
		initial map[string][]string
		want    []string
	}{
		{"add", lib.ActionAdd, nil, []string{"127.0.0.0/8"}},
		{"add merges", lib.ActionAdd, map[string][]string{"test": {"10.0.0.0/8"}}, []string{"10.0.0.0/8", "127.0.0.0/8"}},
		{"remove", lib.ActionRemove, map[string][]string{"test": {"127.0.0.0/7", "::1/128"}}, []string{"126.0.0.0/8", "::1/128"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, _ := newTest(tt.action)
			c, err := conv.Input(newContainerWith(t, tt.initial))
			if err != nil {
				t.Fatal(err)
			}
			assertCIDRs(t, entryCIDRs(t, c, entryNameTest), tt.want)
		})
	}
}

func TestTestInputErrors(t *testing.T) {
	tests := []struct {
		name      string
		action    lib.Action
		container lib.Container
		badCIDR   bool
		wantIs    error
	}{
		{"unknown action", lib.ActionOutput, lib.NewContainer(), false, lib.ErrUnknownAction},
		{"add error", lib.ActionAdd, &mockContainer{Container: lib.NewContainer(), addErr: errMock}, false, errMock},
		{"remove error", lib.ActionRemove, &mockContainer{Container: lib.NewContainer(), removeErr: errMock}, false, errMock},
		{"remove not found", lib.ActionRemove, lib.NewContainer(), false, nil},
		{"invalid cidr", lib.ActionAdd, lib.NewContainer(), true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.badCIDR {
				orig := testCIDRs
				testCIDRs = []string{"invalid"}
				t.Cleanup(func() { testCIDRs = orig })
			}
			conv, _ := newTest(tt.action)
			got, err := conv.Input(tt.container)
			if err == nil || got != nil {
				t.Fatalf("got (%v, %v), want error", got, err)
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Fatalf("err = %v, want %v", err, tt.wantIs)
			}
		})
	}
}
