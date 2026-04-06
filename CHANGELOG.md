# Changelog

All notable changes to Monk Lang are documented here.

Format: [Semantic Versioning](https://semver.org/). Each minor version gets an Urdu/Hindi codename.

---

## Unreleased

### Performance
- **Typed array backing store** — `int[]`, `float[]`, `bool[]` variables now use `int64_t*` / `double*` / `bool*` backing stores instead of `MonkValue*`. Element reads/writes emit `arr.int_array_val->data[i]` — direct pointer access, no union overhead, half the memory stride. Matmul benchmark: ~100ms → ~30ms (inline) → **~20ms**, **11× C → 3× C → ~2× C**.
- New `MONK_INT_ARRAY` / `MONK_FLOAT_ARRAY` / `MONK_BOOL_ARRAY` value kinds in `runtime.h`. New structs `MonkIntArray { int64_t *data; int64_t length }` etc. Converter functions `monk_int_array_from()` / `monk_float_array_from()` / `monk_bool_array_from()` — convert from generic `MONK_ARRAY` (consuming it) or deep-copy from same typed kind.
- All generic runtime functions (`is_array`, `length`, `typeof`, structural mutators, `map`/`filter`/`reduce`) updated to accept and correctly handle typed array kinds.
- **Typed array inline access** (previous session) — `storeIntArray` / `storeFloatArray` / `storeBoolArray` storage kinds in `unbox.go`. `deriveFuncStorage` uses `isRawScalar` so array parameters don't incorrectly participate in the all-scalar fast path. OOB panics (strict). `T?`-annotated variables preserve graceful path.

### Language
- **Typed array index returns `T`, not `T?`** — array element reads on typed arrays (`int[]`, `float[]`, `bool[]`, `string[]`) now return the element type directly. OOB panics (strict), so the result is always the element type. Removes the `+ 0` workaround in user code. Untyped arrays (element type `Any`) still return `Any?`.

### Performance (continued)
- **Unboxed for-in over typed arrays** — for-in loops over `int[]`/`float[]`/`bool[]` now emit raw scalar loop variables (`int64_t`/`double`/`bool`) instead of boxing each element into `MonkValue`. The loop body operates on raw C types — no `monk_int()`/`monk_free()` per element.

### Bug Fixes
- **Use-after-free in typed array conversion** — `monk_typed_to_generic` and `ho_to_generic` freed the typed backing store after conversion (consuming semantics), but the caller's variable still held the freed pointer. Any array passed to multiple builtins (e.g. `map` then `filter`) crashed. Fixed: converters are now non-consuming.
- **Memory leak in structural mutators** — `append`, `prepend`, `pop`, `drop`, `take`, `slice`, `map`, `filter`, `reduce` all leaked the intermediate generic array allocated by `monk_typed_to_generic`. Fixed: `free_generic_intermediate()` called before returning.
- **Index type validation** — `monk_array_get` / `monk_array_set` now validate `index.kind == MONK_INT` before reading the union field, preventing undefined behavior on non-int index values.
- **Coerce guard** — `coerce()` in `unbox.go` now guards against array→scalar conversion (would read wrong union member).

### Benchmarks
- **12 new benchmarks** (9 → 21 total): `bitcount`, `for_in_sum`, `sqrt_sum`, `record_access`, `quicksort`, `closure_invoke`, `string_ops`, `string_concat`, `functional_chain`, `levenshtein`, `nbody`, `fannkuch`. Each with C reference implementation and expected.txt.
- Reveals previously hidden performance gaps: records 25× C, string ops 95× C, closures 20× C, for-in 22× C.

### Tests
- 9 typed-array correctness tests (`TestTypedArray*`): reads, writes, arithmetic chains, mini matmul, float arrays, OOB panic.
- 8 backing-store correctness tests (`TestBackingStore*`): int/float literal decls, range decls, element writes, for-in iteration, `show()`, `is_array()`, `length()`.

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

**Test suite:** 560 tests (195 syntax, 112 types, 63 codegen, 39 CLI, 151 C runtime).

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
