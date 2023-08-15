package s3

import (
	"context"
	"fmt"
	"net/http"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/octohelm/unifs/pkg/strfmt"
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
	if c.Endpoint.Extra.Get("insecure") == "true" || c.Endpoint.Extra.Get("insecure") == "true" {
		insecure = true
	}

	o := &minio.Options{
		Creds:  credentials.NewStaticV4(c.Endpoint.Username, c.Endpoint.Password, ""),
		Secure: !insecure,
		//Transport: &logRoundTripper{
		//	nextRoundTripper: &http.Transport{},
		//},
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
