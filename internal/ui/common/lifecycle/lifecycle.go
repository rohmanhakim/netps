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
