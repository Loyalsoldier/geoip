package plaintext

import (
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/tidwall/gjson"
)

func TestNewTextIn(t *testing.T) {
	tests := []struct {
		name    string
		typ     string
		opts    []lib.InputOption
		wantErr bool
	}{
		{"name and uri", TypeTextIn, []lib.InputOption{WithNameAndURI("cn", "a.txt")}, false},
		{"name and ipOrCIDR", TypeTextIn, []lib.InputOption{WithNameAndURI("cn", ""), WithIPOrCIDR([]string{"1.1.1.1"})}, false},
		{"inputDir only", TypeTextIn, []lib.InputOption{nil, WithInputDir("dir")}, false},
		{"ipOrCIDR for non-text", TypeSurgeRuleSetIn, []lib.InputOption{WithNameAndURI("cn", "a"), WithIPOrCIDR([]string{"1.1.1.1"})}, true},
		{"json missing jsonPath", TypeJSONIn, []lib.InputOption{WithNameAndURI("cn", "a")}, true},
		{"json with jsonPath", TypeJSONIn, []lib.InputOption{WithNameAndURI("cn", "a"), WithJSONPath([]string{"a"})}, false},
		{"missing name", TypeTextIn, []lib.InputOption{WithNameAndURI("", "a")}, true},
		{"missing uri and ipOrCIDR", TypeTextIn, []lib.InputOption{WithNameAndURI("cn", "")}, true},
		{"inputDir with name", TypeTextIn, []lib.InputOption{WithInputDir("dir"), WithNameAndURI("cn", "")}, true},
		{"inputDir with uri", TypeTextIn, []lib.InputOption{WithInputDir("dir"), WithNameAndURI("", "a")}, true},
		{"inputDir with ipOrCIDR", TypeTextIn, []lib.InputOption{WithInputDir("dir"), WithIPOrCIDR([]string{"1.1.1.1"})}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTextIn(tt.typ, "desc", lib.ActionAdd, tt.opts...)
			assertErr(t, err, tt.wantErr)
			if tt.wantErr {
				if got != nil {
					t.Fatalf("expected nil converter on error")
				}
				return
			}
			if got.GetType() != tt.typ || got.GetAction() != lib.ActionAdd || got.GetDescription() != "desc" {
				t.Fatalf("unexpected converter fields: %+v", got)
			}
		})
	}
}

func TestNewTextInFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		typ     string
		data    string
		wantErr bool
		check   func(t *testing.T, in *text_in)
	}{
		{
			name: "full config",
			typ:  TypeJSONIn,
			data: `{"name":" cn ","uri":" a.json ","wantedList":[" cn ",""," us"],"onlyIPType":"ipv4","jsonPath":["p"],"removePrefixesInLine":["x"],"removeSuffixesInLine":["y"]}`,
			check: func(t *testing.T, in *text_in) {
				if in.Name != "cn" || in.URI != "a.json" || in.OnlyIPType != lib.IPv4 {
					t.Fatalf("unexpected fields: %+v", in)
				}
				if len(in.Want) != 2 || !in.Want["CN"] || !in.Want["US"] {
					t.Fatalf("unexpected want: %v", in.Want)
				}
				if len(in.JSONPath) != 1 || len(in.RemovePrefixesInLine) != 1 || len(in.RemoveSuffixesInLine) != 1 {
					t.Fatalf("unexpected slices: %+v", in)
				}
			},
		},
		{name: "inputDir", typ: TypeTextIn, data: `{"inputDir":" ./dir "}`, check: func(t *testing.T, in *text_in) {
			if in.InputDir != "./dir" {
				t.Fatalf("InputDir = %q", in.InputDir)
			}
		}},
		{name: "invalid json", typ: TypeTextIn, data: `{`, wantErr: true},
		{name: "empty data", typ: TypeTextIn, data: ``, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTextInFromBytes(tt.typ, "desc", lib.ActionAdd, []byte(tt.data))
			assertErr(t, err, tt.wantErr)
			if tt.check != nil {
				tt.check(t, got.(*text_in))
			}
		})
	}
}

