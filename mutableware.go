// mutableware is a mutable middleware package.
// You can add, remove, and swap out middleware handlers.
package mutableware

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
)

// ErrHandle is returned when one or more handlers return an
// error in their Handle(...) calls.
var ErrHandle = errors.New("handleError")

// HandlerID identifies a handler. Use this to remove a handler
// from a container.
type HandlerID uint64

// HandlerContainer is an ordered collection of Handlers of the same type.
// When a Request is sent to a HandlerContainer, Handlers are invoked in
// in the reverse order that they were added (the oldest Handler is executed
// last).
//
// Handlers can be removed after being added. This is a distinguishing feature
// of this package versus traditional middleware packages.
type HandlerContainer[Request any, Response any] struct {
	// stack of Handlers. Oldest first.
	stack         []identifiedHandler[Request, Response]
	nextID        uint64
	cachedHandler CurriedHandlerFunc[Request, Response]
	mux           *sync.RWMutex
}

// NewHandlerContainer creates a new container for Handlers of the same type.
func NewHandlerContainer[Request any, Response any]() *HandlerContainer[Request, Response] {
	return &HandlerContainer[Request, Response]{
		stack:         []identifiedHandler[Request, Response]{},
		nextID:        10,
		cachedHandler: nilCurriedHandlerFunc[Request, Response],
		mux:           &sync.RWMutex{},
	}
}

// Add a new handler to the container. Newer handlers are invoked first.
// Retain the returned HandlerID if you need to Remove() this handler later.
func (hc *HandlerContainer[Request, Response]) AddAnonymousHandler(handlerFn HandlerFunc[Request, Response], options ...AddOption) HandlerID {
	return hc.Add(handlerFn.Handler(), options...)
}

// Add a new handler to the container. Newer handlers are invoked first.
// Retain the returned HandlerID if you need to Remove() this handler later.
func (hc *HandlerContainer[Request, Response]) Add(handler Handler[Request, Response], options ...AddOption) HandlerID {
	hc.mux.Lock()
	defer hc.mux.Unlock()
	defer hc.buildHandlers()

	id := HandlerID(hc.nextID)
	hc.nextID = hc.nextID + 1
	addOpts := buildAddOptions(options)

	idHandler := identifiedHandler[Request, Response]{
		Handler: handler,
		info: HandlerInfo{
			ID:   id,
			Name: addOpts.name,
		},
	}

	if addOpts.swapID != HandlerID(0) {
		idx := slices.IndexFunc(hc.stack, func(e identifiedHandler[Request, Response]) bool {
			return e.info.ID == addOpts.swapID
		})
		if idx >= 0 {
			hc.stack[idx] = idHandler
			return id
		}
	}

	if addOpts.last {
		slices.Insert(hc.stack, 0, idHandler)
	} else {
		hc.stack = append(hc.stack, idHandler)
	}

	return id
}

// Remove a handler that was previously added.
func (hc *HandlerContainer[Request, Response]) Remove(id HandlerID) {
	hc.mux.Lock()
	defer hc.mux.Unlock()
	defer hc.buildHandlers()

	hc.stack = slices.DeleteFunc(hc.stack, func(e identifiedHandler[Request, Response]) bool {
		return e.info.ID == id
	})
}

// Handle runs the Handle function of the contained handlers.
// Handlers that were added latest are executed first.
func (hc *HandlerContainer[Request, Response]) Handle(ctx context.Context, request Request) (Response, error) {
	hc.mux.RLock()
	defer hc.mux.RUnlock()

	return hc.cachedHandler(ctx, request)
}

func (hc *HandlerContainer[Request, Response]) buildHandlers() {
	// the last functions to be called will be NOPs.
	curriedHandler := nilCurriedHandlerFunc[Request, Response]

	for _, handler := range hc.stack {
		handler := handler
		prevHandler := curriedHandler
		curriedHandler = func(cx context.Context, msg Request) (Response, error) {
			handlerCtx := contextWithHandlerInfo(cx, handler.info)
			out, err := handler.Handle(handlerCtx, msg, prevHandler)
			if err != nil && !errors.Is(err, ErrHandle) {
				return out, fmt.Errorf("%w handler=%s %w", ErrHandle, handler.info, err)
			}
			return out, err
		}
	}
	hc.cachedHandler = curriedHandler
}

func nilCurriedHandlerFunc[Request any, Response any](ctx context.Context, request Request) (Response, error) {
	var zero Response
	return zero, nil
}

// Handler processes requests.
type Handler[Request any, Response any] interface {
	// Handle runs the handler for the request.
	Handle(ctx context.Context, request Request, next CurriedHandlerFunc[Request, Response]) (Response, error)
}

type HandlerFunc[Request any, Response any] func(ctx context.Context, request Request, next CurriedHandlerFunc[Request, Response]) (Response, error)

type CurriedHandlerFunc[Request any, Response any] func(ctx context.Context, request Request) (Response, error)

// identifiedHandler just attaches an ID to a handler so it can be deleted.
type identifiedHandler[Request any, Response any] struct {
	Handler[Request, Response]
	info HandlerInfo
}

// HandlerInfo contains metadata for a Handler.
type HandlerInfo struct {
	ID   HandlerID
	Name string
}

func (h HandlerInfo) String() string {
	if h.Name != "" {
		return fmt.Sprintf("%d(%s)", h.ID, h.Name)
	}
	return fmt.Sprintf("%d", h.ID)
}
