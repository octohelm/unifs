package aferofsutil

import (
	"io/fs"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/spf13/afero"

	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/unifs/pkg/filesystem"
)

func TestAferoAdaptor(t *testing.T) {
	fsys := From(filesystem.NewMemFS())

	Then(t, "创建目录和文件",
		ExpectMust(func() error {
			return fsys.Mkdir("dir", 0o755)
		}),
		ExpectMust(func() error {
			return fsys.MkdirAll("dir/sub", 0o755)
		}),
		ExpectMust(func() error {
			f, err := fsys.Create("dir/sub/file.txt")
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = f.WriteString("hello")
			return err
		}),
	)

	info := MustValue(t, func() (os.FileInfo, error) {
		return fsys.Stat("dir/sub/file.txt")
	})
	Then(t, "适配文件信息",
		Expect(info.Name(), Equal("file.txt")),
		Expect(info.Size(), Equal(int64(5))),
		Expect(fsys.Name(), Equal("unifs")),
	)

	f := MustValue(t, func() (afero.File, error) {
		return fsys.Open("dir/sub/file.txt")
	})
	defer f.Close()
	buf := make([]byte, 5)
	_, err := f.Read(buf)
	Then(t, "读取文件内容",
		Expect(err, Equal[error](nil)),
		Expect(string(buf), Equal("hello")),
	)

	Then(t, "重命名和删除",
		ExpectMust(func() error {
			return fsys.Rename("dir/sub/file.txt", "dir/sub/renamed.txt")
		}),
		ExpectMust(func() error {
			return fsys.Remove("dir/sub/renamed.txt")
		}),
		ExpectMust(func() error {
			return fsys.RemoveAll("dir/sub")
		}),
		ExpectDo(func() error {
			_, err := fsys.Stat("dir/sub/renamed.txt")
			return err
		}, ErrorIs(fs.ErrNotExist)),
	)
}

func TestAferoAdaptorUnsupportedOperations(t *testing.T) {
	fsys := From(filesystem.NewMemFS())

	Then(t, "不支持的文件系统操作返回 PathError",
		ExpectDo(func() error {
			return fsys.Chmod("x", 0o644)
		}, ErrorIs(fs.ErrPermission)),
		ExpectDo(func() error {
			return fsys.Chown("x", 1, 1)
		}, ErrorIs(fs.ErrPermission)),
		ExpectDo(func() error {
			return fsys.Chtimes("x", time.Now(), time.Now())
		}, ErrorIs(fs.ErrPermission)),
	)
}

func TestAferoFileDirectoryAndRandomAccess(t *testing.T) {
	fsys := From(filesystem.NewMemFS())
	Must(t, func() error {
		return fsys.MkdirAll("dir", 0o755)
	})
	Must(t, func() error {
		return afero.WriteFile(fsys, "dir/a.txt", []byte("abc"), 0o644)
	})
	Must(t, func() error {
		return afero.WriteFile(fsys, "dir/b.txt", []byte("def"), 0o644)
	})

	dir := MustValue(t, func() (afero.File, error) {
		return fsys.Open("dir")
	})
	defer dir.Close()
	names := MustValue(t, func() ([]string, error) {
		return dir.Readdirnames(-1)
	})
	sort.Strings(names)

	file := MustValue(t, func() (afero.File, error) {
		return fsys.OpenFile("dir/a.txt", os.O_RDWR, 0)
	})
	defer file.Close()
	Then(t, "目录适配 afero.File",
		Expect(names, Equal([]string{"a.txt", "b.txt"})),
		ExpectDo(file.Sync),
	)

	Then(t, "底层文件不支持的扩展操作返回权限错误",
		ExpectDo(func() error {
			_, err := file.ReadAt(make([]byte, 1), 0)
			return err
		}, ErrorIs(fs.ErrPermission)),
		ExpectDo(func() error {
			_, err := file.WriteAt([]byte("x"), 0)
			return err
		}, ErrorIs(fs.ErrPermission)),
		ExpectDo(func() error {
			return file.Truncate(1)
		}, ErrorIs(fs.ErrPermission)),
	)
}
