# Personal Engineering Retrospective – Process Detail Screen

> This document is an internal retrospective written to consolidate learning, calibrate judgment, and guide future architectural decisions. It is not marketing material and is not written to impress — it is written to be *useful*.

---

## 1. What Problem I Actually Solved

At the start, the visible problem was:

- Build a TUI screen that shows process details
- Fetch multiple kinds of data asynchronously
- Handle partial failure gracefully

The *real* problem turned out to be:

- Modeling **streaming, partial, cancelable state** in a UI
- Without crashing, freezing, or lying to the user
- While preserving interactivity and navigation safety

This was not a UI styling task. It was a **state–lifecycle modeling problem**.

---

## 2. Key Engineering Challenges Encountered

### 2.1 Asynchronous Hydration Is Not a Phase

My initial mental model assumed:

- Init → Load → Ready

Reality forced a different model:

- Multiple independent data sources
- Each can succeed, fail, retry, or be canceled
- UI must remain usable throughout

**Insight:**
> Any UI that depends on multiple async sources is *streaming by default*, not phased.

This realization fundamentally changed the architecture.

---

### 2.2 Error Handling Is a First-Class State, Not an Exception

Early versions:
- Panicked on failure
- Or treated errors as terminal

Final approach:
- Errors are data
- Errors are categorized
- Errors can be dismissed without being resolved

**Insight:**
> In observability-style tools, errors are *expected*, not exceptional.

---

### 2.3 Cancellation Is a Lifecycle Boundary, Not a Boolean

I learned that:

- Canceling async work is insufficient
- You must also decide **what is allowed to mutate after cancellation**

This led to:
- Side-effect vs UI-state message separation
- Explicit decisions about what continues vs what stops

**Insight:**
> Cancellation defines *ownership of time*, not just execution.

---

## 3. What I Did Well (Objectively)

### 3.1 I Identified the Real Hard Problems

I did not get stuck polishing:
- styling
- key bindings
- layout details

Instead, effort converged on:
- state coherence
- retry semantics
- cancellation correctness

This is a strong signal of architectural maturity.

---

### 3.2 I Refactored Toward Truth, Not Abstraction

I avoided:
- premature interfaces
- dependency injection frameworks
- over-generalized state machines

Instead, I:
- made state explicit
- allowed complexity to surface
- only abstracted when pressure was real

**Result:**
The final model is verbose, but honest.

---

### 3.3 I Stopped at the Right Time

Crucially, I recognized when:

- further changes would be semantic trade-offs
- not correctness improvements

Freezing at this point avoids architectural churn.

---

## 4. Mistakes and Inefficiencies

### 4.1 Over-Reliance on Derived State in Early Iterations

Early designs:
- Cached screen state
- Mixed derived and source-of-truth data

This caused:
- illegal states
- race conditions
- hard-to-reason bugs

**Lesson:**
> Derived state should be computed, or stored. Never both.

---

### 4.2 Underestimating Viewport as a State Machine

I initially treated the viewport as a passive container.

In reality:
- It has internal state
- It reacts to content, size, and timing
- It can easily desynchronize UX

**Lesson:**
> Any scrollable container is a state machine.

---

### 4.3 Iteration Cost Was Higher Than Expected

Many changes required:
- touching multiple functions
- re-validating invariants
- mental re-simulation of flows

This signals that future work should:
- reduce responsibility density
- or introduce clearer coordination abstractions

---

## 5. Architectural Judgement Gained

### 5.1 When to Use Enums vs Flags

- Flags work for *local facts*
- Enums work for *global interpretation*

Trying to replace one with the other caused friction.

---

### 5.2 When NOT to Abstract

I deliberately avoided:
- a generic hydration framework
- a reusable state engine

Because:
- the shape of the problem was still evolving
- abstraction would have frozen wrong assumptions

---

### 5.3 UI Semantics Matter as Much as Correctness

Even when code was correct, UX semantics could be misleading:

- “Finished” did not mean immutable
- “Dismissed” did not mean resolved

This forced explicit product-level decisions.

---

## 6. What This Says About My Engineering Level

This work demonstrates:

- Strong state modeling instincts
- Comfort with async, partial, and failure-heavy systems
- Ability to reason about lifecycle boundaries
- Willingness to stop refactoring intentionally

It also shows:

- A tendency toward density
- A bias toward correctness over simplicity
- A need to consciously manage abstraction growth

Overall assessment:

> This is senior-level systems/UI architecture work, not feature implementation.

---

## 7. How I Will Apply This Going Forward

Concrete takeaways:

1. Default to streaming models in async UIs
2. Treat errors as data, not control flow
3. Design cancellation as a lifecycle transition
4. Freeze earlier once correctness is achieved
5. Write down intentional imperfections

---

## 8. Final Note to Future Me

If you are reading this months later and feel tempted to “clean this up”:

- Re-read the MVP Freeze Checklist
- Re-evaluate *why* each imperfection exists
- Only refactor if the usage context has changed

This screen is **done** for its purpose.
