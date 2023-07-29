package proto

import (
	"errors"
	"io"
)

// Errors in encoding and decoding Protobuf wiretypes.
const (
	_ = -iota
	ErrCodeTruncated
	ErrCodeFieldNumber
	ErrCodeOverflow
	ErrCodeReserved
	ErrCodeEndGroup
	ErrCodeRecursionDepth
)

var (
	errFieldNumber = errors.New("invalid field number")
	errOverflow    = errors.New("variable length integer overflow")
	errReserved    = errors.New("cannot parse reserved wire type")
	errEndGroup    = errors.New("mismatching end group marker")
	errParse       = errors.New("parse error")
)

// ParseError converts an error code into an error value.
// This returns nil if n is a non-negative number.
func ParseError(n int) error {
	if n >= 0 {
		return nil
	}
	switch n {
	case ErrCodeTruncated:
		return io.ErrUnexpectedEOF
	case ErrCodeFieldNumber:
		return errFieldNumber
	case ErrCodeOverflow:
		return errOverflow
	case ErrCodeReserved:
		return errReserved
	case ErrCodeEndGroup:
		return errEndGroup
	default:
		return errParse
	}
}