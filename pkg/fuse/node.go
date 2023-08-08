package fuse

import (
	"context"
	"github.com/davecgh/go-spew/spew"
	"os"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"

	"github.com/octohelm/unifs/pkg/filesystem"
)

type Node interface {
	fs.NodeLookuper

	fs.NodeGetattrer
	fs.NodeSetattrer

	fs.NodeCreater
	fs.NodeOpener
	fs.NodeUnlinker

	fs.NodeReaddirer

	fs.NodeMkdirer
	fs.NodeRmdirer

	fs.NodeRenamer
}

var _ Node = &node{}

type node struct {
	fs.Inode
	root *root
}

func (n *node) Setattr(ctx context.Context, f fs.FileHandle, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	return 0
}

func (n *node) path(names ...string) string {
	return n.root.path(n.EmbeddedInode(), names...)
}

func (n *node) fsi() filesystem.FileSystem {
	return n.root.fsi
}

func (n *node) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	fi, err := n.fsi().Stat(ctx, n.path(name))
	if err != nil {
		return nil, fs.ToErrno(err)
	}
	n.root.setAttrFromFileInfo(fi, &out.Attr)
	ch := n.NewInode(ctx, n.root.newNode(n.EmbeddedInode(), fi), fs.StableAttr{Mode: out.Attr.Mode})
	return ch, 0
}

func (n *node) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	fi, err := n.fsi().Stat(ctx, n.path())
	if err != nil {
		return fs.ToErrno(err)
	}
	n.root.setAttrFromFileInfo(fi, &out.Attr)
	return 0
}

func (n *node) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	fullname := n.path(name)

	if err := n.fsi().Mkdir(ctx, fullname, os.FileMode(mode)); err != nil {
		return nil, fs.ToErrno(err)
	}

	fi, err := n.fsi().Stat(ctx, fullname)
	if err != nil {
		return nil, fs.ToErrno(err)
	}
	n.root.setAttrFromFileInfo(fi, &out.Attr)

	ch := n.NewInode(ctx, n.root.newNode(n.EmbeddedInode(), fi), fs.StableAttr{Mode: out.Attr.Mode})
	return ch, 0
}

func (n *node) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (node *fs.Inode, fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	fullname := n.path(name)

	f, err := n.fsi().OpenFile(ctx, fullname, int(flags), os.FileMode(mode))
	if err != nil {
		return nil, nil, 0, fs.ToErrno(err)
	}

	fi := newFileInfo(name, os.FileMode(mode))

	n.root.setAttrFromFileInfo(fi, &out.Attr)
	ch := n.NewInode(ctx, n.root.newNode(n.EmbeddedInode(), fi), fs.StableAttr{Mode: out.Attr.Mode})

	return ch, &file{f: f}, 0, 0
}

func (n *node) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	f, err := n.fsi().OpenFile(ctx, n.path(), int(flags), os.ModePerm)
	if err != nil {
		spew.Dump(err)
		return nil, 0, fs.ToErrno(err)
	}
	return &file{f: f}, 0, 0
}

func (n *node) Unlink(ctx context.Context, name string) syscall.Errno {
	if err := n.fsi().RemoveAll(ctx, n.path(name)); err != nil {
		return fs.ToErrno(err)
	}
	return 0
}

func (n *node) Rmdir(ctx context.Context, name string) syscall.Errno {
	if err := n.fsi().RemoveAll(ctx, n.path(name)); err != nil {
		return fs.ToErrno(err)
	}
	return 0
}

func (n *node) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno {
	newFullname := n.root.path(newParent.EmbeddedInode(), newName)
	if err := n.fsi().Rename(ctx, n.path(name), newFullname); err != nil {
		return fs.ToErrno(err)
	}
	return 0
}

func (n *node) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	f, err := n.fsi().OpenFile(ctx, n.path(), os.O_RDONLY, os.ModeDir)
	if err != nil {
		return nil, fs.ToErrno(err)
	}

	list, err := f.Readdir(-1)
	if err != nil {
		return nil, fs.ToErrno(err)
	}

	entries := make([]fuse.DirEntry, 0, len(list))

	for _, fi := range list {
		entry := fuse.DirEntry{
			Name: fi.Name(),
			Mode: uint32(fi.Mode()),
		}

		if fi.IsDir() {
			entry.Mode = syscall.S_IFDIR
		}

		entries = append(entries, entry)
	}

	return fs.NewListDirStream(entries), 0
}
