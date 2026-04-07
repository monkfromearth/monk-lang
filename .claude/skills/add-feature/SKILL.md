---
name: add-feature
description: >
  Spec-first workflow for adding a new language feature to Monk. Walks
  through: spec design → negative/positive tests → implementation →
  codegen → error messages → doc sync. Prevents features from landing
  undocumented, untested, or misaligned with the design philosophy.
allowed-tools: Bash, Read, Grep, Glob, Edit, Write
---

# Add Feature

You are adding a new language feature to Monk. The spec is the contract.
The code is the implementation. In that order — always.

If you implement first and spec second, the spec becomes a description
of what you accidentally built. That's not a spec.

---

## Step 0 — Understand the design philosophy first

Before touching anything, read:
1. `spec/REFERENCE.md` — what Monk already IS
2. `spec/ARCHITECTURE_DECISIONS.md` — why the compiler is structured this way
3. The three design rules from `CLAUDE.md`:
   - **Explicit over implicit** — no hidden coercion, no hidden errors
   - **Graceful on reads, strict on operations** — OOB read = none, OOB write = error
   - **Values, not references** — assignment copies, always

If the proposed feature conflicts with any of these, surface the conflict
before proceeding. Don't paper over it. Change the spec or change the feature.

---

## Step 1 — Write the spec entry first

Open `spec/REFERENCE.md`. Add or update the relevant section.

Your spec entry must include:

### Syntax
What does it look like? Show the grammar if it's a new construct.
Show examples of valid syntax.

```monk
# What valid code looks like
let x = feature_example
```

### Semantics
What does it DO? Be precise. Cover:
- Normal case
- Edge cases (what happens at boundaries?)
- Error cases (what should produce an error, and what message?)
- Interaction with existing features (types, modules, closures)

### Design rationale (one paragraph)
Why does it work this way and not another? Connect it to the three
design rules. If a reader asks "why not X?" the spec should answer.

Do NOT move to Step 2 until the spec entry is complete and makes sense
standalone.

---

## Step 2 — Write failing tests before any code

Tests first. This is non-negotiable.

Write tests in the appropriate file:
- Parser behavior → `src/syntax/<feature>_test.go`
- Type checker behavior → `src/types/checker_test.go`
- Code generation → `src/codegen/codegen_test.go`
- CLI behavior → `src/main_test.go`

For each test, follow `test-organization.md`:

```go
// Positive: feature works as specified
func Test<Feature><Condition>(t *testing.T) {
    // uses a valid program that exercises the feature
    // asserts the correct output
}

// Negative: invalid usage produces the right error message
func Test<Feature><Condition>Error(t *testing.T) {
    // uses invalid input
    // asserts both that an error occurred AND that the message matches
    // (don't just assert err != nil)
}
```

Run them. They should fail. If they pass before you write any code,
either the feature is already implemented or the test is wrong.

```bash
cd src && go test ./... -run Test<Feature>
```

---

## Step 3 — Implement

Work through the pipeline in order. Each stage feeds the next.

### 3a. Scanner/Parser (if new syntax)
New keywords → add to `syntax/token.go` keyword table.
New grammar rule → add to `syntax/parse_stmt.go` or `syntax/parse_expr.go`.
New AST node → add to `syntax/ast_stmt.go` or `syntax/ast_expr.go`.
Update `src/syntax/INDEX.md`.

### 3b. Type checker (if new type rules)
Add check logic in `src/types/`.
Thread new type information into `types.Info` if codegen needs it.
Update `src/types/INDEX.md`.

### 3c. Code generator
Add emission logic in the appropriate `src/codegen/gen_*.go` file.
If the feature benefits from scalar unboxing, handle it in `unbox.go`.
Follow `codegen-c-safety.md` for every C emission.
Update `src/codegen/INDEX.md`.

### 3d. Runtime (if new built-in or runtime support needed)
Add to the appropriate `src/runtime/*.c` file (see `runtime/INDEX.md`).
Update `src/runtime/INDEX.md`.

Run tests after each stage:
```bash
cd src && go test ./...
```

Don't move to the next stage until the current stage's tests pass.

---

## Step 4 — Error messages

Every error case in the spec must have a corresponding error message
that follows `error-message-quality.md`.

Checklist per error case:
- [ ] Message includes file, line, column
- [ ] Message says what was found vs what was expected (in user terms)
- [ ] Internal type names (`MONK_STR`, `FuncExpr`) are absent
- [ ] User's identifiers are quoted with single quotes
- [ ] An actionable hint is included if the fix isn't obvious
- [ ] A test asserts the message content (not just that an error occurred)

---

## Step 5 — Add an example

Add a `.monk` file to `examples/` that demonstrates the feature.
The example should:
- Be runnable: `./monk run examples/new-feature.monk`
- Have a comment at the top explaining what it shows
- Demonstrate the most common use case, not the most exotic one

---

## Step 6 — Run the pre-completion battery

```bash
make build
cd src && go test ./...
# C runtime
RT_SRCS=$(ls src/runtime/*.c | xargs)
cc -std=c11 $RT_SRCS -lm -o /tmp/rt_test && /tmp/rt_test && rm /tmp/rt_test
# All examples
for f in examples/*.monk; do ./monk run "$f" > /tmp/out 2>&1 && echo "ok $f" || { echo "FAIL $f"; cat /tmp/out; }; done
```

All must pass before declaring done.

---

## Step 7 — Update the doc chain

Follow `spec-doc-sync.md`. At minimum:

- [ ] `PROGRESS.md` — what was added, test count delta, any design decisions made
- [ ] `ROADMAP.md` — check off the item if it was planned
- [ ] `WALKTHROUGH.md` — if the architecture changed (new pass, new pipeline stage)
- [ ] `knowledge/` — if there's a learnable concept here, run `/knowledge-update`

---

## Common traps

**Don't implement behavior the spec doesn't cover.** If you're writing
codegen for feature X and you notice "it would be easy to also support Y",
stop. Add Y to the spec first, or add it to a future TODO in ROADMAP.md.
Scope creep in compiler features compounds — Y interacts with Z which
breaks W.

**Don't let error cases be an afterthought.** Every invalid program
should produce a clean error. If you implement the happy path and defer
error messages to "later", they never get written and users get panics.

**Don't break existing tests.** Run `go test ./...` after every stage.
A regression in an existing test means your change has unintended scope.
Investigate before continuing.
