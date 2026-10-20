package maxmind

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestNewGeoLite2CountryMMDBIn(t *testing.T) {
	t.Parallel()

	testMMDBInConstructors(t, findMMDBInputType(t, TypeGeoLite2CountryMMDBIn))
}

// TestMMDBIn_Config checks that every MMDB input type can be created from a
// config file and run through lib.Instance.
func TestMMDBIn_Config(t *testing.T) {
	t.Parallel()

	for _, typ := range mmdbInputTypes() {
		file := writeTempFile(t, "input.mmdb", buildMMDB(t, typ.records...))

		t.Run(typ.iType+"/valid args", func(t *testing.T) {
			t.Parallel()

			instance, err := newInstanceFromConfig(t, []map[string]any{{
				"type":   typ.iType,
				"action": lib.ActionAdd,
				"args":   map[string]any{"uri": file, "wantedList": []string{"cn"}, "onlyIPType": lib.IPv4},
			}}, nil)
			assertError(t, err, false, "", nil)

			container := lib.NewContainer()
			assertError(t, instance.RunInput(container), false, "", nil)
			assertContainer(t, container, filterLists(typ.lists, lib.IPv4, "CN"))
		})

		t.Run(typ.iType+"/invalid args", func(t *testing.T) {
			t.Parallel()

			_, err := newInstanceFromConfig(t, []map[string]any{{
				"type":   typ.iType,
				"action": lib.ActionAdd,
				"args":   "invalid",
			}}, nil)
			assertError(t, err, true, "[type "+typ.iType+" | action add]", nil)
		})
	}
}

