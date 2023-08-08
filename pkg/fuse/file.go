package fuse

import (
	"context"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/octohelm/unifs/pkg/filesystem"
	"io"
	"syscall"
)

type File interface {
	fs.FileHandle

	fs.FileReader
	fs.FileWriter
	fs.FileReleaser
	fs.FileFsyncer
}

var _ File = &file{}

type file struct {
	f filesystem.File
}

func (f *file) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	if off > 0 {
		if _, err := f.f.Seek(off, 0); err != nil {
			return nil, syscall.ENOENT
		}
	}
	n, err := f.f.Read(dest)
	if err != nil {
		if err != io.EOF {
			return nil, syscall.ENOENT
		}
	}
	return fuse.ReadResultData(dest[:n]), 0
}

const maxInt = int(^uint(0) >> 1)

func (f *file) Write(ctx context.Context, data []byte, off int64) (written uint32, errno syscall.Errno) {
	if len(data) == 0 {
		return 0, 0
	}

	newLen := off + int64(len(data))

	if newLen > int64(maxInt) {
		return 0, syscall.EFBIG
	}

	n, err := f.f.Write(data)
	if err != nil {
		return 0, fs.ToErrno(err)
	}
	return uint32(n), 0
}

func (f *file) Release(ctx context.Context) syscall.Errno {
	if err := f.f.Close(); err != nil {
		return fs.ToErrno(err)
	}
	return 0
}

func (f *file) Fsync(ctx context.Context, flags uint32) syscall.Errno {
	return 0
}
