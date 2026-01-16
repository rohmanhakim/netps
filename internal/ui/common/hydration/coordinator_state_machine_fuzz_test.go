package hydration

import (
	"errors"
	"testing"
)

/*
Sequential state-machine fuzz test for Coordinator.

Scope:
- Randomized, long operation sequences
- Invariant preservation after every step
- Idempotence and reset safety

Out of scope:
- Formatting and data integrity (unit tests)
- Curated examples (state-machine tests)
- Concurrency (concurrent fuzz tests)
*/

func FuzzCoordinatorStateMachine(f *testing.F) {
	// Seed with short, meaningful sequences
	f.Add([]byte{0, 1, 2, 3})
	f.Add([]byte{4, 5, 6})
	f.Add([]byte{7, 8, 9, 10})
	f.Add([]byte{11, 12, 13, 14, 15})

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

		for step, op := range ops {
			err := applyFuzzOperation(c, fields, op)
			if err != nil {
				t.Fatalf("applyFuzzOperation error: %s", err)
			}
			assertCoordinatorFuzzInvariants(t, c, step, len(fields))
		}
	})
}

/* --- operations --- */

func applyFuzzOperation(c *Coordinator, fields []string, op byte) error {
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
			return err
		}
		f.SetNotAsked()
	case 6:
		// no-op (stability / idempotence check)
	}
	return nil
}

/* --- invariants --- */

func assertCoordinatorFuzzInvariants(
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
		t.Fatalf("step %d: LAW VIOLATION: AllSuccess ⇒ AllFinished", step)
	}

	// LAW 2: hydrating implies not finished (non-empty coordinator)
	if fieldCount > 0 {
		if allHydrating && allFinished {
			t.Fatalf("step %d: LAW VIOLATION: AllHydrating ⇒ ¬AllFinished", step)
		}
	}

	// LAW 3: errors exclude global success
	if errorsExist && allSuccess {
		t.Fatalf("step %d: LAW VIOLATION: ErrorsExist ⇒ ¬AllSuccess", step)
	}

	// LAW 4: finished without success implies errors
	if allFinished && !allSuccess && !errorsExist {
		t.Fatalf(
			"step %d: LAW VIOLATION: AllFinished ∧ ¬AllSuccess ⇒ ErrorsExist",
			step,
		)
	}
}
