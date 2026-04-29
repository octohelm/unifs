package fuse

import (
	"os"
	"testing"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	fusefuse "github.com/hanwen/go-fuse/v2/fuse"

	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/local"
	"github.com/octohelm/unifs/pkg/filesystem/testutil"
)

func _TestFuse(t *testing.T) {
	if os.Getenv("TEST_FUSE") != "1" {
		t.Skip()
	}

	d := mount(t, true)

	t.Run("Simple", func(t *testing.T) {
		testutil.TestSimpleFS(t, local.NewFS(d))
	})

	t.Run("Full", func(t *testing.T) {
		testutil.TestFullFS(t, local.NewFS(d))
	})
}

func mount(t *testing.T, debug bool) string {
	mountPoint := t.TempDir()

	r := FS(filesystem.NewMemFS())

	opt := &fs.Options{}

	opt.FirstAutomaticIno = 1
	opt.Debug = debug

	state := MustValue(t, func() (*fusefuse.Server, error) {
		return fs.Mount(mountPoint, r, opt)
	})

	t.Cleanup(func() {
		for range 5 {
			if err := state.Unmount(); err == nil {
				break
			}
			time.Sleep(time.Second)
		}
	})

	return mountPoint
}
