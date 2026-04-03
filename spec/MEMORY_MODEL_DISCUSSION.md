# Memory Model Discussion — 2026-04-03

This document captures the full design discussion about Monk Lang's memory model.
Decisions marked ✅ are final. Items marked ❓ are still open.

---

## The Problem

No GC means every value needs a deterministic answer to: who owns it, and when is it freed?

Features in the current spec that require a memory management answer:
- Closures mutating captured variables (`create_counter`)
- Array functions returning new arrays (`append`, `map`, etc.) — who frees the old one?
- `guard`/`against` creating/binding values across scope boundaries
- Records passed around freely
- `none` as a return value mixed with real values

---

## Options Evaluated

### Option A: Ownership + Move (Rust-like, simplified)

Every value has exactly one owner. Assignment transfers ownership. Source becomes invalid.

```monk
let a = [1, 2, 3]
let b = a         // ownership MOVES to b. a is now invalid.
// show(a)        // Error: a was moved
let c = copy(b)   // explicit copy
```

Primitives (int, float, boolean, none) always copied (stack-allocated).

**Pros:** Zero runtime overhead. Compiler enforces everything.
**Cons:** Users must think about moves. "Why can't I use a after let b = a?" Rust's biggest learning curve complaint.

**Verdict:** ❌ Rejected. User explicitly said no Rust model, even partially.

### Option B: Reference Counting (Swift-like, ARC)

Every heap value has a hidden counter. Assign/pass = increment. Out of scope = decrement. Zero = freed immediately. Compiler inserts inc/dec automatically — user never sees it.

```
let a = [1, 2, 3]    →  refcount: 1
let b = a             →  refcount: 2 (a and b point to same array)
// b out of scope     →  refcount: 1
// a out of scope     →  refcount: 0 → FREED
```

**Primitives** (int, float, boolean, none) live on stack, always copied, no refcounting.
**Heap types** (string, array, record, closure) are refcounted.

**Pros:** Simple mental model. Values "just work." Deterministic (no GC pauses). Swift proves it scales.
**Cons:** Runtime overhead on every assign/pass. Cycles leak (two records pointing at each other).

**Verdict:** ❓ Proposed. User wants more discussion.

### Option C: Ownership + Borrow (full Rust model)

Like Option A but with explicit borrowing via `ref` for temporary access without moving.

**Verdict:** ❌ Rejected. Rust model excluded entirely.

---

## How Reference Counting Works — Illustrated

### Basic Assignment

```monk
let a = [10, 20, 30]
```
```
Memory:
┌─────────────────────┐
│ [10, 20, 30]        │
│ refcount: 1          │  ← one variable (a) points here
└─────────────────────┘
       ↑
       a
```

```monk
let b = a
```
```
Memory:
┌─────────────────────┐
│ [10, 20, 30]        │
│ refcount: 2          │  ← two variables (a and b) point here
└─────────────────────┘
       ↑       ↑
       a       b
```

No copy. Both point to same array. Counter went 1 → 2.

```monk
// b goes out of scope → refcount: 1
// a goes out of scope → refcount: 0 → FREED
```

### Primitives (No Refcounting)

```monk
let x = 42
let y = x      // y gets its own copy of 42
x = 100
show(y)        // still 42
```
```
Stack:
┌─────┐  ┌─────┐
│  42 │  │  42 │   ← two separate copies
└─────┘  └─────┘
   x        y
```

### Arrays with append

```monk
let a = [1, 2, 3]
let b = append(a, 4)    // NEW array [1, 2, 3, 4]
```
```
┌─────────────┐    ┌──────────────────┐
│ [1, 2, 3]   │    │ [1, 2, 3, 4]    │
│ refcount: 1  │    │ refcount: 1      │
└─────────────┘    └──────────────────┘
       ↑                   ↑
       a                   b
```

Two independent arrays. Each freed when its refcount hits 0.

### Closures

```monk
let create_counter = (start int) () -> int {
    let count = start
    return () int {
        count = count + 1
        return count
    }
}
let counter = create_counter(0)
show(counter())  // 1
show(counter())  // 2
```

When the closure is created, `count` is moved to the heap inside the closure object:

```
Heap:
┌─────────────────────┐
│ Closure object       │
│ - code: { count++ } │
│ - captured: count=0  │
│ refcount: 1          │
└─────────────────────┘
         ↑
      counter
```

