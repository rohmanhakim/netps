package hydration

import (
	"errors"
	"testing"
)

func FuzzFieldWouldChange(f *testing.F) {
	// Seed corpus with meaningful edge cases
	seeds := []struct {
		oldState int
		newState int
		oldErr   int
		newErr   int
	}{
		{int(StateNotAsked), int(StateNotAsked), 0, 0},
		{int(StateError), int(StateError), 1, 2},
		{int(StateHydrating), int(StateSuccess), 0, 1},
		{int(StateSuccess), int(StateError), 0, 1},
		{int(StateError), int(StateSuccess), 1, 0},
	}

	for _, s := range seeds {
		f.Add(s.oldState, s.newState, s.oldErr, s.newErr)
	}

	f.Fuzz(func(
		t *testing.T,
		oldStateInt int,
		newStateInt int,
		oldErrInt int,
		newErrInt int,
	) {
		// Normalize states into valid enum range
		oldState := HydrationState(abs(oldStateInt) % 4)
		newState := HydrationState(abs(newStateInt) % 4)

		// Deterministically map ints to error identities
		oldErr := mapErr(oldErrInt)
		newErr := mapErr(newErrInt)

		field := &Field{
			state: oldState,
			err:   oldErr,
		}

		got := field.WouldChange(newState, newErr)
		want := expectedWouldChange(oldState, oldErr, newState, newErr)

		if got != want {
			t.Fatalf(
				"WouldChange invariant violated: old=(%v,%v) new=(%v,%v): got=%v want=%v",
				oldState, errLabel(oldErr),
				newState, errLabel(newErr),
				got, want,
			)
		}
	})
}

func mapErr(v int) error {
	switch abs(v) % 3 {
	case 0:
		return nil
	case 1:
		return errors.New("A")
	default:
		return errors.New("B")
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
