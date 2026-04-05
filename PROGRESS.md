# Monk Lang — Progress

What's been built, what pivots happened, what's next.

## Current release

**0.0.1 — Buniyaad** (2026-04-04). First local release. Self-contained binary, embedded runtime, works from any directory.

## The compiler

**Status: Phases 1-6 complete. 556 tests (405 Go + 151 C runtime). Working end-to-end.**

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

### What's next (compiler)

| Phase | Topic | Status |
|-------|-------|--------|
| 6 | Type System (static analysis) | **Complete** ✅ |
| 6.5 | Type-informed codegen (unboxed ints/floats/arrays) | Not started |
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
