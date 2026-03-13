package lib

import (
	"testing"
)

func TestErrorVariables(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
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
			if tt.err.Error() != tt.want {
				t.Errorf("%s.Error() = %s, want %s", tt.name, tt.err.Error(), tt.want)
			}
		})
	}
}

func TestErrorsAreNotNil(t *testing.T) {
	errors := []error{
		ErrDuplicatedConverter,
		ErrUnknownAction,
		ErrNotSupportedFormat,
		ErrInvalidIPType,
		ErrInvalidIP,
		ErrInvalidIPLength,
		ErrInvalidIPNet,
		ErrInvalidCIDR,
		ErrInvalidPrefix,
		ErrInvalidPrefixType,
		ErrCommentLine,
	}

	for _, err := range errors {
		if err == nil {
			t.Error("Expected error to be non-nil")
		}
	}
}
