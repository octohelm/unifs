package filesystem

import (
	"os"

	"golang.org/x/net/webdav"

	contextx "github.com/octohelm/x/context"
)

type (
	// FileSystem 是 unifs 后端共用的文件系统接口。
	FileSystem = webdav.FileSystem
	// File 是 FileSystem 返回的共用文件接口。
	File = webdav.File
	// FileInfo 描述文件或目录。
	FileInfo = os.FileInfo
)

// Context 在上下文中存取 FileSystem，用于依赖注入。
var Context = contextx.New[FileSystem]()
