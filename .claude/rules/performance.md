# Performance

Monk compiles to C with `-O3 -flto`. The compiler gets to be fast and
so does the generated code. These are two different concerns — measure
and optimize them separately.

---

## Two separate performance surfaces

| Surface | What it is | How to measure |
|---------|-----------|---------------|
| **Compiler throughput** | How fast does `monk build` turn source into a binary? | `hyperfine "monk build bench/benchmarks/matmul/matmul.monk"` |
| **Runtime performance** | How fast does the generated binary run? | Compile first, then `hyperfine ./matmul` |

Don't mix them. `monk run` measures both at once — useless for either.

---

## Measurement rules (learned the hard way)

1. **Compile the binary first, then time it.** `monk run` has ~300ms
   compile overhead that dwarfs actual runtime on fast programs.
   ```bash
   ./monk build bench/benchmarks/fibonacci/fibonacci.monk -o /tmp/fib_test
   /usr/bin/time -p /tmp/fib_test
   ```

2. **Use `/usr/bin/time -p`, not bash's `time` builtin.** The builtin
   measures wall time only. `/usr/bin/time -p` gives wall, user, and sys.

3. **Use `hyperfine` for proper benchmarking.** It runs multiple
   iterations, warms up, and reports median + stddev. The `bench/run.sh`
   script does this automatically against C/Go/Python/Node/Bun.

4. **Benchmark on a quiet machine.** Background processes add variance.
   Don't benchmark while compiling something else. `hyperfine` handles
   statistical noise — still use it on a reasonably idle machine.

5. **Use `break` and `continue` explicitly in benchmark `.monk` code.**
   Setting a loop variable to a sentinel value (e.g. `d = n` to exit)
   runs extra iterations and skews timing by 3-5×.

6. **Report: before → after → vs C.** Both absolute and relative.
   "Before: 142ms. After: 21ms. 6.7× speedup. Now 2.1× C." gives the
   full picture. Never report just "it's faster."

---

## When to optimize

**Optimize when:**
- A benchmark is measurably slow AND the cause is confirmed by profiling
- A generic win is available (benefits all programs, not just one benchmark)
- It's a pure implementation improvement — no new language features, no
  spec changes, just better code generation or runtime logic

**Don't optimize when:**
- The cause isn't measured yet (guess-and-check wastes time)
- Closing the gap requires a phase-level change (typed-array unboxing is
  a type-system feature, not a performance tweak — it belongs in a phase)
- The overhead is fundamental to the design model (matmul at 12× C is
  the honest cost of value semantics; "fix" it by redesigning, not hacking)
- The benchmark isn't in `bench/benchmarks/` — add it first

---

## The optimization workflow

1. **Establish a baseline.** Run `bench/run.sh` before any change.
   Record the numbers in PROGRESS.md.

2. **Identify the hot path.** Don't guess. Use profiling:
   ```bash
   ./monk build foo.monk -o /tmp/foo
   # macOS:
   xcrun xctrace record --template "Time Profiler" --launch -- /tmp/foo
   # Linux:
   perf record /tmp/foo && perf report
   ```

3. **Make one change at a time.** If you change both the runtime and
   the codegen in the same session, you can't attribute the gain.

4. **Run ALL benchmarks after, not just the target.** An optimization
   that speeds up fibonacci but slows down matmul is a regression.
   ```bash
   make build && bench/run.sh
   ```

5. **Record results in PROGRESS.md.** What changed, methodology, before/after
   table per benchmark, what's still slow and why.

---

## Optimization tiers (in priority order)

1. **Eliminate dispatch.** Tagged-union operations for statically-typed
   values are pure overhead. If the type checker knows a variable is
   `int`, emit `int64_t`, not `MonkValue`. This is the scalar unboxing
   model already in place — extend it before anything else.

2. **Inline hot runtime functions.** `monk_deep_copy` and `monk_free`
   on primitive values are no-ops — but they were real function calls.
   Header-inlined wrappers with primitive short-circuits win without
   changing semantics.

3. **cc flags.** `-O3 -flto` is already in use. Don't add `-march=native`
   — it breaks cross-compile. `-fprofile-generate`/`-fprofile-use` (PGO)
   is worth exploring for Phase 9+.

4. **Algorithmic.** Rare in a compiler of this size, but if a pass is
   O(n²) where O(n) is possible, fix the algorithm before tuning constants.

---

## Honest performance communication

When documenting benchmarks, always state:
- The hardware (Apple M4 Pro, etc.)
- The date (performance changes across phases)
- The comparison baseline (C with `-O3 -flto`)
- What the remaining gap IS and WHY it exists

"We're 12× C on matmul. The gap is value semantics — every array element
read/write goes through a tagged union. This is intentional. Typed-array
unboxing (Phase 6.5) closes it to ~2×." is honest.

"We're still slower than C on arrays" tells the reader nothing.
