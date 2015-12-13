package web

import "golang.org/x/net/context"

func WithDevMode(ctx context.Context, devmode bool) context.Context {
	return context.WithValue(ctx, "web:devmode", devmode)
}

func DevMode(ctx context.Context) bool {
	raw := ctx.Value("web:devmode")
	if raw == nil {
		return false
	}
	return raw.(bool)
}
