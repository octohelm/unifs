package fuse

import (
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/octohelm/unifs/pkg/filesystem"
)

func FS(fsi filesystem.FileSystem) fs.InodeEmbedder {
	return &node{
		root: &root{
			base: "/",
			fsi:  fsi,
		},
	}
}

type root struct {
	base string
	fsi  filesystem.FileSystem
}

func (r *root) path(base *fs.Inode, names ...string) string {
	if len(names) > 0 {
		return filepath.Join(append([]string{r.base, base.Path(base.Root())}, names...)...)
	}
	return filepath.Join(r.base, base.Path(base.Root()))
}

func (r *root) newNode(parent *fs.Inode, fi os.FileInfo) fs.InodeEmbedder {
	return &node{root: r}
}

func (r *root) setAttrFromFileInfo(fi os.FileInfo, out *fuse.Attr) {
	if fi.IsDir() {
		out.Mode = syscall.S_IFDIR
	} else {
		out.Mode = uint32(fi.Mode())
		out.Size = uint64(fi.Size())
	}
	out.Mtime = uint64(fi.ModTime().Unix())
}

func newFileInfo(name string, mode os.FileMode) os.FileInfo {
	return &fileInfo{
		name:    name,
		mode:    mode,
		modTime: time.Now(),
	}
}

type fileInfo struct {
	name    string
	mode    os.FileMode
	size    int64
	modTime time.Time
}

func (f *fileInfo) Name() string {
	return f.name
}

func (f *fileInfo) Size() int64 {
	return f.size
}

func (f *fileInfo) Mode() os.FileMode {
	return f.mode
}

func (f *fileInfo) ModTime() time.Time {
	return f.modTime
}

func (f *fileInfo) IsDir() bool {
	return f.mode&syscall.S_IFDIR != 0
}

func (f *fileInfo) Sys() any {
	return nil
}
