package s3

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
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

func newFakeS3FS(t *testing.T) filesystem.FileSystem {
	e := os.Getenv("TEST_S3_ENDPOINT")

	if e == "" {
		svc := fakeS3Server(t)
		e = svc.URL + fmt.Sprintf("/test?insecure=true")
	}

	endpoint, err := strfmt.ParseEndpoint(e)
	if err != nil {
		t.Fatal(err)
	}

	endpoint.Path = filepath.Clean(fmt.Sprintf("%s/_tmp_%d", endpoint.Path, time.Now().UnixNano()))

	conf := &Config{
		Endpoint: *endpoint,
	}

	t.Log(conf.Endpoint)

	c, err := (conf).Client(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	return NewS3FS(c, conf.Bucket(), conf.Prefix())
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
