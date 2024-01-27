package mutableware_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/erinpentecost/mutableware"
	"github.com/stretchr/testify/require"
)

// TestBasicWrapping tests basic functionality.
func TestBasicWrapping(t *testing.T) {
	output := []int{}
	var ErrNoOdds = fmt.Errorf("odd numbers not allowed")

	hc := mutableware.NewHandlerContainer[int, string]()

	baseID := hc.Add(mutableware.NewAnonymousHandler(
		func(ctx context.Context, message int, next mutableware.CurriedValidatorFunc[int]) error {
			return next(ctx, message)
		},
		func(ctx context.Context, message int, next mutableware.CurriedHandlerFunc[int, string]) (string, error) {
			output = append(output, message)
			// don't continue execution to next in this case
			return "writer", nil
		},
	))

	doublerID := hc.Add(mutableware.NewAnonymousHandler(
		func(ctx context.Context, message int, next mutableware.CurriedValidatorFunc[int]) error {
			return next(ctx, message)
		},
		func(ctx context.Context, message int, next mutableware.CurriedHandlerFunc[int, string]) (string, error) {
			out, err := next(ctx, 2*message)
			if err != nil {
				return out, err
			}
			return fmt.Sprintf("doubler %s", out), nil
		},
	))

	noOddsID := hc.Add(mutableware.NewAnonymousHandler(
		func(ctx context.Context, message int, next mutableware.CurriedValidatorFunc[int]) error {
			if message%2 == 1 {
				return errors.Join(ErrNoOdds, next(ctx, message))
			}
			return next(ctx, message)
		},
		func(ctx context.Context, message int, next mutableware.CurriedHandlerFunc[int, string]) (string, error) {
			return next(ctx, message)
		},
	))

	require.GreaterOrEqual(t, doublerID, baseID)

	// test validation fail path
	err := hc.Validate(context.Background(), 5)
	require.ErrorIs(t, err, ErrNoOdds)
	require.ErrorIs(t, err, mutableware.ErrValidate)
	require.Equal(t, "validateError handler=12 odd numbers not allowed", err.Error())

	// test that we doubled the message
	res, err := hc.Handle(context.Background(), 4)
	require.Equal(t, "doubler writer", res)
	require.NoError(t, err)
	require.Equal(t, []int{4 * 2}, output)

	// test removal of handlers
	hc.Remove(noOddsID)
	hc.Remove(doublerID)
	res, err = hc.Handle(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, "writer", res)
	require.Equal(t, []int{4 * 2, 1}, output)

	// test empty
	hc.Remove(baseID)
	res, err = hc.Handle(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, "", res)
	require.NoError(t, err)
	require.Equal(t, []int{4 * 2, 1}, output)
}

// TestPassByReference confirms that handlers can modify requests.
func TestPassByReference(t *testing.T) {
	output := []*int32{}

	hc := mutableware.NewHandlerContainer[*int32, any]()

	baseID := hc.Add(mutableware.NewAnonymousHandler(
		func(ctx context.Context, message *int32, next mutableware.CurriedValidatorFunc[*int32]) error {
			return next(ctx, message)
		},
		func(ctx context.Context, message *int32, next mutableware.CurriedHandlerFunc[*int32, any]) (any, error) {
			output = append(output, message)
			return next(ctx, message)
		},
	))

	doublerID := hc.Add(mutableware.NewAnonymousHandler(
		func(ctx context.Context, message *int32, next mutableware.CurriedValidatorFunc[*int32]) error {
			return next(ctx, message)
		},
		func(ctx context.Context, message *int32, next mutableware.CurriedHandlerFunc[*int32, any]) (any, error) {
			atomic.SwapInt32(message, 2*(*message))
			return next(ctx, message)
		},
	))

	require.GreaterOrEqual(t, doublerID, baseID)

	input := int32(10)
	inputAddr := &input
	_, err := hc.Handle(context.Background(), inputAddr)
	require.NoError(t, err)
	require.Equal(t, int32(20), input)
	require.NotEmpty(t, output)
	require.True(t, inputAddr == output[0])
	require.True(t, input == *output[0])

}
func TestEmptyContainer(t *testing.T) {
	hc := mutableware.NewHandlerContainer[string, any]()

	resp, err := hc.Handle(context.Background(), "don't panic")
	require.NoError(t, err)
	require.Zero(t, resp)
}

func TestNilAnonFunc(t *testing.T) {
	hc := mutableware.NewHandlerContainer[string, any]()
	hc.Add(mutableware.NewAnonymousHandler[string, any](nil, nil))
	resp, err := hc.Handle(context.Background(), "don't panic")
	require.NoError(t, err)
	require.Zero(t, resp)
}

