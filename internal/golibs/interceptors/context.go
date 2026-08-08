package interceptors

import "context"

type contextKey string

const (
	MerchantIDKey   contextKey = "merchant_id"
	ResourcePathKey contextKey = "resource_path"
	APIPublicKeyKey contextKey = "api_context_key"
)

func InjectMerchantContext(ctx context.Context, merchantID int64, resourcePath string, publicKey string) context.Context {
	ctx = context.WithValue(ctx, MerchantIDKey, merchantID)
	ctx = context.WithValue(ctx, ResourcePathKey, resourcePath)
	ctx = context.WithValue(ctx, APIPublicKeyKey, publicKey)
	return ctx
}

func GetMerchantIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(MerchantIDKey).(int64)
	return id, ok
}

func GetResourcePathFromContext(ctx context.Context) string {
	if path, ok := ctx.Value(ResourcePathKey).(string); ok && path != "" {
		return path
	}
	return "/org/default/"
}
