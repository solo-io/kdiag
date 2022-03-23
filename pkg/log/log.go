package log

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const loggerKey = iota

func NewContext(ctx context.Context, fields ...zapcore.Field) context.Context {
	return context.WithValue(ctx, loggerKey, WithContext(ctx).With(fields...))
}

func InitialContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, loggerKey, New())
}

func InitialCmdContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, loggerKey, NewCmd())
}

// WithContext returns a logger from the given context
func WithContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return nil
	}
	if ctxLogger, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
		return ctxLogger
	}
	return nil
}

func New() *zap.Logger {
	return NewDev()
}

func NewCmd() *zap.Logger {
	if os.Getenv("DEBUG") != "" {
		return NewDev()
	}
	return NewNull()
}

func NewNull() *zap.Logger {
	return zap.NewNop()
}

func NewDev() *zap.Logger {
	config := zap.NewDevelopmentConfig()
	l, err := config.Build(
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
	)
	if err != nil {
		panic(err)
	}
	return l
}

func NewProd() *zap.Logger {
	config := zap.NewProductionConfig()
	l, err := config.Build(
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
	)
	if err != nil {
		panic(err)
	}
	return l
}
