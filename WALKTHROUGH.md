# Code Walkthrough

A guide for developers reading the Monk compiler source code for the first time. No build tools required — just this file and the Go source in `src/`.

---

## The compiler pipeline

Open `src/main.go` and find `generateC`. It dispatches between two paths:

```go
// Single-file path (no imports):
prog, _ := syntax.Parse(source)        // 1. source text → AST
info, _ := types.Check(prog)           // 2. AST → typed Info
return codegen.GenerateWithTypes(prog, sourceFile, info)
                                        // 3. AST + Info → C source

// Multi-module path (has `use` statements):
graph, _ := module.Build(sourceFile)       // 1. resolve all imports
modInfo, _ := types.CheckModules(graph)    // 2. type-check all modules
return codegen.GenerateModules(graph, modInfo)
                                            // 3. all modules → single C file
```

Both paths produce one C source string. The rest of `generateC` pipes it through `cc -O3 -flto` to produce a native binary.

`monk check` runs parse + type-check only. `monk build` runs the full pipeline. `monk run` runs all, executes the binary, then deletes it.

---

## The five packages

### `src/syntax/` — Lexer + Parser + AST

**Input:** source text (string).  
**Output:** `*syntax.Program` — a slice of `syntax.Stmt` nodes.

| File | What it does |
|------|-------------|
| `token.go` | `TokenKind` constants, keyword table, `LookupIdent` |
| `scanner.go` | `Scan()` → `[]Token`. Hand-written, no regex. Handles strings, templates, numbers (hex/bin/oct/underscore), operators, comments. |
| `ast.go` | `Pos`, `Node`/`Expr`/`Stmt` interfaces, `Program`, shared types (`TypeExpr`, `Param`, `RecordField`) |
| `ast_expr.go` | 15 expression node types (`NumberExpr`, `BinaryExpr`, `CallExpr`, `FuncExpr`, ...) |
| `ast_stmt.go` | 14 statement node types (`VarDeclStmt`, `IfStmt`, `ForStmt`, `GuardStmt`, ...) |
| `parser.go` | `Parser` struct, `Parse()` entry point, token helpers |
| `parse_stmt.go` | One parser per statement kind. `parseStmt` dispatches by keyword. |
| `parse_expr.go` | Precedence-climbing expression parser. 13 levels, tightest last. See the comment at the top of the file for the full precedence table. |
| `parse_type.go` | Type annotation parsers (`int`, `string`, `int[]`, `int?`, `(int) -> bool`) |

**Key design decisions:**
- Assignment is a **statement**, not an expression — prevents `if x = 5` bugs.
- `{` at statement level is always a record literal, not a block. Blocks only exist inside `if`/`while`/`for`/`guard`/function bodies.
- Grouped expressions vs function literals are disambiguated by looking at tokens **inside** the `(`, not after `)`. See `parseParenOrFunc`.

### `src/types/` — Static type checker

**Input:** `*syntax.Program`.  
**Output:** `*types.Info` — maps AST nodes to their types.

| File | What it does |
|------|-------------|
| `types.go` | `Kind` enum (9 kinds: Any, Int, Float, Str, Bool, None, Array, Record, Func), `Type` struct, `AssignableTo` (7-rule compatibility function), singleton constructors (`Int`, `Float`, `ArrayOf`, `OptionalOf`, `FuncType`) |
| `checker.go` | `Info` struct, `Check()` entry point, `scope` chain, `Binding`, builtin declarations (40+ functions), `checkProgram` (three sub-passes) |
| `stmts.go` | Statement checks: `checkVarDecl`, `checkAssign`, `checkCompoundOp`, type resolver (`resolveTypeDef`/`resolveTypeExpr`), `funcSignature`, control-flow checks |
| `exprs.go` | Expression inference: `inferExpr`/`inferExprInner`, one `infer*` function per expression kind. Helper predicates (`isNumericOrAny`, `isIntOrAny`, `isNonComparable`) |
| `returns.go` | All-paths-return analysis: `stmtsAlwaysReturn`/`stmtAlwaysReturns`. Conservative — if it can't prove a path returns, it reports an error. |

