package ftp

import (
	"context"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/octohelm/unifs/pkg/strfmt"
)

type Config struct {
	Endpoint strfmt.Endpoint `flag:",upstream"`

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
				return nil, err
			}
			p.ConnectTimeout = d
		}

		if t := c.Endpoint.Extra.Get("maxConnections"); t != "" {
			d, err := strconv.ParseInt(t, 10, 64)
			if err != nil {
				return nil, err
			}
			p.MaxConnections = int32(d)
		}

		c.p = p
	}

	return c.p.Conn(ctx, args...)
}
