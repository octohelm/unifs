package s3

import "errors"

var (
	ErrNotImplemented = errors.New("not implemented")
	ErrNotSupported   = errors.New("not support")
	ErrAlreadyOpened  = errors.New("already opened")
	ErrInvalidSeek    = errors.New("invalid seek")
)
