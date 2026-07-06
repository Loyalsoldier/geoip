package special

import (
	"errors"
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

type mockContainer struct {
	lookupFn func(ipOrCidr string, searchList ...string) ([]string, bool, error)
}

func (m mockContainer) GetEntry(name string) (*lib.Entry, bool) {
	return nil, false
}

func (m mockContainer) Len() int {
	return 0
}

func (m mockContainer) Add(entry *lib.Entry, opts ...lib.IgnoreIPOption) error {
	return nil
}

func (m mockContainer) Remove(entry *lib.Entry, rCase lib.CaseRemove, opts ...lib.IgnoreIPOption) error {
	return nil
}

func (m mockContainer) Loop() <-chan *lib.Entry {
	ch := make(chan *lib.Entry)
	close(ch)
	return ch
}

func (m mockContainer) Lookup(ipOrCidr string, searchList ...string) ([]string, bool, error) {
	return m.lookupFn(ipOrCidr, searchList...)
}

func TestLookupOutputReturnsContainerLookupError(t *testing.T) {
	wantErr := errors.New("lookup failure")
	l := &Lookup{
		Type:        TypeLookup,
		Action:      lib.ActionOutput,
		Description: DescLookup,
		Search:      "1.1.1.1",
	}

	container := mockContainer{
		lookupFn: func(ipOrCidr string, searchList ...string) ([]string, bool, error) {
			return nil, false, wantErr
		},
	}

	if err := l.Output(container); !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}
