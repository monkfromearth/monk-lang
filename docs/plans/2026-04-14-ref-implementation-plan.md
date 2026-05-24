# Ref Implementation Plan

## Goal

Land the first real `ref` feature in Monk without committing to full first-class references yet.

This first slice should support:

- `ref` in function parameter types
- `ref x` at the call site
- mutation through a referenced caller-owned location

This first slice should not yet support:

- storing references in variables
- returning references
- record fields or array elements as `ref` arguments unless we intentionally add that later
- any FFI syntax

The target is a small, explicit, testable feature that matches the updated language direction: values by default, references by intent.

## User-Facing Shape

The current intended source-level shape is:

```monk
let increment = (n ref int) none {
  n = n + 1
}

let total = 0
increment(ref total)
show(to_string(total))
```

Semantics for v1:

- `ref` modifies how a parameter is passed
- `ref` must appear in both the parameter declaration and the call argument
- only mutable named `let` bindings may be passed as `ref`
- `const` values, literals, temporaries, and arbitrary expressions are rejected

This gives Monk explicit pointer/reference behavior without opening the door to a large aliasing surface all at once.

## Scope Decisions

### In scope

- Scanner support for the `ref` keyword
- AST support for `ref` in type expressions and `ref expr` at call sites
- Type-system support for by-reference parameters
- Call-site validation for mutable lvalues
- Codegen support for passing addresses and dereferencing inside function bodies
- Tests for parser, checker, and generated code behavior

### Deliberately out of scope

- First-class reference values
- `let r = ref x`
- reference-typed record fields
- reference returns
- closure-captured references
- FFI design

Keeping v1 this narrow will let us validate the model before broadening it.

## Implementation Strategy

### 1. Parser and AST

Files:

- `src/syntax/token.go`
- `src/syntax/ast.go`
- `src/syntax/ast_expr.go`
- `src/syntax/parse_type.go`
- `src/syntax/parse_expr.go`
- `src/syntax/parser_test.go`
- `src/syntax/scanner_test.go`

Changes:

- Add `Ref` to the token set and keyword map.
- Extend `TypeExpr` with `IsRef bool`.
- Add a dedicated `RefExpr` AST node.
- Parse `ref T` in type positions.
- Parse `ref expr` in expression position.

Important parser rule:

- `ref expr` should parse as a normal expression node, but later phases should restrict where it is legal.
- For v1, the type checker should enforce that `ref expr` only appears as an argument to a `ref` parameter.

Recommended tests:

- parse `let f = (x ref int) none { }`
- parse `swap(ref a, ref b)`
- reject malformed types like `ref`
- reject malformed expressions like bare `ref` with no operand

### 2. Type Representation and Checking

Files:

- `src/types/types.go`
- `src/types/stmts.go`
- `src/types/exprs.go`
- `src/types/checker.go`
- `src/types/checker_test.go`

Recommended representation:

- add `IsRef bool` to `types.Type`
- do not create a separate `KindRef`

Reason:

- `ref` is a modifier on an underlying type, not a whole new runtime kind
- function parameter slots are the main place that need the distinction right now

Checker rules:

- `ref T` parameters accept only `ref expr` arguments
- plain arguments cannot satisfy `ref T`
- `ref expr` must point to an assignable mutable location
- for v1, that means a named `let` binding only
- `const`, literals, calls, binary expressions, and grouped temporaries are errors

Recommended error cases:

- missing `ref` at call site
- `ref` passed to a non-`ref` parameter
- `ref` on a `const`
- `ref 42`
- `ref (a + b)`

One practical simplification:

- keep `AssignableTo` mostly unchanged for value compatibility
- add targeted logic in call checking for `ref` parameters instead of trying to model everything through generic assignability

### 3. Code Generation

Files:

- `src/codegen/gen_expr.go`
- `src/codegen/gen_func.go`
- `src/codegen/unbox.go`
- `src/codegen/gen_stmt.go`
- `src/codegen/codegen_test.go`

Lowering model:

- a `ref` parameter lowers to a C pointer parameter
- a call-site `ref x` lowers to `&x`
- uses of that parameter in the callee lower to `(*ptr)` reads and writes

Examples:

Monk:

```monk
let increment = (n ref int) none {
  n = n + 1
}
```

Conceptual C shape:

```c
static void _monk_func_1(int64_t *mk_n) {
    *mk_n = *mk_n + 1;
}
```

Monk call:

```monk
increment(ref total)
```

Conceptual C call:

```c
_monk_func_1(&mk_total);
```

Important generator constraint:

- the first implementation should only allow `ref` params when the lowered storage is a stable addressable slot

That suggests a safe first implementation path:

- support `ref` first for scalar locals whose storage is already direct and addressable
- keep boxed and more complex storage cases conservative until the basic path works

If needed, we can broaden that in a second pass once the semantics and tests are solid.

### 4. Testing Ladder

1. Scanner and parser
2. Type checker
3. Codegen compilation
4. End-to-end runtime behavior

Recommended test set:

- incrementing an `int`
- swapping two `int`s
- rejecting `increment(total)` when param expects `ref int`
- rejecting `increment(ref total)` when param expects plain `int`
- rejecting `increment(ref answer)` when `answer` is `const`
- rejecting `increment(ref 42)`

Stretch tests after the core passes:

- `ref boolean`
- `ref float`
- nested function calls that still compile cleanly
- interaction with unboxed scalar functions

## Risks and Order of Work

### Biggest technical risk

The current compiler has unboxed scalar fast paths and boxed fallback paths. `ref` will cross that boundary. We should not try to make `ref` work for every storage shape on day one.

### Recommended order

1. Parser and AST
2. Type representation
3. Type checking for legality
4. Minimal scalar codegen
5. End-to-end tests
6. Broaden support only after the scalar path is stable

## Definition of Done for v1

`ref` v1 is done when:

- the parser accepts `ref` in parameter and argument position
- the checker enforces explicit mutable-lvalue passing rules
- scalar `ref` parameters compile and mutate caller state correctly
- incorrect uses fail with clear diagnostics
- docs no longer contradict the existence of `ref`

## Follow-Up After v1

After the first implementation lands, the next design questions should be:

- should array elements and record fields be passable by `ref`
- should `ref` be first-class outside parameter passing
- how should closure-captured references behave
- how will `ref` align with the eventual native/FFI story

Those should come after the core parameter-passing model is proven in code.
