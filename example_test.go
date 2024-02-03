package mutableware_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/erinpentecost/mutableware"
	"github.com/stretchr/testify/require"
)

// Build a new Handler that will tell us which sounds that animals make.
type Animal string
type Sound string

type AnimalSoundHandler struct {
	animal Animal
	sound  Sound
}

func (ash *AnimalSoundHandler) Handle(ctx context.Context, request Animal, next mutableware.CurriedHandlerFunc[Animal, Sound]) (Sound, error) {
	if request == ash.animal {
		return ash.sound, nil
	}
	// We don't handle this type of animal, so just pass along execution to the next handler in the chain.
	return next(ctx, request)
}

func TestExample(t *testing.T) {

	hc := mutableware.NewHandlerContainer[Animal, Sound]()
	// The first handler to be added will act as a sentinel to catch unknown Animal requests.
	hc.AddAnonymousHandler(
		func(ctx context.Context, request Animal, next mutableware.CurriedHandlerFunc[Animal, Sound]) (Sound, error) {
			return "", fmt.Errorf("unknown animal")
		},
	)
	// Now we'll deal with ducks.
	duckHandlerID := hc.Add(&AnimalSoundHandler{
		animal: Animal("duck"),
		sound:  Sound("quack"),
	})
	// And now cows.
	hc.Add(&AnimalSoundHandler{
		animal: Animal("cow"),
		sound:  Sound("moo"),
	})
	// This handler will make the sounds really loud.
	hc.AddAnonymousHandler(
		func(ctx context.Context, request Animal, next mutableware.CurriedHandlerFunc[Animal, Sound]) (Sound, error) {
			response, err := next(ctx, request)
			if err != nil {
				return response, err
			}
			return Sound(fmt.Sprintf("%s!", strings.ToUpper(string(response)))), nil
		},
	)

	// Now lets try sending in requests to the handler container.
	duckSound, err := hc.Handle(context.Background(), "duck")
	require.NoError(t, err)
	require.Equal(t, Sound("QUACK!"), duckSound)
	cowSound, err := hc.Handle(context.Background(), "cow")
	require.NoError(t, err)
	require.Equal(t, Sound("MOO!"), cowSound)
	dogSound, err := hc.Handle(context.Background(), "dog")
	require.Error(t, err)
	require.Zero(t, dogSound)

	// Let's make ducks go "bark" instead.
	hc.Add(&AnimalSoundHandler{
		animal: Animal("duck"),
		sound:  Sound("bark"),
	},
		mutableware.AddOptionSwap(duckHandlerID), // Swap out an existing handler for this new one.
	)

	// Give it another try.
	duckSound, err = hc.Handle(context.Background(), "duck")
	require.NoError(t, err)
	require.Equal(t, Sound("BARK!"), duckSound)
}
