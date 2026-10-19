package plaintext

import (
	"errors"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/tidwall/gjson"
)

func TestNewTextIn(t *testing.T) {
	tests := []struct {
		name    string
		iType   string
		opts    []lib.InputOption
		wantErr string
		check   func(t *testing.T, in *text_in)
	}{
		{
			name:  "text with name and uri",
			iType: TypeTextIn,
			opts:  []lib.InputOption{WithNameAndURI("  cn ", "  ./cn.txt  ")},
			check: func(t *testing.T, in *text_in) {
				if in.Name != "cn" || in.URI != "./cn.txt" {
					t.Errorf("Name/URI = %q/%q", in.Name, in.URI)
				}
			},
		},
		{
			name:  "text with name and ipOrCIDR only",
			iType: TypeTextIn,
			opts: []lib.InputOption{
				WithNameAndURI("cn", ""),
				WithIPOrCIDR([]string{"1.0.0.0/24", "2001:db8::/32", "::ffff:1.2.3.4"}),
			},
			check: func(t *testing.T, in *text_in) {
				compareStrings(t, "IPOrCIDR", in.IPOrCIDR, []string{"1.0.0.0/24", "2001:db8::/32", "::ffff:1.2.3.4"})
			},
		},
		{
			name:  "nil option is ignored",
			iType: TypeTextIn,
			opts:  []lib.InputOption{nil, WithNameAndURI("cn", "a.txt")},
		},
		{
			name:  "all options are applied",
			iType: TypeJSONIn,
			opts: []lib.InputOption{
				WithInputDir("  /tmp/data  "),
				WithInputWantedList([]string{" cn ", "", "  ", "Us"}),
				WithInputOnlyIPType(lib.IPv6),
				WithJSONPath([]string{"a.b"}),
				WithRemovePrefixesInLine([]string{"p"}),
				WithRemoveSuffixesInLine([]string{"s"}),
			},
			check: func(t *testing.T, in *text_in) {
				if in.InputDir != "/tmp/data" {
					t.Errorf("InputDir = %q", in.InputDir)
				}
				if len(in.Want) != 2 || !in.Want["CN"] || !in.Want["US"] {
					t.Errorf("Want = %v", in.Want)
				}
				if in.OnlyIPType != lib.IPv6 {
					t.Errorf("OnlyIPType = %q", in.OnlyIPType)
				}
				compareStrings(t, "JSONPath", in.JSONPath, []string{"a.b"})
				compareStrings(t, "prefixes", in.RemovePrefixesInLine, []string{"p"})
				compareStrings(t, "suffixes", in.RemoveSuffixesInLine, []string{"s"})
			},
		},
		{
			name:    "ipOrCIDR not allowed for non-text type",
			iType:   TypeClashRuleSetIPCIDRIn,
			opts:    []lib.InputOption{WithNameAndURI("cn", "a"), WithIPOrCIDR([]string{"1.1.1.1"})},
			wantErr: "ipOrCIDR is invalid",
		},
		{
			name:    "json missing jsonPath",
			iType:   TypeJSONIn,
			opts:    []lib.InputOption{WithNameAndURI("cn", "a.json")},
			wantErr: "missing jsonPath",
		},
		{
			name:    "missing inputDir and name",
			iType:   TypeTextIn,
			wantErr: "missing inputDir or name",
		},
		{
			name:    "name without uri or ipOrCIDR",
			iType:   TypeSurgeRuleSetIn,
			opts:    []lib.InputOption{WithNameAndURI("cn", "   ")},
			wantErr: "missing uri or ipOrCIDR",
		},
		{
			name:    "inputDir with name",
			iType:   TypeTextIn,
			opts:    []lib.InputOption{WithInputDir("dir"), WithNameAndURI("cn", "")},
			wantErr: "inputDir is not allowed",
		},
		{
			name:    "inputDir with uri",
			iType:   TypeTextIn,
			opts:    []lib.InputOption{WithInputDir("dir"), WithNameAndURI("", "a.txt")},
			wantErr: "inputDir is not allowed",
		},
		{
			name:    "inputDir with ipOrCIDR",
			iType:   TypeTextIn,
			opts:    []lib.InputOption{WithInputDir("dir"), WithIPOrCIDR([]string{"1.1.1.1"})},
			wantErr: "inputDir is not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := newTextIn(tt.iType, "desc", lib.ActionAdd, tt.opts...)
			if checkErr(t, err, tt.wantErr) {
				if conv != nil {
					t.Errorf("converter should be nil on error")
				}
				return
			}
			in := conv.(*text_in)
			if in.Type != tt.iType || in.Description != "desc" || in.Action != lib.ActionAdd {
				t.Errorf("Type/Desc/Action = %q/%q/%q", in.Type, in.Description, in.Action)
			}
			if tt.check != nil {
				tt.check(t, in)
			}
		})
	}
}

func TestNewTextInFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		iType   string
		data    string
		wantErr string
		check   func(t *testing.T, in *text_in)
	}{
		{
			name:    "empty data",
			iType:   TypeTextIn,
			data:    "",
			wantErr: "missing inputDir or name",
		},
		{
			name:    "invalid json",
			iType:   TypeTextIn,
			data:    "{",
			wantErr: "unexpected end of JSON input",
		},
		{
			name:    "wrong field type",
			iType:   TypeTextIn,
			data:    `{"name": 1}`,
			wantErr: "cannot unmarshal",
		},
		{
			name:  "full text config",
			iType: TypeTextIn,
			data: `{"name":" cn ","uri":"http://example.com/a.txt","ipOrCIDR":["1.1.1.1","::ffff:2.2.2.2"],
				"wantedList":["cn"],"onlyIPType":"ipv4",
				"removePrefixesInLine":["a"],"removeSuffixesInLine":["b"]}`,
			check: func(t *testing.T, in *text_in) {
				if in.Name != "cn" || in.URI != "http://example.com/a.txt" {
					t.Errorf("Name/URI = %q/%q", in.Name, in.URI)
				}
				compareStrings(t, "IPOrCIDR", in.IPOrCIDR, []string{"1.1.1.1", "::ffff:2.2.2.2"})
				if !in.Want["CN"] || len(in.Want) != 1 {
					t.Errorf("Want = %v", in.Want)
				}
				if in.OnlyIPType != lib.IPv4 {
					t.Errorf("OnlyIPType = %q", in.OnlyIPType)
				}
				compareStrings(t, "prefixes", in.RemovePrefixesInLine, []string{"a"})
				compareStrings(t, "suffixes", in.RemoveSuffixesInLine, []string{"b"})
			},
		},
		{
			name:  "json config with inputDir",
			iType: TypeJSONIn,
			data:  `{"inputDir":"./data","jsonPath":["x","y"]}`,
			check: func(t *testing.T, in *text_in) {
				if in.InputDir != "./data" {
					t.Errorf("InputDir = %q", in.InputDir)
				}
				compareStrings(t, "JSONPath", in.JSONPath, []string{"x", "y"})
			},
		},
		{
			name:    "json config missing jsonPath",
			iType:   TypeJSONIn,
			data:    `{"inputDir":"./data"}`,
			wantErr: "missing jsonPath",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := newTextInFromBytes(tt.iType, "desc", lib.ActionAdd, []byte(tt.data))
			if checkErr(t, err, tt.wantErr) {
				return
			}
			if tt.check != nil {
				tt.check(t, conv.(*text_in))
			}
		})
	}
}

