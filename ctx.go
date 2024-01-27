package mutableware

import "context"

type ctxKeyType int

const ctxKey = ctxKeyType(123)

func contextWithHandlerInfo(parent context.Context, info HandlerInfo) context.Context {
	if stack, ok := (parent.Value(ctxKey)).([]HandlerInfo); ok {
		return context.WithValue(parent, ctxKey, append(stack, info))
	} else {
		return context.WithValue(parent, ctxKey, []HandlerInfo{info})
	}
}

// GetHandlerInfoFromContext returns the current stack of handlers for a request.
// The latest handler to be invoked will be last in the slice.
func GetHandlerInfoFromContext(ctx context.Context) []HandlerInfo {
	if stack, ok := (ctx.Value(ctxKey)).([]HandlerInfo); ok {
		return stack
	} else {
		return []HandlerInfo{}
	}
}