**Key design decisions:**
- Three sub-passes in `checkProgram`: (1) resolve type declarations, (2) hoist function signatures, (3) check all statements. Order matters — types must exist before function params reference them, signatures must exist before recursive calls.
- The for-loop variable is declared `const` in a scope that is the **parent** of the body scope. This prevents `let i = ...` inside the body from shadowing the loop variable.
- `Info.Types` uses pointer keys — the exact `syntax.Expr` pointer the parser allocated. O(1) lookup, zero ambiguity. Codegen just passes the same pointer it already holds.

### `src/codegen/` — AST → C emitter

**Input:** `*syntax.Program` + `*types.Info`.  
**Output:** a `.c` file (or native binary after `cc` runs).

| File | What it does |
|------|-------------|
| `gen.go` | `generator` struct, `Generate`/`GenerateWithTypes` entry points, `generate` driver loop, `varStorage`/`saveStorage`/`restoreStorage` |
| `gen_stmt.go` | Statement emitters: `emitVarDecl` (3 paths: typed array, scalar probe, boxed), `emitAssign` (fast paths for unboxed scalars, typed array elements, record fields), `emitIf`, `emitWhile`, `emitFor` (typed-array fast path), `emitReturn`, `emitGuard` |
| `gen_expr.go` | **Boxed** expression emission: `emitExpr` returns a C expression of type `MonkValue`. Every arithmetic op goes through `monk_add`/`monk_sub`/etc. |
| `gen_func.go` | Function hoisting, `deriveFuncStorage`, `emitTrampoline`, `emitFuncValueNamed`, `hoistFunctionWithCaptures`. Closure capture save-back on return. |
| `gen_helpers.go` | `mangleName` (prefixes `mk_`), `cString` (escapes for C), `compoundToArith`, `builtinMap` (Monk name → C name) |
| `unbox.go` | **Unboxed** expression emission: `emitExprTyped` returns `(string, storageKind)`. When both sides of `+` are `storeInt`, emits raw `a + b` instead of `monk_add(a, b)`. Also handles typed-array element reads/writes. |
| `capture.go` | `freeVars` — free-variable analysis for closure captures. Walks AST, tracks locals, reports references to outer-scope variables. |
| `gen_bounds.go` | Bounds-check elision: `constVals`, `arrayLens`, `varBounds` tracking. `isBoundedSafe` proves array accesses are in-bounds at compile time so the runtime check can be skipped. |

**The two emission paths:**

1. **Boxed path** (`emitExpr` in `gen_expr.go`): always returns `MonkValue`. All ops go through `monk_*` runtime calls. Correct for everything.
2. **Unboxed path** (`emitExprTyped` in `unbox.go`): returns raw C types when the checker says a value is a scalar. `int + int` → raw `+`. No function call, no tagged union.

Both paths coexist. `boxExpr` wraps unboxed → boxed. `unboxExpr` extracts boxed → raw. `coerce(code, from, to)` handles both directions. When a function has all-scalar params and return, it gets an unboxed C signature (`int64_t fib(int64_t n)` instead of `MonkValue fib(MonkValue n)`).

**Key design decisions:**
- Two output buffers: `g.funcs` (hoisted above `main()`) and `g.body` (inside `main()`). Functions are hoisted so forward references and recursion work.
- `g.storage` maps variable names to storage kinds. `saveStorage`/`restoreStorage` snapshots around scope boundaries (if/else, while, for) so inner-scope variables don't leak.
- Compute new value **before** freeing old value. `x = x + 1` must not free `x` while evaluating the RHS. The boxed-path `emitAssign` uses a temp for this.

### `src/module/` — Module resolver

**Input:** the path of the entry `.monk` file.  
**Output:** `*module.Graph` — all reachable modules in topological order.

