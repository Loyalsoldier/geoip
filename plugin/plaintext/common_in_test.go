package plaintext

import (
	"bufio"
	"encoding/json"
	"errors"
	"maps"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/tidwall/gjson"
)

func TestNewTextIn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		iType   string
		opts    []lib.InputOption
		wantErr string
	}{
		{
			name:  "text with name and uri",
			iType: TypeTextIn,
			opts:  []lib.InputOption{WithNameAndURI("cn", "./cn.txt")},
		},
		{
			name:  "text with name and ipOrCIDR",
			iType: TypeTextIn,
			opts:  []lib.InputOption{WithNameAndURI("cn", ""), WithIPOrCIDR([]string{"1.1.1.1"})},
		},
		{
			name:  "text with name, uri and ipOrCIDR",
			iType: TypeTextIn,
			opts:  []lib.InputOption{WithNameAndURI("cn", "./cn.txt"), WithIPOrCIDR([]string{"1.1.1.1"})},
		},
		{
			name:  "text with inputDir",
			iType: TypeTextIn,
			opts:  []lib.InputOption{WithInputDir("./dir")},
		},
		{
			name:  "json with inputDir and jsonPath",
			iType: TypeJSONIn,
			opts:  []lib.InputOption{WithInputDir("./dir"), WithJSONPath([]string{"ips"})},
		},
		{
			name:  "clashRuleSet with name and uri",
			iType: TypeClashRuleSetIPCIDRIn,
			opts:  []lib.InputOption{WithNameAndURI("cn", "./cn.yaml")},
		},
		{
			name:  "clashRuleSetClassical with inputDir",
			iType: TypeClashRuleSetClassicalIn,
			opts:  []lib.InputOption{WithInputDir("./dir")},
		},
		{
			name:  "surgeRuleSet with name and uri",
			iType: TypeSurgeRuleSetIn,
			opts:  []lib.InputOption{WithNameAndURI("cn", "https://example.com/cn.txt")},
		},
		{
			name:  "nil options are ignored",
			iType: TypeTextIn,
			opts:  []lib.InputOption{nil, WithNameAndURI("cn", "./cn.txt"), nil},
		},
		{
			name:  "whitespace-only inputDir is treated as empty",
			iType: TypeTextIn,
			opts:  []lib.InputOption{WithInputDir("   "), WithNameAndURI("cn", "./cn.txt")},
		},
		{
			name:    "ipOrCIDR is invalid for clashRuleSet",
			iType:   TypeClashRuleSetIPCIDRIn,
			opts:    []lib.InputOption{WithNameAndURI("cn", "./cn.yaml"), WithIPOrCIDR([]string{"1.1.1.1"})},
			wantErr: "ipOrCIDR is invalid for this input format",
		},
		{
			name:    "ipOrCIDR is invalid for clashRuleSetClassical",
			iType:   TypeClashRuleSetClassicalIn,
			opts:    []lib.InputOption{WithNameAndURI("cn", "./cn.yaml"), WithIPOrCIDR([]string{"1.1.1.1"})},
			wantErr: "ipOrCIDR is invalid for this input format",
		},
		{
			name:    "ipOrCIDR is invalid for surgeRuleSet",
			iType:   TypeSurgeRuleSetIn,
			opts:    []lib.InputOption{WithNameAndURI("cn", "./cn.txt"), WithIPOrCIDR([]string{"1.1.1.1"})},
			wantErr: "ipOrCIDR is invalid for this input format",
		},
		{
			name:    "ipOrCIDR is invalid for json",
			iType:   TypeJSONIn,
			opts:    []lib.InputOption{WithNameAndURI("cn", "./cn.json"), WithIPOrCIDR([]string{"1.1.1.1"}), WithJSONPath([]string{"ips"})},
			wantErr: "ipOrCIDR is invalid for this input format",
		},
		{
			name:    "json missing jsonPath",
			iType:   TypeJSONIn,
			opts:    []lib.InputOption{WithNameAndURI("cn", "./cn.json")},
			wantErr: "missing jsonPath",
		},
		{
			name:    "json with empty jsonPath",
			iType:   TypeJSONIn,
			opts:    []lib.InputOption{WithNameAndURI("cn", "./cn.json"), WithJSONPath([]string{})},
			wantErr: "missing jsonPath",
		},
		{
			name:    "no options",
			iType:   TypeTextIn,
			wantErr: "missing inputDir or name",
		},
		{
			name:    "uri without name",
			iType:   TypeTextIn,
			opts:    []lib.InputOption{WithNameAndURI("", "./cn.txt")},
			wantErr: "missing inputDir or name",
		},
		{
			name:    "whitespace-only name",
			iType:   TypeTextIn,
			opts:    []lib.InputOption{WithNameAndURI("   ", "./cn.txt")},
			wantErr: "missing inputDir or name",
		},
		{
			name:    "name without uri or ipOrCIDR",
			iType:   TypeTextIn,
			opts:    []lib.InputOption{WithNameAndURI("cn", "")},
			wantErr: "missing uri or ipOrCIDR",
		},
		{
			name:    "name with whitespace-only uri and empty ipOrCIDR",
			iType:   TypeTextIn,
			opts:    []lib.InputOption{WithNameAndURI("cn", "   "), WithIPOrCIDR([]string{})},
			wantErr: "missing uri or ipOrCIDR",
		},
		{
			name:    "name without uri for clashRuleSet",
			iType:   TypeClashRuleSetIPCIDRIn,
			opts:    []lib.InputOption{WithNameAndURI("cn", "")},
			wantErr: "missing uri or ipOrCIDR",
		},
		{
			name:    "inputDir with name",
			iType:   TypeTextIn,
			opts:    []lib.InputOption{WithInputDir("./dir"), WithNameAndURI("cn", "")},
			wantErr: "inputDir is not allowed to be used with name or uri or ipOrCIDR",
		},
		{
			name:    "inputDir with uri",
			iType:   TypeTextIn,
			opts:    []lib.InputOption{WithInputDir("./dir"), WithNameAndURI("", "./cn.txt")},
			wantErr: "inputDir is not allowed to be used with name or uri or ipOrCIDR",
		},
		{
			name:    "inputDir with ipOrCIDR",
			iType:   TypeTextIn,
			opts:    []lib.InputOption{WithInputDir("./dir"), WithIPOrCIDR([]string{"1.1.1.1"})},
			wantErr: "inputDir is not allowed to be used with name or uri or ipOrCIDR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := newTextIn(tt.iType, "test description", lib.ActionAdd, tt.opts...)
			if tt.wantErr != "" {
				assertErrContains(t, err, tt.wantErr)
				assertErrContains(t, err, "[type "+tt.iType+" | action add]")
				if conv != nil {
					t.Errorf("expected nil converter on error, got %#v", conv)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := conv.GetType(); got != tt.iType {
				t.Errorf("GetType() = %q, want %q", got, tt.iType)
			}
			if got := conv.GetAction(); got != lib.ActionAdd {
				t.Errorf("GetAction() = %q, want %q", got, lib.ActionAdd)
			}
			if got := conv.GetDescription(); got != "test description" {
				t.Errorf("GetDescription() = %q, want %q", got, "test description")
			}
		})
	}
}

