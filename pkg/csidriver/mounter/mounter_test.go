package mounter

import (
	"context"
	"os"
	"regexp"
	"testing"
	"time"

	. "github.com/octohelm/x/testing/v2"
)

func TestNewMounterAndUnmount(t *testing.T) {
	m, err := NewMounter(context.Background(), "file:///tmp")
	Then(t, "创建 file backend mounter",
		Expect(err, Equal[error](nil)),
		Expect(m != nil, Equal(true)),
	)

	Then(t, "不存在路径 unmount 视为成功",
		ExpectDo(func() error {
			return FuseUnmount(t.TempDir() + "/missing")
		}),
	)

	Then(t, "非法 backend 返回错误",
		ExpectDo(func() error {
			_, err := NewMounter(context.Background(), "://bad")
			return err
		}, ErrorMatch(regexpMust("missing protocol scheme"))),
	)
}

func regexpMust(pattern string) *regexp.Regexp {
	return regexp.MustCompile(pattern)
}

func TestMountHelpers(t *testing.T) {
	Then(t, "空挂载点在创建目录时失败",
		ExpectDo(func() error {
			return (&mounter{}).Mount("")
		}, ErrorMatch(regexpMust("no such file"))),
	)

	Then(t, "等待普通目录挂载会超时",
		ExpectDo(func() error {
			return waitForMount(t.TempDir(), time.Millisecond)
		}, ErrorMatch(regexpMust("Timeout waiting for mount|not supported"))),
	)

	Then(t, "读取不存在进程 cmdline 返回错误",
		ExpectDo(func() error {
			_, err := getCmdLine(-1)
			return err
		}, ErrorMatch(regexpMust("no such file"))),
	)

	process := MustValue(t, func() (*os.Process, error) {
		return FindFuseMountProcess("/path/that/should/not/exist")
	})
	Then(t, "找不到 FUSE 进程返回 nil",
		Expect(process, Equal((*os.Process)(nil))),
	)
}
