package client

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
)

type Client interface {
	MkCol(ctx context.Context, name string) error
	PropFind(ctx context.Context, path string, depth Depth, propfind *PropFind) (*MultiStatus, error)
	Move(ctx context.Context, src string, dest string, overwrite bool) error
	Delete(ctx context.Context, name string) error
	OpenRead(ctx context.Context, name string) (io.ReadCloser, error)
	OpenWrite(ctx context.Context, name string) (io.WriteCloser, error)
}

func NewClient(endpoint string) Client {
	return &client{
		endpoint: endpoint,
	}
}

type client struct {
	endpoint string
}

func (c *client) OpenRead(ctx context.Context, name string) (io.ReadCloser, error) {
	req, err := c.req(ctx, http.MethodGet, name, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (c *client) OpenWrite(ctx context.Context, name string) (io.WriteCloser, error) {
	pr, pw := io.Pipe()
	req, err := c.req(ctx, http.MethodPut, name, pr)
	if err != nil {
		_ = pw.Close()
		return nil, err
	}

	done := make(chan error, 1)
	go func() {
		resp, err := c.do(req)
		if err != nil {
			done <- err
			return
		}
		done <- resp.Body.Close()
	}()

	return &fileWriter{pw, done}, nil
}

type fileWriter struct {
	pw   *io.PipeWriter
	done <-chan error
}

func (fw *fileWriter) Write(b []byte) (int, error) {
	return fw.pw.Write(b)
}

func (fw *fileWriter) Close() error {
	if err := fw.pw.Close(); err != nil {
		return err
	}
	return <-fw.done
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
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, strings.TrimLeft(p, "/"))
	return u, nil
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
	return &ms, nil
}
