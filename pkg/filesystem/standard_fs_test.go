package filesystem

import (
	"context"
	iofs "io/fs"
	"regexp"
	"testing"

	. "github.com/octohelm/x/testing/v2"
)

func TestStandardLibraryFSCompatibility(t *testing.T) {
	ctx := context.Background()
	fsys := NewMemFS()

	Then(t, "准备标准库兼容性测试数据",
		ExpectMust(func() error {
			return MkdirAll(ctx, fsys, "root/dir")
		}),
		ExpectMust(func() error {
			return Write(ctx, fsys, "root/dir/a.txt", []byte("a"))
		}),
		ExpectMust(func() error {
			return Write(ctx, fsys, "root/b.txt", []byte("b"))
		}),
	)

	std := AsReadDirFS(Sub(fsys, "root"))
	data := MustValue(t, func() ([]byte, error) {
		return iofs.ReadFile(std, "dir/a.txt")
	})
	info := MustValue(t, func() (iofs.FileInfo, error) {
		return iofs.Stat(std, "dir/a.txt")
	})
	entries := MustValue(t, func() ([]iofs.DirEntry, error) {
		return iofs.ReadDir(std, ".")
	})

	visited := make([]string, 0)
	Must(t, func() error {
		return iofs.WalkDir(std, ".", func(name string, d iofs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			visited = append(visited, name)
			return nil
		})
	})

	Then(t, "AsReadDirFS 支持标准库只读入口",
		Expect(string(data), Equal("a")),
		Expect(info.Name(), Equal("a.txt")),
		Expect([]string{entries[0].Name(), entries[1].Name()}, Equal([]string{"b.txt", "dir"})),
		Expect(visited, Equal([]string{".", "b.txt", "dir", "dir/a.txt"})),
	)

	dirFS := MustValue(t, func() (iofs.FS, error) {
		return iofs.Sub(std, "dir")
	})
	subData := MustValue(t, func() ([]byte, error) {
		return iofs.ReadFile(dirFS, "a.txt")
	})

	Then(t, "标准库 fs.Sub 可继续裁剪 AsReadDirFS",
		Expect(string(subData), Equal("a")),
	)

	Then(t, "非法标准库路径通过 FileSystem 错误返回",
		ExpectDo(func() error {
			_, err := std.Open("../x")
			return err
		}, ErrorMatch(regexp.MustCompile("invalid name"))),
	)
}
