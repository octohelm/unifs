package filesystem

import (
	"os"

	contextx "github.com/octohelm/x/context"
	"golang.org/x/net/webdav"
)

type FileSystem = webdav.FileSystem
type File = webdav.File
type FileInfo = os.FileInfo

var Context = contextx.New[FileSystem]()