| File | What it does |
|------|-------------|
| `module.go` | `Build()` entry point. DFS over import graph: parses each file, validates exports, detects cycles, produces `Graph.Order` (deps first, entry last). `ResolvePath()` resolves `"./math"` to an absolute `.monk` path. `collectExportNames()` walks the AST to find `export` declarations. |
| `module_test.go` | 13 tests: path resolution, single/multi-module graphs, cycle detection, alias imports, wildcard imports. |

**Key design decisions:**
- DFS with gray/black state — standard cycle detection. Gray = currently visiting (cycle if revisited). Black = fully processed (safe to skip).
- The `chain` slice tracks the import path for human-readable cycle error messages: `circular import: a.monk -> b.monk -> a.monk`.
- Topological order is post-order DFS — a module is appended to `Graph.Order` only after all its dependencies are processed. This means `types.CheckModules` and `codegen.GenerateModules` can walk `Order` left-to-right and every dependency is already resolved.
- Module paths must start with `.` — no stdlib, no bare names. Every import is an explicit relative path.

### `src/runtime/` — C runtime library

Not Go. A small C library (~1,200 lines across 8 files) linked into every Monk binary.

| File | What it does |
|------|-------------|
| `runtime.h` | `MonkValue` tagged union, `MonkFunction` struct, all `monk_*` function signatures |
| `internal.h` | Shared helpers not used by generated code |
| `value.c` | Constructors, `monk_deep_copy`/`monk_free`, `monk_show`, typed-array converters |
| `arith.c` | Arithmetic: `monk_add`/`sub`/`mul`/`div`/`mod`, comparison operators |
| `string.c` | String ops: `concat`, `length`, `substring`, `split`, `trim`, `to_upper_case` |
| `container.c` | Array/record ops: `get`/`set`, `append`/`pop`/`slice`, `fill`, `range`, record `get`/`set` |
| `math.c` | `abs`, `floor`/`ceil`/`round`, `sqrt`/`pow`/`log`, trig |
| `builtins.c` | `typeof`, `is_*` type checks, file I/O, `env_get`, `exit`, `args` |
| `error.c` | `guard`/`against`/`throw` via `setjmp`/`longjmp` |
| `higher_order.c` | `map`, `filter`, `reduce` |

---

## The three structs that cross package boundaries

### 1. `*syntax.Program` — the AST

```go
type Program struct { Stmts []Stmt }
```

Every statement implements `Stmt`. Every expression implements `Expr`. Concrete types are in `ast_stmt.go` and `ast_expr.go`. Each node carries a `Pos` (line + column) for error messages and `#line` directives.

### 2. `*types.Info` — the type map

```go
type Info struct {
    Types map[syntax.Expr]*Type         // every expression → its type
    Decls map[*syntax.VarDeclStmt]*Type // every variable → its declared type
    Funcs map[*syntax.FuncExpr]*Type    // every function → its full signature
}
```

Codegen looks up the same AST pointer it already holds. Pointer comparison — O(1), no name resolution needed.

### 3. `*types.Type` — the type representation

```go
type Type struct {
    Kind       Kind                  // KindInt, KindStr, KindArray, ...
    Optional   bool                  // T? — the orthogonal nullable flag
    Elem       *Type                 // for arrays: element type
    Fields     []RecordTypeField     // for records: field names + types
    Params     []*Type               // for functions: parameter types
    Return     *Type                 // for functions: return type
    MinParams  int                   // for functions with defaults
}
```

9 kinds: Any, Int, Float, Str, Bool, None, Array, Record, Func. `AssignableTo(from, to)` is a 7-rule function that determines all type compatibility. Read it in `types.go` — it's the spec in code.

---

## The generator struct

All codegen state lives in one struct. Understanding its fields explains most of what the emitters do.

```go
type generator struct {
    funcs   strings.Builder          // hoisted C functions (above main)
    body    strings.Builder          // main() body

    info    *types.Info              // type annotations from checker
    storage map[string]storageKind   // variable name → C storage kind

    funcNames      map[string]string        // Monk name → C function name
    fnStorage      map[string]funcStorage   // C name → param/return storage kinds
    funcHasCapture map[string]bool          // does this func close over variables?
    funcDefaults   map[string][]syntax.Expr // default param expressions

    retStorage      storageKind      // return type of current function
    currentCaptures []string         // captured variable names (for save-back)

    constVals map[string]int64       // compile-time constants (bounds elision)
    arrayLens map[string]int64       // known array lengths (bounds elision)
    varBounds map[string][2]int64    // loop-counter ranges (bounds elision)
}
```

