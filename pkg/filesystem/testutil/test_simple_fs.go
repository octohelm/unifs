package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"testing"

	"github.com/octohelm/x/slices"
	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/units"
)

func TestSimpleFS(t *testing.T, fs filesystem.FileSystem) {
	data := map[string]any{
		"str": "x",
		"slice": slices.Map(make([]any, 10000), func(e any) any {
			return "1"
		}),
	}

	t.Run("mkdir", func(t *testing.T) {
		t.Run("success when parent dir exists", func(t *testing.T) {
			Then(t, "父目录存在时创建目录成功",
				ExpectMust(func() error {
					return fs.Mkdir(context.Background(), "/x", os.ModeDir|os.ModePerm)
				}),
				ExpectMust(func() error {
					return fs.Mkdir(context.Background(), "/x/b", os.ModeDir|os.ModePerm)
				}),
				ExpectMust(func() error {
					return fs.Mkdir(context.Background(), "/x/b/c", os.ModeDir|os.ModePerm)
				}),
			)
		})

		t.Run("failed when parent dir not exists", func(t *testing.T) {
			Then(t, "父目录不存在时创建目录失败",
				ExpectDo(func() error {
					return fs.Mkdir(context.Background(), "/a/c", os.ModeDir|os.ModePerm)
				}, NotEqual[error](nil)),
			)
		})
	})

	t.Run("write file", func(t *testing.T) {
		f := MustValue(t, func() (filesystem.File, error) {
			return fs.OpenFile(context.Background(), "/1.json", os.O_WRONLY|os.O_CREATE, os.ModePerm)
		})

		Then(t, "文件写入成功",
			ExpectMust(func() error {
				return json.NewEncoder(f).Encode(data)
			}),
			ExpectMust(f.Close),
		)
	})

	t.Run("read file", func(t *testing.T) {
		f := MustValue(t, func() (filesystem.File, error) {
			return fs.OpenFile(context.Background(), "/1.json", os.O_RDONLY, os.ModePerm)
		})
		var rev map[string]any
		Must(t, func() error {
			return json.NewDecoder(f).Decode(&rev)
		})
		_ = f.Close()

		Then(t, "读回的文件内容等于写入内容",
			Expect(rev, Equal(data)),
		)
	})

	t.Run("copy file", func(t *testing.T) {
		dest := MustValue(t, func() (filesystem.File, error) {
			return fs.OpenFile(context.Background(), "/x/b/c/1.json", os.O_WRONLY|os.O_CREATE, os.ModePerm)
		})
		src := MustValue(t, func() (filesystem.File, error) {
			return fs.OpenFile(context.Background(), "/1.json", os.O_RDONLY, os.ModePerm)
		})
		MustValue(t, func() (int64, error) {
			return io.Copy(dest, src)
		})

		_ = src.Close()
		_ = dest.Close()
	})

	t.Run("stat /x/", func(t *testing.T) {
		f := MustValue(t, func() (os.FileInfo, error) {
			return fs.Stat(context.Background(), "/x")
		})

		Then(t, "目录 stat 返回目录信息",
			Expect(f.IsDir(), Equal(true)),
		)

		for i := range 4 {
			size := (i + 1) * (i + 1)

			t.Run(fmt.Sprintf("write large files %dMiB", size), func(t *testing.T) {
				f := MustValue(t, func() (filesystem.File, error) {
					return fs.OpenFile(context.Background(), fmt.Sprintf("/x/large-%dMiB.bin", i), os.O_WRONLY|os.O_CREATE, os.ModePerm)
				})
				Then(t, "大文件写入成功",
					ExpectMust(func() error {
						_, err := io.CopyN(f, CharFill('1'), int64(units.BinarySize(size)*units.MiB+units.BinarySize(rand.IntN(1024))))
						return err
					}),
					ExpectMust(f.Close),
				)
			})
		}
	})

	t.Run("stat file", func(t *testing.T) {
		f := MustValue(t, func() (os.FileInfo, error) {
			return fs.Stat(context.Background(), "/1.json")
		})

		Then(t, "文件 stat 返回非零大小",
			Expect(f.Size() > 0, Equal(true)),
		)
	})

	t.Run("list", func(t *testing.T) {
		f := MustValue(t, func() (filesystem.File, error) {
			return fs.OpenFile(context.Background(), "/", os.O_RDONLY, os.ModePerm)
		})

		list := MustValue(t, func() ([]os.FileInfo, error) {
			return f.Readdir(-1)
		})

		Then(t, "根目录列出两个条目",
			Expect(len(list), Equal(2)),
		)
	})

	t.Run("rename", func(t *testing.T) {
		Then(t, "文件重命名到新目录",
			ExpectMust(func() error {
				return fs.Mkdir(context.Background(), "/x/a", os.ModeDir|os.ModePerm)
			}),
			ExpectMust(func() error {
				return fs.Rename(context.Background(), "/1.json", "/x/a/1.json")
			}),
			ExpectMust(func() error {
				_, err := fs.Stat(context.Background(), "/x/a/1.json")
				return err
			}),
			ExpectDo(func() error {
				_, err := fs.Stat(context.Background(), "/1.json")
				return err
			}, NotEqual[error](nil)),
		)
	})

	t.Run("rename dir", func(t *testing.T) {
		Then(t, "目录重命名后原路径消失且新路径可读",
			ExpectMust(func() error {
				return fs.Rename(context.Background(), "/x/a", "/x/a1")
			}),
			ExpectDo(func() error {
				_, err := fs.Stat(context.Background(), "/x/a/1.json")
				return err
			}, NotEqual[error](nil)),
			ExpectMust(func() error {
				_, err := fs.Stat(context.Background(), "/x/a1/1.json")
				return err
			}),
		)

		f := MustValue(t, func() (filesystem.File, error) {
			return fs.OpenFile(context.Background(), "/x/a1/2.json", os.O_WRONLY|os.O_CREATE, os.ModePerm)
		})
		Then(t, "重命名后的目录仍可写入文件",
			ExpectMust(func() error {
				return json.NewEncoder(f).Encode(data)
			}),
			ExpectMust(f.Close),
		)
	})

	t.Run("removeAll", func(t *testing.T) {
		Then(t, "递归删除目录后目录内容被清空",
			ExpectMust(func() error {
				return fs.RemoveAll(context.Background(), "/x")
			}),
			ExpectDo(func() error {
				_, err := fs.Stat(context.Background(), "/x/a/1.json")
				return err
			}, NotEqual[error](nil)),
		)

		f := MustValue(t, func() (filesystem.File, error) {
			return fs.OpenFile(context.Background(), "/", os.O_RDONLY, os.ModePerm)
		})
		list := MustValue(t, func() ([]os.FileInfo, error) {
			return f.Readdir(-1)
		})
		Then(t, "根目录为空",
			Expect(len(list), Equal(0)),
		)
	})
}

type CharFill byte

func (b CharFill) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(b)
	}
	return len(p), nil
}
