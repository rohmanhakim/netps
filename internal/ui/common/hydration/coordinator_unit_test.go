package hydration

import (
	"errors"
	"reflect"
	"testing"
)

/*
Unit tests for Coordinator and Field.

Scope:
- Deterministic, single-step behavior
- API-level guarantees
- Formatting and data integrity
- Mandatory gating semantics

Out of scope:
- State-machine behavior
- Cross-step invariants
- Fuzzing / concurrency
*/

func newUnitCoordinator() (*Coordinator, error) {
	c := NewCoordinator()
	err := c.Register("mandatory", RarityRare, true, true)
	err = c.Register("optional", RarityCommon, false, false)
	return c, err
}

/* --- Registration & initialization --- */

func TestRegister_InitialFieldState(t *testing.T) {
	c, err := newUnitCoordinator()

	if err != nil {
		t.Fatalf("registration error: %s", err)
	}

	f, err := c.GetField("mandatory")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}

	if f == nil {
		t.Fatal("expected field to be registered")
	}

	if f.state != StateNotAsked {
		t.Fatalf("expected StateNotAsked, got %v", f.state)
	}
	if f.err != nil {
		t.Fatal("expected err to be nil")
	}
	if f.errorRarity != RarityRare {
		t.Fatalf("expected rarity %v, got %v", RarityRare, f.errorRarity)
	}
	if !f.errorRetryable {
		t.Fatal("expected mandatory field to be retryable")
	}
}

func TestRegister_FieldAleadyExist(t *testing.T) {
	c, err := newUnitCoordinator()
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}
	err = c.Register("mandatory", RarityRare, true, false)

	if !errors.Is(err, ErrFieldAlreadyExist) {
		t.Fatalf("expected ErrFieldAlreadyExist, got %v", err)
	}
}

func TestGetField_UnknownReturnsError(t *testing.T) {
	c, err := newUnitCoordinator()
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}

	f, err := c.GetField("does-not-exist")
	if !errors.Is(err, ErrFieldNotRegistered) { // ✅ Expect error
		t.Fatalf("expected ErrFieldNotRegistered, got %v", err)
	}
	if f != nil {
		t.Fatal("expected nil field on error")
	}
}

/* --- Field state transitions --- */

func TestHydrate_SetsExactState(t *testing.T) {
	c, err := newUnitCoordinator()
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}

	c.HydrateField("mandatory")
	f, err := c.GetField("mandatory")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}

	if f.state != StateHydrating {
		t.Fatalf("expected StateHydrating, got %v", f.state)
	}
	if f.err != nil {
		t.Fatal("expected err to be nil after hydrate")
	}
}

func TestSetSuccess_ClearsError(t *testing.T) {
	f := &Field{}
	f.SetError(errors.New("boom"))
	f.SetSuccess()

	if f.state != StateSuccess {
		t.Fatalf("expected StateSuccess, got %v", f.state)
	}
	if f.err != nil {
		t.Fatal("expected error to be cleared on success")
	}
}

func TestSetError_SetsExactState(t *testing.T) {
	f := &Field{}
	err := errors.New("fail")

	f.SetError(err)

	if f.state != StateError {
		t.Fatalf("expected StateError, got %v", f.state)
	}
	if f.err != err {
		t.Fatal("error value not preserved")
	}
}

/* --- Reset semantics --- */

func TestResetAllField_ResetsStateAndError(t *testing.T) {
	c, err := newUnitCoordinator()
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}

	f, err := c.GetField("mandatory")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetError(errors.New("x"))

	f, err = c.GetField("optional")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetSuccess()

	c.ResetAllFields()

	for name, f := range c.fieldMap {
		if f.state != StateNotAsked {
			t.Fatalf("[%s] expected StateNotAsked, got %v", name, f.state)
		}
		if f.err != nil {
			t.Fatalf("[%s] expected err nil, got %v", name, f.err)
		}
	}
}

/* --- Aggregate predicates (single-step) --- */

func TestAllFinished_Strict(t *testing.T) {
	c, err := newUnitCoordinator()
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}

	if c.AllFinished() {
		t.Fatal("expected AllFinished false initially")
	}

	f, err := c.GetField("mandatory")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetSuccess()

	if c.AllFinished() {
		t.Fatal("expected AllFinished false with partial completion")
	}

	f, err = c.GetField("optional")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetError(errors.New("x"))

	if !c.AllFinished() {
		t.Fatal("expected AllFinished true when all fields finished")
	}
}