---

## How closures work

1. `freeVars()` in `capture.go` walks the function body and finds variables referenced but not declared locally.
2. The hoisted C function gets an extra `MonkFunction *_self` parameter.
3. At function entry, captured variables are loaded from `_self->captures[i]` into local C variables.
4. Before every `return`, `emitCaptureSaveBack()` writes mutated locals back to `_self->captures[i]`.
5. A trampoline bridges the generic `monk_call(fn, args, argc)` interface to the specific hoisted function signature.
6. Direct calls (where the compiler knows the function at the call site) bypass the trampoline.

---

## How unboxing works

When the checker says a variable is `int`, codegen can emit `int64_t` instead of `MonkValue`. The `storageKind` enum tracks this:

```
storeBoxed     → MonkValue (the default, always correct)
storeInt       → int64_t
storeFloat     → double
storeBool      → bool
storeIntArray  → MONK_INT_ARRAY (int64_t* backing store)
storeFloatArray → MONK_FLOAT_ARRAY (double* backing store)
storeBoolArray  → MONK_BOOL_ARRAY (bool* backing store)
```

`emitVarDecl` in `gen_stmt.go` shows all three paths in one function:
1. **Typed array** → `monk_int_array_from()` conversion, records array length for bounds elision.
2. **Scalar probe** → tries `emitExprTyped`. If the RHS produces a raw scalar, promotes the variable to unboxed.
3. **Boxed fallback** → `MonkValue` with `monk_deep_copy`.

For functions, `deriveFuncStorage` checks if ALL params AND the return type are raw scalars. If yes, the function gets an unboxed C signature. One boxed param forces the whole function to stay boxed.

---

## Bounds-check elision

Every typed-array access emits a bounds check by default. `gen_bounds.go` tracks three things to prove some accesses are safe:

1. **`constVals`** — compile-time constants (`let N = 400` → `constVals["N"] = 400`).
2. **`arrayLens`** — array lengths from `range()` calls (`let arr int[] = range(N)` → `arrayLens["arr"] = N`).
3. **`varBounds`** — loop-counter ranges (`while i < N` → `varBounds["i"] = [0, N-1]`).

`isBoundedSafe(arr, idx)` evaluates the index expression's range and confirms `min >= 0 && max < arr.length`. If both hold, the check is dropped. The analysis is conservative — unknown values always preserve the bounds check.

---

## Suggested reading order

| Step | File(s) | Time | Why |
|------|---------|------|-----|
| 1 | `src/syntax/token.go` | 5 min | Every keyword and operator. Understand the vocabulary. |
| 2 | `src/syntax/ast_stmt.go` + `ast_expr.go` | 10 min | The shape of every AST node. Skip the marker methods. |
| 3 | `src/types/types.go` | 10 min | The `Kind` enum and `AssignableTo`. These 7 rules determine all type compatibility. |
| 4 | `src/main.go` → `runCompile` | 5 min | The pipeline as code. See how the three packages wire together. |
| 5 | `src/codegen/gen.go` | 15 min | The `generator` struct and `GenerateWithTypes`. Understand the two-buffer layout. |
| 6 | `src/codegen/gen_stmt.go` → `emitVarDecl` | 20 min | Shows all three emission paths (typed array, scalar, boxed) in one function. |
| 7 | `src/codegen/unbox.go` → `emitExprTyped` | 15 min | The unboxed emission path side-by-side with `emitExpr` in `gen_expr.go`. |
| 8 | `src/runtime/runtime.h` | 10 min | What `MonkValue` looks like in C. Then the emitted strings become concrete. |

---

## How to add a new feature

### New AST node (e.g. a new expression)

