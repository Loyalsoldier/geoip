package singbox

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

var (
	// cnInput holds unaligned prefixes that merge into one prefix per family.
	cnInput = []string{"192.168.0.3/24", "192.168.1.5/24", "2001:db8::/33", "2001:db8:8000::/33"}
	cnCIDRs = []string{"192.168.0.0/23", "2001:db8::/32"}

	usInput = []string{"10.0.0.0/8", "11.0.0.0/8", "2400:cb00::/32"}
	usCIDRs = []string{"10.0.0.0/7", "2400:cb00::/32"}

	v4Input = []string{"1.1.1.1/24"}
	v4CIDRs = []string{"1.1.1.0/24"}
)

func TestNewSRSIn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		opts         []lib.InputOption
		wantErr      string
		wantName     string
		wantURI      string
		wantInputDir string
	}{
		{name: "no option", wantErr: "missing name or uri or inputDir"},
		{name: "nil option is ignored", opts: []lib.InputOption{nil}, wantErr: "missing name or uri or inputDir"},
		{name: "blank values", opts: []lib.InputOption{WithNameAndURI(" ", "\t"), WithInputDir("  ")}, wantErr: "missing name or uri or inputDir"},
		{name: "name without uri", opts: []lib.InputOption{WithNameAndURI("cn", "")}, wantErr: "name and uri must be specified together"},
		{name: "uri without name", opts: []lib.InputOption{WithNameAndURI("", "cn.srs")}, wantErr: "name and uri must be specified together"},
		{name: "input dir with name only", opts: []lib.InputOption{WithInputDir("dir"), WithNameAndURI("cn", " ")}, wantErr: "name and uri must be specified together"},
		{
			name:     "name and uri are trimmed",
			opts:     []lib.InputOption{nil, WithNameAndURI(" cn ", " ./cn.srs\n")},
			wantName: "cn",
			wantURI:  "./cn.srs",
		},
		{
			name:         "input dir is trimmed",
			opts:         []lib.InputOption{WithInputDir("  ./srs  ")},
			wantInputDir: "./srs",
		},
		{
			name:         "input dir with name and uri",
			opts:         []lib.InputOption{WithInputDir("dir"), WithNameAndURI("cn", "cn.srs")},
			wantName:     "cn",
			wantURI:      "cn.srs",
			wantInputDir: "dir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewSRSIn(lib.ActionAdd, tt.opts...)
			if tt.wantErr != "" {
				assertErrorContains(t, err, "[type singboxSRS | action add]", tt.wantErr)
				if conv != nil {
					t.Errorf("got converter %v, want nil", conv)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			in := conv.(*srs_in)
			if in.Name != tt.wantName || in.URI != tt.wantURI || in.InputDir != tt.wantInputDir {
				t.Errorf("got name %q uri %q inputDir %q, want %q %q %q", in.Name, in.URI, in.InputDir, tt.wantName, tt.wantURI, tt.wantInputDir)
			}
			if got := conv.GetType(); got != TypeSRSIn {
				t.Errorf("GetType: got %q, want %q", got, TypeSRSIn)
			}
			if got := conv.GetAction(); got != lib.ActionAdd {
				t.Errorf("GetAction: got %q, want %q", got, lib.ActionAdd)
			}
			if got := conv.GetDescription(); got != DescSRSIn {
				t.Errorf("GetDescription: got %q, want %q", got, DescSRSIn)
			}
		})
	}
}

func TestSRSIn_WithInputWantedList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		lists []string
		want  map[string]bool
	}{
		{name: "nil list", lists: nil, want: map[string]bool{}},
		{name: "blank items are dropped", lists: []string{"", " ", "\n"}, want: map[string]bool{}},
		{name: "items are trimmed and upper-cased", lists: []string{" cn ", "Us", "CN"}, want: map[string]bool{"CN": true, "US": true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			in := &srs_in{}
			WithInputWantedList(tt.lists)(in)
			if len(in.Want) != len(tt.want) || !maps.Equal(in.Want, tt.want) {
				t.Errorf("got %v, want %v", in.Want, tt.want)
			}
		})
	}
}

func TestSRSIn_WithInputOnlyIPType(t *testing.T) {
	t.Parallel()

	for _, ipType := range []lib.IPType{"", lib.IPv4, lib.IPv6} {
		in := &srs_in{}
		WithInputOnlyIPType(ipType)(in)
		if in.OnlyIPType != ipType {
			t.Errorf("got %q, want %q", in.OnlyIPType, ipType)
		}
	}
}

func TestNewSRSInFromBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		data      string
		wantErr   string
		wantErrAs any
		want      srs_in
	}{
		{name: "invalid json", data: "{", wantErr: "[type singboxSRS | action add]", wantErrAs: new(*json.SyntaxError)},
		{name: "wrong field type", data: `{"wantedList": "cn"}`, wantErr: "[type singboxSRS | action add]", wantErrAs: new(*json.UnmarshalTypeError)},
		{name: "empty data", data: "", wantErr: "missing name or uri or inputDir"},
		{name: "uri only", data: `{"uri": "cn.srs"}`, wantErr: "name and uri must be specified together"},
		{
			name: "name and uri",
			data: `{"name": " cn ", "uri": " https://example.com/cn.srs ", "wantedList": ["cn", ""], "onlyIPType": "ipv4"}`,
			want: srs_in{Name: "cn", URI: "https://example.com/cn.srs", Want: map[string]bool{"CN": true}, OnlyIPType: lib.IPv4},
		},
		{
			name: "input dir",
			data: `{"inputDir": " ./srs ", "onlyIPType": "ipv6"}`,
			want: srs_in{InputDir: "./srs", Want: map[string]bool{}, OnlyIPType: lib.IPv6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewSRSInFromBytes(lib.ActionAdd, []byte(tt.data))
			if tt.wantErr != "" {
				assertErrorContains(t, err, tt.wantErr)
				if tt.wantErrAs != nil && !errors.As(err, tt.wantErrAs) {
					t.Errorf("error %v is not %T", err, tt.wantErrAs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			in := conv.(*srs_in)
			if in.Name != tt.want.Name || in.URI != tt.want.URI || in.InputDir != tt.want.InputDir {
				t.Errorf("got name %q uri %q inputDir %q, want %q %q %q", in.Name, in.URI, in.InputDir, tt.want.Name, tt.want.URI, tt.want.InputDir)
			}
			if len(in.Want) != len(tt.want.Want) || !maps.Equal(in.Want, tt.want.Want) {
				t.Errorf("Want: got %v, want %v", in.Want, tt.want.Want)
			}
			if in.OnlyIPType != tt.want.OnlyIPType {
				t.Errorf("OnlyIPType: got %q, want %q", in.OnlyIPType, tt.want.OnlyIPType)
			}
		})
	}
}

func TestSRSIn_generateEntries_Data(t *testing.T) {
	t.Parallel()

	v4Item := rawIPSetItem([2]string{"1.0.0.0", "1.0.0.255"})
	validRule := rawDefaultRule(v4Item)
	valid := srsBytes(t, ipcidrRule(cnInput...))

	tests := []struct {
		name       string
		data       []byte
		want       []string
		wantErr    string
		wantErrIs  error
		wantAnyErr bool
	}{
		// Well-formed rule-sets.
		{name: "prefixes are merged", data: valid, want: cnCIDRs},
		{name: "rule-set version 1", data: srsBytesVersion(t, constant.RuleSetVersion1, ipcidrRule(usInput...)), want: usCIDRs},
		{name: "latest rule-set version", data: srsBytesVersion(t, constant.RuleSetVersionCurrent, ipcidrRule(usInput...)), want: usCIDRs},
		{name: "bare addresses", data: srsBytes(t, ipcidrRule("1.1.1.1", "2001:db8::1")), want: []string{"1.1.1.1/32", "2001:db8::1/128"}},
		{
			name: "prefixes of several rules are merged",
			data: srsBytes(t, ipcidrRule("192.168.0.0/24", "2001:db8::/33"), ipcidrRule("192.168.1.0/24", "2001:db8:8000::/33")),
			want: cnCIDRs,
		},
		{
			name: "unaligned range from raw ip set is split",
			data: rawSRS(t, 1, rawRuleSet(rawDefaultRule(rawIPSetItem([2]string{"1.1.1.1", "1.1.1.10"})))),
			want: []string{"1.1.1.1/32", "1.1.1.2/31", "1.1.1.4/30", "1.1.1.8/31", "1.1.1.10/32"},
		},
		{name: "raw ipv4 range", data: rawSRS(t, 1, rawRuleSet(validRule)), want: []string{"1.0.0.0/24"}},
		{name: "no rule", data: rawSRS(t, 1, rawRuleSet()), want: nil},
		{name: "trailing data after rules is ignored", data: rawSRS(t, 1, join(rawRuleSet(validRule), []byte("trailing"))), want: []string{"1.0.0.0/24"}},
		// Rules or items ignored by the converter.
		{name: "domain rule", data: srsBytes(t, option.HeadlessRule{Type: constant.RuleTypeDefault, DefaultOptions: option.DefaultHeadlessRule{Domain: []string{"example.com"}}}), want: nil},
		{name: "source ip cidr is ignored", data: srsBytes(t, option.HeadlessRule{Type: constant.RuleTypeDefault, DefaultOptions: option.DefaultHeadlessRule{SourceIPCIDR: []string{"10.0.0.0/8"}}}), want: nil},
		{name: "ip cidr nested in logical rule is ignored", data: srsBytes(t, logicalRule(ipcidrRule("10.0.0.0/8"), ipcidrRule("2001:db8::/32"))), want: nil},
		{name: "only default rules are read", data: srsBytes(t, logicalRule(ipcidrRule("10.0.0.0/8")), ipcidrRule(v4Input...)), want: v4CIDRs},
		// Invalid ranges of the IP set are dropped without error.
		{name: "inverted range is dropped", data: rawSRS(t, 1, rawRuleSet(rawDefaultRule(rawIPSetItem([2]string{"1.0.0.255", "1.0.0.0"})))), want: nil},
		{name: "mixed family range is dropped", data: rawSRS(t, 1, rawRuleSet(rawDefaultRule(rawIPSetItem([2]string{"1.0.0.0", "2001:db8::"})))), want: nil},
		{
			name: "invalid ranges are dropped but valid ones kept",
			data: rawSRS(t, 1, rawRuleSet(rawDefaultRule(rawIPSetItem([2]string{"1.0.0.255", "1.0.0.0"}, [2]string{"2001:db8::", "2001:db8::ffff"})))),
			want: []string{"2001:db8::/112"},
		},
		// Malformed data.
		{name: "empty data", data: nil, wantErrIs: io.EOF},
		{name: "truncated magic", data: []byte("SR"), wantErrIs: io.ErrUnexpectedEOF},
		{name: "bad magic", data: append([]byte("XYZ"), valid[3:]...), wantErr: "invalid sing-box rule-set file"},
		{name: "missing version", data: []byte("SRS"), wantErrIs: io.EOF},
		{name: "unsupported version", data: rawSRS(t, constant.RuleSetVersionCurrent+1, rawRuleSet(validRule)), wantErr: "unsupported version"},
		{name: "missing compressed data", data: []byte("SRS\x01"), wantErrIs: io.ErrUnexpectedEOF},
		{name: "not zlib data", data: []byte("SRS\x01garbage"), wantErrIs: zlib.ErrHeader},
		{name: "truncated compressed data", data: valid[:len(valid)/2], wantAnyErr: true},
		{name: "missing rule count", data: rawSRS(t, 1, nil), wantErrIs: io.EOF},
		{name: "missing rule", data: rawSRS(t, 1, uvarint(1)), wantErr: "read rule[0]", wantErrIs: io.EOF},
		{name: "fewer rules than count", data: rawSRS(t, 1, join(uvarint(2), validRule)), wantErr: "read rule[1]", wantErrIs: io.EOF},
		{name: "unknown rule type", data: rawSRS(t, 1, join(uvarint(1), []byte{2})), wantErr: "unknown rule type: 2"},
		{name: "unknown logical mode", data: rawSRS(t, 1, join(uvarint(1), []byte{1, 2})), wantErr: "unknown logical mode: 2"},
		{name: "unknown rule item", data: rawSRS(t, 1, join(uvarint(1), []byte{0, 0xFE})), wantErr: "unknown rule item type: 254"},
		{name: "missing final item", data: rawSRS(t, 1, join(uvarint(1), []byte{0}, v4Item)), wantErrIs: io.EOF},
		{name: "missing invert flag", data: rawSRS(t, 1, join(uvarint(1), []byte{0}, v4Item, []byte{0xFF})), wantErrIs: io.EOF},
		{name: "missing ip set version", data: rawSRS(t, 1, join(uvarint(1), []byte{0, 6})), wantErrIs: io.EOF},
		{name: "unsupported ip set version", data: rawSRS(t, 1, join(uvarint(1), []byte{0, 6, 2}, v4Item[2:])), wantErrIs: os.ErrInvalid},
		{name: "truncated range count", data: rawSRS(t, 1, join(uvarint(1), []byte{0, 6, 1, 0, 0})), wantErrIs: io.ErrUnexpectedEOF},
		{name: "missing range", data: rawSRS(t, 1, join(uvarint(1), []byte{0, 6, 1}, binary.BigEndian.AppendUint64(nil, 1))), wantErrIs: io.EOF},
		{name: "invalid address length", data: rawSRS(t, 1, join(uvarint(1), []byte{0, 6, 1}, binary.BigEndian.AppendUint64(nil, 1), uvarint(5), []byte{1, 2, 3, 4, 5})), wantErrIs: os.ErrInvalid},
		{name: "zero address length", data: rawSRS(t, 1, join(uvarint(1), []byte{0, 6, 1}, binary.BigEndian.AppendUint64(nil, 1), uvarint(0))), wantErrIs: os.ErrInvalid},
		{name: "truncated address", data: rawSRS(t, 1, join(uvarint(1), []byte{0, 6, 1}, binary.BigEndian.AppendUint64(nil, 1), uvarint(4), []byte{1, 2})), wantErrIs: io.ErrUnexpectedEOF},
		{name: "missing range end", data: rawSRS(t, 1, join(uvarint(1), []byte{0, 6, 1}, binary.BigEndian.AppendUint64(nil, 1), rawAddr("1.0.0.0"))), wantErrIs: io.EOF},
		{name: "fewer ranges than count", data: rawSRS(t, 1, join(uvarint(1), []byte{0, 6, 1}, binary.BigEndian.AppendUint64(nil, 2), v4Item[10:])), wantErrIs: io.EOF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entries := make(map[string]*lib.Entry)
			err := (&srs_in{}).generateEntries("cn", bytes.NewReader(tt.data), entries)
			if tt.wantErr != "" || tt.wantErrIs != nil || tt.wantAnyErr {
				if err == nil {
					t.Fatal("got nil error, want error")
				}
				if tt.wantErr != "" {
					assertErrorContains(t, err, tt.wantErr)
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("error %v is not %v", err, tt.wantErrIs)
				}
				if len(entries) != 0 {
					t.Errorf("got %d entries, want 0", len(entries))
				}
				return
			}
			if err != nil {
				t.Fatalf("generateEntries: %v", err)
			}
			if len(entries) != 1 {
				t.Fatalf("got %d entries, want 1", len(entries))
			}
			entry, found := entries["CN"]
			if !found {
				t.Fatalf("entry CN not found in %v", entries)
			}
			if entry.GetName() != "CN" {
				t.Errorf("entry name: got %q, want %q", entry.GetName(), "CN")
			}
			assertStrings(t, "CIDRs", entryText(entry), tt.want)
		})
	}
}

// TestSRSIn_generateEntries_IPv4Mapped checks IP sets holding IPv4-mapped
// IPv6 ranges. sing-box keeps the 16-byte form, so the converter receives
// strings such as "::ffff:1.2.3.0/120", which must become IPv4 prefixes.
func TestSRSIn_generateEntries_IPv4Mapped(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		ranges [][2]string
		// cidrs are the strings sing-box recovers from the IP set.
		cidrs []string
		want  []string
	}{
		{
			name:   "mapped /120",
			ranges: [][2]string{{"::ffff:1.2.3.0", "::ffff:1.2.3.255"}},
			cidrs:  []string{"::ffff:1.2.3.0/120"},
			want:   []string{"1.2.3.0/24"},
		},
		{
			name:   "mapped /128",
			ranges: [][2]string{{"::ffff:1.2.3.4", "::ffff:1.2.3.4"}},
			cidrs:  []string{"::ffff:1.2.3.4/128"},
			want:   []string{"1.2.3.4/32"},
		},
		{
			name:   "mapped /104",
			ranges: [][2]string{{"::ffff:192.0.0.0", "::ffff:192.255.255.255"}},
			cidrs:  []string{"::ffff:192.0.0.0/104"},
			want:   []string{"192.0.0.0/8"},
		},
		{
			name:   "whole mapped space",
			ranges: [][2]string{{"::ffff:0.0.0.0", "::ffff:255.255.255.255"}},
			cidrs:  []string{"::ffff:0.0.0.0/96"},
			want:   []string{"0.0.0.0/0"},
		},
		{
			name:   "unaligned mapped range",
			ranges: [][2]string{{"::ffff:1.1.1.1", "::ffff:1.1.1.4"}},
			cidrs:  []string{"::ffff:1.1.1.1/128", "::ffff:1.1.1.2/127", "::ffff:1.1.1.4/128"},
			want:   []string{"1.1.1.1/32", "1.1.1.2/31", "1.1.1.4/32"},
		},
		{
			name:   "mapped ranges merged with ipv4 and kept apart from ipv6",
			ranges: [][2]string{{"10.0.0.0", "10.0.0.255"}, {"::ffff:10.0.1.0", "::ffff:10.0.1.255"}, {"2001:db8::", "2001:db8::ff"}},
			cidrs:  []string{"10.0.0.0/24", "::ffff:10.0.1.0/120", "2001:db8::/120"},
			want:   []string{"10.0.0.0/23", "2001:db8::/120"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := rawSRS(t, 1, rawRuleSet(rawDefaultRule(rawIPSetItem(tt.ranges...))))
			assertStrings(t, "recovered CIDR strings", []string(readSRS(t, data).Options.Rules[0].DefaultOptions.IPCIDR), tt.cidrs)

			entries := make(map[string]*lib.Entry)
			if err := (&srs_in{}).generateEntries("cn", bytes.NewReader(data), entries); err != nil {
				t.Fatalf("generateEntries: %v", err)
			}
			if len(entries) != 1 || entries["CN"] == nil {
				t.Fatalf("got entries %v, want CN only", entries)
			}
			assertStrings(t, "CIDRs", entryText(entries["CN"]), tt.want)

			// Through Input, only ipv4 must keep every mapped prefix.
			path := writeFile(t, filepath.Join(t.TempDir(), "cn.srs"), data)
			conv, err := NewSRSIn(lib.ActionAdd, WithNameAndURI("cn", path), WithInputOnlyIPType(lib.IPv4))
			if err != nil {
				t.Fatalf("NewSRSIn: %v", err)
			}
			got, err := conv.Input(lib.NewContainer())
			if err != nil {
				t.Fatalf("Input: %v", err)
			}
			var wantV4 []string
			for _, cidr := range tt.want {
				if !strings.Contains(cidr, ":") {
					wantV4 = append(wantV4, cidr)
				}
			}
			assertContainer(t, got, map[string][]string{"CN": wantV4})
		})
	}
}

