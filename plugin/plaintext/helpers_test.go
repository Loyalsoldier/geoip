package plaintext

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"maps"
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

// newTestTLSServer starts an HTTPS server with a self-signed certificate that
// responds to every request with status 200 and the given body. TLS handshake
// errors are not logged. The server is closed when the test finishes.
func newTestTLSServer(t *testing.T, body string) *httptest.Server {
	t.Helper()

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, body)
	}))
	srv.Config.ErrorLog = log.New(io.Discard, "", 0)
	srv.StartTLS()
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

// mustMarshalJSON marshals v to JSON or fails the test.
func mustMarshalJSON(t *testing.T, v any) []byte {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal %v: %v", v, err)
	}
	return data
}

// runInput creates an input converter of iType from the JSON args, runs it
// against a container pre-populated with initial and returns the result.
// Construction errors fail the test immediately.
func runInput(t *testing.T, iType string, action lib.Action, args map[string]any, initial map[string][]string) (lib.Container, error) {
	t.Helper()

	conv, err := newTextInFromBytes(iType, inputDescriptions[iType], action, mustMarshalJSON(t, args))
	if err != nil {
		t.Fatalf("failed to create input converter: %v", err)
	}

	container := newTestContainer(t, initial)
	got, err := conv.Input(container)
	if err == nil && got != container {
		t.Errorf("Input() should return the container it was given")
	}
	if err != nil && got != nil {
		t.Errorf("Input() should return a nil container on error, got %#v", got)
	}
	return got, err
}

// assertInputResult checks the outcome of runInput against the expectations
// of a test case.
func assertInputResult(t *testing.T, iType string, action lib.Action, got lib.Container, err error, want map[string][]string, wantErr string, wantErrIs error) {
	t.Helper()

	assertErr(t, err, wantErr, wantErrIs)
	if wantErr != "" || wantErrIs != nil {
		assertErrContains(t, err, "❌ [type "+iType+" | action "+string(action)+"]")
		return
	}
	assertContainer(t, got, want)
}

// runOutput creates an output converter of iType from the JSON args with the
// output dir set to a fresh temporary directory, runs it against a container
// holding lists and returns the output dir together with the error.
func runOutput(t *testing.T, iType string, args map[string]any, lists map[string][]string) (string, error) {
	t.Helper()

	outputDir := filepath.Join(t.TempDir(), "output")
	args = maps.Clone(args)
	if args == nil {
		args = make(map[string]any)
	}
	args["outputDir"] = outputDir

	conv, err := newTextOutFromBytes(iType, outputDescriptions[iType], lib.ActionOutput, mustMarshalJSON(t, args))
	if err != nil {
		t.Fatalf("failed to create output converter: %v", err)
	}

	return outputDir, conv.Output(newTestContainer(t, lists))
}

// assertOutputFiles checks that outputDir contains exactly the files in want.
// An empty want means the output dir must not have been created at all.
func assertOutputFiles(t *testing.T, outputDir string, want map[string]string) {
	t.Helper()

	if len(want) == 0 {
		if _, err := os.Stat(outputDir); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("expected output dir %s not to exist, got stat error %v", outputDir, err)
		}
		return
	}
	assertFiles(t, readDirFiles(t, outputDir), want)
}

// inputSource is a way of feeding a file to an input converter.
type inputSource string

const (
	sourceLocalFile inputSource = "local file"
	sourceInputDir  inputSource = "input dir"
	sourceRemote    inputSource = "remote"
)

// allInputSources lists every supported way of feeding a file to an input
// converter.
var allInputSources = []inputSource{sourceLocalFile, sourceInputDir, sourceRemote}

// withInputSource serves content as the list "cn" (file name "cn"+ext) from
// source and returns a copy of args extended with the matching "name" and
// "uri", or "inputDir" fields.
func withInputSource(t *testing.T, source inputSource, ext, content string, args map[string]any) map[string]any {
	t.Helper()

	args = maps.Clone(args)
	if args == nil {
		args = make(map[string]any)
	}

	filename := "cn" + ext
	switch source {
	case sourceLocalFile:
		args["name"] = "cn"
		args["uri"] = writeTestFile(t, t.TempDir(), filename, content)
	case sourceInputDir:
		dir := t.TempDir()
		writeTestFile(t, dir, filename, content)
		args["inputDir"] = dir
	case sourceRemote:
		args["name"] = "cn"
		args["uri"] = newTestServer(t, http.StatusOK, content).URL + "/" + filename
	default:
		t.Fatalf("unknown input source %q", source)
	}

	return args
}

