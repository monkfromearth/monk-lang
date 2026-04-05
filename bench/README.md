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
| `matmul` | 400×400 naive int matmul | Array index hot path |

Each benchmark:
- Prints a single integer checksum (avoids float-formatting mismatches across languages)
- Uses the same algorithm in every language (no SIMD, no numpy, no smart tricks)
- Is compiled/run with stock flags: `cc -O2`, `go build`, `python3`, `node`, `bun`

Monk compiles to C, which is then compiled with `cc -O2` — so Monk inherits whatever optimizations cc applies to the generated code.

## Interpreting the numbers

**Monk's design tradeoff:** pure value semantics (deep-copy on assignment, every read/write through tagged-union dispatch). This shows up in the numbers.

**Current standings** (after the 2026-04-05 codegen performance pass — see `spec/PERFORMANCE.md`):

- **Mandelbrot — 1.0× C.** Tight float loops with no arrays inline fully. At parity with C, beats Go/Bun/Node.
- **Fibonacci — 1.6× C.** Pure recursion. Beats Bun/Node/Python, close to Go.
- **Matmul — 14× C.** Hot array indexing. Remaining gap is the value-semantics tagged array. Needs Phase 6 type inference + unboxed int arrays to close.

**The honest story:** Monk is compute-competitive with Go and sometimes C. The allocation-heavy / array-heavy cases are its weakness until the type system lands. If you need matmul performance today, write it in C and FFI-call it (Phase 8).

## Known limitations

- **hyperfine variance** — re-run on a quiet machine for tight numbers.
- **Startup time dominates Fibonacci** — each process's startup is ~10-80ms depending on runtime. At fib(35), C takes ~17ms total including startup. Bigger N would separate the signal from the noise, but fib(38)+ can stack-overflow Python or slow the suite to a crawl.
- **Python is a floor, not a ceiling** — CPython 3.14 on `matmul` takes 7.7s. Don't read this as "Monk is 10× faster than Python" — Monk 90× slower than C and Python 760× slower than C both means "don't do numerics here."

## Adding a benchmark

1. Create `bench/benchmarks/<name>/` with `<name>.monk`, `<name>.c` minimum.
2. Add `expected.txt` with the canonical single-int output.
3. Add the name to `BENCHMARKS` in `bench/run.sh`.
4. Optional: add `<name>.go`, `<name>.py`, `<name>.js` for other languages.

See `bench/PLAN.md` for design notes and v2 ideas.
