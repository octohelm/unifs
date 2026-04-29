package ftp

import (
	"errors"
	"fmt"
	"net/textproto"
	"os"
	"strings"

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
		wrapped := fmt.Errorf("%s: %w", strings.TrimSpace(fmt.Sprintln(values...)), err)
		if len(values) == 2 {
			if key, ok := values[0].(string); ok {
				wrapped = fmt.Errorf("%s %s: %w", key, fmt.Sprint(values[1]), err)
			}
		}
		return &os.PathError{
			Op:   op,
			Path: path,
			Err:  wrapped,
		}
	}

	return &os.PathError{
		Op:   op,
		Path: path,
		Err:  err,
	}
}
