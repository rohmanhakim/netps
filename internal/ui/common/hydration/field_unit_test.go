package hydration

import (
	"errors"
	"testing"
)

/*
Unit tests for Field.

Scope:
- Constructor invariants
- Exact state transitions (mutators)
- Predicate truth table

Out of scope:
- Semantic diff logic (WouldChange tests)
- Fuzzing (WouldChange fuzz tests)
*/

/* --- constructor invariants --- */

func TestNewFieldDefaults(t *testing.T) {
	f := NewField()

	if f.state != StateNotAsked {
		t.Fatalf("expected state %v, got %v", StateNotAsked, f.state)
	}
	if f.err != nil {
		t.Fatalf("expected err to be nil, got %v", f.err)
	}
	if f.errorRarity != RarityCommon {
		t.Fatalf("expected rarity %v, got %v", RarityCommon, f.errorRarity)
	}
	if !f.errorRetryable {
		t.Fatalf("expected errorRetryable to default to true")
	}
}

/* --- exact state transitions --- */

func TestFieldStateMutators(t *testing.T) {
	tests := []struct {
		name      string
		apply     func(*Field)
		wantState HydrationState
		wantErr   bool
	}{
		{
			name:      "SetNotAsked",
			apply:     func(f *Field) { f.SetNotAsked() },
			wantState: StateNotAsked,
			wantErr:   false,
		},
		{
			name:      "SetHydrating",
			apply:     func(f *Field) { f.SetHydrating() },
			wantState: StateHydrating,
			wantErr:   false,
		},
		{
			name:      "SetSuccess",
			apply:     func(f *Field) { f.SetSuccess() },
			wantState: StateSuccess,
			wantErr:   false,
		},
		{
			name:      "SetError",
			apply:     func(f *Field) { f.SetError(errors.New("err")) },
			wantState: StateError,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewField()
			tt.apply(f)

			if f.state != tt.wantState {
				t.Fatalf("expected state %v, got %v", tt.wantState, f.state)
			}
			if (f.err != nil) != tt.wantErr {
				t.Fatalf("expected err=%v, got %v", tt.wantErr, f.err)
			}
		})
	}
}

/* --- predicate truth table --- */

func TestFieldPredicates(t *testing.T) {
	err := errors.New("err")

	tests := []struct {
		name        string
		state       HydrationState
		err         error
		isError     bool
		isFinished  bool
		isSuccess   bool
		isHydrating bool
	}{
		{
			name:        "not asked",
			state:       StateNotAsked,
			err:         nil,
			isError:     false,
			isFinished:  false,
			isSuccess:   false,
			isHydrating: false,
		},
		{
			name:        "hydrating",
			state:       StateHydrating,
			err:         nil,
			isError:     false,
			isFinished:  false,
			isSuccess:   false,
			isHydrating: true,
		},
		{
			name:        "success",
			state:       StateSuccess,
			err:         nil,
			isError:     false,
			isFinished:  true,
			isSuccess:   true,
			isHydrating: false,
		},
		{
			name:        "error with err",
			state:       StateError,
			err:         err,
			isError:     true,
			isFinished:  true,
			isSuccess:   false,
			isHydrating: false,
		},
		{
			name:        "error with nil err",
			state:       StateError,
			err:         nil,
			isError:     false,
			isFinished:  true,
			isSuccess:   false,
			isHydrating: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Field{
				state: tt.state,
				err:   tt.err,
			}

			if f.IsError() != tt.isError {
				t.Fatalf("isError: expected %v, got %v", tt.isError, f.IsError())
			}
			if f.IsFinished() != tt.isFinished {
				t.Fatalf("isFinished: expected %v, got %v", tt.isFinished, f.IsFinished())
			}
			if f.IsSuccess() != tt.isSuccess {
				t.Fatalf("isSuccess: expected %v, got %v", tt.isSuccess, f.IsSuccess())
			}
			if f.IsHydrating() != tt.isHydrating {
				t.Fatalf("isHydrating: expected %v, got %v", tt.isHydrating, f.IsHydrating())
			}
		})
	}
}
