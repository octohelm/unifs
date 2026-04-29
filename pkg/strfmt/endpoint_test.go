package strfmt

import (
	"net/url"
	"regexp"
	"testing"

	. "github.com/octohelm/x/testing/v2"
)

func TestEndpoint(t *testing.T) {
	endpoint := MustValue(t, func() (*Endpoint, error) {
		return ParseEndpoint("s3s://user:secret@example.com:9443/bucket/prefix?region=ap-southeast-1")
	})

	Then(t, "解析 endpoint 字段",
		Expect(*endpoint, Equal(Endpoint{
			Scheme:   "s3s",
			Hostname: "example.com",
			Port:     9443,
			Path:     "/bucket/prefix",
			Username: "user",
			Password: "secret",
			Extra: url.Values{
				"region": []string{"ap-southeast-1"},
			},
		})),
		Expect(endpoint.Base(), Equal("bucket")),
		Expect(endpoint.Host(), Equal("example.com:9443")),
		Expect(endpoint.IsZero(), Equal(false)),
		Expect(endpoint.IsTLS(), Equal(true)),
		Expect(endpoint.SecurityString(), Equal("s3s://user:------@example.com:9443/bucket/prefix?region=ap-southeast-1")),
	)

	text := MustValue(t, endpoint.MarshalText)
	Then(t, "marshal endpoint 文本",
		Expect(string(text), Equal("s3s://user:secret@example.com:9443/bucket/prefix?region=ap-southeast-1")),
	)

	var decoded Endpoint
	Must(t, func() error {
		return decoded.UnmarshalText(text)
	})
	Then(t, "unmarshal endpoint 文本",
		Expect(decoded, Equal(*endpoint)),
	)
}

func TestEndpointZeroAndInvalid(t *testing.T) {
	Then(t, "空 endpoint 判断为 zero",
		Expect(Endpoint{}.IsZero(), Equal(true)),
		Expect(Endpoint{Hostname: "example.com"}.Host(), Equal("example.com")),
		Expect(Endpoint{Path: "/only"}.Base(), Equal("only")),
		Expect(Endpoint{}.Base(), Equal("")),
		Expect((&Endpoint{}).IsTLS(), Equal(false)),
	)

	Then(t, "非法 endpoint 返回错误",
		ExpectDo(func() error {
			_, err := ParseEndpoint("://bad")
			return err
		}, ErrorMatch(regexp.MustCompile("missing protocol scheme"))),
	)
}
