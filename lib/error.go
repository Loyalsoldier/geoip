package lib

import "errors"

var (
	ErrDuplicatedConverter = errors.New("duplicated converter")
	ErrUnknownAction       = errors.New("unknown action")
	ErrNotSupportedFormat  = errors.New("not supported format")
	ErrInvalidIPType       = errors.New("invalid IP type")
	ErrInvalidIP           = errors.New("invalid IP address")
	ErrInvalidIPLength     = errors.New("invalid IP address length")
	ErrInvalidIPNet        = errors.New("invalid IPNet address")
	ErrInvalidCIDR         = errors.New("invalid CIDR")
	ErrInvalidPrefix       = errors.New("invalid prefix")
	ErrInvalidPrefixType   = errors.New("invalid prefix type")
	ErrCommentLine         = errors.New("comment line")
)
