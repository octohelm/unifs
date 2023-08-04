package webdav

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/net/webdav"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/webdav/client"
)

func NewWebdavFS(endpoint string) filesystem.FileSystem {
	return &webdavfs{
		c: client.NewClient(endpoint),
	}
}

type webdavfs struct {
	c client.Client
}

func (fs *webdavfs) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
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

func (fs *webdavfs) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	if flag&os.O_APPEND != 0 {
		return nil, errors.New("not support")
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

func (fs *webdavfs) RemoveAll(ctx context.Context, name string) error {
	if name == "/" {
		return errors.Wrap(os.ErrPermission, "rm '/' not allow")
	}
	return fs.c.Delete(ctx, name)
}

func (fs *webdavfs) Rename(ctx context.Context, oldName, newName string) error {
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

	if strings.Contains(strings.TrimLeft(newName, "/"), "/") {
		if err := fs.Mkdir(ctx, filepath.Dir(newName), os.ModeDir); err != nil {
			if !os.IsExist(err) {
				return err
			}
		}
	}

	return fs.c.Move(ctx, oldName, newName, false)
}

func (fs *webdavfs) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	ms, err := fs.c.PropFind(ctx, name, 0, fileInfoPropFind)
	if err != nil {
		return nil, err
	}

	// If the client followed a redirect, the Href might be different from the request path
	if len(ms.Responses) != 1 {
		return nil, fmt.Errorf("PROPFIND with Depth: 0 returned %d responses", len(ms.Responses))
	}

	info, err := ms.Responses[0].FileInfo()
	if err != nil {
		if client.IsNotFound(err) {
			return nil, &os.PathError{
				Op:   "stat",
				Path: name,
				Err:  os.ErrNotExist,
			}
		}
		return nil, err
	}

	return info, nil
}
