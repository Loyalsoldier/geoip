package mihomo

import (
	"bytes"
	"errors"
	"net/netip"
	"slices"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"go4.org/netipx"
)

type failingWriter struct {
	err    error
	writes int
}

func (w *failingWriter) Write([]byte) (int, error) {
	w.writes++
	return 0, w.err
}

func TestMRSFlushError(t *testing.T) {
	writeErr := errors.New("write failed")
	writer := &failingWriter{err: writeErr}
	ranges := []netipx.IPRange{
		netipx.IPRangeFrom(netip.MustParseAddr("192.0.2.0"), netip.MustParseAddr("192.0.2.255")),
	}
	err := (&MRSOut{}).convertToMrs(ranges, writer)
	if writer.writes == 0 {
		t.Fatal("compressed data was never written")
	}
	if !errors.Is(err, writeErr) {
		t.Fatalf("convertToMrs returned %v, want %v", err, writeErr)
	}
}

func TestMRSRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		prefixes []string
	}{
		{name: "IPv4", prefixes: []string{"192.0.2.0/24", "198.51.100.7/32"}},
		{name: "IPv6", prefixes: []string{"2001:db8::/126", "2001:db8:1::1/128"}},
		{name: "mixed", prefixes: []string{"192.0.2.0/24", "2001:db8::/126"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := lib.NewEntry("test")
			for _, prefix := range tt.prefixes {
				if err := entry.AddPrefix(prefix); err != nil {
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
			decoded := lib.NewEntry("test")
			if err := (&MRSIn{}).parseMRS(data.Bytes(), decoded); err != nil {
				t.Fatal(err)
			}
			want, err := entry.MarshalPrefix()
			if err != nil {
				t.Fatal(err)
			}
			got, err := decoded.MarshalPrefix()
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, want) {
				t.Fatalf("decoded prefixes %v, want %v", got, want)
			}
		})
	}
}
