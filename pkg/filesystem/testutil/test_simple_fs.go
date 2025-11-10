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
	testingx "github.com/octohelm/x/testing"

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
			err := fs.Mkdir(context.Background(), "/x", os.ModeDir|os.ModePerm)
			testingx.Expect(t, err, testingx.Be[error](nil))

			err = fs.Mkdir(context.Background(), "/x/b", os.ModeDir|os.ModePerm)
			testingx.Expect(t, err, testingx.Be[error](nil))

			err = fs.Mkdir(context.Background(), "/x/b/c", os.ModeDir|os.ModePerm)
			testingx.Expect(t, err, testingx.Be[error](nil))
		})

		t.Run("failed when parent dir not exists", func(t *testing.T) {
			err := fs.Mkdir(context.Background(), "/a/c", os.ModeDir|os.ModePerm)
			testingx.Expect(t, err, testingx.Not(testingx.Be[error](nil)))
		})
	})

	t.Run("write file", func(t *testing.T) {
		f, err := fs.OpenFile(context.Background(), "/1.json", os.O_WRONLY|os.O_CREATE, os.ModePerm)
		testingx.Expect(t, err, testingx.Be[error](nil))
		err = json.NewEncoder(f).Encode(data)
		testingx.Expect(t, err, testingx.Be[error](nil))
		err = f.Close()
		testingx.Expect(t, err, testingx.Be[error](nil))
	})

	t.Run("read file", func(t *testing.T) {
		f, err := fs.OpenFile(context.Background(), "/1.json", os.O_RDONLY, os.ModePerm)
		testingx.Expect(t, err, testingx.Be[error](nil))
		var rev map[string]any
		err = json.NewDecoder(f).Decode(&rev)
		testingx.Expect(t, err, testingx.Be[error](nil))
		_ = f.Close()
		testingx.Expect(t, rev, testingx.Equal(data))
	})

	t.Run("copy file", func(t *testing.T) {
		dest, err := fs.OpenFile(context.Background(), "/x/b/c/1.json", os.O_WRONLY|os.O_CREATE, os.ModePerm)
		testingx.Expect(t, err, testingx.Be[error](nil))
		src, err := fs.OpenFile(context.Background(), "/1.json", os.O_RDONLY, os.ModePerm)
		testingx.Expect(t, err, testingx.Be[error](nil))
		_, err = io.Copy(dest, src)
		testingx.Expect(t, err, testingx.Be[error](nil))

		_ = src.Close()
		_ = dest.Close()
	})

	t.Run("stat /x/", func(t *testing.T) {
		f, err := fs.Stat(context.Background(), "/x")
		testingx.Expect(t, err, testingx.Be[error](nil))
		testingx.Expect(t, f.IsDir(), testingx.Be(true))

		for i := range 4 {
			size := (i + 1) * (i + 1)

			t.Run(fmt.Sprintf("write large files %dMiB", size), func(t *testing.T) {
				f, err := fs.OpenFile(context.Background(), fmt.Sprintf("/x/large-%dMiB.bin", i), os.O_WRONLY|os.O_CREATE, os.ModePerm)
				testingx.Expect(t, err, testingx.Be[error](nil))
				_, err = io.CopyN(f, CharFill('1'), int64(units.BinarySize(size)*units.MiB+units.BinarySize(rand.IntN(1024))))
				testingx.Expect(t, err, testingx.Be[error](nil))
				err = f.Close()
				testingx.Expect(t, err, testingx.Be[error](nil))
			})
		}
	})

	t.Run("stat file", func(t *testing.T) {
		f, err := fs.Stat(context.Background(), "/1.json")

		testingx.Expect(t, err, testingx.Be[error](nil))
		testingx.Expect(t, f.Size() > 0, testingx.Be(true))
	})

	t.Run("list", func(t *testing.T) {
		f, err := fs.OpenFile(context.Background(), "/", os.O_RDONLY, os.ModePerm)
		testingx.Expect(t, err, testingx.Be[error](nil))

		list, err := f.Readdir(-1)
		testingx.Expect(t, err, testingx.Be[error](nil))
		testingx.Expect(t, len(list), testingx.Be(2))
	})

	t.Run("rename", func(t *testing.T) {
		err := fs.Mkdir(context.Background(), "/x/a", os.ModeDir|os.ModePerm)
		testingx.Expect(t, err, testingx.Be[error](nil))

		err = fs.Rename(context.Background(), "/1.json", "/x/a/1.json")
		testingx.Expect(t, err, testingx.Be[error](nil))

		_, err = fs.Stat(context.Background(), "/x/a/1.json")
		testingx.Expect(t, err, testingx.Be[error](nil))

		_, err = fs.Stat(context.Background(), "/1.json")
		testingx.Expect(t, err, testingx.Not(testingx.Be[error](nil)))
	})

	t.Run("rename dir", func(t *testing.T) {
		err := fs.Rename(context.Background(), "/x/a", "/x/a1")
		testingx.Expect(t, err, testingx.Be[error](nil))

		_, err = fs.Stat(context.Background(), "/x/a/1.json")
		testingx.Expect(t, err, testingx.Not(testingx.Be[error](nil)))

		_, err = fs.Stat(context.Background(), "/x/a1/1.json")
		testingx.Expect(t, err, testingx.Be[error](nil))

		f, err := fs.OpenFile(context.Background(), "/x/a1/2.json", os.O_WRONLY|os.O_CREATE, os.ModePerm)
		testingx.Expect(t, err, testingx.Be[error](nil))
		err = json.NewEncoder(f).Encode(data)
		testingx.Expect(t, err, testingx.Be[error](nil))
		err = f.Close()
		testingx.Expect(t, err, testingx.Be[error](nil))
	})

	t.Run("removeAll", func(t *testing.T) {
		err := fs.RemoveAll(context.Background(), "/x")
		testingx.Expect(t, err, testingx.Be[error](nil))

		_, err = fs.Stat(context.Background(), "/x/a/1.json")
		testingx.Expect(t, err, testingx.Not(testingx.Be[error](nil)))

		f, err := fs.OpenFile(context.Background(), "/", os.O_RDONLY, os.ModePerm)
		testingx.Expect(t, err, testingx.Be[error](nil))
		list, err := f.Readdir(-1)
		testingx.Expect(t, err, testingx.Be[error](nil))
		testingx.Expect(t, len(list), testingx.Be(0))
	})
}

type CharFill byte

func (b CharFill) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(b)
	}
	return len(p), nil
}
