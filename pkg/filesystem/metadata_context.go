package filesystem

import (
	"context"

	"github.com/octohelm/courier/pkg/courier"
)

type metadataCtx struct{}

func MetadataFromContext(ctx context.Context) courier.Metadata {
	if v, ok := ctx.Value(metadataCtx{}).(courier.Metadata); ok {
		return v
	}
	return courier.Metadata{}
}

func MetadataInjectContext(ctx context.Context, meta courier.Metadata) context.Context {
	return context.WithValue(ctx, metadataCtx{}, meta)
}