func TestSRSIn_generateEntries(t *testing.T) {
	t.Parallel()

	t.Run("read error", func(t *testing.T) {
		t.Parallel()

		entries := make(map[string]*lib.Entry)
		err := (&srs_in{}).generateEntries("cn", errReader{}, entries)
		assertErrorContains(t, err, "forced read failure")
		if len(entries) != 0 {
			t.Errorf("got %d entries, want 0", len(entries))
		}
	})

	t.Run("unwanted list is skipped without reading", func(t *testing.T) {
		t.Parallel()

		in := &srs_in{}
		WithInputWantedList([]string{"us"})(in)
		entries := make(map[string]*lib.Entry)
		if err := in.generateEntries("cn", errReader{}, entries); err != nil {
			t.Fatalf("generateEntries: %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("got %d entries, want 0", len(entries))
		}
	})

	t.Run("wanted list matches upper-cased name", func(t *testing.T) {
		t.Parallel()

		in := &srs_in{}
		WithInputWantedList([]string{"CN"})(in)
		entries := make(map[string]*lib.Entry)
		if err := in.generateEntries("cn", bytes.NewReader(srsBytes(t, ipcidrRule(cnInput...))), entries); err != nil {
			t.Fatalf("generateEntries: %v", err)
		}
		if len(entries) != 1 || entries["CN"] == nil {
			t.Fatalf("got entries %v, want CN only", entries)
		}
		assertStrings(t, "CIDRs of CN", entryText(entries["CN"]), cnCIDRs)
	})

	t.Run("merged into existing entry", func(t *testing.T) {
		t.Parallel()

		existing := lib.NewEntry("CN")
		if err := existing.AddPrefix("192.168.0.0/24"); err != nil {
			t.Fatal(err)
		}
		entries := map[string]*lib.Entry{"CN": existing}
		data := srsBytes(t, ipcidrRule("192.168.1.0/24", "2001:db8::/32"))
		if err := (&srs_in{}).generateEntries("cn", bytes.NewReader(data), entries); err != nil {
			t.Fatalf("generateEntries: %v", err)
		}
		if len(entries) != 1 || entries["CN"] != existing {
			t.Fatalf("got entries %v, want the existing CN entry only", entries)
		}
		assertStrings(t, "CIDRs of CN", entryText(existing), cnCIDRs)
	})
}

func TestSRSIn_Input_NameAndURI(t *testing.T) {
	t.Parallel()

	path := writeSRS(t, filepath.Join(t.TempDir(), "file.srs"), cnInput...)

	tests := []struct {
		name      string
		action    lib.Action
		listName  string
		opts      []lib.InputOption
		want      map[string][]string
		wantErr   string
		wantErrIs error
	}{
		{name: "add", action: lib.ActionAdd, listName: " cn ", want: map[string][]string{"CN": cnCIDRs}},
		{name: "list name is not derived from file name", action: lib.ActionAdd, listName: "my-list", want: map[string][]string{"MY-LIST": cnCIDRs}},
		{name: "only ipv4", action: lib.ActionAdd, listName: "cn", opts: []lib.InputOption{WithInputOnlyIPType(lib.IPv4)}, want: map[string][]string{"CN": {"192.168.0.0/23"}}},
		{name: "only ipv6", action: lib.ActionAdd, listName: "cn", opts: []lib.InputOption{WithInputOnlyIPType(lib.IPv6)}, want: map[string][]string{"CN": {"2001:db8::/32"}}},
		{name: "wanted list contains name", action: lib.ActionAdd, listName: "cn", opts: []lib.InputOption{WithInputWantedList([]string{"CN"})}, want: map[string][]string{"CN": cnCIDRs}},
		{name: "wanted list misses name", action: lib.ActionAdd, listName: "cn", opts: []lib.InputOption{WithInputWantedList([]string{"us"})}, wantErr: "no entry is generated"},
		{name: "unknown action", action: lib.Action("unknown"), listName: "cn", wantErr: "[type singboxSRS | action unknown]", wantErrIs: lib.ErrUnknownAction},
		{name: "output action", action: lib.ActionOutput, listName: "cn", wantErr: "[type singboxSRS | action output]", wantErrIs: lib.ErrUnknownAction},
		{name: "remove from empty container", action: lib.ActionRemove, listName: "cn", wantErr: "not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewSRSIn(tt.action, append([]lib.InputOption{WithNameAndURI(tt.listName, path)}, tt.opts...)...)
			if err != nil {
				t.Fatalf("NewSRSIn: %v", err)
			}
			got, err := conv.Input(lib.NewContainer())
			if tt.wantErr != "" || tt.wantErrIs != nil {
				if err == nil {
					t.Fatal("got nil error, want error")
				}
				if tt.wantErr != "" {
					assertErrorContains(t, err, tt.wantErr)
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("error %v is not %v", err, tt.wantErrIs)
				}
				if got != nil {
					t.Errorf("got container %v, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Input: %v", err)
			}
			assertContainer(t, got, tt.want)
		})
	}
}

func TestSRSIn_Input_Remove(t *testing.T) {
	t.Parallel()

	path := writeSRS(t, filepath.Join(t.TempDir(), "cn.srs"), "10.0.1.0/24", "2001:db8::/33")
	initial := map[string][]string{
		"CN": {"10.0.0.0/23", "2001:db8::/32"},
		"US": {"8.8.8.0/24"},
	}

	tests := []struct {
		name       string
		onlyIPType lib.IPType
		want       map[string][]string
	}{
		{name: "both families", want: map[string][]string{"CN": {"10.0.0.0/24", "2001:db8:8000::/33"}, "US": {"8.8.8.0/24"}}},
		{name: "only ipv4", onlyIPType: lib.IPv4, want: map[string][]string{"CN": {"10.0.0.0/24", "2001:db8::/32"}, "US": {"8.8.8.0/24"}}},
		{name: "only ipv6", onlyIPType: lib.IPv6, want: map[string][]string{"CN": {"10.0.0.0/23", "2001:db8:8000::/33"}, "US": {"8.8.8.0/24"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewSRSIn(lib.ActionRemove, WithNameAndURI("cn", path), WithInputOnlyIPType(tt.onlyIPType))
			if err != nil {
				t.Fatalf("NewSRSIn: %v", err)
			}
			got, err := conv.Input(newTestContainer(t, initial))
			if err != nil {
				t.Fatalf("Input: %v", err)
			}
			assertContainer(t, got, tt.want)
		})
	}
}

func TestSRSIn_Input_AddToExistingContainer(t *testing.T) {
	t.Parallel()

	path := writeSRS(t, filepath.Join(t.TempDir(), "cn.srs"), "192.168.1.0/24", "2001:db8:8000::/33")
	conv, err := NewSRSIn(lib.ActionAdd, WithNameAndURI("cn", path))
	if err != nil {
		t.Fatalf("NewSRSIn: %v", err)
	}
	got, err := conv.Input(newTestContainer(t, map[string][]string{"CN": {"192.168.0.0/24", "2001:db8::/33"}, "US": usInput}))
	if err != nil {
		t.Fatalf("Input: %v", err)
	}
	assertContainer(t, got, map[string][]string{"CN": cnCIDRs, "US": usCIDRs})
}

func TestSRSIn_Input_LocalFileErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tests := []struct {
		name      string
		uri       string
		wantErr   string
		wantErrIs error
	}{
		{name: "file does not exist", uri: filepath.Join(dir, "missing.srs"), wantErrIs: fs.ErrNotExist},
		{name: "path is a directory", uri: dir, wantErr: "[type singboxSRS | action add]"},
		{name: "not an SRS file", uri: writeFile(t, filepath.Join(dir, "bad.srs"), []byte("not SRS")), wantErr: "invalid sing-box rule-set file"},
		{name: "empty file", uri: writeFile(t, filepath.Join(dir, "empty.srs"), nil), wantErrIs: io.EOF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewSRSIn(lib.ActionAdd, WithNameAndURI("cn", tt.uri))
			if err != nil {
				t.Fatalf("NewSRSIn: %v", err)
			}
			got, err := conv.Input(lib.NewContainer())
			if err == nil {
				t.Fatal("got nil error, want error")
			}
			assertErrorContains(t, err, "[type singboxSRS | action add]", tt.wantErr)
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("error %v is not %v", err, tt.wantErrIs)
			}
			if got != nil {
				t.Errorf("got container %v, want nil", got)
			}
		})
	}
}

