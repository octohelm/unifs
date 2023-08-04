package webdav

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/testutil"
	testingx "github.com/octohelm/x/testing"
	"net/http"
	"os"
	"testing"

	"golang.org/x/net/webdav"
	"net/http/httptest"
)

func TestWebdavFs(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		fs := newWebdavFS(t, true)
		data := map[string]any{
			"str": "x",
		}

		t.Run("mkdir", func(t *testing.T) {
			t.Run("success when parent dir exists", func(t *testing.T) {
				err := fs.Mkdir(context.Background(), "/x", os.ModePerm)
				testingx.Expect(t, err, testingx.Be[error](nil))
			})

			t.Run("failed when parent dir not exists", func(t *testing.T) {
				err := fs.Mkdir(context.Background(), "/a/c", os.ModePerm)
				testingx.Expect(t, err, testingx.Not(testingx.Be[error](nil)))
			})
		})

		t.Run("write file", func(t *testing.T) {
			f, err := fs.OpenFile(context.Background(), "/1.json", os.O_WRONLY, os.ModePerm)
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
			_ = f.Close()
			testingx.Expect(t, rev, testingx.Equal(data))
		})

		t.Run("stat /x/", func(t *testing.T) {
			f, err := fs.Stat(context.Background(), "/x")
			testingx.Expect(t, err, testingx.Be[error](nil))
			testingx.Expect(t, f.IsDir(), testingx.Be(true))
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
			err := fs.Rename(context.Background(), "/1.json", "/x/a/1.json")
			testingx.Expect(t, err, testingx.Be[error](nil))

			_, err = fs.Stat(context.Background(), "/x/a/1.json")
			testingx.Expect(t, err, testingx.Be[error](nil))

			f, err := fs.OpenFile(context.Background(), "/x/a/", os.O_RDONLY, os.ModePerm)
			testingx.Expect(t, err, testingx.Be[error](nil))
			list, err := f.Readdir(-1)
			testingx.Expect(t, err, testingx.Be[error](nil))
			testingx.Expect(t, len(list), testingx.Be(1))
		})

		t.Run("removeAll", func(t *testing.T) {
			err := fs.RemoveAll(context.Background(), "/x")
			testingx.Expect(t, err, testingx.Be[error](nil))

			_, err = fs.Stat(context.Background(), "/x/a/1.txt")
			testingx.Expect(t, err, testingx.Not(testingx.Be[error](nil)))

			_, err = fs.Stat(context.Background(), "/x")
			testingx.Expect(t, err, testingx.Not(testingx.Be[error](nil)))
		})
	})

	t.Run("Full", func(t *testing.T) {
		testutil.TestFS(t, newWebdavFS(t, false))
	})
}

func newWebdavFS(t *testing.T, debug bool) filesystem.FileSystem {
	svc := webdavServer(t, debug)
	return NewWebdavFS(svc.URL)
}

func webdavServer(t *testing.T, debug bool) *httptest.Server {
	svc := httptest.NewServer(&webdav.Handler{
		FileSystem: webdav.NewMemFS(),
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if debug {
				fmt.Println(r.Method, r.URL.String(), r.Header, err)
			}
		},
	})
	t.Cleanup(func() {
		svc.Close()
	})
	return svc
}
