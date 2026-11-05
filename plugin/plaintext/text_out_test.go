package plaintext

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func newTestContainer(t *testing.T, data map[string][]string) lib.Container {
	t.Helper()
	c := lib.NewContainer()
	for name, cidrs := range data {
		e := lib.NewEntry(name)
		for _, cidr := range cidrs {
			if err := e.AddPrefix(cidr); err != nil {
				t.Fatal(err)
			}
		}
		if err := c.Add(e); err != nil {
			t.Fatal(err)
		}
	}
	return c
}

func TestTextOutFilterAndSortList(t *testing.T) {
	container := newTestContainer(t, map[string][]string{
		"CN": {"1.0.1.0/24"},
		"US": {"2.0.0.0/24"},
		"JP": {"3.0.0.0/24"},
	})
	tests := []struct {
		name    string
		want    []string
		exclude []string
		result  []string
	}{
		{name: "all", result: []string{"CN", "JP", "US"}},
		{name: "exclude", exclude: []string{" us ", ""}, result: []string{"CN", "JP"}},
		{name: "wanted", want: []string{" us", "cn", ""}, result: []string{"CN", "US"}},
		{name: "wanted and excluded", want: []string{"us", "cn"}, exclude: []string{"cn"}, result: []string{"US"}},
		{name: "wanted all excluded", want: []string{"cn"}, exclude: []string{"cn"}, result: []string{}},
		{name: "wanted only blank", want: []string{" "}, result: []string{"CN", "JP", "US"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &text_out{Want: tt.want, Exclude: tt.exclude}
			got := o.filterAndSortList(container)
			if len(got) != len(tt.result) || !slices.Equal(got, tt.result) {
				t.Fatalf("got %v, want %v", got, tt.result)
			}
		})
	}
}

func TestTextOutOutput(t *testing.T) {
	container := newTestContainer(t, map[string][]string{
		"CN": {"1.0.1.0/24", "2001:db8::/32"},
		"US": {"2.0.0.0/24"},
	})

	t.Run("success with missing wanted entry", func(t *testing.T) {
		dir := t.TempDir()
		o := &text_out{Type: TypeTextOut, OutputDir: dir, OutputExt: ".txt", Want: []string{"cn", "us", "none"}}
		if err := o.Output(container); err != nil {
			t.Fatal(err)
		}
		files := map[string]string{"cn.txt": "1.0.1.0/24\n2001:db8::/32\n", "us.txt": "2.0.0.0/24\n"}
		entries, _ := os.ReadDir(dir)
		if len(entries) != len(files) {
			t.Fatalf("got %d files, want %d", len(entries), len(files))
		}
		for name, want := range files {
			got, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil || string(got) != want {
				t.Fatalf("%s = %q, err = %v, want %q", name, got, err, want)
			}
		}
	})

	t.Run("marshal error", func(t *testing.T) {
		o := &text_out{Type: "unknown", OutputDir: t.TempDir(), OutputExt: ".txt"}
		if err := o.Output(container); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("write error", func(t *testing.T) {
		dir := t.TempDir()
		blocker := filepath.Join(dir, "file")
		os.WriteFile(blocker, nil, 0644)
		o := &text_out{Type: TypeTextOut, OutputDir: filepath.Join(blocker, "sub"), OutputExt: ".txt"}
		if err := o.Output(container); err == nil {
			t.Fatal("expected error")
		}
	})
}
