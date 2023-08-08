package webdav

import (
	"context"
	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/webdav/client"
	"golang.org/x/sync/errgroup"
	"io"
	"os"
	"strings"
)

type file struct {
	node   *node
	writer io.WriteCloser
	file   client.File
}

func (f *file) c() client.Client {
	return f.node.root.c
}

func (f *file) Name() string { return f.node.name }

func (f *file) Readdir(n int) ([]os.FileInfo, error) {
	// ListObjects treats leading slashes as part of the directory name
	// It also needs a trailing slash to list contents of a directory.
	name := strings.TrimPrefix(f.Name(), "/")

	// For the root of the bucket, we need to remove any prefix
	if name != "" && !strings.HasSuffix(name, "/") {
		name += "/"
	}

	ms, err := f.c().PropFind(context.Background(), name, 1, client.FileInfoPropFind)
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
	// TODO use the cached
	return f.node.root.Stat(context.Background(), f.Name())
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
		if f.file != nil {
			return f.file.Close()
		}
		return nil
	})

	return eg.Wait()
}

func (f *file) Write(p []byte) (int, error) {
	if f.writer == nil {
		return 0, os.ErrInvalid
	}
	return f.writer.Write(p)
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	return f.file.Seek(offset, whence)
}

func (f *file) Read(p []byte) (n int, err error) {
	if f.file == nil {
		return 0, os.ErrInvalid
	}
	return f.file.Read(p)
}
