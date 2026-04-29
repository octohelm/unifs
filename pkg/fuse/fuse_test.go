package fuse

import (
	"context"
	"errors"
	"io"
	"os"
	"syscall"
	"testing"
	"time"

	gofs "github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"

	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/fsutil"
)

func TestRootAndFileInfo(t *testing.T) {
	r := &root{base: "/", fsi: filesystem.NewMemFS()}
	dirInfo := fsutil.NewDirFileInfo("dir")
	fileInfo := fsutil.NewFileInfo("file.txt", 12, time.Unix(123, 0))
	var dirAttr fuse.Attr
	var fileAttr fuse.Attr
	r.setAttrFromFileInfo(dirInfo, &dirAttr)
	r.setAttrFromFileInfo(fileInfo, &fileAttr)
	created := newFileInfo("x", 0o644)

	Then(t, "root 转换文件属性",
		Expect(FS(r.fsi) != nil, Equal(true)),
		Expect(r.newNode(nil, dirInfo) != nil, Equal(true)),
		Expect(dirAttr.Mode, Equal(uint32(syscall.S_IFDIR))),
		Expect(fileAttr.Mode, Equal(uint32(0o664))),
		Expect(fileAttr.Size, Equal(uint64(12))),
		Expect(fileAttr.Mtime, Equal(uint64(123))),
		Expect(created.Name(), Equal("x")),
		Expect(created.Size(), Equal(int64(0))),
		Expect(created.Mode(), Equal(os.FileMode(0o644))),
		Expect(created.IsDir(), Equal(false)),
		Expect(created.Sys(), Equal[any](nil)),
	)
}

func TestFileHandle(t *testing.T) {
	ctx := context.Background()
	fsys := filesystem.NewMemFS()
	Must(t, func() error {
		return filesystem.Write(ctx, fsys, "file.txt", []byte("hello"))
	})
	base := MustValue(t, func() (filesystem.File, error) {
		return fsys.OpenFile(ctx, "file.txt", os.O_RDWR, 0o644)
	})
	f := &file{f: base}
	buf := make([]byte, 5)
	result, readErrno := f.Read(ctx, buf, 0)
	written, writeErrno := f.Write(ctx, []byte("!"), 0)
	emptyWritten, emptyErrno := f.Write(ctx, nil, 0)
	fsyncErrno := f.Fsync(ctx, 0)
	releaseErrno := f.Release(ctx)

	Then(t, "file handle 读写和关闭",
		Expect(result != nil, Equal(true)),
		Expect(readErrno, Equal(syscall.Errno(0))),
		Expect(written, Equal(uint32(1))),
		Expect(writeErrno, Equal(syscall.Errno(0))),
		Expect(emptyWritten, Equal(uint32(0))),
		Expect(emptyErrno, Equal(syscall.Errno(0))),
		Expect(fsyncErrno, Equal(syscall.Errno(0))),
		Expect(releaseErrno, Equal(syscall.Errno(0))),
	)
}

func TestFileHandleErrors(t *testing.T) {
	ctx := context.Background()
	f := &file{f: errFile{}}
	_, readErrno := f.Read(ctx, make([]byte, 1), 1)
	_, writeErrno := f.Write(ctx, []byte("x"), 0)
	releaseErrno := f.Release(ctx)

	Then(t, "file handle 转换底层错误",
		Expect(readErrno, Equal(syscall.ENOENT)),
		Expect(writeErrno != 0, Equal(true)),
		Expect(releaseErrno != 0, Equal(true)),
	)
}