func TestSRSIn_Input_Remote(t *testing.T) {
	t.Parallel()

	body := srsBytes(t, ipcidrRule(cnInput...))
	okServer := newServer(t, http.StatusOK, body)
	notFoundServer := newServer(t, http.StatusNotFound, body)
	garbageServer := newServer(t, http.StatusOK, []byte("garbage"))
	tlsServer := httptest.NewTLSServer(http.NotFoundHandler())
	t.Cleanup(tlsServer.Close)

	tests := []struct {
		name         string
		uri          string
		want         map[string][]string
		wantErr      string
		wantURLError bool
	}{
		{name: "status 200", uri: okServer.URL + "/cn.srs", want: map[string][]string{"CN": cnCIDRs}},
		{name: "upper-case scheme", uri: strings.Replace(okServer.URL, "http://", "HTTP://", 1) + "/cn.srs", want: map[string][]string{"CN": cnCIDRs}},
		{name: "status 404", uri: notFoundServer.URL + "/cn.srs", wantErr: "http status code 404"},
		{name: "status 200 with invalid content", uri: garbageServer.URL + "/cn.srs", wantErr: "invalid sing-box rule-set file"},
		{name: "connection aborted", uri: failingServerURL(t) + "/cn.srs", wantURLError: true},
		// The default client does not trust the test certificate: the error
		// proves https URIs are fetched remotely instead of opened locally.
		{name: "https is fetched remotely", uri: tlsServer.URL + "/cn.srs", wantURLError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conv, err := NewSRSIn(lib.ActionAdd, WithNameAndURI("cn", tt.uri))
			if err != nil {
				t.Fatalf("NewSRSIn: %v", err)
			}
			got, err := conv.Input(lib.NewContainer())
			switch {
			case tt.wantErr != "":
				assertErrorContains(t, err, "[type singboxSRS | action add]", tt.wantErr)
			case tt.wantURLError:
				var urlErr *url.Error
				if !errors.As(err, &urlErr) {
					t.Fatalf("got error %v, want *url.Error", err)
				}
			default:
				if err != nil {
					t.Fatalf("Input: %v", err)
				}
				assertContainer(t, got, tt.want)
			}
		})
	}
}

