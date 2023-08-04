package local

import (
	"context"
	"os"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/pkg/errors"
	"path/filepath"
)

func NewLocalFS(prefix string) filesystem.FileSystem {
	return &localFS{prefix: prefix}
}

type localFS struct {
	prefix string
}

func (l *localFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	err := os.Mkdir(l.path(name), perm)
	if err != nil {
		return NormalizeError(err, l.prefix)
	}
	return nil
}

func (l *localFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (filesystem.File, error) {
	f, err := os.OpenFile(l.path(name), flag, perm)
	if err != nil {
		return nil, NormalizeError(err, l.prefix)
	}
	return f, nil
}

func (l *localFS) RemoveAll(ctx context.Context, name string) error {
	if name == "/" {
		return errors.Wrap(os.ErrPermission, "rm '/' not allow")
	}
	return os.RemoveAll(l.path(name))
}

func (l *localFS) Rename(ctx context.Context, oldName, newName string) error {
	err := os.Rename(l.path(oldName), l.path(newName))
	if err != nil {
		return NormalizeError(err, l.prefix)
	}
	return nil
}

func (l *localFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	s, err := os.Stat(l.path(name))
	if err != nil {
		return nil, NormalizeError(err, l.prefix)
	}
	return s, nil
}

func (l *localFS) path(name string) string {
	return filepath.Join(l.prefix, name)
}
