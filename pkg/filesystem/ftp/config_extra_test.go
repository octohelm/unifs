package ftp

import (
	"context"
	"encoding/base64"
	"net/url"
	"regexp"
	"testing"

	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/unifs/pkg/strfmt"
)

func TestConfig(t *testing.T) {
	conf := &Config{Endpoint: strfmt.Endpoint{
		Scheme:   "ftps",
		Hostname: "127.0.0.1",
		Port:     21,
		Path:     "/base",
		Username: "user",
		Password: "pass",
		Extra: url.Values{
			"timeout":            []string{"bad"},
			"enableDebug":        []string{"true"},
			"insecureSkipVerify": []string{"true"},
			"explicitTLS":        []string{"true"},
		},
	}}

	Then(t, "BasePath 返回 endpoint path",
		Expect(conf.BasePath(), Equal("/base")),
		ExpectDo(func() error {
			_, err := conf.Conn(context.Background())
			return err
		}, ErrorMatch(regexpMust("invalid duration"))),
	)
}

func TestConfigConnParseBranches(t *testing.T) {
	Then(t, "enableDebug 非法返回错误",
		ExpectDo(func() error {
			_, err := (&Config{Endpoint: strfmt.Endpoint{
				Scheme: "ftp",
				Extra: url.Values{
					"timeout":     []string{"1ms"},
					"enableDebug": []string{"bad"},
				},
			}}).Conn(context.Background())
			return err
		}, ErrorMatch(regexpMust("invalid syntax"))),
	)

	Then(t, "ftps bool 参数非法返回错误",
		ExpectDo(func() error {
			_, err := (&Config{Endpoint: strfmt.Endpoint{
				Scheme: "ftps",
				Extra: url.Values{
					"insecureSkipVerify": []string{"bad"},
				},
			}}).Conn(context.Background())
			return err
		}, ErrorMatch(regexpMust("invalid syntax"))),
		ExpectDo(func() error {
			_, err := (&Config{Endpoint: strfmt.Endpoint{
				Scheme: "ftps",
				Extra: url.Values{
					"insecureSkipVerify": []string{"true"},
					"explicitTLS":        []string{"bad"},
				},
			}}).Conn(context.Background())
			return err
		}, ErrorMatch(regexpMust("invalid syntax"))),
	)
}

func TestTLSCertificateErrors(t *testing.T) {
	Then(t, "证书 base64 非法时报错",
		ExpectDo(func() error {
			_, err := (TLS{CertData: "%"}).Certificate()
			return err
		}, ErrorMatch(regexpMust("illegal base64"))),
		ExpectDo(func() error {
			_, err := (TLS{
				CertData: base64.StdEncoding.EncodeToString([]byte("cert")),
				KeyData:  "%",
			}).Certificate()
			return err
		}, ErrorMatch(regexpMust("illegal base64"))),
	)
}

func regexpMust(pattern string) *regexp.Regexp {
	return regexp.MustCompile(pattern)
}
