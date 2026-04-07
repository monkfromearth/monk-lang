# Error Message Quality

Error messages are part of the language. A developer's first encounter
with a Monk error message shapes their opinion of the entire project.
Bad errors → frustrated users. Good errors → trust.

This rule applies to every error emitted by the type checker, parser,
and runtime. Treat every error string like public API.

---

## The Standard

Every error message must answer: **"What went wrong, where, and how do
I fix it?"**

### Required components

1. **Location** — file, line, column. Always 1-indexed.
   `foo.monk:5:3:`

2. **What** — what the compiler found vs what it expected. Concrete,
   not abstract.
   `expected int, got string` ✓
   `type mismatch` ✗

3. **Context** — the user's own code, not the compiler's internals.
   `cannot reassign 'x': declared int, value is string` ✓
   `AssignableTo check failed for MONK_STR → MONK_INT` ✗

4. **Actionable hint** (when non-obvious) — what to change.
   `use 'to_int(x)' to convert a string to int` ✓
   (no hint needed for obvious errors like wrong arity)

---

## Format

```
<file>:<line>:<col>: <what went wrong>
```

For multi-line context (e.g. function signature mismatch):
```
foo.monk:12:5: wrong argument types
  expected: (int, int) -> bool
  got:      (string, int)
```

Be consistent. If one type error says `expected X, got Y`, all type
errors say `expected X, got Y` — not sometimes `got Y, expected X`.

---

## Naming rules (from GCC guidelines, applied to Monk)

- **Source terms, not compiler terms.** Write `variable 'x'`, not
  `identifier node 'x'`. Write `function`, not `FuncExpr`. Write
  `array`, not `MONK_ARRAY`.
- **Quote user-supplied names** with single quotes: `variable 'foo'`,
  `field 'name'`, not `variable foo` or `variable "foo"`.
- **Describe the type as the user wrote it**, not the internal
  representation: `int[]`, not `Array(Int)`.
- **Say "cannot" not "illegal"** — illegal implies law-breaking.
  `cannot assign string to int variable` ✓
  `illegal assignment` ✗

---

## Error cascades

When one error causes downstream errors that are all symptoms of the
same root cause, emit only the root cause. A type error on line 5
should not produce 15 follow-on errors about the same variable.

The type checker should mark a variable as "error type" after its first
error and suppress further errors on that variable in the same scope.

---

## Panic is never acceptable on user input

If a `.monk` file causes the compiler to panic (Go panic, C crash,
segfault), that is a compiler bug — not a user error.

Rule: for any input a user could reasonably type, the compiler must
produce either a valid output or a clean error. It must never:
- Panic with a Go stack trace
- Crash the `cc` subprocess with a malformed C emit
- Hang

Write a test for any panic path you find:
`TestCheckDoesNotPanicOnX`, `TestBuildDoesNotCrashOnY`.

---

## Testing error messages

Error message tests belong in the type checker and parser test files.
The pattern:

```go
func TestTypeErrorMessage(t *testing.T) {
    _, err := types.Check(mustParse(`let x int = "hello"`))
    if err == nil {
        t.Fatal("expected error")
    }
    // Test the message content, not just that an error occurred
    if !strings.Contains(err.Error(), "expected int, got string") {
        t.Errorf("unhelpful error: %q", err.Error())
    }
}
```

Don't just assert `err != nil`. Assert that the message is the right
message — this prevents the error from silently changing to something
less useful.