func TestTextInScanFile(t *testing.T) {
	longLine := strings.Repeat("1", 70*1024) // exceeds bufio.Scanner default max token size

	tests := []struct {
		name     string
		iType    string
		jsonPath []string
		prefixes []string
		suffixes []string
		content  string
		useErrRd bool
		want     []string
		wantErr  string
	}{
		// ---------- text ----------
		{
			name:  "text: ipv4, ipv6, ipv4-mapped ipv6, comments, merging",
			iType: TypeTextIn,
			content: "1.0.0.0/24\n" +
				"1.0.1.0/24 # trailing comment\n" +
				"# full line comment\n" +
				"// another comment\n" +
				"/* block comment */\n" +
				"\n" +
				"   \n" +
				"2001:DB8::/32\n" +
				"2001:db8:1::1 // inline\n" +
				"::ffff:10.0.0.0/104\n" +
				"::FFFF:192.168.1.1\n" +
				"8.8.8.8\n" +
				"1.2.3.4/24\n",
			want: []string{"1.0.0.0/23", "1.2.3.0/24", "8.8.8.8/32", "10.0.0.0/8", "192.168.1.1/32", "2001:db8::/32"},
		},
		{
			name:     "text: remove prefixes and suffixes",
			iType:    TypeTextIn,
			prefixes: []string{" IP-CIDR, ", "IP-CIDR6,", ""},
			suffixes: []string{",NO-RESOLVE", ""},
			content: "IP-CIDR,1.1.1.1/32,no-resolve\n" +
				"ip-cidr6,2606:4700::/32\n" +
				"IP-CIDR,::ffff:9.9.9.0/120\n" +
				"IP-CIDR,,no-resolve\n",
			want: []string{"1.1.1.1/32", "9.9.9.0/24", "2606:4700::/32"},
		},
		{
			name:    "text: only comments yields nothing",
			iType:   TypeTextIn,
			content: "# nothing\n// here\n",
			want:    nil,
		},
		{
			name:    "text: invalid ip",
			iType:   TypeTextIn,
			content: "1.1.1.1\nnot-an-ip\n",
			wantErr: "invalid",
		},
		{
			name:    "text: invalid ipv4-mapped ipv6 cidr",
			iType:   TypeTextIn,
			content: "::ffff:1.2.3.0/64\n",
			wantErr: "invalid",
		},
		{
			name:    "text: scanner error on overlong line",
			iType:   TypeTextIn,
			content: longLine,
			wantErr: "token too long",
		},

		// ---------- surge ----------
		{
			name:  "surge: ip-cidr and ip-cidr6 lines",
			iType: TypeSurgeRuleSetIn,
			content: "IP-CIDR,1.0.0.0/24,no-resolve\n" +
				"IP-CIDR6,2001:DB8::/32\n" +
				"DOMAIN,example.com\n" +
				"# comment\n" +
				"\n" +
				"IP-CIDR,::ffff:1.0.1.0/120 // comment\n" +
				"ip-cidr6,::ffff:8.8.8.8\n" +
				"IP-CIDR,\n" +
				"IP-CIDR, ,no-resolve\n",
			want: []string{"1.0.0.0/23", "8.8.8.8/32", "2001:db8::/32"},
		},
		{
			name:    "surge: invalid cidr",
			iType:   TypeSurgeRuleSetIn,
			content: "IP-CIDR,999.0.0.0/8\n",
			wantErr: "invalid",
		},
		{
			name:    "surge: scanner error",
			iType:   TypeSurgeRuleSetIn,
			content: longLine,
			wantErr: "token too long",
		},

		// ---------- clash ipcidr ----------
		{
			name:  "clash ipcidr: valid payload",
			iType: TypeClashRuleSetIPCIDRIn,
			content: "payload:\n" +
				"  - '1.0.0.0/24'\n" +
				"  - ''\n" +
				"  - '  '\n" +
				"  - '2001:db8::/32'\n" +
				"  - '::ffff:1.0.1.0/120'\n" +
				"  - '2001:db8:ffff::1'\n",
			want: []string{"1.0.0.0/23", "2001:db8::/32"},
		},
		{
			name:    "clash ipcidr: empty payload",
			iType:   TypeClashRuleSetIPCIDRIn,
			content: "payload: []\n",
			want:    nil,
		},
		{
			name:    "clash ipcidr: invalid yaml",
			iType:   TypeClashRuleSetIPCIDRIn,
			content: "payload: [",
			wantErr: "yaml",
		},
		{
			name:    "clash ipcidr: invalid cidr",
			iType:   TypeClashRuleSetIPCIDRIn,
			content: "payload:\n  - 'abc'\n",
			wantErr: "invalid",
		},
		{
			name:     "clash ipcidr: read error",
			iType:    TypeClashRuleSetIPCIDRIn,
			useErrRd: true,
			wantErr:  "forced read error",
		},

		// ---------- clash classical ----------
		{
			name:  "clash classical: valid payload",
			iType: TypeClashRuleSetClassicalIn,
			content: "payload:\n" +
				"  - IP-CIDR,1.0.0.0/24,no-resolve\n" +
				"  - IP-CIDR6,2001:DB8::/32\n" +
				"  - DOMAIN,example.com\n" +
				"  - ''\n" +
				"  - 'IP-CIDR,'\n" +
				"  - IP-CIDR,::ffff:1.0.1.0/120\n" +
				"  - ip-cidr,8.8.8.8\n",
			want: []string{"1.0.0.0/23", "8.8.8.8/32", "2001:db8::/32"},
		},
		{
			name:    "clash classical: invalid yaml",
			iType:   TypeClashRuleSetClassicalIn,
			content: "payload: [",
			wantErr: "yaml",
		},
		{
			name:    "clash classical: invalid cidr",
			iType:   TypeClashRuleSetClassicalIn,
			content: "payload:\n  - IP-CIDR,1.1.1.1/33\n",
			wantErr: "invalid",
		},

		// ---------- json ----------
		{
			name:     "json: strings, arrays, nested arrays, ipv4-mapped",
			iType:    TypeJSONIn,
			jsonPath: []string{"v4", "  v6.list  ", "mapped", "nested"},
			content: `{"v4":["1.0.0.0/24"," 1.0.1.0/24 "],` +
				`"v6":{"list":["2001:DB8::/32"]},` +
				`"mapped":"::ffff:8.8.8.8",` +
				`"nested":[["9.9.9.9"],["::ffff:10.0.0.0/104"]]}`,
			want: []string{"1.0.0.0/23", "8.8.8.8/32", "9.9.9.9/32", "10.0.0.0/8", "2001:db8::/32"},
		},
		{
			name:     "json: gjson #-query syntax",
			iType:    TypeJSONIn,
			jsonPath: []string{"prefixes.#.ip_prefix", "ipv6_prefixes.#.ipv6_prefix"},
			content: `{"prefixes":[{"ip_prefix":"3.0.0.0/8"},{"ip_prefix":"4.0.0.0/8"}],` +
				`"ipv6_prefixes":[{"ipv6_prefix":"2600:1f00::/24"}]}`,
			want: []string{"3.0.0.0/8", "4.0.0.0/8", "2600:1f00::/24"},
		},
		{
			name:     "json: invalid json",
			iType:    TypeJSONIn,
			jsonPath: []string{"a"},
			content:  `{"a":`,
			wantErr:  "invalid JSON data",
		},
		{
			name:     "json: path not exist",
			iType:    TypeJSONIn,
			jsonPath: []string{"missing"},
			content:  `{"a":"1.1.1.1"}`,
			wantErr:  "value not exist",
		},
		{
			name:     "json: empty string",
			iType:    TypeJSONIn,
			jsonPath: []string{"a"},
			content:  `{"a":"   "}`,
			wantErr:  "empty string",
		},
		{
			name:     "json: number value",
			iType:    TypeJSONIn,
			jsonPath: []string{"a"},
			content:  `{"a":123}`,
			wantErr:  "please check your specified JSON path",
		},
		{
			name:     "json: array containing non-string",
			iType:    TypeJSONIn,
			jsonPath: []string{"a"},
			content:  `{"a":["1.1.1.1", {"x":1}]}`,
			wantErr:  "failed to process JSON",
		},
		{
			name:     "json: invalid ip string",
			iType:    TypeJSONIn,
			jsonPath: []string{"a"},
			content:  `{"a":"1.1.1.300"}`,
			wantErr:  "invalid",
		},
		{
			name:     "json: read error",
			iType:    TypeJSONIn,
			jsonPath: []string{"a"},
			useErrRd: true,
			wantErr:  "forced read error",
		},

		// ---------- unsupported ----------
		{
			name:    "unsupported type",
			iType:   "unknown",
			content: "1.1.1.1",
			wantErr: lib.ErrNotSupportedFormat.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := &text_in{
				Type:                 tt.iType,
				JSONPath:             tt.jsonPath,
				RemovePrefixesInLine: tt.prefixes,
				RemoveSuffixesInLine: tt.suffixes,
			}
			entry := lib.NewEntry("test")

			var err error
			if tt.useErrRd {
				err = in.scanFile(errReader{}, entry)
			} else {
				err = in.scanFile(strings.NewReader(tt.content), entry)
			}
			if checkErr(t, err, tt.wantErr) {
				return
			}
			compareStrings(t, "entry", entryText(entry), tt.want)
		})
	}
}

