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
	"github.com/octohelm/unifs/pkg/units"
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

	f := &file{
		name:      name,
		flags:     flags,
		ctx:       ctx,
		fs:        fs,
		writeable: true,
	}

	// wrap as pre-signed
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

func openFileForRead(ctx context.Context, fs *fs, name string, flags int) (filesystem.File, error) {
	f := &file{name: name, flags: flags, ctx: ctx, fs: fs}

	if _, err := fs.Stat(ctx, name); err != nil {
		return nil, err
	}

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
	name  string
	flags int

	ctx context.Context
	fs  *fs

	// write
	writeable     bool
	pw            *io.PipeWriter
	errCh         chan error
	writeInitOnce sync.Once

	// read
	object *minio.Object
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
	return f.object.Seek(offset, whence)
}

func (f *file) Read(p []byte) (int, error) {
	if f.object == nil {
		return -1, os.ErrNotExist
	}

	return f.object.Read(p)
}

func (f *file) Write(p []byte) (int, error) {
	if !f.writeable {
		return -1, os.ErrPermission
	}

	f.writeInitOnce.Do(func() {
		pr, pw := io.Pipe()

		f.errCh = make(chan error, 1)
		f.pw = pw

		putObjectOptions := minio.PutObjectOptions{}

		metadata := filesystem.MetadataFromContext(f.ctx)
		if v := metadata.Get("Content-Type"); v != "" {
			putObjectOptions.ContentType = v
		}
		if v := metadata.Get("Cache-Control"); v != "" {
			putObjectOptions.CacheControl = v
		}

		go func() {
			defer pr.Close()

			var err error
			defer func() {
				f.errCh <- err
			}()

			c := context.WithoutCancel(f.ctx)

			if f.flags&os.O_CREATE != 0 {
				// when create new file
				// to put 0x00 as placeholder
				_, err = f.fs.s3Client.PutObject(c, f.fs.bucket, f.fs.path(f.name), bytes.NewBuffer([]byte{0x00}), 1, putObjectOptions)
				if err != nil {
					return
				}
			}

			// https://github.com/minio/minio-go/issues?q=PartSize%20
			putObjectOptions.PartSize = uint64(5 * units.MiB)

			_, err = f.fs.s3Client.PutObject(c, f.fs.bucket, f.fs.path(f.name), pr, -1, putObjectOptions)
			return
		}()
	})

	return f.pw.Write(p)
}

func (f *file) Close() error {
	if f.pw != nil {
		if err := f.pw.Close(); err != nil {
			return err
		}
		return <-f.errCh
	}

	if f.object != nil {
		return f.object.Close()
	}

	return nil
}
