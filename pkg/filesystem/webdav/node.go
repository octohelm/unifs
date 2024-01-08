package webdav

import (
	"os"
	"time"
)

type node struct {
	root *fs

	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}