func TestAddLast(t *testing.T) {
	hc := mutableware.NewHandlerContainer[string, any]()
	hc.Add(mutableware.NewAnonymousHandler[string, any](nil,
		func(ctx context.Context, request string, next mutableware.CurriedHandlerFunc[string, any]) (any, error) {
			return "a", nil
		}), mutableware.AddOptionName("a"))
	hc.Add(mutableware.NewAnonymousHandler[string, any](nil,
		func(ctx context.Context, request string, next mutableware.CurriedHandlerFunc[string, any]) (any, error) {
			return "b", nil
		}), mutableware.AddOptionLast(), mutableware.AddOptionName("b"))
	resp, err := hc.Handle(context.Background(), "")
	require.NoError(t, err)
	require.Equal(t, "a", resp)
}

func TestHandleErr(t *testing.T) {
	expectedErr := fmt.Errorf("an_error")
	hc := mutableware.NewHandlerContainer[string, any]()
	hc.Add(mutableware.NewAnonymousHandler[string, any](nil, nil))
	errHandlerID := hc.Add(mutableware.NewAnonymousHandler[string, any](nil,
		func(ctx context.Context, request string, next mutableware.CurriedHandlerFunc[string, any]) (any, error) {
			return nil, expectedErr
		}))
	hc.Add(mutableware.NewAnonymousHandler[string, any](nil, nil))
	hc.Add(mutableware.NewAnonymousHandler[string, any](nil, nil))

	resp, err := hc.Handle(context.Background(), "should throw an error")
	require.ErrorIs(t, err, expectedErr)
	require.ErrorIs(t, err, mutableware.ErrHandle)
	require.Equal(t, mutableware.HandlerID(11), errHandlerID)
	require.Equal(t, "handleError handler=11 an_error", err.Error())
	require.Zero(t, resp)

	// swap handler with one that has a name
	newID := hc.Add(mutableware.NewAnonymousHandler[string, any](nil,
		func(ctx context.Context, request string, next mutableware.CurriedHandlerFunc[string, any]) (any, error) {
			return nil, expectedErr
		},
	), mutableware.AddOptionSwap(errHandlerID), mutableware.AddOptionName("new"))
	resp, err = hc.Handle(context.Background(), "should throw an error")
	require.ErrorIs(t, err, expectedErr)
	require.ErrorIs(t, err, mutableware.ErrHandle)
	require.Equal(t, mutableware.HandlerID(14), newID)
	require.Equal(t, "handleError handler=14(new) an_error", err.Error())
	require.Zero(t, resp)
}

func TestContext(t *testing.T) {
	hc := mutableware.NewHandlerContainer[string, any]()
	var firstValidationCtx context.Context
	var secondValidationCtx context.Context
	var firstHandleCtx context.Context
	var secondHandleCtx context.Context
	hc.Add(mutableware.NewAnonymousHandler[string, any](nil, nil))
	firstHandlerID := hc.Add(mutableware.NewAnonymousHandler[string, any](
		func(ctx context.Context, request string, next mutableware.CurriedValidatorFunc[string]) error {
			firstValidationCtx = ctx
			return next(ctx, request)
		},
		func(ctx context.Context, request string, next mutableware.CurriedHandlerFunc[string, any]) (any, error) {
			firstHandleCtx = ctx
			return next(ctx, request)
		}), mutableware.AddOptionName("first"))
	secondHandlerID := hc.Add(mutableware.NewAnonymousHandler[string, any](
		func(ctx context.Context, request string, next mutableware.CurriedValidatorFunc[string]) error {
			secondValidationCtx = ctx
			return next(ctx, request)
		},
		func(ctx context.Context, request string, next mutableware.CurriedHandlerFunc[string, any]) (any, error) {
			secondHandleCtx = ctx
			return next(ctx, request)
		}), mutableware.AddOptionName("second"))

	_ = hc.Validate(context.Background(), "ok")

	expectedFirstStack := []mutableware.HandlerInfo{
		{
			ID:   secondHandlerID,
			Name: "second",
		},
		{
			ID:   firstHandlerID,
			Name: "first",
		},
	}
	expectedSecondStack := []mutableware.HandlerInfo{
		{
			ID:   secondHandlerID,
			Name: "second",
		},
	}
	require.Equal(t, expectedFirstStack, mutableware.GetHandlerInfoFromContext(firstValidationCtx))
	require.Equal(t, expectedSecondStack, mutableware.GetHandlerInfoFromContext(secondValidationCtx))

	_, _ = hc.Handle(context.Background(), "ok")

	require.Equal(t, expectedFirstStack, mutableware.GetHandlerInfoFromContext(firstHandleCtx))
	require.Equal(t, expectedSecondStack, mutableware.GetHandlerInfoFromContext(secondHandleCtx))

	require.Equal(t, []mutableware.HandlerInfo{}, mutableware.GetHandlerInfoFromContext(context.Background()))
}
