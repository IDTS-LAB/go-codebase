package utils

import (
	"context"
)

type contextKey string

const errorInfoKey contextKey = "error_info"

type ErrorInfo struct {
	Err   error
	Stack string
}

func SetErrorInfo(ctx context.Context, err error, stack string) context.Context {
	return context.WithValue(ctx, errorInfoKey, &ErrorInfo{Err: err, Stack: stack})
}

func GetErrorInfo(ctx context.Context) (*ErrorInfo, bool) {
	info, ok := ctx.Value(errorInfoKey).(*ErrorInfo)
	if !ok {
		return nil, false
	}
	return info, true
}
