package ftp

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/octohelm/unifs/pkg/strfmt"
)

type TLS struct {
	CertData string
	KeyData  string
}

func (x TLS) Certificate() (tls.Certificate, error) {
	cert, err := base64.StdEncoding.DecodeString(x.CertData)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("decode TLS cert data: %w", err)
	}
	key, err := base64.StdEncoding.DecodeString(x.KeyData)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("decode TLS key data: %w", err)
	}
	certificate, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("parse TLS key pair: %w", err)
	}
	return certificate, nil
}

type Config struct {
	Endpoint strfmt.Endpoint `flag:",upstream"`
	TLS      TLS             `json:"tls,omitzero"`

	p  *Pool
	mu sync.Mutex
}

func (c *Config) BasePath() string {
	return c.Endpoint.Path
}

func (c *Config) Conn(ctx context.Context, args ...any) (Conn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.p == nil {
		p := &Pool{
			Addr: c.Endpoint.Host(),
		}

		if c.Endpoint.Username != "" {
			p.Auth = url.UserPassword(c.Endpoint.Username, c.Endpoint.Password)
		}

		p.ConnectTimeout = 5 * time.Second

		if t := c.Endpoint.Extra.Get("timeout"); t != "" {
			d, err := time.ParseDuration(t)
			if err != nil {
				return nil, fmt.Errorf("parse FTP timeout %q: %w", t, err)
			}
			p.ConnectTimeout = d
		}

		if t := c.Endpoint.Extra.Get("enableDebug"); t != "" {
			d, err := strconv.ParseBool(t)
			if err != nil {
				return nil, fmt.Errorf("parse FTP enableDebug %q: %w", t, err)
			}
			p.EnableDebug = d
		}

		if c.Endpoint.Scheme == "ftps" {
			p.TLSConfig = &tls.Config{}

			for _, bound := range []struct {
				name  string
				value *uint16
			}{
				{name: "tlsMinVersion", value: &p.TLSConfig.MinVersion},
				{name: "tlsMaxVersion", value: &p.TLSConfig.MaxVersion},
			} {
				if c.Endpoint.Extra.Has(bound.name) {
					version, err := parseTLSVersion(c.Endpoint.Extra.Get(bound.name))
					if err != nil {
						return nil, fmt.Errorf("解析 FTP %s: %w", bound.name, err)
					}
					*bound.value = version
				}
			}
			if p.TLSConfig.MinVersion != 0 && p.TLSConfig.MaxVersion != 0 && p.TLSConfig.MinVersion > p.TLSConfig.MaxVersion {
				return nil, fmt.Errorf("FTP tlsMinVersion 不能大于 tlsMaxVersion")
			}

			if t := c.Endpoint.Extra.Get("insecureSkipVerify"); t != "" {
				d, err := strconv.ParseBool(t)
				if err != nil {
					return nil, fmt.Errorf("parse FTP insecureSkipVerify %q: %w", t, err)
				}
				p.TLSConfig.InsecureSkipVerify = d
			}

			if t := c.Endpoint.Extra.Get("explicitTLS"); t != "" {
				d, err := strconv.ParseBool(t)
				if err != nil {
					return nil, fmt.Errorf("parse FTP explicitTLS %q: %w", t, err)
				}
				p.ExplicitTLS = d
			}
		}

		c.p = p
	}

	return c.p.Conn(ctx, args...)
}

// parseTLSVersion 将 1.x 格式转换为 TLS 版本编号，不限定当前 Go 支持的版本。
func parseTLSVersion(value string) (uint16, error) {
	minor, ok := strings.CutPrefix(value, "1.")
	if !ok || minor == "" {
		return 0, fmt.Errorf("TLS 版本 %q 格式无效，应为 1.x", value)
	}
	for _, digit := range minor {
		if digit < '0' || digit > '9' {
			return 0, fmt.Errorf("TLS 版本 %q 格式无效，应为 1.x", value)
		}
	}
	n, err := strconv.ParseUint(minor, 10, 16)
	if err != nil || n > (1<<16-1)-uint64(tls.VersionTLS10) {
		return 0, fmt.Errorf("TLS 版本 %q 超出 uint16 可表示的范围", value)
	}
	return tls.VersionTLS10 + uint16(n), nil
}
