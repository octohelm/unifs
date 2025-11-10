package filesystem

import (
	"golang.org/x/net/webdav"
)

func NewMemFS() FileSystem {
	return webdav.NewMemFS()
}
