package testutil

import (
	"os"
	"path"
	"testing"
)

func LocalTempDir(t *testing.T, name string) string {
	p := path.Join(".tmp", name)
	_ = os.RemoveAll(p)
	_ = os.Mkdir(p, os.ModePerm)
	return p
}
