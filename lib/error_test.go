package lib

import (
	"errors"
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
			if tt.err == nil {
				t.Errorf("%s is nil", tt.name)
			}
			if tt.err.Error() != tt.expected {
				t.Errorf("%s.Error() = %q, want %q", tt.name, tt.err.Error(), tt.expected)
			}
		})
	}
}

func TestErrorsAreDistinct(t *testing.T) {
	errorList := []error{
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

	// Check that all errors are distinct
	for i, err1 := range errorList {
		for j, err2 := range errorList {
			if i != j && errors.Is(err1, err2) {
				t.Errorf("errors at index %d and %d are the same", i, j)
			}
		}
	}
}
