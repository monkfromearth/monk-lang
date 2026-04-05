# Monk Lang Performance

How Monk performs, why, and what can make it faster.

---

## Current numbers (2026-04-05, Apple M4 Pro)

| Benchmark | Monk | vs C | vs Go | vs Bun | vs Node | vs Python |
|---|---:|---:|---:|---:|---:|---:|
| fibonacci (n=35) | 27.5 ms | 1.6× | 1.3× faster | 1.5× faster | 3.0× faster | 24× faster |
| mandelbrot (800² × 50) | 17.0 ms | **1.0×** | **1.2× faster** | **1.6× faster** | **3.1× faster** | **148× faster** |
| matmul (400² int) | 142 ms | 13.9× | 5.5× slower | 2.7× slower | 1.5× slower | 54× faster |

Mandelbrot hits C parity. Fibonacci beats every language except C. Matmul is the weak spot.

See `bench/` for methodology and `bench/results/` for raw data.

---

## Why the differences

**Monk's hot path per operation:** tagged-union dispatch. Every `+`, `*`, `[i]` goes through a `MonkValue` struct (16 bytes: 8-byte union + 4-byte kind tag + padding), with a runtime kind check to decide between int/float/string semantics. That dispatch is the cost.

**Why mandelbrot hits C parity.** Pure float compute in a tight loop, no arrays in the hot path. After LTO, every `monk_add` / `monk_mul` inlines into `cc`-native double arithmetic. The kind check compiles to one `cmp` that branch-predicts perfectly. The loop vectorizes the same way C does.

**Why fibonacci is 1.6× C.** Recursive calls pass two `MonkValue`s (32 bytes) on the stack instead of one `long` (8 bytes). Four extra memory ops per call × 30M calls = 120M extra loads/stores.

**Why matmul is 14× C.** The inner loop does:
```
C[i*N+j] = C[i*N+j] + aik * B[k*N+j]
```
Each `arr[idx]` access: (1) kind check for `arr`, (2) kind check for `idx`, (3) bounds check, (4) pointer dereference, (5) 16-byte load. That's ~10 instructions per access vs 2 in C. C also auto-vectorizes the inner loop with NEON; Monk's tagged elements can't be vectorized.

---

## Optimizations applied (2026-04-05)

### `-O3 -flto` on generated C

Upgraded from `-O2` to `-O3 -flto` in `monk build` / `monk run`. LTO lets `cc` inline across translation units, so `monk_add`/`monk_mul`/`monk_array_get` all fold into the caller.

**Impact:** matmul 911ms → 379ms (~2.4×). Mandelbrot 242ms → ~80ms.

### Inline fast-paths for `monk_deep_copy` and `monk_free`

The old codepath: `monk_array_set` calls `monk_free(old_element)` + `monk_deep_copy(new_element)` on every write. For int elements both are no-ops, but they were real function calls with 16-byte argument passing.

Fix: split into `monk_deep_copy` / `monk_free` (static inline, in `runtime.h`, primitive fast path) and `monk_deep_copy_heap` / `monk_free_heap` (in `runtime.c`, full logic). For primitive values the call disappears entirely — just a kind-check branch inline.

**Impact:** matmul 379ms → 142ms (another ~2.7×). Mandelbrot down to 17ms (parity with C).

### Combined improvement

| Benchmark | Before | After | Speedup |
|---|---:|---:|---:|
| fibonacci | 110 ms | 27.5 ms | 4.0× |
| mandelbrot | 242 ms | 17.0 ms | 14.2× |
| matmul | 912 ms | 142 ms | 6.4× |

---

## What's next (the honest ceiling)

The remaining 14× gap on matmul is **fundamental to the value-semantics design** as currently specified. Closing it requires changes that need the type system (Phase 6).

### Phase 6-dependent wins

**Static type inference in codegen.** When Monk can prove a variable is `int` at compile time, emit raw `int64_t` C ops instead of `MonkValue` through `monk_add`. This removes tagged-union dispatch entirely from hot loops. **Expected: fib, float compute near 1.0× C. Matmul to ~3-5× C.**

**Unboxed arrays.** `let arr int[] = ...` should compile to `int64_t*`, not `MonkValue*`. Removes the tagged-union cost per element. Enables `cc` auto-vectorization of the inner loop. **Expected: matmul to 1.5-2× C.**

These two together would put Monk's numerics within the Go/Rust range.

### Phase 8+ wins

**FFI to C numerics.** For matrix ops, image processing, crypto, users should reach for BLAS/OpenBLAS/LAPACK via C FFI instead of reimplementing in Monk. That's Phase 8 and the C ecosystem's job, not the language's.

---

## What we will NOT do

Some optimizations tempting but poisonous:

- **Mutable references (`ref`/pointers) in the core language.** Breaks value semantics. The whole point of Monk is predictable data ownership.
- **Garbage collection.** Same reason. No refcount overhead is part of the model.
- **Cheating per-benchmark.** No fast paths specialized for loops that look like matmul. Generic wins only.
- **`-ffast-math` or similar unsafe-math flags.** Changes observable float behavior.
- **`-march=native`.** Benchmarks should reflect what users will see from a standard `brew install monk` build.

---

## Measuring performance yourself

```bash
./bench/run.sh                 # all benchmarks
./bench/run.sh matmul          # one benchmark
```

Results land in `bench/results/{timestamp}-results.md`. See `bench/README.md` for methodology and `bench/PLAN.md` for design notes.
