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
	testingx "github.com/octohelm/x/testing"

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

	t.Run("Bench", func(t *testing.T) {
		b := &testutil.Benchmark{}
		b.SetDefaults()
		b.RunT(t, newFakeS3FS(t))
	})
}

func TestS3WithPresignAs(t *testing.T) {
	fsys := newFakeS3FS(t, forPresign("https://rw:fake@x.io"))
	err := fsys.Mkdir(context.Background(), "/x", os.ModePerm|os.ModeDir)
	testingx.Expect(t, err, testingx.BeNil[error]())

	err = filesystem.Write(context.Background(), fsys, "x.txt", []byte("123"))
	testingx.Expect(t, err, testingx.BeNil[error]())

	f, err := fsys.OpenFile(context.Background(), "x.txt", os.O_RDONLY, os.ModePerm)
	testingx.Expect(t, err, testingx.BeNil[error]())
	defer f.Close()

	testingx.Expect(t, f.(courierhttp.RedirectDescriber).Location().Host, testingx.Be("test.x.io"))
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
		e = svc.URL + fmt.Sprintf("/test?insecure=true")
	}

	endpoint, err := strfmt.ParseEndpoint(e)
	if err != nil {
		t.Fatal(err)
	}

	endpoint.Path = path.Clean(fmt.Sprintf("%s/_tmp_%d", endpoint.Path, time.Now().UnixNano()))

	conf := &Config{
		Endpoint: *endpoint,
	}

	for _, opt := range opts {
		opt(conf)
	}

	t.Log(conf.Endpoint)

	fsys, err := (conf).AsFileSystem(context.Background())
	if err != nil {
		t.Fatal(err)
	}

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
