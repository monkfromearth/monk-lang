# `src/codegen/` — AST → C emitter

Entry: `Generate(prog, filename)` or `GenerateWithTypes(prog, filename, info)`.

The generator walks the AST twice: functions get **hoisted** above `main()`
as static C functions, everything else goes into the body of `main()`.

## Files

| File              | Contains                                                   |
| ----------------- | ---------------------------------------------------------- |
| `gen.go`          | `Generate` / `GenerateWithTypes` entry, `generator` struct, driver loop |
| `gen_stmt.go`     | statement emission (var decl, assign, control flow, guard) |
| `gen_expr.go`     | expression emission (**boxed** MonkValue path)             |
| `gen_func.go`     | function hoisting, trampolines, closures, capture save-back |
| `gen_helpers.go`  | `mangleName`, `cString`, `compoundToArith`, `builtinMap`   |
| `unbox.go`        | scalar-unboxing path (raw `int64_t`/`double`/`bool`)       |
| `capture.go`      | `freeVars` — free variable analysis for closure captures   |

## Two emission paths

1. **Boxed path** (`emitExpr`, `emitBinary`, `emitCall`) — every Monk value
   is a `MonkValue` tagged union. All operations go through `monk_*` runtime
   calls. Always correct, always available.
2. **Unboxed path** (`emitExprTyped` in `unbox.go`) — when the type checker
   proves a variable is a scalar, we emit raw C types and arithmetic. This
   is what delivers C-parity performance on numeric benchmarks.

Both paths coexist. When a boxed context consumes an unboxed value (e.g.
passing a raw `int64_t` where a builtin expects `MonkValue`), the generator
inserts a `boxExpr` wrapper. See `varStorage` and `coerce` in `unbox.go`.

## Key invariant

Compute the **new** value before freeing the **old** one. Otherwise
`x = x + 1` frees `x` while evaluating the RHS. The boxed-path `emitAssign`
uses a temp specifically for this reason.
