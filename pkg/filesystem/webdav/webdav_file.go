package webdav

import (
	"context"
	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/webdav/client"
	"golang.org/x/sync/errgroup"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func openDir(ctx context.Context, fs *webdavfs, name string) (filesystem.File, error) {
	dir := &file{ctx: ctx, fs: fs, name: name}

	info, err := fs.Stat(ctx, name)
	if err != nil {
		if os.IsNotExist(err) {
			if parent := filepath.Dir(strings.TrimRight(name, "/")); parent != "/" {
				if _, err := fs.Stat(ctx, parent); err != nil {
					return nil, err
				}
			}
			if err := fs.c.MkCol(ctx, name); err != nil {
				return nil, err
			}
			return dir, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, &os.PathError{
			Op:   "stat",
			Path: name,
			Err:  os.ErrExist,
		}
	}
	return dir, nil
}

func openFileForWrite(ctx context.Context, fs *webdavfs, name string) (filesystem.File, error) {
	if parent := filepath.Dir(strings.TrimRight(name, "/")); parent != "/" {
		if _, err := fs.Stat(ctx, parent); err != nil {
			return nil, err
		}
	}

	f := &file{ctx: ctx, fs: fs, name: name}

	writer, err := f.fs.c.OpenWrite(f.ctx, f.name)
	if err != nil {
		return nil, err
	}
	f.writer = writer

	return f, nil
}

func openFileForRead(ctx context.Context, fs *webdavfs, name string) (filesystem.File, error) {
	f := &file{ctx: ctx, fs: fs, name: name}

	r, err := fs.c.OpenRead(ctx, name)
	if err != nil {
		return nil, err
	}
	// support seek
	f.reader = r
	return f, nil
}

type file struct {
	ctx  context.Context
	fs   *webdavfs
	name string

	// reader
	reader io.ReadCloser

	// writer
	writer io.WriteCloser
}

func (f *file) Name() string { return f.name }

func (f *file) Readdir(n int) ([]os.FileInfo, error) {
	// ListObjects treats leading slashes as part of the directory name
	// It also needs a trailing slash to list contents of a directory.
	name := strings.TrimPrefix(f.Name(), "/")

	// For the root of the bucket, we need to remove any prefix
	if name != "" && !strings.HasSuffix(name, "/") {
		name += "/"
	}

	ms, err := f.fs.c.PropFind(f.ctx, name, 1, fileInfoPropFind)
	if err != nil {
		return nil, err
	}

	var fileInfos []os.FileInfo

	if n > 0 {
		fileInfos = make([]filesystem.FileInfo, 0, len(ms.Responses))
	}

	idx := 0

	for _, resp := range ms.Responses {
		p, err := resp.Path()
		if err != nil {
			return nil, err
		}

		if p == f.Name() || (p == f.Name()+"/") {
			continue
		}

		fi, err := resp.FileInfo()
		if err != nil {
			return nil, err
		}

		fileInfos = append(fileInfos, fi)

		idx++
		if n > 0 && idx > n {
			break
		}
	}

	return fileInfos, nil
}

func (f *file) Stat() (os.FileInfo, error) {
	return f.fs.Stat(f.ctx, f.Name())
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	return -1, ErrNotSupported
}

func (f *file) Read(p []byte) (int, error) {
	if f.reader == nil {
		return -1, ErrNotSupported
	}
	return f.reader.Read(p)
}

func (f *file) Write(p []byte) (int, error) {
	if f.writer == nil {
		return -1, ErrNotSupported
	}
	return f.writer.Write(p)
}

func (f *file) Close() error {
	eg := errgroup.Group{}

	eg.Go(func() error {
		if f.writer != nil {
			return f.writer.Close()
		}
		return nil
	})

	eg.Go(func() error {
		if f.reader != nil {
			return f.reader.Close()
		}
		return nil
	})

	return eg.Wait()
}

var fileInfoPropFind = client.NewPropNamePropFind(
	client.ResourceTypeName,
	client.GetContentLengthName,
	client.GetLastModifiedName,
	client.GetContentTypeName,
	client.GetETagName,
)
