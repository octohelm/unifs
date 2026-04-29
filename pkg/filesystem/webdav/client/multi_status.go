package client

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/fsutil"
)

// https://tools.ietf.org/html/rfc4918#section-14.16
type MultiStatus struct {
	XMLName             xml.Name    `xml:"DAV: multistatus"`
	Responses           []*Response `xml:"response"`
	ResponseDescription string      `xml:"responsedescription,omitempty"`
	SyncToken           string      `xml:"sync-token,omitempty"`
}

func NewMultiStatus(resps ...*Response) *MultiStatus {
	return &MultiStatus{Responses: resps}
}

// https://tools.ietf.org/html/rfc4918#section-14.24
type Response struct {
	XMLName             xml.Name   `xml:"DAV: response"`
	Hrefs               []Href     `xml:"href"`
	PropStats           []PropStat `xml:"propstat,omitempty"`
	ResponseDescription string     `xml:"responsedescription,omitempty"`
	Status              *Status    `xml:"status,omitempty"`
	Error               *Error     `xml:"error,omitempty"`
	Location            *Location  `xml:"location,omitempty"`
	Prefix              string     `xml:"-"`
}

func (resp *Response) FileInfo() (filesystem.FileInfo, error) {
	pathname, err := resp.Path()
	if err != nil {
		return nil, fmt.Errorf("response path: %w", err)
	}

	var resType ResourceType
	if err := resp.DecodeProp(&resType); err != nil {
		return nil, fmt.Errorf("resource type %q: %w", pathname, err)
	}

	if resType.Is(CollectionName) {
		return fsutil.NewDirFileInfo(path.Base(pathname)), nil
	}

	var getLen GetContentLength
	if err := resp.DecodeProp(&getLen); err != nil {
		return nil, fmt.Errorf("content length %q: %w", pathname, err)
	}

	var getLastModified GetLastModified
	if err := resp.DecodeProp(&getLastModified); err != nil && !IsNotFound(err) {
		return nil, fmt.Errorf("last modified %q: %w", pathname, err)
	}

	return fsutil.NewFileInfo(path.Base(pathname), getLen.Length, time.Time(getLastModified.LastModified)), nil
}

func NewOKResponse(path string) *Response {
	href := Href{Path: path}
	return &Response{
		Hrefs:  []Href{href},
		Status: &Status{Code: http.StatusOK},
	}
}

func NewErrorResponse(path string, err error) *Response {
	code := http.StatusInternalServerError
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		code = httpErr.Code
	}

	var errElt *Error
	errors.As(err, &errElt)

	href := Href{Path: path}
	return &Response{
		Hrefs:               []Href{href},
		Status:              &Status{Code: code},
		ResponseDescription: err.Error(),
		Error:               errElt,
	}
}

func (resp *Response) Err() error {
	if resp.Status == nil || resp.Status.Code/100 == 2 {
		return nil
	}

	var err error = resp.Error
	if resp.ResponseDescription != "" {
		if err != nil {
			err = fmt.Errorf("%s: %w", resp.ResponseDescription, err)
		} else {
			err = fmt.Errorf("%s", resp.ResponseDescription)
		}
	}

	return &HTTPError{
		Code: resp.Status.Code,
		Err:  err,
	}
}

func (resp *Response) Path() (string, error) {
	err := resp.Err()
	var path string
	if len(resp.Hrefs) == 1 {
		path = resp.Hrefs[0].Path
	} else if err == nil {
		err = fmt.Errorf("webdav: malformed response: expected exactly one href element, got %d", len(resp.Hrefs))
	}

	if path != "" && (resp.Prefix != "" && resp.Prefix != "/") {
		path = strings.TrimPrefix(path, resp.Prefix)
	}

	if err != nil {
		return path, fmt.Errorf("resolve response path: %w", err)
	}
	return path, nil
}

func (resp *Response) DecodeProp(values ...any) error {
	for _, v := range values {
		// TODO 为错误补充更多上下文，例如 XML name。
		name, err := valueXMLName(v)
		if err != nil {
			return fmt.Errorf("resolve property XML name for %T: %w", v, err)
		}
		if err := resp.Err(); err != nil {
			return newPropError(name, err)
		}
		for _, propstat := range resp.PropStats {
			raw := propstat.Prop.Get(name)
			if raw == nil {
				continue
			}
			if err := propstat.Status.Err(); err != nil {
				return newPropError(name, err)
			}
			if err := raw.Decode(v); err != nil {
				return newPropError(name, err)
			}
			return nil
		}
		return newPropError(name, &HTTPError{
			Code: http.StatusNotFound,
			Err:  fmt.Errorf("missing property"),
		})
	}

	return nil
}

func newPropError(name xml.Name, err error) error {
	return fmt.Errorf("property <%s %s>: %w", name.Space, name.Local, err)
}

func (resp *Response) EncodeProp(code int, v any) error {
	raw, err := EncodeRawXMLElement(v)
	if err != nil {
		return fmt.Errorf("encode property: %w", err)
	}

	for i := range resp.PropStats {
		propstat := &resp.PropStats[i]
		if propstat.Status.Code == code {
			propstat.Prop.Raw = append(propstat.Prop.Raw, *raw)
			return nil
		}
	}

	resp.PropStats = append(resp.PropStats, PropStat{
		Status: Status{Code: code},
		Prop:   Prop{Raw: []RawXMLValue{*raw}},
	})

	return nil
}
