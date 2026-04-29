package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"syscall"
)

var (
	// SkipDir 由 WalkDir 回调返回，用于跳过当前目录。
	SkipDir = fs.SkipDir
	// SkipAll 由 WalkDir 回调返回，用于无错误地停止遍历。
	SkipAll = fs.SkipAll
)

// Write 创建或截断 name，并写入 data。
func Write(ctx context.Context, system FileSystem, name string, data []byte) error {
	f, err := system.OpenFile(ctx, name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return fmt.Errorf("open %q for write: %w", name, err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write %q: %w", name, err)
	}
	return nil
}

// Open 以只读方式打开 name。
func Open(ctx context.Context, system FileSystem, name string) (File, error) {
	return system.OpenFile(ctx, name, os.O_RDONLY, 0)
}

// Create 创建或截断 name，并以读写方式打开。
func Create(ctx context.Context, system FileSystem, name string) (File, error) {
	return system.OpenFile(ctx, name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o666)
}

// MkdirAll 创建路径及所有缺失的父目录。
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
	for j > 0 && !os.IsPathSeparator(path[j-1]) { // 向后扫描当前路径片段。
		j--
	}

	if j > 1 {
		// 创建父目录。
		err = MkdirAll(ctx, fsys, path[:j-1])
		if err != nil {
			return fmt.Errorf("mkdir parent %q: %w", path[:j-1], err)
		}
	}

	// 父目录已存在，调用 Mkdir 并使用其结果。
	err = fsys.Mkdir(ctx, path, os.ModePerm)
	if err != nil {
		// 处理 "foo/." 这类参数时，再次确认目录是否已存在。
		dir, err1 := Stat(ctx, fsys, path)
		if err1 == nil && dir.IsDir() {
			return nil
		}
		return fmt.Errorf("mkdir %q: %w", path, err)
	}
	return nil
}

// WalkDir 按字典序遍历以根路径为起点的文件树。
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

// Stat 返回 name 的文件信息。
func Stat(ctx context.Context, fsys FileSystem, name string) (FileInfo, error) {
	return fsys.Stat(ctx, name)
}

// ReadDir 读取 name，并按字典序返回目录项。
func ReadDir(ctx context.Context, fsys FileSystem, name string) ([]os.DirEntry, error) {
	file, err := fsys.OpenFile(ctx, name, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return nil, fmt.Errorf("open dir %q: %w", name, err)
	}
	defer file.Close()

	dir, ok := file.(fs.ReadDirFile)
	if !ok {
		infos, err := file.Readdir(-1)
		if err != nil {
			return nil, fmt.Errorf("readdir %q: %w", name, err)
		}
		entries := make([]os.DirEntry, len(infos))
		for i := range infos {
			entries[i] = fs.FileInfoToDirEntry(infos[i])
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		return entries, nil
	}

	list, err := dir.ReadDir(-1)
	if err != nil {
		return nil, fmt.Errorf("readdir %q: %w", name, err)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name() < list[j].Name() })
	return list, nil
}

func walkDir(ctx context.Context, fsys FileSystem, name string, d fs.DirEntry, walkDirFn fs.WalkDirFunc) error {
	if err := walkDirFn(name, d, nil); err != nil || !d.IsDir() {
		if errors.Is(err, SkipDir) && d.IsDir() {
			// 成功跳过目录。
			err = nil
		}
		return err
	}

	dirs, err := ReadDir(ctx, fsys, name)
	if err != nil {
		// 第二次调用，用于报告 ReadDir 错误。
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
