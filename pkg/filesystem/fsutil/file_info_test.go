package fsutil

import (
	"io/fs"
	"testing"
	"time"

	. "github.com/octohelm/x/testing/v2"
)

func TestFileInfo(t *testing.T) {
	modTime := time.Unix(123, 0)

	file := NewFileInfo("file.txt", 42, modTime)
	Then(t, "构造普通文件信息",
		Expect(file.Name(), Equal("file.txt")),
		Expect(file.Size(), Equal(int64(42))),
		Expect(file.Mode(), Equal(fs.FileMode(0o664))),
		Expect(file.ModTime(), Equal(modTime)),
		Expect(file.IsDir(), Equal(false)),
		Expect(file.Sys(), Equal[any](nil)),
	)

	dir := NewDirFileInfo("dir")
	Then(t, "构造目录信息",
		Expect(dir.Name(), Equal("dir")),
		Expect(dir.Size(), Equal(int64(0))),
		Expect(dir.Mode(), Equal(fs.FileMode(0o755))),
		Expect(dir.ModTime(), Equal(time.Unix(0, 0))),
		Expect(dir.IsDir(), Equal(true)),
		Expect(dir.Sys(), Equal[any](nil)),
	)
}
