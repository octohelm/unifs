package testutil

import (
	"context"
	iofs "io/fs"
	"regexp"
	"testing"

	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/unifs/pkg/filesystem"
)

// TestStandardFS 验证 FileSystem 通过 AsReadDirFS 暴露给标准库 io/fs 后的只读语义。
func TestStandardFS(t *testing.T, fsys filesystem.FileSystem) {
	ctx := context.Background()

	Then(t, "准备标准库 io/fs 兼容性测试数据",
		ExpectMust(func() error {
			return filesystem.MkdirAll(ctx, fsys, "std/dir")
		}),
		ExpectMust(func() error {
			return filesystem.Write(ctx, fsys, "std/dir/a.txt", []byte("a"))
		}),
		ExpectMust(func() error {
			return filesystem.Write(ctx, fsys, "std/b.txt", []byte("b"))
		}),
	)

	std := filesystem.AsReadDirFS(filesystem.Sub(fsys, "std"))
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

	Then(t, "标准库只读入口读取同一棵文件树",
		Expect(string(data), Equal("a")),
		Expect(info.Name(), Equal("a.txt")),
		Expect([]string{entries[0].Name(), entries[1].Name()}, Equal([]string{"b.txt", "dir"})),
		Expect(visited, Equal([]string{".", "b.txt", "dir", "dir/a.txt"})),
	)

	Then(t, "非法标准库路径返回错误",
		ExpectDo(func() error {
			_, err := std.Open("../x")
			return err
		}, ErrorMatch(regexp.MustCompile("invalid name"))),
	)
}