func TestAllSuccess_Strict(t *testing.T) {
	c, err := newUnitCoordinator()
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}

	f, err := c.GetField("mandatory")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetSuccess()

	f, err = c.GetField("optional")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetSuccess()

	if !c.AllSuccess() {
		t.Fatal("expected AllSuccess true when all succeed")
	}

	f, err = c.GetField("optional")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetError(errors.New("x"))
	if c.AllSuccess() {
		t.Fatal("expected AllSuccess false with error")
	}
}

func TestAllHydrating_Strict(t *testing.T) {
	c, err := newUnitCoordinator()
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}

	c.HydrateAllFields()

	if !c.AllHydrating() {
		t.Fatal("expected AllHydrating true after HydrateAll")
	}

	f, err := c.GetField("mandatory")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetSuccess()
	if c.AllHydrating() {
		t.Fatal("expected AllHydrating false when one finishes")
	}
}

func TestErrorsExist_Strict(t *testing.T) {
	c, err := newUnitCoordinator()
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}

	f, err := c.GetField("mandatory")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetError(nil)

	if c.ErrorsExist() {
		t.Fatal("nil error must not count")
	}

	f, err = c.GetField("mandatory")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetError(errors.New("boom"))
	if !c.ErrorsExist() {
		t.Fatal("expected ErrorsExist true")
	}
}

/* --- Mandatory gating --- */

func TestMandatorySucceeded_Strict(t *testing.T) {
	c, err := newUnitCoordinator()
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}

	// not finished
	if c.MandatorySucceeded() {
		t.Fatal("mandatory must not succeed before completion")
	}

	// error
	f, err := c.GetField("mandatory")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetError(errors.New("fail"))
	if c.MandatorySucceeded() {
		t.Fatal("mandatory error must block success")
	}

	// success
	f, err = c.GetField("mandatory")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetSuccess()
	if !c.MandatorySucceeded() {
		t.Fatal("mandatory success should allow progression")
	}
}

/* --- Error reporting --- */

func TestGetFieldsForRetry_ReturnsAllErrorFields(t *testing.T) {
	c, err := newUnitCoordinator()
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}

	f, err := c.GetField("mandatory")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetError(errors.New("a"))

	f, err = c.GetField("optional")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetError(errors.New("b"))

	fields := c.GetFieldsForRetry()
	expected := []string{"mandatory", "optional"}

	if !sameElements(fields, expected) {
		t.Fatalf("expected %v, got %v", expected, fields)
	}
}

func TestGetErrors_MapIntegrity(t *testing.T) {
	c, err := newUnitCoordinator()
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}

	f, err := c.GetField("mandatory")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	err = errors.New("boom")
	f.SetError(err)

	errs := c.GetErrors()

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs["mandatory"] != err {
		t.Fatal("returned error does not match original")
	}
}

func TestGetErrorsAsString_ExactFormat(t *testing.T) {
	c, err := newUnitCoordinator()
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}

	f, err := c.GetField("mandatory")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetError(errors.New("failure"))

	errs := c.GetErrorsAsString()

	expected := "[rare][retryable][mandatory] failure"
	if len(errs) != 1 {
		t.Fatalf("expected 1 error string, got %d", len(errs))
	}
	if errs[0] != expected {
		t.Fatalf("expected %q, got %q", expected, errs[0])
	}
}

func TestGetErrorsAsString_NoErrors(t *testing.T) {
	c, err := newUnitCoordinator()
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}
	errs := c.GetErrorsAsString()
	if !reflect.DeepEqual(errs, []string{}) {
		t.Fatalf("expected empty slice, got %v", errs)
	}
}

func TestGetHydrationProgress(t *testing.T) {
	c := NewCoordinator()
	err := c.Register("a", RarityCommon, true, false)
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}
	err = c.Register("b", RarityCommon, true, false)
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}
	err = c.Register("c", RarityCommon, true, false)
	if err != nil {
		t.Fatalf("registration error: %s", err)
	}
	// initial state
	completed, total := c.GetHydrationProgress()
	if completed != 0 || total != 3 {
		t.Fatalf("expected (0,3), got (%d,%d)", completed, total)
	}

	// partial completion
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

	completed, total = c.GetHydrationProgress()
	if completed != 2 || total != 3 {
		t.Fatalf("expected (2,3), got (%d,%d)", completed, total)
	}

	// all finished
	f, err = c.GetField("c")
	if err != nil {
		t.Fatalf("get field error: %s", err)
	}
	f.SetSuccess()

	completed, total = c.GetHydrationProgress()
	if completed != 3 || total != 3 {
		t.Fatalf("expected (3,3), got (%d,%d)", completed, total)
	}
}

/* --- helpers --- */

func sameElements(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := map[string]int{}
	for _, v := range a {
		m[v]++
	}
	for _, v := range b {
		m[v]--
	}
	for _, c := range m {
		if c != 0 {
			return false
		}
	}
	return true
}
