package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
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
			if parent := path.Dir(strings.TrimRight(name, "/")); parent != "/" {
				if _, err := fs.Stat(ctx, parent); err != nil {
					return nil, fmt.Errorf("stat parent dir %q: %w", parent, err)
				}
			}

			_, err := fs.s3Client.PutObject(ctx, fs.bucket, fs.path(path.Join(name, dirHolder)), bytes.NewBuffer(nil), 0, minio.PutObjectOptions{})
			if err != nil {
				return nil, fmt.Errorf("put s3 dir holder %q: %w", path.Join(name, dirHolder), err)
			}
			return dir, nil
		}
		return nil, fmt.Errorf("stat s3 dir %q: %w", name, err)
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
	if parent := path.Dir(strings.TrimRight(name, "/")); parent != "/" {
		if _, err := fs.Stat(ctx, parent); err != nil {
			return nil, fmt.Errorf("stat parent dir %q: %w", parent, err)
		}
	}

	f := &file{
		name:      name,
		flags:     flags,
		ctx:       ctx,
		fs:        fs,
		writeable: true,
	}

	// 按配置将写入请求封装为预签名地址。
	if presignAs, ok := fs.presignForWrite(); ok {
		u, err := fs.presignClient().PresignedPutObject(ctx, fs.bucket, fs.path(name), 5*time.Minute)
		if err != nil {
			return nil, fmt.Errorf("presign put object %q: %w", name, err)
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
		return nil, fmt.Errorf("stat s3 object %q: %w", name, err)
	}

	o, err := fs.s3Client.GetObject(ctx, fs.bucket, fs.path(name), minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get s3 object %q: %w", name, err)
	}

	f.object = o

	if presignAs, ok := fs.presignForRead(); ok {
		u, err := fs.presignClient().PresignedGetObject(ctx, fs.bucket, fs.path(name), 5*time.Minute, nil)
		if err != nil {
			return nil, fmt.Errorf("presign get object %q: %w", name, err)
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

	// 写入对象
	writeable     bool
	pw            *io.PipeWriter
	errCh         chan error
	writeInitOnce sync.Once

	// 读取对象
	object *minio.Object
}

func (f *file) Name() string { return f.name }

func (f *file) Readdir(n int) ([]os.FileInfo, error) {
	// ListObjects 会把前导斜杠视为目录名的一部分；
	// 列出目录内容时也需要尾随斜杠。
	name := strings.TrimPrefix(f.fs.path(f.Name()), "/")

	// 对 bucket 根目录，需要移除所有 prefix。
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
			return nil, fmt.Errorf("list s3 dir %q: %w", f.Name(), obj.Err)
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
				// 创建新文件时写入 0x00 作为占位符。
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
			return fmt.Errorf("close s3 write pipe %q: %w", f.Name(), err)
		}
		if err := <-f.errCh; err != nil {
			return fmt.Errorf("put s3 object %q: %w", f.Name(), err)
		}
		return nil
	}

	if f.object != nil {
		if err := f.object.Close(); err != nil {
			return fmt.Errorf("close s3 object %q: %w", f.Name(), err)
		}
	}

	return nil
}
