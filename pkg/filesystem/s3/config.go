package s3

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/octohelm/unifs/pkg/filesystem"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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
			return nil, err
		}
		presignAs = u
	}

	signatureType := credentials.SignatureV4
	if signatureTypeStr := c.Endpoint.Extra.Get("signatureType"); signatureTypeStr != "" {
		switch signatureTypeStr {
		case "v2":
			signatureType = credentials.SignatureV2
		}
	}

	o := &minio.Options{
		Creds:  credentials.NewStatic(c.Endpoint.Username, c.Endpoint.Password, "", signatureType),
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
		return nil, fmt.Errorf("new s3 client failed: %w", err)
	}

	ok, err := client.BucketExists(ctx, c.Endpoint.Base())
	if err != nil {
		return nil, fmt.Errorf("check bucket failed %w", err)
	}
	if !ok {
		_ = client.MakeBucket(ctx, c.Endpoint.Base(), minio.MakeBucketOptions{})
	}

	f := &fs{
		s3Client: client,
		bucket:   c.Bucket(),
		prefix:   c.Prefix(),
	}

	if presignAs != nil {
		clientForPresign, err := minio.New(presignAs.Host, o)
		if err != nil {
			return nil, fmt.Errorf("new s3 client failed: %w", err)
		}

		presignAs.Host = c.Bucket() + "." + presignAs.Host

		f.presignAs = presignAs
		f.s3ClientForPresign = clientForPresign
	}

	c.fs = f

	return c.fs, nil
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
