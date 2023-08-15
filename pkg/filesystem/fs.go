package filesystem

import (
	"os"

	"golang.org/x/net/webdav"
)

type FileSystem = webdav.FileSystem
type File = webdav.File
type FileInfo = os.FileInfo
