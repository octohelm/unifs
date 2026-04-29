package s3

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/minio/minio-go/v7"

	"github.com/octohelm/unifs/pkg/filesystem"
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
		return fmt.Errorf("open s3 dir %q: %w", name, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close s3 dir %q: %w", name, err)
	}
	return nil
}

func (fsys *fs) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (filesystem.File, error) {
	// S3 不支持追加写。理论上可以通过以下步骤模拟：
	// - 将现有文件复制到新位置，例如 $file.previous
	// - 写入新文件，并把旧文件内容流式写入其中
	// - 写入需要追加的数据
	// 该方式网络开销很高，大量使用会导致性能很差。
	if flag&os.O_APPEND != 0 {
		return nil, os.ErrPermission
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

	f, err := openFileForRead(ctx, fsys, name, flag)
	if err != nil {
		return nil, fmt.Errorf("open s3 object %q for read: %w", name, err)
	}
	return f, nil
}

func (fsys *fs) Rename(ctx context.Context, oldName, newName string) error {
	if newName == oldName {
		return nil
	}

	info, err := fsys.Stat(ctx, oldName)
	if err != nil {
		return fmt.Errorf("stat rename source %q: %w", oldName, err)
	}

	// /x 不能移动到 /x/a/b/x 这类子路径。
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
			return fmt.Errorf("readdir rename source %q: %w", oldName, err)
		}

		if err := fsys.Mkdir(ctx, newName, os.ModePerm); err != nil {
			return fmt.Errorf("mkdir rename destination %q: %w", newName, err)
		}

		for _, fi := range fileInfos {
			fullPath := path.Join(f.Name(), fi.Name())
			destFullPath := path.Join(newName, fi.Name())
			if err := fsys.Rename(ctx, fullPath, destFullPath); err != nil {
				return fmt.Errorf("rename child %q to %q: %w", fullPath, destFullPath, err)
			}
		}

		if err := fsys.forceRemove(ctx, oldName, true); err != nil {
			return fmt.Errorf("remove renamed source dir %q: %w", oldName, err)
		}
		return nil
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
		return fmt.Errorf("copy s3 object %q to %q: %w", oldName, newName, err)
	}

	if err := fsys.forceRemove(ctx, oldName, false); err != nil {
		return fmt.Errorf("remove renamed source file %q: %w", oldName, err)
	}
	return nil
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
		return fmt.Errorf("readdir remove %q: %w", name, err)
	}

	for _, fi := range fileInfos {
		fullPath := path.Join(f.Name(), fi.Name())

		if fi.IsDir() {
			if err := fsys.RemoveAll(ctx, fullPath); err != nil {
				return fmt.Errorf("remove child dir %q: %w", fullPath, err)
			}
		} else {
			if err := fsys.forceRemove(ctx, fullPath, false); err != nil {
				return fmt.Errorf("remove child file %q: %w", fullPath, err)
			}
		}
	}

	if err := fsys.forceRemove(ctx, path.Clean(f.Name())+"/", true); err != nil {
		return fmt.Errorf("remove dir marker %q: %w", name, err)
	}

	return nil
}

func (fsys *fs) forceRemove(ctx context.Context, name string, isDir bool) error {
	if isDir {
		if err := fsys.s3Client.RemoveObject(ctx, fsys.bucket, fsys.path(path.Join(name, dirHolder)), minio.RemoveObjectOptions{
			ForceDelete: true,
		}); err != nil {
			return fmt.Errorf("remove s3 dir holder %q: %w", path.Join(name, dirHolder), err)
		}
	}

	if err := fsys.s3Client.RemoveObject(ctx, fsys.bucket, fsys.path(name), minio.RemoveObjectOptions{
		ForceDelete: true,
	}); err != nil {
		return fmt.Errorf("remove s3 object %q: %w", name, err)
	}
	return nil
}

func (fsys *fs) path(name string) (s string) {
	if fsys.prefix == "" || fsys.prefix == "/" {
		return strings.TrimPrefix(name, "/")
	}
	return strings.TrimPrefix(path.Join(fsys.prefix, name), "/")
}

func (fsys *fs) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	if name == "/" || name == "." {
		return fsutil.NewDirFileInfo("/"), nil
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
