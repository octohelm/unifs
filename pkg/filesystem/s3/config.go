package s3

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	courierhttpclient "github.com/octohelm/courier/pkg/courierhttp/client"
	"github.com/rhnvrm/simples3"

	"github.com/innoai-tech/infra/pkg/http/middleware"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/strfmt"
)

type Config struct {
	Endpoint strfmt.Endpoint `flag:",upstream"`

	fs filesystem.FileSystem `flag:"-"`
}

func (c *Config) AsFileSystem(ctx context.Context) (filesystem.FileSystem, error) {
	if c.fs != nil {
		return c.fs, nil
	}

	insecure := false
	if c.Endpoint.Extra.Get("insecure") == "true" {
		insecure = true
	}

	var presignAs *url.URL

	if presignAsStr := c.Endpoint.Extra.Get("presignAs"); presignAsStr != "" {
		u, err := url.Parse(presignAsStr)
		if err != nil {
			return nil, fmt.Errorf("parse presignAs %q: %w", presignAsStr, err)
		}
		presignAs = u
	}

	region := c.Endpoint.Extra.Get("region")
	if region == "" {
		region = "us-east-1"
	}

	client := newClient(region, c.Endpoint.Username, c.Endpoint.Password, endpointURL(c.Endpoint.Host(), insecure))
	if c.Endpoint.Extra.Get("skipBucketCheck") == "true" {
		client.SetClient(&http.Client{
			Transport: &fakeBucket{
				name:   c.Bucket(),
				prefix: c.Prefix(),
			},
		})
	}

	f := &fs{
		s3Client:   client,
		httpClient: client.Client,
		bucket:     c.Bucket(),
		prefix:     c.Prefix(),
		region:     region,
	}

	if err := f.ensureBucket(ctx); err != nil {
		return nil, fmt.Errorf("check bucket %q: %w", c.Endpoint.Base(), err)
	}

	if presignAs != nil {
		clientForPresign := newClient(region, c.Endpoint.Username, c.Endpoint.Password, presignEndpointURL(presignAs))

		presignAs.Host = c.Bucket() + "." + presignAs.Host

		f.presignAs = presignAs
		f.s3ClientForPresign = clientForPresign
	}

	c.fs = f

	return c.fs, nil
}

func newClient(region string, accessKey string, secretKey string, endpoint string) *simples3.S3 {
	client := simples3.New(region, accessKey, secretKey)
	client.SetEndpoint(endpoint)
	return client
}

func endpointURL(host string, insecure bool) string {
	scheme := "https"
	if insecure {
		scheme = "http"
	}
	return scheme + "://" + host
}

func presignEndpointURL(u *url.URL) string {
	scheme := u.Scheme
	if scheme == "" {
		scheme = "https"
	}
	return scheme + "://" + u.Host
}

func (c *Config) Bucket() string {
	return c.Endpoint.Base()
}

func (c *Config) Prefix() string {
	n := len(c.Bucket() + "/")
	if len(c.Endpoint.Path) > n {
		return c.Endpoint.Path[n:]
	}
	return "/"
}

type fakeBucket struct {
	name   string
	prefix string
}

func (rt *fakeBucket) RoundTrip(req *http.Request) (*http.Response, error) {
	if (req.Method == http.MethodGet || req.Method == http.MethodHead) && req.URL.Path == "/"+rt.name+"/" {
		r := httptest.NewRecorder()
		r.WriteHeader(http.StatusOK)
		_, _ = r.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
	<GetBucketResult>
	  <Bucket>` + rt.name + `</Bucket>
	  <PublicAccessBlockEnabled>false</PublicAccessBlockEnabled>
	  <CreationDate>` + time.Now().Format(time.RFC3339) + `</CreationDate>
	</GetBucketResult>
	`)

		return r.Result(), nil
	}

	if req.URL.Path == "/"+rt.name+rt.prefix {
		resp := httptest.NewRecorder()
		resp.Header().Set("Last-Modified", time.Now().Format(rfc822TimeFormat))
		resp.WriteHeader(http.StatusOK)
		return resp.Result(), nil
	}

	cc := courierhttpclient.GetReasonableClientContext(req.Context(), middleware.NewLogRoundTripper())

	resp, err := cc.Transport.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("round trip %s %s: %w", req.Method, req.URL.String(), err)
	}

	if req.Method == http.MethodHead {
		if resp.StatusCode > http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			// force set 200
			resp.StatusCode = http.StatusOK
		}
	}

	resp.Header.Set("Last-Modified", time.Now().Format(rfc822TimeFormat))
	return resp, nil
}

const (
	rfc822TimeFormat = "Mon, 2 Jan 2006 15:04:05 GMT"
)
