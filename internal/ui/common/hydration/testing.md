# Testing Guide – `hydration` Package

This package uses a **strict, layered testing strategy** to validate correctness at multiple levels:

* local correctness (single objects)
* aggregate correctness (derived predicates)
* temporal correctness (state over time)
* robustness under randomness
* robustness under concurrent interleavings

The tests are designed to **encode the design intent of the system**, not merely to catch regressions.

If all tests pass, the coordinator’s semantics are internally consistent.

---

## Mental Model: What Is Being Tested?

### Field

A **`Field`** represents the lifecycle of a single hydrated value:

```
NotAsked → Hydrating → (Success | Error)
```

A field exposes:

* mutators (`SetHydrating`, `SetSuccess`, `SetError`, `SetNotAsked`)
* predicates (`isFinished`, `isSuccess`, `isError`, `isHydrating`)
* semantic diff logic (`WouldChange`)

---

### Coordinator

A **`Coordinator`** manages many fields and exposes **aggregate predicates**, such as:

* `AllFinished`
* `AllSuccess`
* `AllHydrating`
* `ErrorsExist`
* `MandatorySucceeded`

The coordinator **does not encode outcomes as states**.
Instead, it derives facts from the underlying fields.

---

## The Test Pyramid (from simplest to strongest)

Each layer answers a different question.
Higher layers assume lower layers are correct.

---

## 1. Field Unit Tests (Foundation)

**Question answered:**

> “Does a single `Field` obey its local contract?”

### Guarantees

* Constructor defaults are correct
* Mutators set *exact* state and error behavior
* Predicates return correct results for every valid state

### Files

```
field_unit_test.go
```

### Notes

* These tests are deterministic and table-driven
* They define the **ground truth** for field behavior
* If these fail, there is a **local bug**

---

## 2. Field Semantic Diff Tests (`WouldChange`)

**Question answered:**

> “Does the system correctly detect meaningful changes?”

### Guarantees

* Any state change is detected
* Error identity matters *only* in `StateError`
* No false positives when nothing semantically changes

### Files

```
field_wouldchange_test.go
field_wouldchange_test_helper.go
field_wouldchange_fuzz_test.go
```

### Why this is special

`WouldChange` is a **semantic contract**, not a simple helper.

It has:

* a **deterministic specification test**
* a **fuzz test** to protect against future refactors

These tests are authoritative.

---

## 3. Coordinator Unit Tests

**Question answered:**

> “Do coordinator methods behave correctly in isolation?”

### Guarantees

* Field registration is correct
* Reset semantics are correct
* Mandatory gating logic is correct
* Error reporting and formatting are correct
* Aggregate predicates behave correctly in single-step scenarios

### Files

```
coordinator_unit_test.go
```

### Notes

* These tests are deterministic
* They do **not** test sequences or timing
* They form the contract relied upon by higher layers

---

## 4. Coordinator State-Machine Tests (Curated Sequences)

**Question answered:**

> “Does the coordinator remain logically consistent across real usage flows?”

### What is tested

Predefined, meaningful sequences such as:

* hydrate → success
* hydrate all → partial failure
* reset mid-flight
* success before hydrate (allowed, but must remain consistent)

After **every step**, invariants are asserted.

### Invariants enforced

* `AllSuccess ⇒ AllFinished`
* `AllHydrating ⇒ ¬AllFinished` (non-empty coordinator)
* `ErrorsExist ⇒ ¬AllSuccess`
* `AllFinished ∧ ¬AllSuccess ⇒ ErrorsExist`

### Files

```
coordinator_state_machine_test.go
```

### Why this matters

Some bugs only appear **after multiple transitions**.
These tests catch those.

---

## 5. Coordinator State-Machine Fuzz Tests (Random Sequences)

**Question answered:**

> “Are the invariants stable under arbitrary sequences of operations?”

### What happens

* Random operation sequences are generated
* Operations are applied step-by-step
* Invariants are asserted after **every operation**

### Files

```
coordinator_state_machine_fuzz_test.go
```

### Why this is powerful

This test explores **far more states** than any hand-written test could.

If this fails, it almost always indicates a **real semantic violation**.

---

## 6. Concurrent State-Machine Fuzz Tests (Interleavings)

**Question answered:**

> “Do observable invariants hold under concurrent interleavings?”

### What this test does (and does not do)

✔ Asserts:

* Aggregate predicates never contradict each other
* Invariants hold at synchronization points

✘ Does **not** assert:

* Thread safety
* Linearizability
* Absence of data races

### Files

```
coordinator_concurrent_fuzz_test.go
```

### Important note

⚠️ The current coordinator implementation is **not concurrency-safe**.

This test exists to:

* expose race conditions with `-race`
* document concurrency assumptions
* protect invariants if synchronization is added later

---

## How to Run Tests

### Standard test run (fast)

```bash
go test ./...
```

---

### Sequential fuzzing

```bash
go test -fuzz=FuzzCoordinatorStateMachine -run=^$
```

Recommended:

* after refactors
* before releases

---

### Concurrent fuzzing with race detector

```bash
go test -race -fuzz=FuzzCoordinatorConcurrentStateMachine -run=^$
```

⚠️ Expected to report races unless synchronization is added.

---

## How to Add or Modify Tests Safely

### Adding new behavior

1. Start with **unit tests**
2. Decide if new invariants are introduced
3. Ensure state-machine tests still pass
4. Let fuzz tests explore edge cases

---

### Refactoring existing code

* Do **not** weaken invariants
* If a state-machine or fuzz test fails, assume the bug is real
* Update tests only if the **semantic contract has changed**

---

## Design Philosophy Encoded in Tests

* States are **phases**, not outcomes
* Outcomes are **derived facts**
* Invariants matter more than individual functions
* Tests are documentation

If the tests pass, the model is internally consistent.

---

## Final Takeaway

You are working in a codebase where:

* Tests encode **semantic laws**
* Correctness is validated across:

  * time
  * randomness
  * concurrency
* Failures are high-signal, not noise

Read the tests as you would read a specification.

If you understand this document, you understand the system.
