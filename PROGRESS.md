# Monk Lang — Progress

What's been built, what pivots happened, what's next.

## Current release

**0.0.1 — Buniyaad** (2026-04-04). First local release. Self-contained binary, embedded runtime, works from any directory.

## The compiler

**Status: Phases 1-6 complete (all deferred items shipped). 631 tests (468 Go + 163 C runtime). Working end-to-end.**

`monk build hello.monk` compiles to a native binary via C. `monk run` compiles and runs in one step. `monk check` validates syntax. `monk version` prints `monk 0.0.1 — Buniyaad`.

### Build history

**Phase 1 — Lexer.** Hand-written scanner in Go. 100 tests. Reads source text, produces tokens. Handles keywords (27), operators (~26), delimiters (9), number literals (decimal, hex, binary, octal, underscores), strings (double-quoted with escapes, backtick template literals), comments. Maximal munch for multi-character operators.

**Phase 2 — Parser.** Recursive descent, 13-level operator precedence. 119 tests. Produces an AST with 29 node types (15 expressions, 14 statements). Key decisions: assignment is a statement (not expression — prevents `if x = 5` bugs), functions are expressions assigned to variables (no `func` keyword), space-separated type annotations (`let x int = 42` not `let x: int`), every node embeds `Pos{Line, Column}` for source mapping.

**Phase 3 — C Runtime.** Small C library (~830 lines) linked into every binary. MonkValue tagged union (int, float, string, bool, none, array, record, function). Pure value semantics — deep copy on every assignment, no refcounting, no GC. 40+ built-in functions (math, strings, arrays, file I/O, type checking). Error handling via setjmp/longjmp for guard/against/throw. 146 C tests.

**Phase 4 — Code Generation.** AST-to-C emitter. Two-pass: first collects and hoists functions as static C functions above main(), second emits program body. Temp variables for intermediate expression results. `#line` directives for source mapping. All operations go through runtime functions (`monk_add`, not C `+`). Caught and fixed a use-after-free bug (must compute new value before freeing old). 39 integration tests (compile + run + check stdout).

**Phase 5 — CLI.** `monk build`, `monk run`, `monk check`, `monk version`. ~280 lines, no framework. `-o` flag controls output: `.c` extension = emit C source, anything else = compile to binary. Runtime embedded in Go binary via `go:embed` — the `monk` binary is self-contained and works from any directory. Extracts runtime to `~/.cache/monk/runtime/` on first use if needed. Uses system `cc`. 28 CLI tests.

**Phase 6 — Type System + scalar unboxing codegen.** Static checker in `src/types/` runs after parse, before codegen — wired into all three commands. 110 checker tests. Catches: first-assignment inference mismatches, reassignment type drift, array element violations (both literals and index assignment), typed-record shape (missing/extra/wrong-type fields), cross-type equality (5 == "5" errors), loop variable const violations, missing returns on non-none functions, function call arity/type mismatches, function-type parameters like `(f (int) -> int, x int)`. Strict on equality (no deep-compare on arrays/records/functions), permissive on truthiness. The checker's type Info is threaded into codegen (`src/codegen/unbox.go`), which emits raw C scalars (`int64_t`, `double`, `bool`) for statically-typed scalar variables and raw C arithmetic between them. Scalar benchmarks now hit C parity: fib 1.0× C, mandelbrot 1.0× C, leibniz 1.0× C. Arrays are still tagged-union (matmul stays at ~12× C) — typed-array unboxing is the next performance frontier.

### Restructure (2026-04-04)

Flattened `src/cmd/monk/` to `src/`. The `src/` directory is now the Go module root:

```
src/
  main.go           CLI entry point
  embed.go          go:embed for runtime
  go.mod
  syntax/           Lexer + Parser + AST
  codegen/          AST → C emitter
  runtime/          C runtime (runtime.h, runtime.c)
```

Eliminated: `src/cmd/monk/`, `src/cmd/monk/runtime_files/` (copy hack), root-level `runtime/`, root-level `go.mod`, `src/monk.go`. Runtime embeds directly from `src/runtime/` — zero copies needed.

Build is now `make build` / `make install` / `make test` from project root.

### Hardening pass (2026-04-05)

Senior-engineer review of uncommitted changes. Ran CodeRabbit, go vet, staticcheck, golangci-lint v2.11.4, govulncheck — all clean. Fixed bugs and added tests:

**Real bugs fixed:**
- **Dangling `-o` flag silently ignored.** `monk build foo.monk -o` with no value used to fall through to the default output path. Now errors: "monk build: -o requires an output path".
- **Nested `-o` paths failed with "no such file".** `monk build foo.monk -o build/bin/app` now runs `os.MkdirAll` on the parent directory first.
- **Cache write race.** `~/.cache/monk/runtime/runtime.{h,c}` was rewritten on every build invocation despite the comment claiming it only wrote when missing. Two parallel `monk build` commands could produce a half-written header mid-read. Fixed with `writeIfChanged` (content-compare before write).
- **Homebrew symlink not resolved.** `findRuntime` used `os.Executable()` directly, which returns the symlink path. Now calls `filepath.EvalSymlinks` first so `/opt/homebrew/bin/monk → Cellar/...` finds the runtime next to the real binary.
- **#line directive didn't escape the filename.** Paths with backslashes (Windows), quotes, or newlines produced broken C. Now run through existing `cString()` helper. `cString()` also gained `\r` escaping.
- **`monk run` leaked temp dirs on non-zero child exit.** `os.Exit(exitCode)` was bypassing the deferred `os.RemoveAll`. Fixed by having `cmdRun` return the exit code and letting `main` call `os.Exit` after defers run.

**Documentation fixes:**
- `spec/C_RUNTIME.md` Type Checking section — noted `monk_typeof` returns `MONK_STRING`, not `MONK_BOOL`.
- `editor/vscode/src/builtins.ts` — `to_int`/`to_float` said "Returns none on failure" but the runtime throws via `monk_panic`. Changed to "Throws on invalid input".

**Hygiene:**
- Removed `monk-lang-0.1.0.vsix` from git (build artifact). Added `editor/vscode/*.vsix` to `.gitignore`.

