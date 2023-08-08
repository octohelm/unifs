package fuse

import (
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/local"
	"github.com/octohelm/unifs/pkg/filesystem/testutil"
	"os"
	"testing"
	"time"
)

func TestFuse(t *testing.T) {
	if os.Getenv("TEST_FUSE") != "1" {
		t.Skip()
	}

	d := mount(t, true)

	t.Run("Simple", func(t *testing.T) {
		testutil.TestSimpleFS(t, local.NewLocalFS(d))
	})

	t.Run("Full", func(t *testing.T) {
		testutil.TestFullFS(t, local.NewLocalFS(d))
	})
}

func mount(t *testing.T, debug bool) string {
	mountPoint := t.TempDir()

	r := FS(filesystem.NewMemFS())

	opt := &fs.Options{}

	opt.FirstAutomaticIno = 1
	opt.Debug = debug

	state, err := fs.Mount(mountPoint, r, opt)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		for i := 0; i < 5; i++ {
			if err := state.Unmount(); err == nil {
				break
			}
			time.Sleep(time.Second)
		}
	})

	return mountPoint
}
