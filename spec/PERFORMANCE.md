# Monk Lang Performance

How Monk performs, why, and what can make it faster.

---

## Current numbers (2026-04-05, Apple M4 Pro, post-scalar-unboxing)

This table is the last fully tabulated benchmark run. Later typed-array work in the changelog narrowed array-heavy workloads further, so treat `matmul` here as the 2026-04-05 baseline rather than the current ceiling.

| Benchmark | Monk | vs C | vs Go | vs Bun | vs Node | vs Python |
|---|---:|---:|---:|---:|---:|---:|
| fibonacci (n=35) | 17.3 ms | **1.0×** | **1.3× faster** | **2.4× faster** | **4.9× faster** | **39× faster** |
| mandelbrot (800² × 50) | 14.5 ms | **1.0×** | **1.1× faster** | **1.9× faster** | **3.8× faster** | **180× faster** |
| leibniz (π, 50M iter) | 29.2 ms | **1.0×** | **1.2× faster** | **1.4× faster** | **2.5× faster** | **174× faster** |
| trial_primes (<200k) | 5.9 ms | **1.0×** | **1.0×** | **2.2× faster** | **7.8× faster** | **92× faster** |
| matmul (400² int) | 122 ms | 11.8× | 4.6× | 2.3× | 1.2× | 64× faster |

**Pure scalar benchmarks are at C parity.** Matmul was still the array hot path in this baseline, but later typed-array backing-store and bounds-check-elision work pushed it much closer to C. The remaining big gaps are semantic ones: string-heavy workloads, closure-heavy workloads, and array-heavy numerics that are better served by Phase 8 FFI.

See `bench/` for methodology and `bench/results/` for raw data.

---

## Why the differences

**Monk's hot path per operation:** tagged-union dispatch. Every `+`, `*`, `[i]` goes through a `MonkValue` struct (16 bytes: 8-byte union + 4-byte kind tag + padding), with a runtime kind check to decide between int/float/string semantics. That dispatch is the cost.

**Why fibonacci / mandelbrot / leibniz hit C parity.** Scalar unboxing (see below) emits raw `int64_t` and `double` everywhere — no MonkValue wrapping, no tag checks, no function calls for arithmetic. The generated C is indistinguishable from a hand-written scalar loop.

**Why matmul was ~12× C in the 2026-04-05 baseline.** The inner loop does:
```
C[i*N+j] = C[i*N+j] + aik * B[k*N+j]
```
Each `arr[idx]` access: (1) kind check for `arr`, (2) kind check for `idx`, (3) bounds check, (4) pointer dereference, (5) 16-byte load. That's ~10 instructions per access vs 2 in C. C also auto-vectorizes the inner loop with NEON; Monk's tagged elements can't be vectorized.

Later work replaced the generic tagged path for typed arrays, then added bounds-check elision for provably-safe loops, which is why the changelog now reports matmul much closer to C.

---

## Optimizations applied (2026-04-05)

### `-O3 -flto` on generated C

Upgraded from `-O2` to `-O3 -flto` in `monk build` / `monk run`. LTO lets `cc` inline across translation units, so `monk_add`/`monk_mul`/`monk_array_get` all fold into the caller.

**Impact:** matmul 911ms → 379ms (~2.4×). Mandelbrot 242ms → ~80ms.

### Scalar unboxing (Phase 6)

When the type checker proves a variable's type is `int`, `float`, or `boolean`, codegen stores it as a raw `int64_t`/`double`/`bool` instead of a tagged `MonkValue`. Arithmetic between scalar operands emits raw C (`a + b * c`), function signatures become unboxed (`static int64_t fib(int64_t n)`), and `if`/`while` conditions skip `monk_is_truthy` when they produce a typed boolean.

**Impact:** fib 28ms → 17ms. Mandelbrot 17ms → 14.5ms. Both now at C parity.

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

Scalar unboxing is done. In the 2026-04-05 baseline, matmul was still at 12× C because **arrays were still tagged-union** — each `arr[i]` did bounds-check + tag dispatch + 16-byte memcpy.

### Typed-array path (already delivered)

The typed-array backing store work is already in place. `typeof` / `is_*` inlining for known kinds is already in place too. The next big wins are broader semantic optimizations and Phase 8 FFI for array-heavy numerics.

### Unboxed for-loop variables

`for i in range(N)` is already compiled as a raw counter loop when the iterable is statically known, so the loop variable stays unboxed in hot typed-array paths. The remaining loop-heavy wins come from reducing allocation and string work, not from reintroducing scalar boxing.

### Phase 8+ wins

**FFI to C numerics.** For matrix ops, image processing, crypto, users should reach for BLAS/OpenBLAS/LAPACK via C FFI instead of reimplementing in Monk. That's Phase 8 and the C ecosystem's job, not the language's.

---

## What we will NOT do

Some optimizations tempting but poisonous:

- **Hidden aliasing or implicit pointer semantics.** If Monk has references, they should stay explicit in source via `ref`, not appear as invisible optimizer magic.
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