**New tests (14):**
- `TestBuildDashOMissingValue` — dangling `-o` errors
- `TestBuildDashOBeforeSource` — documents current behavior, guards against regression
- `TestBuildNonMonkExtension` — source without `.monk` ext → `.out` suffix, original not overwritten
- `TestBuildNestedOutputPath` / `TestBuildNestedOutputCPath` — nested `-o` auto-creates dirs
- `TestMonkRuntimeDirInvalidFallsThrough` — invalid env var → embedded runtime, no stderr leak
- `TestEmbeddedRuntimeExtraction` — cold-start path ($HOME cache extraction)
- `TestEmbeddedRuntimeCacheIsIdempotent` — mtime unchanged on identical content, rewrites after corruption
- `TestRunCleansUpTempDirOnNonZeroExit` — no `/tmp/monk-run-*` leak when child exits non-zero
- `TestCString` — unit table for escape function (added `\r` case)
- `TestCodegenFilenameWithBackslashes` — integration: Windows-style path → valid compilable C

Tests: 446 → 460. Linters: all clean. Binary builds, 3.0 MB.

### Benchmark suite (2026-04-05)

Built `bench/` — first performance numbers against C, Go, Python, Node, Bun. See `bench/README.md` for methodology, `bench/PLAN.md` for design notes.

**Results on Apple M4 Pro (× = slower than C):**

| Benchmark | C | Monk | Go | Bun | Node | Python |
|---|---|---|---|---|---|---|
| fibonacci (n=35) | 1× | 6.6× | 1.3× | 2.4× | 5.0× | 40× |
| mandelbrot (800×800, 50 iter) | 1× | 17× | 1.1× | 1.9× | 3.8× | 185× |
| matmul (400×400 int) | 1× | 90× | 2.6× | 5.3× | 9.5× | 760× |

**What the numbers say:**
- **Fibonacci** — Monk ~6.5× C. Call overhead + tagged-union dispatch is the cost. No allocation.
- **Mandelbrot** — Monk ~17× C. Every float op goes through `monk_mul`/`monk_add`. Tagged union prevents cc from vectorizing or keeping floats in registers.
- **Matmul** — Monk ~90× C. Hot-path array indexing is the worst case for value semantics — each `arr[i]` triggers bounds check + unwrap + deep copy. This is the honest cost of the design model.

Ahead of Python in all three, behind Go/Bun/Node. Not a target — a baseline to measure against as the compiler matures.

**Bugs surfaced while writing the benchmarks:**
- Parser treats `(fn_call(x) / y)` as a function literal, fails with "expected parameter name". Workaround: intermediate variables.
- `to_float` only accepts strings — no int→float conversion. Workaround: `x * 1.0`.

Both logged in `bench/PLAN.md` for future fixes.

### Codegen performance pass (2026-04-05)

Initial benchmarks were bad. matmul was 90× C, mandelbrot 17× C, fibonacci 6.6× C. Went hunting for generic wins that don't require Phase 6's type system.

**Applied two changes. No new features, no benchmark-specific code paths.**

