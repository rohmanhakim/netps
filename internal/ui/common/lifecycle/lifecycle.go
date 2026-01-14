package lifecycle

import "context"

type Mode int

const (
	ModeIdle Mode = iota
	ModeSendSignal
)

// helper to check if we should cancel *side effects*
func ShouldCancelSideEffects(ctx context.Context) bool {
	return ctx.Err() != nil
}

type ScreenState int

/*
 * Screen States
 *
 * During data hydration,the screen is streaming, not strictly phased.
 * User can still interact with the process (e.g sending signal)
 * The reason is so that a critical operation may be in an urgency
 * Errors should not prevent users to do critical operations
 * Screen states describe data completeness, not UI lock-in.
 */
const (
	StateInit ScreenState = iota
	StateHydrationsInProgress
	StateOneHydrationFinished
	StateHydrationsFinishedErrorsExist
	StateHydrationsFinishedErrorDismissed
	StateHydrationsFinishedAllOK
	StateRetryHydrations
)
