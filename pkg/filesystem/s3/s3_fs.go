package s3

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/pkg/errors"
	"golang.org/x/net/webdav"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/fsutil"
)

func NewS3FS(c *minio.Client, bucket string) filesystem.FileSystem {
	return &s3fs{
		bucket: bucket,
		c:      c,
	}
}

type s3fs struct {
	bucket string
	c      *minio.Client
}

func (fs *s3fs) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	if _, err := fs.Stat(ctx, name); err == nil {
		return &os.PathError{
			Op:   "mkdir",
			Path: name,
			Err:  os.ErrExist,
		}
	}
	f, err := fs.OpenFile(ctx, fmt.Sprintf("%s/", path.Clean(name)), os.O_CREATE, perm)
	if err != nil {
		return err
	}
	_ = f.Close()
	return nil

}

func (fs *s3fs) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	// Appending is not supported by S3. It's do-able though by:
	// - Copying the existing file to a new place (for example $file.previous)
	// - Writing a new file, streaming the content of the previous file in it
	// - Writing the data you want to append
	// Quite network intensive, if used in abondance this would lead to terrible performances.
	if flag&os.O_APPEND != 0 {
		return nil, ErrNotSupported
	}

	if flag&os.O_CREATE != 0 {
		flag |= os.O_WRONLY
	}

	if strings.HasSuffix(name, "/") {
		return openDir(ctx, fs, name)
	}

	if flag&os.O_WRONLY != 0 {
		return openFileForWrite(ctx, fs, name)
	}

	return openFileForRead(ctx, fs, name)
}

func (fs *s3fs) Rename(ctx context.Context, oldName, newName string) error {
	if newName == oldName {
		return nil
	}

	if strings.HasPrefix(newName, oldName) {
		return &os.LinkError{
			Op:  "rename",
			Old: oldName,
			New: newName,
			Err: os.ErrPermission,
		}
	}

	info, err := fs.Stat(ctx, oldName)
	if err != nil {
		return err
	}

	if info.IsDir() {
		f := &file{
			ctx:  ctx,
			fs:   fs,
			name: oldName,
		}

		fileInfos, err := f.Readdir(0)
		if err != nil {
			return err
		}

		if err := fs.Mkdir(ctx, newName, os.ModePerm); err != nil {
			return err
		}

		for _, fi := range fileInfos {
			fullPath := path.Join(f.Name(), fi.Name())
			destFullPath := path.Join(newName, fi.Name())
			if err := fs.Rename(ctx, fullPath, destFullPath); err != nil {
				return err
			}
		}

		return fs.forceRemove(ctx, oldName, true)
	}

	_, err = fs.c.CopyObject(
		ctx,
		minio.CopyDestOptions{
			Bucket: fs.bucket,
			Object: newName,
		},
		minio.CopySrcOptions{
			Bucket: fs.bucket,
			Object: oldName,
		},
	)

	if err != nil {
		return errors.Wrap(err, "copy failed")
	}

	return fs.forceRemove(ctx, oldName, false)
}

func (fs *s3fs) RemoveAll(ctx context.Context, name string) error {
	if name == "/" {
		return errors.Wrap(os.ErrPermission, "rm '/' not allow")
	}

	f := &file{
		ctx:  ctx,
		fs:   fs,
		name: name,
	}

	fileInfos, err := f.Readdir(0)
	if err != nil {
		return err
	}

	for _, fi := range fileInfos {
		fullPath := path.Join(f.Name(), fi.Name())

		if fi.IsDir() {
			if err := fs.RemoveAll(ctx, fullPath); err != nil {
				return err
			}
		} else {
			if err := fs.forceRemove(ctx, fullPath, false); err != nil {
				return err
			}
		}
	}

	if err := fs.forceRemove(ctx, path.Clean(f.Name())+"/", true); err != nil {
		return err
	}

	return nil
}

func (fs *s3fs) forceRemove(ctx context.Context, name string, isDir bool) error {
	if isDir {
		if err := fs.c.RemoveObject(ctx, fs.bucket, filepath.Join(name, dirHolder), minio.RemoveObjectOptions{
			ForceDelete: true,
		}); err != nil {
			return err
		}
	}

	return fs.c.RemoveObject(ctx, fs.bucket, name, minio.RemoveObjectOptions{
		ForceDelete: true,
	})
}

func (fs *s3fs) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	if name == "/" {
		return fsutil.NewDirFileInfo(name), nil
	}

	info, err := fs.c.StatObject(ctx, fs.bucket, name, minio.StatObjectOptions{})
	if err != nil {
		var errorResponse minio.ErrorResponse
		if errors.As(err, &errorResponse) {
			if errorResponse.StatusCode == http.StatusNotFound {
				return fs.statDirectory(ctx, name)
			}
		}

		return nil, &os.PathError{
			Op:   "stat",
			Path: name,
			Err:  err,
		}
	}

	return fsutil.NewFileInfo(
		path.Base(name),
		info.Size,
		info.LastModified,
	), nil
}

func (fs *s3fs) statDirectory(ctx context.Context, name string) (os.FileInfo, error) {
	nameClean := path.Clean(name)

	objects := fs.c.ListObjects(ctx, fs.bucket, minio.ListObjectsOptions{
		Prefix:  strings.TrimPrefix(nameClean, "/"),
		MaxKeys: 1,
	})

	for range objects {
		return fsutil.NewDirFileInfo(path.Base(name)), nil
	}

	return nil, &os.PathError{
		Op:   "stat",
		Path: name,
		Err:  os.ErrNotExist,
	}
}
