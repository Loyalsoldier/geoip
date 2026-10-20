package v2ray

import (
	"bytes"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestGeoIPProto_Getters(t *testing.T) {
	t.Parallel()

	t.Run("nil messages", func(t *testing.T) {
		t.Parallel()

		var cidr *CIDR
		var geoip *GeoIP
		var list *GeoIPList
		if cidr.GetIp() != nil || cidr.GetPrefix() != 0 {
			t.Error("nil CIDR getters should return zero values")
		}
		if geoip.GetCountryCode() != "" || geoip.GetCidr() != nil {
			t.Error("nil GeoIP getters should return zero values")
		}
		if list.GetEntry() != nil {
			t.Error("nil GeoIPList getters should return zero values")
		}
	})

	t.Run("populated messages", func(t *testing.T) {
		t.Parallel()

		cidr := &CIDR{Ip: []byte{1, 2, 3, 0}, Prefix: 24}
		geoip := &GeoIP{CountryCode: "CN", Cidr: []*CIDR{cidr}}
		list := &GeoIPList{Entry: []*GeoIP{geoip}}
		if !bytes.Equal(cidr.GetIp(), []byte{1, 2, 3, 0}) || cidr.GetPrefix() != 24 {
			t.Errorf("CIDR getters: got %v/%d", cidr.GetIp(), cidr.GetPrefix())
		}
		if geoip.GetCountryCode() != "CN" || len(geoip.GetCidr()) != 1 || geoip.GetCidr()[0] != cidr {
			t.Errorf("GeoIP getters: got %q %v", geoip.GetCountryCode(), geoip.GetCidr())
		}
		if len(list.GetEntry()) != 1 || list.GetEntry()[0] != geoip {
			t.Errorf("GeoIPList getters: got %v", list.GetEntry())
		}
	})
}

func TestGeoIPProto_Messages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		msg      interface{ proto.Message }
		fullName string
		index    []int
		contains string
	}{
		{name: "CIDR", msg: &CIDR{Ip: []byte{1, 2, 3, 4}, Prefix: 32}, fullName: "geoip.plugin.v2ray.CIDR", index: []int{0}, contains: "prefix:32"},
		{name: "GeoIP", msg: &GeoIP{CountryCode: "CN"}, fullName: "geoip.plugin.v2ray.GeoIP", index: []int{1}, contains: `country_code:"CN"`},
		{name: "GeoIPList", msg: &GeoIPList{Entry: []*GeoIP{{CountryCode: "US"}}}, fullName: "geoip.plugin.v2ray.GeoIPList", index: []int{2}, contains: `country_code:"US"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := string(tt.msg.ProtoReflect().Descriptor().FullName()); got != tt.fullName {
				t.Errorf("full name: got %q, want %q", got, tt.fullName)
			}

			str := tt.msg.(interface{ String() string }).String()
			if !strings.Contains(strings.ReplaceAll(str, " ", ""), strings.ReplaceAll(tt.contains, " ", "")) {
				t.Errorf("String() = %q, want it to contain %q", str, tt.contains)
			}

			desc, index := tt.msg.(interface{ Descriptor() ([]byte, []int) }).Descriptor()
			if len(desc) == 0 {
				t.Error("Descriptor returned an empty file descriptor")
			}
			assertInts(t, index, tt.index)

			tt.msg.(interface{ ProtoMessage() }).ProtoMessage()

			clone := proto.Clone(tt.msg)
			if !proto.Equal(clone, tt.msg) {
				t.Errorf("clone %v differs from %v", clone, tt.msg)
			}
			clone.(interface{ Reset() }).Reset()
			if size := proto.Size(clone); size != 0 {
				t.Errorf("Reset left %d encoded bytes in %v", size, clone)
			}
		})
	}
}

func TestGeoIPProto_NilProtoReflect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  proto.Message
	}{
		{name: "CIDR", msg: (*CIDR)(nil)},
		{name: "GeoIP", msg: (*GeoIP)(nil)},
		{name: "GeoIPList", msg: (*GeoIPList)(nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.msg.ProtoReflect().IsValid() {
				t.Error("ProtoReflect of a nil message should not be valid")
			}
		})
	}
}

func TestGeoIPProto_FileInit(t *testing.T) {
	t.Parallel()

	before := File_geoip_proto
	file_geoip_proto_init()
	if File_geoip_proto != before {
		t.Error("file_geoip_proto_init should be idempotent")
	}
	if got := File_geoip_proto.Path(); got != "geoip.proto" {
		t.Errorf("file path: got %q, want %q", got, "geoip.proto")
	}
	if got := File_geoip_proto.Messages().Len(); got != 3 {
		t.Errorf("got %d messages, want 3", got)
	}
}

func assertInts(t *testing.T, got, want []int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d]: got %d, want %d", i, got[i], want[i])
		}
	}
}
