package special

import (
	"errors"
	"slices"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestNewLookup(t *testing.T) {
	tests := []struct {
		name     string
		opts     []lib.OutputOption
		wantErr  bool
		wantS    string
		wantList []string
	}{
		{"valid", []lib.OutputOption{nil, WithLookupSearch(" 1.1.1.1 "), WithLookupSearchList([]string{"cn"})}, false, "1.1.1.1", []string{"cn"}},
		{"no search", nil, true, "", nil},
		{"blank search", []lib.OutputOption{WithLookupSearch("  ")}, true, "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewLookup(lib.ActionOutput, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			l := got.(*lookup)
			if l.GetType() != TypeLookup || l.GetAction() != lib.ActionOutput || l.GetDescription() != DescLookup {
				t.Fatalf("unexpected metadata: %+v", l)
			}
			if l.Search != tt.wantS || !slices.Equal(l.SearchList, tt.wantList) {
				t.Fatalf("got %+v", l)
			}
		})
	}
}

func TestNewLookupFromBytes(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		wantErr  bool
		wantS    string
		wantList []string
	}{
		{"valid", `{"search":"1.0.0.0/24","searchList":["cn","us"]}`, false, "1.0.0.0/24", []string{"cn", "us"}},
		{"invalid json", `{`, true, "", nil},
		{"empty", ``, true, "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewLookupFromBytes(lib.ActionOutput, []byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			l := got.(*lookup)
			if l.Search != tt.wantS || !slices.Equal(l.SearchList, tt.wantList) {
				t.Fatalf("got %+v", l)
			}
		})
	}
}

func TestLookupOutput(t *testing.T) {
	data := map[string][]string{
		"CN": {"1.0.0.0/16", "2001:db8::/32"},
		"US": {"1.0.0.0/24", "8.8.8.0/24"},
	}
	tests := []struct {
		name    string
		search  string
		list    []string
		wantOut string
		wantErr bool
	}{
		{"ip in multiple", "1.0.0.1", nil, "cn,us\n", false},
		{"ip in one", "1.0.1.1", nil, "cn\n", false},
		{"ip with search list", "1.0.0.1", []string{"us"}, "us\n", false},
		{"ipv6 ip", "2001:db8::1", nil, "cn\n", false},
		{"cidr", "1.0.0.0/24", nil, "cn,us\n", false},
		{"ipv6 cidr", "2001:db8::/48", nil, "cn\n", false},
		{"not found ip", "9.9.9.9", nil, "false\n", false},
		{"not found cidr", "1.0.0.0/8", nil, "false\n", false},
		{"invalid ip", "1.1.1.256", nil, "", true},
		{"invalid cidr", "1.1.1.1/33", nil, "", true},
		{"lookup error", "::ffff:1.0.0.0/64", nil, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := NewLookup(lib.ActionOutput, WithLookupSearch(tt.search), WithLookupSearchList(tt.list))
			if err != nil {
				t.Fatal(err)
			}
			c := newContainerWith(t, data)
			var outErr error
			out := captureStdout(t, func() { outErr = conv.Output(c) })
			if (outErr != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", outErr, tt.wantErr)
			}
			if out != tt.wantOut {
				t.Fatalf("output = %q, want %q", out, tt.wantOut)
			}
		})
	}
}

func TestLookupOutputContainerError(t *testing.T) {
	conv, err := NewLookup(lib.ActionOutput, WithLookupSearch("1.1.1.1"))
	if err != nil {
		t.Fatal(err)
	}
	mc := &mockContainer{Container: lib.NewContainer(), lookupErr: errMock}
	if err := conv.Output(mc); !errors.Is(err, errMock) {
		t.Fatalf("err = %v, want mock error", err)
	}
}
