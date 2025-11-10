package ftp

import (
	"context"
	"errors"
	"os"
	"path"
	"strings"

	"github.com/jlaffaye/ftp"
	"golang.org/x/net/webdav"

	"github.com/octohelm/unifs/pkg/filesystem"
)

func NewFS(c *Config) filesystem.FileSystem {
	if basePath := c.BasePath(); basePath != "" && basePath != "/" {
		return filesystem.Sub(&fs{c: c}, basePath)
	}
	return &fs{c: c}
}

type fs struct {
	c *Config
}

func (f *fs) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	if name == "" {
		name = "."
	}

	c, err := f.c.Conn(ctx)
	if err != nil {
		return nil, normalizeError("openfile", name, err)
	}
	defer c.Close()

	createWhenNotExists := flag&os.O_CREATE != 0

	if createWhenNotExists {
		// ensure parent exists
		if strings.Contains(name, "/") {
			_, err := c.GetEntry(path.Dir(name))
			if err != nil {
				return nil, normalizeError("openfile", name, err)
			}
		}
	}

	ftpEntry, err := c.GetEntry(name)
	if err != nil {
		e := normalizeError("openfile", name, err)
		if os.IsNotExist(e) {
			if !createWhenNotExists {
				return nil, e
			}
		} else {
			return nil, e
		}
	}

	if ftpEntry == nil {
		if !createWhenNotExists {
			return nil, normalizeError("openfile", name, os.ErrNotExist)
		}

		if perm.IsDir() {
			ftpEntry = &ftp.Entry{
				Type: ftp.EntryTypeFolder,
			}
		} else {
			ftpEntry = &ftp.Entry{
				Type: ftp.EntryTypeFile,
			}
		}
	}

	// ftp entry will return the full path
	// we need to set the path as the virtual name
	ftpEntry.Name = name

	return &file{
		ctx:    ctx,
		client: f.c,
		entry:  ftpEntry,
		flag:   flag,
	}, nil
}

func (f *fs) RemoveAll(ctx context.Context, name string) error {
	if name == "/" {
		return normalizeError("removeall", name, os.ErrPermission)
	}

	c, err := f.c.Conn(ctx)
	if err != nil {
		return normalizeError("removeall", name, err)
	}
	defer c.Close()

	ftpEntry, err := c.GetEntry(name)
	if err != nil {
		e := normalizeError("removeall", name, err)
		if errors.Is(e, os.ErrNotExist) {
			return nil
		}
		return e
	}

	if ftpEntry.Type == ftp.EntryTypeFolder {
		if err := c.RemoveDirRecur(name); err != nil {
			e := normalizeError("removeall", name, err)
			if errors.Is(e, os.ErrNotExist) {
				return nil
			}
			return e
		}
	} else {
		if err := c.Delete(name); err != nil {
			e := normalizeError("removeall", name, err)
			if errors.Is(e, os.ErrNotExist) {
				return nil
			}
			return e
		}
	}

	return nil
}

func (f *fs) Rename(ctx context.Context, oldName, newName string) error {
	c, err := f.c.Conn(ctx)
	if err != nil {
		return normalizeError("rename", newName, err, "from", oldName)
	}
	defer c.Close()

	if err := c.Rename(oldName, newName); err != nil {
		return normalizeError("rename", newName, err, "from", oldName)
	}
	return nil
}

func (f *fs) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	c, err := f.c.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	ftpEntry, err := c.GetEntry(name)
	if err != nil {
		return nil, normalizeError("stat", name, err)
	}

	return &entry{name: path.Base(ftpEntry.Name), entry: ftpEntry}, nil
}

func (f *fs) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	c, err := f.c.Conn(ctx)
	if err != nil {
		return err
	}
	defer c.Close()

	ftpEntry, _ := c.GetEntry(name)
	if ftpEntry != nil {
		return normalizeError("mkdir", name, os.ErrExist)
	}

	if err := c.MakeDir(name); err != nil {
		return normalizeError("mkdir", name, err)
	}
	return nil
}
