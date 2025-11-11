package filesystem

import (
	"os"

	"golang.org/x/net/webdav"

	contextx "github.com/octohelm/x/context"
)

type (
	FileSystem = webdav.FileSystem
	File       = webdav.File
	FileInfo   = os.FileInfo
)

var Context = contextx.New[FileSystem]()