func TestSRSIn_Input_InputDir(t *testing.T) {
	t.Parallel()

	valid := func(cidrs ...string) []byte { return srsBytes(t, ipcidrRule(cidrs...)) }
	malformed := []byte("not SRS")

	tests := []struct {
		name    string
		files   map[string][]byte
		opts    []lib.InputOption
		want    map[string][]string
		wantErr string
	}{
		{
			name: "walks sub directories",
			files: map[string][]byte{
				"cn.srs":                valid(cnInput...),
				"sub/us.srs":            valid(usInput...),
				"sub/deeper/v4only.srs": valid(v4Input...),
			},
			want: map[string][]string{"CN": cnCIDRs, "US": usCIDRs, "V4ONLY": v4CIDRs},
		},
		{
			name: "entry names from file names",
			files: map[string][]byte{
				"geo.ip.srs":       valid(v4Input...),
				".hidden":          valid(v4Input...),
				"noext":            valid(v4Input...),
				"Mixed-Case_1.SRS": valid(v4Input...),
			},
			want: map[string][]string{"GEO.IP": v4CIDRs, ".HIDDEN": v4CIDRs, "NOEXT": v4CIDRs, "MIXED-CASE_1": v4CIDRs},
		},
		{
			name:  "hidden file with extension",
			files: map[string][]byte{".hidden.srs": valid(v4Input...)},
			want:  map[string][]string{".HIDDEN": v4CIDRs},
		},
		{
			name: "rule-set without ip cidr gives an empty list",
			files: map[string][]byte{
				"cn.srs":     valid(cnInput...),
				"domain.srs": srsBytes(t, option.HeadlessRule{Type: constant.RuleTypeDefault, DefaultOptions: option.DefaultHeadlessRule{Domain: []string{"example.com"}}}),
			},
			want: map[string][]string{"CN": cnCIDRs, "DOMAIN": nil},
		},
		{
			name: "wanted list skips other files without parsing them",
			files: map[string][]byte{
				"cn.srs":  valid(cnInput...),
				"us.srs":  valid(usInput...),
				"bad.srs": malformed,
			},
			opts: []lib.InputOption{WithInputWantedList([]string{" us ", "cn"})},
			want: map[string][]string{"CN": cnCIDRs, "US": usCIDRs},
		},
		{
			name: "only ipv6",
			files: map[string][]byte{
				"cn.srs": valid(cnInput...),
				"v4.srs": valid(v4Input...),
			},
			opts: []lib.InputOption{WithInputOnlyIPType(lib.IPv6)},
			want: map[string][]string{"CN": {"2001:db8::/32"}, "V4": nil},
		},
		{
			name: "only ipv4",
			files: map[string][]byte{
				"cn.srs": valid(cnInput...),
				"us.srs": valid(usInput...),
			},
			opts: []lib.InputOption{WithInputOnlyIPType(lib.IPv4)},
			want: map[string][]string{"CN": {"192.168.0.0/23"}, "US": {"10.0.0.0/7"}},
		},
		{
			name:  "input dir takes precedence over name and uri",
			files: map[string][]byte{"cn.srs": valid(cnInput...)},
			opts:  []lib.InputOption{WithNameAndURI("us", "/nonexistent/us.srs")},
			want:  map[string][]string{"CN": cnCIDRs},
		},
		{
			name:    "space in file name",
			files:   map[string][]byte{"bad name.srs": valid(v4Input...)},
			wantErr: "filename bad name.srs cannot be entry name",
		},
		{
			name:    "plus sign in file name",
			files:   map[string][]byte{"a+b.srs": valid(v4Input...)},
			wantErr: "filename a+b.srs cannot be entry name",
		},
		{
			name:    "non-ascii file name",
			files:   map[string][]byte{"中国.srs": valid(v4Input...)},
			wantErr: "cannot be entry name",
		},
		{
			name:    "duplicated list in the same directory",
			files:   map[string][]byte{"cn.srs": valid(cnInput...), "cn.json": valid(cnInput...)},
			wantErr: "found duplicated list CN",
		},
		{
			name:    "duplicated list in sub directory",
			files:   map[string][]byte{"cn.srs": valid(cnInput...), "sub/CN.srs": valid(cnInput...)},
			wantErr: "found duplicated list CN",
		},
		{
			name:    "malformed file",
			files:   map[string][]byte{"cn.srs": malformed},
			wantErr: "invalid sing-box rule-set file",
		},
		{
			name:    "empty directory",
			files:   map[string][]byte{},
			wantErr: "no entry is generated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			for name, data := range tt.files {
				writeFile(t, filepath.Join(dir, filepath.FromSlash(name)), data)
			}
			conv, err := NewSRSIn(lib.ActionAdd, append([]lib.InputOption{WithInputDir(dir)}, tt.opts...)...)
			if err != nil {
				t.Fatalf("NewSRSIn: %v", err)
			}
			got, err := conv.Input(lib.NewContainer())
			if tt.wantErr != "" {
				assertErrorContains(t, err, "[type singboxSRS | action add]", tt.wantErr)
				if got != nil {
					t.Errorf("got container %v, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Input: %v", err)
			}
			assertContainer(t, got, tt.want)
		})
	}

	t.Run("directory does not exist", func(t *testing.T) {
		t.Parallel()

		conv, err := NewSRSIn(lib.ActionAdd, WithInputDir(filepath.Join(t.TempDir(), "missing")))
		if err != nil {
			t.Fatalf("NewSRSIn: %v", err)
		}
		_, err = conv.Input(lib.NewContainer())
		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("got error %v, want %v", err, fs.ErrNotExist)
		}
	})
}

