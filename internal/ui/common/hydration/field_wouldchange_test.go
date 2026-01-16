package hydration

import (
	"errors"
	"testing"
)

var (
	errA = errors.New("A")
	errB = errors.New("B")
)

func TestWouldChange_StateTransitionProperties(t *testing.T) {
	states := []HydrationState{
		StateNotAsked,
		StateHydrating,
		StateSuccess,
		StateError,
	}

	errors := []error{
		nil,
		errA,
		errB,
	}

	for _, oldState := range states {
		for _, oldErr := range errors {
			f := &Field{
				state: oldState,
				err:   oldErr,
			}

			for _, newState := range states {
				for _, newErr := range errors {
					got := f.WouldChange(newState, newErr)
					want := expectedWouldChange(oldState, oldErr, newState, newErr)

					if got != want {
						t.Fatalf(
							"WouldChange mismatch: old=(%v,%v) new=(%v,%v): got %v, want %v",
							oldState, errLabel(oldErr),
							newState, errLabel(newErr),
							got, want,
						)
					}
				}
			}
		}
	}
}
