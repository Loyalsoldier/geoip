package lib

import (
	"fmt"
	"testing"
)

func TestContainerLoopSnapshot(t *testing.T) {
	for _, size := range []int{0, 1, 301, 1000} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			container := NewContainer()
			for i := 0; i < size; i++ {
				if err := container.Add(NewEntry(fmt.Sprint(i))); err != nil {
					t.Fatal(err)
				}
			}

			entries := container.Loop()
			if got := len(entries); got != size || cap(entries) != size {
				for range entries {
				}
				t.Fatalf("Loop returned len=%d cap=%d, want %d buffered entries", got, cap(entries), size)
			}

			if err := container.Add(NewEntry("later")); err != nil {
				t.Fatal(err)
			}
			seen := make(map[string]bool)
			for entry := range entries {
				name := entry.GetName()
				if name == "LATER" || seen[name] {
					t.Fatalf("unexpected snapshot entry %q", name)
				}
				seen[name] = true
				if err := container.Remove(entry, CaseRemoveEntry); err != nil {
					t.Fatal(err)
				}
			}
			if len(seen) != size || container.Len() != 1 {
				t.Fatalf("visited %d entries, %d remain; want %d visited and 1 remaining", len(seen), container.Len(), size)
			}
		})
	}
}