func TestNodeWithoutMountedInode(t *testing.T) {
	ctx := context.Background()
	fsys := filesystem.NewMemFS()
	Must(t, func() error { return fsys.Mkdir(ctx, "dir", 0o755) })
	Must(t, func() error { return filesystem.Write(ctx, fsys, "dir/file.txt", []byte("hello")) })

	n := &node{root: &root{base: "/", fsi: fsys}}
	var attr fuse.AttrOut
	var entry fuse.EntryOut
	lookupNode, lookupErrno := n.Lookup(ctx, "dir", &entry)
	mkdirNode, mkdirErrno := n.Mkdir(ctx, "made", 0o755, &entry)
	createNode, createdFH, _, createErrno := n.Create(ctx, "created.txt", uint32(os.O_CREATE|os.O_RDWR), 0o644, &entry)
	renameErrno := n.Rename(ctx, "created.txt", n, "renamed.txt", 0)
	fh, _, openErrno := n.Open(ctx, uint32(os.O_RDONLY))
	dirStream, readDirErrno := n.Readdir(ctx)
	unlinkErrno := n.Unlink(ctx, "dir/file.txt")
	rmdirErrno := n.Rmdir(ctx, "dir")

	Then(t, "未挂载 inode 的轻量分支",
		Expect(n.Setattr(ctx, nil, nil, nil), Equal(syscall.Errno(0))),
		Expect(n.path("dir"), Equal("/dir")),
		Expect(n.fsi(), Equal(fsys)),
		Expect(n.Getattr(ctx, nil, &attr), Equal(syscall.Errno(0))),
		Expect(lookupErrno, Equal(syscall.Errno(0))),
		Expect(lookupNode, Equal((*gofs.Inode)(nil))),
		Expect(mkdirErrno, Equal(syscall.Errno(0))),
		Expect(mkdirNode, Equal((*gofs.Inode)(nil))),
		Expect(createErrno, Equal(syscall.Errno(0))),
		Expect(createNode, Equal((*gofs.Inode)(nil))),
		Expect(createdFH != nil, Equal(true)),
		Expect(renameErrno, Equal(syscall.Errno(0))),
		Expect(openErrno, Equal(syscall.Errno(0))),
		Expect(fh != nil, Equal(true)),
		Expect(readDirErrno, Equal(syscall.Errno(0))),
		Expect(dirStream != nil, Equal(true)),
		Expect(unlinkErrno, Equal(syscall.Errno(0))),
		Expect(rmdirErrno, Equal(syscall.Errno(0))),
	)

	missing := &node{root: &root{base: "/", fsi: errFS{}}}
	_, lookupMissing := missing.Lookup(ctx, "missing", &entry)
	_, mkdirMissing := missing.Mkdir(ctx, "missing", 0o755, &entry)
	_, _, _, createMissing := missing.Create(ctx, "missing", uint32(os.O_CREATE|os.O_RDWR), 0o644, &entry)
	_, _, openMissing := missing.Open(ctx, uint32(os.O_RDONLY))
	_, readdirMissing := missing.Readdir(ctx)
	Then(t, "底层文件系统错误转 errno",
		Expect(missing.Getattr(ctx, nil, &attr) != 0, Equal(true)),
		Expect(lookupMissing != 0, Equal(true)),
		Expect(mkdirMissing != 0, Equal(true)),
		Expect(createMissing != 0, Equal(true)),
		Expect(openMissing != 0, Equal(true)),
		Expect(readdirMissing != 0, Equal(true)),
	)
}

type errFile struct{}

func (errFile) Read([]byte) (int, error)                             { return 0, io.ErrUnexpectedEOF }
func (errFile) Write([]byte) (int, error)                            { return 0, os.ErrPermission }
func (errFile) Close() error                                         { return os.ErrPermission }
func (errFile) Readdir(int) ([]os.FileInfo, error)                   { return nil, os.ErrPermission }
func (errFile) Stat() (os.FileInfo, error)                           { return nil, os.ErrPermission }
func (errFile) Seek(int64, int) (int64, error)                       { return 0, os.ErrPermission }
func (errFile) Fd() uintptr                                          { return 0 }
func (errFile) ReadAt([]byte, int64) (int, error)                    { return 0, os.ErrPermission }
func (errFile) WriteAt([]byte, int64) (int, error)                   { return 0, os.ErrPermission }
func (errFile) Truncate(int64) error                                 { return os.ErrPermission }
func (errFile) Sync() error                                          { return os.ErrPermission }
func (errFile) WriteString(string) (int, error)                      { return 0, os.ErrPermission }
func (errFile) Chmod(os.FileMode) error                              { return os.ErrPermission }
func (errFile) Chown(int, int) error                                 { return os.ErrPermission }
func (errFile) Chtimes(time.Time, time.Time) error                   { return os.ErrPermission }
func (errFile) Name() string                                         { return "err" }
func (errFile) Readdirnames(int) ([]string, error)                   { return nil, os.ErrPermission }
func (errFile) Getattr(context.Context, *fuse.AttrOut) syscall.Errno { return 0 }

var _ gofs.FileHandle = (*file)(nil)

type errFS struct{}

func (errFS) Mkdir(context.Context, string, os.FileMode) error { return os.ErrPermission }
func (errFS) OpenFile(context.Context, string, int, os.FileMode) (filesystem.File, error) {
	return nil, os.ErrPermission
}
func (errFS) RemoveAll(context.Context, string) error { return os.ErrPermission }
func (errFS) Rename(context.Context, string, string) error {
	return os.ErrPermission
}

func (errFS) Stat(context.Context, string) (os.FileInfo, error) {
	return nil, errors.New("stat failed")
}
