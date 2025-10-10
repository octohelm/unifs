package logr

import (
	"context"
	"os"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/x/logr"
)

func Wrap(fsys filesystem.FileSystem, logger logr.Logger) filesystem.FileSystem {
	return &fs{fs: fsys, logger: logger}
}

type fs struct {
	fs     filesystem.FileSystem
	logger logr.Logger
}

func (f *fs) done(err error, op string, path string, values ...any) {
	l := f.logger.WithValues("op", op, "path", path)

	if len(values) > 0 {
		l = l.WithValues(values...)
	}

	if err != nil {
		l.Error(err)
	} else {
		l.Debug("")
	}
}

func (f *fs) Mkdir(ctx context.Context, name string, perm os.FileMode) (err error) {
	defer f.done(err, "mkdir", name)

	return f.fs.Mkdir(ctx, name, perm)
}

func (f *fs) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (file filesystem.File, err error) {
	defer f.done(err, "openfile", name)

	return f.fs.OpenFile(ctx, name, flag, perm)
}

func (f *fs) RemoveAll(ctx context.Context, name string) (err error) {
	defer f.done(err, "removeall", name)

	return f.fs.RemoveAll(ctx, name)
}

func (f *fs) Rename(ctx context.Context, oldName, newName string) (err error) {
	defer f.done(err, "rename", newName, "from", oldName)

	return f.fs.Rename(ctx, oldName, newName)
}

func (f *fs) Stat(ctx context.Context, name string) (info os.FileInfo, err error) {
	defer f.done(err, "stat", name)

	return f.fs.Stat(ctx, name)
}
