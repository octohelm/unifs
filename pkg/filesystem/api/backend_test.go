package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"testing"

	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/strfmt"
)

func TestFileSystemBackend(t *testing.T) {
	ctx := context.Background()

	var disabled FileSystemBackend
	Then(t, "空 backend 被禁用",
		Expect(disabled.Disabled(ctx), Equal(true)),
		ExpectMust(func() error {
			return disabled.Init(ctx)
		}),
	)

	backend := &FileSystemBackend{
		Backend: strfmt.Endpoint{
			Scheme:   "file",
			Hostname: ".",
			Path:     "/tmp",
		},
		PathOverwrite:     "/",
		UsernameOverwrite: "user",
		PasswordOverwrite: "pass",
		ExtraOverwrite:    "debug=true",
	}
	err := backend.Init(ctx)
	_, injected := filesystem.Context.MayFrom(backend.InjectContext(ctx))
	Then(t, "初始化本地文件系统 backend",
		Expect(err, Equal[error](nil)),
		Expect(backend.Disabled(ctx), Equal(false)),
		Expect(backend.FileSystem() != nil, Equal(true)),
		Expect(injected, Equal(true)),
		Expect(backend.Backend.Username, Equal("")),
	)

	ftpBackend := &FileSystemBackend{
		Backend: strfmt.Endpoint{
			Scheme:   "ftp",
			Hostname: "127.0.0.1",
			Port:     21,
		},
	}
	ftpErr := ftpBackend.Init(ctx)
	Then(t, "初始化 FTP backend 不立即连接",
		Expect(ftpErr, Equal[error](nil)),
		Expect(ftpBackend.FileSystem() != nil, Equal(true)),
	)

	s3Backend := &FileSystemBackend{
		Backend: strfmt.Endpoint{
			Scheme:   "s3",
			Hostname: "127.0.0.1",
			Port:     9000,
			Path:     "/bucket",
			Extra: url.Values{
				"insecure":        []string{"true"},
				"skipBucketCheck": []string{"true"},
			},
		},
	}
	s3Err := s3Backend.Init(ctx)
	Then(t, "初始化 S3 backend 使用跳过 bucket 检查分支",
		Expect(s3Err, Equal[error](nil)),
		Expect(s3Backend.FileSystem() != nil, Equal(true)),
	)

	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`<D:multistatus xmlns:D="DAV:"></D:multistatus>`))
	}))
	defer svc.Close()
	u := MustValue(t, func() (*url.URL, error) {
		return url.Parse(svc.URL)
	})
	webdavBackend := &FileSystemBackend{
		Backend: strfmt.Endpoint{
			Scheme:   "webdav",
			Hostname: u.Hostname(),
			Path:     "/",
			Extra:    url.Values{"insecure": []string{"true"}},
		},
	}
	if port := u.Port(); port != "" {
		p, _ := strconv.ParseUint(port, 10, 16)
		webdavBackend.Backend.Port = uint16(p)
	}
	webdavErr := webdavBackend.Init(ctx)
	Then(t, "初始化 WebDAV backend",
		Expect(webdavErr, Equal[error](nil)),
		Expect(webdavBackend.FileSystem() != nil, Equal(true)),
	)
}

func regexpMust(pattern string) *regexp.Regexp {
	return regexp.MustCompile(pattern)
}

func TestFileSystemBackendErrors(t *testing.T) {
	ctx := context.Background()

	Then(t, "非法 extra 返回错误",
		ExpectDo(func() error {
			return (&FileSystemBackend{
				Backend:        strfmt.Endpoint{Scheme: "file", Path: "/tmp"},
				ExtraOverwrite: "%",
			}).Init(ctx)
		}, ErrorMatch(regexpMust("invalid URL escape"))),
	)

	Then(t, "不支持的 scheme 返回错误",
		ExpectDo(func() error {
			return (&FileSystemBackend{
				Backend: strfmt.Endpoint{Scheme: "unknown"},
			}).Init(ctx)
		}, ErrorMatch(regexpMust("unsupported"))),
	)

	values, err := url.ParseQuery("a=b")
	Then(t, "确认测试依赖的 query 解析可用",
		Expect(err, Equal[error](nil)),
		Expect(values.Get("a"), Equal("b")),
	)
}