func TestMMDBIn_Input(t *testing.T) {
	t.Parallel()

	notFoundURL := newTestServer(t, http.StatusNotFound, []byte("not found")).URL + "/input.mmdb"
	brokenURL := failingServerURL(t)
	invalidFile := writeTempFile(t, "invalid.mmdb", []byte("this is not a mmdb file"))
	missingFile := filepath.Join(t.TempDir(), "missing.mmdb")

	for _, typ := range mmdbInputTypes() {
		content := buildMMDB(t, typ.records...)
		file := writeTempFile(t, "input.mmdb", content)
		remoteURL := newTestServer(t, http.StatusOK, content).URL + "/input.mmdb"
		corruptedFile := writeTempFile(t, "corrupted.mmdb", corruptMMDB(t, content))
		decodeErrorFile := writeTempFile(t, "decode-error.mmdb", buildMMDB(t, decodeErrorRecords(typ.iType)...))

		newFullContainer := func(t *testing.T) lib.Container {
			t.Helper()

			return fullContainer(t, typ.lists)
		}

		tests := []struct {
			name        string
			action      lib.Action
			opts        []lib.InputOption
			container   func(t *testing.T) lib.Container
			want        map[string][]string
			wantErr     bool
			errContains string
			target      error
		}{
			{
				name:   "add all lists from local file",
				action: lib.ActionAdd,
				opts:   []lib.InputOption{WithURI(file)},
				want:   typ.lists,
			},
			{
				name:   "add all lists from remote URL",
				action: lib.ActionAdd,
				opts:   []lib.InputOption{WithURI(remoteURL)},
				want:   typ.lists,
			},
			{
				name:   "add wanted lists",
				action: lib.ActionAdd,
				opts:   []lib.InputOption{WithURI(file), WithInputWantedList([]string{" cn ", "de", "not-exist"})},
				want:   filterLists(typ.lists, "", "CN", "DE"),
			},
			{
				name:   "add only IPv4",
				action: lib.ActionAdd,
				opts:   []lib.InputOption{WithURI(file), WithInputOnlyIPType(lib.IPv4)},
				want:   filterLists(typ.lists, lib.IPv4),
			},
			{
				name:   "add only IPv6",
				action: lib.ActionAdd,
				opts:   []lib.InputOption{WithURI(file), WithInputOnlyIPType(lib.IPv6)},
				want:   filterLists(typ.lists, lib.IPv6),
			},
			{
				name:   "add into existing container",
				action: lib.ActionAdd,
				opts:   []lib.InputOption{WithURI(file), WithInputWantedList([]string{"CN"})},
				container: func(t *testing.T) lib.Container {
					return newContainer(t, map[string][]string{
						"CN":      {extraIPv4, "1.0.0.0/24", extraIPv6},
						otherList: {otherCIDR},
					})
				},
				want: map[string][]string{
					"CN":      {"1.0.0.0/23", extraIPv4, "2402:4e00::/32", extraIPv6},
					otherList: {otherCIDR},
				},
			},
			{
				name:      "remove all lists",
				action:    lib.ActionRemove,
				opts:      []lib.InputOption{WithURI(file)},
				container: newFullContainer,
				want:      afterRemove(typ.lists, ""),
			},
			{
				name:      "remove only IPv4",
				action:    lib.ActionRemove,
				opts:      []lib.InputOption{WithURI(file), WithInputOnlyIPType(lib.IPv4)},
				container: newFullContainer,
				want:      afterRemove(typ.lists, lib.IPv4),
			},
			{
				name:      "remove only IPv6",
				action:    lib.ActionRemove,
				opts:      []lib.InputOption{WithURI(remoteURL), WithInputOnlyIPType(lib.IPv6)},
				container: newFullContainer,
				want:      afterRemove(typ.lists, lib.IPv6),
			},
			{
				name:        "remove list not in container",
				action:      lib.ActionRemove,
				opts:        []lib.InputOption{WithURI(file), WithInputWantedList([]string{"CN"})},
				wantErr:     true,
				errContains: "entry CN not found",
			},
			{
				name:        "unknown action",
				action:      lib.Action("unknown"),
				opts:        []lib.InputOption{WithURI(file)},
				wantErr:     true,
				errContains: "[type " + typ.iType + " | action unknown]",
				target:      lib.ErrUnknownAction,
			},
			{
				name:        "wanted list matches nothing",
				action:      lib.ActionAdd,
				opts:        []lib.InputOption{WithURI(file), WithInputWantedList([]string{"not-exist"})},
				wantErr:     true,
				errContains: "no entry is generated",
			},
			{
				name:    "missing local file",
				action:  lib.ActionAdd,
				opts:    []lib.InputOption{WithURI(missingFile)},
				wantErr: true,
				target:  os.ErrNotExist,
			},
			{
				name:        "remote URL not found",
				action:      lib.ActionAdd,
				opts:        []lib.InputOption{WithURI(notFoundURL)},
				wantErr:     true,
				errContains: "404",
			},
			{
				name:        "remote connection aborted",
				action:      lib.ActionAdd,
				opts:        []lib.InputOption{WithURI(brokenURL)},
				wantErr:     true,
				errContains: "[type " + typ.iType + " | action add]",
			},
			{
				name:        "invalid mmdb",
				action:      lib.ActionAdd,
				opts:        []lib.InputOption{WithURI(invalidFile)},
				wantErr:     true,
				errContains: "[type " + typ.iType + " | action add]",
			},
			{
				name:        "corrupted mmdb",
				action:      lib.ActionAdd,
				opts:        []lib.InputOption{WithURI(corruptedFile)},
				wantErr:     true,
				errContains: "[type " + typ.iType + " | action add]",
			},
			{
				name:        "record decode error",
				action:      lib.ActionAdd,
				opts:        []lib.InputOption{WithURI(decodeErrorFile)},
				wantErr:     true,
				errContains: "cannot unmarshal",
			},
		}

		for _, tt := range tests {
			t.Run(typ.iType+"/"+tt.name, func(t *testing.T) {
				t.Parallel()

				converter, err := typ.newIn(tt.action, tt.opts...)
				assertError(t, err, false, "", nil)

				container := lib.NewContainer()
				if tt.container != nil {
					container = tt.container(t)
				}

				got, err := converter.Input(container)
				assertError(t, err, tt.wantErr, tt.errContains, tt.target)
				if tt.wantErr {
					if got != nil {
						t.Fatalf("Input() returned non-nil container on error")
					}
					return
				}

				if got != container {
					t.Fatalf("Input() returned a different container")
				}
				assertContainer(t, got, tt.want)
			})
		}
	}
}

