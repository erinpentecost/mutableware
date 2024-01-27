package mutableware

import "context"

// NewAnonymousHandler creates a handler from validation and handle functions.
// validatorFn and handlerFn can be nil if you don't want to implement them.
// This is just a helper function.
// You could just use any struct that implements the Handler interface instead.
func NewAnonymousHandler[Request any, Response any](validatorFn ValidatorFunc[Request], handlerFn HandlerFunc[Request, Response]) Handler[Request, Response] {
	h := &anonHandler[Request, Response]{validatorFn: validatorFn, handlerFn: handlerFn}
	if h.validatorFn == nil {
		h.validatorFn = func(ctx context.Context, request Request, next CurriedValidatorFunc[Request]) error {
			return next(ctx, request)
		}
	}
	if h.handlerFn == nil {
		h.handlerFn = func(ctx context.Context, request Request, next CurriedHandlerFunc[Request, Response]) (Response, error) {
			return next(ctx, request)
		}
	}
	return h
}

// anonHandler holds a handler defined by a pair of functions.
type anonHandler[Request any, Response any] struct {
	validatorFn ValidatorFunc[Request]
	handlerFn   HandlerFunc[Request, Response]
}

func (a *anonHandler[Request, Response]) Validate(ctx context.Context, request Request, next CurriedValidatorFunc[Request]) error {
	return a.validatorFn(ctx, request, next)
}

func (a *anonHandler[Request, Response]) Handle(ctx context.Context, request Request, next CurriedHandlerFunc[Request, Response]) (Response, error) {
	return a.handlerFn(ctx, request, next)
}
