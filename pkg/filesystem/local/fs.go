package local

import (
	"golang.org/x/net/webdav"

	"github.com/octohelm/unifs/pkg/filesystem"
)

func NewFS(prefix string) filesystem.FileSystem {
	return webdav.Dir(prefix)
}
