package filesystem

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path"
	"sort"
	"syscall"
)

var (
	SkipDir = fs.SkipDir
	SkipAll = fs.SkipAll
)

func Write(ctx context.Context, system FileSystem, name string, data []byte) error {
	f, err := system.OpenFile(ctx, name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func Open(ctx context.Context, system FileSystem, name string) (File, error) {
	return system.OpenFile(ctx, name, os.O_RDONLY, 0)
}

func Create(ctx context.Context, system FileSystem, name string) (File, error) {
	return system.OpenFile(ctx, name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

func MkdirAll(ctx context.Context, fsys FileSystem, path string) error {
	dir, err := Stat(ctx, fsys, path)
	if err == nil {
		if dir.IsDir() {
			return nil
		}
		return &fs.PathError{Op: "mkdir", Path: path, Err: syscall.ENOTDIR}
	}

	i := len(path)
	for i > 0 && os.IsPathSeparator(path[i-1]) {
		i--
	}

	j := i
	for j > 0 && !os.IsPathSeparator(path[j-1]) { // Scan backward over element.
		j--
	}

	if j > 1 {
		// Create parent.
		err = MkdirAll(ctx, fsys, path[:j-1])
		if err != nil {
			return err
		}
	}

	// Parent now exists; invoke Mkdir and use its result.
	err = fsys.Mkdir(ctx, path, os.ModePerm)
	if err != nil {
		// Handle arguments like "foo/." by
		// double-checking that directory doesn't exist.
		dir, err1 := Stat(ctx, fsys, path)
		if err1 == nil && dir.IsDir() {
			return nil
		}
		return err
	}
	return nil
}

func WalkDir(ctx context.Context, fsys FileSystem, root string, fn func(path string, d fs.DirEntry, err error) error) error {
	info, err := Stat(ctx, fsys, root)
	if err != nil {
		err = fn(root, nil, err)
	} else {
		err = walkDir(ctx, fsys, root, &statDirEntry{info}, fn)
	}
	if errors.Is(err, SkipDir) || errors.Is(err, SkipAll) {
		return nil
	}
	return err
}

func Stat(ctx context.Context, fsys FileSystem, name string) (FileInfo, error) {
	return fsys.Stat(ctx, name)
}

func ReadDir(ctx context.Context, fsys FileSystem, name string) ([]os.DirEntry, error) {
	file, err := fsys.OpenFile(ctx, name, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	dir, ok := file.(fs.ReadDirFile)
	if !ok {
		infos, err := file.Readdir(-1)
		if err != nil {
			return nil, err
		}
		entries := make([]os.DirEntry, len(infos))
		for i := range infos {
			entries[i] = fs.FileInfoToDirEntry(infos[i])
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		return entries, nil
	}

	list, err := dir.ReadDir(-1)
	sort.Slice(list, func(i, j int) bool { return list[i].Name() < list[j].Name() })
	return list, err
}

func walkDir(ctx context.Context, fsys FileSystem, name string, d fs.DirEntry, walkDirFn fs.WalkDirFunc) error {
	if err := walkDirFn(name, d, nil); err != nil || !d.IsDir() {
		if errors.Is(err, SkipDir) && d.IsDir() {
			// Successfully skipped directory.
			err = nil
		}
		return err
	}

	dirs, err := ReadDir(ctx, fsys, name)
	if err != nil {
		// Second call, to report ReadDir error.
		err = walkDirFn(name, d, err)
		if err != nil {
			if errors.Is(err, SkipDir) && d.IsDir() {
				err = nil
			}
			return err
		}
	}

	for _, d1 := range dirs {
		name1 := path.Join(name, d1.Name())
		if err := walkDir(ctx, fsys, name1, d1, walkDirFn); err != nil {
			if errors.Is(err, SkipDir) {
				break
			}
			return err
		}
	}
	return nil
}

type statDirEntry struct {
	info FileInfo
}

func (d *statDirEntry) Name() string            { return d.info.Name() }
func (d *statDirEntry) IsDir() bool             { return d.info.IsDir() }
func (d *statDirEntry) Type() fs.FileMode       { return d.info.Mode().Type() }
func (d *statDirEntry) Info() (FileInfo, error) { return d.info, nil }
func (d *statDirEntry) String() string {
	return fs.FormatDirEntry(d)
}
