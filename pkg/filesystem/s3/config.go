package s3

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/octohelm/unifs/pkg/strfmt"
)

type Config struct {
	Endpoint strfmt.Endpoint `flag:",upstream"`
	Bucket   string

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

	client, err := minio.New(
		c.Endpoint.Host(),
		&minio.Options{
			Creds:  credentials.NewStaticV4(c.Endpoint.Username, c.Endpoint.Password, ""),
			Secure: !insecure,
		},
	)
	if err != nil {
		return nil, err
	}

	ok, err := client.BucketExists(ctx, c.Bucket)
	if err != nil {
		return nil, err
	}

	if !ok {
		_ = client.MakeBucket(ctx, c.Bucket, minio.MakeBucketOptions{})
	}

	c.c = client
	return client, nil
}
