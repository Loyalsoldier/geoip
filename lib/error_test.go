package lib

import (
	"testing"
)

func TestErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrDuplicatedConverter", ErrDuplicatedConverter, "duplicated converter"},
		{"ErrUnknownAction", ErrUnknownAction, "unknown action"},
		{"ErrNotSupportedFormat", ErrNotSupportedFormat, "not supported format"},
		{"ErrInvalidIPType", ErrInvalidIPType, "invalid IP type"},
		{"ErrInvalidIP", ErrInvalidIP, "invalid IP address"},
		{"ErrInvalidIPLength", ErrInvalidIPLength, "invalid IP address length"},
		{"ErrInvalidIPNet", ErrInvalidIPNet, "invalid IPNet address"},
		{"ErrInvalidCIDR", ErrInvalidCIDR, "invalid CIDR"},
		{"ErrInvalidPrefix", ErrInvalidPrefix, "invalid prefix"},
		{"ErrInvalidPrefixType", ErrInvalidPrefixType, "invalid prefix type"},
		{"ErrCommentLine", ErrCommentLine, "comment line"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("%s.Error() = %q, expected %q", tt.name, tt.err.Error(), tt.expected)
			}
		})
	}
}
