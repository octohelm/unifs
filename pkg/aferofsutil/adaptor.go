package aferofsutil

import (
	"context"
	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/spf13/afero"
	"io"
	"io/fs"
	"os"
	"time"
)

func From(fs filesystem.FileSystem) afero.Fs {
	return &adaptor{
		fs: fs,
	}
}

type adaptor struct {
	fs filesystem.FileSystem
}

func (a *adaptor) Create(name string) (afero.File, error) {
	return a.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

func (a *adaptor) Mkdir(name string, perm os.FileMode) error {
	return a.fs.Mkdir(context.Background(), name, perm)
}

func (a *adaptor) MkdirAll(path string, perm os.FileMode) error {
	return filesystem.MkdirAll(context.Background(), a.fs, path)
}

func (a *adaptor) Open(name string) (afero.File, error) {
	return a.OpenFile(name, os.O_RDONLY, 0)
}

func (a *adaptor) Remove(name string) error {
	return a.fs.RemoveAll(context.Background(), name)
}

func (a *adaptor) RemoveAll(path string) error {
	return a.fs.RemoveAll(context.Background(), path)
}

func (a *adaptor) Rename(oldname, newname string) error {
	return a.fs.Rename(context.Background(), oldname, newname)
}

func (a *adaptor) Stat(name string) (os.FileInfo, error) {
	return a.fs.Stat(context.Background(), name)
}

func (a adaptor) Name() string {
	return "unifs"
}

func (adaptor) Chmod(name string, mode os.FileMode) error {
	return notImplemented("chmod", name)
}

func (adaptor) Chown(name string, uid, gid int) error {
	return notImplemented("chown", name)
}

func (adaptor) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return notImplemented("chtimes", name)
}

func (a *adaptor) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	f, err := a.fs.OpenFile(context.Background(), name, flag, perm)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	return &file{File: f, info: info}, nil
}

type file struct {
	info fs.FileInfo
	filesystem.File
}

func (f *file) ReadAt(p []byte, off int64) (n int, err error) {
	readerAt, ok := f.File.(io.ReaderAt)
	if !ok {
		return -1, notImplemented("readat", f.Name())
	}
	return readerAt.ReadAt(p, off)
}

func (f *file) WriteAt(p []byte, off int64) (n int, err error) {
	writerAt, ok := f.File.(io.WriterAt)
	if !ok {
		return -1, notImplemented("writeat", f.Name())
	}
	return writerAt.WriteAt(p, off)
}

func (f *file) Name() string {
	return f.info.Name()
}

func (f *file) Readdirnames(n int) ([]string, error) {
	entries, err := f.File.Readdir(n)
	if err != nil {
		return nil, err
	}

	ret := make([]string, len(entries))
	for i := range entries {
		ret[i] = entries[i].Name()
	}
	return ret, nil
}

func (f *file) Sync() error {
	if c, ok := f.File.(filesystem.FileSyncer); ok {
		return c.Sync()
	}
	return nil
}

func (f *file) Truncate(size int64) error {
	if c, ok := f.File.(filesystem.FileTruncator); ok {
		return c.Truncate(size)
	}
	return notImplemented("truncate", f.Name())
}

func (f *file) WriteString(s string) (ret int, err error) {
	return io.WriteString(f, s)
}

func notImplemented(op, path string) error {
	return &fs.PathError{Op: op, Path: path, Err: fs.ErrPermission}
}