func TestScanFile(t *testing.T) {
	tests := []struct {
		name     string
		typ      string
		jsonPath []string
		prefixes []string
		suffixes []string
		input    string
		want     []string
		wantErr  bool
	}{
		{
			name: "text",
			typ:  TypeTextIn,
			input: "# comment\n1.1.1.1 # c\n2.2.2.0/24 // c\n3.3.3.3 /* c\n\n" +
				"2001:DB8::/32\n",
			want: []string{"1.1.1.1/32", "2.2.2.0/24", "3.3.3.3/32", "2001:db8::/32"},
		},
		{
			name:     "text remove prefixes and suffixes",
			typ:      TypeTextIn,
			prefixes: []string{" IP-CIDR, "},
			suffixes: []string{" ,No-Resolve "},
			input:    "IP-CIDR,1.1.1.0/24,no-resolve\nip-cidr,\n10.0.0.1\n",
			want:     []string{"1.1.1.0/24", "10.0.0.1/32"},
		},
		{name: "text invalid", typ: TypeTextIn, input: "invalid\n", wantErr: true},
		{name: "text scanner error", typ: TypeTextIn, input: strings.Repeat("a", 70000), wantErr: true},
		{
			name:  "clash ipcidr",
			typ:   TypeClashRuleSetIPCIDRIn,
			input: "payload:\n  - '1.1.1.0/24'\n  - ''\n  - '2001:db8::/32'\n",
			want:  []string{"1.1.1.0/24", "2001:db8::/32"},
		},
		{name: "clash ipcidr invalid cidr", typ: TypeClashRuleSetIPCIDRIn, input: "payload:\n  - 'bad'\n", wantErr: true},
		{name: "clash ipcidr invalid yaml", typ: TypeClashRuleSetIPCIDRIn, input: "payload: [", wantErr: true},
		{
			name:  "clash classical",
			typ:   TypeClashRuleSetClassicalIn,
			input: "payload:\n  - IP-CIDR,1.1.1.0/24\n  - IP-CIDR6,2001:db8::/32,no-resolve\n  - ''\n  - IP-CIDR, \n  - DOMAIN,example.com\n",
			want:  []string{"1.1.1.0/24", "2001:db8::/32"},
		},
		{name: "clash classical invalid cidr", typ: TypeClashRuleSetClassicalIn, input: "payload:\n  - IP-CIDR,bad\n", wantErr: true},
		{name: "clash classical invalid yaml", typ: TypeClashRuleSetClassicalIn, input: "payload: [", wantErr: true},
		{
			name:  "surge",
			typ:   TypeSurgeRuleSetIn,
			input: "# c\nIP-CIDR,1.1.1.0/24 // c\nIP-CIDR6,2001:db8::/32,no-resolve /* c\nIP-CIDR, \nDOMAIN,example.com\n\n",
			want:  []string{"1.1.1.0/24", "2001:db8::/32"},
		},
		{name: "surge invalid cidr", typ: TypeSurgeRuleSetIn, input: "IP-CIDR,bad\n", wantErr: true},
		{name: "surge scanner error", typ: TypeSurgeRuleSetIn, input: strings.Repeat("a", 70000), wantErr: true},
		{
			name:     "json",
			typ:      TypeJSONIn,
			jsonPath: []string{" a ", "b.#.c"},
			input:    `{"a":"1.1.1.1","b":[{"c":"2.2.2.0/24"},{"c":["2001:db8::/32"]}]}`,
			want:     []string{"1.1.1.1/32", "2.2.2.0/24", "2001:db8::/32"},
		},
		{name: "json invalid", typ: TypeJSONIn, jsonPath: []string{"a"}, input: `{`, wantErr: true},
		{name: "json path not exist", typ: TypeJSONIn, jsonPath: []string{"x"}, input: `{"a":"1.1.1.1"}`, wantErr: true},
		{name: "json empty string", typ: TypeJSONIn, jsonPath: []string{"a"}, input: `{"a":" "}`, wantErr: true},
		{name: "json invalid cidr", typ: TypeJSONIn, jsonPath: []string{"a"}, input: `{"a":"bad"}`, wantErr: true},
		{name: "json number value", typ: TypeJSONIn, jsonPath: []string{"a"}, input: `{"a":1}`, wantErr: true},
		{name: "json array with invalid item", typ: TypeJSONIn, jsonPath: []string{"a"}, input: `{"a":["1.1.1.1",1]}`, wantErr: true},
		{name: "unsupported type", typ: "unknown", input: "1.1.1.1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := &text_in{Type: tt.typ, JSONPath: tt.jsonPath, RemovePrefixesInLine: tt.prefixes, RemoveSuffixesInLine: tt.suffixes}
			entry := lib.NewEntry("test")
			err := in.scanFile(strings.NewReader(tt.input), entry)
			assertErr(t, err, tt.wantErr)
			if tt.wantErr {
				return
			}
			assertCIDRs(t, entryCIDRs(t, entry), tt.want)
		})
	}
}

func TestScanFileReadError(t *testing.T) {
	for _, typ := range []string{TypeJSONIn, TypeClashRuleSetIPCIDRIn, TypeClashRuleSetClassicalIn} {
		t.Run(typ, func(t *testing.T) {
			in := &text_in{Type: typ, JSONPath: []string{"a"}}
			if err := in.scanFile(errReader{}, lib.NewEntry("test")); err == nil {
				t.Fatal("expected read error")
			}
		})
	}
}

func TestProcessJSONResultDirect(t *testing.T) {
	in := &text_in{}
	entry := lib.NewEntry("test")
	if err := in.processJSONResult(gjson.Parse(`["1.1.1.1","2001:db8::1"]`), entry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertCIDRs(t, entryCIDRs(t, entry), []string{"1.1.1.1/32", "2001:db8::1/128"})
}
