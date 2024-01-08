package ftp

import (
	"github.com/jlaffaye/ftp"
	"net/textproto"
	"os"

	"github.com/pkg/errors"
)

func normalizeError(op string, path string, err error, values ...any) error {
	if tpErr, ok := err.(*textproto.Error); ok {
		switch tpErr.Code {
		case ftp.StatusFileUnavailable:
			err = os.ErrNotExist
		case ftp.StatusPathCreated:
			err = os.ErrExist
		}
	}

	if len(values) > 0 {
		return &os.PathError{
			Op:   op,
			Path: path,
			Err:  errors.Wrapf(err, "%v", values),
		}
	}

	return &os.PathError{
		Op:   op,
		Path: path,
		Err:  err,
	}
}
