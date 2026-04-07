# `src/codegen/` — AST → C emitter

Entry: `Generate(prog, filename)` or `GenerateWithTypes(prog, filename, info)`.
Multi-module: `GenerateModules(graph, modInfo)` — single `.c` from a module graph.

The generator walks the AST twice: functions get **hoisted** above `main()`
as static C functions, everything else goes into the body of `main()`.

## Files

| File              | Contains                                                   |
| ----------------- | ---------------------------------------------------------- |
| `gen.go`          | `Generate` / `GenerateWithTypes` entry, `generator` struct, driver loop |
| `gen_stmt.go`     | statement emission: `emitStmt` dispatcher, `emitVarDecl`, `emitModuleVarDecl`, `emitAssign` |
| `gen_flow.go`     | control-flow emission: `emitIf`, `emitWhile`, `emitFor`, `emitReturn`, `emitGuard`, `emitCondition` |
| `gen_expr.go`     | expression emission (**boxed** MonkValue path)             |
| `gen_func.go`     | function hoisting, trampolines, closures, capture save-back |
| `gen_module.go`   | `GenerateModules` — multi-module → single `.c`, init functions, module-prefixed names |
| `gen_helpers.go`  | `mangledName`, `cString`, `compoundToArith`, `builtinMap`  |
| `unbox.go`        | scalar and typed-array unboxing (`int64_t`/`double`/`bool`, `storeIntArray` etc.), typed-array COW uniqueness tracking |
| `capture.go`      | `freeVars` — free variable analysis for closure captures   |
| `gen_bounds.go`   | static bounds analysis for bounds-check elision (`constVals`, `arrayLens`, `varBounds`, `isBoundedSafe`) |

## Two emission paths

1. **Boxed path** (`emitExpr`, `emitBinary`, `emitCall`) — every Monk value
   is a `MonkValue` tagged union. All operations go through `monk_*` runtime
   calls. Always correct, always available.
2. **Unboxed path** (`emitExprTyped` in `unbox.go`) — when the type checker
   proves a variable is a scalar, we emit raw C types and arithmetic. This
   is what delivers C-parity performance on numeric benchmarks.
3. **Typed-array backing-store path** (`emitIndexTyped` in `unbox.go`) — when
   the array is `int[]`/`float[]`/`bool[]`, the variable holds a `MONK_INT_ARRAY`
   etc. with a raw `int64_t*`/`double*`/`bool*` backing store. Element reads emit
   `arr.int_array_val->data[i]` (no union, no tag, cache-friendly). Writes emit
   a COW detach barrier only when the array may be shared, then
   `arr.int_array_val->data[i] = rhs`. Declarations call `monk_int_array_from()`
   which converts from generic `MONK_ARRAY` (e.g. `range(N)`) or deep-copies an
   existing typed array. Previous "inline access" path with `.array_val->data[i].int_val`
   replaced by this approach — halves element memory stride.
4. **Counter-loop path** (`emitFor` in `gen_flow.go`) — `for x in range(N)` is
   detected before emission and compiled as `for(int64_t x=0; x<N; x++)`. No
   allocation, no runtime call. The loop variable is `storeInt`.
5. **Specialized allocation** (`emitVarDecl` in `gen_stmt.go`) — `fill(N, val)` on
   typed arrays emits `monk_fill_bool`/`int`/`float` which allocate the backing
   store directly. `range(N)` on `int[]` emits `monk_range_int`. Avoids
   intermediate MonkValue arrays.

Both paths coexist. When a boxed context consumes an unboxed value (e.g.
passing a raw `int64_t` where a builtin expects `MonkValue`), the generator
inserts a `boxExpr` wrapper. See `varStorage` and `coerce` in `unbox.go`.

## Key invariant

Compute the **new** value before freeing the **old** one. Otherwise
`x = x + 1` frees `x` while evaluating the RHS. The boxed-path `emitAssign`
uses a temp specifically for this reason.
