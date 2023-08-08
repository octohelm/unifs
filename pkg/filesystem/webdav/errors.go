package webdav

import "github.com/pkg/errors"

var (
	ErrNotImplemented = errors.New("not implemented")
	ErrNotSupported   = errors.New("not support")
	ErrAlreadyOpened  = errors.New("already opened")
	ErrInvalidSeek    = errors.New("invalid seek")
)
