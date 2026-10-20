package maxmind

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/oschwald/maxminddb-golang/v2"
)

// mmdbRecord is a single network/data pair used to build MMDB fixtures.
type mmdbRecord struct {
	cidr string
	data mmdbtype.DataType
}

// buildMMDB builds an in-memory MMDB database containing the given records.
func buildMMDB(t *testing.T, records ...mmdbRecord) []byte {
	t.Helper()

	writer, err := mmdbwriter.New(mmdbwriter.Options{
		DatabaseType:            "Test-DB",
		Description:             map[string]string{"en": "Test database"},
		RecordSize:              24,
		IncludeReservedNetworks: true,
	})
	if err != nil {
		t.Fatalf("mmdbwriter.New() error = %v", err)
	}

	for _, r := range records {
		_, network, err := net.ParseCIDR(r.cidr)
		if err != nil {
			t.Fatalf("net.ParseCIDR(%q) error = %v", r.cidr, err)
		}
		if err := writer.Insert(network, r.data); err != nil {
			t.Fatalf("writer.Insert(%q) error = %v", r.cidr, err)
		}
	}

	var buf bytes.Buffer
	if _, err := writer.WriteTo(&buf); err != nil {
		t.Fatalf("writer.WriteTo() error = %v", err)
	}

	return buf.Bytes()
}

// corruptMMDB returns a copy of a valid MMDB database whose data section
// separator is corrupted, so that it can still be opened but fails Verify().
func corruptMMDB(t *testing.T, content []byte) []byte {
	t.Helper()

	db, err := maxminddb.OpenBytes(content)
	if err != nil {
		t.Fatalf("maxminddb.OpenBytes() error = %v", err)
	}
	treeSize := db.Metadata.NodeCount * db.Metadata.RecordSize / 4
	db.Close()

	corrupted := slices.Clone(content)
	corrupted[treeSize] = 0xFF

	db, err = maxminddb.OpenBytes(corrupted)
	if err != nil {
		t.Fatalf("maxminddb.OpenBytes(corrupted) error = %v", err)
	}
	defer db.Close()
	if err := db.Verify(); err == nil {
		t.Fatal("corrupted MMDB unexpectedly passed verification")
	}

	return corrupted
}

// writeTempFile writes content into a new file under a test temp dir and
// returns the file path.
func writeTempFile(t *testing.T, name string, content []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}

	return path
}

// newTestServer starts an HTTP server that responds to every request with
// the given status code and body.
func newTestServer(t *testing.T, status int, body []byte) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	return srv
}

// sourceURIs serves the same content as a local file, a remote http URL and
// a remote http URL with an upper case scheme, keyed by a descriptive name.
func sourceURIs(t *testing.T, content []byte) map[string]string {
	t.Helper()

	srv := newTestServer(t, http.StatusOK, content)

	return map[string]string{
		"local file":                    writeTempFile(t, "source.mmdb", content),
		"remote URL":                    srv.URL + "/source.mmdb",
		"remote URL with upper scheme": "HTTP://" + strings.TrimPrefix(srv.URL, "http://") + "/source.mmdb",
	}
}

func mustParsePrefix(t *testing.T, s string) netip.Prefix {
	t.Helper()

	prefix, err := netip.ParsePrefix(s)
	if err != nil {
		t.Fatalf("netip.ParsePrefix(%q) error = %v", s, err)
	}

	return prefix
}

// unreachableURL returns an http URL that refuses connections.
func unreachableURL(t *testing.T) string {
	t.Helper()

	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL + "/unreachable"
	srv.Close()

	return url
}

// newContainer builds a container from a map of list name to CIDRs.
func newContainer(t *testing.T, lists map[string][]string) lib.Container {
	t.Helper()

	container := lib.NewContainer()
	for name, cidrs := range lists {
		entry := lib.NewEntry(name)
		for _, cidr := range cidrs {
			if err := entry.AddPrefix(cidr); err != nil {
				t.Fatalf("entry.AddPrefix(%q) error = %v", cidr, err)
			}
		}
		if err := container.Add(entry); err != nil {
			t.Fatalf("container.Add(%q) error = %v", name, err)
		}
	}

	return container
}

// containerSnapshot returns all lists in the container with their CIDRs.
func containerSnapshot(t *testing.T, container lib.Container) map[string][]string {
	t.Helper()

	snapshot := make(map[string][]string, container.Len())
	for entry := range container.Loop() {
		cidrs, err := entry.MarshalText()
		if err != nil {
			// An entry whose prefixes were all removed has no prefix.
			cidrs = []string{}
		}
		snapshot[entry.GetName()] = cidrs
	}

	return snapshot
}

