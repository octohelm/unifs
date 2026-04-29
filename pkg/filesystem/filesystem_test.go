package filesystem

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"regexp"
	"syscall"
	"testing"

	"github.com/octohelm/courier/pkg/courier"

	. "github.com/octohelm/x/testing/v2"
)

func TestMemFSUtilities(t *testing.T) {
	ctx := context.Background()
	fsys := NewMemFS()

	Then(t, "创建目录并写入文件",
		ExpectMust(func() error {
			return MkdirAll(ctx, fsys, "a/b")
		}),
		ExpectMust(func() error {
			return Write(ctx, fsys, "a/b/file.txt", []byte("hello"))
		}),
	)

	created := MustValue(t, func() (File, error) {
		return Create(ctx, fsys, "a/created.txt")
	})
	_, err := created.Write([]byte("created"))
	Must(t, created.Close)
	Then(t, "Create 创建可写文件",
		Expect(err, Equal[error](nil)),
	)

	f := MustValue(t, func() (File, error) {
		return Open(ctx, fsys, "a/b/file.txt")
	})
	defer f.Close()

	buf := make([]byte, 5)
	_, err = f.Read(buf)
	Then(t, "读取写入内容",
		Expect(err, Equal[error](nil)),
		Expect(string(buf), Equal("hello")),
	)

	entries := MustValue(t, func() ([]os.DirEntry, error) {
		return ReadDir(ctx, fsys, "a")
	})
	Then(t, "按名称读取目录项",
		Expect(len(entries), Equal(2)),
		Expect(entries[0].Name(), Equal("b")),
		Expect(entries[1].Name(), Equal("created.txt")),
	)

	visited := make([]string, 0)
	Must(t, func() error {
		return WalkDir(ctx, fsys, "a", func(name string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			visited = append(visited, name)
			return nil
		})
	})
	Then(t, "WalkDir 按字典序遍历",
		Expect(visited, Equal([]string{"a", "a/b", "a/b/file.txt", "a/created.txt"})),
	)
}

func TestMkdirAllExistingFile(t *testing.T) {
	ctx := context.Background()
	fsys := NewMemFS()

	Must(t, func() error {
		return Write(ctx, fsys, "file", []byte("x"))
	})

	Then(t, "路径已存在但不是目录时返回错误",
		ExpectDo(func() error {
			return MkdirAll(ctx, fsys, "file")
		}, ErrorIs(syscall.ENOTDIR)),
		ExpectDo(func() error {
			return MkdirAll(ctx, fsys, "file/child")
		}, ErrorIs(syscall.ENOTDIR)),
		ExpectDo(func() error {
			_, err := ReadDir(ctx, fsys, "missing")
			return err
		}, ErrorIs(fs.ErrNotExist)),
	)
}

func TestSubAndAsReadDirFS(t *testing.T) {
	ctx := context.Background()
	fsys := NewMemFS()
	Must(t, func() error {
		return MkdirAll(ctx, fsys, "root/child")
	})
	Must(t, func() error {
		return Write(ctx, fsys, "root/child/a.txt", []byte("a"))
	})

	sub := Sub(fsys, "root")
	Then(t, "Sub 将路径限制到子目录",
		ExpectMust(func() error {
			return Write(ctx, sub, "b.txt", []byte("b"))
		}),
		ExpectMust(func() error {
			return sub.Mkdir(ctx, "made", 0o755)
		}),
		ExpectMust(func() error {
			return sub.Rename(ctx, "b.txt", "renamed.txt")
		}),
	)

	info := MustValue(t, func() (FileInfo, error) {
		return sub.Stat(ctx, "renamed.txt")
	})
	Then(t, "Sub 写入源文件系统的子目录",
		Expect(info.Name(), Equal("renamed.txt")),
	)

	Then(t, "Sub 删除源文件系统的子目录内容",
		ExpectMust(func() error {
			return sub.RemoveAll(ctx, "renamed.txt")
		}),
		ExpectDo(func() error {
			_, err := fsys.Stat(ctx, "root/renamed.txt")
			return err
		}, ErrorIs(fs.ErrNotExist)),
	)

	readDirFS := AsReadDirFS(sub)
	opened := MustValue(t, func() (fs.File, error) {
		return readDirFS.Open("child/a.txt")
	})
	Must(t, opened.Close)
	entries := MustValue(t, func() ([]fs.DirEntry, error) {
		return readDirFS.ReadDir(".")
	})
	Then(t, "AsReadDirFS 适配标准库接口",
		Expect([]string{entries[0].Name(), entries[1].Name()}, Equal([]string{"child", "made"})),
	)

	Then(t, "Sub 空路径返回源文件系统",
		Expect(Sub(fsys, ""), Equal(fsys)),
		Expect(Sub(fsys, "/"), Equal(fsys)),
	)

	Then(t, "Sub 拒绝非法路径",
		ExpectDo(func() error {
			_, err := sub.OpenFile(ctx, "../x", os.O_RDONLY, 0)
			return err
		}, ErrorMatch(regexp.MustCompile("invalid name"))),
		ExpectDo(func() error {
			_, err := sub.Stat(ctx, "missing")
			return err
		}, ErrorIs(fs.ErrNotExist)),
	)
}

