package fsutil

import (
	"io/fs"
	"time"

	"github.com/octohelm/unifs/pkg/filesystem"
)

// NewDirFileInfo 返回目录的 FileInfo。
func NewDirFileInfo(name string) filesystem.FileInfo {
	return &fileInfo{
		name:      name,
		directory: true,
		modTime:   time.Unix(0, 0),
	}
}

// NewFileInfo 返回普通文件的 FileInfo。
func NewFileInfo(name string, size int64, modTime time.Time) filesystem.FileInfo {
	return &fileInfo{
		name:    name,
		modTime: modTime,
		size:    size,
	}
}

type fileInfo struct {
	directory bool
	name      string
	modTime   time.Time
	size      int64
}

func (fi *fileInfo) Name() string {
	return fi.name
}

func (fi *fileInfo) Size() int64 {
	return fi.size
}

func (fi *fileInfo) Mode() fs.FileMode {
	if fi.directory {
		return 0o755
	}
	return 0o664
}

func (fi *fileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi *fileInfo) IsDir() bool {
	return fi.directory
}

func (fi *fileInfo) Sys() any {
	return nil
}
