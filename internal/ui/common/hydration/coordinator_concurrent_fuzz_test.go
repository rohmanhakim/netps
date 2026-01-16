package hydration

import (
	"errors"
	"sync"
	"testing"
)

/*
Concurrent state-machine fuzz test for Coordinator.

IMPORTANT:
- This test does NOT assert thread safety.
- The Coordinator is not synchronized.
- The purpose is to assert that, even under arbitrary interleavings,
  the exposed aggregate predicates never contradict each other.

Scope:
- Concurrent interleavings
- Synchronization-point invariant checks

Out of scope:
- Linearizability
- Race detection
- Ordering guarantees
*/

func FuzzCoordinatorConcurrentStateMachine(f *testing.F) {
	// Seed with short operation streams
	f.Add([]byte{0, 1, 2, 3, 4})
	f.Add([]byte{5, 6, 7, 8})
	f.Add([]byte{9, 10, 11, 12})

	f.Fuzz(func(t *testing.T, ops []byte) {
		c := NewCoordinator()

		fields := []string{"a", "b", "c"}
		for _, name := range fields {
			// optional, retryable fields
			err := c.Register(name, RarityCommon, true, false)

			if err != nil {
				t.Fatalf("registration error: %s", err)
			}
		}

		const workers = 4
		chunkSize := max(1, len(ops)/workers)

		for round := 0; round*chunkSize < len(ops); round++ {
			var wg sync.WaitGroup

			for w := 0; w < workers; w++ {
				start := (round*workers + w) * chunkSize
				end := min(start+chunkSize, len(ops))
				if start >= len(ops) {
					continue
				}

				wg.Add(1)
				go func(slice []byte) {
					defer wg.Done()
					for _, op := range slice {
						err := applyConcurrentOperation(c, fields, op)
						if err != nil {
							t.Fatalf("applyConcurrentOperation error: %s", err)
						}
					}
				}(ops[start:end])
			}

			wg.Wait()

			// Assert invariants at synchronization point
			assertCoordinatorConcurrentInvariants(
				t,
				c,
				round,
				len(fields),
			)
		}
	})
}

/* --- concurrent operations --- */

func applyConcurrentOperation(c *Coordinator, fields []string, op byte) error {
	field := fields[int(op)%len(fields)]

	switch op % 7 {
	case 0:
		c.HydrateField(field)
	case 1:
		c.HydrateAllFields()
	case 2:
		c.ResetAllFields()
	case 3:
		f, err := c.GetField(field)
		if err != nil {
			return err
		}
		f.SetSuccess()
	case 4:
		f, err := c.GetField(field)
		if err != nil {
			return err
		}
		f.SetError(errors.New("err"))
	case 5:
		f, err := c.GetField(field)
		if err != nil {
			f.SetNotAsked()
		}
	case 6:
		// no-op: encourages interleavings
	}
	return nil
}

/* --- invariants --- */

func assertCoordinatorConcurrentInvariants(
	t *testing.T,
	c *Coordinator,
	step int,
	fieldCount int,
) {
	t.Helper()

	allFinished := c.AllFinished()
	allSuccess := c.AllSuccess()
	allHydrating := c.AllHydrating()
	errorsExist := c.ErrorsExist()

	// LAW 1: global success implies completion
	if allSuccess && !allFinished {
		t.Fatalf(
			"step %d: LAW VIOLATION: AllSuccess ⇒ AllFinished",
			step,
		)
	}

	// LAW 2: hydrating implies not finished (non-empty coordinator)
	if fieldCount > 0 {
		if allHydrating && allFinished {
			t.Fatalf(
				"step %d: LAW VIOLATION: AllHydrating ⇒ ¬AllFinished",
				step,
			)
		}
	}

	// LAW 3: errors exclude global success
	if errorsExist && allSuccess {
		t.Fatalf(
			"step %d: LAW VIOLATION: ErrorsExist ⇒ ¬AllSuccess",
			step,
		)
	}

	// LAW 4: finished without success implies errors
	if allFinished && !allSuccess && !errorsExist {
		t.Fatalf(
			"step %d: LAW VIOLATION: AllFinished ∧ ¬AllSuccess ⇒ ErrorsExist",
			step,
		)
	}

	completed, total := c.GetHydrationProgress()
	if completed < 0 || completed > total {
		t.Fatalf(
			"step %d: invalid hydration progress (%d/%d)",
			step, completed, total,
		)
	}
}

/* --- helpers --- */

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
