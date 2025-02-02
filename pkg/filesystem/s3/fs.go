package s3

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"golang.org/x/net/webdav"

	"github.com/octohelm/unifs/pkg/filesystem/fsutil"
)

type fs struct {
	s3Client           *minio.Client
	s3ClientForPresign *minio.Client
	presignAs          *url.URL

	bucket string
	prefix string
}

func (fsys *fs) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	if _, err := fsys.Stat(ctx, name); err == nil {
		return &os.PathError{
			Op:   "mkdir",
			Path: name,
			Err:  os.ErrExist,
		}
	}
	f, err := fsys.OpenFile(ctx, fmt.Sprintf("%s/", path.Clean(name)), os.O_CREATE, perm)
	if err != nil {
		return err
	}
	_ = f.Close()
	return nil
}

func (fsys *fs) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
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
		return openDir(ctx, fsys, name)
	}

	if flag&os.O_WRONLY != 0 {
		return openFileForWrite(ctx, fsys, name, flag)
	}

	return openFileForRead(ctx, fsys, name)
}

func (fsys *fs) Rename(ctx context.Context, oldName, newName string) error {
	if newName == oldName {
		return nil
	}

	info, err := fsys.Stat(ctx, oldName)
	if err != nil {
		return err
	}

	//  /x could not mv to its child path like /x/a/b/x
	if oldName == "/" || strings.HasPrefix(newName, oldName+"/") {
		return &os.LinkError{
			Op:  "rename",
			Old: oldName,
			New: newName,
			Err: os.ErrPermission,
		}
	}

	if info.IsDir() {
		f := &file{
			ctx:  ctx,
			fs:   fsys,
			name: oldName,
		}

		fileInfos, err := f.Readdir(0)
		if err != nil {
			return err
		}

		if err := fsys.Mkdir(ctx, newName, os.ModePerm); err != nil {
			return err
		}

		for _, fi := range fileInfos {
			fullPath := path.Join(f.Name(), fi.Name())
			destFullPath := path.Join(newName, fi.Name())
			if err := fsys.Rename(ctx, fullPath, destFullPath); err != nil {
				return err
			}
		}

		return fsys.forceRemove(ctx, oldName, true)
	}

	_, err = fsys.s3Client.CopyObject(
		ctx,
		minio.CopyDestOptions{
			Bucket: fsys.bucket,
			Object: fsys.path(newName),
		},
		minio.CopySrcOptions{
			Bucket: fsys.bucket,
			Object: fsys.path(oldName),
		},
	)
	if err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}

	return fsys.forceRemove(ctx, oldName, false)
}

func (fsys *fs) RemoveAll(ctx context.Context, name string) error {
	if name == "/" {
		return fmt.Errorf("rm '/' not allow: %w", os.ErrPermission)
	}

	f := &file{
		ctx:  ctx,
		fs:   fsys,
		name: name,
	}

	fileInfos, err := f.Readdir(0)
	if err != nil {
		return err
	}

	for _, fi := range fileInfos {
		fullPath := path.Join(f.Name(), fi.Name())

		if fi.IsDir() {
			if err := fsys.RemoveAll(ctx, fullPath); err != nil {
				return err
			}
		} else {
			if err := fsys.forceRemove(ctx, fullPath, false); err != nil {
				return err
			}
		}
	}

	if err := fsys.forceRemove(ctx, path.Clean(f.Name())+"/", true); err != nil {
		return err
	}

	return nil
}

func (fsys *fs) forceRemove(ctx context.Context, name string, isDir bool) error {
	if isDir {
		if err := fsys.s3Client.RemoveObject(ctx, fsys.bucket, fsys.path(filepath.Join(name, dirHolder)), minio.RemoveObjectOptions{
			ForceDelete: true,
		}); err != nil {
			return err
		}
	}

	return fsys.s3Client.RemoveObject(ctx, fsys.bucket, fsys.path(name), minio.RemoveObjectOptions{
		ForceDelete: true,
	})
}

func (fsys *fs) path(name string) (s string) {
	if fsys.prefix == "" || fsys.prefix == "/" {
		return strings.TrimPrefix(name, "/")
	}
	return strings.TrimPrefix(filepath.Join(fsys.prefix, name), "/")
}

func (fsys *fs) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	if name == "/" {
		return fsutil.NewDirFileInfo(name), nil
	}

	info, err := fsys.s3Client.StatObject(ctx, fsys.bucket, fsys.path(name), minio.StatObjectOptions{})
	if err != nil {
		var errorResponse minio.ErrorResponse
		if errors.As(err, &errorResponse) {
			if errorResponse.StatusCode == http.StatusNotFound {
				return fsys.statDirectory(ctx, name)
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

func (fsys *fs) statDirectory(ctx context.Context, name string) (os.FileInfo, error) {
	nameClean := path.Clean(name)

	objects := fsys.s3Client.ListObjects(ctx, fsys.bucket, minio.ListObjectsOptions{
		Prefix:  fsys.path(nameClean),
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

func (fsys *fs) presignClient() *minio.Client {
	if presignAs := fsys.presignAs; presignAs != nil {

		if presignAs.User != nil {
			if pwd, ok := presignAs.User.Password(); ok && pwd == "fake" {
				return fsys.s3Client
			}
		}
		return fsys.s3ClientForPresign
	}

	return fsys.s3Client
}

func (fsys *fs) presignForWrite() (*url.URL, bool) {
	if fsys.presignAs != nil {
		if fsys.presignAs.User != nil && fsys.presignAs.User.Username() == "rw" {
			return fsys.presignAs, true
		}
	}

	return nil, false
}

func (fsys *fs) presignForRead() (*url.URL, bool) {
	if fsys.presignAs != nil {
		return fsys.presignAs, true
	}
	return nil, false
}
