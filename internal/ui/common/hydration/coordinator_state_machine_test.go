package hydration

import (
	"errors"
	"testing"
)

/*
State-machine tests for Coordinator.

Scope:
- Curated multi-step sequences
- Legal but tricky interleavings
- Invariant preservation after every step

Out of scope:
- Formatting / data integrity (unit tests)
- Exhaustive exploration (fuzz tests)
- Concurrency (concurrent fuzz tests)
*/

type operation func(*Coordinator) error

/* --- operations --- */

func opHydrate(name string) operation {
	return func(c *Coordinator) error {
		c.HydrateField(name)
		return nil
	}
}

func opHydrateAll() operation {
	return func(c *Coordinator) error {
		c.HydrateAllFields()
		return nil
	}
}

func opResetAll() operation {
	return func(c *Coordinator) error {
		c.ResetAllFields()
		return nil
	}
}

func opSuccess(name string) operation {
	return func(c *Coordinator) error {
		f, err := c.GetField(name)
		if err != nil {
			return err
		}
		f.SetSuccess()
		return nil
	}
}

func opError(name string) operation {
	return func(c *Coordinator) error {
		f, err := c.GetField(name)
		if err != nil {
			return err
		}
		f.SetError(errors.New("err"))
		return nil
	}
}

/* --- test --- */

func TestCoordinator_StateMachineSequences(t *testing.T) {
	fields := []string{"a", "b"}

	sequences := []struct {
		name string
		ops  []operation
	}{
		{
			name: "single field success",
			ops: []operation{
				opHydrate("a"),
				opSuccess("a"),
			},
		},
		{
			name: "all hydrate then all succeed",
			ops: []operation{
				opHydrateAll(),
				opSuccess("a"),
				opSuccess("b"),
			},
		},
		{
			name: "single field error",
			ops: []operation{
				opHydrate("a"),
				opError("a"),
			},
		},
		{
			name: "mixed error and success",
			ops: []operation{
				opHydrateAll(),
				opError("a"),
				opSuccess("b"),
			},
		},
		{
			name: "reset mid-flight",
			ops: []operation{
				opHydrateAll(),
				opResetAll(),
			},
		},
		{
			name: "error then reset",
			ops: []operation{
				opHydrate("a"),
				opError("a"),
				opResetAll(),
			},
		},
		{
			name: "interleaved hydration",
			ops: []operation{
				opHydrate("a"),
				opHydrate("b"),
				opSuccess("a"),
				opError("b"),
			},
		},
		{
			name: "hydrate after partial completion",
			ops: []operation{
				opHydrate("a"),
				opSuccess("a"),
				opHydrate("b"),
				opSuccess("b"),
			},
		},
		{
			name: "pathological: success without hydrate",
			ops: []operation{
				opSuccess("a"),
				opSuccess("b"),
			},
		},
		{
			name: "pathological: error then hydrate then success",
			ops: []operation{
				opError("a"),
				opHydrate("a"),
				opSuccess("a"),
			},
		},
	}

	for _, tt := range sequences {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCoordinator()
			for _, f := range fields {
				// optional, retryable fields
				err := c.Register(f, RarityCommon, true, false)

				if err != nil {
					t.Fatalf("regsitration error: %s", err)
				}
			}

			for step, op := range tt.ops {
				err := op(c)
				if err != nil {
					t.Fatalf("operation error: %s", err)
				}
				assertCoordinatorInvariants(t, c, step, len(fields))
			}
		})
	}
}

/* --- invariants --- */

func assertCoordinatorInvariants(
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
