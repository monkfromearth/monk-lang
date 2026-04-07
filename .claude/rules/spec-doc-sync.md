# Spec and Doc Sync

`spec/REFERENCE.md` is the single source of truth for what Monk IS.
Every other document — PROGRESS.md, WALKTHROUGH.md, knowledge/ pages,
examples/ — is derived from it. When they disagree, the spec wins.

But the spec only wins if everything downstream gets updated.
This rule encodes the propagation chain.

---

## The Chain

```
spec/REFERENCE.md           ← update first (spec-first development)
    ↓
spec/QUICK_REFERENCE.md     ← update syntax tables, module section, builtin tables
spec/C_RUNTIME.md           ← update if runtime API, structs, or typed-array kinds changed
    ↓
src/ (implementation)       ← update second
    ↓
src/**/*_test.go            ← update/add tests in the same commit
    ↓
PROGRESS.md                 ← record what changed and why
    ↓
ROADMAP.md                  ← check off the checkbox if phase complete
    ↓
WALKTHROUGH.md              ← update the guided tour if the architecture changed
    ↓
knowledge/                  ← update the lesson if the feature changed
    ↓
examples/                   ← update or add an example if behavior changed
```

Not every change propagates to every level. Use judgment:
- **Language behavior change** (new syntax, changed semantics, new builtin)
  → everything
- **Compiler internals change** (new codegen optimization, new pass)
  → PROGRESS.md + WALKTHROUGH.md (if the architecture section covers it)
- **Bug fix** → PROGRESS.md only, unless the fix changes user-visible behavior
- **New phase complete** → everything, plus ROADMAP.md checkbox

---

## Mandatory: Update PROGRESS.md after every meaningful session

If it's not in PROGRESS.md, it didn't happen.

Write in past tense, concrete. Bad: "Improved the type checker."
Good: "Added cross-type equality checks — `5 == "5"` now errors at the
check phase, not silently at runtime."

Include: what changed, what bugs were fixed, test count delta.

---

## Mandatory: Spec before code

Before implementing any new feature or changing existing behavior:

1. Open `spec/REFERENCE.md`
2. Write the spec entry (syntax, semantics, examples)
3. Write a failing test that matches the spec
4. Implement until the test passes
5. Then update the rest of the chain

If the spec is silent on an edge case, resolve it using the three design
rules (explicit over implicit / graceful reads, strict ops / values not
references), write the resolution into the spec, then implement.

Never implement first and spec second — the spec becomes a post-hoc
rationalization instead of a contract.

---

## Detecting Drift

Signs that the chain has drifted:
- A knowledge/ lesson describes syntax that no longer works
- WALKTHROUGH.md references a file or function that was moved/renamed
- An `examples/` program fails with a type error or parse error
- PROGRESS.md says a feature was added but `spec/REFERENCE.md` has no entry
- ROADMAP.md still shows a phase as unchecked after the phase completed

When you find drift: fix it in the same session. Don't file a note.
Drift compounds — one stale page becomes three stale pages become a
codebase that nobody trusts.

---

## After a Rename or Restructure

File renames and package restructures (like the `src/cmd/monk/` → `src/`
flatten) touch many links at once. After any restructure:

1. `grep -r "src/cmd/monk" .` — find all stale references
2. Fix every one of them in the same commit
3. Run `go test ./...` to confirm nothing is still looking at the old path
4. Check WALKTHROUGH.md and knowledge/ for hardcoded paths
