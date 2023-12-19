package filesystem

import (
	"context"
	"io/fs"
	"os"
)

func AsReadDirFS(fsys FileSystem) fs.ReadDirFS {
	return &readDirFS{fsys: fsys}
}

type readDirFS struct {
	fsys FileSystem
}

func (r *readDirFS) Open(name string) (fs.File, error) {
	return r.fsys.OpenFile(context.Background(), name, os.O_RDONLY, os.ModePerm)
}

func (r *readDirFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return ReadDir(context.Background(), r.fsys, name)
}
