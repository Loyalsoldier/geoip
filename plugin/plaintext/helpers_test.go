package plaintext

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

// errTestRead is returned by readers that are designed to fail in tests.
var errTestRead = errors.New("test read error")

// inputDescriptions maps every input type of this package to its description.
var inputDescriptions = map[string]string{
	TypeTextIn:                  DescTextIn,
	TypeJSONIn:                  DescJSONIn,
	TypeClashRuleSetIPCIDRIn:    DescClashRuleSetIPCIDRIn,
	TypeClashRuleSetClassicalIn: DescClashRuleSetClassicalIn,
	TypeSurgeRuleSetIn:          DescSurgeRuleSetIn,
}

// outputDescriptions maps every output type of this package to its description.
var outputDescriptions = map[string]string{
	TypeTextOut:                  DescTextOut,
	TypeClashRuleSetIPCIDROut:    DescClashRuleSetIPCIDROut,
	TypeClashRuleSetClassicalOut: DescClashRuleSetClassicalOut,
	TypeSurgeRuleSetOut:          DescSurgeRuleSetOut,
}

// writeTestFile writes content to dir/name, creating parent directories as
// needed, and returns the full path of the file.
func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}

	return path
}

// readDirFiles returns the names and contents of all regular files directly
// inside dir.
func readDirFiles(t *testing.T, dir string) map[string]string {
	t.Helper()

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read directory %s: %v", dir, err)
	}

	files := make(map[string]string, len(dirEntries))
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, de.Name()))
		if err != nil {
			t.Fatalf("failed to read file %s: %v", de.Name(), err)
		}
		files[de.Name()] = string(data)
	}

	return files
}

// assertFiles checks that got contains exactly the files in want with
// exactly the same contents.
func assertFiles(t *testing.T, got, want map[string]string) {
	t.Helper()

	gotNames := sortedKeys(got)
	wantNames := sortedKeys(want)
	assertStrings(t, "output file names", gotNames, wantNames)

	for name, wantContent := range want {
		if gotContent := got[name]; gotContent != wantContent {
			t.Errorf("content of %s mismatch:\ngot:\n%s\nwant:\n%s", name, gotContent, wantContent)
		}
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

// assertStrings checks both the length and the exact element-wise content of
// two string slices.
func assertStrings(t *testing.T, what string, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s: length mismatch: got %d %q, want %d %q", what, len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s: element %d mismatch: got %q, want %q (full got %q, want %q)", what, i, got[i], want[i], got, want)
		}
	}
}

// entryCIDRs returns the merged and sorted CIDRs of entry.
func entryCIDRs(t *testing.T, entry *lib.Entry) []string {
	t.Helper()

	cidrs, err := entry.MarshalText()
	if err != nil {
		t.Fatalf("failed to marshal entry %s: %v", entry.GetName(), err)
	}
	return cidrs
}

// assertContainer checks that container holds exactly the lists in want,
// each with exactly the expected CIDRs. A nil CIDR slice in want means the
// list must exist but contain no prefix at all.
func assertContainer(t *testing.T, container lib.Container, want map[string][]string) {
	t.Helper()

	gotNames := make([]string, 0, container.Len())
	for entry := range container.Loop() {
		gotNames = append(gotNames, entry.GetName())
	}
	slices.Sort(gotNames)
	assertStrings(t, "list names", gotNames, sortedKeys(want))

	for name, wantCIDRs := range want {
		entry, found := container.GetEntry(name)
		if !found {
			t.Fatalf("list %s not found in container", name)
		}
		if wantCIDRs == nil {
			if cidrs, err := entry.MarshalText(); err == nil {
				t.Errorf("list %s: expected no prefix, got %q", name, cidrs)
			}
			continue
		}
		assertStrings(t, "CIDRs of list "+name, entryCIDRs(t, entry), wantCIDRs)
	}
}

// newTestEntry builds an entry from the given IPs/CIDRs.
func newTestEntry(t *testing.T, name string, cidrs ...string) *lib.Entry {
	t.Helper()

	entry := lib.NewEntry(name)
	for _, cidr := range cidrs {
		if err := entry.AddPrefix(cidr); err != nil {
			t.Fatalf("failed to add %q to entry %s: %v", cidr, name, err)
		}
	}
	return entry
}

