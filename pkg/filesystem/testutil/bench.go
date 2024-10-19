package testutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/octohelm/unifs/pkg/filesystem"
)

type Benchmark struct {
	BigFileSize    uint64
	SmallFileCount uint64
	Workspace      string
}

func (b *Benchmark) SetDefaults() {
	if b.BigFileSize == 0 {
		b.BigFileSize = 512
	}

	if b.SmallFileCount == 0 {
		b.SmallFileCount = 100
	}

	if b.Workspace == "" {
		b.Workspace = fmt.Sprintf("/_bench_%d", time.Now().UnixNano())
	}
}

func (b *Benchmark) RunT(t *testing.T, fsi filesystem.FileSystem) {
	ctx := context.Background()

	if err := b.Test(ctx, fsi); err != nil {
		t.Fatal(err)
	}
}

func (b *Benchmark) Test(ctx context.Context, fsi filesystem.FileSystem) error {
	if err := fsi.Mkdir(ctx, b.Workspace, os.ModePerm); err != nil {
		return fmt.Errorf("mkdir failed: %w", err)
	}

	defer func() {
		_ = fsi.RemoveAll(ctx, b.Workspace)
	}()

	ret, err := b.testBigFileWrite(ctx, fsi)
	if err != nil {
		return err
	}
	fmt.Println(ret)

	ret, err = b.testBigFileRead(ctx, fsi)
	if err != nil {
		return err
	}
	fmt.Println(ret)

	return nil
}

const bufSize = 1024 * 1024 // 1 MiB

func (b *Benchmark) testBigFileWrite(ctx context.Context, fsi filesystem.FileSystem) (string, error) {
	started := time.Now()

	f, err := fsi.OpenFile(ctx, filepath.Join(b.Workspace, "big_file"), os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return "", err
	}
	defer f.Close()

	for i := 0; i < int(b.BigFileSize); i++ {
		buf := bytes.NewBuffer(make([]byte, bufSize))

		_, err := io.Copy(f, buf)
		if err != nil {
			return "", err
		}

	}

	return fmt.Sprintf("TestBigFileWrite: %s", FileSize(b.BigFileSize*uint64(bufSize)).Speed(time.Since(started))), nil
}

func (b *Benchmark) testBigFileRead(ctx context.Context, fsi filesystem.FileSystem) (string, error) {
	started := time.Now()

	f, err := fsi.OpenFile(ctx, filepath.Join(b.Workspace, "big_file"), os.O_RDONLY, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("open failed: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(io.Discard, f)
	if err != nil {
		return "", fmt.Errorf("copy failed: %w", err)
	}

	return fmt.Sprintf("TestBigFileRead: %s", FileSize(b.BigFileSize*uint64(bufSize)).Speed(time.Since(started))), nil
}

type FileSize int64

func (f FileSize) String() string {
	b := int64(f)
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func (f FileSize) Speed(total time.Duration) string {
	return fmt.Sprintf("%s/s", FileSize(float64(f)/(float64(total)/float64(time.Second))))
}
