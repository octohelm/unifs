package s3

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"testing"

	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/strfmt"
)

func TestConfigBranches(t *testing.T) {
	cached := &Config{fs: filesystem.NewMemFS()}
	fsys := MustValue(t, func() (filesystem.FileSystem, error) {
		return cached.AsFileSystem(context.Background())
	})

	conf := &Config{Endpoint: strfmt.Endpoint{Path: "/bucket/prefix"}}

	Then(t, "Config 基础分支",
		Expect(fsys != nil, Equal(true)),
		Expect(conf.Bucket(), Equal("bucket")),
		Expect(conf.Prefix(), Equal("/prefix")),
		Expect((&Config{Endpoint: strfmt.Endpoint{Path: "/bucket"}}).Prefix(), Equal("/")),
	)

	Then(t, "非法 presignAs 返回错误",
		ExpectDo(func() error {
			_, err := (&Config{
				Endpoint: strfmt.Endpoint{
					Scheme: "s3",
					Path:   "/bucket",
					Extra:  url.Values{"presignAs": []string{"://bad"}},
				},
			}).AsFileSystem(context.Background())
			return err
		}, ErrorMatch(regexpMust("missing protocol scheme"))),
	)
}

func regexpMust(pattern string) *regexp.Regexp {
	return regexp.MustCompile(pattern)
}

func TestFakeBucket(t *testing.T) {
	rt := &fakeBucket{name: "bucket", prefix: "/prefix"}
	req := MustValue(t, func() (*http.Request, error) {
		return http.NewRequest(http.MethodHead, "http://example.com/bucket/", nil)
	})
	resp := MustValue(t, func() (*http.Response, error) {
		return rt.RoundTrip(req)
	})
	prefixReq := MustValue(t, func() (*http.Request, error) {
		return http.NewRequest(http.MethodHead, "http://example.com/bucket/prefix", nil)
	})
	prefixResp := MustValue(t, func() (*http.Response, error) {
		return rt.RoundTrip(prefixReq)
	})

	Then(t, "fakeBucket 返回 bucket 存在",
		Expect(resp.StatusCode, Equal(http.StatusOK)),
		Expect(prefixResp.StatusCode, Equal(http.StatusOK)),
	)
}
