package webdav

import (
	"context"
	"net/url"

	"github.com/pkg/errors"

	"github.com/octohelm/unifs/pkg/filesystem/webdav/client"
	"github.com/octohelm/unifs/pkg/strfmt"
)

type Config struct {
	Endpoint strfmt.Endpoint `flag:",upstream"`

	c client.Client
}

func (c *Config) Client(ctx context.Context) (client.Client, error) {
	if c.c != nil {
		return c.c, nil
	}

	u := &url.URL{}
	u.Scheme = "https"

	if c.Endpoint.Extra.Get("insecure") == "true" || c.Endpoint.Extra.Get("insecure") == "true" {
		u.Scheme = "http"
	}

	u.Host = c.Endpoint.Host()
	u.Path = c.Endpoint.Path

	if c.Endpoint.Username != "" {
		u.User = url.UserPassword(c.Endpoint.Username, c.Endpoint.Password)
	}

	c.c, _ = client.NewClient(u.String())

	_, err := c.c.PropFind(ctx, "/", 0, nil)
	if err != nil {
		return nil, err
	}

	if c.Endpoint.Path != "" {
		err := c.c.MkCol(ctx, "/")
		if err != nil {
			return nil, errors.Wrap(err, "MkCol failed")
		}
	}

	return c.c, nil
}
