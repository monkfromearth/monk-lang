<div align="center">

<br/>

# Monk Lang

**Performance** of compiled languages. **Simplicity** of Python. **Memory safety** inspired by Rust.

A minimalist, readable, and performant programming language for the modern age.

[![Build](https://img.shields.io/badge/build-passing-brightgreen?style=flat)](#status)
[![Tests](https://img.shields.io/badge/tests-553_passing-brightgreen?style=flat)](#status)
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

### Value Semantics

Assignment copies. Function args copy. Your data is yours.

```javascript
let original = [3, 1, 2]
let sorted = bubble_sort(original)
// original is still [3, 1, 2]
```

## Examples

The [`examples/`](examples/) directory has 9 working programs:

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
source.monk  -->  [Go compiler]  -->  generated.c  -->  [cc -O2]  -->  native binary
```

<table>
<tr>
<td width="50%">

**Compiler pipeline**

1. **Lexer** — source text to tokens
2. **Parser** — tokens to AST (29 node types, 13 precedence levels)
3. **Codegen** — AST to C11 source
4. **cc/clang** — C source + runtime to native binary

</td>
<td width="50%">

**C runtime** (~830 lines)

- `MonkValue` tagged union for all types
- Deep copy for value semantics
- 40+ builtin functions
- UTF-8 string handling
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

## Design Philosophy

Three rules resolve every edge case:

1. **Explicit over implicit.** No hidden coercion, no hidden errors, no hidden mutation. One exception: truthiness (`false`, `none`, `0` are falsy).
2. **Graceful on reads, strict on operations.** Reading missing data = `none` or `[]`. Operating on invalid data = error.
3. **Values, not references.** Assignment copies. Function args copy. Closures copy. No exceptions.

## Status

**0.0.1 — Buniyaad** (2026-04-04). 553 tests passing (402 Go + 151 C runtime).

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
  codegen/            AST → C code generator + scalar unboxing (56 tests)
  runtime/            C runtime library (runtime.h, runtime.c, 151 C tests)
spec/               Language specification (REFERENCE.md is the source of truth)
knowledge/          Learning course — monkfromearth.github.io/monk-lang/
examples/           13 working .monk programs
bench/              Benchmark suite (Monk vs C/Go/Py/Node/Bun)
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