1. Add the struct to `src/syntax/ast_expr.go`. Include `Pos` field.
2. Add `nodeKind()` + `exprNode()` marker methods in the same file.
3. Add a parser in `src/syntax/parse_expr.go` (or `parse_stmt.go` for statements).
4. If it needs a new token, add to `src/syntax/token.go`.
5. Add an `infer*` case to `src/types/exprs.go` (or `check*` to `stmts.go`).
6. Add an emit case to `src/codegen/gen_expr.go` (boxed path).
7. Optionally add an unboxed case to `src/codegen/unbox.go`.
8. Add a test in `src/codegen/codegen_test.go` (integration: parse → generate → compile → run → check stdout).
9. Update the directory's `INDEX.md`.

### New import form (e.g. `use X from "url"`)

1. Add any new tokens to `src/syntax/token.go`.
2. Extend the `UseStmt` AST node in `src/syntax/ast_stmt.go` if needed.
3. Update `parseUse` in `src/syntax/parse_stmt.go`.
4. Update `collectExportNames` / `Build` in `src/module/module.go` if the form changes how names are resolved.
5. Update `types.CheckModules` in `src/types/checker.go` to bind the new names.
6. Update `codegen.GenerateModules` to emit the right C declarations.
7. Add a test in `src/module/module_test.go` and `src/codegen/codegen_test.go`.

### New builtin function

1. Implement the C function in the appropriate `src/runtime/*.c` file.
2. Add its declaration to `src/runtime/runtime.h`.
3. Add the mapping to `builtinMap` in `src/codegen/gen_helpers.go`.
4. Add the type signature to `declareBuiltins()` in `src/types/checker.go`.
5. Add a test in `src/codegen/codegen_test.go`.
6. Update `src/runtime/INDEX.md`.

### New storage kind (e.g. a new unboxed type)

1. Add the constant to `storageKind` in `src/codegen/unbox.go`.
2. Add `storageFor()` mapping from `types.Kind` → new storage.
3. Add `cTypeName()`, `boxExpr()`, `unboxExpr()` cases.
4. Handle in `emitExprTyped` and relevant `emitStmt` paths.
5. If it's an array kind, add conversion/access helpers.

---

## Running tests

```bash
make test          # runs everything below

# Individual steps:
cd src
go test ./...      # all Go tests (parser, checker, codegen integration)
go vet ./...       # static analysis

# C runtime tests:
cc -std=c11 src/runtime/*.c -lm -o /tmp/rt_test && /tmp/rt_test

# All examples:
for f in examples/*.monk; do ./monk run "$f"; done

# Benchmarks (compile first, then time the binary):
./monk build bench/benchmarks/fibonacci/fibonacci.monk -o /tmp/fib
/usr/bin/time -p /tmp/fib
```

---

## Project conventions

- **Names:** `mk_` prefix on all Monk variable names in generated C (`mangleName`). `_monk_func_N` for hoisted functions. `_tmp_N` for temporaries.
- **Tests:** Integration tests, not unit tests. Each test compiles a Monk program, runs the binary, checks stdout.
- **Branches:** `phase-N-name` branched from main. Merge with `--no-ff`.
- **Spec-first:** Features start in `spec/REFERENCE.md`, then get implemented. If code disagrees with spec, one of them has a bug.
- **INDEX.md:** Every `src/` subdirectory has an `INDEX.md` listing files and contents. Update when files change.

---

## Further reading

| Resource | What it is |
|----------|-----------|
| `spec/REFERENCE.md` | The language spec. Source of truth for syntax and semantics. |
| `spec/ARCHITECTURE_DECISIONS.md` | Why Go, why compile-to-C, what can change later. |
| `PROGRESS.md` | Project history — what was built, pivots, mistakes, what's next. |
| `ROADMAP.md` | Phased build plan with checkboxes. |
| `knowledge/` site | Visual learning course (requires `npm run dev` to view locally, or visit [monkfromearth.github.io/monk-lang](https://monkfromearth.github.io/monk-lang/)). |
| `src/*/INDEX.md` | Per-directory file guides with design notes. |
