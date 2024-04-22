package filesystem

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/webdav"
)

func Sub(source FileSystem, dir string) FileSystem {
	if dir == "/" || dir == "" {
		return source
	}

	return &subFS{
		source: source,
		dir:    dir,
	}
}

type subFS struct {
	source FileSystem
	dir    string
}

func (f *subFS) shorten(name string) (rel string, ok bool) {
	if name == f.dir {
		return ".", true
	}
	if len(name) >= len(f.dir)+2 && name[len(f.dir)] == '/' && name[:len(f.dir)] == f.dir {
		return name[len(f.dir)+1:], true
	}
	return "", false
}

// fixErr shortens any reported names in PathErrors by stripping f.dir.
func (f *subFS) fixErr(err error) error {
	var e *fs.PathError
	if errors.As(err, &e) {
		if short, ok := f.shorten(e.Path); ok {
			e.Path = short
		}
	}
	return err
}

func (f *subFS) fullName(op, name string) (string, error) {
	if strings.HasPrefix(name, "/") {
		name = name[1:]
	}
	if name == "" {
		name = "."
	}
	if !fs.ValidPath(name) {
		return "", &fs.PathError{Op: op, Path: name, Err: errors.New("invalid name")}
	}
	return filepath.Join(f.dir, name), nil
}

func (f *subFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	fullName, err := f.fullName("mkdir", name)
	if err != nil {
		return err
	}
	return f.fixErr(f.source.Mkdir(ctx, fullName, perm))
}

func (f *subFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	fullName, err := f.fullName("openfile", name)
	if err != nil {
		return nil, err
	}
	file, err := f.source.OpenFile(ctx, fullName, flag, perm)
	return file, f.fixErr(err)
}

func (f *subFS) RemoveAll(ctx context.Context, name string) error {
	fixed, err := f.fullName("remove_all", name)
	if err != nil {
		return err
	}
	return f.fixErr(f.source.RemoveAll(ctx, fixed))
}

func (f *subFS) Rename(ctx context.Context, oldName, newName string) error {
	oldFullName, err := f.fullName("rename", oldName)
	if err != nil {
		return err
	}
	newFullName, err := f.fullName("rename", newName)
	if err != nil {
		return err
	}
	return f.fixErr(f.source.Rename(ctx, oldFullName, newFullName))
}

func (f *subFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	fullName, err := f.fullName("stat", name)
	if err != nil {
		return nil, err
	}
	info, err := f.source.Stat(ctx, fullName)
	return info, f.fixErr(err)
}
