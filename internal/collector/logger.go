package collector

import "context"

type Logger interface {
	Printf(format string, args ...any)
}

type loggerKey struct{}

func WithLogger(ctx context.Context, l Logger) context.Context {
	if l == nil {
		return ctx
	}
	return context.WithValue(ctx, loggerKey{}, l)
}

func loggerFromContext(ctx context.Context) Logger {
	if ctx == nil {
		return nil
	}
	if v := ctx.Value(loggerKey{}); v != nil {
		if l, ok := v.(Logger); ok {
			return l
		}
	}
	return nil
}

