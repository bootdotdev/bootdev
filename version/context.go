package version

import "context"

var contextKey = struct{ string }{"version"}

func WithContext(ctx context.Context, version *VersionInfo) context.Context {
	return context.WithValue(ctx, contextKey, version)
}

func FromContext(ctx context.Context) *VersionInfo {
	if c, ok := ctx.Value(contextKey).(*VersionInfo); ok {
		return c
	}

	return nil
}