// assertContainer checks that the container holds exactly the wanted lists,
// comparing both the number and the exact content of CIDRs of every list.
func assertContainer(t *testing.T, container lib.Container, want map[string][]string) {
	t.Helper()

	if container == nil {
		t.Fatal("container is nil")
	}

	got := containerSnapshot(t, container)
	if len(got) != len(want) {
		t.Fatalf("container has %d lists %v, want %d lists %v", len(got), sortedKeys(got), len(want), sortedKeys(want))
	}

	for name, wantCIDRs := range want {
		entry, found := container.GetEntry(name)
		if !found {
			t.Fatalf("list %q not found in container", name)
		}
		gotCIDRs, err := entry.MarshalText()
		if err != nil {
			gotCIDRs = []string{}
		}
		assertCIDRs(t, name, gotCIDRs, wantCIDRs)
	}
}

// assertCIDRs compares both the length and the exact content of CIDR lists.
func assertCIDRs(t *testing.T, name string, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("list %q has %d CIDRs %v, want %d CIDRs %v", name, len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("list %q CIDR[%d] = %q, want %q (got %v, want %v)", name, i, got[i], want[i], got, want)
		}
	}
}

// assertError checks whether err matches the expectation. When wantErr is
// false, err must be nil. When wantErr is true, err must be non-nil and, if
// provided, contain errContains and/or wrap target.
func assertError(t *testing.T, err error, wantErr bool, errContains string, target error) {
	t.Helper()

	if !wantErr {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}

	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if errContains != "" && !strings.Contains(err.Error(), errContains) {
		t.Fatalf("error %q does not contain %q", err.Error(), errContains)
	}
	if target != nil && !errors.Is(err, target) {
		t.Fatalf("error %q does not wrap %q", err.Error(), target.Error())
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// fakeInputConverter is an input converter that does not belong to this
// package. It is used to check that options ignore unknown converters.
type fakeInputConverter struct{}

func (fakeInputConverter) GetType() string                                 { return "fake" }
func (fakeInputConverter) GetAction() lib.Action                           { return lib.ActionAdd }
func (fakeInputConverter) GetDescription() string                          { return "fake" }
func (fakeInputConverter) Input(c lib.Container) (lib.Container, error) { return c, nil }

type (
	mStr = mmdbtype.String
	mU32 = mmdbtype.Uint32
	mMap = mmdbtype.Map
)

// geoLite2StyleRecords returns records using the GeoLite2/DB-IP country
// schema. The resulting lists are:
//
//	CN: 1.0.0.0/23, 2402:4e00::/32 (country)
//	US: 2.0.0.0/16, 2400:cb00::/32 (registered_country)
//	IT: 3.0.0.0/16                 (represented_country)
//	DE: 2001:db8::/32              (country, without continent)
//
// 4.0.0.0/16 and 5.0.0.0/16 have no ISO code at all and must be skipped.
func geoLite2StyleRecords() []mmdbRecord {
	return []mmdbRecord{
		{"1.0.0.0/24", mMap{
			"continent": mMap{
				"code":       mStr("AS"),
				"geoname_id": mU32(6255147),
				"names":      mMap{"en": mStr("Asia"), "zh-CN": mStr("亚洲"), "fa": mStr("آسیا")},
			},
			"country": mMap{
				"iso_code":   mStr(" cn "),
				"geoname_id": mU32(1814991),
				"names":      mMap{"en": mStr("China"), "de": mStr("China"), "ko": mStr("중국")},
			},
			// country has a higher priority than registered_country
			"registered_country": mMap{"iso_code": mStr("US")},
		}},
		{"1.0.1.0/24", mMap{
			"country": mMap{
				"iso_code": mStr("CN"),
				"names":    mMap{"en": mStr("Ignored China")},
			},
		}},
		{"2.0.0.0/16", mMap{
			"continent": mMap{
				"code":       mStr("NA"),
				"geoname_id": mU32(6255149),
				"names":      mMap{"en": mStr("North America")},
			},
			"registered_country": mMap{
				"iso_code":   mStr("US"),
				"geoname_id": mU32(6252001),
				"names":      mMap{"en": mStr("United States")},
			},
			// registered_country has a higher priority than represented_country
			"represented_country": mMap{"iso_code": mStr("JP")},
		}},
		{"3.0.0.0/16", mMap{
			"continent": mMap{
				"code":       mStr("EU"),
				"geoname_id": mU32(6255148),
				"names":      mMap{"en": mStr("Europe")},
			},
			"represented_country": mMap{
				"iso_code":             mStr(" it "),
				"geoname_id":           mU32(3175395),
				"is_in_european_union": mmdbtype.Bool(true),
				"names":                mMap{"en": mStr("Italy")},
				"type":                 mStr("military"),
			},
		}},
		{"4.0.0.0/16", mMap{"continent": mMap{"code": mStr("EU")}}},
		{"5.0.0.0/16", mMap{}},
		{"2001:db8::/32", mMap{
			"country": mMap{
				"iso_code":             mStr("DE"),
				"geoname_id":           mU32(2921044),
				"is_in_european_union": mmdbtype.Bool(true),
				"names":                mMap{"en": mStr("Germany")},
			},
		}},
		{"2400:cb00::/32", mMap{"registered_country": mMap{"iso_code": mStr("us")}}},
		{"2402:4e00::/32", mMap{"country": mMap{"iso_code": mStr("CN")}}},
	}
}

// geoLite2StyleLists is the expected result of reading geoLite2StyleRecords.
func geoLite2StyleLists() map[string][]string {
	return map[string][]string{
		"CN": {"1.0.0.0/23", "2402:4e00::/32"},
		"US": {"2.0.0.0/16", "2400:cb00::/32"},
		"IT": {"3.0.0.0/16"},
		"DE": {"2001:db8::/32"},
	}
}

// ipInfoStyleRecords returns records using the IPInfo Lite schema. The
// resulting lists are:
//
//	CN: 1.0.0.0/23, 2402:4e00::/32
//	US: 2.0.0.0/16
//	DE: 2001:db8::/32
func ipInfoStyleRecords() []mmdbRecord {
	return []mmdbRecord{
		{"1.0.0.0/24", mMap{
			"asn":            mStr("AS13335"),
			"as_name":        mStr("Cloudflare, Inc."),
			"as_domain":      mStr("cloudflare.com"),
			"continent":      mStr("Asia"),
			"continent_code": mStr("AS"),
			"country":        mStr("China"),
			"country_code":   mStr(" cn "),
		}},
		{"1.0.1.0/24", mMap{"country_code": mStr("CN"), "country": mStr("Ignored China")}},
		{"2.0.0.0/16", mMap{
			"asn":            mStr("AS7018"),
			"continent":      mStr("North America"),
			"continent_code": mStr("NA"),
			"country":        mStr("United States"),
			"country_code":   mStr("us"),
		}},
		{"2001:db8::/32", mMap{
			"continent":      mStr("Europe"),
			"continent_code": mStr("EU"),
			"country":        mStr("Germany"),
			"country_code":   mStr("DE"),
		}},
		{"2402:4e00::/32", mMap{"country_code": mStr("CN")}},
	}
}

// ipInfoStyleLists is the expected result of reading ipInfoStyleRecords.
func ipInfoStyleLists() map[string][]string {
	return map[string][]string{
		"CN": {"1.0.0.0/23", "2402:4e00::/32"},
		"US": {"2.0.0.0/16"},
		"DE": {"2001:db8::/32"},
	}
}

// decodeErrorRecords returns records that can not be decoded into the
// record type of the given converter type.
func decodeErrorRecords(iType string) []mmdbRecord {
	switch iType {
	case TypeIPInfoCountryMMDBIn:
		return []mmdbRecord{{"1.0.0.0/24", mMap{"country_code": mU32(5)}}}
	default:
		return []mmdbRecord{{"1.0.0.0/24", mMap{"country": mStr("CN")}}}
	}
}

// mmdbInputType describes one of the MMDB input formats of this package
// together with a fixture database and the lists expected from it.
type mmdbInputType struct {
	iType          string
	desc           string
	defaultURI     string
	newIn          func(lib.Action, ...lib.InputOption) (lib.InputConverter, error)
	newInFromBytes func(lib.Action, []byte) (lib.InputConverter, error)
	records        []mmdbRecord
	lists          map[string][]string
}

func mmdbInputTypes() []mmdbInputType {
	return []mmdbInputType{
		{
			iType:          TypeGeoLite2CountryMMDBIn,
			desc:           DescGeoLite2CountryMMDBIn,
			defaultURI:     defaultGeoLite2CountryMMDBFile,
			newIn:          NewGeoLite2CountryMMDBIn,
			newInFromBytes: NewGeoLite2CountryMMDBInFromBytes,
			records:        geoLite2StyleRecords(),
			lists:          geoLite2StyleLists(),
		},
		{
			iType:          TypeDBIPCountryMMDBIn,
			desc:           DescDBIPCountryMMDBIn,
			defaultURI:     defaultDBIPCountryMMDBFile,
			newIn:          NewDBIPCountryMMDBIn,
			newInFromBytes: NewDBIPCountryMMDBInFromBytes,
			records:        geoLite2StyleRecords(),
			lists:          geoLite2StyleLists(),
		},
		{
			iType:          TypeIPInfoCountryMMDBIn,
			desc:           DescIPInfoCountryMMDBIn,
			defaultURI:     defaultIPInfoCountryMMDBFile,
			newIn:          NewIPInfoCountryMMDBIn,
			newInFromBytes: NewIPInfoCountryMMDBInFromBytes,
			records: append(ipInfoStyleRecords(),
				// records without country code must be skipped
				mmdbRecord{"4.0.0.0/16", mMap{"asn": mStr("AS1")}},
				mmdbRecord{"5.0.0.0/16", mMap{"country_code": mStr("   ")}},
			),
			lists: ipInfoStyleLists(),
		},
	}
}

// splitCIDRs splits CIDRs into IPv4 and IPv6 ones, keeping their order.
func splitCIDRs(cidrs []string) (ipv4, ipv6 []string) {
	ipv4, ipv6 = []string{}, []string{}
	for _, cidr := range cidrs {
		if strings.Contains(cidr, ":") {
			ipv6 = append(ipv6, cidr)
		} else {
			ipv4 = append(ipv4, cidr)
		}
	}
	return ipv4, ipv6
}

// filterLists returns a copy of lists, keeping only the lists in names (all
// lists when names is empty) and only the CIDRs of onlyIPType (all CIDRs
// when onlyIPType is empty).
func filterLists(lists map[string][]string, onlyIPType lib.IPType, names ...string) map[string][]string {
	result := make(map[string][]string)
	for name, cidrs := range lists {
		if len(names) > 0 && !slices.Contains(names, name) {
			continue
		}
		ipv4, ipv6 := splitCIDRs(cidrs)
		switch onlyIPType {
		case lib.IPv4:
			result[name] = ipv4
		case lib.IPv6:
			result[name] = ipv6
		default:
			result[name] = slices.Concat(ipv4, ipv6)
		}
	}
	return result
}

// newInstanceFromConfig creates a lib.Instance from a config made of the
// given input and output converter configs.
func newInstanceFromConfig(t *testing.T, inputs, outputs []map[string]any) (lib.Instance, error) {
	t.Helper()

	config := map[string]any{"input": inputs, "output": outputs}
	content, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	instance, err := lib.NewInstance()
	if err != nil {
		t.Fatalf("lib.NewInstance() error = %v", err)
	}

	return instance, instance.InitConfigFromBytes(content)
}

// assertInputRegistered checks that an input converter and its config
// creator are registered with the given type.
func assertInputRegistered(t *testing.T, iType string) {
	t.Helper()

	if err := lib.RegisterInputConverter(iType, fakeInputConverter{}); !errors.Is(err, lib.ErrDuplicatedConverter) {
		t.Errorf("input converter %q is not registered: RegisterInputConverter() error = %v", iType, err)
	}
	if err := lib.RegisterInputConfigCreator(iType, nil); err == nil {
		t.Errorf("input config creator %q is not registered", iType)
	}
}

// assertOutputRegistered checks that an output converter and its config
// creator are registered with the given type.
func assertOutputRegistered(t *testing.T, iType string) {
	t.Helper()

	if err := lib.RegisterOutputConverter(iType, &geolite2_country_mmdb_out{}); !errors.Is(err, lib.ErrDuplicatedConverter) {
		t.Errorf("output converter %q is not registered: RegisterOutputConverter() error = %v", iType, err)
	}
	if err := lib.RegisterOutputConfigCreator(iType, nil); err == nil {
		t.Errorf("output config creator %q is not registered", iType)
	}
}

// readMMDBNetworks opens an MMDB file and returns a map of network CIDR to
// decoded record, as well as the database metadata.
func readMMDBNetworks(t *testing.T, path string) (map[string]map[string]any, maxminddb.Metadata) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}

	db, err := maxminddb.OpenBytes(content)
	if err != nil {
		t.Fatalf("maxminddb.OpenBytes() error = %v", err)
	}
	defer db.Close()

	if err := db.Verify(); err != nil {
		t.Fatalf("db.Verify() error = %v", err)
	}

	networks := make(map[string]map[string]any)
	for network := range db.Networks() {
		var record map[string]any
		if err := network.Decode(&record); err != nil {
			t.Fatalf("network.Decode() error = %v", err)
		}
		networks[network.Prefix().String()] = record
	}

	return networks, db.Metadata
}
