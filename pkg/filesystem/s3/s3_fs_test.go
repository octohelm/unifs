package s3

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	testingx "github.com/octohelm/x/testing"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/testutil"
	"github.com/octohelm/unifs/pkg/strfmt"
)

func TestS3Fs(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		fs := newFakeS3FS(t)

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
			f, err := fs.OpenFile(context.Background(), "/1.txt", os.O_WRONLY, os.ModePerm)
			testingx.Expect(t, err, testingx.Be[error](nil))
			_, _ = f.Write([]byte("123"))
			_ = f.Close()
		})

		t.Run("read file", func(t *testing.T) {
			f, err := fs.OpenFile(context.Background(), "/1.txt", os.O_RDONLY, os.ModePerm)
			testingx.Expect(t, err, testingx.Be[error](nil))
			data, _ := io.ReadAll(f)
			_ = f.Close()
			testingx.Expect(t, string(data), testingx.Be("123"))
		})

		t.Run("stat /x/", func(t *testing.T) {
			f, err := fs.Stat(context.Background(), "/x")
			testingx.Expect(t, err, testingx.Be[error](nil))
			testingx.Expect(t, f.IsDir(), testingx.Be(true))
		})

		t.Run("stat file", func(t *testing.T) {
			f, err := fs.Stat(context.Background(), "/1.txt")

			testingx.Expect(t, err, testingx.Be[error](nil))
			testingx.Expect(t, f.Size(), testingx.Be[int64](3))
		})

		t.Run("list", func(t *testing.T) {
			f, err := fs.OpenFile(context.Background(), "/", os.O_RDONLY, os.ModePerm)
			testingx.Expect(t, err, testingx.Be[error](nil))

			list, err := f.Readdir(-1)
			testingx.Expect(t, err, testingx.Be[error](nil))
			testingx.Expect(t, len(list), testingx.Be(2))
		})

		t.Run("rename", func(t *testing.T) {
			err := fs.Rename(context.Background(), "/1.txt", "/x/a/1.txt")
			testingx.Expect(t, err, testingx.Be[error](nil))

			f, err := fs.OpenFile(context.Background(), "/x/a", os.O_RDONLY, os.ModePerm)
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
		testutil.TestFS(t, newFakeS3FS(t))
	})
}

func newFakeS3FS(t *testing.T) filesystem.FileSystem {
	svc := fakeS3Server(t)
	endpoint, _ := strfmt.ParseEndpoint(svc.URL + "/test?insecure=true")
	conf := &Config{
		Endpoint: *endpoint,
		Bucket:   fmt.Sprintf("test-%d", rand.Int()),
	}
	c, _ := (conf).Client(context.Background())
	return NewS3FS(c, conf.Bucket)
}

func fakeS3Server(t *testing.T) *httptest.Server {
	backend := s3mem.New()
	faker := gofakes3.New(backend)
	svc := httptest.NewServer(faker.Server())
	t.Cleanup(func() {
		svc.Close()
	})
	return svc
}
