package filesystem

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
)

// AsReadDirFS 将 FileSystem 适配为 fs.ReadDirFS。
func AsReadDirFS(fsys FileSystem) fs.ReadDirFS {
	return &readDirFS{fsys: fsys}
}

type readDirFS struct {
	fsys FileSystem
}

func (r *readDirFS) Open(name string) (fs.File, error) {
	file, err := r.fsys.OpenFile(context.Background(), name, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", name, err)
	}
	return &stdFile{name: path.Base(name), File: file}, nil
}

func (r *readDirFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := ReadDir(context.Background(), r.fsys, name)
	if err != nil {
		return nil, fmt.Errorf("readdir %q: %w", name, err)
	}
	ret := make([]fs.DirEntry, 0, len(entries))
	for _, entry := range entries {
		ret = append(ret, &stdDirEntry{name: path.Base(entry.Name()), DirEntry: entry})
	}
	return ret, nil
}

type stdFile struct {
	name string
	File
}

func (f *stdFile) Stat() (os.FileInfo, error) {
	info, err := f.File.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat %q: %w", f.name, err)
	}
	return &stdFileInfo{name: f.name, FileInfo: info}, nil
}

type stdDirEntry struct {
	name string
	fs.DirEntry
}

func (e *stdDirEntry) Name() string { return e.name }

func (e *stdDirEntry) Info() (fs.FileInfo, error) {
	info, err := e.DirEntry.Info()
	if err != nil {
		return nil, fmt.Errorf("dir entry info %q: %w", e.name, err)
	}
	return &stdFileInfo{name: e.name, FileInfo: info}, nil
}

type stdFileInfo struct {
	name string
	os.FileInfo
}

func (i *stdFileInfo) Name() string {
	return i.name
}
