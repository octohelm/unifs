package local

import "os"

func NormalizeError(err error, prefix string) error {
	switch x := err.(type) {
	case *os.PathError:
		return &os.PathError{
			Op:   x.Op,
			Err:  x.Err,
			Path: x.Path[len(prefix):],
		}
	case *os.LinkError:
		return &os.LinkError{
			Op:  x.Op,
			Err: x.Err,
			Old: x.Old[len(prefix):],
			New: x.New[len(prefix):],
		}
	default:
		return x
	}
}
