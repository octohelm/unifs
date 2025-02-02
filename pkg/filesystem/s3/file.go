package s3

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/octohelm/courier/pkg/courierhttp"
	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/fsutil"
	"golang.org/x/sync/errgroup"
)

func openDir(ctx context.Context, fs *fs, name string) (filesystem.File, error) {
	dir := &file{ctx: ctx, fs: fs, name: name}

	info, err := fs.Stat(ctx, name)
	if err != nil {
		if os.IsNotExist(err) {
			if parent := filepath.Dir(strings.TrimRight(name, "/")); parent != "/" {
				if _, err := fs.Stat(ctx, parent); err != nil {
					return nil, err
				}
			}

			_, err := fs.s3Client.PutObject(ctx, fs.bucket, fs.path(path.Join(name, dirHolder)), bytes.NewBuffer(nil), 0, minio.PutObjectOptions{})
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

func openFileForWrite(ctx context.Context, fs *fs, name string, flags int) (filesystem.File, error) {
	if parent := filepath.Dir(strings.TrimRight(name, "/")); parent != "/" {
		if _, err := fs.Stat(ctx, parent); err != nil {
			return nil, err
		}
	}

	f := &file{ctx: ctx, fs: fs, name: name}

	reader, writer := io.Pipe()

	f.streamWriter = writer
	f.streamWriterErrCh = make(chan error)

	putOptions := minio.PutObjectOptions{}

	metadata := filesystem.MetadataFromContext(ctx)

	if v := metadata.Get("Content-Type"); v != "" {
		putOptions.ContentType = v
	}

	if v := metadata.Get("Cache-Control"); v != "" {
		putOptions.CacheControl = v
	}

	go func() {
		var err error
		defer func() {
			f.streamWriterErrCh <- err
		}()

		if flags&os.O_CREATE != 0 {
			// when create new file
			// to put 0x00 as placeholder
			_, err := f.fs.s3Client.PutObject(f.ctx, f.fs.bucket, f.fs.path(f.name), bytes.NewBuffer([]byte{0x00}), 1, putOptions)
			if err != nil {
				_ = writer.Close()
				return
			}
		}

		_, err = f.fs.s3Client.PutObject(f.ctx, f.fs.bucket, f.fs.path(f.name), reader, -1, putOptions)
		if err != nil {
			_ = writer.Close()
		}
	}()

	if presignAs, ok := fs.presignForWrite(); ok {
		u, err := fs.presignClient().PresignedPutObject(ctx, fs.bucket, fs.path(name), 5*time.Minute)
		if err != nil {
			return nil, err
		}

		u.Scheme = presignAs.Scheme
		u.Host = presignAs.Host

		return &preSignedFile{
			file: f,
			u:    u,
		}, nil
	}

	return f, nil
}

func openFileForRead(ctx context.Context, fs *fs, name string) (filesystem.File, error) {
	f := &file{ctx: ctx, fs: fs, name: name}

	o, err := fs.s3Client.GetObject(ctx, fs.bucket, fs.path(name), minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	f.object = o

	if presignAs, ok := fs.presignForRead(); ok {
		u, err := fs.presignClient().PresignedGetObject(ctx, fs.bucket, fs.path(name), 5*time.Minute, nil)
		if err != nil {
			return nil, err
		}

		u.Scheme = presignAs.Scheme
		u.Host = presignAs.Host

		return &preSignedFile{
			file: f,
			u:    u,
		}, nil
	}

	return f, nil
}

var _ courierhttp.RedirectDescriber = &preSignedFile{}

type preSignedFile struct {
	*file
	u *url.URL
}

func (preSignedFile) StatusCode() int {
	return http.StatusTemporaryRedirect
}

func (f *preSignedFile) Location() *url.URL {
	return f.u
}

type file struct {
	ctx  context.Context
	fs   *fs
	name string

	// reader
	object *minio.Object

	// writer
	streamWriter      *io.PipeWriter
	streamWriterErrCh chan error

	mu sync.Mutex
}

func (f *file) Name() string { return f.name }

func (f *file) Readdir(n int) ([]os.FileInfo, error) {
	// ListObjects treats leading slashes as part of the directory name
	// It also needs a trailing slash to list contents of a directory.
	name := strings.TrimPrefix(f.fs.path(f.Name()), "/")

	// For the root of the bucket, we need to remove any prefix
	if name != "" && !strings.HasSuffix(name, "/") {
		name += "/"
	}

	objCh := f.fs.s3Client.ListObjects(context.Background(), f.fs.bucket, minio.ListObjectsOptions{
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
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.object.Seek(offset, whence)
}

func (f *file) Read(p []byte) (int, error) {
	if f.object == nil {
		return -1, ErrNotSupported
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.object.Read(p)
}

func (f *file) Write(p []byte) (int, error) {
	if f.streamWriter == nil {
		return -1, ErrNotSupported
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.streamWriter.Write(p)
}

func (f *file) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

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
