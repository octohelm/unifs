package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/net/webdav"
)

// Sub 返回以 dir 为根的文件系统。
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

// fixErr 通过去掉 f.dir 缩短 PathError 中报告的路径。
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
	return filepath.ToSlash(path.Join(f.dir, name)), nil
}

func (f *subFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	fullName, err := f.fullName("mkdir", name)
	if err != nil {
		return fmt.Errorf("resolve mkdir sub path %q: %w", name, err)
	}
	if err := f.fixErr(f.source.Mkdir(ctx, fullName, perm)); err != nil {
		return fmt.Errorf("mkdir sub path %q: %w", name, err)
	}
	return nil
}

func (f *subFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	fullName, err := f.fullName("openfile", name)
	if err != nil {
		return nil, fmt.Errorf("resolve open sub path %q: %w", name, err)
	}
	file, err := f.source.OpenFile(ctx, fullName, flag, perm)
	if err != nil {
		return nil, fmt.Errorf("open sub path %q: %w", name, f.fixErr(err))
	}
	return file, nil
}

func (f *subFS) RemoveAll(ctx context.Context, name string) error {
	fixed, err := f.fullName("remove_all", name)
	if err != nil {
		return fmt.Errorf("resolve remove sub path %q: %w", name, err)
	}
	if err := f.fixErr(f.source.RemoveAll(ctx, fixed)); err != nil {
		return fmt.Errorf("remove sub path %q: %w", name, err)
	}
	return nil
}

func (f *subFS) Rename(ctx context.Context, oldName, newName string) error {
	oldFullName, err := f.fullName("rename", oldName)
	if err != nil {
		return fmt.Errorf("resolve old rename sub path %q: %w", oldName, err)
	}
	newFullName, err := f.fullName("rename", newName)
	if err != nil {
		return fmt.Errorf("resolve new rename sub path %q: %w", newName, err)
	}
	if err := f.fixErr(f.source.Rename(ctx, oldFullName, newFullName)); err != nil {
		return fmt.Errorf("rename sub path %q to %q: %w", oldName, newName, err)
	}
	return nil
}

func (f *subFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	fullName, err := f.fullName("stat", name)
	if err != nil {
		return nil, fmt.Errorf("resolve stat sub path %q: %w", name, err)
	}
	info, err := f.source.Stat(ctx, fullName)
	if err != nil {
		return nil, fmt.Errorf("stat sub path %q: %w", name, f.fixErr(err))
	}
	return info, nil
}
