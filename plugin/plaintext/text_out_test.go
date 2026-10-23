package plaintext

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestTextOutConstructors(t *testing.T) {
	t.Parallel()

	testOutputConstructors(t, TypeTextOut, DescTextOut, filepath.Join("output", "text"), NewTextOut, NewTextOutFromBytes)
}

func TestTextOut_Config(t *testing.T) {
	t.Parallel()

	outputDir := filepath.Join(t.TempDir(), "text")
	instance, err := newInstanceFromConfig(t, map[string]any{
		"output": []map[string]any{
			{
				"type":   "text",
				"action": "output",
				"args": map[string]any{
					"outputDir":       outputDir,
					"outputExtension": ".conf",
					"excludedList":    []string{"us"},
					"addPrefixInLine": "IP-CIDR,",
					"addSuffixInLine": ",no-resolve",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	container := newTestContainer(t, map[string][]string{
		"CN": {"192.168.0.3/24", "192.168.1.5/24", "2001:db8::/33", "2001:db8:8000::/33"},
		"US": {"8.8.8.8"},
	})
	if err := instance.RunOutput(container); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertOutputFiles(t, outputDir, map[string]string{
		"cn.conf": "IP-CIDR,192.168.0.0/23,no-resolve\nIP-CIDR,2001:db8::/32,no-resolve\n",
	})
}

func TestTextOut_Getters(t *testing.T) {
	t.Parallel()

	to := &text_out{Type: "my-type", Action: lib.Action("my-action"), Description: "my description"}
	if got := to.GetType(); got != "my-type" {
		t.Errorf("GetType() = %q, want %q", got, "my-type")
	}
	if got := to.GetAction(); got != lib.Action("my-action") {
		t.Errorf("GetAction() = %q, want %q", got, "my-action")
	}
	if got := to.GetDescription(); got != "my description" {
		t.Errorf("GetDescription() = %q, want %q", got, "my description")
	}
}

func TestTextOut_FilterAndSortList(t *testing.T) {
	t.Parallel()

	lists := []string{"US", "CN", "JP", "PRIVATE"}

	tests := []struct {
		name    string
		lists   []string
		want    []string
		exclude []string
		result  []string
	}{
		{
			name:   "all lists sorted",
			lists:  lists,
			result: []string{"CN", "JP", "PRIVATE", "US"},
		},
		{
			name:   "empty container",
			lists:  nil,
			result: []string{},
		},
		{
			name:    "excluded lists are case-insensitive and trimmed",
			lists:   lists,
			exclude: []string{" cn ", "Private", "", "missing"},
			result:  []string{"JP", "US"},
		},
		{
			name:    "all lists excluded",
			lists:   lists,
			exclude: []string{"us", "cn", "jp", "private"},
			result:  []string{},
		},
		{
			name:   "wanted lists are upper-cased, trimmed and sorted",
			lists:  lists,
			want:   []string{" us", "cn ", "", "  "},
			result: []string{"CN", "US"},
		},
		{
			name:   "wanted lists not in container are kept",
			lists:  lists,
			want:   []string{"missing", "cn"},
			result: []string{"CN", "MISSING"},
		},
		{
			name:    "wanted lists minus excluded lists",
			lists:   lists,
			want:    []string{"us", "cn", "jp"},
			exclude: []string{"CN"},
			result:  []string{"JP", "US"},
		},
		{
			name:    "all wanted lists excluded",
			lists:   lists,
			want:    []string{"us", "cn"},
			exclude: []string{"cn", "us"},
			result:  []string{},
		},
		{
			name:   "blank-only wanted list is ignored",
			lists:  lists,
			want:   []string{"", "   "},
			result: []string{"CN", "JP", "PRIVATE", "US"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			container := lib.NewContainer()
			for _, name := range tt.lists {
				if err := container.Add(newTestEntry(t, name, "1.1.1.1")); err != nil {
					t.Fatalf("failed to add entry: %v", err)
				}
			}

			to := &text_out{Type: TypeTextOut, Want: tt.want, Exclude: tt.exclude}
			got := to.filterAndSortList(container)
			if got == nil {
				t.Fatal("filterAndSortList() should never return nil")
			}
			assertStrings(t, "lists", got, tt.result)
		})
	}
}

func TestTextOut_Output(t *testing.T) {
	t.Parallel()

	lists := map[string][]string{
		"CN":      {"192.168.0.3/24", "192.168.1.5/24", "2001:db8::/33", "2001:db8:8000::/33"},
		"US":      {"8.8.8.8", "::ffff:8.8.4.4"},
		"V6ONLY":  {"2001:db8:1::/48"},
		"PRIVATE": {"10.0.0.0/8", "fc00::/7"},
	}

	tests := []struct {
		name  string
		args  map[string]any
		lists map[string][]string
		want  map[string]string
	}{
		{
			name:  "all lists with default extension",
			lists: lists,
			want: map[string]string{
				"cn.txt":      "192.168.0.0/23\n2001:db8::/32\n",
				"us.txt":      "8.8.4.4/32\n8.8.8.8/32\n",
				"v6only.txt":  "2001:db8:1::/48\n",
				"private.txt": "10.0.0.0/8\nfc00::/7\n",
			},
		},
		{
			name:  "custom extension, wanted list, prefix and suffix",
			args:  map[string]any{"outputExtension": ".conf", "wantedList": []string{"cn", "us"}, "addPrefixInLine": "IP-CIDR,", "addSuffixInLine": ",no-resolve"},
			lists: lists,
			want: map[string]string{
				"cn.conf": "IP-CIDR,192.168.0.0/23,no-resolve\nIP-CIDR,2001:db8::/32,no-resolve\n",
				"us.conf": "IP-CIDR,8.8.4.4/32,no-resolve\nIP-CIDR,8.8.8.8/32,no-resolve\n",
			},
		},
		{
			name:  "excluded list",
			args:  map[string]any{"excludedList": []string{"private", "v6only"}, "addPrefixInLine": "HOST,"},
			lists: lists,
			want: map[string]string{
				"cn.txt": "HOST,192.168.0.0/23\nHOST,2001:db8::/32\n",
				"us.txt": "HOST,8.8.4.4/32\nHOST,8.8.8.8/32\n",
			},
		},
		{
			name:  "only IPv6",
			args:  map[string]any{"onlyIPType": "ipv6", "addSuffixInLine": ";"},
			lists: map[string][]string{"CN": lists["CN"], "V6ONLY": lists["V6ONLY"]},
			want: map[string]string{
				"cn.txt":     "2001:db8::/32;\n",
				"v6only.txt": "2001:db8:1::/48;\n",
			},
		},
		{
			name:  "only IPv4 with wanted list",
			args:  map[string]any{"onlyIPType": "ipv4", "wantedList": []string{"cn", "private"}},
			lists: lists,
			want: map[string]string{
				"cn.txt":      "192.168.0.0/23\n",
				"private.txt": "10.0.0.0/8\n",
			},
		},
		{
			name:  "wanted list with missing list",
			args:  map[string]any{"wantedList": []string{"missing", "us"}},
			lists: lists,
			want: map[string]string{
				"us.txt": "8.8.4.4/32\n8.8.8.8/32\n",
			},
		},
		{
			name:  "all wanted lists excluded writes nothing",
			args:  map[string]any{"wantedList": []string{"cn"}, "excludedList": []string{"cn"}},
			lists: lists,
			want:  nil,
		},
		{
			name:  "empty container writes nothing",
			lists: nil,
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			outputDir, err := runOutput(t, TypeTextOut, tt.args, tt.lists)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertOutputFiles(t, outputDir, tt.want)
		})
	}
}

func TestTextOut_Output_Errors(t *testing.T) {
	t.Parallel()

	t.Run("list without prefix of the wanted IP type", func(t *testing.T) {
		t.Parallel()

		_, err := runOutput(t, TypeTextOut,
			map[string]any{"onlyIPType": "ipv4"},
			map[string][]string{"CN": {"1.1.1.1"}, "V6ONLY": {"2001:db8::/32"}},
		)
		assertErrContains(t, err, "❌ [type text | action output] entry V6ONLY has no prefix")
	})

	t.Run("empty list", func(t *testing.T) {
		t.Parallel()

		_, err := runOutput(t, TypeTextOut, nil, map[string][]string{"EMPTY": nil})
		assertErrContains(t, err, "❌ [type text | action output] entry EMPTY has no prefix")
	})

	t.Run("unsupported type", func(t *testing.T) {
		t.Parallel()

		to := &text_out{Type: "unknown", Action: lib.ActionOutput, OutputDir: t.TempDir(), OutputExt: ".txt"}
		err := to.Output(newTestContainer(t, map[string][]string{"CN": {"1.1.1.1"}}))
		if !errors.Is(err, lib.ErrNotSupportedFormat) {
			t.Fatalf("expected %v, got %v", lib.ErrNotSupportedFormat, err)
		}
		assertErrContains(t, err, "❌ [type unknown | action output]")
	})

	t.Run("output dir cannot be created", func(t *testing.T) {
		t.Parallel()

		file := writeTestFile(t, t.TempDir(), "file", "content")
		to := &text_out{Type: TypeTextOut, Action: lib.ActionOutput, OutputDir: file, OutputExt: ".txt"}
		err := to.Output(newTestContainer(t, map[string][]string{"CN": {"1.1.1.1"}}))
		assertErrContains(t, err, "❌ [type text | action output]")
	})

	t.Run("output file cannot be written", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "us.txt"), 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		to := &text_out{Type: TypeTextOut, Action: lib.ActionOutput, OutputDir: dir, OutputExt: ".txt"}
		err := to.Output(newTestContainer(t, map[string][]string{"CN": {"1.1.1.1"}, "US": {"2.2.2.2"}}))
		assertErrContains(t, err, "❌ [type text | action output]")
		assertErrContains(t, err, "us.txt")

		// Lists are processed in sorted order, so CN was written before US failed.
		files := readDirFiles(t, dir)
		assertFiles(t, files, map[string]string{"cn.txt": "1.1.1.1/32\n"})
	})
}

// TestTextOut_Output_Logs must not run in parallel since it captures the
// output of the global logger.
func TestTextOut_Output_Logs(t *testing.T) {
	logs := captureLog(t)

	outputDir, err := runOutput(t, TypeTextOut,
		map[string]any{"wantedList": []string{"missing", "cn"}},
		map[string][]string{"CN": {"1.1.1.1"}, "US": {"2.2.2.2"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertOutputFiles(t, outputDir, map[string]string{"cn.txt": "1.1.1.1/32\n"})

	gotLines := strings.Split(strings.TrimSuffix(logs.String(), "\n"), "\n")
	wantLines := []string{
		"✅ [text] cn.txt --> " + outputDir,
		"❌ entry MISSING not found",
	}
	assertStrings(t, "log lines", gotLines, wantLines)
}
