package s3

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/fsutil"
)

func openDir(ctx context.Context, fs *s3fs, name string) (filesystem.File, error) {
	dir := &file{ctx: ctx, fs: fs, name: name}

	info, err := fs.Stat(ctx, name)
	if err != nil {
		if os.IsNotExist(err) {
			if parent := filepath.Dir(strings.TrimRight(name, "/")); parent != "/" {
				if _, err := fs.Stat(ctx, parent); err != nil {
					return nil, err
				}
			}

			_, err := fs.c.PutObject(ctx, fs.bucket, path.Join(name, dirHolder), bytes.NewBuffer(nil), 0, minio.PutObjectOptions{})
			if err != nil {
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

const dirHolder = ".fs_dir"

func openFileForWrite(ctx context.Context, fs *s3fs, name string) (filesystem.File, error) {
	if parent := filepath.Dir(strings.TrimRight(name, "/")); parent != "/" {
		if _, err := fs.Stat(ctx, parent); err != nil {
			return nil, err
		}
	}

	f := &file{ctx: ctx, fs: fs, name: name}

	reader, writer := io.Pipe()

	f.streamWriter = writer
	f.streamWriterErrCh = make(chan error)

	go func() {
		_, err := f.fs.c.PutObject(f.ctx, f.fs.bucket, f.name, reader, -1, minio.PutObjectOptions{})
		if err != nil {
			_ = writer.Close()
		}
		f.streamWriterErrCh <- errors.Wrapf(err, "write %s failed", f.name)
	}()

	return f, nil
}

func openFileForRead(ctx context.Context, fs *s3fs, name string) (filesystem.File, error) {
	f := &file{ctx: ctx, fs: fs, name: name}

	o, err := fs.c.GetObject(ctx, fs.bucket, name, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	f.object = o

	return f, nil
}

type file struct {
	ctx  context.Context
	fs   *s3fs
	name string

	// reader
	object *minio.Object

	// writer
	streamWriter      *io.PipeWriter
	streamWriterErrCh chan error
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

	objCh := f.fs.c.ListObjects(context.Background(), f.fs.bucket, minio.ListObjectsOptions{
		Prefix: name,
	})

	var fileInfos []os.FileInfo

	if n > 0 {
		fileInfos = make([]os.FileInfo, 0, n)
	}

	idx := 0

	for obj := range objCh {
		if obj.Err != nil {
			return nil, obj.Err
		}

		if strings.HasSuffix(obj.Key, dirHolder) {
			continue
		}

		var fi filesystem.FileInfo

		if strings.HasSuffix(obj.Key, "/") {
			fi = fsutil.NewDirFileInfo(path.Base("/" + obj.Key))
		} else {
			fi = fsutil.NewFileInfo(
				path.Base("/"+obj.Key),
				obj.Size,
				obj.LastModified,
			)
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
	return f.object.Seek(offset, whence)
}

func (f *file) Read(p []byte) (int, error) {
	if f.object == nil {
		return -1, ErrNotSupported
	}
	return f.object.Read(p)
}

func (f *file) Write(p []byte) (int, error) {
	if f.streamWriter == nil {
		return -1, ErrNotSupported
	}
	return f.streamWriter.Write(p)
}

func (f *file) Close() error {
	eg := errgroup.Group{}

	eg.Go(func() error {
		if f.streamWriter != nil {
			if err := f.streamWriter.Close(); err != nil {
				return err
			}
			err := <-f.streamWriterErrCh
			close(f.streamWriterErrCh)
			return err
		}
		return nil
	})

	eg.Go(func() error {
		if f.object != nil {
			return f.object.Close()
		}
		return nil
	})

	return eg.Wait()
}