func TestGeoLite2CountryMMDBIn_UnsupportedType(t *testing.T) {
	t.Parallel()

	content := buildMMDB(t, geoLite2StyleRecords()...)
	file := writeTempFile(t, "input.mmdb", content)

	t.Run("Input", func(t *testing.T) {
		t.Parallel()

		g := &geolite2_country_mmdb_in{Type: "unknown", Action: lib.ActionAdd, URI: file}
		got, err := g.Input(lib.NewContainer())
		assertError(t, err, true, "[type unknown | action add]", lib.ErrNotSupportedFormat)
		if got != nil {
			t.Fatalf("Input() returned non-nil container on error")
		}
	})

	t.Run("generateEntries", func(t *testing.T) {
		t.Parallel()

		g := &geolite2_country_mmdb_in{Type: "unknown"}
		entries := make(map[string]*lib.Entry)
		err := g.generateEntries(content, entries)
		if !errors.Is(err, lib.ErrNotSupportedFormat) {
			t.Fatalf("generateEntries() error = %v, want %v", err, lib.ErrNotSupportedFormat)
		}
		if len(entries) != 0 {
			t.Fatalf("generateEntries() generated %d entries, want 0", len(entries))
		}
	})

	t.Run("generateEntries with empty database", func(t *testing.T) {
		t.Parallel()

		// No network means the type is never checked.
		g := &geolite2_country_mmdb_in{Type: "unknown"}
		entries := make(map[string]*lib.Entry)
		assertError(t, g.generateEntries(buildMMDB(t), entries), false, "", nil)
		if len(entries) != 0 {
			t.Fatalf("generateEntries() generated %d entries, want 0", len(entries))
		}
	})
}

func TestGeoLite2CountryMMDBIn_generateEntries(t *testing.T) {
	t.Parallel()

	content := buildMMDB(t, geoLite2StyleRecords()...)

	tests := []struct {
		name     string
		want     map[string]bool
		existing map[string][]string
		expected map[string][]string
	}{
		{
			name:     "all lists",
			expected: geoLite2StyleLists(),
		},
		{
			name:     "wanted lists",
			want:     map[string]bool{"US": true, "IT": true},
			expected: filterLists(geoLite2StyleLists(), "", "US", "IT"),
		},
		{
			name:     "existing entries are extended",
			want:     map[string]bool{"DE": true},
			existing: map[string][]string{"DE": {"10.0.0.0/8"}, "FR": {"2001:db9::/32"}},
			expected: map[string][]string{"DE": {"10.0.0.0/8", "2001:db8::/32"}, "FR": {"2001:db9::/32"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entries := make(map[string]*lib.Entry)
			for name, cidrs := range tt.existing {
				entry := lib.NewEntry(name)
				for _, cidr := range cidrs {
					if err := entry.AddPrefix(cidr); err != nil {
						t.Fatalf("AddPrefix(%q) error = %v", cidr, err)
					}
				}
				entries[name] = entry
			}

			g := &geolite2_country_mmdb_in{Type: TypeGeoLite2CountryMMDBIn, Want: tt.want}
			assertError(t, g.generateEntries(content, entries), false, "", nil)

			if len(entries) != len(tt.expected) {
				t.Fatalf("generateEntries() generated %d entries %v, want %d entries %v", len(entries), sortedKeys(entries), len(tt.expected), sortedKeys(tt.expected))
			}
			for name, wantCIDRs := range tt.expected {
				entry, found := entries[name]
				if !found {
					t.Fatalf("entry %q not generated", name)
				}
				if entry.GetName() != name {
					t.Fatalf("entry name = %q, want %q", entry.GetName(), name)
				}
				gotCIDRs, err := entry.MarshalText()
				assertError(t, err, false, "", nil)
				assertCIDRs(t, name, gotCIDRs, wantCIDRs)
			}
		})
	}
}
