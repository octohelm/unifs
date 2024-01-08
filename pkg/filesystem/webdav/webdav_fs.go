package webdav

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"golang.org/x/net/webdav"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/webdav/client"
)

func NewFS(c client.Client) filesystem.FileSystem {
	return &fs{
		c: c,
	}
}

type fs struct {
	c client.Client
}

func (fs *fs) addNode(fi filesystem.FileInfo) *node {
	return &node{
		root:    fs,
		name:    fi.Name(),
		mode:    fi.Mode(),
		size:    fi.Size(),
		modTime: fi.ModTime(),
	}
}

func (fs *fs) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
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

func (fs *fs) RemoveAll(ctx context.Context, name string) error {
	if name == "/" {
		return errors.Wrap(os.ErrPermission, "rm '/' not allow")
	}
	return fs.c.Delete(ctx, name)
}

func (fs *fs) Rename(ctx context.Context, oldName, newName string) error {
	if newName == oldName {
		return nil
	}
	return fs.c.Move(ctx, oldName, newName, false)
}

func (fs *fs) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	ms, err := fs.c.PropFind(ctx, name, 0, client.FileInfoPropFind)
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

func (fs *fs) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	if flag&os.O_APPEND != 0 {
		return nil, ErrNotSupported
	}

	if flag&os.O_CREATE != 0 {
		flag |= os.O_WRONLY
	}

	if strings.HasSuffix(name, "/") {
		return fs.openDir(ctx, name)
	}

	return fs.openFile(ctx, name, flag)
}

func (fs *fs) openDir(ctx context.Context, name string) (filesystem.File, error) {
	fi, err := fs.Stat(ctx, name)
	if err != nil {
		if os.IsNotExist(err) {
			if parent := filepath.Dir(strings.TrimRight(name, "/")); parent != "/" {
				if _, err := fs.Stat(ctx, parent); err != nil {
					return nil, err
				}
			}
			if err := fs.c.MkCol(ctx, name); err != nil {
				return nil, err
			}
			fi, err := fs.Stat(ctx, name)
			if err != nil {
				return nil, err
			}
			return &file{node: fs.addNode(fi)}, nil
		}
		return nil, err
	}
	if !fi.IsDir() {
		return nil, &os.PathError{
			Op:   "stat",
			Path: name,
			Err:  os.ErrExist,
		}
	}
	return &file{node: fs.addNode(fi)}, nil
}

func (fs *fs) openFile(ctx context.Context, name string, flag int) (filesystem.File, error) {
	// check parent path when create
	if flag&os.O_CREATE != 0 {
		if parent := filepath.Dir(strings.TrimRight(name, "/")); parent != "/" {
			if _, err := fs.Stat(ctx, parent); err != nil {
				return nil, err
			}
		}
	}

	f := &file{
		node: &node{
			root: fs,
			name: name,
		},
	}

	if flag&os.O_WRONLY != 0 || flag&os.O_RDWR != 0 {
		w, err := fs.c.OpenWrite(context.Background(), f.Name())
		if err != nil {
			return nil, err
		}
		f.writer = w
	} else {
		ff, err := fs.c.Open(context.Background(), f.Name())
		if err != nil {
			return nil, err
		}
		f.file = ff
	}

	return f, nil
}
