# Monk Lang Benchmarks

Small benchmark suite comparing Monk against C, Go, Python, Node, and Bun.

## Running

```bash
./bench/run.sh               # all benchmarks, all available languages
./bench/run.sh fibonacci     # single benchmark
```

**Requirements:** `hyperfine`, `jq`, `cc`, and the `./monk` binary (run `make build`). Go / Python / Node / Bun are auto-detected and skipped if missing.

Results are written to `bench/results/{timestamp}-results.md`. Raw hyperfine JSON goes to `bench/results/{timestamp}-raw/`.

## Benchmarks

| Name | Params | What it measures |
|------|--------|------------------|
| `fibonacci` | `fib(35)` | Recursive call overhead + int arithmetic |
| `mandelbrot` | 800×800, 50 iter | Float compute in a tight loop |
| `leibniz` | 50M iter | Float arithmetic + loop + conditional |
| `trial_primes` | n < 200,000 | Nested int loops with `break` |
| `matmul` | 400×400 naive int matmul | Array index hot path |
| `sieve` | Primes up to 1M | Array alloc + index-write in nested loops |
| `ackermann` | A(3, 11) | Deep recursion stress test |
| `collatz` | Longest chain, n ≤ 1M | While-loop + conditional branching |
| `binary_trees` | Depth 14, pool-based | Array allocation + iteration stress test |

Each benchmark:
- Prints a single integer checksum (avoids float-formatting mismatches across languages)
- Uses the same algorithm in every language (no SIMD, no numpy, no smart tricks)
- Is compiled/run with stock flags: `cc -O3 -flto`, `go build`, `python3`, `node`, `bun`

Monk compiles to C, which is then compiled with `cc -O3 -flto` — so Monk inherits whatever optimizations cc applies to the generated code.

## Interpreting the numbers

**Monk's design tradeoff:** pure value semantics (deep-copy on assignment, every read/write through tagged-union dispatch). Phase 6's scalar unboxing codegen uses type-checker output to skip this dispatch entirely when a variable is statically scalar (`int`/`float`/`bool`). Most arrays still use the tagged model; typed arrays have their own backing stores.

**Note:** the table below is the 2026-04-05 baseline. Later typed-array backing-store and bounds-check-elision work narrowed array-heavy workloads further, so treat `matmul` here as historical comparison rather than the current ceiling.

**Current standings** (post-scalar-unboxing):

- **fibonacci — 1.0× C.** Recursive `int` function, fully unboxed.
- **mandelbrot — 1.0× C.** Tight float loop, no arrays. Unboxed end-to-end.
- **leibniz — 1.0× C.** π approximation via float arithmetic.
- **trial_primes — 1.0× C.** Nested int loops with `break`, counts primes.
- **matmul — baseline ~12× C in the 2026-04-05 run.** Hot array indexing was the bottleneck in that snapshot; later typed-array work improved this substantially.
- **sieve — array-heavy.** Allocates 1M-element array, writes in nested loops. Measures tagged-array index-write overhead.
- **ackermann — deep recursion.** A(3,11) = 16381 via millions of recursive calls. Pure function-call overhead.
- **collatz — while-loop branching.** Longest Collatz chain for n ≤ 1M. Int arithmetic + conditionals in a tight loop.
- **binary_trees — allocation stress.** Pool-based binary tree build + walk at depth 14. Array alloc, index write/read, iteration.

**The honest story:** Monk matches C on every benchmark that doesn't use arrays, and the typed-array work has already narrowed the array-heavy gap a lot. For the widest numerics workloads, Phase 8 FFI is still the escape hatch.

## Known limitations

- **hyperfine variance** — re-run on a quiet machine for tight numbers.
- **Startup time dominates short runs** — each process has ~10-80ms startup depending on runtime. Benchmarks are tuned so C work takes ~5-30ms, keeping startup a fraction of total.
- **Python is a floor, not a ceiling** — CPython on `matmul` takes ~8s. Don't read this as "Monk is 60× faster than Python" — Monk 12× slower than C and Python 760× slower than C both mean "don't do numerics here."

## Adding a benchmark

1. Create `bench/benchmarks/<name>/` with `<name>.monk`, `<name>.c` minimum.
2. Add `expected.txt` with the canonical single-int output.
3. Add the name to `BENCHMARKS` in `bench/run.sh`.
4. Optional: add `<name>.go`, `<name>.py`, `<name>.js` for other languages.

See `bench/PLAN.md` for design notes and v2 ideas.
