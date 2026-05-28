package s3

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/octohelm/courier/pkg/courierhttp"

	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/testutil"
	"github.com/octohelm/unifs/pkg/strfmt"
)

func TestS3Fs(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		testutil.TestSimpleFS(t, newFakeS3FS(t))
	})

	t.Run("Full", func(t *testing.T) {
		testutil.TestFullFS(t, newFakeS3FS(t))
	})

	t.Run("Standard", func(t *testing.T) {
		testutil.TestStandardFS(t, newFakeS3FS(t))
	})

	t.Run("Bench", func(t *testing.T) {
		b := &testutil.Benchmark{}
		b.SetDefaults()
		b.RunT(t, newFakeS3FS(t))
	})
}

func TestS3WithPresignAs(t *testing.T) {
	fsys := newFakeS3FS(t, forPresign("https://rw:fake@x.io"))
	Then(t, "准备 presign 文件",
		ExpectMust(func() error {
			return fsys.Mkdir(context.Background(), "/x", os.ModePerm|os.ModeDir)
		}),
		ExpectMust(func() error {
			return filesystem.Write(context.Background(), fsys, "x.txt", []byte("123"))
		}),
	)

	f := MustValue(t, func() (filesystem.File, error) {
		return fsys.OpenFile(context.Background(), "x.txt", os.O_RDONLY, os.ModePerm)
	})
	defer f.Close()

	Then(t, "presignAs 覆盖重定向主机",
		Expect(f.(courierhttp.RedirectDescriber).Location().Host, Equal("test.x.io")),
	)
}

func forPresign(endpoint string) func(c *Config) {
	return func(c *Config) {
		c.Endpoint.Username = "admin"
		c.Endpoint.Password = "Admin123"
		c.Endpoint.Extra.Set("presignAs", endpoint)
		c.Endpoint.Extra.Set("signatureType", "v2")
	}
}

func newFakeS3FS(t *testing.T, opts ...func(c *Config)) filesystem.FileSystem {
	e := os.Getenv("TEST_S3_ENDPOINT")

	if e == "" {
		svc := fakeS3Server(t)
		e = fmt.Sprintf("%s/test?insecure=true", svc.URL)
	}

	endpoint := MustValue(t, func() (*strfmt.Endpoint, error) {
		return strfmt.ParseEndpoint(e)
	})

	endpoint.Path = path.Clean(fmt.Sprintf("%s/_tmp_%d", endpoint.Path, time.Now().UnixNano()))

	conf := &Config{
		Endpoint: *endpoint,
	}

	for _, opt := range opts {
		opt(conf)
	}

	t.Log(conf.Endpoint)

	fsys := MustValue(t, func() (filesystem.FileSystem, error) {
		return conf.AsFileSystem(context.Background())
	})

	return fsys
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
