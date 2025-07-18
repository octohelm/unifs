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
	ResponseDescription string      `xml:"responsedescription,omitzero"`
	SyncToken           string      `xml:"sync-token,omitzero"`
}

func NewMultiStatus(resps ...*Response) *MultiStatus {
	return &MultiStatus{Responses: resps}
}

// https://tools.ietf.org/html/rfc4918#section-14.24
type Response struct {
	XMLName             xml.Name   `xml:"DAV: response"`
	Hrefs               []Href     `xml:"href"`
	PropStats           []PropStat `xml:"propstat,omitzero"`
	ResponseDescription string     `xml:"responsedescription,omitzero"`
	Status              *Status    `xml:"status,omitzero"`
	Error               *Error     `xml:"error,omitzero"`
	Location            *Location  `xml:"location,omitzero"`
	Prefix              string     `xml:"-"`
}

func (resp *Response) FileInfo() (filesystem.FileInfo, error) {
	pathname, err := resp.Path()
	if err != nil {
		return nil, err
	}

	var resType ResourceType
	if err := resp.DecodeProp(&resType); err != nil {
		return nil, err
	}

	if resType.Is(CollectionName) {
		return fsutil.NewDirFileInfo(path.Base(pathname)), nil
	}

	var getLen GetContentLength
	if err := resp.DecodeProp(&getLen); err != nil {
		return nil, err
	}

	var getLastModified GetLastModified
	if err := resp.DecodeProp(&getLastModified); err != nil && !IsNotFound(err) {
		return nil, err
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
			err = fmt.Errorf("%v (%w)", resp.ResponseDescription, err)
		} else {
			err = fmt.Errorf("%v", resp.ResponseDescription)
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
		err = fmt.Errorf("webdav: malformed response: expected exactly one href element, got %v", len(resp.Hrefs))
	}

	if path != "" && (resp.Prefix != "" && resp.Prefix != "/") {
		path = strings.TrimPrefix(path, resp.Prefix)
	}

	return path, err
}

func (resp *Response) DecodeProp(values ...interface{}) error {
	for _, v := range values {
		// TODO wrap errors with more context (XML name)
		name, err := valueXMLName(v)
		if err != nil {
			return err
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
	return fmt.Errorf("property <%v %v>: %w", name.Space, name.Local, err)
}

func (resp *Response) EncodeProp(code int, v interface{}) error {
	raw, err := EncodeRawXMLElement(v)
	if err != nil {
		return err
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
