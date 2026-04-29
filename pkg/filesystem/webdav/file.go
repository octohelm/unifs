package webdav

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/webdav/client"
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
	// PROPFIND 会把前导斜杠视为目录名的一部分；
	// 列出目录内容时也需要尾随斜杠。
	name := strings.TrimPrefix(f.Name(), "/")

	// 对根目录，需要移除所有 prefix。
	if name != "" && !strings.HasSuffix(name, "/") {
		name += "/"
	}

	ms, err := f.c().PropFind(context.Background(), name, 1, client.FileInfoPropFind)
	if err != nil {
		return nil, fmt.Errorf("propfind %q: %w", name, err)
	}

	var fileInfos []os.FileInfo

	if n > 0 {
		fileInfos = make([]filesystem.FileInfo, 0, len(ms.Responses))
	}

	idx := 0

	for _, resp := range ms.Responses {
		p, err := resp.Path()
		if err != nil {
			return nil, fmt.Errorf("response path: %w", err)
		}

		normalizedPath := strings.Trim(strings.TrimPrefix(p, "/"), "/")
		normalizedName := strings.Trim(strings.TrimPrefix(f.Name(), "/"), "/")
		if normalizedPath == normalizedName {
			continue
		}

		fi, err := resp.FileInfo()
		if err != nil {
			return nil, fmt.Errorf("response fileinfo %q: %w", p, err)
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
	// TODO 使用已缓存的节点信息。
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
