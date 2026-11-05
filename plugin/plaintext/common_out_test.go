package plaintext

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestGetDefaultOutputDir(t *testing.T) {
	tests := []struct{ typ, want string }{
		{TypeTextOut, defaultOutputDirForTextOut},
		{TypeClashRuleSetClassicalOut, defaultOutputDirForClashRuleSetClassicalOut},
		{TypeClashRuleSetIPCIDROut, defaultOutputDirForClashRuleSetIPCIDROut},
		{TypeSurgeRuleSetOut, defaultOutputDirForSurgeRuleSetOut},
		{"unknown", defaultOutputDirForTextOut},
	}
	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			if got := getDefaultOutputDir(tt.typ); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewTextOutFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		typ     string
		data    string
		wantErr bool
		want    text_out
	}{
		{
			name: "defaults",
			typ:  TypeSurgeRuleSetOut,
			want: text_out{OutputDir: defaultOutputDirForSurgeRuleSetOut, OutputExt: defaultOutputExt},
		},
		{
			name: "full config",
			typ:  TypeTextOut,
			data: `{"outputDir":" out ","outputExtension":" .list ","wantedList":["cn"],"excludedList":["us"],"onlyIPType":"ipv6","addPrefixInLine":"p","addSuffixInLine":"s"}`,
			want: text_out{OutputDir: "out", OutputExt: ".list", OnlyIPType: lib.IPv6, AddPrefixInLine: "p", AddSuffixInLine: "s"},
		},
		{name: "invalid json", typ: TypeTextOut, data: `{`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTextOutFromBytes(tt.typ, "desc", lib.ActionOutput, []byte(tt.data))
			assertErr(t, err, tt.wantErr)
			if tt.wantErr {
				return
			}
			o := got.(*text_out)
			if o.Type != tt.typ || o.Action != lib.ActionOutput || o.Description != "desc" ||
				o.OutputDir != tt.want.OutputDir || o.OutputExt != tt.want.OutputExt || o.OnlyIPType != tt.want.OnlyIPType ||
				o.AddPrefixInLine != tt.want.AddPrefixInLine || o.AddSuffixInLine != tt.want.AddSuffixInLine {
				t.Fatalf("unexpected output converter: %+v", o)
			}
		})
	}
}

func TestNewTextOutNilOption(t *testing.T) {
	got, err := newTextOut(TypeTextOut, "desc", lib.ActionOutput, nil)
	if err != nil || got == nil {
		t.Fatalf("unexpected result: %v, %v", got, err)
	}
}

func TestMarshalBytes(t *testing.T) {
	entry := lib.NewEntry("test")
	for _, c := range []string{"1.1.1.0/24", "2001:db8::/32"} {
		if err := entry.AddPrefix(c); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name    string
		out     text_out
		entry   *lib.Entry
		want    string
		wantErr bool
	}{
		{name: "text", out: text_out{Type: TypeTextOut}, entry: entry, want: "1.1.1.0/24\n2001:db8::/32\n"},
		{name: "text prefix suffix", out: text_out{Type: TypeTextOut, AddPrefixInLine: "<", AddSuffixInLine: ">"}, entry: entry, want: "<1.1.1.0/24>\n<2001:db8::/32>\n"},
		{name: "text only ipv4", out: text_out{Type: TypeTextOut, OnlyIPType: lib.IPv4}, entry: entry, want: "1.1.1.0/24\n"},
		{name: "text only ipv6", out: text_out{Type: TypeTextOut, OnlyIPType: lib.IPv6}, entry: entry, want: "2001:db8::/32\n"},
		{name: "clash classical", out: text_out{Type: TypeClashRuleSetClassicalOut}, entry: entry, want: "payload:\n  - IP-CIDR,1.1.1.0/24\n  - IP-CIDR6,2001:db8::/32\n"},
		{name: "clash ipcidr", out: text_out{Type: TypeClashRuleSetIPCIDROut}, entry: entry, want: "payload:\n  - '1.1.1.0/24'\n  - '2001:db8::/32'\n"},
		{name: "surge", out: text_out{Type: TypeSurgeRuleSetOut, AddSuffixInLine: ",no-resolve"}, entry: entry, want: "IP-CIDR,1.1.1.0/24,no-resolve\nIP-CIDR6,2001:db8::/32,no-resolve\n"},
		{name: "surge no suffix", out: text_out{Type: TypeSurgeRuleSetOut}, entry: entry, want: "IP-CIDR,1.1.1.0/24\nIP-CIDR6,2001:db8::/32\n"},
		{name: "unsupported type", out: text_out{Type: "unknown"}, entry: entry, wantErr: true},
		{name: "empty entry", out: text_out{Type: TypeTextOut}, entry: lib.NewEntry("empty"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.out.marshalBytes(tt.entry)
			assertErr(t, err, tt.wantErr)
			if !tt.wantErr && string(got) != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMarshalBytesInvalidCIDR(t *testing.T) {
	o := &text_out{}
	if err := o.marshalBytesForClashRuleSetClassicalOut(new(bytes.Buffer), []string{"bad"}); err == nil {
		t.Fatal("expected error for clash classical")
	}
	if err := o.marshalBytesForSurgeRuleSetOut(new(bytes.Buffer), []string{"bad"}); err == nil {
		t.Fatal("expected error for surge")
	}
}

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "file")
	if err := os.WriteFile(blocker, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "isdir"), 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		dir      string
		filename string
		wantErr  bool
	}{
		{"ok", filepath.Join(dir, "a", "b"), "x.txt", false},
		{"mkdir error", filepath.Join(blocker, "sub"), "x.txt", true},
		{"write error", dir, "isdir", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &text_out{Type: TypeTextOut, OutputDir: tt.dir}
			err := o.writeFile(tt.filename, []byte("data"))
			assertErr(t, err, tt.wantErr)
			if !tt.wantErr {
				got, err := os.ReadFile(filepath.Join(tt.dir, tt.filename))
				if err != nil || string(got) != "data" {
					t.Fatalf("file content = %q, err = %v", got, err)
				}
			}
		})
	}
}
