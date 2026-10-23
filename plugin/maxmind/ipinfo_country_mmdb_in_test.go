package maxmind

import "testing"

func TestNewIPInfoCountryMMDBIn(t *testing.T) {
	t.Parallel()

	testMMDBInConstructors(t, findMMDBInputType(t, TypeIPInfoCountryMMDBIn))
}
