package webdav

import "github.com/pkg/errors"

var (
	ErrNotImplemented = errors.New("not implemented")
	ErrNotSupported   = errors.New("s3 doesn't support this operation")
	ErrAlreadyOpened  = errors.New("already opened")
	ErrInvalidSeek    = errors.New("invalid seek")
)
