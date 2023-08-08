package client

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

type Client interface {
	MkCol(ctx context.Context, name string) error
	PropFind(ctx context.Context, path string, depth Depth, propfind *PropFind) (*MultiStatus, error)
	Move(ctx context.Context, src string, dest string, overwrite bool) error
	Delete(ctx context.Context, name string) error

	OpenWrite(ctx context.Context, name string) (io.WriteCloser, error)
	Open(ctx context.Context, name string) (File, error)
}

func NewClient(endpoint string) (Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	return &client{endpoint: u}, nil
}

type client struct {
	endpoint *url.URL
}

func (c *client) Open(ctx context.Context, name string) (File, error) {
	ms, err := c.PropFind(ctx, name, 0, FileInfoPropFind)
	if err != nil {
		return nil, err
	}

	// If the client followed a redirect, the Href might be different from the request path
	if len(ms.Responses) != 1 {
		return nil, fmt.Errorf("PROPFIND with Depth: 0 returned %d responses", len(ms.Responses))
	}

	info, err := ms.Responses[0].FileInfo()
	if err != nil {
		if IsNotFound(err) {
			return nil, &os.PathError{
				Op:   "stat",
				Path: name,
				Err:  os.ErrNotExist,
			}
		}
		return nil, err
	}

	f := &file{
		info: info,
		doRequest: func(offset int64, end int64) (io.ReadCloser, error) {
			req, err := c.req(ctx, http.MethodGet, name, nil)
			if err != nil {
				return nil, err
			}

			setRange(req.Header, offset, end)

			resp, err := c.do(req)
			if err != nil {
				return nil, err
			}

			if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
				return nil, io.EOF
			}

			return resp.Body, nil
		},
	}

	return f, nil
}

func setRange(o http.Header, start, end int64) {
	switch {
	case start == 0 && end < 0:
		// Read last '-end' bytes. `bytes=-N`.
		o.Set("Range", fmt.Sprintf("bytes=%d", end))
	case 0 < start && end == 0:
		// Read everything starting from offset
		// 'start'. `bytes=N-`.
		o.Set("Range", fmt.Sprintf("bytes=%d-", start))
	case 0 <= start && start <= end:
		// Read everything starting at 'start' till the
		// 'end'. `bytes=N-M`
		o.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	default:
	}
	return
}

func (c *client) OpenWrite(ctx context.Context, name string) (io.WriteCloser, error) {
	pr, pw := io.Pipe()
	go func() {
		req, err := c.req(ctx, http.MethodPut, name, pr)
		if err != nil {
			return
		}
		resp, err := c.do(req)
		if err != nil {
			return
		}
		_ = resp.Body.Close()
	}()

	return pw, nil
}

func (c *client) Move(ctx context.Context, src string, dest string, overwrite bool) error {
	d, err := c.ResolveHref(dest)
	if err != nil {
		return err
	}

	r, err := c.req(ctx, "MOVE", src, nil)
	if err != nil {
		return err
	}

	r.Header.Set("Destination", d.String())
	r.Header.Set("Overwrite", FormatOverwrite(overwrite))

	return c.doSimple(r)
}

func (c *client) Delete(ctx context.Context, name string) error {
	r, err := c.req(ctx, "DELETE", name, nil)
	if err != nil {
		return err
	}
	err = c.doSimple(r)
	if err != nil {
		if IsNotFound(err) {
			// 404 means deleted
			return nil
		}
		return err
	}
	return nil
}

func (c *client) MkCol(ctx context.Context, name string) error {
	r, err := c.req(ctx, "MKCOL", name, nil)
	if err != nil {
		return err
	}
	return c.doSimple(r)
}

func (c *client) PropFind(ctx context.Context, path string, depth Depth, propfind *PropFind) (*MultiStatus, error) {
	if propfind == nil {
		propfind = FileInfoPropFind
	}

	r, err := c.reqXML(ctx, "PROPFIND", path, propfind)
	if err != nil {
		return nil, err
	}
	r.Header.Add("Depth", depth.String())
	return c.doMultiStatus(r)
}

func (c *client) reqXML(ctx context.Context, method string, path string, v any) (*http.Request, error) {
	buf := bytes.NewBuffer(nil)
	buf.WriteString(xml.Header)
	if err := xml.NewEncoder(buf).Encode(v); err != nil {
		return nil, err
	}

	req, err := c.req(ctx, method, path, buf)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", mime.FormatMediaType("text/xml", map[string]string{
		"charset": "utf-8",
	}))

	return req, nil
}

func (c *client) req(ctx context.Context, method string, path string, reader io.Reader) (*http.Request, error) {
	h, err := c.ResolveHref(path)
	if err != nil {
		return nil, err
	}
	return http.NewRequestWithContext(ctx, method, h.String(), reader)
}

func (c *client) ResolveHref(p string) (*url.URL, error) {
	u := *c.endpoint
	u.Path = path.Join(u.Path, strings.TrimLeft(p, "/"))
	return &u, nil
}

func (c *client) do(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req)
}

func (c *client) doSimple(req *http.Request) error {
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return &HTTPError{
			Code: resp.StatusCode,
		}
	}
	return nil
}

func (c *client) doMultiStatus(req *http.Request) (*MultiStatus, error) {
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		return NewMultiStatus(NewErrorResponse(
			req.URL.Path,
			&HTTPError{
				Code: resp.StatusCode,
			},
		)), nil
	}

	// TODO: the response can be quite large, support streaming Response elements
	var ms MultiStatus
	if err := xml.NewDecoder(resp.Body).Decode(&ms); err != nil {
		return nil, err
	}

	for _, r := range ms.Responses {
		r.Prefix = c.endpoint.Path
	}

	return &ms, nil
}
