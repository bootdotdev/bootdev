package version

import "context"

var ContextKey = struct{ string }{"version"}

func WithContext(ctx context.Context, version *VersionInfo) context.Context {
	return context.WithValue(ctx, ContextKey, version)
}

func FromContext(ctx context.Context) *VersionInfo {
	if c, ok := ctx.Value(ContextKey).(*VersionInfo); ok {
		return c
	}

	return nil
}