func TestNewTextIn_OptionsSetFields(t *testing.T) {
	t.Parallel()

	t.Run("name and uri based", func(t *testing.T) {
		t.Parallel()

		conv, err := newTextIn(TypeTextIn, DescTextIn, lib.ActionRemove,
			WithNameAndURI("  cn  ", "  ./cn.txt  "),
			WithIPOrCIDR([]string{" 1.1.1.1 ", "2001:db8::/32"}),
			WithInputWantedList([]string{"cn", " us ", "", "   "}),
			WithInputOnlyIPType(lib.IPv6),
			WithJSONPath([]string{"a", "b"}),
			WithRemovePrefixesInLine([]string{"HOST,", "IP-CIDR,"}),
			WithRemoveSuffixesInLine([]string{",no-resolve"}),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ti := asTextIn(t, conv)
		if ti.Name != "cn" {
			t.Errorf("Name = %q, want %q", ti.Name, "cn")
		}
		if ti.URI != "./cn.txt" {
			t.Errorf("URI = %q, want %q", ti.URI, "./cn.txt")
		}
		if ti.InputDir != "" {
			t.Errorf("InputDir = %q, want empty", ti.InputDir)
		}
		assertStrings(t, "IPOrCIDR", ti.IPOrCIDR, []string{" 1.1.1.1 ", "2001:db8::/32"})
		if want := map[string]bool{"CN": true, "US": true}; !maps.Equal(ti.Want, want) {
			t.Errorf("Want = %v, want %v", ti.Want, want)
		}
		if ti.OnlyIPType != lib.IPv6 {
			t.Errorf("OnlyIPType = %q, want %q", ti.OnlyIPType, lib.IPv6)
		}
		assertStrings(t, "JSONPath", ti.JSONPath, []string{"a", "b"})
		assertStrings(t, "RemovePrefixesInLine", ti.RemovePrefixesInLine, []string{"HOST,", "IP-CIDR,"})
		assertStrings(t, "RemoveSuffixesInLine", ti.RemoveSuffixesInLine, []string{",no-resolve"})
		if ti.Type != TypeTextIn || ti.Action != lib.ActionRemove || ti.Description != DescTextIn {
			t.Errorf("unexpected type/action/description: %q/%q/%q", ti.Type, ti.Action, ti.Description)
		}
	})

	t.Run("inputDir based", func(t *testing.T) {
		t.Parallel()

		conv, err := newTextIn(TypeTextIn, DescTextIn, lib.ActionAdd, WithInputDir("  ./data  "))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ti := asTextIn(t, conv)
		if ti.InputDir != "./data" {
			t.Errorf("InputDir = %q, want %q", ti.InputDir, "./data")
		}
		if ti.Name != "" || ti.URI != "" || len(ti.IPOrCIDR) != 0 {
			t.Errorf("expected empty name/uri/ipOrCIDR, got %q/%q/%q", ti.Name, ti.URI, ti.IPOrCIDR)
		}
		if ti.Want != nil {
			t.Errorf("Want = %v, want nil", ti.Want)
		}
	})
}

func TestWithInputWantedList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		lists []string
		want  map[string]bool
	}{
		{name: "nil", lists: nil, want: map[string]bool{}},
		{name: "empty", lists: []string{}, want: map[string]bool{}},
		{name: "only blanks", lists: []string{"", "  ", "\t"}, want: map[string]bool{}},
		{
			name:  "trimmed, upper-cased and deduplicated",
			lists: []string{"cn", " us ", "Cn", "", "jp\t"},
			want:  map[string]bool{"CN": true, "US": true, "JP": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ti := &text_in{}
			WithInputWantedList(tt.lists)(ti)
			if ti.Want == nil {
				t.Fatal("Want should never be nil after applying WithInputWantedList")
			}
			if !maps.Equal(ti.Want, tt.want) {
				t.Errorf("Want = %v, want %v", ti.Want, tt.want)
			}
		})
	}
}

func TestNewTextInFromBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		iType   string
		action  lib.Action
		data    []byte
		wantErr string
		check   func(t *testing.T, ti *text_in)
	}{
		{
			name:    "nil data",
			iType:   TypeTextIn,
			action:  lib.ActionAdd,
			data:    nil,
			wantErr: "missing inputDir or name",
		},
		{
			name:    "empty data",
			iType:   TypeTextIn,
			action:  lib.ActionAdd,
			data:    []byte{},
			wantErr: "missing inputDir or name",
		},
		{
			name:    "empty object",
			iType:   TypeTextIn,
			action:  lib.ActionAdd,
			data:    []byte(`{}`),
			wantErr: "missing inputDir or name",
		},
		{
			name:    "json null",
			iType:   TypeTextIn,
			action:  lib.ActionAdd,
			data:    []byte(`null`),
			wantErr: "missing inputDir or name",
		},
		{
			name:    "ipOrCIDR with clashRuleSet",
			iType:   TypeClashRuleSetIPCIDRIn,
			action:  lib.ActionAdd,
			data:    []byte(`{"name":"cn","uri":"./cn.yaml","ipOrCIDR":["1.1.1.1"]}`),
			wantErr: "ipOrCIDR is invalid for this input format",
		},
		{
			name:    "json type without jsonPath",
			iType:   TypeJSONIn,
			action:  lib.ActionAdd,
			data:    []byte(`{"name":"cn","uri":"./cn.json"}`),
			wantErr: "missing jsonPath",
		},
		{
			name:   "all text args",
			iType:  TypeTextIn,
			action: lib.ActionRemove,
			data: []byte(`{
				"name": " cn ",
				"uri": " ./cn.txt ",
				"ipOrCIDR": ["1.1.1.1", "2001:db8::/32"],
				"wantedList": ["cn", " us ", ""],
				"onlyIPType": "ipv4",
				"removePrefixesInLine": ["HOST,"],
				"removeSuffixesInLine": [",no-resolve"]
			}`),
			check: func(t *testing.T, ti *text_in) {
				t.Helper()
				if ti.Type != TypeTextIn || ti.Action != lib.ActionRemove || ti.Description != DescTextIn {
					t.Errorf("unexpected type/action/description: %q/%q/%q", ti.Type, ti.Action, ti.Description)
				}
				if ti.Name != "cn" || ti.URI != "./cn.txt" {
					t.Errorf("unexpected name/uri: %q/%q", ti.Name, ti.URI)
				}
				assertStrings(t, "IPOrCIDR", ti.IPOrCIDR, []string{"1.1.1.1", "2001:db8::/32"})
				if want := map[string]bool{"CN": true, "US": true}; !maps.Equal(ti.Want, want) {
					t.Errorf("Want = %v, want %v", ti.Want, want)
				}
				if ti.OnlyIPType != lib.IPv4 {
					t.Errorf("OnlyIPType = %q, want %q", ti.OnlyIPType, lib.IPv4)
				}
				assertStrings(t, "RemovePrefixesInLine", ti.RemovePrefixesInLine, []string{"HOST,"})
				assertStrings(t, "RemoveSuffixesInLine", ti.RemoveSuffixesInLine, []string{",no-resolve"})
				if len(ti.JSONPath) != 0 {
					t.Errorf("JSONPath = %q, want empty", ti.JSONPath)
				}
			},
		},
		{
			name:   "json args with inputDir",
			iType:  TypeJSONIn,
			action: lib.ActionAdd,
			data:   []byte(`{"inputDir":" ./json ","wantedList":["cn"],"onlyIPType":"ipv6","jsonPath":["prefixes.#.ipv4Prefix","prefixes.#.ipv6Prefix"]}`),
			check: func(t *testing.T, ti *text_in) {
				t.Helper()
				if ti.Type != TypeJSONIn || ti.Description != DescJSONIn {
					t.Errorf("unexpected type/description: %q/%q", ti.Type, ti.Description)
				}
				if ti.InputDir != "./json" {
					t.Errorf("InputDir = %q, want %q", ti.InputDir, "./json")
				}
				if want := map[string]bool{"CN": true}; !maps.Equal(ti.Want, want) {
					t.Errorf("Want = %v, want %v", ti.Want, want)
				}
				if ti.OnlyIPType != lib.IPv6 {
					t.Errorf("OnlyIPType = %q, want %q", ti.OnlyIPType, lib.IPv6)
				}
				assertStrings(t, "JSONPath", ti.JSONPath, []string{"prefixes.#.ipv4Prefix", "prefixes.#.ipv6Prefix"})
			},
		},
		{
			name:   "unknown fields are ignored",
			iType:  TypeSurgeRuleSetIn,
			action: lib.ActionAdd,
			data:   []byte(`{"name":"cn","uri":"./cn.txt","foo":"bar"}`),
			check: func(t *testing.T, ti *text_in) {
				t.Helper()
				if ti.Type != TypeSurgeRuleSetIn || ti.Name != "cn" || ti.URI != "./cn.txt" {
					t.Errorf("unexpected type/name/uri: %q/%q/%q", ti.Type, ti.Name, ti.URI)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := newTextInFromBytes(tt.iType, inputDescriptions[tt.iType], tt.action, tt.data)
			if tt.wantErr != "" {
				assertErrContains(t, err, tt.wantErr)
				if conv != nil {
					t.Errorf("expected nil converter on error, got %#v", conv)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.check(t, asTextIn(t, conv))
		})
	}
}

func TestNewTextInFromBytes_InvalidJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		data      string
		checkType func(error) bool
	}{
		{
			name: "truncated json",
			data: `{"name":"cn"`,
			checkType: func(err error) bool {
				var target *json.SyntaxError
				return errors.As(err, &target)
			},
		},
		{
			name: "not json",
			data: `name: cn`,
			checkType: func(err error) bool {
				var target *json.SyntaxError
				return errors.As(err, &target)
			},
		},
		{
			name: "wrong type for name",
			data: `{"name":123,"uri":"./cn.txt"}`,
			checkType: func(err error) bool {
				var target *json.UnmarshalTypeError
				return errors.As(err, &target)
			},
		},
		{
			name: "wrong type for ipOrCIDR",
			data: `{"name":"cn","ipOrCIDR":"1.1.1.1"}`,
			checkType: func(err error) bool {
				var target *json.UnmarshalTypeError
				return errors.As(err, &target)
			},
		},
		{
			name: "array instead of object",
			data: `["cn"]`,
			checkType: func(err error) bool {
				var target *json.UnmarshalTypeError
				return errors.As(err, &target)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := newTextInFromBytes(TypeTextIn, DescTextIn, lib.ActionAdd, []byte(tt.data))
			assertErrContains(t, err, "[type text | action add]")
			if !tt.checkType(err) {
				t.Errorf("unexpected error type %T: %v", errors.Unwrap(err), err)
			}
			if conv != nil {
				t.Errorf("expected nil converter on error, got %#v", conv)
			}
		})
	}
}

