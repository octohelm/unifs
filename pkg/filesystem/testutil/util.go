package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func LocalTempDir(t *testing.T, name string) string {
	p := filepath.Join(".tmp", name)
	_ = os.RemoveAll(p)
	_ = os.Mkdir(p, os.ModePerm)
	return p
}
