package lib

import (
	"errors"
	"testing"
)

func TestErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrDuplicatedConverter", ErrDuplicatedConverter},
		{"ErrUnknownAction", ErrUnknownAction},
		{"ErrNotSupportedFormat", ErrNotSupportedFormat},
		{"ErrInvalidIPType", ErrInvalidIPType},
		{"ErrInvalidIP", ErrInvalidIP},
		{"ErrInvalidIPLength", ErrInvalidIPLength},
		{"ErrInvalidIPNet", ErrInvalidIPNet},
		{"ErrInvalidCIDR", ErrInvalidCIDR},
		{"ErrInvalidPrefix", ErrInvalidPrefix},
		{"ErrInvalidPrefixType", ErrInvalidPrefixType},
		{"ErrCommentLine", ErrCommentLine},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
			if !errors.Is(tt.err, tt.err) {
				t.Errorf("%s should match itself", tt.name)
			}
		})
	}
}
