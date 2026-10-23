package maxmind

import "testing"

func TestNewDBIPCountryMMDBIn(t *testing.T) {
	t.Parallel()

	testMMDBInConstructors(t, findMMDBInputType(t, TypeDBIPCountryMMDBIn))
}
