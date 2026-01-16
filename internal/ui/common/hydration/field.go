package hydration

type HydrationState int
type ErrorRarity string

const (
	StateNotAsked HydrationState = iota
	StateHydrating
	StateSuccess
	StateError
)

const (
	RarityCommon ErrorRarity = "common"
	RarityRare   ErrorRarity = "rare"
)

type Field struct {
	err              error
	errorRarity      ErrorRarity
	errorRetryable   bool
	state            HydrationState
	successMandatory bool
}

func NewField() *Field {
	return &Field{
		err:            nil,
		errorRarity:    RarityCommon,
		errorRetryable: true,
		state:          StateNotAsked,
	}
}

/*
WouldChange:

  - reports whether transitioning to (newState, newErr)

  - represents a semantically observable change.

    This function is intentionally pure and side-effect free.
    It is safe to call repeatedly and is suitable for diffing,
    rendering decisions, or message suppression.
*/
func (f *Field) WouldChange(newState HydrationState, newError error) bool {
	// State changes always matter
	if f.state != newState {
		return true
	}

	// Same state: only error identity matters in StateError
	if f.state == StateError && newState == StateError {
		return f.err != newError
	}

	// Same non-error state: no change
	return false
}

func (f *Field) SetSuccess() {
	f.state = StateSuccess
	f.err = nil
}

func (f *Field) SetHydrating() {
	f.state = StateHydrating
	f.err = nil
}

func (f *Field) SetNotAsked() {
	f.state = StateNotAsked
	f.err = nil
}

func (f *Field) SetError(err error) {
	f.state = StateError
	f.err = err
}

func (f *Field) IsError() bool {
	return f.state == StateError && f.err != nil
}

func (f *Field) IsFinished() bool {
	return f.state == StateSuccess || f.state == StateError
}

func (f *Field) IsSuccess() bool {
	return f.state == StateSuccess && f.err == nil
}

func (f *Field) IsHydrating() bool {
	return f.state == StateHydrating && f.err == nil
}

func (f *Field) CanProcess() bool {
	return (f.successMandatory && f.IsSuccess()) || (!f.successMandatory && f.IsFinished())
}