func TestSubInternals(t *testing.T) {
	sub := Sub(NewMemFS(), "root").(*subFS)

	rel, ok := sub.shorten("root/child/file.txt")
	dot, dotOK := sub.shorten("root")
	missing, missingOK := sub.shorten("other")
	full := MustValue(t, func() (string, error) {
		return sub.fullName("open", "/child/file.txt")
	})
	err := sub.fixErr(&fs.PathError{Op: "open", Path: "root/child/file.txt", Err: fs.ErrNotExist})

	Then(t, "subFS 内部路径转换",
		Expect(rel, Equal("child/file.txt")),
		Expect(ok, Equal(true)),
		Expect(dot, Equal(".")),
		Expect(dotOK, Equal(true)),
		Expect(missing, Equal("")),
		Expect(missingOK, Equal(false)),
		Expect(full, Equal("root/child/file.txt")),
		Expect(err.(*fs.PathError).Path, Equal("child/file.txt")),
	)
}

func TestWalkDirSkipAndErrors(t *testing.T) {
	ctx := context.Background()
	fsys := NewMemFS()
	Must(t, func() error {
		return MkdirAll(ctx, fsys, "root/skip")
	})
	Must(t, func() error {
		return Write(ctx, fsys, "root/skip/file.txt", []byte("x"))
	})
	Must(t, func() error {
		return Write(ctx, fsys, "root/keep.txt", []byte("x"))
	})

	visited := make([]string, 0)
	Must(t, func() error {
		return WalkDir(ctx, fsys, "root", func(name string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			visited = append(visited, name)
			if name == "root/skip" {
				return SkipDir
			}
			return nil
		})
	})
	Then(t, "SkipDir 跳过当前目录",
		Expect(visited, Equal([]string{"root", "root/keep.txt", "root/skip"})),
	)

	Then(t, "根路径不存在时将错误传入回调",
		ExpectDo(func() error {
			return WalkDir(ctx, fsys, "missing", func(name string, d fs.DirEntry, err error) error {
				if name == "missing" && err != nil {
					return SkipAll
				}
				return errors.New("unexpected callback")
			})
		}),
	)

	Then(t, "文件节点返回 SkipDir 会停止遍历且不报错",
		ExpectDo(func() error {
			return WalkDir(ctx, fsys, "root/keep.txt", func(name string, d fs.DirEntry, err error) error {
				return SkipDir
			})
		}),
	)
}

func TestMetadataContext(t *testing.T) {
	ctx := context.Background()
	meta := courier.Metadata{"x-request-id": []string{"1"}}
	ctx = MetadataInjectContext(ctx, meta)

	Then(t, "context 中读取 metadata",
		Expect(MetadataFromContext(ctx), Equal(meta)),
		Expect(MetadataFromContext(context.Background()), Equal(courier.Metadata{})),
	)
}

func TestStatDirEntry(t *testing.T) {
	info := MustValue(t, func() (FileInfo, error) {
		return NewMemFS().Stat(context.Background(), ".")
	})
	entry := &statDirEntry{info: info}
	entryInfo := MustValue(t, entry.Info)

	Then(t, "statDirEntry 适配 fs.DirEntry",
		Expect(entry.Name(), Equal("/")),
		Expect(entry.IsDir(), Equal(true)),
		Expect(entry.Type(), Equal(fs.FileMode(os.ModeDir))),
		Expect(entryInfo, Equal(info)),
		Expect(entry.String(), Equal("d //")),
	)
}
