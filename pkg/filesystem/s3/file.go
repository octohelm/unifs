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

	"github.com/octohelm/courier/pkg/courierhttp"
	"github.com/rhnvrm/simples3"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/fsutil"
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

			if err := fs.putObject(ctx, path.Join(name, dirHolder), bytes.NewReader(nil), 0, nil); err != nil {
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
		u, err := fs.presignedURL(ctx, fs.presignClient(), http.MethodPut, fs.path(name), nil)
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

	info, err := fs.Stat(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("stat s3 object %q: %w", name, err)
	}

	f.object = &s3Object{
		ctx:  ctx,
		fs:   fs,
		key:  fs.path(name),
		size: info.Size(),
	}

	if presignAs, ok := fs.presignForRead(); ok {
		u, err := fs.presignedURL(ctx, fs.presignClient(), http.MethodGet, fs.path(name), nil)
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
	writeInitOnce sync.Once
	writeInitErr  error
	writeFile     *os.File

	// 读取对象
	object *s3Object
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

	objects, prefixes, err := f.fs.listObjects(f.ctx, simples3.ListInput{
		Bucket:    f.fs.bucket,
		Prefix:    name,
		Delimiter: "/",
	})
	if err != nil {
		return nil, fmt.Errorf("list s3 dir %q: %w", f.Name(), err)
	}

	var fileInfos []os.FileInfo

	if n > 0 {
		fileInfos = make([]os.FileInfo, 0, n)
	}

	idx := 0

	for _, obj := range objects {
		if strings.HasSuffix(obj.Key, dirHolder) {
			continue
		}

		fileInfos = append(fileInfos, fsutil.NewFileInfo(
			path.Base("/"+obj.Key),
			obj.Size,
			parseS3Time(obj.LastModified),
		))
		idx++

		if n > 0 && idx >= n {
			break
		}
	}

	for _, prefix := range prefixes {
		if n > 0 && idx >= n {
			break
		}

		name := path.Base(strings.TrimSuffix(prefix, "/"))
		if name == "" || name == "." || name == dirHolder {
			continue
		}

		fileInfos = append(fileInfos, fsutil.NewDirFileInfo(name))
		idx++
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
		f.writeFile, f.writeInitErr = os.CreateTemp("", "unifs-s3-*")
	})
	if f.writeInitErr != nil {
		return -1, fmt.Errorf("create s3 write temp file %q: %w", f.Name(), f.writeInitErr)
	}

	return f.writeFile.Write(p)
}

func (f *file) Close() error {
	if f.writeFile != nil {
		name := f.writeFile.Name()
		defer func() {
			_ = os.Remove(name)
		}()
		defer f.writeFile.Close()

		if _, err := f.writeFile.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("seek s3 write temp file %q: %w", f.Name(), err)
		}

		info, err := f.writeFile.Stat()
		if err != nil {
			return fmt.Errorf("stat s3 write temp file %q: %w", f.Name(), err)
		}

		if err := f.fs.putObject(context.WithoutCancel(f.ctx), f.name, f.writeFile, info.Size(), uploadHeaders(f.ctx)); err != nil {
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

func uploadHeaders(ctx context.Context) http.Header {
	headers := http.Header{}
	metadata := filesystem.MetadataFromContext(ctx)
	if v := metadata.Get("Content-Type"); v != "" {
		headers.Set("Content-Type", v)
	}
	if v := metadata.Get("Cache-Control"); v != "" {
		headers.Set("Cache-Control", v)
	}
	return headers
}
