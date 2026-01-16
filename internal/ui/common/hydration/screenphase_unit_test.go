package hydration

import (
	"errors"
	"testing"
)

/*
Unit tests for ScreenPhase derivation.

Scope:
- Verify ScreenPhase derivation from Coordinator aggregate predicates
- Ensure phase depends ONLY on data completeness, not outcomes

Out of scope:
- Coordinator predicate correctness (covered elsewhere)
- Success / error semantics
- Mandatory gating
*/

func newScreenPhaseTestCoordinator() (*Coordinator, error) {
	c := NewCoordinator()
	err := c.Register("a", RarityCommon, true, false)
	err = c.Register("b", RarityCommon, true, false)
	return c, err
}

func TestDeriveScreenPhase_Init(t *testing.T) {
	c, err := newScreenPhaseTestCoordinator()
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}

	phase := DeriveScreenPhase(c)

	if phase != PhaseInit {
		t.Fatalf("expected PhaseInit, got %v", phase)
	}
}

func TestDeriveScreenPhase_InProgress_AllHydrating(t *testing.T) {
	c, err := newScreenPhaseTestCoordinator()
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}

	c.HydrateAllFields()

	phase := DeriveScreenPhase(c)

	if phase != PhaseHydrationsInProgress {
		t.Fatalf("expected PhaseHydrationsInProgress, got %v", phase)
	}
}

func TestDeriveScreenPhase_InProgress_PartialFinished(t *testing.T) {
	c, err := newScreenPhaseTestCoordinator()
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}

	c.HydrateAllFields()
	f, err := c.GetField("a")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetSuccess()

	phase := DeriveScreenPhase(c)

	if phase != PhaseHydrationsInProgress {
		t.Fatalf("expected PhaseHydrationsInProgress, got %v", phase)
	}
}

func TestDeriveScreenPhase_Finished_AllSuccess(t *testing.T) {
	c, err := newScreenPhaseTestCoordinator()
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}

	f, err := c.GetField("a")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetSuccess()
	f, err = c.GetField("b")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetSuccess()

	phase := DeriveScreenPhase(c)

	if phase != PhaseHydrationFinished {
		t.Fatalf("expected PhaseHydrationFinished, got %v", phase)
	}
}

func TestDeriveScreenPhase_Finished_MixedOutcomes(t *testing.T) {
	c, err := newScreenPhaseTestCoordinator()
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}

	f, err := c.GetField("a")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetSuccess()
	f, err = c.GetField("b")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetError(errors.New("err"))

	phase := DeriveScreenPhase(c)

	if phase != PhaseHydrationFinished {
		t.Fatalf("expected PhaseHydrationFinished, got %v", phase)
	}
}

func TestDeriveScreenPhase_Finished_AllErrors(t *testing.T) {
	c, err := newScreenPhaseTestCoordinator()
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}

	f, err := c.GetField("a")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetError(errors.New("a"))
	f, err = c.GetField("b")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetError(errors.New("b"))

	phase := DeriveScreenPhase(c)

	if phase != PhaseHydrationFinished {
		t.Fatalf("expected PhaseHydrationFinished, got %v", phase)
	}
}