`create_counter` returns. Stack frame gone. But `count` lives inside the closure on heap. Freed when `counter` goes out of scope (refcount 0).

### Two Closures Sharing a Variable

```monk
let x = [1, 2, 3]
let f = () int { return length(x) }
let g = () int { return length(x) }
```
```
┌─────────────┐
│ [1, 2, 3]   │
│ refcount: 3  │  ← x, f's capture, g's capture
└─────────────┘
  ↑    ↑    ↑
  x    f    g
```

All three point to same array. Freed when last reference drops.

### The Cycle Problem

```monk
let a = { partner: none }
let b = { partner: none }
a.partner = b    // a points to b
b.partner = a    // b points to a → CYCLE
```
```
┌──────────────┐     ┌──────────────┐
│ a             │────▶│ b             │
│ refcount: 2   │◀────│ refcount: 2   │
└──────────────┘     └──────────────┘
```

Both go out of scope → refcount 1 each. Never hits 0. Memory leak.

**Proposed solutions:**
1. Don't worry about it now. Cycles are rare. Add `weak` references later. (Swift's approach.)
2. Cycle detector (mini-GC). Contradicts "no GC."
3. Prevent cycles structurally. Too restrictive.

---

## What the Spec's Memory Section Becomes Under Refcounting

- `ref` keyword stays: means "pass by reference" — function gets access to same value, refcount inc/dec automatic.
- `const` values can't be mutated. `let` values can.
- No "mutability exclusivity" rule. Multiple references are fine.
- No "lifetime tracking." Refcounting handles it.
- No "dangling references." If you hold a ref, refcount > 0, value exists.

Safety story: **compiler prevents mutating `const` values. Refcounting prevents leaks and use-after-free.**

---

## Impact on Implementation Language

| Language | Fit | Notes |
|----------|-----|-------|
| Rust | Good | `Rc<T>` built in |
| Zig | Good | Implement refcount yourself. Full control. |
| C | Good | CPython does exactly this. Proven. |
| C++ | Good | `shared_ptr` built in. |
| TS/Bun | Bad | V8 has GC underneath. "No GC" is a lie. |

---

## User Decisions So Far

- ❌ No Rust model (ownership, moves, borrow checker) — even partially or fully
- ❌ No GC — strict requirement
- ❓ Reference counting — proposed, user wants more discussion
- ❓ Implementation language — TBD

---

## Decisions Made Since Initial Discussion

### Value Semantics (✅ Decided 2026-04-03, updated 2026-04-03)
- Assignment COPIES arrays and records. `let b = a` gives `b` an independent copy.
- Function arguments are copies. A function cannot modify the caller's data.
- Closures capture by COPY (like C++ `[x]` lambdas). No shared state. No exceptions to value semantics.
- Eager deep copy for v1. Copy-on-write may be added as an invisible optimization later.

### `ref` Deferred (✅ Decided 2026-04-03)
- `ref` keyword is reserved but not implemented in the current spec.
- Without `ref`, functions cannot modify external state. All mutation is return-value based.
- Plan: add `ref` parameters later (explicit on both sides — declaration and call site) if needed.
- Memory Management section removed from spec. Will be rewritten when model is finalized.

### Deep Const (✅ Decided 2026-04-03)
- `const` freezes the variable AND its contents. No element/field mutation.
- `let` = fully mutable (variable + contents).

### Records Are Shapes (✅ Decided 2026-04-03)
- Records have a fixed set of fields from creation. Cannot add new fields.
- This simplifies memory layout — no dynamic hash map growth.

### Impact of Value Semantics on Memory Model
With value semantics, the refcounting discussion changes:
- `let b = a` is a COPY, not a shared reference. No refcount needed for assignment.
- Function args are copies. No refcount needed for calls.
- Only closures share state — closure captures are the only place where multiple references exist.
- This dramatically reduces the surface area for memory management complexity.
- Copy-on-write optimization is still possible under the hood (the compiler can defer the copy until mutation, but semantics are always value-based).

## Open Questions

1. What is the runtime representation? Copy everything eagerly? Copy-on-write optimization?
2. How are closure-captured variables managed? (Heap-allocated, refcounted between closure and enclosing scope?)
3. Implementation language — affects what strategies are practical.
4. When/if `ref` parameters are added, what is the full design? Explicit on both sides (`ref` in declaration + `ref` at call site)?
5. Cycle prevention strategy for closure captures that reference each other.
