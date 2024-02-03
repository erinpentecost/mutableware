package mutableware

import "context"

func (hf HandlerFunc[Request, Response]) Handler() Handler[Request, Response] {
	if hf != nil {
		return &anonHandler[Request, Response]{handlerFn: hf}
	}
	return &anonHandler[Request, Response]{handlerFn: nilHandlerFunc[Request, Response]}
}

func nilHandlerFunc[Request any, Response any](ctx context.Context, request Request, next CurriedHandlerFunc[Request, Response]) (Response, error) {
	return next(ctx, request)
}

// anonHandler holds a handler defined by a pair of functions.
type anonHandler[Request any, Response any] struct {
	handlerFn HandlerFunc[Request, Response]
}

func (a *anonHandler[Request, Response]) Handle(ctx context.Context, request Request, next CurriedHandlerFunc[Request, Response]) (Response, error) {
	return a.handlerFn(ctx, request, next)
}
