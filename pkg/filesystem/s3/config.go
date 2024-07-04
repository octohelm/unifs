package s3

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/octohelm/unifs/pkg/strfmt"
	"net/http"
	"net/http/httptest"
	"time"
)

type Config struct {
	Endpoint strfmt.Endpoint `flag:",upstream"`

	c *minio.Client `flag:"-"`
}

func (c *Config) Client(ctx context.Context) (*minio.Client, error) {
	if c.c != nil {
		return c.c, nil
	}

	insecure := false
	if c.Endpoint.Extra.Get("insecure") == "true" {
		insecure = true
	}

	o := &minio.Options{
		Creds:  credentials.NewStaticV4(c.Endpoint.Username, c.Endpoint.Password, ""),
		Secure: !insecure,
	}

	if c.Endpoint.Extra.Get("skipBucketCheck") == "true" {
		o.Transport = &fakeBucket{
			name:             c.Bucket(),
			prefix:           c.Prefix(),
			nextRoundTripper: &http.Transport{},
		}
	}

	client, err := minio.New(c.Endpoint.Host(), o)
	if err != nil {
		return nil, err
	}

	ok, err := client.BucketExists(ctx, c.Endpoint.Base())
	if err != nil {
		return nil, err
	}

	if !ok {
		_ = client.MakeBucket(ctx, c.Endpoint.Base(), minio.MakeBucketOptions{})
	}

	c.c = client

	return client, nil
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
	nextRoundTripper http.RoundTripper
	name             string
	prefix           string
}

func (rt *fakeBucket) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodGet && req.URL.Path == "/"+rt.name+"/" {
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

	resp, err := rt.nextRoundTripper.RoundTrip(req)
	if err != nil {
		return resp, nil
	}
	resp.Header.Set("Last-Modified", time.Now().Format(rfc822TimeFormat))
	return resp, nil
}

const (
	rfc822TimeFormat = "Mon, 2 Jan 2006 15:04:05 GMT"
)

type logRoundTripper struct {
	nextRoundTripper http.RoundTripper
}

func (rt *logRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := rt.nextRoundTripper.RoundTrip(req)

	if err == nil {
		fmt.Println(req.Method, req.URL.String(), resp.StatusCode)
	}

	return resp, err
}