func TestTextIn_ScanFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		iType     string
		jsonPath  []string
		content   string
		want      []string
		wantErrIs error
	}{
		{
			name:    "text",
			iType:   TypeTextIn,
			content: "1.1.1.1\n2001:db8::1\n",
			want:    []string{"1.1.1.1/32", "2001:db8::1/128"},
		},
		{
			name:     "json",
			iType:    TypeJSONIn,
			jsonPath: []string{"ips"},
			content:  `{"ips":["1.1.1.1","2001:db8::1"]}`,
			want:     []string{"1.1.1.1/32", "2001:db8::1/128"},
		},
		{
			name:    "clashRuleSetClassical",
			iType:   TypeClashRuleSetClassicalIn,
			content: "payload:\n  - IP-CIDR,1.1.1.1/32\n  - IP-CIDR6,2001:db8::1/128\n",
			want:    []string{"1.1.1.1/32", "2001:db8::1/128"},
		},
		{
			name:    "clashRuleSet",
			iType:   TypeClashRuleSetIPCIDRIn,
			content: "payload:\n  - 1.1.1.1/32\n  - '2001:db8::1/128'\n",
			want:    []string{"1.1.1.1/32", "2001:db8::1/128"},
		},
		{
			name:    "surgeRuleSet",
			iType:   TypeSurgeRuleSetIn,
			content: "IP-CIDR,1.1.1.1/32\nIP-CIDR6,2001:db8::1/128\n",
			want:    []string{"1.1.1.1/32", "2001:db8::1/128"},
		},
		{
			name:      "unsupported type",
			iType:     "unknown",
			content:   "1.1.1.1\n",
			wantErrIs: lib.ErrNotSupportedFormat,
		},
		{
			name:      "empty type",
			iType:     "",
			content:   "1.1.1.1\n",
			wantErrIs: lib.ErrNotSupportedFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ti := &text_in{Type: tt.iType, JSONPath: tt.jsonPath}
			entry := lib.NewEntry("test")
			err := ti.scanFile(strings.NewReader(tt.content), entry)
			assertErr(t, err, "", tt.wantErrIs)
			if tt.wantErrIs != nil {
				return
			}
			assertStrings(t, "CIDRs", entryCIDRs(t, entry), tt.want)
		})
	}
}

func TestTextIn_ScanFile_ReaderError(t *testing.T) {
	t.Parallel()

	for _, iType := range []string{
		TypeTextIn,
		TypeJSONIn,
		TypeClashRuleSetClassicalIn,
		TypeClashRuleSetIPCIDRIn,
		TypeSurgeRuleSetIn,
	} {
		t.Run(iType, func(t *testing.T) {
			t.Parallel()

			ti := &text_in{Type: iType, JSONPath: []string{"ips"}}
			err := ti.scanFile(iotest.ErrReader(errTestRead), lib.NewEntry("test"))
			if !errors.Is(err, errTestRead) {
				t.Fatalf("expected %v, got %v", errTestRead, err)
			}
		})
	}
}

func TestTextIn_ScanFile_LineTooLong(t *testing.T) {
	t.Parallel()

	content := "1.1.1.1\n" + strings.Repeat("1", bufio.MaxScanTokenSize+1) + "\n"
	for _, iType := range []string{TypeTextIn, TypeSurgeRuleSetIn} {
		t.Run(iType, func(t *testing.T) {
			t.Parallel()

			ti := &text_in{Type: iType}
			err := ti.scanFile(strings.NewReader(content), lib.NewEntry("test"))
			if !errors.Is(err, bufio.ErrTooLong) {
				t.Fatalf("expected %v, got %v", bufio.ErrTooLong, err)
			}
		})
	}
}

