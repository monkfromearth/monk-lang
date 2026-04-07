# Codegen C Safety

Monk's compiler emits C. Bugs in codegen are silent — Go compiles clean,
the generated C compiles clean, and you get wrong output or a crash at
runtime. This rule encodes the failure patterns we have actually hit.

---

## 1. Compute the new value BEFORE freeing the old one

**Pattern:** `monk_free(old); old = compute(old);` — use-after-free.

**Rule:** Always compute the new value into a temp, then free the old, then assign.

```c
// WRONG — frees before computing
monk_free(arr);
arr = monk_array_set(arr, idx, val);   // arr is dangling

// RIGHT — compute first, free after
MonkValue new_arr = monk_array_set(arr, idx, val);
monk_free(arr);
arr = new_arr;
```

In Go codegen, this means: emit the RHS computation into a temp variable,
then emit the free, then emit the assignment. Check any `gen_stmt.go`
assignment path for this ordering.

---

## 2. Never reference a variable after its closing `}`

C scoping is strict. A temp variable declared inside `{ ... }` is gone
after the `}`. Codegen emits blocks as Go `strings.Builder` buffers and
flushes them — the flush point matters.

**Risk:** You buffer a variable declaration inside a block body,
then emit "save-back" or "default return" code after the buffer flushes.
The variable is out of scope at the C level even though it looked fine
in Go.

**Rule:** Any C code that uses a variable must be emitted inside the
same `{ ... }` block that declared it, before the closing brace.

If you need a value to survive a block, declare it in the outer scope
before the opening `{`.

---

## 3. Evaluate every expression exactly once

**Pattern:** `monk_array_set(arr, idx, f())` where the caller also uses
`f()`'s return value elsewhere — `f()` runs twice.

**Rule:** If a Monk RHS expression has side effects (function call,
subscript with mutation), emit it into a named C temp first. Never
inline a non-trivial expression into multiple C argument positions.

```c
// WRONG — f() called twice if codegen duplicates the node
monk_array_set(arr, idx, monk_call(f, args));
monk_print(monk_call(f, args));

// RIGHT — evaluate once
MonkValue _t = monk_call(f, args);
monk_array_set(arr, idx, _t);
monk_print(_t);
```

In Go, this means: if the same AST `Expr` node appears in multiple
generated C positions, emit it into a fresh temp and reference the temp.

---

## 4. Escape filenames in `#line` directives

`#line N "filename"` breaks if the filename contains backslashes,
double quotes, or newlines. All three are legal on some OS.

**Rule:** Always run source file paths through `cString()` before
embedding them in `#line` directives. `cString()` handles `\`, `"`,
`\n`, and `\r`.

```go
// codegen: emitting a line directive
fmt.Fprintf(w, "#line %d %s\n", pos.Line, cString(sourceFile))
```

---

## 5. No interpreter patterns

Monk is a compiler. Generated code must not contain:
- A `switch` over value tags to dispatch at runtime for operations that
  could be resolved at compile time
- A loop that re-evaluates AST nodes
- A runtime `eval` or `exec` equivalent
- Any call back into Go from generated C

If you find yourself emitting a `switch (val.tag)` for something the
type checker already resolved, use the unboxed codegen path in `unbox.go`
instead.

---

## 6. Temp variable names must not collide

Generated temp names like `_t0`, `_t1` are scoped to a function. If you
emit two independent temps with the same name in the same C function,
the second declaration shadows or errors.

**Rule:** Use a monotonically incrementing counter (`g.tmp()` or
equivalent) per-function, never per-expression.

---

## 7. Check the three layers when a test fails

Monk bugs surface at three independent layers. Don't stop at the first
one that looks wrong:

1. **Go compilation** — `go build ./...` fails → bug in Go codegen code
2. **C compilation** — `cc` rejects the emitted `.c` → structural codegen bug
   (missing semicolon, bad scope, undeclared variable, wrong `#include`)
3. **Runtime behavior** — binary runs but produces wrong output →
   semantic codegen bug (wrong operation, wrong value copy semantics)

A passing `go build` does not mean the generated C is correct.
A passing `cc` compile does not mean the program is semantically right.
Run the full integration test suite (`go test ./...`) — it exercises all
three layers.
