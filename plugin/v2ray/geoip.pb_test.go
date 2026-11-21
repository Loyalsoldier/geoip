package v2ray

import (
	"slices"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestGeneratedProtoMessages(t *testing.T) {
	tests := []struct {
		name      string
		msg       func() proto.Message
		nilMsg    protoreflect.ProtoMessage
		fullName  protoreflect.FullName
		index     []int
		stringHas string
	}{
		{
			name:      "CIDR",
			msg:       func() proto.Message { return &CIDR{Ip: []byte{1, 2, 3, 4}, Prefix: 24} },
			nilMsg:    (*CIDR)(nil),
			fullName:  "geoip.plugin.v2ray.CIDR",
			index:     []int{0},
			stringHas: "prefix:24",
		},
		{
			name: "GeoIP",
			msg: func() proto.Message {
				return &GeoIP{CountryCode: "CN", Cidr: []*CIDR{{Ip: []byte{1, 0, 0, 0}, Prefix: 8}}}
			},
			nilMsg:    (*GeoIP)(nil),
			fullName:  "geoip.plugin.v2ray.GeoIP",
			index:     []int{1},
			stringHas: `country_code:"CN"`,
		},
		{
			name:      "GeoIPList",
			msg:       func() proto.Message { return &GeoIPList{Entry: []*GeoIP{{CountryCode: "US"}}} },
			nilMsg:    (*GeoIPList)(nil),
			fullName:  "geoip.plugin.v2ray.GeoIPList",
			index:     []int{2},
			stringHas: `country_code:"US"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.msg()
			if got := strings.ReplaceAll(m.(interface{ String() string }).String(), " ", ""); !strings.Contains(got, strings.ReplaceAll(tt.stringHas, " ", "")) {
				t.Errorf("String() = %q, want containing %q", got, tt.stringHas)
			}
			if got := m.ProtoReflect().Descriptor().FullName(); got != tt.fullName {
				t.Errorf("FullName = %s, want %s", got, tt.fullName)
			}
			if got := tt.nilMsg.ProtoReflect().Descriptor().FullName(); got != tt.fullName {
				t.Errorf("nil FullName = %s, want %s", got, tt.fullName)
			}

			desc, idx := m.(interface{ Descriptor() ([]byte, []int) }).Descriptor()
			if len(desc) == 0 || !slices.Equal(idx, tt.index) {
				t.Errorf("Descriptor() = %d bytes, %v; want non-empty, %v", len(desc), idx, tt.index)
			}

			m.(interface{ ProtoMessage() }).ProtoMessage()
			m.(interface{ Reset() }).Reset()
			if !proto.Equal(m, tt.nilMsg.ProtoReflect().New().Interface()) {
				t.Errorf("Reset() did not clear message: %v", m)
			}
		})
	}
}

func TestGeneratedGetters(t *testing.T) {
	var nilCIDR *CIDR
	var nilGeoIP *GeoIP
	var nilList *GeoIPList
	if nilCIDR.GetIp() != nil || nilCIDR.GetPrefix() != 0 {
		t.Error("nil CIDR getters should return zero values")
	}
	if nilGeoIP.GetCountryCode() != "" || nilGeoIP.GetCidr() != nil {
		t.Error("nil GeoIP getters should return zero values")
	}
	if nilList.GetEntry() != nil {
		t.Error("nil GeoIPList getter should return nil")
	}

	c := &CIDR{Ip: []byte{1, 2, 3, 4}, Prefix: 32}
	g := &GeoIP{CountryCode: "CN", Cidr: []*CIDR{c}}
	l := &GeoIPList{Entry: []*GeoIP{g}}
	if !slices.Equal(c.GetIp(), []byte{1, 2, 3, 4}) || c.GetPrefix() != 32 {
		t.Errorf("CIDR getters = %v %d", c.GetIp(), c.GetPrefix())
	}
	if g.GetCountryCode() != "CN" || len(g.GetCidr()) != 1 || g.GetCidr()[0] != c {
		t.Errorf("GeoIP getters = %s %v", g.GetCountryCode(), g.GetCidr())
	}
	if len(l.GetEntry()) != 1 || l.GetEntry()[0] != g {
		t.Errorf("GeoIPList getter = %v", l.GetEntry())
	}
}
