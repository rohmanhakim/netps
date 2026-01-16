package hydration

type ScreenPhase int

/*
ScreenPhase

	ScreenPhase represents the *data-completeness phase* of a hydration session,
	derived from the Coordinator’s aggregate predicates.

	IMPORTANT:
	- ScreenPhase is NOT a UI state machine.
	- ScreenPhase does NOT encode success, failure, or mandatory gating.
	- ScreenPhase describes *whether hydration data is complete*, nothing more.

	Design principles:
	- Hydration is streaming: partial results may arrive over time.
	- Users may continue interacting with the system while hydration is in progress.
	- Errors must NOT block critical or time-sensitive user actions.

	As a result:
	- Phases are intentionally coarse.
	- Outcomes (success / error / mandatory gating) are handled elsewhere
	  via derived predicates, not phase transitions.
*/
const (
	// PhaseInit indicates that no hydration has started yet.
	PhaseInit ScreenPhase = iota

	// PhaseHydrationsInProgress indicates that at least one hydration
	// has started and not all hydrations are finished yet.
	PhaseHydrationsInProgress

	// PhaseHydrationFinished indicates that all hydrations have finished,
	// regardless of success or error.
	PhaseHydrationFinished
)

/*
DeriveScreenPhase

	DeriveScreenPhase is a pure projection from Coordinator state to ScreenPhase.

	Characteristics:
	- Pure function (no mutation, no side effects)
	- Idempotent and deterministic
	- Safe to call repeatedly from multiple subsystems

	Responsibilities:
	- Centralize the definition of “screen phase”
	- Prevent phase logic from being duplicated or re-derived inconsistently

	Non-responsibilities:
	- Does NOT encode UI lock-in or interaction rules
	- Does NOT encode success / error semantics
	- Does NOT perform caching (callers may cache if needed)

	NOTE:
	Any logic that depends on *outcomes* (errors, mandatory success, etc.)
	must be derived from Coordinator predicates directly, not from ScreenPhase.
*/
func DeriveScreenPhase(c *Coordinator) ScreenPhase {
	switch {
	case c.AllFinished():
		return PhaseHydrationFinished
	case c.AnyHydrating():
		return PhaseHydrationsInProgress
	default:
		return PhaseInit
	}
}
