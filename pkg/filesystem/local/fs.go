package local

import (
	"github.com/octohelm/unifs/pkg/filesystem"
	"golang.org/x/net/webdav"
)

func NewFS(prefix string) filesystem.FileSystem {
	return webdav.Dir(prefix)
}
