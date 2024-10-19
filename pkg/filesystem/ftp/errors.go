package ftp

import (
	"errors"
	"fmt"
	"net/textproto"
	"os"

	"github.com/jlaffaye/ftp"
)

func normalizeError(op string, path string, err error, values ...any) error {
	tpErr := &textproto.Error{}
	if errors.As(err, &tpErr) {
		switch tpErr.Code {
		case ftp.StatusFileUnavailable, ftp.StatusFileActionIgnored:
			err = os.ErrNotExist
		case ftp.StatusPathCreated:
			err = os.ErrExist
		}
	}

	if len(values) > 0 {
		return &os.PathError{
			Op:   op,
			Path: path,
			Err:  fmt.Errorf("%v: %w", values, err),
		}
	}

	return &os.PathError{
		Op:   op,
		Path: path,
		Err:  err,
	}
}