func TestTextIn_ScanFileForTextIn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		content      string
		prefixes     []string
		suffixes     []string
		want         []string
		wantNoPrefix bool
		wantErrIs    error
	}{
		{
			name:    "plain IPv4 and IPv6 addresses",
			content: "1.1.1.1\n2001:db8::1\n",
			want:    []string{"1.1.1.1/32", "2001:db8::1/128"},
		},
		{
			name:    "CIDRs are normalized",
			content: "1.2.3.4/24\n2001:db8::1/32\n",
			want:    []string{"1.2.3.0/24", "2001:db8::/32"},
		},
		{
			name:    "IPv4-mapped IPv6 address",
			content: "::ffff:1.2.3.4\n",
			want:    []string{"1.2.3.4/32"},
		},
		{
			name:    "IPv4-mapped IPv6 CIDR",
			content: "::ffff:1.2.3.0/120\n",
			want:    []string{"1.2.3.0/24"},
		},
		{
			name:    "IPv4-mapped IPv6 CIDR covering whole IPv4 space",
			content: "::ffff:0:0/96\n",
			want:    []string{"0.0.0.0/0"},
		},
		{
			name:    "upper-case IPv6 is lower-cased",
			content: "2001:DB8::ABCD\n",
			want:    []string{"2001:db8::abcd/128"},
		},
		{
			name: "comments and blank lines",
			content: "# full line comment\n" +
				"\n" +
				"   \n" +
				"\t\n" +
				"1.1.1.1 # trailing hash comment\n" +
				"2.2.2.2 // trailing slash comment\n" +
				"3.3.3.3 /* trailing block comment */\n" +
				"// full line slash comment\n" +
				"/* full line block comment */\n" +
				"  2001:db8::1  \n",
			want: []string{"1.1.1.1/32", "2.2.2.2/32", "3.3.3.3/32", "2001:db8::1/128"},
		},
		{
			name:    "windows line endings",
			content: "1.1.1.1\r\n2001:db8::1\r\n",
			want:    []string{"1.1.1.1/32", "2001:db8::1/128"},
		},
		{
			name:    "no trailing newline",
			content: "1.1.1.1",
			want:    []string{"1.1.1.1/32"},
		},
		{
			name:    "adjacent IPv4 CIDRs are merged",
			content: "192.168.0.3/24\n192.168.1.5/24\n",
			want:    []string{"192.168.0.0/23"},
		},
		{
			name:    "adjacent IPv6 CIDRs are merged",
			content: "2001:db8::/33\n2001:db8:8000::/33\n",
			want:    []string{"2001:db8::/32"},
		},
		{
			name:    "overlapping and duplicated CIDRs are merged",
			content: "10.0.0.0/8\n10.1.0.0/16\n10.0.0.1\n10.0.0.0/8\n",
			want:    []string{"10.0.0.0/8"},
		},
		{
			name:    "IPv4 first then IPv6, each sorted",
			content: "2001:db8::/32\n10.0.0.0/8\n::1\n1.0.0.0/24\n",
			want:    []string{"1.0.0.0/24", "10.0.0.0/8", "::1/128", "2001:db8::/32"},
		},
		{
			name: "remove prefixes and suffixes",
			content: "IP-CIDR,1.0.0.0/24,no-resolve\n" +
				"IP-CIDR6,2001:db8::/32,no-resolve\n" +
				"HOST,8.8.8.8\n" +
				"9.9.9.9;\n",
			prefixes: []string{"IP-CIDR,", " ip-cidr6, ", "Host,"},
			suffixes: []string{",NO-RESOLVE", ";"},
			want:     []string{"1.0.0.0/24", "8.8.8.8/32", "9.9.9.9/32", "2001:db8::/32"},
		},
		{
			name:     "whitespace around value after removing prefix and suffix",
			content:  "HOST,   1.1.1.1   ;\n",
			prefixes: []string{"host,"},
			suffixes: []string{";"},
			want:     []string{"1.1.1.1/32"},
		},
		{
			name:     "line empty after removing prefix is skipped",
			content:  "HOST,\n1.1.1.1\n",
			prefixes: []string{"host,"},
			want:     []string{"1.1.1.1/32"},
		},
		{
			name:     "line empty after removing suffix is skipped",
			content:  ",no-resolve\n1.1.1.1,no-resolve\n",
			suffixes: []string{",no-resolve"},
			want:     []string{"1.1.1.1/32"},
		},
		{
			name:     "each prefix is removed at most once",
			content:  "host,host,1.1.1.1\n",
			prefixes: []string{"host,"},
			wantErrIs: lib.ErrInvalidIP,
		},
		{
			name:         "only comments and blank lines",
			content:      "# comment\n\n// comment\n/* comment */\n",
			wantNoPrefix: true,
		},
		{
			name:         "empty content",
			content:      "",
			wantNoPrefix: true,
		},
		{
			name:      "invalid IP",
			content:   "1.1.1.1\nnot-an-ip\n",
			wantErrIs: lib.ErrInvalidIP,
		},
		{
			name:      "out of range IPv4",
			content:   "300.1.1.1\n",
			wantErrIs: lib.ErrInvalidIP,
		},
		{
			name:      "invalid IPv4 prefix length",
			content:   "1.2.3.4/33\n",
			wantErrIs: lib.ErrInvalidCIDR,
		},
		{
			name:      "invalid IPv6 prefix length",
			content:   "2001:db8::/129\n",
			wantErrIs: lib.ErrInvalidCIDR,
		},
		{
			name:      "IPv4-mapped IPv6 CIDR with too short prefix",
			content:   "::ffff:1.2.3.0/95\n",
			wantErrIs: lib.ErrInvalidCIDR,
		},
		{
			name:      "IPv4-mapped IPv6 CIDR with too long prefix",
			content:   "::ffff:1.2.3.4/129\n",
			wantErrIs: lib.ErrInvalidCIDR,
		},
		{
			name:      "suffix not removed without configuration",
			content:   "1.1.1.1,no-resolve\n",
			wantErrIs: lib.ErrInvalidIP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ti := &text_in{
				Type:                 TypeTextIn,
				RemovePrefixesInLine: tt.prefixes,
				RemoveSuffixesInLine: tt.suffixes,
			}
			entry := lib.NewEntry("test")
			err := ti.scanFileForTextIn(strings.NewReader(tt.content), entry)
			assertErr(t, err, "", tt.wantErrIs)
			if tt.wantErrIs != nil {
				return
			}
			if tt.wantNoPrefix {
				if cidrs, err := entry.MarshalText(); err == nil {
					t.Fatalf("expected no prefix, got %q", cidrs)
				}
				return
			}
			assertStrings(t, "CIDRs", entryCIDRs(t, entry), tt.want)
		})
	}
}

func TestTextIn_ReadClashRuleSetYAMLFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    []string
		wantErr bool
	}{
		{
			name: "valid payload with quotes and comments",
			content: "# comment\n" +
				"payload:\n" +
				"  - '1.1.1.1/32' # trailing comment\n" +
				"  - \"2001:db8::/32\"\n" +
				"  - 10.0.0.0/8\n" +
				"  - ''\n",
			want: []string{"1.1.1.1/32", "2001:db8::/32", "10.0.0.0/8", ""},
		},
		{
			name:    "flow sequence",
			content: "payload: ['1.1.1.1', '2.2.2.2']\n",
			want:    []string{"1.1.1.1", "2.2.2.2"},
		},
		{
			name:    "empty file",
			content: "",
			want:    nil,
		},
		{
			name:    "missing payload key",
			content: "rules:\n  - 1.1.1.1\n",
			want:    nil,
		},
		{
			name:    "empty payload",
			content: "payload: []\n",
			want:    []string{},
		},
		{
			name:    "unclosed flow sequence",
			content: "payload: ['1.1.1.1'\n",
			wantErr: true,
		},
		{
			name:    "tab indentation",
			content: "payload:\n\t- 1.1.1.1\n",
			wantErr: true,
		},
		{
			name:    "payload is a mapping",
			content: "payload:\n  a: b\n",
			wantErr: true,
		},
		{
			name:    "payload is a scalar",
			content: "payload: 1.1.1.1/32\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ti := &text_in{Type: TypeClashRuleSetIPCIDRIn}
			got, err := ti.readClashRuleSetYAMLFile(strings.NewReader(tt.content))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got payload %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertStrings(t, "payload", got, tt.want)
		})
	}
}

func TestTextIn_ScanFileForClashIPCIDRRuleSetIn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		content      string
		want         []string
		wantNoPrefix bool
		wantErr      bool
		wantErrIs    error
	}{
		{
			name: "mixed IPv4, IPv6 and IPv4-mapped with merging",
			content: "payload:\n" +
				"  - '192.168.0.3/24'\n" +
				"  - '192.168.1.5/24'\n" +
				"  - ''\n" +
				"  - '   '\n" +
				"  - '2001:db8::/33'\n" +
				"  - '2001:db8:8000::/33'\n" +
				"  - '  10.0.0.1  '\n" +
				"  - '::ffff:1.2.3.0/120'\n" +
				"  - '::ffff:5.6.7.8'\n",
			want: []string{"1.2.3.0/24", "5.6.7.8/32", "10.0.0.1/32", "192.168.0.0/23", "2001:db8::/32"},
		},
		{
			name:    "comment inside value is stripped",
			content: "payload:\n  - '1.1.1.1 # note'\n  - '2.2.2.2 // note'\n",
			want:    []string{"1.1.1.1/32", "2.2.2.2/32"},
		},
		{
			name:         "empty payload",
			content:      "payload: []\n",
			wantNoPrefix: true,
		},
		{
			name:      "IPv4-mapped CIDR with invalid prefix",
			content:   "payload:\n  - '1.1.1.1'\n  - '::ffff:1.2.3.0/95'\n",
			wantErrIs: lib.ErrInvalidCIDR,
		},
		{
			name:      "invalid IP",
			content:   "payload:\n  - 'example.com'\n",
			wantErrIs: lib.ErrInvalidIP,
		},
		{
			name:      "classical rule is not accepted",
			content:   "payload:\n  - 'IP-CIDR,1.1.1.1/32'\n",
			wantErrIs: lib.ErrInvalidCIDR,
		},
		{
			name:    "invalid YAML",
			content: "payload: [\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ti := &text_in{Type: TypeClashRuleSetIPCIDRIn}
			entry := lib.NewEntry("test")
			err := ti.scanFileForClashIPCIDRRuleSetIn(strings.NewReader(tt.content), entry)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			assertErr(t, err, "", tt.wantErrIs)
			if tt.wantErrIs != nil {
				return
			}
			if tt.wantNoPrefix {
				if cidrs, err := entry.MarshalText(); err == nil {
					t.Fatalf("expected no prefix, got %q", cidrs)
				}
				return
			}
			assertStrings(t, "CIDRs", entryCIDRs(t, entry), tt.want)
		})
	}
}