func TestSRSIn_Input_MissingArguments(t *testing.T) {
	t.Parallel()

	// The constructor rejects these configurations, so build them directly
	// to reach the defensive branch of Input.
	tests := []struct {
		name string
		in   *srs_in
	}{
		{name: "nothing set", in: &srs_in{Type: TypeSRSIn, Action: lib.ActionAdd}},
		{name: "name without uri", in: &srs_in{Type: TypeSRSIn, Action: lib.ActionAdd, Name: "cn"}},
		{name: "uri without name", in: &srs_in{Type: TypeSRSIn, Action: lib.ActionAdd, URI: "cn.srs"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.in.Input(lib.NewContainer())
			assertErrorContains(t, err, "[type singboxSRS | action add]", "config missing argument inputDir or name or uri")
			if got != nil {
				t.Errorf("got container %v, want nil", got)
			}
		})
	}
}

// TestSRSIn_Registration checks the converter is reachable through the lib
// config registry set up in init.
func TestSRSIn_Registration(t *testing.T) {
	t.Parallel()

	if err := lib.RegisterInputConverter(TypeSRSIn, &srs_in{}); !errors.Is(err, lib.ErrDuplicatedConverter) {
		t.Errorf("RegisterInputConverter: got %v, want %v", err, lib.ErrDuplicatedConverter)
	}
	if err := lib.RegisterInputConfigCreator(TypeSRSIn, nil); err == nil {
		t.Error("RegisterInputConfigCreator: got nil error for an already registered type")
	}

	path := writeSRS(t, filepath.Join(t.TempDir(), "cn.srs"), cnInput...)
	args, err := json.Marshal(map[string]any{"name": "cn", "uri": path, "onlyIPType": "ipv4"})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		config  string
		want    map[string][]string
		wantErr bool
	}{
		{
			name:   "valid args",
			config: `{"input": [{"type": "singboxSRS", "action": "add", "args": ` + string(args) + `}]}`,
			want:   map[string][]string{"CN": {"192.168.0.0/23"}},
		},
		{name: "invalid args", config: `{"input": [{"type": "singboxSRS", "action": "add", "args": {"name": 1}}]}`, wantErr: true},
		{name: "missing args", config: `{"input": [{"type": "singboxSRS", "action": "add"}]}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			instance, err := lib.NewInstance()
			if err != nil {
				t.Fatal(err)
			}
			err = instance.InitConfigFromBytes([]byte(tt.config))
			if tt.wantErr {
				if err == nil {
					t.Fatal("got nil error, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("InitConfigFromBytes: %v", err)
			}
			container := lib.NewContainer()
			if err := instance.RunInput(container); err != nil {
				t.Fatalf("RunInput: %v", err)
			}
			assertContainer(t, container, tt.want)
		})
	}
}
