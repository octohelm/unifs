package filesystem

import (
	"golang.org/x/net/webdav"
)

// NewMemFS 返回内存 FileSystem。
func NewMemFS() FileSystem {
	return webdav.NewMemFS()
}
