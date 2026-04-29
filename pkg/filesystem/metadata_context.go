package filesystem

import (
	"context"

	"github.com/octohelm/courier/pkg/courier"
)

type metadataCtx struct{}

// MetadataFromContext 返回 ctx 中存储的 courier metadata。
func MetadataFromContext(ctx context.Context) courier.Metadata {
	if v, ok := ctx.Value(metadataCtx{}).(courier.Metadata); ok {
		return v
	}
	return courier.Metadata{}
}

// MetadataInjectContext 返回携带 courier metadata 的上下文。
func MetadataInjectContext(ctx context.Context, meta courier.Metadata) context.Context {
	return context.WithValue(ctx, metadataCtx{}, meta)
}