// testInputConstructors checks the exported constructors of an input type.
// validOpts and validArgs must describe a valid configuration.
func testInputConstructors(
	t *testing.T,
	iType, desc string,
	newFn func(lib.Action, ...lib.InputOption) (lib.InputConverter, error),
	newFromBytesFn func(lib.Action, []byte) (lib.InputConverter, error),
	validOpts []lib.InputOption,
	validArgs string,
) {
	t.Helper()

	assertConv := func(t *testing.T, conv lib.InputConverter, action lib.Action) {
		t.Helper()
		if conv.GetType() != iType || conv.GetAction() != action || conv.GetDescription() != desc {
			t.Errorf("type/action/description = %q/%q/%q, want %q/%q/%q",
				conv.GetType(), conv.GetAction(), conv.GetDescription(), iType, action, desc)
		}
		if ti := asTextIn(t, conv); ti.Type != iType {
			t.Errorf("Type = %q, want %q", ti.Type, iType)
		}
	}

	t.Run("registered by init", func(t *testing.T) {
		t.Parallel()
		assertInputRegistered(t, iType)
	})

	t.Run("New", func(t *testing.T) {
		t.Parallel()

		conv, err := newFn(lib.ActionAdd, validOpts...)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertConv(t, conv, lib.ActionAdd)
	})

	t.Run("New without options", func(t *testing.T) {
		t.Parallel()

		conv, err := newFn(lib.ActionRemove)
		assertErrContains(t, err, "❌ [type "+iType+" | action remove]")
		if conv != nil {
			t.Errorf("expected nil converter, got %#v", conv)
		}
	})

	t.Run("NewFromBytes", func(t *testing.T) {
		t.Parallel()

		conv, err := newFromBytesFn(lib.ActionRemove, []byte(validArgs))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertConv(t, conv, lib.ActionRemove)
	})

	t.Run("NewFromBytes with invalid JSON", func(t *testing.T) {
		t.Parallel()

		conv, err := newFromBytesFn(lib.ActionAdd, []byte(`{"name":`))
		assertErrContains(t, err, "❌ [type "+iType+" | action add]")
		var syntaxErr *json.SyntaxError
		if !errors.As(err, &syntaxErr) {
			t.Errorf("expected *json.SyntaxError, got %v", err)
		}
		if conv != nil {
			t.Errorf("expected nil converter, got %#v", conv)
		}
	})
}

// testOutputConstructors checks the exported constructors of an output type.
func testOutputConstructors(
	t *testing.T,
	iType, desc, defaultDir string,
	newFn func(lib.Action, ...lib.OutputOption) (lib.OutputConverter, error),
	newFromBytesFn func(lib.Action, []byte) (lib.OutputConverter, error),
) {
	t.Helper()

	t.Run("registered by init", func(t *testing.T) {
		t.Parallel()
		assertOutputRegistered(t, iType)
	})

	t.Run("New with defaults", func(t *testing.T) {
		t.Parallel()

		conv, err := newFn(lib.ActionOutput)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertTextOut(t, asTextOut(t, conv), &text_out{
			Type:        iType,
			Action:      lib.ActionOutput,
			Description: desc,
			OutputDir:   defaultDir,
			OutputExt:   ".txt",
		})
	})

	t.Run("New with options", func(t *testing.T) {
		t.Parallel()

		conv, err := newFn(lib.ActionOutput, WithOutputDir("./out"), WithOutputExtension(".yaml"), WithOutputOnlyIPType(lib.IPv6))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertTextOut(t, asTextOut(t, conv), &text_out{
			Type:        iType,
			Action:      lib.ActionOutput,
			Description: desc,
			OutputDir:   "./out",
			OutputExt:   ".yaml",
			OnlyIPType:  lib.IPv6,
		})
	})

	t.Run("NewFromBytes", func(t *testing.T) {
		t.Parallel()

		conv, err := newFromBytesFn(lib.ActionOutput, []byte(`{"outputExtension":".list","wantedList":["cn"]}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertTextOut(t, asTextOut(t, conv), &text_out{
			Type:        iType,
			Action:      lib.ActionOutput,
			Description: desc,
			OutputDir:   defaultDir,
			OutputExt:   ".list",
			Want:        []string{"cn"},
		})
	})

	t.Run("NewFromBytes with invalid JSON", func(t *testing.T) {
		t.Parallel()

		conv, err := newFromBytesFn(lib.ActionOutput, []byte(`{"outputDir":}`))
		assertErrContains(t, err, "❌ [type "+iType+" | action output]")
		var syntaxErr *json.SyntaxError
		if !errors.As(err, &syntaxErr) {
			t.Errorf("expected *json.SyntaxError, got %v", err)
		}
		if conv != nil {
			t.Errorf("expected nil converter, got %#v", conv)
		}
	})
}

// newInstanceFromConfig parses config with lib.Instance, which goes through
// the config creators registered by the init functions of this package.
func newInstanceFromConfig(t *testing.T, config map[string]any) (lib.Instance, error) {
	t.Helper()

	instance, err := lib.NewInstance()
	if err != nil {
		t.Fatalf("failed to create instance: %v", err)
	}
	return instance, instance.InitConfigFromBytes(mustMarshalJSON(t, config))
}