**1. `cc -O3 -flto` for generated C.** Was `-O2`. LTO lets `cc` inline across translation units, so `monk_add`/`monk_mul`/`monk_array_get` all fold into callers. Changed in [src/main.go:130,185](src/main.go#L130).

**2. Inline fast-path for `monk_deep_copy` / `monk_free`.** These fire on every `monk_array_set` (once each). For primitive (int/float/bool/none) elements both are no-ops, but they were real function calls with 16-byte argument passing. Split into header-inlined wrappers (primitive short-circuit) + `_heap` slow paths in runtime.c. Generic — helps any code using arrays of primitives.

**Results on Apple M4 Pro — speedup over pre-optimization Monk:**

| Benchmark | Before | After | Speedup | vs C |
|---|---:|---:|---:|---:|
| fibonacci | 110 ms | 27.5 ms | 4.0× | 1.6× |
| mandelbrot | 242 ms | 17.0 ms | 14.2× | **1.0×** |
| matmul | 912 ms | 142 ms | 6.4× | 13.9× |

Mandelbrot hits parity with C. Fibonacci beats Bun/Node/Python. Matmul still slow — that remaining gap is value-semantics fundamental and needs Phase 6 (type inference → unboxed int arrays) to close.

All 460 tests pass. All example programs still work. See `spec/PERFORMANCE.md` for the full analysis.

### Phase 6 — Type System (2026-04-05)

New `src/types/` package, ~650 lines. Runs between parse and codegen.

**Type model:**
- Sum type via `Kind`: Any, Int, Float, Str, Bool, None, Array, Record, Func
- `Optional` flag orthogonal to kind (T? for any T)
- `AssignableTo(src, dst)` encodes compatibility: identity, Any wildcards, numeric widening (int→float), optional acceptance (none→T?, T→T?), structural records, element-wise arrays, exact function signatures

**Checks enforced:**
- First-assignment inference + reassignment consistency
- Typed array element enforcement (both at literal and on `arr[i] = x`)
- Typed record shape — missing/extra/wrong fields caught
- Cross-type equality is a type error (`5 == "5"` fails)
- Arrays/records/functions can't be compared with `==` (no deep-compare)
- Loop variable is const (spec requirement)
- All-paths-return: non-none fn must return on every path (if/else + throw recognized, loops conservatively rejected)
- Function call arity + per-argument type checks
- Function-type annotations: `(f (int) -> int, x int)`
- Forward-reference hoisting: recursion + functions calling each other work

**Scope boundaries (intentional):**
- Any flows through unknowns — builtins are typed as Any in slots where we can't express union types today (e.g. `length` of array-or-string)
- Codegen is NOT touched. AST still flows through to codegen unchanged, still emits tagged-union MonkValue. Unboxing = Phase 6.5.

Two paper cuts fixed en route:
- Parser bug: `(to_float(y) / 2.0)` misread as function literal. Fixed by scanning to matching `)` and checking for `->`.
- `to_int`/`to_float` widened to accept int/float/string (were string-only). Spec philosophy: these are THE explicit coercion functions.

Post-merge PR review caught 4 more issues:
- **Compound assignment bypassed arithmetic check.** `let s = "hi"; s -= "world"` passed the checker and crashed at runtime. Added `checkCompoundOp` that validates the implied binary operation on all three assignment targets (identifier, index, property). Codegen updated to dispatch `+=` through `monk_string_concat` for the string-concat overload (mirrors binary `+`).
- **parseFuncExpr didn't accept LeftParen as return-type start.** `(k int) (int) -> int { ... }` failed to parse.
- **For-loop body shared scope with loop variable.** A `let i = ...` inside the body would overwrite the const loop binding. Now body gets its own child scope (Go-style shadowing allowed, direct mutation still rejected).
- **inferEquality asymmetric error messages.** `42 == [1,2]` and `[1,2] == 42` now produce the same message.

**Verification:**
- 110 checker tests + 2 CLI integration tests
- All 13 example programs typecheck AND run (added types.monk, records.monk, optionals.monk, guards.monk)
- All 3 benchmark programs typecheck
- All linters clean (go vet, staticcheck, golangci-lint, govulncheck)

### Scalar codegen unboxing (same session, rolled into Phase 6)

The type checker was sitting on rich information that codegen was ignoring — we were still emitting `MonkValue` everywhere. Unboxing closes that gap.

**What changed:**
- `types.Check` now returns `*types.Info` with per-expression types, per-decl types, per-function signatures
- New `src/codegen/unbox.go` — the `storageKind` sum type (MonkValue / int64_t / double / bool), the `coerce` helper, typed variants of expr/binary/unary/call emission
- Variables declared with a scalar type (int/float/bool) are now stored as raw C scalars, not MonkValue
- Arithmetic between two scalar operands stays raw — `a + b * c` compiles to `(a + (b * c))`, no `monk_add` calls
- Functions whose params AND return are all scalar get unboxed C signatures: `static int64_t _monk_func_1(int64_t mk_n)` instead of `static MonkValue`
- `if` / `while` conditions that evaluate to typed booleans become raw C, bypassing `monk_is_truthy`
- `to_int(float_var)` / `to_float(int_var)` inline as direct C casts, skipping the runtime call

**Benchmark results (Apple M4 Pro, cc -O3 -flto, new runs after unboxing):**

| Benchmark | Monk | vs C | Before unboxing |
|---|---:|---:|---:|
| fibonacci (n=35) | 17.3 ms | **1.0× C** | 1.6× C (28 ms) |
| mandelbrot (800²×50) | 14.5 ms | **1.0× C** | 1.2× C (17 ms) |
| leibniz (π, 50M iter, NEW) | 29.2 ms | **1.0× C** | — |
| trial_primes (<200k, NEW) | 5.9 ms | **1.0× C** | — |
| matmul (400²) | 122 ms | 11.8× C | 13.9× C |

Matmul didn't move much — arrays are still tagged. Trial_primes initially wrote the inner loop's "early exit" as `d = n` (assignment to break the loop condition) rather than Monk's real `break` keyword; once switched to `break`, Monk hit C parity. Lesson documented for future benchmarks.

**New benchmark programs:**
- `bench/benchmarks/leibniz/` — π via Leibniz series, pure float compute
- `bench/benchmarks/trial_primes/` — prime counting by trial division, nested int loops

**Codegen tests:** 11 new unboxing tests assert BOTH the generated C shape AND the runtime result for each optimization path.

### File-structure refactor (2026-04-05)

Big files were masking what each module does. Split each monolithic file into
focused, single-purpose files. No semantic changes — tests remained green
across the refactor.

**Before → After (line counts):**

- `syntax/ast.go` (348) → `ast.go` (91) + `ast_expr.go` (132) + `ast_stmt.go` (141)
- `syntax/parser.go` (1174) → `parser.go` (99) + `parse_stmt.go` (398) + `parse_expr.go` (516) + `parse_type.go` (185)
- `codegen/codegen.go` (883) → `gen.go` (136) + `gen_stmt.go` (337) + `gen_expr.go` (215) + `gen_func.go` (126) + `gen_helpers.go` (108)
- `runtime/runtime.c` (858) → `value.c` (337) + `arith.c` (112) + `string.c` (156) + `container.c` (130) + `math.c` (61) + `builtins.c` (76) + `error.c` (50) + new `internal.h` (37)

Each directory now has an `INDEX.md` pointing to the right file.

**C runtime split details.** Previously `cc` took a single `runtime.c`; now
it takes the 7 split `.c` files. Internal helpers (`monk_malloc`,
`monk_strdup`, `monk_realloc`, `utf8_strlen`, `utf8_offset`, `monk_to_go_float`
renamed to `monk_as_c_double`, and `value_to_str` renamed to
`monk_value_to_cstr`) became non-static and moved behind `internal.h`. The Go side embeds the entire `runtime/` directory via `embed.FS` — adding a new `.c` file requires only updating `runtime/INDEX.md`; no Go code changes needed.

**Parser edge cases fixed en route:**

- `(x none) ...` — param with `none` type. `none` is its own token kind
  (not `Identifier`), so the heuristic that decides param-vs-grouped-expr
  wasn't recognizing it. Now uses `isTypeName` for the check.
- `() (T) -> T { ... }` — zero-param function with function-type return.
  Dispatch after seeing `()` now accepts `(` as a valid next token (for
  the function-return annotation), not just type-name and `{`.

Both had regression tests added (`TestParseFuncParamWithNoneType`,
`TestParseFuncNoParamsFuncReturnType`). The earlier-reported
`(to_float(y) / 2.0)` false-positive was already fixed in Phase 6; removed
the stale note from `bench/PLAN.md`.

**Codegen bug surfaced by splitting + running all examples:** the hoisted-
function storage map was being mutated BEFORE `saveStorage()`. When a
fully-unboxed function (e.g. `validate_age (age int) int`) was hoisted,
its param's storage (`mk_age → storeInt`) persisted into the next
hoisted function's body. A later boxed function with a same-named param
(`process (age int) string`) inherited that stale binding, and its
`validate_age(age)` call compiled as `_monk_func_2(mk_age)` — passing a
`MonkValue` where an `int64_t` was expected. Fixed by snapshotting
storage BEFORE writing the param bindings, and by always recording the
param's storage (even `storeBoxed`) so inner lookups can't fall through
to a sibling's binding. Regression test: `TestUnboxFuncParamsDoNotLeakBetweenSiblings`.

**CodeRabbit review findings applied:** Ran `coderabbit review --plain
-t uncommitted` on the refactor branch. Two passes surfaced ~17 findings
total (LLM non-determinism — different issues flagged each run). All
pre-existing pre-refactor code, exposed by the smaller file sizes.
Judged per `.claude/rules/code-review-workflow.md`:

- **ACT (fixed):** `realloc(old, 0)` is implementation-defined — added
  the same `size > 0 ? size : 1` guard that `monk_malloc_internal` has.
- **ACT (fixed):** emitted `malloc(_clen+1)` in the string for-loop
  didn't check for NULL — now panics on OOM, matching the runtime's
  convention.
- **ACT (fixed):** `monk_array_set(&obj, …)` and `monk_record_set(&obj, …)`
  emitted invalid C when target.Object was a non-identifier rvalue
  (e.g. `matrix[i][j] = x`). Detect the IdentExpr common case for the
  fast in-place path; route complex targets through a temp so the C
  compiles. The no-op-on-nested-mutation behavior is correct per Monk's
  value semantics (matrix[i] returns a deep copy). Regression tests:
  `TestCodegenNestedIndexAssignCompiles`, `TestCodegenNestedPropertyAssignCompiles`.
- **ACT (doc):** added a comment to `monk_to_upper_case`/`monk_to_lower_case`
  explaining these are ASCII-only — Monk's tiny-runtime goal excludes
  pulling in ICU tables.
- **ACT (fixed):** `monk_neg(INT64_MIN)` and `monk_abs(INT64_MIN)` —
  negating INT64_MIN overflows int64_t (C undefined behavior). Both
  now panic with a clear overflow message.
- **ACT (fixed):** `monk_file_write` ignored the fputs return value —
  silent write failures on disk-full / permission errors. Now captures
  the return, closes the file, and panics on EOF or fclose failure.
- **ACT (defensive):** `parseTypeExpr` read `p.current().Text`
  unconditionally. Callers already guard with `isTypeName` checks, but
  added a defensive fallback that records a "expected type name"
  error — malformed input now surfaces a readable message instead
  of swallowing a junk token into an AST.
- **SKIP:** bitwise ops accessing `.int_val` directly — type checker
  rejects non-int operands at all bitwise sites; runtime guards would
  duplicate checker work and none of the existing bitwise ops
  (Amp/Pipe/Caret/Shift*/Tilde) guard. Consistent.
- **SKIP:** anonymous-function values returned as `monk_none()` —
  documented deferred feature; first-class function values need Phase 8
  (FFI / closure captures).
- **SKIP:** potential use-after-free on string for-loop's `_ch` — value
  semantics guarantees every downstream consumer deep-copies the loop
  variable, so captured references always own their strings by the time
  `free(_ch)` runs. Added a comment explaining why the raw `char*` /
  `free` pattern is safe here.
- **SKIP:** `monk_floor`/`ceil`/`round` cast double to int64_t (UB for
  values outside ±9.2e18) — matches spec's graceful-on-reads policy;
  bolting range checks onto every math builtin adds runtime cost for
  an edge case users don't hit in numeric computation.
- **SKIP:** `monk_add` Plus path — reviewer flagged `right` as
  potentially double-evaluated, but it's in the two branches of a
  ternary operator which evaluates exactly one branch.
- **SKIP:** `emitUnboxedCall` accessing `fs.Params[i]` without arity
  guard — type checker verifies arity match at every call site before
  codegen runs.

### Pre-completion check process

Added `.claude/rules/pre-completion-checks.md` codifying the mandatory
check battery before declaring work done: gofmt, build, `go vet`,
`staticcheck`, `golangci-lint`, `govulncheck`, full Go test suite with
cleared cache, C runtime tests, all 13 examples, all 5 benchmarks with
expected-checksum verification, CodeRabbit uncommitted review, and
`INDEX.md` updates for every touched directory. Referenced from CLAUDE.md
alongside the existing code-review-workflow rule.

### Pivots and mistakes

- **Started with Zig, switched to Go.** The compiler is a text-in/text-out translator. Go's tree manipulation and string handling fit better. Zig/Rust deferred to future native backend.
- **Built an interpreter, then deleted it.** 2,792 lines of tree-walking interpreter + Go builtins removed. Monk is a compiler with no REPL. Lesson: read the architecture doc before writing code.
- **Assignment was originally an expression.** Interpreter thinking. Fixed to statement in both spec and parser.
- **AST nodes had no position info.** Added `Pos{Line, Column}` after realizing codegen needs `#line` directives.
- **Use-after-free in codegen.** `monk_free(old); old = monk_deep_copy(expr_using_old)` — freed before computing. Fixed: compute first, then free.
- **Made creative decisions without asking.** Sanskrit release names, FFI syntax. Lesson: present options for naming/branding.
- **Runtime embedding required file copies.** `go:embed` only sees files within the package directory. Initially hacked with `runtime_files/` copy. Fixed by moving runtime into `src/runtime/`.

### VS Code / Cursor Extension (2026-04-04)

Built `editor/vscode/` — a full VS Code extension for Monk v2:

- **Syntax highlighting** — TextMate grammar covering all 27 keywords, operators, builtins, number formats (hex/binary/octal/underscores), strings, template literals, comments
- **Autocomplete** — completions for all 40+ builtins (with signatures and docs), keywords (with smart snippet expansion), types, array types, optional types, and user-defined variables extracted from the document
- **Hover docs** — hover over any builtin to see signature, description, and return type. Hover over keywords and types for descriptions.
- **Snippets** — 30+ snippets: `fn`, `for`, `guard`, `rec`, `closure`, `fread`, `fwrite`, etc.
- **File icons** — SVG candle icons (light + dark) matching the Monk brand
- **Markdown injection** — syntax highlighting in ```monk code blocks
- **Icon theme** — `.monk` and `.mnk` files get the candle icon in the file explorer

Packaged as `monk-lang-0.1.0.vsix`. Works in VS Code and Cursor.

### Examples and benchmark expansion (2026-04-06)

Added 7 new example programs (13 → 20) and 4 new benchmarks (5 → 9) to showcase what the language can do today and to establish broader performance baselines.

**New examples:**
- `strings.monk` — split, trim, index_of, substring, to_upper/lower, char iteration, template literals
- `bitwise.monk` — `&` `|` `^` `~` `<<` `>>`, hex/binary literals, popcount, power-of-2 check
- `math.monk` — abs, floor/ceil/round, pow, sqrt, log, sin/cos/tan, asin/acos/atan, min/max, distance formula
- `binary_search.monk` — while loops, integer arithmetic, array indexing
- `value_semantics.monk` — copy-on-assign for arrays/records, const deep freeze, return-new-data pattern, sort proof
- `gcd_lcm.monk` — Euclid's algorithm, recursion, functions calling functions, coprimality check
- `sieve.monk` — Sieve of Eratosthenes, array alloc + mutation, twin prime detection

**New benchmarks (all with C/Go/JS/Python reference implementations + expected.txt):**

| Benchmark | What it measures | Expected checksum |
|-----------|-----------------|-------------------|
| `sieve` | Array alloc + index-write in nested loops (primes up to 1M) | 78498 |
| `ackermann` | Deep recursion stress (A(3,11), millions of recursive calls) | 16381 |
| `collatz` | While-loop + conditional branching (longest chain, n ≤ 1M) | 837799 |
| `binary_trees` | Array allocation/iteration stress (pool-based, depth 14) | 1622016 |

**Bugs and limitations surfaced while writing examples:**
- **Function param deep copy missing.** Codegen passes arrays/records to functions without deep-copying at entry. Calling code's data is mutated through the function's parameter. Spec says function args are copies. Workaround: assign param to a local variable inside the function before mutating.
- **`abs()` returns float for int args.** Type checker rejects `abs(a*b)` in int-returning functions. Workaround: `let my_abs = (x int) int { if x < 0 { return 0 - x } return x }`.
- **Hex underscore literals generate invalid C.** Lexer accepts `0xFF_80_00` but codegen passes the literal through to C, which doesn't support underscores. Workaround: don't use underscores in hex literals.
- **map/filter/reduce not wired.** Spec defines them, runtime has them, but type checker doesn't recognize them as builtins. Can't use higher-order functions in examples.
- **Default parameter values not implemented.** Parser captures them but type checker rejects calls with fewer args than params.
- **First-class function values not in codegen.** Returning functions from functions, storing closures — all parse and type-check but codegen emits `monk_none()`. Blocks closures example.

All six logged in ROADMAP.md Phase 6 deferred section.

### Three bug fixes (2026-04-06)

Fixed three bugs surfaced while writing the new examples:

**1. Function param deep copy (value semantics violation).**
Codegen passed arrays/records to functions without deep-copying. A function mutating `arr[0] = 999` modified the CALLER's data — violating spec rule 3 ("values, not references"). Fix: emit `monk_deep_copy()` for every boxed param at function entry in `gen_func.go`. Scalar params (int/float/bool) skip the copy — they're stack values with nothing to alias. 2 integration tests.

**2. `abs()` forced float return type.**
The type checker declared `abs` as `(Any) -> Float`, but the spec says `abs(x: number) -> number` — type-preserving. The C runtime already preserves the kind (int in → int out). Fix: changed `abs` declaration to `(Any) -> Any` in `checker.go`, matching `min`/`max`. 2 checker tests.

**3. Underscore numeric literals generated invalid C.**
Lexer correctly accepts `1_000_000` and `0xFF_80_00` per spec, but codegen passed `e.Value` through to C, which doesn't support underscores in constants. Fix: `strings.ReplaceAll(e.Value, "_", "")` in both boxed (`gen_expr.go`) and unboxed (`unbox.go`) emission paths. 4 integration tests.

Tests: 565 → 575 (424 Go + 151 C runtime). All linters clean.

### Phase 6 deferred items (2026-04-06)

Completed all six items deferred from Phase 6. Previously blocked higher-order patterns, closures, and default arguments.

**1. First-class function values + closures in codegen.**

Rewrote `src/codegen/gen_func.go`. Every anonymous function expression now emits:
- A hoisted static C function `_monk_func_N(...)` with the actual body
- A trampoline `_monk_func_N_thunk(MonkFunction *_self, MonkValue *args, int64_t argc)` matching the `MonkFuncPtr` typedef
- A `monk_make_function(thunk, captures, count)` C expression at the use site

Closures capture via `_self->captures[]`. At function entry, captured variables are loaded as locals. Mutations are written back to `_self->captures[]` before every `return` (explicit or fallback). This implements capture-by-copy with persistent state within a closure instance — matching the spec.

New `src/codegen/capture.go` computes free variables (variables referenced in the body but not declared there or as params). Sibling hoisted-function names are excluded from captures — they're available as MonkValue locals.

`MonkFuncPtr` signature gained a `MonkFunction *self` first parameter (was `(MonkValue *args, int64_t argc)`) so closures can access their capture slot. All trampoline and runtime code updated.

**2. Default parameter values.**

Type checker (`src/types/stmts.go`): validates default expressions, enforces trailing constraint (required params before defaults), computes `MinParams` on the `Type` struct. Arity check (`src/types/exprs.go`): `MinParams ≤ len(args) ≤ len(Params)`.

Codegen: `padDefaults(cName, args)` in `gen_expr.go` fills missing trailing args from the `funcDefaults` map at call site. No runtime cost — defaults emitted inline.

**3. map / filter / reduce wired.**

New `src/runtime/higher_order.c`: C implementations calling through `fn.func_val->fn(fn.func_val, ...)`. Memory ownership: `monk_map` deep-copies via `monk_array` and frees callback results; `monk_filter` borrows from source then deep-copies on output; `monk_reduce` owns accumulator, frees after each step.

Type checker (`src/types/checker.go`): added `map/filter/reduce` as known builtins with typed signatures. `funcExactMatch` updated to treat `KindAny` as a wildcard in both param and return positions — allows `(int)->int` callbacks where `(Any)->Any` is expected.

**4. `abs()` return type, underscore literals, param deep copy** — these three were shipped as part of the "Three bug fixes" session above and are listed there.

**Infrastructure improvements (same session):**

- `pre-completion-checks.md` rewritten to be fully self-discovering: C runtime file list via glob, example count via glob, benchmark expected values read from `expected.txt` per directory. No manual updates when adding files.
- `src/embed.go` replaced 10 individual `//go:embed` vars with a single `//go:embed runtime` + `embed.FS`. `runtimeSources()` and `extractEmbeddedRuntime()` now derive file lists from the FS. `codegen_test.go` uses `os.ReadDir`. Adding a new `.c` runtime file requires zero changes to Go code.
- **Void closure bug (CodeRabbit find):** closures with no explicit `return` were silently discarding capture mutations. The fallback save-back was emitted OUTSIDE the inner body block, so captured variables were out of C scope. Moved to inside the block. Regression test added.
- `higher_order.c` C-level unit tests added to `runtime_test.c` (12 new tests: map/filter/reduce on empty arrays, single elements, and multi-element arrays).

**New examples:** `closures.monk`, `higher_order.monk`, `default_params.monk`.

Tests: 575 → 631 (468 Go + 163 C runtime). All 23 examples pass. All 9 benchmarks match expected. All linters clean.

### Typed array unboxing (2026-04-06)

`int[]`, `float[]`, `bool[]` arrays now get inline element access instead of `monk_array_get` / `monk_array_set` calls. The MonkValue wrapper is preserved (boxing at function boundaries is free), but individual element reads and writes emit direct C struct-field access:

```
arr[i]     →  ({int64_t _t=i; ...arr.array_val->data[_t].int_val;})
arr[i] = v →  { int64_t _t=i; if (OOB) panic; arr.array_val->data[_t].int_val=v; }
```

**Three new storage kinds in `unbox.go`:** `storeIntArray`, `storeFloatArray`, `storeBoolArray`. They share `MonkValue` as the C type (the container is still a `MonkArray*`), but `emitExprTyped` now has an `IndexExpr` case that returns raw scalars for typed arrays.

**Scalar promotion in `emitVarDecl`:** When a variable has no explicit type annotation (e.g. `let aik = A[i*N+k]`), and the RHS emits as a raw scalar, the variable is promoted to that scalar kind. `aik` becomes `int64_t` instead of `MonkValue`. This lets the full arithmetic chain inside matmul stay raw.

**OOB behavior:** Typed array access is STRICT (panics on OOB) rather than graceful (none). This follows the spec's "operating on invalid data = error" rule. Explicitly-annotated optional variables (`let x int? = arr[i]`) preserve the graceful `monk_array_get` path — the scalar promotion only applies to unannotated declarations.

**`deriveFuncStorage` fixed:** the `All` flag now uses `isRawScalar()` instead of `!= storeBoxed`, so functions with `int[]` params don't incorrectly claim to be fully unboxed (their C signature stays `MonkValue`).

**Matmul results (Apple M4 Pro, cc -O3 -flto):**

| Benchmark | Before | After | Speedup | vs C |
|---|---:|---:|---:|---:|
| matmul (400×400 int) | ~100 ms | ~30 ms | **3.3×** | ~3× C |

The remaining 3× gap vs C is the `MonkValue[]` element layout (16 bytes vs 8 bytes for raw `int64_t`) — 2× worse cache stride plus bounds checks. Closing to 1× C requires a separate `int64_t*` backing store (future: typed array storage refactor). All other benchmarks unchanged (scalar benchmarks already at C parity).

**9 new correctness tests** in `TestTypedArray*` covering int/float reads, writes, arithmetic chains, the 2×2 matmul micro-kernel, and OOB panic verification.

Tests: 631 → 640 (477 Go + 163 C runtime). All examples pass. All benchmarks match expected. All linters clean.

### Typed array backing store (2026-04-06)

`int[]`, `float[]`, `bool[]` variables now use a tight C backing store — `int64_t*`, `double*`, `bool*` — instead of `MonkValue*`. Every element drops from 16 bytes (tagged union) to 8 bytes (`int64_t`) or less. Cache-line efficiency doubles for the common int case.

**Three new C runtime types in `runtime.h`:** `MonkIntArray { int64_t *data; int64_t length }`, and float/bool variants. Three new `MonkValueKind` values: `MONK_INT_ARRAY`, `MONK_FLOAT_ARRAY`, `MONK_BOOL_ARRAY`. The `MonkValue` union gains three new pointer fields (`int_array_val` etc.). From the Monk language perspective, typed arrays are still `"array"` — `typeof`, `is_array`, `length` all return the same values as for generic arrays.

**Three converter functions:** `monk_int_array_from(v)`, `monk_float_array_from(v)`, `monk_bool_array_from(v)`. Each handles two inputs:
- `MONK_ARRAY` input: extracts scalar fields from each element, frees the input, returns typed array.
- Same-typed input: deep-copies the `int64_t*` data, leaves input intact.

Codegen emits `monk_int_array_from(rhs)` at every `int[]` variable declaration. For `let A = range(N)`, `range(N)` returns a generic `MONK_ARRAY`, and `monk_int_array_from` converts and frees it in one shot.

**Codegen changes:**
- `emitVarDecl`: calls `monk_int_array_from()` / `monk_float_array_from()` / `monk_bool_array_from()` instead of `monk_deep_copy()` for typed array variables.
- `emitIndexTyped`: switched from `arr.array_val->data[i].int_val` to `arr.int_array_val->data[i]` — direct pointer dereference, no union.
- `emitAssign` typed write: `arr.int_array_val->data[i] = rhs` instead of `arr.array_val->data[i].int_val = rhs`.
- `emitFor`: added typed-array iteration paths — boxes each element back to `MonkValue` for the loop body.

**Runtime updates:** All structural mutators (`append`, `prepend`, `pop`, `drop`, `take`, `slice`) and higher-order functions (`map`, `filter`, `reduce`) convert typed arrays to generic `MONK_ARRAY` before operating (using static `monk_typed_to_generic` helper). `monk_array_get` / `monk_array_set` handle typed arrays directly. `monk_is_array`, `monk_length`, `monk_type_name`, `monk_value_to_cstr` all updated for new kinds.

**CodeRabbit finding:** `ho_to_generic` in `higher_order.c` used raw `malloc` (bypassing OOM panic) and leaked the original typed-array backing store after conversion. Fixed: switched to `monk_malloc_internal` and added `free(v.int_array_val->data); free(v.int_array_val)` after each conversion.

**Benchmark results (Apple M4 Pro, cc -O3 -flto):**

| Benchmark | Inline access (prev) | Backing store (now) | vs C |
|---|---:|---:|---:|
| matmul (400×400 int) | ~30 ms | **~20 ms** | **~2× C** |

Progression: 100 ms (boxed) → 30 ms (inline access) → 20 ms (backing store). The remaining 2× gap vs C is bounds-check overhead per element access (2 comparisons per read/write). Closing to 1× C requires bounds-check elision for proven-safe loops — see `spec/ARCHITECTURE_DECISIONS.md §4C`.

**8 new correctness tests** in `TestBackingStore*` covering: int literal decl (checks for `monk_int_array_from`), range decl, element write (checks `int_array_val`, no `monk_array_set`), float[] array, for-in iteration, `show()` display, `is_array()`, `length()`.

Tests: 640 → 648 (485 Go + 163 C runtime). All 23 examples pass. All 9 benchmarks match expected. All linters clean.

### Use-after-free fix + memory leak fix (2026-04-06)

**Use-after-free in typed array conversion.** `monk_typed_to_generic` (container.c) and `ho_to_generic` (higher_order.c) had consuming semantics — they freed the typed backing store after boxing elements into a generic array. But the caller's `MonkValue` variable (passed by C value) still held the freed pointer. Any subsequent use of the same array variable (e.g. passing `nums` to both `map` and `filter`) was use-after-free. Crashed the `higher_order.monk` example. Fix: made both converters non-consuming — they copy but don't free the original.

**Memory leak in structural mutators.** The flip side: `monk_append`, `monk_prepend`, `monk_pop`, `monk_drop`, `monk_take`, `monk_slice`, `monk_map`, `monk_filter`, `monk_reduce` all called `monk_typed_to_generic` which allocates a fresh intermediate generic array. That intermediate was never freed after the operation completed — leaked on every call with a typed array input. Fix: track `was_typed` flag, free the intermediate via `free_generic_intermediate()` before returning.

**Deduplication.** `ho_to_generic` was a full copy of `monk_typed_to_generic`. Made `monk_typed_to_generic` non-static, declared it in `internal.h`, removed the duplicate from `higher_order.c`.

**Defensive fixes:**
- `monk_array_get` / `monk_array_set`: validate `index.kind == MONK_INT` before reading `index.int_val`
- `coerce()` in `unbox.go`: guard against array→scalar conversion (would be UB if triggered)

### Benchmark suite expansion (2026-04-06)

Expanded from 9 to 21 benchmarks to cover features the original suite was blind to. The old suite only tested scalar int/float arithmetic, int[] element access, and allocation stress — missing strings, records, closures, higher-order functions, for-in loops, and mixed workloads.

**12 new benchmarks (all with C reference + expected.txt):**

| Benchmark | What it tests | Monk vs C |
|-----------|-------------|-----------|
| `bitcount` | Bitwise ops (`&`, `>>`) in tight loop | 1.0x |
| `for_in_sum` | For-in over typed int[] array | 22x (boxing per element) |
| `sqrt_sum` | Math builtin (sqrt) in hot loop | 1.0x |
| `record_access` | Record field read/write | 25x (hash lookup per access) |
| `quicksort` | Array element swap, int arithmetic | 1.1x |
| `closure_invoke` | Closure creation + invocation per iteration | 20x (struct alloc per closure) |
| `string_ops` | to_upper/to_lower repeated on fixed string | 95x (alloc per call) |
| `string_concat` | String building in loop (O(n²) concat) | 11x |
| `functional_chain` | map/filter/reduce pipeline with closures | 6.5x |
| `levenshtein` | 2D array + substring per character | 52x (string alloc per char) |
| `nbody` | Float arrays + sqrt (gravitational sim) | 1.9x |
| `fannkuch` | Array permutation + reversal | 0.7x (faster than C!) |

**Full scoreboard (21 benchmarks, Apple M4 Pro, cc -O3 -flto):**

At C parity (10/21): fibonacci, leibniz, mandelbrot, collatz, ackermann, trial_primes, bitcount, quicksort, sqrt_sum, fannkuch.

2-3x C (3/21): matmul, sieve, nbody — all typed array bounds-check overhead.

6-95x C (8/21): for_in_sum, functional_chain, string_concat, closure_invoke, record_access, binary_trees, levenshtein, string_ops — fundamental runtime overhead in strings, records, closures, and value semantics.

**Root cause analysis:**

- **Records 25x C.** Every field access goes through hash-lookup-style dispatch. Zero unboxing. Records are the most common data structure after arrays.
- **String operations 95x C.** `to_upper_case` allocates a new string every call. UTF-8 indexing + allocation model makes string-heavy code extremely slow.
- **For-in over typed arrays 22x C.** Each element gets boxed from `int64_t` back to `MonkValue` for the loop body. Creates and destroys a `MonkValue` per element.
- **Closures 20x C.** Each iteration allocates a closure struct, captures variables, invokes through function pointer — vs C's direct call.
- **Levenshtein 52x C.** `substring(s, i, i+1)` allocates a new string for every character comparison. Character-level string access is fundamentally expensive.
- **fannkuch 0.7x C (faster!)** — suspicious, likely compiler generating luckier branch layout. Needs investigation.

**Optimization priority by broadest impact:**

| Optimization | Benchmarks helped | Complexity |
|---|---|---|
| Unboxed for-in over typed arrays | for_in_sum (22x→~1x), sieve, matmul | Low — emit raw loop, skip boxing |
| Record field unboxing | record_access (25x→~1x), nbody | Medium — track record layouts at compile time |
| Closure inlining (non-escaping) | closure_invoke (20x→~1x), functional_chain | Medium — escape analysis |
| String char access (return int codepoint) | levenshtein (52x→~5x) | Spec change needed |
| String operation fast-path (ASCII) | string_ops (95x→~5x) | Medium — fast memcpy path for ASCII |
| COW arrays | binary_trees (31x→~1x) | Medium — refcount backing store |

Tests: 648 → 648 (no new Go tests). All 23 examples pass. All 21 benchmarks match expected. All linters clean.

### Typed array index returns T, not T? (2026-04-06)

Array element reads on typed arrays (`int[]`, `float[]`, `bool[]`, `string[]`) now return the element type directly instead of wrapping in Optional. Typed arrays use strict OOB semantics (panic), so the result is always the element type — never `none`. Untyped arrays (element type `Any`) still return `Any?` for graceful reads.

This removes the `+ 0` workaround that was needed in quicksort, fannkuch, and other array-heavy code to unwrap `int?` to `int`. Type checker change only — `inferIndex` in `exprs.go` checks whether `objType.Elem.Kind != KindAny` to decide between strict (T) and graceful (T?) return.

### Unboxed for-in over typed arrays (2026-04-06)

For-in loops over `int[]`/`float[]`/`bool[]` now emit raw scalar loop variables. Previously, each element was boxed back into `MonkValue` via `monk_int()` — now the loop variable is `int64_t`/`double`/`bool` directly.

Generated C for `for x in arr` where `arr` is `int[]`:
```c
int64_t _len = (_iter.kind == MONK_INT_ARRAY) ? _iter.int_array_val->length : _iter.array_val->length;
for (int64_t _i = 0; _i < _len; _i++) {
    int64_t mk_x = (_iter.kind == MONK_INT_ARRAY) ? _iter.int_array_val->data[_i] : _iter.array_val->data[_i].int_val;
    // body operates on raw int64_t — += etc. use C arithmetic directly
}
```

Handles both `MONK_INT_ARRAY` (typed backing store) and `MONK_ARRAY` (generic) at runtime via a single branch per element. No conversion or allocation at loop setup. The loop body's arithmetic stays fully unboxed via the existing scalar assignment fast path.

Note: the `for_in_sum` benchmark (10M elements) didn't improve because its bottleneck is `range(10M)` allocation (80MB), not iteration overhead. Real programs iterating over existing arrays will benefit.

### What's next (compiler)

| Phase | Topic | Status |
|-------|-------|--------|
| 6 | Type System (static analysis) | **Complete** ✅ |
| 6 | Typed array unboxing (inline access) | **Complete** ✅ |
| 6 | Typed array backing store (`int64_t*`) | **Complete** ✅ — matmul ~2× C |
| 6 | Unboxed for-in over typed arrays | **Complete** ✅ — raw scalar loop variable |
| 6 | Typed array index returns T not T? | **Complete** ✅ — removes + 0 workaround |
| 6 | Bounds-check elision for typed arrays | Deferred — matmul/sieve/nbody to ~1× C |
| 6 | Record field unboxing | Deferred — record_access 25x→~1x |
| 6 | Copy-on-write for arrays | Deferred — binary_trees 31x→~1x |
| 6 | Closure inlining (non-escaping) | Deferred — closure_invoke 20x→~1x |
| 7 | Module System | Not started |
| 8 | C FFI | Not started |
| 9 | Linter & Formatter | Not started |
| 10 | LSP & Editor Support | Partial (extension built, LSP server not started) |
| 11 | Distribution (Homebrew, installers, zig cc bundling) | Partial (embedded runtime, make install) |

---

## The knowledge site

**Status: 26 pages live.** https://monkfromearth.github.io/monk-lang/

Astro + Solid.js + Tailwind. Auto-deploys to GitHub Pages on push to main.

### Compiler course ("Build Monk Lang")

Teaches how the compiler was built. Analogies first, then concepts, then build exercises.

| Unit | Topic | Pages |
|------|-------|-------|
| 0 | Foundations | 2 — How Compilers Work, Setting Up |
| 1 | Lexer | 4 — What Is a Lexer, Monk's Tokens, Building, References |
| 2 | Parser | 5 — What Is a Parser, Precedence, AST Design, Building, References |
| 3 | C Runtime | 6 — Why a Runtime, Values in Memory, Value Semantics, Builtins, Error Handling, References |
| 4 | Code Generation | 5 — What Is Codegen, Translation Rules, Function Hoisting, Building, References |
| 5 | CLI | 3 — Building the CLI, Compilation Pipeline, References |
| 6-11 | (future phases) | Blocked on compiler work |

### Language Guide ("Learn Monk")

User-facing tutorial for writing Monk, not building the compiler. Spec in `LANGUAGE_GUIDE_PROMPT.md`. **Not started.**

10 pages planned: first program, variables, functions, control flow, arrays, records, error handling, value semantics, builtins, guide index.

### Infrastructure

- Astro base path `/monk-lang/` — all links use `import.meta.env.BASE_URL`
- Deploy workflow: `.github/workflows/deploy-knowledge.yml` (Bun, triggers on `knowledge/` changes)
- Components: LessonLayout, LessonHeader, LessonNav, InsightBox, ExerciseBox, StepNumber, TagBadge, ChecklistItem
- Shiki theme: monk-dark (`#1C1917`). No emojis. No purple.

### Decisions

- **Unit 3 is the C Runtime, not an interpreter.** The interpreter was built and deleted. Course skips it — parser straight to C runtime + codegen.
- **No actual code from the repo.** Pseudocode and analogies, referencing symbols and approaches.
- **One page per concept, references page per phase.** External resources collected at end of each unit.
- **JSX brace escaping.** Inline `<code>` with `{`/`}` must use `&#123;`/`&#125;`. `<Code code={...}>` handles this automatically.
