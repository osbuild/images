package mockos

import (
	"context"
)

type ctxKey int

const (
	execFuncKey ctxKey = iota
	existsFuncKey
	grepFuncKey
	readFileFuncKey
)

type ExecFn func(name string, arg ...string) ([]byte, []byte, error)

func WithExecFunc(ctx context.Context, f ExecFn) context.Context {
	return context.WithValue(ctx, execFuncKey, f)
}

func ExecFunc(ctx context.Context) ExecFn {
	if f, ok := ctx.Value(execFuncKey).(ExecFn); ok {
		return f
	}

	return nil
}

type ExistsFn func(name string) bool

func WithExistsFunc(ctx context.Context, f ExistsFn) context.Context {
	return context.WithValue(ctx, existsFuncKey, f)
}

func ExistsFunc(ctx context.Context) ExistsFn {
	if f, ok := ctx.Value(existsFuncKey).(ExistsFn); ok {
		return f
	}

	return nil
}

type GrepFn func(pattern, filename string) (bool, error)

func WithGrepFunc(ctx context.Context, f GrepFn) context.Context {
	return context.WithValue(ctx, grepFuncKey, f)
}

func GrepFunc(ctx context.Context) GrepFn {
	if f, ok := ctx.Value(grepFuncKey).(GrepFn); ok {
		return f
	}

	return nil
}

type ReadFileFn func(filename string) ([]byte, error)

func WithReadFileFunc(ctx context.Context, f ReadFileFn) context.Context {
	return context.WithValue(ctx, readFileFuncKey, f)
}

func ReadFileFunc(ctx context.Context) ReadFileFn {
	if f, ok := ctx.Value(readFileFuncKey).(ReadFileFn); ok {
		return f
	}

	return nil
}
