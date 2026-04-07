<div align="center">

<br/>

# Monk Lang

**Performance** of compiled languages. **Simplicity** of Python. **Memory safety** inspired by Rust.

A minimalist, readable, and performant programming language for the modern age.

[![Build](https://img.shields.io/badge/build-passing-brightgreen?style=flat)](#status)
[![Tests](https://img.shields.io/badge/tests-620_passing-brightgreen?style=flat)](#status)
[![Phase](https://img.shields.io/badge/phase-6_of_11-blue?style=flat)](#status)
[![Go](https://img.shields.io/badge/Go-1.26.1+-00ADD8?style=flat&logo=go&logoColor=white)](#install)
[![License](https://img.shields.io/badge/license-MIT-green?style=flat)](#license)

</div>

<br/>

```javascript
let fibonacci = (n int) int {
    if n <= 1 { return n }
    return fibonacci(n - 1) + fibonacci(n - 2)
}

for i in range(10) {
    show(to_string(fibonacci(i)))
}
```

<br/>

## Quick Start

```bash
# Build a native binary
monk build hello.monk

# Compile and run in one step
monk run hello.monk

# Validate without compiling
monk check hello.monk

# Emit C source instead of binary
monk build hello.monk -o hello.c
```

## Install

Requires Go 1.26.1+ and a C compiler (cc/gcc/clang).

```bash
git clone https://github.com/monkfromearth/monk-lang.git
cd monk-lang
make install    # builds and installs to ~/.local/bin/monk
```

## The Language

### Variables

```javascript
let name = "Monk"          // mutable
const pi = 3.14159         // immutable (deep freeze)
```

### Types

`int` (64-bit) · `float` (64-bit) · `string` (UTF-8) · `boolean` · `none` · `array` · `record`

```javascript
let age = 25                 // int
let temp = 36.6              // float
let greeting = "hello"       // string
let active = true            // boolean
let numbers = [1, 2, 3]     // array (homogeneous)
let point = {x: 10, y: 20}  // record (fixed shape)
```

### Functions

Functions are expressions with typed parameters and return types.

```javascript
let add = (a int, b int) int {
    return a + b
}

let greet = (name string) none {
    show("Hello, " + name + "!")
}
```

### Control Flow

```javascript
if x > 0 {
    show("positive")
} else if x == 0 {
    show("zero")
} else {
    show("negative")
}

for item in collection {
    show(to_string(item))
}

while condition {
    // ...
}
```

### Error Handling

No try/catch. Monk uses `guard`/`against`/`throw`:

```javascript
let divide = (a int, b int) int {
    if b == 0 { throw "division by zero" }
    return a / b
}

guard result = divide(10, 0) against error {
    show("Caught: " + to_string(error))
    result = 0
}
```

### Type System

Monk has a static type checker. Annotations are optional — the checker infers from first assignment.

```javascript
let x = 42             // inferred: int
let name = "monk"      // inferred: string
let scores int[] = [1, 2, 3]  // typed array — backed by int64_t* in C
let maybe int? = none  // optional type — int or none
```

Caught at compile time:
- Type drift on reassignment (`x = "hi"` when `x` is `int` → error)
- Cross-type equality (`5 == "5"` → error)
- Missing record fields, wrong field types
- Missing return paths in non-`none` functions
- Arity and argument type mismatches
- Loop variable mutation (`for i in arr { i = 0 }` → error)

**Typed arrays compile to raw C types.** `int[]` uses `int64_t*` backing, not `MonkValue*`. Element reads/writes are direct pointer access. Scalar variables (`int`, `float`, `bool`) emit as `int64_t`, `double`, `bool` — no union overhead in arithmetic.

### Value Semantics

Assignment copies. Function args copy. Your data is yours.

```javascript
let original = [3, 1, 2]
let sorted = bubble_sort(original)
// original is still [3, 1, 2]
```

## Examples

The [`examples/`](examples/) directory has 23 working programs:

| File | What it shows |
|------|---------------|
| [`hello.monk`](examples/hello.monk) | Hello world |
| [`fibonacci.monk`](examples/fibonacci.monk) | Recursion, typed functions |
| [`fizzbuzz.monk`](examples/fizzbuzz.monk) | Conditionals, `for` loops, modulo |
| [`arrays.monk`](examples/arrays.monk) | Array operations, iteration, `append` |
| [`error_handling.monk`](examples/error_handling.monk) | `guard`/`against`/`throw` |
| [`newton_sqrt.monk`](examples/newton_sqrt.monk) | Float arithmetic, recursion, error boundaries |
| [`todo_list.monk`](examples/todo_list.monk) | Records, value semantics, data modeling |
| [`collatz.monk`](examples/collatz.monk) | `while` loops, arrays, the Collatz conjecture |
| [`sort.monk`](examples/sort.monk) | Bubble sort, value semantics proof |
| [`types.monk`](examples/types.monk) | Type annotations, inference, typed arrays, optionals |
| [`records.monk`](examples/records.monk) | Nested records with structural typing |
| [`optionals.monk`](examples/optionals.monk) | `T?` semantics, `is_none`, graceful reads |
| [`guards.monk`](examples/guards.monk) | `guard`/`against`/`throw` scoping |
| [`closures.monk`](examples/closures.monk) | First-class functions, capture-by-copy |
| [`higher_order.monk`](examples/higher_order.monk) | `map`, `filter`, `reduce` with typed arrays |
| [`default_params.monk`](examples/default_params.monk) | Default parameter values |
| [`value_semantics.monk`](examples/value_semantics.monk) | Assignment copies, function args copy |
| [`strings.monk`](examples/strings.monk) | String operations |
| [`math.monk`](examples/math.monk) | Math builtins |
| [`binary_search.monk`](examples/binary_search.monk) | Binary search with typed arrays |
| [`bitwise.monk`](examples/bitwise.monk) | Bitwise operators |
| [`gcd_lcm.monk`](examples/gcd_lcm.monk) | GCD/LCM, number theory |
| [`sieve.monk`](examples/sieve.monk) | Sieve of Eratosthenes, typed arrays |

## CLI

```
monk <command> [arguments]

Commands:
  build <file.monk> [-o output]   Compile to native binary
  run <file.monk>                  Compile and run
  check <file.monk>                Parse and validate (no compilation)
  version                          Print version
  help                             Show help
```

The `-o` flag controls output format:

| Command | Output |
|---------|--------|
| `monk build hello.monk` | `hello` (native binary) |
| `monk build hello.monk -o app` | `app` (native binary) |
| `monk build hello.monk -o hello.c` | `hello.c` (C source, no compilation) |

## How It Works

Monk is a **compiler**, not an interpreter. There is no REPL.

```
source.monk  -->  [Go compiler]  -->  generated.c  -->  [cc -O3 -flto]  -->  native binary
```

<table>
<tr>
<td width="50%">

**Compiler pipeline**

1. **Lexer** — source text to tokens
2. **Parser** — tokens to AST (29 node types, 13 precedence levels)
3. **Type checker** — static analysis, inference, unboxing hints
4. **Codegen** — AST to C11 source (raw scalars for typed vars)
5. **cc/clang** — C source + runtime to native binary

</td>
<td width="50%">

**C runtime** (~1,200 lines across 8 files)

- `MonkValue` tagged union for boxed types
- `int64_t*`/`double*`/`bool*` backing for typed arrays
- Deep copy for value semantics
- 40+ builtin functions
- Error handling via `setjmp`/`longjmp`

</td>
</tr>
</table>

## Builtins

| Category | Functions |
|----------|-----------|
| **Output** | `show` |
| **Conversion** | `to_string` `to_int` `to_float` |
| **Type check** | `typeof` `is_array` `is_record` `is_string` `is_int` `is_float` |
| **String** | `length` `substring` `split` `join` `trim` `to_upper_case` `to_lower_case` `starts_with` `ends_with` `contains` `replace` `index_of` `char_at` |
| **Array** | `append` `pop` `slice` `range` `length` |
| **Math** | `sqrt` `pow` `abs` `floor` `ceil` `round` `sin` `cos` `tan` `log` `min` `max` |
| **File I/O** | `file_read` `file_write` `file_exists` |

## Performance

Benchmarks on Apple M4 Pro (`cc -O3 -flto`, hyperfine, lower is better). 21 benchmarks total.

**At or below C (11/21):**

| Benchmark | Monk | vs C |
|---|---:|---:|
| fibonacci (n=35) | 17.1 ms | **0.9×** |
| mandelbrot (800²×50) | 15.3 ms | **1.0×** |
| leibniz (π, 50M iter) | 28.4 ms | **1.0×** |
| trial_primes (<200k) | 8.7 ms | **1.0×** |
| collatz | 93.6 ms | **1.0×** |
| ackermann | 459 ms | **1.0×** |
| bitcount | 60.6 ms | **0.9×** |
| sqrt_sum | 9.2 ms | **1.0×** |
| quicksort | 2.4 ms | **0.9×** |
| fannkuch | 122 ms | **0.8×** |
| record_access | 4.2 ms | **1.4×** |

**Near C — typed array overhead (3/21):**

| Benchmark | Monk | vs C | Root cause |
|---|---:|---:|---|
| nbody (float[], 500k steps) | 17.9 ms | **1.5×** | Extra pointer indirection: `MonkValue → MonkFloatArray → data` |
| matmul (400² int[]) | 19.8 ms | **1.6×** | Same + no `restrict` — compiler can't prove non-aliasing |
| sieve (int[], 1M elements) | 6.3 ms | **2.0×** | `int64_t` (8 bytes) vs C's `char` (1 byte): 8× memory bandwidth |

**Structural gaps — semantic overhead (7/21):**

| Benchmark | Monk | vs C | Root cause |
|---|---:|---:|---|
| functional_chain | 10.5 ms | **3.8×** | `map`/`filter`/`reduce` allocate intermediate arrays |
| string_concat | 39.2 ms | **6.9×** | Immutable strings; concat is O(n²) |
| closure_invoke | 21.5 ms | **11.9×** | Heap-allocated closure struct per iteration; C uses a direct call |
| for_in_sum | 26.0 ms | **15.3×** | `range(10M)` allocates 80 MB; C reference uses a formula (no allocation) |
| levenshtein | 68.0 ms | **26.2×** | `substring()` allocates a new string per character comparison |
| binary_trees | 199 ms | **24.9×** | Value-semantics deep copy on every tree node assignment |
| string_ops | 88.4 ms | **52.0×** | `to_upper_case` allocates a new string per call |

The typed array benchmarks (matmul, sieve, nbody) use `int[]`/`float[]` with `int64_t*`/`double*` backing stores and bounds-check elision. The remaining typed-array gap is two pointer hops (`MonkValue → TypedArray → data`) and, for sieve, the 8× memory stride difference versus C's `char` array. Structural gaps require COW arrays, string views, or closure escape analysis — none are small changes.

See [`bench/`](bench/) for the harness and [`spec/PERFORMANCE.md`](spec/PERFORMANCE.md) for methodology.

## Design Philosophy

Three rules resolve every edge case:

1. **Explicit over implicit.** No hidden coercion, no hidden errors, no hidden mutation. One exception: truthiness (`false`, `none`, `0` are falsy).
2. **Graceful on reads, strict on operations.** Reading missing data = `none` or `[]`. Operating on invalid data = error.
3. **Values, not references.** Assignment copies. Function args copy. Closures copy. No exceptions.

## Status

**0.0.1 — Buniyaad** (2026-04-04). 630 tests passing (467 Go + 163 C runtime). 21 benchmarks, 23 examples.

| Phase | Status |
|-------|--------|
| 1. Lexer | :white_check_mark: Done |
| 2. Parser | :white_check_mark: Done |
| 3. C Runtime Library | :white_check_mark: Done |
| 4. C Code Generation | :white_check_mark: Done |
| 5. CLI | :white_check_mark: Done |
| 6. Type System + scalar unboxing codegen | :white_check_mark: Done |
| 7. Module System | :arrow_left: Next |
| 8. C FFI | Planned |
| 9. Linter & Formatter | Planned |
| 10. LSP + Editor | Planned |
| 11. Distribution | Planned |

## Project Structure

```
src/                Go compiler (module root)
  main.go             CLI entry point (39 tests)
  embed.go            Embedded runtime (self-contained binary)
  syntax/             Lexer + Parser + AST (195 tests)
  types/              Static type checker (112 tests)
  codegen/            AST → C code generator + unboxing (63 tests)
    unbox.go            Scalar/typed-array unboxing, storage kind inference
  runtime/            C runtime library (8 .c files, 163 C tests)
    runtime.h           Public API (MonkValue + function declarations)
    internal.h          Shared helpers
    value.c, arith.c, string.c, container.c, math.c, builtins.c, error.c, higher_order.c
spec/               Language specification (REFERENCE.md is the source of truth)
knowledge/          Learning course — monkfromearth.github.io/monk-lang/
examples/           23 working .monk programs
bench/              Benchmark suite (21 benchmarks, Monk vs C)
Makefile            make build/install/test/clean
```

## Links

- [Language Reference](spec/REFERENCE.md) — the source of truth for syntax and semantics
- [Architecture Decisions](spec/ARCHITECTURE_DECISIONS.md) — why Go, why compile-to-C
- [Roadmap](ROADMAP.md) — phased build plan with checkboxes
- [Changelog](CHANGELOG.md) — release history
- [Knowledge Site](https://monkfromearth.github.io/monk-lang/) — compiler course and learning resources
- [v1 Archive](https://github.com/monkfromearth/monk-lang-v1) — original TypeScript/Bun interpreter

## License

MIT
