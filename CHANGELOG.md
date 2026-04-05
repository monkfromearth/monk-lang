# Changelog

All notable changes to Monk Lang are documented here.

Format: [Semantic Versioning](https://semver.org/). Each minor version gets an Urdu/Hindi codename.

---

## 0.0.1 — Buniyaad (2026-04-04)

The foundation. Monk compiles, runs, and produces native binaries.

### What works

**Compiler pipeline:** Monk source → C → native binary. No interpreter, no VM.

**CLI commands:**
- `monk build <file.monk>` — compile to native binary
- `monk build <file.monk> -o <out.c>` — emit generated C source
- `monk run <file.monk>` — compile and run in one step
- `monk check <file.monk>` — parse and validate without compiling
- `monk version` — prints version and codename

**Language features:**
- All value types: int, float, string, boolean, none
- Arrays (homogeneous) and records (fixed shape)
- Functions with closures (capture by copy)
- Control flow: if/else/else-if, while, for-in, break, continue
- Guard/against/throw error handling (setjmp/longjmp)
- Operator precedence (13 levels)
- Const (deep freeze) and let (fully mutable)
- Value semantics everywhere — assignment copies, function args copy, closures capture by copy

**Runtime builtins (40+):**
- Output: show, to_string
- String: length, substring, index_of, split, trim, to_upper_case, to_lower_case
- Array: append, prepend, pop, drop, take, slice, range
- Math: abs, floor, ceil, round, sqrt, pow, log, sin, cos, tan, min, max
- Type: typeof, is_number, is_string, is_boolean, is_array, is_record, is_function, is_none
- Conversion: to_int, to_float
- File system: file_read, file_write, file_exists
- Environment: env_get, exit, args

**Self-contained binary:** Runtime is embedded via go:embed. The `monk` binary works from any directory without needing the source tree.

**Test suite:** 540 tests (195 syntax, 110 types, 45 codegen, 39 CLI, 151 C runtime).

### Hardening (2026-04-05)

Post-release review caught several bugs and gaps:

- `monk build foo.monk -o` (dangling flag) now errors instead of silently ignoring
- `monk build foo.monk -o build/bin/app` auto-creates parent directories
- Runtime cache no longer rewrites files on every build (prevents concurrent-build races)
- Homebrew symlinks resolved before runtime search (Phase 11 readiness)
- `#line` directives escape filenames so Windows-style paths produce valid C
- `monk run` cleans up `/tmp/monk-run-*` even when child program exits non-zero
- Removed `.vsix` build artifact from git

### Performance (2026-04-05)

First benchmark numbers + two generic optimizations:

- `cc -O3 -flto` on generated C (was `-O2`) — enables cross-TU inlining
- Inline fast-paths for `monk_deep_copy` / `monk_free` on primitive values

Results: mandelbrot now at parity with C. Fibonacci 1.6× C. Matmul 14× C (the weak spot — needs type system for unboxed arrays).

See `spec/PERFORMANCE.md` and `bench/` for methodology.

### Phase 6 — Type System (2026-04-05)

Static type checker. Runs after parse, before codegen — wired into `monk build`, `monk run`, `monk check`.

Catches at compile time:
- First-assignment inference: `let x = 42` → x locked to int
- Reassignment drift: `x = "hi"` on an int errors
- Array element violations (literals AND index assignment)
- Typed-record shape (missing/extra/wrong-type fields)
- Cross-type equality: `5 == "5"` is a type error
- Collection equality: `arr == arr` rejected (no deep-compare in the language)
- Loop-variable const violations
- Missing returns on non-none functions
- Function call arity and per-argument types
- Function-type parameters: `(f (int) -> int, x int)`

Two paper-cut fixes along the way:
- Parser bug on `(fn_call(x) / y)` — misread as function literal with
  function-type parameter. Now checks for `->` before committing.
- `to_int` / `to_float` now accept int/float/string (was string-only). Matches
  the spec's "explicit coercion" philosophy.

110 checker tests. Codegen consumes the checker's type Info to emit raw C
scalars (int64_t, double, bool) for statically-typed scalar variables, raw
arithmetic between scalar operands, unboxed function signatures when all
params and return are scalar, and raw conditions in if/while. The string-
concat overload on `+=` dispatches through monk_string_concat to match
binary `+`.

**Benchmark impact (vs C, lower is better):**
- fibonacci: 1.6× → **1.0× C** (parity)
- mandelbrot: 1.2× → **1.0× C** (parity)
- leibniz (new): **1.0× C**
- matmul: 14× → 12× C (arrays still tagged, typed-array unboxing is future work)

4 new runnable examples demonstrating each check in action:
- `examples/types.monk` — inference, annotations, arrays, records, optionals
- `examples/records.monk` — nested records with structural typing
- `examples/optionals.monk` — T? and the none case
- `examples/guards.monk` — guard/against/throw scoping

### What's not here yet

- Module system (use/export) — Phase 7
- C FFI — Phase 8
- Linter and formatter — Phase 9
- LSP and editor support — Phase 10
- Distribution (Homebrew, install script, CI) — Phase 11
- `ref` parameters — deferred
- Closures as first-class values passed to other functions — deferred
- Default parameter values — deferred

### Requirements

- Go 1.26.1+ (to build the compiler)
- A C compiler (cc/gcc/clang) on the system
- macOS or Linux

### Build from source

```bash
cd projects/monk-lang
make install   # builds and copies to ~/.local/bin/monk
```
