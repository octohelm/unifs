package ftp

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/local"
	"github.com/octohelm/unifs/pkg/filesystem/testutil"
	"github.com/octohelm/unifs/pkg/ftp"
	"github.com/octohelm/unifs/pkg/strfmt"
)

func TestFTPFS(t *testing.T) {
	ftpServer := &ftp.Server{}
	ftpServer.SetDefaults()

	cwd, _ := os.Getwd()

	go func() {
		ctx := filesystem.Context.Inject(context.Background(), local.NewFS(cwd+"/testdata"))

		_ = ftpServer.Serve(ctx)
	}()

	time.Sleep(1 * time.Second)

	t.Cleanup(func() {
		_ = ftpServer.Shutdown(context.Background())
	})

	_ = os.RemoveAll(cwd + "/testdata")
	_ = os.Mkdir(cwd+"/testdata", os.ModePerm)

	c := &Config{}
	e, _ := strfmt.ParseEndpoint("ftp://" + ftpServer.Addr)
	c.Endpoint = *e
	c.Endpoint.Extra = url.Values{}
	c.Endpoint.Extra.Set("maxConnections", "2")

	t.Run("Simple", func(t *testing.T) {
		testutil.TestSimpleFS(t, NewFS(c))
		fmt.Println(c.p.count)
	})

	t.Run("Full", func(t *testing.T) {
		testutil.TestFullFS(t, NewFS(c))
		fmt.Println(c.p.count)
	})
}
