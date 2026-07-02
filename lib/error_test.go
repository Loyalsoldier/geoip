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
		{
			name:     "ErrDuplicatedConverter",
			err:      ErrDuplicatedConverter,
			expected: "duplicated converter",
		},
		{
			name:     "ErrUnknownAction",
			err:      ErrUnknownAction,
			expected: "unknown action",
		},
		{
			name:     "ErrNotSupportedFormat",
			err:      ErrNotSupportedFormat,
			expected: "not supported format",
		},
		{
			name:     "ErrInvalidIPType",
			err:      ErrInvalidIPType,
			expected: "invalid IP type",
		},
		{
			name:     "ErrInvalidIP",
			err:      ErrInvalidIP,
			expected: "invalid IP address",
		},
		{
			name:     "ErrInvalidIPLength",
			err:      ErrInvalidIPLength,
			expected: "invalid IP address length",
		},
		{
			name:     "ErrInvalidIPNet",
			err:      ErrInvalidIPNet,
			expected: "invalid IPNet address",
		},
		{
			name:     "ErrInvalidCIDR",
			err:      ErrInvalidCIDR,
			expected: "invalid CIDR",
		},
		{
			name:     "ErrInvalidPrefix",
			err:      ErrInvalidPrefix,
			expected: "invalid prefix",
		},
		{
			name:     "ErrInvalidPrefixType",
			err:      ErrInvalidPrefixType,
			expected: "invalid prefix type",
		},
		{
			name:     "ErrCommentLine",
			err:      ErrCommentLine,
			expected: "comment line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("Error message mismatch. Expected: %s, Got: %s", tt.expected, tt.err.Error())
			}

			// Test that errors are proper error types
			if !errors.Is(tt.err, tt.err) {
				t.Errorf("Error should be comparable with itself using errors.Is")
			}
		})
	}
}

func TestErrorTypes(t *testing.T) {
	// Test that all defined errors implement the error interface
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

	for i, err := range errorList {
		if err == nil {
			t.Errorf("Error at index %d should not be nil", i)
		}

		if err.Error() == "" {
			t.Errorf("Error at index %d should have a non-empty error message", i)
		}
	}
}