// newTestContainer builds a container from a map of list name to IPs/CIDRs.
func newTestContainer(t *testing.T, lists map[string][]string) lib.Container {
	t.Helper()

	container := lib.NewContainer()
	for name, cidrs := range lists {
		if err := container.Add(newTestEntry(t, name, cidrs...)); err != nil {
			t.Fatalf("failed to add entry %s to container: %v", name, err)
		}
	}
	return container
}

// assertErrContains fails the test if err is nil or does not contain substr.
func assertErrContains(t *testing.T, err error, substr string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error containing %q, got nil", substr)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("expected error containing %q, got %q", substr, err.Error())
	}
}

// assertErr checks err against the expectations of a test case: wantErr is a
// substring of the error message and wantErrIs is a sentinel error that must be
// found in the error chain. If both are empty, err must be nil.
func assertErr(t *testing.T, err error, wantErr string, wantErrIs error) {
	t.Helper()

	if wantErr == "" && wantErrIs == nil {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	if err == nil {
		t.Fatalf("expected error (contains %q, is %v), got nil", wantErr, wantErrIs)
	}
	if wantErr != "" && !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("expected error containing %q, got %q", wantErr, err.Error())
	}
	if wantErrIs != nil && !errors.Is(err, wantErrIs) {
		t.Fatalf("expected error wrapping %v, got %v", wantErrIs, err)
	}
}

// newTestServer starts an HTTP server that responds to every request with the
// given status code and body. The server is closed when the test finishes.
func newTestServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// closedServerURL returns the URL of an HTTP server which has already been
// shut down, so that any request to it fails without real network access.
func closedServerURL(t *testing.T) string {
	t.Helper()

	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	return url
}

// captureLog redirects the standard logger to a buffer until the test ends.
// Tests using it must not call t.Parallel.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	prevWriter, prevFlags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(prevWriter)
		log.SetFlags(prevFlags)
	})
	return &buf
}

// asTextIn type-asserts an input converter to *text_in.
func asTextIn(t *testing.T, conv lib.InputConverter) *text_in {
	t.Helper()

	ti, ok := conv.(*text_in)
	if !ok {
		t.Fatalf("expected *text_in, got %T", conv)
	}
	return ti
}

// asTextOut type-asserts an output converter to *text_out.
func asTextOut(t *testing.T, conv lib.OutputConverter) *text_out {
	t.Helper()

	to, ok := conv.(*text_out)
	if !ok {
		t.Fatalf("expected *text_out, got %T", conv)
	}
	return to
}

// assertInputRegistered verifies that init() registered both the input
// converter and the input config creator for iType. Registering an already
// registered type returns an error and leaves the registry untouched.
func assertInputRegistered(t *testing.T, iType string) {
	t.Helper()

	if err := lib.RegisterInputConverter(iType, &text_in{}); !errors.Is(err, lib.ErrDuplicatedConverter) {
		t.Errorf("input converter %s: expected %v, got %v", iType, lib.ErrDuplicatedConverter, err)
	}
	err := lib.RegisterInputConfigCreator(iType, func(lib.Action, json.RawMessage) (lib.InputConverter, error) {
		return nil, nil
	})
	if err == nil {
		t.Errorf("input config creator %s: expected already registered error, got nil", iType)
	}
}

// assertOutputRegistered verifies that init() registered both the output
// converter and the output config creator for iType.
func assertOutputRegistered(t *testing.T, iType string) {
	t.Helper()

	if err := lib.RegisterOutputConverter(iType, &text_out{}); !errors.Is(err, lib.ErrDuplicatedConverter) {
		t.Errorf("output converter %s: expected %v, got %v", iType, lib.ErrDuplicatedConverter, err)
	}
	err := lib.RegisterOutputConfigCreator(iType, func(lib.Action, json.RawMessage) (lib.OutputConverter, error) {
		return nil, nil
	})
	if err == nil {
		t.Errorf("output config creator %s: expected already registered error, got nil", iType)
	}
}
