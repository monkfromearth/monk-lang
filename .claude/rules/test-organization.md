# Test Organization

Tests are the executable spec. A test suite that's organized by
implementation detail becomes useless when you refactor. A test suite
organized by language feature tells you exactly what's broken and why.

---

## Name by spec feature, not implementation

Bad: `TestParser_IfStatement`
Good: `TestIfRequiresBoolCondition`, `TestIfElseChain`, `TestIfWithoutElse`

The test name should answer: "what language behavior does this verify?"
If the test is renamed after a refactor, it wasn't testing behavior.

---

## Positive + Negative per feature

Every feature needs at least two tests:

| Kind | Tests that... |
|------|--------------|
| **Positive** | valid input produces the correct output |
| **Negative** | invalid input produces the correct *error* |

A feature with only positive tests will silently accept garbage input
after a regression. A feature with only negative tests doesn't prove
the happy path works.

```go
// Positive: valid type annotation accepted
func TestVarDeclIntLiteral(t *testing.T) { ... }

// Negative: wrong type rejected with right message
func TestVarDeclTypeMismatchError(t *testing.T) { ... }
```

---

## Test names are the regression log

When a test fails in CI, the name is all you see at a glance. Make it
self-documenting. If the test is `TestFoo` and it fails, you know
nothing. If it's `TestGuardMissingAgainstClauseErrors`, you know exactly
what broke.

Pattern: `Test<Feature><Condition><Result>`

Examples:
- `TestForLoopBreakExitsEarly`
- `TestFuncMissingReturnErrors`
- `TestArrayLiteralMixedTypesRejected`
- `TestRecordMissingFieldErrors`
- `TestConstReassignmentRejected`

---

## Golden file tests for error messages

For complex error messages (multi-line, with context), use golden files
instead of string assertions. This prevents silent message degradation.

Pattern:
1. Store the expected error in `testdata/<feature>_error.txt`
2. On first run with `-update` flag, write the actual output to the file
3. On subsequent runs, diff actual vs file

```go
func TestTypeMismatchMessage(t *testing.T) {
    got := runCheckAndGetError(`let x int = "hello"`)
    golden := "testdata/type_mismatch_int_string.txt"
    if *update {
        os.WriteFile(golden, []byte(got), 0644)
        return
    }
    want, _ := os.ReadFile(golden)
    if got != string(want) {
        t.Errorf("error message changed:\ngot:  %q\nwant: %q", got, string(want))
    }
}
```

This catches message regressions automatically — if you refactor the
type checker and accidentally change "expected int, got string" to
"type error", the golden file diff catches it.

---

## Conformance matrix

For each language feature in `spec/REFERENCE.md`, there should be a
corresponding test. Track coverage in a comment at the top of the
relevant test file:

```go
// Type checker tests — spec/REFERENCE.md §Types
//
// Covered:
//   [x] First-assignment type inference
//   [x] Reassignment consistency
//   [x] Array element type enforcement
//   [x] Record shape checking
//   [x] Cross-type equality (5 == "5" → error)
//   [ ] Optional type propagation (T? → T? operations)
//   [ ] Function type compatibility
```

The `[ ]` items are the gaps. They surface naturally in code review
and don't require a separate tracking system.

---

## Integration vs unit

| Use integration tests for... | Use unit tests for... |
|-----------------------------|-----------------------|
| Compile + run → check stdout | Parser returning correct AST node |
| Type checker rejecting a file | Scanner tokenizing a specific input |
| CLI flag behavior | `cString()` escaping specific chars |

Don't unit test the generated C structure — it's an implementation
detail. Integration test the output behavior. If the generated C changes
shape but the binary output is the same, no test should fail.

---

## Bench/expected.txt is also a test

The `bench/benchmarks/<name>/expected.txt` pattern is already in use.
This is a form of golden file testing. Every new benchmark must have an
`expected.txt`. Never add a benchmark without it.
