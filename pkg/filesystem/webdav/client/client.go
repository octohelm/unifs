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
		return nil, fmt.Errorf("parse webdav endpoint %q: %w", endpoint, err)
	}

	return &client{endpoint: u}, nil
}

type client struct {
	endpoint *url.URL
}

func (c *client) Open(ctx context.Context, name string) (File, error) {
	ms, err := c.PropFind(ctx, name, 0, FileInfoPropFind)
	if err != nil {
		return nil, fmt.Errorf("propfind webdav %q: %w", name, err)
	}

	// 如果客户端跟随了重定向，Href 可能不同于请求路径。
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
		return nil, fmt.Errorf("webdav fileinfo %q: %w", name, err)
	}

	f := &file{
		info: info,
		doRequest: func(offset int64, end int64) (io.ReadCloser, error) {
			req, err := c.req(ctx, http.MethodGet, name, nil)
			if err != nil {
				return nil, fmt.Errorf("create GET request %q: %w", name, err)
			}

			setRange(req.Header, offset, end)

			resp, err := c.do(req)
			if err != nil {
				return nil, fmt.Errorf("GET %q: %w", name, err)
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
		// 读取末尾指定数量的字节，格式为 `bytes=-N`。
		o.Set("Range", fmt.Sprintf("bytes=%d", end))
	case 0 < start && end == 0:
		// 从起始偏移开始读取全部内容，格式为 `bytes=N-`。
		o.Set("Range", fmt.Sprintf("bytes=%d-", start))
	case 0 <= start && start <= end:
		// 从起始偏移读取到结束偏移，格式为 `bytes=N-M`。
		o.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	default:
		return
	}
}

func (c *client) OpenWrite(ctx context.Context, name string) (io.WriteCloser, error) {
	pr, pw := io.Pipe()
	done := make(chan error, 1)
	go func() {
		req, err := c.req(ctx, http.MethodPut, name, pr)
		if err != nil {
			err = fmt.Errorf("create PUT request %q: %w", name, err)
			_ = pw.CloseWithError(err)
			done <- err
			return
		}
		resp, err := c.do(req)
		if err != nil {
			err = fmt.Errorf("PUT %q: %w", name, err)
			_ = pw.CloseWithError(err)
			done <- err
			return
		}
		if resp.StatusCode >= http.StatusBadRequest {
			err = &HTTPError{Code: resp.StatusCode}
			_ = pw.CloseWithError(err)
			_ = resp.Body.Close()
			done <- err
			return
		}
		if err := resp.Body.Close(); err != nil {
			done <- fmt.Errorf("close PUT response body %q: %w", name, err)
			return
		}
		done <- nil
	}()

	return &putWriteCloser{
		PipeWriter: pw,
		done:       done,
	}, nil
}

type putWriteCloser struct {
	*io.PipeWriter
	done <-chan error
}

func (w *putWriteCloser) Close() error {
	if err := w.PipeWriter.Close(); err != nil {
		return fmt.Errorf("close PUT request body: %w", err)
	}
	if err := <-w.done; err != nil {
		return fmt.Errorf("finish PUT request: %w", err)
	}
	return nil
}

func (c *client) Move(ctx context.Context, src string, dest string, overwrite bool) error {
	d, err := c.ResolveHref(dest)
	if err != nil {
		return fmt.Errorf("resolve move destination %q: %w", dest, err)
	}

	r, err := c.req(ctx, "MOVE", src, nil)
	if err != nil {
		return fmt.Errorf("create MOVE request %q: %w", src, err)
	}

	r.Header.Set("Destination", d.String())
	r.Header.Set("Overwrite", FormatOverwrite(overwrite))

	if err := c.doSimple(r); err != nil {
		return fmt.Errorf("MOVE %q to %q: %w", src, dest, err)
	}
	return nil
}

func (c *client) Delete(ctx context.Context, name string) error {
	r, err := c.req(ctx, "DELETE", name, nil)
	if err != nil {
		return fmt.Errorf("create DELETE request %q: %w", name, err)
	}
	err = c.doSimple(r)
	if err != nil {
		if IsNotFound(err) {
			// 404 means deleted
			return nil
		}
		return fmt.Errorf("DELETE %q: %w", name, err)
	}
	return nil
}

func (c *client) MkCol(ctx context.Context, name string) error {
	r, err := c.req(ctx, "MKCOL", name, nil)
	if err != nil {
		return fmt.Errorf("create MKCOL request %q: %w", name, err)
	}
	if err := c.doSimple(r); err != nil {
		return fmt.Errorf("MKCOL %q: %w", name, err)
	}
	return nil
}

func (c *client) PropFind(ctx context.Context, path string, depth Depth, propfind *PropFind) (*MultiStatus, error) {
	if propfind == nil {
		propfind = FileInfoPropFind
	}

	r, err := c.reqXML(ctx, "PROPFIND", path, propfind)
	if err != nil {
		return nil, fmt.Errorf("create PROPFIND request %q: %w", path, err)
	}
	r.Header.Add("Depth", depth.String())
	ms, err := c.doMultiStatus(r)
	if err != nil {
		return nil, fmt.Errorf("PROPFIND %q: %w", path, err)
	}
	return ms, nil
}

func (c *client) reqXML(ctx context.Context, method string, path string, v any) (*http.Request, error) {
	buf := bytes.NewBuffer(nil)
	buf.WriteString(xml.Header)
	if err := xml.NewEncoder(buf).Encode(v); err != nil {
		return nil, fmt.Errorf("encode %s XML request body: %w", method, err)
	}

	req, err := c.req(ctx, method, path, buf)
	if err != nil {
		return nil, fmt.Errorf("create %s XML request %q: %w", method, path, err)
	}

	req.Header.Add("Content-Type", mime.FormatMediaType("text/xml", map[string]string{
		"charset": "utf-8",
	}))

	return req, nil
}

func (c *client) req(ctx context.Context, method string, path string, reader io.Reader) (*http.Request, error) {
	h, err := c.ResolveHref(path)
	if err != nil {
		return nil, fmt.Errorf("resolve href %q: %w", path, err)
	}
	req, err := http.NewRequestWithContext(ctx, method, h.String(), reader)
	if err != nil {
		return nil, fmt.Errorf("new %s request %q: %w", method, h.String(), err)
	}
	return req, nil
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
		return fmt.Errorf("%s %s: %w", req.Method, req.URL.Path, err)
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
		return nil, fmt.Errorf("%s %s: %w", req.Method, req.URL.Path, err)
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

	// TODO 响应体可能很大，后续支持流式处理 Response 元素。
	var ms MultiStatus
	if err := xml.NewDecoder(resp.Body).Decode(&ms); err != nil {
		return nil, fmt.Errorf("decode multistatus response %q: %w", req.URL.Path, err)
	}

	for _, r := range ms.Responses {
		r.Prefix = c.endpoint.Path
	}

	return &ms, nil
}
