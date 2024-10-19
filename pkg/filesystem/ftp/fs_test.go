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
	t.Run("ftp server without MLTS", func(t *testing.T) {
		ftpServer := &ftp.Server{}
		ftpServer.DisableMLST = true
		ftpServer.SetDefaults()

		dir := t.TempDir()

		go func() {
			ctx := filesystem.Context.Inject(context.Background(), local.NewFS(dir))
			_ = ftpServer.Serve(ctx)
		}()

		time.Sleep(1 * time.Second)

		t.Cleanup(func() {
			_ = os.RemoveAll(dir)
			_ = ftpServer.Shutdown(context.Background())
		})

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
	})

	t.Run("ftp server", func(t *testing.T) {
		ftpServer := &ftp.Server{}
		ftpServer.SetDefaults()

		dir := t.TempDir()

		go func() {
			ctx := filesystem.Context.Inject(context.Background(), local.NewFS(dir))
			_ = ftpServer.Serve(ctx)
		}()

		time.Sleep(1 * time.Second)

		t.Cleanup(func() {
			_ = os.RemoveAll(dir)
			_ = ftpServer.Shutdown(context.Background())
		})

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
	})
}