func TestTextInScanFileUnsupportedIsSentinel(t *testing.T) {
	in := &text_in{Type: "unknown"}
	err := in.scanFile(strings.NewReader(""), lib.NewEntry("x"))
	if !errors.Is(err, lib.ErrNotSupportedFormat) {
		t.Fatalf("err = %v, want ErrNotSupportedFormat", err)
	}
}

func TestTextInProcessJSONResult(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		path    string
		want    []string
		wantErr string
	}{
		{name: "ipv4 string", json: `{"a":"1.1.1.0/24"}`, path: "a", want: []string{"1.1.1.0/24"}},
		{name: "ipv6 string", json: `{"a":"2001:db8::1"}`, path: "a", want: []string{"2001:db8::1/128"}},
		{name: "ipv4-mapped string", json: `{"a":"::ffff:1.1.1.1"}`, path: "a", want: []string{"1.1.1.1/32"}},
		{name: "empty array", json: `{"a":[]}`, path: "a", want: nil},
		{name: "not exist", json: `{"a":1}`, path: "b", wantErr: "value not exist"},
		{name: "bool", json: `{"a":true}`, path: "a", wantErr: "invalid IP address or CIDR"},
		{name: "null", json: `{"a":null}`, path: "a", wantErr: "invalid IP address or CIDR"},
		{name: "object", json: `{"a":{"b":"1.1.1.1"}}`, path: "a", wantErr: "invalid IP address or CIDR"},
		{name: "empty string in array", json: `{"a":["1.1.1.1",""]}`, path: "a", wantErr: "empty string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := &text_in{Type: TypeJSONIn}
			entry := lib.NewEntry("x")
			err := in.processJSONResult(gjson.Get(tt.json, tt.path), entry)
			if checkErr(t, err, tt.wantErr) {
				return
			}
			compareStrings(t, "entry", entryText(entry), tt.want)
		})
	}
}
