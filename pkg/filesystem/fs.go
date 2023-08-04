package filesystem

import (
	"golang.org/x/net/webdav"
	"os"
)

type FileSystem = webdav.FileSystem
type File = webdav.File
type FileInfo = os.FileInfo
