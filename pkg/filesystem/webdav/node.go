package webdav

import (
	"os"
	"time"
)

type node struct {
	root *webdavfs

	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}
