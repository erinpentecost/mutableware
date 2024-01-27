package mutableware

type builtAddOptions struct {
	name   string
	swapID HandlerID
	last   bool
}

// AddOption is an option for the Add(...) function.
type AddOption func(*builtAddOptions)

// AddOptionName attaches a name to the handler to aid in debugging.
// This will appear in wrapped errors and is available through the
// passed-in context.
func AddOptionName(name string) AddOption {
	return func(o *builtAddOptions) {
		o.name = name
	}
}

// AddOptionSwap removes the target handler and inserts this one in its place.
// If the target handler doesn't exist, normal handler insertion occurs.
func AddOptionSwap(id HandlerID) AddOption {
	return func(o *builtAddOptions) {
		o.swapID = id
	}
}

// AddOptionLast inserts the handler so it's executed last instead of first.
func AddOptionLast() AddOption {
	return func(o *builtAddOptions) {
		o.last = true
	}
}

func buildAddOptions(opts []AddOption) *builtAddOptions {
	built := &builtAddOptions{}
	for _, opt := range opts {
		opt(built)
	}
	return built
}