func TestTextIn_ScanFileForClashClassicalRuleSetIn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		content      string
		want         []string
		wantNoPrefix bool
		wantErr      bool
		wantErrIs    error
	}{
		{
			name: "mixed rules with merging",
			content: "payload:\n" +
				"  - DOMAIN-SUFFIX,example.com\n" +
				"  - IP-CIDR,192.168.0.3/24\n" +
				"  - ip-cidr,192.168.1.5/24,no-resolve\n" +
				"  - IP-CIDR6,2001:db8::/33\n" +
				"  - IP-CIDR6,2001:db8:8000::/33,no-resolve\n" +
				"  - '  IP-CIDR, 8.8.8.8 ,no-resolve  '\n" +
				"  - IP-ASN,13335\n" +
				"  - GEOIP,CN\n" +
				"  - ''\n" +
				"  - IP-CIDR,\n" +
				"  - IP-CIDR,,no-resolve\n" +
				"  - IP-CIDR6,::ffff:1.2.3.0/120\n" +
				"  - MATCH,DIRECT\n",
			want: []string{"1.2.3.0/24", "8.8.8.8/32", "192.168.0.0/23", "2001:db8::/32"},
		},
		{
			name:    "rule keyword is not validated against address family",
			content: "payload:\n  - IP-CIDR,2001:db8::1\n  - IP-CIDR6,1.1.1.1\n",
			want:    []string{"1.1.1.1/32", "2001:db8::1/128"},
		},
		{
			name:    "rule keyword with space before comma is ignored",
			content: "payload:\n  - IP-CIDR ,1.1.1.1\n  - IP-CIDR,2.2.2.2\n",
			want:    []string{"2.2.2.2/32"},
		},
		{
			name:         "only non-IP rules",
			content:      "payload:\n  - DOMAIN,example.com\n  - IP-ASN,13335\n",
			wantNoPrefix: true,
		},
		{
			name:         "empty file",
			content:      "",
			wantNoPrefix: true,
		},
		{
			name:      "invalid IPv4 prefix length",
			content:   "payload:\n  - IP-CIDR,1.2.3.4/33\n",
			wantErrIs: lib.ErrInvalidCIDR,
		},
		{
			name:      "IPv4-mapped CIDR with invalid prefix",
			content:   "payload:\n  - IP-CIDR6,::ffff:1.2.3.4/129\n",
			wantErrIs: lib.ErrInvalidCIDR,
		},
		{
			name:      "invalid IP",
			content:   "payload:\n  - IP-CIDR,abc\n",
			wantErrIs: lib.ErrInvalidIP,
		},
		{
			name:    "invalid YAML",
			content: "payload:\n\t- IP-CIDR,1.1.1.1\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ti := &text_in{Type: TypeClashRuleSetClassicalIn}
			entry := lib.NewEntry("test")
			err := ti.scanFileForClashClassicalRuleSetIn(strings.NewReader(tt.content), entry)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			assertErr(t, err, "", tt.wantErrIs)
			if tt.wantErrIs != nil {
				return
			}
			if tt.wantNoPrefix {
				if cidrs, err := entry.MarshalText(); err == nil {
					t.Fatalf("expected no prefix, got %q", cidrs)
				}
				return
			}
			assertStrings(t, "CIDRs", entryCIDRs(t, entry), tt.want)
		})
	}
}

func TestTextIn_ScanFileForSurgeRuleSetIn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		content      string
		want         []string
		wantNoPrefix bool
		wantErrIs    error
	}{
		{
			name: "mixed rules with comments and merging",
			content: "# Surge rule set\n" +
				"DOMAIN-SUFFIX,example.com\n" +
				"\n" +
				"IP-CIDR,192.168.0.3/24\n" +
				"ip-cidr,192.168.1.5/24,no-resolve // comment\n" +
				"IP-CIDR6,2001:db8::/33 # comment\n" +
				"IP-CIDR6,2001:db8:8000::/33,no-resolve /* comment */\n" +
				"  IP-CIDR, 8.8.8.8 ,no-resolve  \n" +
				"IP-CIDR,\n" +
				"IP-CIDR,,no-resolve\n" +
				"IP-ASN,13335\n" +
				"// IP-CIDR,9.9.9.9/32\n" +
				"# IP-CIDR,9.9.9.9/32\n" +
				"/* IP-CIDR,9.9.9.9/32 */\n",
			want: []string{"8.8.8.8/32", "192.168.0.0/23", "2001:db8::/32"},
		},
		{
			name:    "IPv4-mapped IPv6",
			content: "IP-CIDR,::ffff:1.2.3.4\nIP-CIDR6,::ffff:1.2.3.0/120\n",
			want:    []string{"1.2.3.0/24"},
		},
		{
			name:    "windows line endings",
			content: "IP-CIDR,1.1.1.1/32\r\nIP-CIDR6,2001:db8::1/128\r\n",
			want:    []string{"1.1.1.1/32", "2001:db8::1/128"},
		},
		{
			name:         "only comments",
			content:      "# comment\n// comment\n\n",
			wantNoPrefix: true,
		},
		{
			name:         "plain IP lines are ignored",
			content:      "1.1.1.1\n2001:db8::/32\n",
			wantNoPrefix: true,
		},
		{
			name:      "IPv4-mapped CIDR with invalid prefix",
			content:   "IP-CIDR,::ffff:1.2.3.0/95\n",
			wantErrIs: lib.ErrInvalidCIDR,
		},
		{
			name:      "invalid IP",
			content:   "IP-CIDR,999.1.1.1\n",
			wantErrIs: lib.ErrInvalidIP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ti := &text_in{Type: TypeSurgeRuleSetIn}
			entry := lib.NewEntry("test")
			err := ti.scanFileForSurgeRuleSetIn(strings.NewReader(tt.content), entry)
			assertErr(t, err, "", tt.wantErrIs)
			if tt.wantErrIs != nil {
				return
			}
			if tt.wantNoPrefix {
				if cidrs, err := entry.MarshalText(); err == nil {
					t.Fatalf("expected no prefix, got %q", cidrs)
				}
				return
			}
			assertStrings(t, "CIDRs", entryCIDRs(t, entry), tt.want)
		})
	}
}

