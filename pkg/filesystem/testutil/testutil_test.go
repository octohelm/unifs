package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/unifs/pkg/filesystem"
)

func TestHelpersWithMemFS(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		TestSimpleFS(t, filesystem.NewMemFS())
	})

	t.Run("full", func(t *testing.T) {
		TestFullFS(t, filesystem.NewMemFS())
	})

	t.Run("standard", func(t *testing.T) {
		TestStandardFS(t, filesystem.NewMemFS())
	})
}

func TestBenchmarkAndFileSize(t *testing.T) {
	b := &Benchmark{BigFileSize: 1, SmallFileCount: 1, Workspace: "/bench"}
	Then(t, "benchmark 使用文件系统读写大文件",
		ExpectMust(func() error {
			return b.Test(context.Background(), filesystem.NewMemFS())
		}),
		Expect(FileSize(512).String(), Equal("512 B")),
		Expect(FileSize(1024).String(), Equal("1.0 KiB")),
		Expect(FileSize(1024).Speed(time.Second), Equal("1.0 KiB/s")),
	)
}

func TestLocalTempDir(t *testing.T) {
	_ = os.MkdirAll(".tmp", 0o755)
	dir := LocalTempDir(t, "test-local-temp-dir")
	Then(t, "创建临时目录文件系统",
		Expect(dir, Equal(".tmp/test-local-temp-dir")),
	)
}
