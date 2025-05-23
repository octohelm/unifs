package webdav

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/testutil"
	"github.com/octohelm/unifs/pkg/strfmt"

	"golang.org/x/net/webdav"
)

func TestWebdavFs(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		testutil.TestSimpleFS(t, newWebdavFS(t, true))
	})

	t.Run("Full", func(t *testing.T) {
		testutil.TestFullFS(t, newWebdavFS(t, false))
	})

	t.Run("Bench", func(t *testing.T) {
		b := &testutil.Benchmark{}
		b.SetDefaults()
		b.RunT(t, newWebdavFS(t, false))
	})
}

func newWebdavFS(t *testing.T, debug bool) filesystem.FileSystem {
	e := os.Getenv("TEST_WEBDAV_ENDPOINT")
	if e == "" {
		svc := webdavServer(t, debug)
		e = svc.URL + fmt.Sprintf("?insecure=true")
	}

	endpoint, err := strfmt.ParseEndpoint(e)
	if err != nil {
		t.Fatal(err)
	}

	conf := &Config{
		Endpoint: *endpoint,
	}

	t.Log(conf.Endpoint)

	c, err := (conf).Client(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	return NewFS(c)
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
