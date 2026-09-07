package mihomo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"net/netip"
	"slices"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/klauspost/compress/zstd"
	"go4.org/netipx"
)

func TestMRSPrefixRoundTrip(t *testing.T) {
	tests := [][]string{
		{"192.0.2.0/24"},
		{"2001:db8::/32"},
		{"::/80"},
		{"::/0"},
		{"192.0.2.0/24", "::/80", "2001:db8::/32"},
	}
	for _, cidrs := range tests {
		t.Run(cidrs[0], func(t *testing.T) {
			entry := lib.NewEntry("test")
			for _, cidr := range cidrs {
				if err := entry.AddPrefix(cidr); err != nil {
					t.Fatal(err)
				}
			}
			ranges, err := entry.MarshalIPRange()
			if err != nil {
				t.Fatal(err)
			}
			var data bytes.Buffer
			if err := (&MRSOut{}).convertToMrs(ranges, &data); err != nil {
				t.Fatal(err)
			}
			result := lib.NewEntry("test")
			if err := (&MRSIn{}).parseMRS(data.Bytes(), result); err != nil {
				t.Fatal(err)
			}
			got, err := result.MarshalText()
			if err != nil {
				t.Fatal(err)
			}
			want, err := entry.MarshalText()
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, want) {
				t.Errorf("round-trip CIDRs = %v, want %v", got, want)
			}
		})
	}
}

func encodeMRSTestData(t *testing.T, extraLength int64, extra []byte, from, to string) []byte {
	t.Helper()
	var data bytes.Buffer
	for _, field := range []any{
		mrsMagicBytes, byte(1), int64(1), extraLength, extra,
		byte(1), int64(1), netip.MustParseAddr(from).As16(), netip.MustParseAddr(to).As16(),
	} {
		if err := binary.Write(&data, binary.BigEndian, field); err != nil {
			t.Fatal(err)
		}
	}
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	return encoder.EncodeAll(data.Bytes(), nil)
}

func TestMRSExtraLength(t *testing.T) {
	tests := []struct {
		name    string
		length  int64
		extra   []byte
		wantErr bool
	}{
		{name: "no extra"},
		{name: "reserved bytes", length: 4, extra: []byte{1, 2, 3, 4}},
		{name: "negative", length: -1, wantErr: true},
		{name: "truncated", length: 1000, wantErr: true},
		{name: "oversized", length: math.MaxInt64, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if value := recover(); value != nil {
					t.Errorf("parseMRS panicked: %v", value)
				}
			}()
			data := encodeMRSTestData(t, tt.length, tt.extra, "192.0.2.0", "192.0.2.255")
			entry := lib.NewEntry("test")
			err := (&MRSIn{}).parseMRS(data, entry)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseMRS error = %v, want error %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				got, err := entry.MarshalText()
				if err != nil {
					t.Fatal(err)
				}
				if !slices.Equal(got, []string{"192.0.2.0/24"}) {
					t.Errorf("CIDRs = %v, want [192.0.2.0/24]", got)
				}
			}
		})
	}
}

func TestMRSRejectsReversedRanges(t *testing.T) {
	for _, endpoints := range [][2]string{
		{"192.0.2.255", "192.0.2.0"},
		{"2001:db8::ffff", "2001:db8::"},
	} {
		t.Run(endpoints[0], func(t *testing.T) {
			data := encodeMRSTestData(t, 0, nil, endpoints[0], endpoints[1])
			if err := (&MRSIn{}).parseMRS(data, lib.NewEntry("test")); err == nil {
				t.Fatal("parseMRS accepted a reversed range")
			}
		})
	}
}

type failingMRSWriter struct {
	err error
}

func (w failingMRSWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestMRSPropagatesFinalWriteError(t *testing.T) {
	writeErr := errors.New("write failed")
	ranges := []netipx.IPRange{
		netipx.RangeOfPrefix(netip.MustParsePrefix("192.0.2.0/24")),
	}
	err := (&MRSOut{}).convertToMrs(ranges, failingMRSWriter{err: writeErr})
	if !errors.Is(err, writeErr) {
		t.Errorf("convertToMrs error = %v, want %v", err, writeErr)
	}
}