func TestTextIn_ScanFileForJSONIn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		content      string
		paths        []string
		want         []string
		wantNoPrefix bool
		wantErr      string
	}{
		{
			name: "google cloud style prefixes with merging",
			content: `{"prefixes":[
				{"ipv4Prefix":"192.168.0.3/24"},
				{"ipv4Prefix":"192.168.1.5/24"},
				{"ipv6Prefix":"2001:db8::/33"},
				{"ipv6Prefix":"2001:db8:8000::/33"}
			]}`,
			paths: []string{"prefixes.#.ipv4Prefix", "  prefixes.#.ipv6Prefix  "},
			want:  []string{"192.168.0.0/23", "2001:db8::/32"},
		},
		{
			name:    "single string value",
			content: `{"ip":"1.1.1.1"}`,
			paths:   []string{"ip"},
			want:    []string{"1.1.1.1/32"},
		},
		{
			name:    "string value with surrounding whitespace",
			content: `{"ip":"  10.0.0.0/8  "}`,
			paths:   []string{"ip"},
			want:    []string{"10.0.0.0/8"},
		},
		{
			name:    "nested arrays and multiple paths",
			content: `{"a":[["1.1.1.1","2.2.2.2"],["::ffff:3.3.3.3"]],"b":"2001:db8::1"}`,
			paths:   []string{"a", "b"},
			want:    []string{"1.1.1.1/32", "2.2.2.2/32", "3.3.3.3/32", "2001:db8::1/128"},
		},
		{
			name:    "top level array",
			content: `["1.1.1.1","::ffff:1.2.3.0/120"]`,
			paths:   []string{"@this"},
			want:    []string{"1.1.1.1/32", "1.2.3.0/24"},
		},
		{
			name:         "empty array",
			content:      `{"a":[]}`,
			paths:        []string{"a"},
			wantNoPrefix: true,
		},
		{
			name:    "invalid JSON",
			content: `{"a":`,
			paths:   []string{"a"},
			wantErr: "invalid JSON data",
		},
		{
			name:    "empty content",
			content: ``,
			paths:   []string{"a"},
			wantErr: "invalid JSON data",
		},
		{
			name:    "path does not exist",
			content: `{"a":"1.1.1.1"}`,
			paths:   []string{"a", "b"},
			wantErr: "value not exist",
		},
		{
			name:    "empty string value",
			content: `{"a":"   "}`,
			paths:   []string{"a"},
			wantErr: "empty string",
		},
		{
			name:    "number value",
			content: `{"a":123}`,
			paths:   []string{"a"},
			wantErr: "invalid IP address or CIDR, please check",
		},
		{
			name:    "object value",
			content: `{"a":{"b":"1.1.1.1"}}`,
			paths:   []string{"a"},
			wantErr: "invalid IP address or CIDR, please check",
		},
		{
			name:    "boolean value",
			content: `{"a":true}`,
			paths:   []string{"a"},
			wantErr: "invalid IP address or CIDR, please check",
		},
		{
			name:    "null value",
			content: `{"a":null}`,
			paths:   []string{"a"},
			wantErr: "invalid IP address or CIDR, please check",
		},
		{
			name:    "array containing a number",
			content: `{"a":["1.1.1.1",1]}`,
			paths:   []string{"a"},
			wantErr: "invalid IP address or CIDR, please check",
		},
		{
			name:    "invalid IP string",
			content: `{"a":"example.com"}`,
			paths:   []string{"a"},
			wantErr: "failed to process JSON: " + lib.ErrInvalidIP.Error(),
		},
		{
			name:    "IPv4-mapped CIDR with invalid prefix",
			content: `{"a":["::ffff:1.2.3.0/95"]}`,
			paths:   []string{"a"},
			wantErr: "failed to process JSON: " + lib.ErrInvalidCIDR.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ti := &text_in{Type: TypeJSONIn, JSONPath: tt.paths}
			entry := lib.NewEntry("test")
			err := ti.scanFileForJSONIn(strings.NewReader(tt.content), entry)
			assertErr(t, err, tt.wantErr, nil)
			if tt.wantErr != "" {
				return
			}
			if tt.wantNoPrefix {
				if cidrs, err := entry.MarshalText(); err == nil {
					t.Fatalf("expected no prefix, got %q", cidrs)
				}
				return
			}
			assertStrings(t, "CIDRs", entryCIDRs(t, entry), tt.want)
		})
	}
}

func TestTextIn_ProcessJSONResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		result    gjson.Result
		want      []string
		wantErr   string
		wantErrIs error
	}{
		{name: "not exist", result: gjson.Result{}, wantErr: "value not exist"},
		{name: "string", result: gjson.Parse(`"1.1.1.1"`), want: []string{"1.1.1.1/32"}},
		{name: "nested array", result: gjson.Parse(`["1.1.1.1",["2001:db8::/32"]]`), want: []string{"1.1.1.1/32", "2001:db8::/32"}},
		{name: "empty string", result: gjson.Parse(`""`), wantErr: "empty string"},
		{name: "invalid string", result: gjson.Parse(`"1.1.1.1/33"`), wantErrIs: lib.ErrInvalidCIDR},
		{name: "number", result: gjson.Parse(`1`), wantErr: "invalid IP address or CIDR, please check"},
		{name: "object", result: gjson.Parse(`{}`), wantErr: "invalid IP address or CIDR, please check"},
		{name: "true", result: gjson.Parse(`true`), wantErr: "invalid IP address or CIDR, please check"},
		{name: "null", result: gjson.Parse(`null`), wantErr: "invalid IP address or CIDR, please check"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ti := &text_in{Type: TypeJSONIn}
			entry := lib.NewEntry("test")
			err := ti.processJSONResult(tt.result, entry)
			assertErr(t, err, tt.wantErr, tt.wantErrIs)
			if tt.wantErr != "" || tt.wantErrIs != nil {
				return
			}
			assertStrings(t, "CIDRs", entryCIDRs(t, entry), tt.want)
		})
	}
}
