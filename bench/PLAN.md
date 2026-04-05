# Monk Benchmark Suite — Plan

## Design decisions

**Why these 3 benchmarks.** Fibonacci, Mandelbrot, and Matrix Multiply are classic "language shootout" problems. They cover three different performance profiles:

- **Fibonacci (recursion-heavy)** — stresses call dispatch, stack frames, int arithmetic. No allocation.
- **Mandelbrot (float-heavy)** — tight inner loop, float math, no allocation.
- **Matmul (array-index-heavy)** — hot-path random access into flat arrays. Exposes tagged-union dispatch cost per index.

**Why checksum output.** Every benchmark prints a single integer. This sidesteps floating-point format differences between languages (`%.9f` in C ≠ Python's `str(float)` ≠ Monk's `to_string(float)`). The checksum doubles as a correctness gate — the harness diffs against `expected.txt` before running hyperfine.

**Why integer matmul.** Float matmul would add float-formatting noise to the checksum. Using `long`/`int64` keeps checksums exact across languages.

**Deliberately excluded from v1:**
- **Binary trees** — would be the "honest weakness" showcase for Monk's value-semantics allocation cost. Adding in v2 is important for intellectual honesty.
- **N-body / spectral-norm** — good benchmarks, but overlap with Mandelbrot for the compute story.

## Current status (2026-04-05)

- 3 benchmarks × 6 languages = 18 implementations
- Harness runs end-to-end with correctness gate
- First numbers captured on Apple M4 Pro

## Known issues surfaced by building this suite

**Monk parser bug.** `(to_float(y) / 2.0)` fails with "expected parameter name" — the parser treats any `(` followed by an identifier as the start of a function literal. Workaround: extract intermediate variables. Fix is in Phase 2 parser — should require `{` lookahead before committing to function-literal parse. Low priority but user-visible.

**Monk has no int→float builtin.** `to_float(x)` only accepts strings. Workaround: `x * 1.0` coerces via the mixed-type arithmetic rules. A real `to_float(int)` or `float(x)` builtin would be cleaner.

## v2 ideas

- **Peak-RSS measurement** — `/usr/bin/time -l` on macOS gives peak resident set size. Add a second results table.
- **`binary-trees` benchmark** at depth 14 — the allocation-heavy case. Expected: Monk slower than Python here. Include honestly.
- **CI integration** — `.github/workflows/bench.yml` on every push to a `bench` branch, posting results as PR comments.
- **Chart generation** — gnuplot or matplotlib bar charts showing log-scale ratios.
- **More languages** — Zig, Rust, OCaml, Crystal. Comparable small-compiled-language cohort.
- **Separate compile-time vs runtime tracking** — right now we only measure runtime. Monk's compilation time (source → binary) is relevant for dev iteration.

## Harness invariants

- Each language implementation uses the same algorithm. See each `.monk` / `.c` / `.go` / `.py` / `.js` — if they diverge, that's a bug, not an optimization.
- No language may use SIMD intrinsics, `numpy`, `Float64Array`, or similar. Plain arrays / lists only.
- `cc -O2` is the baseline. No `-march=native`, no `-ffast-math`, no PGO.
- Correctness gate blocks benchmarking if any output diffs from `expected.txt`.

## Non-goals

- Declaring Monk "faster than Python" or "competitive with C". These benchmarks surface *tradeoffs*, not marketing points.
- Tuning benchmarks to favor Monk. The algorithms are idiomatic-to-ugly in every language equally.
- Micro-benchmarks (sub-millisecond). Each benchmark is tuned to run 10ms+ in C so hyperfine's startup-noise floor doesn't dominate.
